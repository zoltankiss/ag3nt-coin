#!/usr/bin/env bash
# End-to-end on a local Anvil chain — no testnet, no faucet, fully self-contained.
# Boots anvil, deploys (escrow deploys token + pre-mints to Agent A), funds the
# worker with ag3nt for its bond, starts the worker, then has Agent A post a real
# job. The worker claims it, runs `claude -p` to title the ticket, and submits;
# Agent A judges and settles. Proves the whole loop end to end.
set -euo pipefail

AGENT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
ROOT="$(cd "$AGENT_DIR/.." && pwd)"
RPC=http://127.0.0.1:8545
FORGE="$HOME/.foundry/bin/forge"
CAST="$HOME/.foundry/bin/cast"
ANVIL="$HOME/.foundry/bin/anvil"
BUN="$(command -v bun || echo "$HOME/.bun/bin/bun")"

# Anvil's well-known deterministic accounts (public test keys — local only).
DEPLOYER_PK=0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80   # acct0
AGENT_A_PK=0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d    # acct1
WORKER_PK=0x5de4111afa1a4b94908f83103eb1f1706367c2e68ca870fc3fb9a804cdab365a    # acct2
AGENT_A_ADDR=0x70997970C51812dc3A010C7d01b50e0d17dc79C8
WORKER_ADDR=0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC

ANVIL_PID=""; WORKER_PID=""
cleanup() { [ -n "$WORKER_PID" ] && kill "$WORKER_PID" 2>/dev/null || true
            [ -n "$ANVIL_PID" ]  && kill "$ANVIL_PID"  2>/dev/null || true; }
trap cleanup EXIT

echo "▸ starting anvil…"
"$ANVIL" --silent & ANVIL_PID=$!
sleep 1

echo "▸ deploying (escrow deploys token, pre-mints 1M AGNT to Agent A)…"
cd "$ROOT/contracts"
DEPLOY_OUT=$(AGENT_A_ADDRESS=$AGENT_A_ADDR "$FORGE" script script/Deploy.s.sol \
  --rpc-url "$RPC" --private-key "$DEPLOYER_PK" --broadcast 2>&1)
ESCROW=$(echo "$DEPLOY_OUT" | grep -i 'JobEscrow' | grep -oE '0x[a-fA-F0-9]{40}' | head -1)
TOKEN=$(echo "$DEPLOY_OUT"  | grep -i 'Ag3nt'     | grep -oE '0x[a-fA-F0-9]{40}' | head -1)
if [ -z "$ESCROW" ] || [ -z "$TOKEN" ]; then echo "✗ deploy parse failed:"; echo "$DEPLOY_OUT"; exit 1; fi
echo "  escrow = $ESCROW"
echo "  token  = $TOKEN"

echo "▸ funding worker with 1000 AGNT for bonds…"
WEI=$("$CAST" to-wei 1000 ether)
"$CAST" send "$TOKEN" "transfer(address,uint256)" "$WORKER_ADDR" "$WEI" \
  --rpc-url "$RPC" --private-key "$AGENT_A_PK" >/dev/null

echo "▸ starting worker…"
cd "$ROOT/agent"
RPC_URL=$RPC CHAIN_ID=31337 ESCROW_ADDRESS=$ESCROW HERMES_PK=$WORKER_PK HERMES_NAME=hermes-local \
  "$BUN" run src/hermes.ts & WORKER_PID=$!
sleep 2

echo "▸ Agent A posts a job…"
RPC_URL=$RPC CHAIN_ID=31337 ESCROW_ADDRESS=$ESCROW AGENT_A_PK=$AGENT_A_PK \
  "$BUN" run src/agent-a.ts "${1:-}"

echo "▸ done."
