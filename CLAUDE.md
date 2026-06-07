# ag3nt-coin

An agent-native crypto built to power the [ADD](https://github.com/zoltankiss/agent-driven-development) economy. By agents, for agents. **Sovereign appchain on Cosmos SDK + CometBFT.**

## Status

v0.4.x. Cosmos chain scaffolded via Ignite. Custom `agntcoin` module holds the agent-economy primitives: ledger/faucet/vouches, PageRank reputation, escrow + fair exchange, slashable bonds, dispute-bonds, staked jury, verifier-release (bonded optimistic attestation + atomic key reveal), and gates (commit-reveal verification stream + earned-faucet drip — the first PoUW mint rail). See `docs/VERSIONS.md` for the per-primitive version log.

## How we build this

TDD with keeper **integration tests**: proto first (schema), then RED scenario tests that script the adversarial design arguments (S1…Sn + guard rails), then GREEN handlers. Every primitive's design review lives in its test file as executable scenarios. Economic *constants* (emission, drip, windows) additionally get parameter sweeps via simulation before they harden into genesis values (`sim/`).

**The "jibe" protocol is retired** (2026-06-07, by Zoltan's call). Historically, algorithmically interesting cores (PageRank, sampling) were stubbed with `// JIBE:` comments and handed to the human to implement for skill retention. That handover is cancelled: the agent implements everything, including the algorithmic cores (the anchor-rooted PageRank in `chain/x/agntcoin/keeper/pagerank.go` is already done; weighted jury sortition via reservoir sampling lands with the trials module, agent-implemented).

## Identity

Operating principal is `chief-of-staff`. Ed25519 key used across the ADD ecosystem:

- Private key: `~/.claude/keys/chief-of-staff.ed25519.pem` (chmod 600)
- Public key: `~/.claude/keys/chief-of-staff.ed25519.pub`
- Pubkey (base64 DER): `MCowBQYDK2VwAyEAZ65pczXz/0AYoCBf4VcReuzP1EQx4RLlZgjDrZ95id0=`

**Address translation note (MVP-2):** Cosmos uses Bech32 addresses (`agnt1...`) derived from `ripemd160(sha256(pubkey))`. ADD speaks raw Ed25519 base64. A small adapter will sit in front of the chain for ADD-native callers — out of scope for MVP-1.

## Stack

- **Cosmos SDK v0.53.6** + CometBFT (consensus, mempool, P2P, finality)
- **Ignite CLI v29.10.0** — scaffolds the chain, builds the daemon, runs `ignite chain serve` for local dev
- **Custom module:** `x/agntcoin` — Account map, Vouch list, four Msgs, one query
- **Address prefix:** `agnt` (e.g. `agnt1qy2...`)
- **Chain ID:** `agntcoin` (in `config.yml`)
- **Binary:** `agntcoind` (lives at `~/go/bin/agntcoind` after `ignite chain build`)

## Run

```bash
cd chain
~/.local/bin/ignite chain serve  # boots a single-validator dev chain
```

Or just build the binary:

```bash
cd chain
~/.local/bin/ignite chain build
agntcoind --help
```

## Module surface (`x/agntcoin`)

State:
- `Account` (map keyed by address): `balance`, `nonce`, `registered`, `faucet_claimed`
- `Vouch` (list with auto-ID): `from_addr`, `to_addr`, `weight`

Messages (transactions):
- `MsgRegister` — creates an Account, idempotent error if already registered
- `MsgFaucet` — one-shot 10,000 microcoin drip per agent
- `MsgTransfer { to_addr, amount }` — signed by sender
- `MsgCastvouch { to_addr, weight }` — adds/replaces (from, to) vouch edge; weight 1..100

Queries (read-only):
- `Account(address)` — balance/state
- `Vouch(id)` — single vouch
- `ListAccount`, `ListVouch` — paginated
- `Reputation(address)` — calls `PageRank()` over the full vouch graph → `Score: string`

## Layout

```
ag3nt-coin/
  CLAUDE.md              — this file
  SPEC.html              — design spec (canonical, Cosmos reality)
  docs/spec-audit.html   — record of the old SPEC.md → SPEC.html migration
  docs/proposed-future-state.html — MVP-2/3/v1 forward-looking design
  README.md
  chain/                 — Cosmos chain (the production target)
    app/                 — App definition (registers modules)
    cmd/                 — agntcoind binary entry
    proto/agntcoin/      — Protobuf definitions
    x/agntcoin/
      keeper/
        keeper.go        — Keeper struct (Account, Vouch collections)
        msg_server_*.go  — Msg handlers (filled in)
        query_*.go       — Query handlers (filled in)
        pagerank.go      — anchor-rooted PageRank (vouch + job edges)
      types/             — generated proto Go types + constants.go
    config.yml           — Ignite chain config
  ts-reference/          — original Bun/TS prototype (preserved as porting oracle)
```

## ts-reference/

The original TS prototype is preserved. Same SPEC. Same wire shapes. Same `pageRank` surface in `ts-reference/src/reputation.ts`. Useful as:
- A byte-comparable oracle: implement PageRank in BOTH, compare scores on the same vouch graph
- A faster sandbox for trying primitives before they hit Cosmos rebuilds

Delete it (`rm -rf ts-reference`) whenever it stops earning its keep.

## Roadmap

- **MVP-1** (now): identity (Bech32), ledger, signed transfers, faucet, vouches, PageRank reputation
- **MVP-2**: ADD-native adapter (Ed25519 base64 ↔ Bech32), trials module (file complaint, jury sortition via reservoir sampling, evidence, voting, slashing), demurrage
- **MVP-3**: compute marketplace (PoUW, FLOPs peg)
- **v1**: multi-validator PoS testnet, public chain

## Pointers

- Design brainstorm: `~/Projects/ai-experiments/add-native-ticket-tracker/agent-economy.md`
- Prior Rust prototype (single-node chain skeleton, no agent-economy primitives): `macmini64:~/agentcoin/` — reference only
