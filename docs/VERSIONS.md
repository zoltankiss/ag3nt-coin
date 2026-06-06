# ag3nt-coin versions

ag3nt-coin is versioned by **forged primitive / chain capability**, not by calendar. **Versioning begins at
`0.0.1`** — the first version we formally cut. Everything before is **pre-versioning** (unversioned history;
the forge's `iteration-1 … iteration-16` archives, left flat). Forge iterations are run *against* a version
(each run's `manifest.chain_sha` pins the exact commit), and grouped in `ag3nt-coin-forge/iterations/v<version>/<slug>/` —
**apples-to-apples**: everything under a version dir ran against the byte-identical chain.

## Scheme (pre-launch)
See [`VERSIONING.md`](VERSIONING.md) for the full policy. In short: **MINOR = a new forged primitive**
(bonds, dispute-bonds, juror-stake…); **PATCH = a non-breaking fix within one**; **`1.0.0` = public testnet
launch**. `0.0.1` was the pre-versioning baseline; the first primitive cut under the policy (dispute-bond) is
`0.1.0`. Tag each version: `git tag v<X.Y.Z> <sha> && git push origin v<X.Y.Z>`. Current version is in [`/VERSION`](../VERSION).

## Versions
| Version | Chain capability forged | Forged / exercised in | Key SHA |
|---|---|---|---|
| _pre-0.0.1_ | Native token, **escrow** + fair-exchange, anchor-rooted **reputation** + vouch, **k-of-n jury**, **jury-bound escrow** (`no_auto_release`) + `delivery_hash` | iterations 1–16 (unversioned) | see each iteration's `manifest.chain_sha` |
| **0.0.1** | **Slashable bond** primitive (`PostBond`/`ReleaseBond`/`SlashBond`, no self-withdraw — the keystone) | forged + shakedown `it17`; RED `it18` (jury-ddos) | `6f065a4` (findings `e04be15`) |
| **0.1.0** | **Dispute-bond** (`OpenDispute`-requires-bond; frivolous/lost → slash to griefed worker; upheld → returned; verifiable bond↔escrow ref) | built post-`it18`; GREEN re-validation = `it19` | tag `v0.1.0` |
| **0.2.0** | **Juror-stake / jury-v1** (`CastVote`-requires-stake — 3rd bond parameterization; `ResolveDispute` requires a real quorum so a single juror can't rush a 1-0 verdict; a juror incoherent with the resolved verdict is slashed to the wronged party). Defends a *minority* colluder; *majority* collusion → `it25`. | RED→GREEN `it20 red-jury-collusion` | tag `v0.2.0` |

## Roadmap (prospective)
`0.3.0` checker-market/canary · `0.4.0` PoUW mint rails (proto-mining) · `0.5.0` mint-capstone ·
`0.6.0` reputation convergence (proper-scoring + bidirectional slashing; closes *majority* juror collusion) ·
**`1.0.0` beachhead testnet genesis.** (One MINOR per forged primitive — see [`VERSIONING.md`](VERSIONING.md) + the forge `design-context`.)
