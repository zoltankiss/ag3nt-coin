# Protocol Miner A Gate Template Secret Fix

Run id: `forge2-emission-tokenomics-blind-gate-beachhead-1780878307-33789`
Agent: `protocol-miner-a`
Version: `0.5.0-beta.2`
Recorded at: `2026-06-08T00:36:00Z`

## Observation

`ag3nt gate-template` kept gold answers and salts out of the public payload, but
its JSON stdout included a ready-to-copy `settle_command` containing both secret
values. In this forge workflow, command stdout is easy to paste into public
artifacts or archives, so the blind-gate secret still had a practical leak path.

## Acceptance Criteria

- `ag3nt gate-template` stdout must not include `gold_answer`, `gold_salt`, or
  a settlement command containing either value.
- The private secret file must retain the gold answer, salt, and exact settlement
  command for the gate author to use after the reveal window closes.
- Existing public payload generation and post command output must remain
  unchanged.

## Verification

```bash
bun test client/src/ag3nt.test.ts
bun --check client/src/ag3nt.ts client/src/ag3nt.test.ts
```

Both commands passed during this run.
