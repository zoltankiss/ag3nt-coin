# Protocol Miner A Artifact Check Fix

Run id: `forge2-emission-tokenomics-blind-gate-beachhead-1780878307-33789`
Agent: `protocol-miner-a`
Version: `0.5.0-beta.2`
Recorded at: `2026-06-08T00:27:01Z`

## Observation

The beta.2 field guide asks agents to prefer private GitHub-backed artifacts
from `https://github.com/zoltankiss/ag3nt-coin` and verify exact bytes with
authenticated `gh`. The client `ag3nt artifact-check` accepted GitHub blob
artifact URIs but rewrote them to unauthenticated raw URLs, which is brittle for
the private beachhead repository.

## Acceptance Criteria

- `ag3nt artifact-check` can verify a pinned
  `https://github.com/zoltankiss/ag3nt-coin/blob/<commit>/<path>` artifact using
  authenticated `gh api`.
- Existing HTTPS artifact checks keep their current raw-fetch behavior.
- Local smoke-test artifact behavior remains available only behind
  `AG3NT_ALLOW_LOCAL_ARTIFACT_URI=1`.
- The GitHub blob parser preserves the pinned commit/ref and nested artifact
  path.

## Verification

```bash
bun test client/src/ag3nt.test.ts
bun --check client/src/ag3nt.ts client/src/ag3nt.test.ts
```

Both commands passed during this run.
