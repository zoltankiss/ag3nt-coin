# Founder Self-Award Preflight Fix

AGNT_FORGE_RUN_ID=forge2-emission-tokenomics-blind-gate-beachhead-1780878307-33789

Role: emission-founder
Created: 2026-06-08T01:02:00Z

## Observation

During the beta.2 beachhead, contribution award id `0` minted 25 AGNT to the
founder/anchor address even though the reviewed contribution request identified
the contributor wallet as `agnt1ekfgyfgehnktzcp8ejs2ym4895y5kzhvclgtgt`.

This did not change the emission-accounting result: `total_mined` and
`epoch_mined` increased by exactly 25 for the award, and gate settlement later
minted 6 more. The issue is award attribution and operator preflight, not the
21M supply cap.

## Fix

The `ag3nt contribution-award` client now rejects awards where the recipient is
the signing anchor unless the operator passes `--allow-self-award`.

The explicit override keeps founder-authored work possible, but makes accidental
self-awards visible at the command line and reserves them for independently
reviewed founder-authored contributions.

## Verification

```bash
bun test client/src/ag3nt.test.ts
```

Result: 9 tests passed.

```bash
bun --check client/src/ag3nt.ts client/src/ag3nt.test.ts client/src/cli.ts
```

Result: pass.

```bash
client/bin/ag3nt contribution-award agnt1tt8yenqef0v7mvg5h0s49qrww3h3rmjf3j3xr2 https://github.com/zoltankiss/ag3nt-coin - 2c55815fe7716a19631ab07e8c6affdb7c68887b https://example.invalid/artifact.md 0000000000000000000000000000000000000000000000000000000000000000 1111111111111111111111111111111111111111111111111111111111111111 code_fix - 1
```

Result:

```text
error: contribution-award recipient matches the signing anchor; pass --allow-self-award only for independently reviewed founder-authored work
```

Follow-up: this founder-authored fix should not receive a contribution award
without independent review evidence.
