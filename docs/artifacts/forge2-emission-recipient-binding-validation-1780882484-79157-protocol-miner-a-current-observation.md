# protocol-miner-a current-run observation

run_id: forge2-emission-recipient-binding-validation-1780882484-79157
agent: protocol-miner-a
agent_address: agnt1ekfgyfgehnktzcp8ejs2ym4895y5kzhvclgtgt
configured_wallet: /Users/macmini64/projects/ag3nt/ag3nt-coin-forge-2/logs/wallets/protocol-miner-a.json
chain_api: http://localhost:1317
observed_at_role_time: 2026-06-07T20:55:00-05:00

## Scope

This artifact records protocol-miner-a's current state in the
0.5.0-beta.3 recipient-binding validation run. It does not copy evidence from
prior run ids. The wallet was already registered when this process inspected
the chain, so this is a continuation observation, not a fresh 0 balance
registration artifact.

## Current Chain State

`ag3nt emission` after the accepted award and settled gate showed:

```json
{
  "current_epoch": "1",
  "epoch_mined": "27",
  "total_mined": "27",
  "remaining_epoch": "20973"
}
```

`ag3nt balance` for this wallet showed:

```json
{
  "registered": true,
  "balance": "22"
}
```

Follow-up monitor note: at wall-clock `2026-06-07T21:02:25-05:00`, the same
wallet showed `balance: "23"` while `ag3nt emission` still showed
`total_mined: "27"` and `epoch_mined: "27"`. Gate `0` remained open with two
revealed unpaid answers, and gate `1` remained settled with two paid answers
that were not this wallet address. This looks like a non-emission balance
movement observed during the run, not a new contribution mint.

## Accepted Award Binding

`ag3nt contribution-award-get 0` showed a valid non-founder contribution award:

```json
{
  "recipient": "agnt1ekfgyfgehnktzcp8ejs2ym4895y5kzhvclgtgt",
  "commit_sha": "9ea10110703861dcee92ffdf82085dd3334c3a8e",
  "amount": "25",
  "contributor": "agnt1ekfgyfgehnktzcp8ejs2ym4895y5kzhvclgtgt",
  "founder_authored": false
}
```

Observed invariant: `recipient == contributor` for protocol-miner-a.

## GitHub-Backed Artifact Check

The earlier protocol-miner-a acceptance artifact is pinned at:

```text
https://github.com/zoltankiss/ag3nt-coin/blob/23c0f939074cd7d4187dbdf47adc4d7fd8e532d7/docs/artifacts/forge2-emission-recipient-binding-validation-1780882484-79157-protocol-miner-a-binding-output.md
```

Expected SHA-256:

```text
9498f8e9e75cf2e26de49e629e1eeea044a271aecc817a82c0e4fdb97f6c43cf
```

Verification command:

```bash
ag3nt artifact-check 'https://github.com/zoltankiss/ag3nt-coin/blob/23c0f939074cd7d4187dbdf47adc4d7fd8e532d7/docs/artifacts/forge2-emission-recipient-binding-validation-1780882484-79157-protocol-miner-a-binding-output.md' 9498f8e9e75cf2e26de49e629e1eeea044a271aecc817a82c0e4fdb97f6c43cf
```

Result:

```json
{
  "ok": true,
  "access_method": "gh",
  "actual_sha256": "9498f8e9e75cf2e26de49e629e1eeea044a271aecc817a82c0e4fdb97f6c43cf",
  "bytes": 2583
}
```

## Gate Observation

`ag3nt gate 1` showed a settled blind gate using the current run id in its
payload URI:

```json
{
  "id": "1",
  "status": "settled",
  "drip": "1",
  "max_answers": "2",
  "gold_answer": "Y,Y,Y,Y,Y",
  "consensus": "Y,Y,Y,Y,Y"
}
```

The public gate payload URI was:

```text
http://127.0.0.1:4313/artifacts/live/emission-founder-recipient-binding-review-gate-forge2-emission-recipient-binding-validation-1780882484-79157.public-gate.md
```

## Notes

- `ag3nt onboard` was not used by protocol-miner-a in this observation.
- Private diary was written to `/harness/memory` on the forge sidecar at
  `http://127.0.0.1:4313/harness/memory`.
- The accepted award's on-chain artifact URI is local sidecar evidence. The
  GitHub-backed artifact check above provides an authenticated private-repo
  byte check for the committed protocol-miner-a acceptance artifact.
