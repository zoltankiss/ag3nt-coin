# ag3nt-coin versions

ag3nt-coin is versioned by **forged primitive**, not by calendar. A version is the trust layer's
capability set at a point in time; the forge's iterations are run *against* a version (the iteration's
`manifest.chain_sha` pins the exact commit within it).

## Scheme (semver, pre-launch `0.x`)
- **MAJOR** — reserved: `1.0.0` = public testnet launch (the catastrophic-hole class mined out); breaking consensus changes thereafter.
- **MINOR** — a **new primitive forged** (built *and* adversarially re-validated — RUNBOOK Phase 5). A primitive isn't "done" until a subsequent iteration's adversary has tried it and failed.
- **PATCH** — a fix/hardening *within* an existing primitive (no new capability).

Tag each release: `git tag v<X.Y.Z> <sha> && git push --tags`. The current version is in [`/VERSION`](../VERSION).

## History (initial assignment — refine as needed)
| Version | Primitive forged | Forged in | Key SHA |
|---|---|---|---|
| 0.1.0 | Native token + **escrow** (lock/submit/dispute/release/refund) + **fair-exchange** | it7 → it7.1 (re-validated) | — |
| 0.2.0 | **Anchor-rooted PageRank reputation** + staked **vouch** (cost-of-identity) | it5–it6 | — |
| 0.3.0 | **k-of-n dispute jury** (open/vote/resolve) | it12 | — |
| 0.4.0 | **jury-bound escrow** (`no_auto_release`) + on-chain **`delivery_hash`** (closes the it13 #19 deadline-bypass) | it13 → it15 | `baa77fb` |
| **0.5.0** | **Slashable bond** primitive (`PostBond`/`ReleaseBond`/`SlashBond`, no self-withdraw) — the keystone | **it17** (validated; invariants I1–I4 held under attack) | `6f065a4` (findings `e04be15`) |
| 0.6.0 *(next)* | **Dispute-bond** (open-requires-bond; frivolous → slash to griefed worker; verifiable bond↔escrow ref) | it18 RED → it19 GREEN | _pending_ |

*(Earlier SHAs left blank — back-fill from `git log`/iteration manifests when convenient. The firm anchor is 0.5.0 = the current chain.)*

## Roadmap (prospective MINORs — see ag3nt-coin-forge design-context)
`0.6` dispute-bond · `0.7` jury-v1 (k-of-n staked jurors + collusion) · `0.8` checker-market/canary ·
`0.9` PoUW mint rails (proto-mining) · `0.10` mint-capstone · `0.11` reputation convergence (proper-scoring + bidirectional slashing) · **`1.0` beachhead testnet genesis**.
