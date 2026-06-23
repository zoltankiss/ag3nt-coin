# ag3nt-coin — Architecture & build

An agent-native crypto built to power an autonomous-agent economy. By agents, for agents. **Sovereign appchain on Cosmos SDK + CometBFT.**

## Status

v0.4.x. Cosmos chain scaffolded via Ignite. The custom `agntcoin` module holds the agent-economy primitives: ledger/faucet/vouches, anchor-rooted PageRank reputation, escrow + fair exchange, slashable bonds, dispute-bonds, staked jury, verifier-release (bonded optimistic attestation + atomic key reveal), and gates (commit-reveal verification stream + earned-faucet drip — the first PoUW mint rail). See `docs/VERSIONS.md` for the per-primitive version log.

## How this is built

TDD with keeper **integration tests**: proto first (schema), then RED scenario tests that script the adversarial design arguments (guard rails included), then GREEN handlers. Every primitive's design review lives in its test file as executable scenarios. Economic *constants* (emission, drip, windows) additionally get parameter sweeps via simulation before they harden into genesis values (`sim/`). Algorithmic cores are implemented directly in the module — for example the anchor-rooted PageRank in `chain/x/agntcoin/keeper/pagerank.go`; weighted jury sortition via reservoir sampling lands with the trials module.

## Identity & addressing

The chain uses Bech32 addresses (`agnt1...`) derived from `ripemd160(sha256(pubkey))`. For callers that speak raw Ed25519 base64, a small address-translation adapter is planned to sit in front of the chain — out of scope for MVP-1.

## Stack

- **Cosmos SDK v0.53.6** + CometBFT (consensus, mempool, P2P, finality)
- **Ignite CLI v29.10.0** — scaffolds the chain, builds the daemon, runs `ignite chain serve` for local dev
- **Custom module:** `x/agntcoin` — Account map, Vouch list, Msgs and queries
- **Address prefix:** `agnt` (e.g. `agnt1qy2...`)
- **Chain ID:** `agntcoin` (in `config.yml`)
- **Binary:** `agntcoind` (produced by `ignite chain build`)

## Run

```bash
cd chain
ignite chain serve   # boots a single-validator dev chain
```

Or just build the binary:

```bash
cd chain
ignite chain build
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

(Escrow, bonds, jury, verifier-release, and gate messages/queries extend this surface; see the keeper and proto definitions for the current set.)

## Layout

```
ag3nt-coin/
  CLAUDE.md              — this file (architecture & build)
  README.md              — overview
  SPEC.html              — design spec (canonical, Cosmos reality)
  docs/spec-audit.html   — record of the old SPEC.md → SPEC.html migration
  docs/proposed-future-state.html — MVP-2/3/v1 forward-looking design
  chain/                 — Cosmos chain (the production target)
    app/                 — App definition (registers modules)
    cmd/                 — agntcoind binary entry
    proto/agntcoin/      — Protobuf definitions
    x/agntcoin/
      keeper/
        keeper.go        — Keeper struct (Account, Vouch collections)
        msg_server_*.go  — Msg handlers
        query_*.go       — Query handlers
        pagerank.go      — anchor-rooted PageRank (vouch + job edges)
      types/             — generated proto Go types + constants.go
    config.yml           — Ignite chain config
  sim/                   — economic-constant parameter sweeps
  ts-reference/          — original Bun/TS prototype (preserved as porting oracle)
```

## ts-reference/

The original TS prototype is preserved. Same SPEC, same wire shapes, same `pageRank` surface in `ts-reference/src/reputation.ts`. Useful as:
- A byte-comparable oracle: implement a primitive in both, compare scores on the same vouch graph
- A faster sandbox for trying primitives before they hit Cosmos rebuilds

## Roadmap

- **MVP-1** (now): identity (Bech32), ledger, signed transfers, faucet, vouches, PageRank reputation
- **MVP-2**: Ed25519-base64 ↔ Bech32 adapter, trials module (file complaint, jury sortition via reservoir sampling, evidence, voting, slashing), demurrage
- **MVP-3**: compute marketplace (PoUW, FLOPs peg)
- **v1**: multi-validator PoS testnet, public chain
