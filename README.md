# ag3nt-coin

A sovereign appchain for an agent economy, built on **Cosmos SDK + CometBFT** and scaffolded with **Ignite**. The custom `x/agntcoin` module implements the on-chain primitives an autonomous-agent marketplace needs: identity and a native ledger, reputation, trust-minimized exchange, and a proof-of-useful-work mint rail.

## Implemented primitives

All live in `chain/x/agntcoin`:

- **Ledger / faucet / vouches** — registered accounts, a one-shot faucet drip, and a weighted vouch graph.
- **Anchor-rooted PageRank reputation** — reputation scores computed over the vouch (and job) graph, rooted at a trusted anchor set so a clique cannot mint reputation for itself.
- **Escrow + fair exchange** — protocol-held escrow with atomic key-reveal, so payment and delivery settle together.
- **Slashable bonds & dispute-bonds** — economic skin-in-the-game for participants, slashed on proven misbehavior; dispute-bonds price the cost of raising a frivolous dispute.
- **Staked jury** — disputes resolved by a staked, sortition-selected jury with stake and bond settlement rules.
- **Bonded optimistic verifier-release** — a bonded verifier optimistically attests delivery; releases can be challenged within a window, with fraud penalties.
- **Commit-reveal PoUW gates** — a commit-reveal verification stream that gates an earned-faucet drip: the first proof-of-useful-work mint rail.

## Build & run

Requires the [Ignite CLI](https://docs.ignite.com/) and Go.

```bash
cd chain
ignite chain serve     # boots a single-validator dev chain (hot-reloads on change)
```

Or build the daemon directly:

```bash
cd chain
ignite chain build     # produces the agntcoind binary
agntcoind --help
```

- Address prefix: `agnt` (Bech32, e.g. `agnt1qy2...`)
- Chain ID: `agntcoin` (see `chain/config.yml`)

## Tests

The module is built test-first. Design arguments for each primitive are captured as executable adversarial scenarios in the keeper test files.

```bash
cd chain
go test ./...          # 108 keeper integration tests
```

## Architecture

```
ag3nt-coin/
  chain/                       Cosmos SDK + CometBFT appchain (the production target)
    app/                       app wiring (registers modules)
    proto/agntcoin/            protobuf message + query definitions
    x/agntcoin/keeper/         msg/query handlers, escrow, jury, verifier, pagerank.go
    config.yml                 Ignite chain config
  sim/                         economic-constant parameter sweeps (emission, drip, windows)
  ts-reference/                original TS prototype, kept as a porting oracle
```

See **[SPEC.html](./SPEC.html)** for the full design spec (open in a browser) and `docs/VERSIONS.md` for the per-primitive version log.

## License

Apache-2.0. See [LICENSE](./LICENSE).
