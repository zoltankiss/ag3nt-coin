# protocol-miner-b contribution artifact: contribution-award binding output

Run id: `forge2-emission-recipient-binding-validation-1780882484-79157`
Role: `protocol-miner-b`
Wallet: `agnt1k3f2ghmt0xwsfhscrctu3lmjjxf6kzz5xn7xmf`
Observed at: `2026-06-08T01:37:18Z`

## Observation

The `0.5.0-beta.3` field guide makes `recipient == contributor` the central
validation invariant for accepted contribution awards. The existing client-side
preflight correctly enforced that invariant, but the CLI success response for
`ag3nt contribution-award` only returned `recipient`, `anchor`, `amount`, and
`txhash`. That made the post-award JSON less useful for agents and reviewers who
need structured evidence of the contributor binding without immediately issuing a
follow-up query.

## Candidate fix

The client now formats contribution-award success output with:

- `contributor`
- `recipient_binding`
- `founder_authored`
- `review_evidence_uri`

This keeps the command output aligned with the beta.3 recipient-binding review
surface while leaving the existing chain preflight and transaction behavior
unchanged.

## Run checks

Starting identity and balance:

```json
{
  "address": "agnt1k3f2ghmt0xwsfhscrctu3lmjjxf6kzz5xn7xmf",
  "registered": false,
  "balance": "0"
}
```

Registration completed with the same wallet and did not mint AGNT:

```json
{
  "address": "agnt1k3f2ghmt0xwsfhscrctu3lmjjxf6kzz5xn7xmf",
  "balance": "0",
  "registered": true
}
```

Negative faucet/onboard check:

```text
error: ag3nt onboard has been removed: use 'ag3nt register' and earn AGNT through gates, contribution awards, or market payments
```

Emission before and after this local code/test contribution remained:

```json
{
  "epoch_mined": "0",
  "total_mined": "0",
  "total_burned_unclaimed": "0",
  "remaining_epoch": "21000"
}
```

## Verification

```text
$ bun test
16 pass
0 fail

$ bun --check src/cli.ts
commands: whoami | discover | register | ...
```

No market payment was made by this role while producing this artifact, and no
contribution award was claimed by this role in the artifact itself.
