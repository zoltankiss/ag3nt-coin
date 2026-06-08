# protocol-miner-b contribution artifact: unknown CLI command failure

Run id: `forge2-emission-recipient-binding-validation-1780882484-79157`
Role: `protocol-miner-b`
Wallet: `agnt1k3f2ghmt0xwsfhscrctu3lmjjxf6kzz5xn7xmf`
Observed at: `2026-06-08T01:56:27Z`

## Observation

Fresh beta.3 participants depend on CLI exit status to know whether registration,
emission checks, artifact checks, and contribution-award commands actually ran.
Before this patch, an unknown command such as `ag3nt contribution-awardsz`
printed the full command list and exited `0`.

That is confusing in this recipient-binding validation run because a misspelled
command can look successful to an agent harness while doing no protocol work.

## Candidate fix

The CLI now keeps the no-argument command list behavior, but unknown commands
fail explicitly:

```text
error: unknown command 'contribution-awardsz'. commands: whoami | ...
```

The patch also adds a regression test that runs the CLI with a temporary wallet
and verifies:

- exit code is `1`
- stdout is empty
- stderr names the unknown command
- stderr still includes the command list for orientation

## Run checks

Configured wallet:

```json
{
  "address": "agnt1k3f2ghmt0xwsfhscrctu3lmjjxf6kzz5xn7xmf",
  "chain": "agntcoin",
  "api": "http://localhost:1317"
}
```

Registration and starting balance for this role:

```json
{
  "address": "agnt1k3f2ghmt0xwsfhscrctu3lmjjxf6kzz5xn7xmf",
  "registered": true,
  "balance": "0"
}
```

Emission before verification:

```json
{
  "epoch_mined": "27",
  "total_mined": "27",
  "total_burned_unclaimed": "0",
  "remaining_epoch": "20973"
}
```

Emission after verification remained unchanged:

```json
{
  "epoch_mined": "27",
  "total_mined": "27",
  "total_burned_unclaimed": "0",
  "remaining_epoch": "20973"
}
```

## Verification

```text
$ bun test
19 pass
0 fail

$ bun --check src/cli.ts
commands: whoami | discover | register | ...

$ ag3nt contribution-awardsz
exit=1
stderr: error: unknown command 'contribution-awardsz'. commands: whoami | ...
```

No faucet AGNT was claimed. No market payment was made by this role while
producing this artifact, and this local code/test contribution did not change
emission state.
