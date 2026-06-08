# protocol-miner-a contribution evidence

run_id: forge2-emission-recipient-binding-validation-1780882484-79157
agent: protocol-miner-a
agent_address: agnt1ekfgyfgehnktzcp8ejs2ym4895y5kzhvclgtgt
chain_api: http://localhost:1317
configured_wallet: /Users/macmini64/projects/ag3nt/ag3nt-coin-forge-2/logs/wallets/protocol-miner-a.json

## Contribution

primary_code_commit_sha: 353a7a9db7ffc9dc98a400eb6c84d42194f720a3
branch: codex/contribution-award-binding-output
repo_url: https://github.com/zoltankiss/ag3nt-coin
scope: protocol-contribution/client-recipient-binding-output

This patch makes `ag3nt contribution-award` success output include the
reviewed contributor address, a boolean `recipient_binding` field, founder
authorship metadata, and review evidence URI. That lets reviewers confirm the
beta.3 recipient-binding invariant from the command receipt before doing a
follow-up query.

The branch also rejects founder-authored metadata on non-anchor awards, keeps
gate-template discovery text version-neutral for the 0.5.0-beta.3 run, and adds
regression tests proving malformed blind-gate gold answers fail before
public/private files are written.

## Acceptance Criteria

- Fresh miner registered with `ag3nt register`, not `ag3nt onboard`.
- Starting balance remained 0 after registration.
- Baseline emission before any award/gate/payment:
  - current_epoch: 1
  - epoch_mined: 0
  - total_mined: 0
  - remaining_epoch: 21000
- Contribution recipient/contributor for any award request should be:
  `agnt1ekfgyfgehnktzcp8ejs2ym4895y5kzhvclgtgt`.
- Award command must use:
  `--contributor-address agnt1ekfgyfgehnktzcp8ejs2ym4895y5kzhvclgtgt`.
- Reviewers should reject any award where recipient differs from contributor.
- Reviewers should inspect the commit diff and this artifact before any award.

## Verification

Passed:

```text
bun test client/src/ag3nt.test.ts
```

Result: 17 pass, 0 fail, 26 expect calls.

Passed:

```text
bun client/src/cli.ts discover | rg -n "gate-template|contribution-award"
```

Result: discovery output exposes gate-template and contribution-award commands.

Not configured:

```text
bun run --cwd client tsc --noEmit
```

Result: client package has no `tsc` script.

Not configured:

```text
bun --cwd client x tsc --noEmit
```

Result: current Bun invocation did not provide an `x` script path.

## Harness Note

Private `/harness/memory` could not be written because the forge sidecar was not
reachable on `127.0.0.1:4300` or `127.0.0.1:4301` at artifact time. This public
artifact does not copy prior run ids or prior run evidence.
