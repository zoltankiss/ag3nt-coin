# ag3ntcoin Tokenomics and Emission Policy

**Status:** Implemented as the `0.5.0-beta.1` emission-accounting beta, with
raw AGNT integer accounting. The `0.5.0-beta.2` blind-gate hygiene beta keeps
that emission accounting unchanged while tightening review-gate artifact
discipline.
**Current chain version:** `0.5.0-beta.2`.

This document specifies the hard-cap emission model for agntcoin. The beta.1
chain code enforces total supply, epoch accounting, scheduled issuance, and
burn-equivalent accounting for unclaimed scheduled reward.

## Goals

- `MAX_SUPPLY` is a hard upper bound, like Bitcoin's 21M cap.
- Genesis gives the founder trust-root reputation, not AGNT.
- AGNT enters circulation through useful work, not premine discretion.
- Protocol rewards continue to have a positive mathematical schedule for every
  fixed-length epoch.
- Unused scheduled issuance is permanently burned, so final mined supply can be
  less than the hard cap.
- The protocol should be conservative: scheduled reward is an upper bound, not
  an obligation to mint.

## Constants

```text
MAX_SUPPLY = 21,000,000 AGNT
GENESIS_PREMINE = 0 AGNT
FOUNDER_INITIAL_AGNT = 0 AGNT
EPOCH_LENGTH = 7 days
EPOCH_REWARD_RATE = 0.001
```

`EPOCH_REWARD_RATE = 0.001` means each epoch schedules 0.1% of the remaining
unissued supply.

## Emission Formula

Epochs are fixed time windows. Epoch numbering starts at `1`.

```text
scheduled_reward_n = MAX_SUPPLY * r * (1 - r)^(n - 1)

where:
  r = 0.001
  n = epoch number, starting at 1
```

For ag3ntcoin:

```text
scheduled_reward_n = 21,000,000 * 0.001 * 0.999^(n - 1)
```

The infinite scheduled issuance is:

```text
sum(scheduled_reward_n, n = 1..infinity) = 21,000,000 AGNT
scheduled_reward_n > 0 for all n, mathematically
```

## Mined Plus Burned Identity

Each epoch's scheduled reward is split into actually mined AGNT and unclaimed
burned issuance.

```text
actual_mined_n <= scheduled_reward_n
burned_unclaimed_n = scheduled_reward_n - actual_mined_n
scheduled_reward_n = actual_mined_n + burned_unclaimed_n
```

Therefore:

```text
sum(actual_mined_n + burned_unclaimed_n, n = 1..infinity)
  = 21,000,000 AGNT
```

The protocol-level hard cap is:

```text
total_mined <= 21,000,000 AGNT
```

If an epoch has no accepted useful work, then:

```text
actual_mined_n = 0
burned_unclaimed_n = scheduled_reward_n
```

The burned unclaimed amount is not minted first. It is permanently removed from
future issuance at epoch close. In accounting terms it is burn-equivalent
unissued supply.

## First 20 Scheduled Epoch Rewards

Each row is the maximum scheduled reward for that epoch. The actual minted
amount can be lower. The unclaimed remainder is burned.

```text
scheduled_reward = actual_mined + burned_unclaimed
```

| Epoch | Scheduled Reward |
|---:|---:|
| 1 | 21,000.000000 AGNT |
| 2 | 20,979.000000 AGNT |
| 3 | 20,958.021000 AGNT |
| 4 | 20,937.062979 AGNT |
| 5 | 20,916.125916 AGNT |
| 6 | 20,895.209790 AGNT |
| 7 | 20,874.314580 AGNT |
| 8 | 20,853.440266 AGNT |
| 9 | 20,832.586825 AGNT |
| 10 | 20,811.754239 AGNT |
| 11 | 20,790.942484 AGNT |
| 12 | 20,770.151542 AGNT |
| 13 | 20,749.381390 AGNT |
| 14 | 20,728.632009 AGNT |
| 15 | 20,707.903377 AGNT |
| 16 | 20,687.195474 AGNT |
| 17 | 20,666.508278 AGNT |
| 18 | 20,645.841770 AGNT |
| 19 | 20,625.195928 AGNT |
| 20 | 20,604.570732 AGNT |

After 20 epochs, if every scheduled reward were fully mined:

```text
total scheduled mined+burned = 416,033.838580 AGNT
remaining scheduled supply = 20,583,966.161420 AGNT
```

## Long-Range Checkpoints

At epoch 500:

```text
elapsed time ~= 9.58 years
scheduled_reward_500 = 12,746.7045466315 AGNT
total scheduled mined+burned through epoch 500 ~= 8,266,042.157915 AGNT
remaining scheduled supply ~= 12,733,957.842085 AGNT
scheduled share used ~= 39.36%
```

At epoch 2000:

```text
elapsed time ~= 38.33 years
scheduled_reward_2000 = 2,842.0404738213 AGNT
total scheduled mined+burned through epoch 2000 ~= 18,160,801.566653 AGNT
remaining scheduled supply ~= 2,839,198.433347 AGNT
scheduled share used ~= 86.48%
```

## Reward Is Not Automatic

The scheduled reward is not a faucet target. It is the maximum claimable
protocol issuance for that epoch.

Accepted useful work may include:

- protocol code patches
- protocol documentation fixes
- security/audit findings
- calibrated review-gate work
- artifact integrity work
- future approved protocol-public work categories

Every mint event should remain evidence-bound:

```text
recipient
amount
epoch
rail
scope
artifact_uri
artifact_sha256
evidence_uri
evidence_sha256
rationale_hash
```

## Conservative Minting Rule

The protocol should not try to exhaust every epoch's scheduled reward. It should
mint only for accepted, reviewable useful work and burn the rest.

Practical interpretation:

```text
epoch scheduled reward is an upper bound, not a target
mint conservatively
burn aggressively
```

## Suggested Pool Split

The first implementation can begin with a single epoch reward pool. Before
public launch, split the epoch reward into rails so one activity does not crowd
out all others.

Proposed starting split:

```text
40% protocol code/docs contributions
25% security and audit work
20% review gates and reviewer calibration
10% ecosystem tooling
5% emergency/governance reserve
```

Unused reward in each pool expires at epoch close and is burned. It does not
roll over unless a later governance primitive explicitly changes this policy.

## `0.5.0-beta.1` Implementation Notes

The beta.1 implementation uses raw AGNT integer units because the existing
chain balances are `uint64` AGNT. That is intentionally not the final
production precision story. It validates the monetary-policy shape while
keeping the first implementation small enough to audit.

The ideal mathematical curve is implemented discretely:

```text
epoch_scheduled = remaining_scheduled / 1000
```

When integer division would otherwise schedule `0` while a positive raw-AGNT
remainder still exists, the beta.1 implementation schedules the remaining
amount as the final discrete reward. This preserves the hard cap with the
current integer balance model, but it is not a 10,000-year tail design.

Production hardening choices before a real-money launch:

```text
option A: migrate all balances/amounts to a high-precision base unit
option B: keep integer AGNT and define an explicit finite-tail rule
option C: keep emission accounting in a high-precision accumulator while
          wallet balances settle only claimable whole/base units
```

Ethereum-style `10^18` base units would not fit the current `uint64` amount
fields at `21,000,000 AGNT`, so any high-precision migration is a chain-wide
type change, not a constant tweak.

Runway note: with weekly epochs, raw integer beta.1 scheduling reaches the
final discrete tail zone around epoch `9,947`, roughly `191` years after
genesis. That is acceptable for beta beachhead validation, but it should still
be revisited before any irreversible production-denomination commitment.

Implemented state:

```text
max_supply
epoch_length_seconds
epoch_reward_divisor
genesis_time
current_epoch
epoch_started_at
epoch_scheduled
epoch_mined
total_mined
total_burned_unclaimed
```

Implemented protocol mint rails:

```text
gate settlement drip
accepted contribution awards
```

The legacy one-shot module faucet is disabled after `0.5.0-beta.2`; `MsgFaucet`
rejects claims instead of minting. Fresh agents register at `0` AGNT and earn
through gate drips, accepted contribution awards, or ordinary market payments.

Market payments, escrow release/refund, bonds, and vouch stake movement are not
protocol issuance. They move existing balances and do not consume epoch reward.

Implemented query/client surface:

```text
ag3nt emission
```

## Beta.2 Beachhead Evidence Checklist

For beta.2 runs, public artifacts should make emission changes
independently reviewable without relying on private run notes. Every accepted
contribution award, gate drip, or faucet test should include:

```text
run_id
recipient_address
rail
amount
artifact_uri
artifact_sha256
evidence_sha256
emission_before
emission_after
balance_before
balance_after
verification_command
```

Contribution awards should pin the reviewed work to a repository URL and commit
SHA. Private GitHub artifacts are acceptable only for private beachhead runs
when reviewers can refetch the exact bytes with authenticated `gh` or `git` and
verify the recorded SHA-256. A public chain should not treat local GitHub
credentials as an artifact availability primitive.

Blind gate artifacts should publish only the payload hash, public question set,
commit/reveal transaction references, and settlement emission snapshots. The
gold answer and salt stay private until settlement.

## Future Precision Notes

The mathematical schedule has positive reward for every epoch. A real chain has
finite integer base units, so the implementation needs fixed precision.

Recommended practical precision:

```text
BASE_UNITS_PER_AGNT = 10^18
MAX_SUPPLY_BASE = 21,000,000 * BASE_UNITS_PER_AGNT
```

The protocol should keep explicit accounting:

```text
total_mined_base
total_burned_unclaimed_base
effective_remaining_base
current_epoch
epoch_scheduled_base
epoch_mined_base
epoch_burned_unclaimed_base
```

All protocol mint rails should route through one keeper-level function, for
example:

```text
mintProtocolReward(recipient, amount, rail, evidence)
```

That function should enforce:

```text
total_mined_base + amount <= MAX_SUPPLY_BASE
epoch_mined_base + amount <= epoch_scheduled_base
amount > 0
evidence is hash-pinned
duplicate evidence cannot mint twice
recipient identity is registered
rail-specific rules pass
```

At epoch close, the chain should record:

```text
burned_unclaimed_base = epoch_scheduled_base - epoch_mined_base
total_burned_unclaimed_base += burned_unclaimed_base
```

## Versioning Recommendation

The chain `VERSION` advanced from `0.5.0-beta.1` to `0.5.0-beta.2` after the
emission primitive was in place. Beta.2 keeps the monetary-policy primitive and
adds safer blind gate construction around it.

Recommended sequence:

```text
0.4.0-beta.3
  current artifact-integrity / founder-mining beta

0.5.0-beta.1
  implements MAX_SUPPLY, fixed 7-day epochs, scheduled reward accounting,
  unclaimed burn accounting, and routes faucet/gate/contribution mints through
  the common protocol reward function

0.5.0-beta.2
  keeps beta.1 emission accounting and adds gate-template tooling plus private
  GitHub artifact hygiene for blind PR-review gates; disables the legacy module
  faucet after the beachhead exposed that cap-bounded faucet minting is still
  not Sybil-safe

post-beta.2
  add rail splits, artifact availability checks at mint time, and richer
  emission queries

0.5.0
  cut only after simulations and forge runs validate the emission primitive
```

This changes the old prospective `0.5.0` roadmap item. Verifier-market
hardening should move to a later minor version if tokenomics becomes the next
forged primitive.

## Open Questions

- Should epoch boundaries be wall-clock block time or block height?
- Should epoch reward be split from day one, or begin as one pool?
- What rail-specific maximums should exist inside the epoch cap?
- How should late submissions near epoch close be handled?
- Should burned-unclaimed accounting be queryable as first-class chain state?
- Should post-mint burns be tracked separately from unclaimed scheduled burns?
