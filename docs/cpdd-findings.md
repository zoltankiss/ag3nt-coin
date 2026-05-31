# ag3nt-coin findings from CPDD runs

Issues surfaced by *dogfooding* ag3nt-coin inside [CPDD](https://github.com/zoltankiss/cpdd) experiments. Each is a real defect/gap the experiments exposed, with a proposed patch. This is the compounding dividend of using ag3nt-coin as the money layer for CPDD: every run is also a test of the chain.

## From CPDD iteration 2 (2026-05-31)

### 1. Onboarding is not protocol-native gasless yet
A brand-new keypair has no `x/auth` account, so it can't sign *anything* — including the module faucet that would fund it. The CPDD client works around this by first hitting the **dev faucet** (Ignite's HTTP faucet) to create the account, *then* registering and claiming the module faucet. That's a 3-step dance with an external dependency, contradicting the SPEC's "the chain mints into existence on the agent's own signature" claim.
- **Patch:** make `MsgFaucet` (or a combined `MsgRegister`+claim) create the `x/auth` BaseAccount for the signer if absent, so a penniless agent self-onboards in **one signed tx** with no dev-faucet dependency. (Requires the ante to admit the first tx from an accountless-but-validly-signed address, or a custom onboarding ante path.)

### 2. Transfers are not auditable or referenceable
`MsgTransfer` is `{creator, to_addr, amount}` only — **no memo/reference**, and `msg_server_transfer.go` emits **no event** (verified: no `EmitEvent`/`EmitTypedEvent`). So you can't tie a payment to "project 5", and you can't reconstruct a transfer history without scraping raw txs.
- **Patch:** add an optional `memo` (or `ref`) field to `MsgTransfer`, and emit a typed `EventTransfer{from,to,amount,memo}`. High value for marketplace settlement (payment → invoice correlation) and for the CPDD `memory.jsonl` debugging story (clean, queryable money history).

### 3. One-shot faucet blocks balance resets on a persistent address
`Account.faucet_claimed` makes the faucet one-shot forever. Fine for production, but it means you **cannot reset a persistent identity's balance** between experiment runs without wiping the entire chain (re-genesis). That couples "reset balances" to "reset reputation," which we don't always want.
- **Patch (test/dev only):** a re-claimable or cooldown faucet, or a dev-gated `MsgDevReset`/`MsgDevMint`, so a persistent persona (same keypair across runs) can be re-funded without losing its accumulated vouch graph / reputation. Gate behind a genesis/config flag so it never ships to a real network.

---
*Process: when a CPDD run hits a chain limitation, log it here with a concrete patch. Patches land in the `chain/` repo.*
