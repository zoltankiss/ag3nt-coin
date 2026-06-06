# ag3nt-coin versions

ag3nt-coin is versioned by **forged primitive / chain capability**, not by calendar. **Versioning begins at
`0.0.1`** — the first version we formally cut. Everything before is **pre-versioning** (unversioned history;
the forge's `iteration-1 … iteration-16` archives, left flat). Forge iterations are run *against* a version
(each run's `manifest.chain_sha` pins the exact commit), and grouped in `ag3nt-coin-forge/iterations/v<version>/<slug>/` —
**apples-to-apples**: everything under a version dir ran against the byte-identical chain.

## Scheme (pre-launch)
- Pre-launch we don't pretend release-grade minor/major semantics. **Each distinct chain version we run
  iterations against gets the next `0.0.N`** (even a tiny fix → its own version → its own dir).
- **`1.0.0`** is reserved for the **public testnet launch** (the catastrophic-hole class mined out).
- Tag each version: `git tag v<X.Y.Z> <sha> && git push origin v<X.Y.Z>`. Current version is in [`/VERSION`](../VERSION).

## Versions
| Version | Chain capability forged | Forged / exercised in | Key SHA |
|---|---|---|---|
| _pre-0.0.1_ | Native token, **escrow** + fair-exchange, anchor-rooted **reputation** + vouch, **k-of-n jury**, **jury-bound escrow** (`no_auto_release`) + `delivery_hash` | iterations 1–16 (unversioned) | see each iteration's `manifest.chain_sha` |
| **0.0.1** | **Slashable bond** primitive (`PostBond`/`ReleaseBond`/`SlashBond`, no self-withdraw — the keystone) | forged + shakedown `it17`; RED `it18` (jury-ddos) | `6f065a4` (findings `e04be15`) |
| 0.1.0 *(next)* | **Dispute-bond** (open-requires-bond; frivolous → slash to griefed worker; verifiable bond↔escrow ref) | RED `it18` wrote the spec → GREEN `it19` | _pending_ |

## Roadmap (prospective)
`0.1.0` dispute-bond · `0.2.0` jury-v1 (k-of-n staked jurors + collusion) · `0.3.0` checker-market/canary ·
`0.4.0` PoUW mint rails (proto-mining) · `0.5.0` mint-capstone · `0.6.0` reputation convergence ·
**`1.0.0` beachhead testnet genesis.** (Numbers are sequential, not significance-weighted — see the forge `design-context`.)
