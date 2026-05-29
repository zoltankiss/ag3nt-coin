#!/usr/bin/env bash
# Run the live loop against contracts already deployed on Base Sepolia.
# Reads ESCROW_ADDRESS + keys from agent/.env.testnet. Starts the worker, then
# Agent A posts a job; worker claims + titles via claude -p + submits; Agent A
# judges + settles. No deploy / no funding (those are one-time).
set -euo pipefail

AGENT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
BUN="$(command -v bun || echo "$HOME/.bun/bin/bun")"
SCAN="https://sepolia.basescan.org"

set -a; . "$AGENT_DIR/.env.testnet"; set +a
[ -n "${ESCROW_ADDRESS:-}" ] || { echo "✗ ESCROW_ADDRESS not set in .env.testnet"; exit 1; }
cd "$AGENT_DIR"

echo "▸ escrow $ESCROW_ADDRESS  (chain $CHAIN_ID via $RPC_URL)"
echo "▸ starting worker (hermes-64)…"
WORKER_PID=""
cleanup() { [ -n "$WORKER_PID" ] && kill "$WORKER_PID" 2>/dev/null || true; }
trap cleanup EXIT
RPC_URL=$RPC_URL CHAIN_ID=$CHAIN_ID ESCROW_ADDRESS=$ESCROW_ADDRESS \
  HERMES_PK=$WORKER64_PK HERMES_NAME=hermes-64 "$BUN" run src/hermes.ts & WORKER_PID=$!
sleep 4

echo "▸ Agent A posts a job…"
RPC_URL=$RPC_URL CHAIN_ID=$CHAIN_ID ESCROW_ADDRESS=$ESCROW_ADDRESS \
  AGENT_A_PK=$AGENT_A_PK "$BUN" run src/agent-a.ts "${1:-}"

echo ""
echo "▸ BaseScan:"
echo "  escrow $SCAN/address/$ESCROW_ADDRESS"
echo "  buyer  $SCAN/address/$AGENT_A_ADDR"
echo "  worker $SCAN/address/$WORKER64_ADDR"
