# ag3nt — the pluggable client

A non-custodial, **Ed25519-native** client for ag3nt-coin. An agent that has nothing but a keypair can onboard (get coins) and pay/vouch with **zero** knowledge of Cosmos. The agent holds its own key and signs its own txs locally — the client is a *translator*, never a custodian.

This is the piece that drops into a [CPDD](../../cpdd) session so customer/entrepreneur agents can transact.

## Why "easy for agents"

- **One key = identity + wallet.** The agent's Ed25519 key *is* its `agnt1…` address (`address = bech32(agnt, sha256(pubkey)[:20])`). No separate wallet, no second keypair.
- **Gasless.** The chain accepts 0-fee txs; onboarding mints coin on the agent's own signature.
- **Non-custodial.** The client signs locally with the agent's key. Nothing else ever holds it.
- **No Cosmos knowledge required.** The agent runs `ag3nt pay <addr> <amount>`. That's it.

## Commands

```
ag3nt whoami                  # show my address (creates my key on first run)
ag3nt discover                # ADD-native self-description (zero-doc onboarding)
ag3nt onboard                 # one-time: bootstrap + claim 10,000 ag3nt-coin
ag3nt balance [addr]          # ag3nt-coin balance
ag3nt pay <addr> <amount>     # send ag3nt-coin to another agent
ag3nt vouch <addr> <weight>   # stake 1..100 trust (feeds reputation/PageRank)
ag3nt reputation [addr]       # reputation score (weighted PageRank over vouches)
```

## Setup

Requires a running ag3nt-coin chain (local dev) + [bun](https://bun.sh).

```bash
# 1) start the chain (from repo root)
cd chain && ignite chain serve            # RPC :26657, REST :1317, faucet :4500

# 2) install the client
cd client && bun install

# 3) use it (each identity is its own keyfile via AG3NT_KEY)
AG3NT_KEY=/tmp/alice.json bun run src/cli.ts onboard
AG3NT_KEY=/tmp/alice.json ./bin/ag3nt balance
```

Config is env-overridable: `AG3NT_API`, `AG3NT_RPC`, `AG3NT_FAUCET`, `AG3NT_CHAIN_ID`, `AG3NT_KEY`.

## Plugging into a CPDD session

CPDD agents are `claude -p` subprocesses with Bash. To give them money + reputation:

1. **Mint a per-agent identity.** In the CPDD runner, give each agent its own keyfile:
   ```bash
   export AG3NT_KEY="$ROOT/logs/wallets/$name.json"
   ```
2. **Add `client/bin` to the agents' PATH** (or alias `ag3nt`).
3. **Drop this into the agent prompt:**

   > You have an ag3nt-coin wallet (Ed25519, non-custodial). Run `ag3nt discover` to see what you can do. Your address: `ag3nt whoami`. Onboard once with `ag3nt onboard` to claim your starting balance. To pay another agent, `ag3nt pay <their-address> <amount>`. To signal trust in an agent who did good work, `ag3nt vouch <their-address> <weight 1-100>` — this builds their public reputation. Check anyone's standing with `ag3nt reputation <address>`.

That's the whole integration. The marketplace logic (escrow, who-pays-whom) stays in the CPDD app the entrepreneur builds — ag3nt-coin just provides the money and the reputation primitive underneath it.

## What this is / isn't (MVP scope)

- **Is:** a working, non-custodial, Ed25519, gasless agent economy — onboard, pay, vouch, reputation (real weighted PageRank) — usable today against a local chain.
- **Isn't (yet):** escrow primitives, a public testnet, or fully protocol-native account creation (onboarding currently bootstraps the `x/auth` account via the dev faucet before the gasless module faucet — a real deploy would fold this into a custom ante). These are deliberate "later" items; this MVP exists to be **pluggable into CPDD now**.
