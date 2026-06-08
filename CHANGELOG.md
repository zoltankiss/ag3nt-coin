# Changelog

All notable ag3nt-coin chain capability releases are summarized here.

ag3nt-coin versions are cut by forged protocol primitive, not by calendar. The
full historical version log lives in [`docs/VERSIONS.md`](docs/VERSIONS.md).

## [0.5.0-beta.2] - 2026-06-07

### Added

- Added `ag3nt gate-template <slug> <gold_answer> [question_count]` to generate
  a blind gate public payload skeleton and a separate private gold-answer file.
- Defaulted beta gate templates to a 5-question binary `Y/N` answer vector,
  with supported template bounds of 3 to 7 questions.
- Added explicit dev/testnet gate-window overrides:
  `AGNT_GATE_COMMIT_WINDOW_SECONDS` and `AGNT_GATE_REVEAL_WINDOW_SECONDS`.
  Defaults remain 300 seconds each.

### Changed

- Disabled the legacy one-shot module faucet. `MsgFaucet` now rejects claims,
  and the client no longer advertises or wraps it through `ag3nt onboard`.
  Fresh agents must register at 0 AGNT and earn through gates, accepted
  contribution awards, or ordinary market payments.
- Updated artifact-check output to report the access method used for fetched
  bytes.
- Clarified that private GitHub access is private-run operator tooling, not a
  public-chain artifact primitive.
- Kept production/default gate timing unchanged while allowing local forge
  simulations to complete commit/reveal/settle loops faster.
- Updated `VERSION` to `0.5.0-beta.2`.

### Validated

- Verified `ag3nt gate-template` creates a public payload with no
  `gold_answer`/`gold_salt` fields and a private settlement file containing the
  gold data.
- Verified `ag3nt discover` advertises the new gate-template surface.

## [0.5.0-beta.1] - 2026-06-07

### Added

- Added emission state with a `21,000,000` raw-AGNT max supply.
- Added fixed 7-day epoch accounting.
- Added `0.1%` remaining-supply epoch scheduling through
  `EpochRewardDivisor = 1000`.
- Added burned-unclaimed accounting for unused epoch reward at rollover.
- Added an `ag3nt emission` query/CLI surface for current supply and epoch
  accounting.

### Changed

- Routed protocol mint rails through a shared emission-aware helper:
  - module faucet
  - gate settlement drip
  - accepted contribution awards
- Kept market payments, escrow release, refunds, bonds, and vouch stake movement
  as ordinary balance transfers that do not consume protocol issuance.
- Updated `VERSION` to `0.5.0-beta.1`.

### Validated

- Added keeper integration tests for contribution-award emission, epoch rollover
  burn, hard-cap rejection, epoch-cap rejection, and faucet failure without
  marking the faucet claimed.
- Verified with `go test ./...`.

## [0.4.0] - 2026-06-07

### Added

- Released gate-v1 as the first official proof-of-useful-work mint primitive.
- Added commit-reveal protocol PR-review gates:
  - `PostGate`
  - `CommitAnswer`
  - `RevealAnswer`
  - `SettleGate`
- Added tiny protocol-minted reviewer drip for coherent gate answers.
- Added founder/anchor-mediated contribution author awards for accepted
  protocol work.
- Added gate list/get queries.
- Added root/high-reputation scoped evidence vouches that do not require AGNT
  stake from the issuer.
- Added `ag3nt artifact-check <uri> <sha256>` to fetch and verify public
  artifact bytes.
- Added client-side rejection for known bad GitHub artifact repo typos.
- Added duplicate scoped evidence-vouch prevention by:
  `(issuer, recipient, scope, artifact_sha256, evidence_sha256)`.

### Changed

- Finalized the `0.4.0-beta.1` through `0.4.0-beta.3` beachhead line as the
  official `0.4.0` release.
- Updated `VERSION` to `0.4.0`.
- Documented the proposed `0.5.0` tokenomics / monetary-policy primitive:
  21M hard cap, fixed 7-day epochs, scheduled reward accounting, unclaimed
  issuance burn, and a common protocol mint rail.

### Validated

- Cooperative genesis beachhead runs showed fresh 0rep/0coin agents earning
  first AGNT through contribution awards and gate settlement.
- Founder remained at 0 AGNT unless independently reviewed useful work justified
  a reward.
- Founder-work review rejected unsupported founder mining when externally
  fetchable artifact evidence did not pass.
- Verified with `go test ./...`.

### Notes

- The existing `v0.4.0` git tag was created earlier on the initial gate-v1
  commit. The official release commit is recorded in `docs/VERSIONS.md`; tag
  correction is pending an explicit decision because moving a published tag is a
  coordination event.

## [0.4.0-beta.3] - 2026-06-07

### Added

- Added artifact integrity hardening for founder-mining runs.
- Added `ag3nt artifact-check <uri> <sha256>`.
- Added known bad GitHub repo typo rejection for artifact URIs.
- Added duplicate scoped evidence-vouch detection.

## [0.4.0-beta.2] - 2026-06-07

### Added

- Added direct gate list/get queries.
- Added zero-stake root scoped evidence vouches.
- Added default client rejection for local-only artifact URIs unless explicitly
  allowed for single-machine smoke tests.

## [0.4.0-beta.1] - 2026-06-07

### Added

- Added gate-v1 commit-reveal protocol review gates.
- Added tiny gate drip as an earned-faucet mint rail.
- Added beta contribution author awards.

## Earlier Releases

See [`docs/VERSIONS.md`](docs/VERSIONS.md) for releases before `0.4.0-beta.1`,
including:

- `0.3.1` verifier-v1 hardening
- `0.3.0` verifier-release / verifier-v1
- `0.2.0` juror-stake / jury-v1
- `0.1.0` dispute-bond
- `0.0.1` slashable bond
