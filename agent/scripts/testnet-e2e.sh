#!/usr/bin/env bash
# End-to-end on Base Sepolia (live testnet). Uses the dedicated wallets in
# agent/.env.testnet. The deployer (funded from the faucet) spreads a little ETH
# to the buyer + worker for gas, deploys the contracts, the buyer is seeded with
# 1M AGNT, then we run the real loop: Agent A posts → worker claims + titles via
# claude -p → submits → Agent A settles. Prints BaseScan links.
set -euo pipefail

AGENT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
ROOT="$(cd "$AGENT_DIR/.." && pwd)"
FORGE="$HOME/.foundry/bin/forge"
CAST="$HOME/.foundry/bin/cast"
BUN="$(command -v bun || echo "$HOME/.bun/bin/bun")"

set -a; . "$AGENT_DIR/.env.testnet"; set +a   # load keys + RPC_URL + addrs
RPC="$RPC_URL"
SEND() { "$CAST" send "$@" --rpc-url "$RPC" >/dev/null; }
SCAN="https://sepolia.basescan.org"

echo "▸ deployer balance: $("$CAST" from-wei "$("$CAST" balance "$DEPLOYER_ADDR" --rpc-url "$RPC")") ETH"

echo "▸ spreading gas ETH to buyer + worker…"
# Explicit sequential nonces + async broadcast — the public RPC lags between
# back-to-back sends, so letting cast auto-pick the nonce collides.
N=$("$CAST" nonce "$DEPLOYER_ADDR" --rpc-url "$RPC")
"$CAST" send "$AGENT_A_ADDR"  --value 0.00003ether --private-key "$DEPLOYER_PK" --rpc-url "$RPC" --nonce "$N"        --async >/dev/null
"$CAST" send "$WORKER64_ADDR" --value 0.00003ether --private-key "$DEPLOYER_PK" --rpc-url "$RPC" --nonce "$((N+1))" --async >/dev/null
echo "  broadcast (nonces $N, $((N+1))); waiting for confirmations…"
sleep 10

echo "▸ deploying JobEscrow (+ Ag3nt, pre-mint 1M to Agent A)…"
cd "$ROOT/contracts"
DEPLOY_OUT=$(AGENT_A_ADDRESS=$AGENT_A_ADDR "$FORGE" script script/Deploy.s.sol \
  --rpc-url "$RPC" --private-key "$DEPLOYER_PK" --broadcast --slow 2>&1)
ESCROW=$(echo "$DEPLOY_OUT" | grep -i 'JobEscrow' | grep -oE '0x[a-fA-F0-9]{40}' | head -1)
TOKEN=$(echo "$DEPLOY_OUT"  | grep -i 'Ag3nt'     | grep -oE '0x[a-fA-F0-9]{40}' | head -1)
if [ -z "$ESCROW" ] || [ -z "$TOKEN" ]; then echo "✗ deploy failed:"; echo "$DEPLOY_OUT"; exit 1; fi
echo "  escrow = $ESCROW"
echo "  token  = $TOKEN"

# persist escrow address for later (re-runs)
if grep -q '^ESCROW_ADDRESS=' "$AGENT_DIR/.env.testnet"; then
  sed -i '' "s|^ESCROW_ADDRESS=.*|ESCROW_ADDRESS=$ESCROW|" "$AGENT_DIR/.env.testnet"
fi

echo "▸ seeding worker with 1000 AGNT for its bond…"
WEI=$("$CAST" to-wei 1000 ether)
SEND "$TOKEN" "transfer(address,uint256)" "$WORKER64_ADDR" "$WEI" --private-key "$AGENT_A_PK"

echo "▸ starting worker (hermes-64, running locally for now)…"
cd "$ROOT/agent"
WORKER_PID=""
cleanup() { [ -n "$WORKER_PID" ] && kill "$WORKER_PID" 2>/dev/null || true; }
trap cleanup EXIT
RPC_URL=$RPC CHAIN_ID=84532 ESCROW_ADDRESS=$ESCROW HERMES_PK=$WORKER64_PK HERMES_NAME=hermes-64 \
  "$BUN" run src/hermes.ts & WORKER_PID=$!
sleep 3

echo "▸ Agent A posts a job…"
RPC_URL=$RPC CHAIN_ID=84532 ESCROW_ADDRESS=$ESCROW AGENT_A_PK=$AGENT_A_PK \
  "$BUN" run src/agent-a.ts "${1:-}"

echo ""
echo "▸ BaseScan:"
echo "  escrow   $SCAN/address/$ESCROW"
echo "  token    $SCAN/token/$TOKEN"
echo "  buyer    $SCAN/address/$AGENT_A_ADDR"
echo "  worker   $SCAN/address/$WORKER64_ADDR"
echo "▸ done."
