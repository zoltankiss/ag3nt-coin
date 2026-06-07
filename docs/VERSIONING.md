# ag3nt-coin versioning policy

ag3nt-coin uses **semver** (`MAJOR.MINOR.PATCH`), with the pre-1.0 caveat that we are **pre-public-launch**,
so below `1.0.0` there is **no backward-compatibility promise** — the levels are a *legible guide*, not a contract.
The actual version log (what each version forged + its SHA) is in [`VERSIONS.md`](VERSIONS.md).

## The levels
- **PATCH** (`x.y.Z`) — a **non-breaking fix or security patch** *within* an existing primitive. No new capability, no state/consensus break. e.g. a bug in a bond handler.
- **MINOR** (`x.Y.0`) — a **new primitive or notable capability**, chain-level backward-compatible. e.g. slashable bonds, dispute-bonds, jury-v1, the checker market. *Most pre-launch work lands here.*
- **MAJOR** (`X.0.0`) — a **breaking change requiring a coordinated hard fork** (not backward compatible).
  - **`1.0.0`** = the **first public launch** (testnet beta, per the launch posture). This is the inflection: real outsiders now depend on the chain, so the consensus + genesis + tx/client contract become **load-bearing**, and breaking them costs a coordinated fork.
  - **`2.0.0`+** = any later non-backward-compatible change needing a coordinated hard fork: a **consensus / app-state-machine migration** (the "PoW→PoS"-style fundamental change), a **token-economics overhaul** (e.g. the **PoUW mint going live**, changing emission/supply), a **chain-id / genesis migration** (testnet → mainnet), or **breaking the tx/client contract**.

## Pre-1.0 (now)
- **Versioning began at `0.0.1`** — the *baseline* snapshot of everything built before we started versioning (native token, escrow + fair-exchange, anchor-rooted reputation + vouch, k-of-n jury, jury-bound escrow, slashable bonds). Everything before that is **unversioned history** (forge `iteration-1 … iteration-16`, left flat).
- The policy governs changes **from the baseline forward**: the first *new primitive* cut under it (the dispute-bond) is **`0.1.0`**; a non-breaking fix to the baseline would be `0.0.2`.
- Below `1.0.0`, expect churn — anything may change. The levels keep the history readable; they are not a compatibility guarantee yet.
- Pre-release suffixes such as `0.4.0-beta.1` mean the primitive exists but is not yet validated enough to cut the final version. Use beta/rc labels for beachhead-facing economics where simulations or forge runs still determine constants, scope, or launch posture.

## Discipline (do this on every chain change a forge iteration will run against)
1. Bump [`/VERSION`](../VERSION).
2. **Tag it:** `git tag v<X.Y.Z> <sha> && git push origin v<X.Y.Z>` (or `v<X.Y.Z-beta.N>` for pre-release validation cuts) — so every distinct chain state is pinned + named.
3. **Log it** in [`VERSIONS.md`](VERSIONS.md): version → capability → forging iteration → SHA.
4. Each distinct version gets its **own forge iterations dir** (apples-to-apples): `ag3nt-coin-forge/iterations/v<X.Y.Z>/<slug>/` — everything under a version dir ran against the byte-identical chain.
