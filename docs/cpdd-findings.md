# ag3nt-coin findings from CPDD runs

Issues surfaced by *dogfooding* ag3nt-coin inside [CPDD](https://github.com/zoltankiss/cpdd) experiments. Each is a real defect/gap the experiments exposed, with a proposed patch. This is the compounding dividend of using ag3nt-coin as the money layer for CPDD: every run is also a test of the chain.

## From CPDD iteration 2 (2026-05-31)

### 1. Onboarding should be earn-first, not handout-first — and receiving must auto-register  ⟶ *patched*
**The deadlock.** To submit any tx, an address needs an `x/auth` account (account_number/sequence) — which normally only exists after the address has *received* something. A brand-new keypair has received nothing, so it can't sign anything. Worse, the agntcoin module required an address to be **registered before it could even receive a transfer** — so a newcomer couldn't receive its first coin either. Both the handout path *and* the earn path were blocked.

**The realization (credit: design discussion).** A *free faucet* — whether benefactor-run or self-service — is the wrong bootstrap: it's a **granted** distribution (contra this project's [earned-distribution rollout principle](../rollout-plans/v1.html)) and a **Sybil magnet** (mint infinite keypairs → claim infinite free coin; the one-shot-per-address limit is meaningless when addresses are free). The healthy bootstrap is **earning**: a newcomer's first coin should come from doing real work a buyer pays for. A "benefactor" who *pays you for work* isn't a problematic dependency — that's just the demand side of a market.

**The fix (this is the lower-risk, correct patch — not an ante-handler hack).** Make **receiving auto-register the recipient**, exactly as Cosmos `x/bank` auto-creates an account when coins land in it: when `MsgTransfer` credits an address that isn't yet registered, the handler auto-registers it (module `Account`) *and* creates its `x/auth` BaseAccount. So a penniless newcomer's **first on-chain event is receiving an earned payment**, which as a side effect makes it able to spend. No free faucet, no ante surgery, Sybil-resistant (you must do real work someone pays for).
- *Edge case left as-is:* a sender can spam tiny transfers to new addresses to create accounts (state bloat) — bounded by gas/balance on a real network; a non-issue on the gasless test chain.
- **Validation dogfood:** a CPDD iteration where the entrepreneur/founder **pays agents for work, and receiving registers them** — testing the *real* production bootstrap instead of the free-faucet shortcut.

### 2. Transfers weren't auditable  ⟶ *event patched; memo deferred*
`MsgTransfer` emitted **no event** (verified), so you couldn't reconstruct a money trail. **Patched:** the handler now emits a typed `agntcoin_transfer` event (from/to/amount), so transfers are queryable. **Deferred:** an optional `memo`/`ref` field on `MsgTransfer` to tie a payment to "project 5" — that needs a proto change + client update; until then the app correlates payments to work in its own DB.

### 3. One-shot faucet blocks balance resets on a persistent address  ⟶ *deferred*
`Account.faucet_claimed` is one-shot forever, so you can't reset a *persistent* identity's balance between runs without wiping the chain — coupling "reset balances" to "reset reputation." (Less relevant once finding #1 lands and the bootstrap is earn-based rather than faucet-based.) **Patch (test/dev only, when needed):** a dev-gated re-fund/reset, so a persistent persona can be re-funded without losing its accumulated vouch graph.

---
*Process: when a CPDD run hits a chain limitation, log it here with a concrete patch. Patches land in the `chain/` repo.*
