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

## From CPDD iteration 3 — the compute market (2026-05-31)

### 4. No native escrow → apps reinvent custodial escrow (the honeypot)  ⟶ *implementing*
The entrepreneur built a coding-bounty market but, lacking an escrow primitive, **routed all in-flight funds through its own wallet via a CLI bridge** — a centralized custodial hot-wallet, the exact target the blackhat named (*"real target is app escrow once built"*). Every marketplace would rebuild this, each a fresh chance to get custody wrong.
**Patch:** a native escrow / conditional-payment primitive in x/agntcoin — `MsgLockEscrow{payer,payee,amount,ref,dispute_seconds}` (funds held by the module, not any wallet), `MsgReleaseEscrow` (payer anytime, or anyone after the deadline → payee), `MsgRefundEscrow` (payer within the window → refund). The Base `JobEscrow` model ported to Cosmos; **no oracle needed for v1** (buyer-release + dispute-window). Removes the custodial honeypot and makes trustless settlement a one-call primitive. Design rationale + the release-condition options (why *not* Chainlink; native k-of-n as the dispute backstop) in [`compute-market-design.md`](compute-market-design.md).

### 2 (revisited). Transfer memo/ref  ⟶ *implementing*
Now being added: `memo` on `MsgTransfer` (surfaced in the event) + the escrow `ref` field — so each payment is on-chain auditable and tied to its job. (The event itself was patched in iteration 2.)

### 5. Account-creation spam guard (side effect of the onboarding patch)  ⟶ *implementing*
Receive-auto-registers (finding #1) introduced a cheap account-creation vector: a sender can spawn unlimited `x/auth` accounts via dust transfers.
**Patch:** require a minimum credited amount when the recipient/payee does not yet exist (a `MinNewAccountCredit` constant). Bounds dust-spam state bloat.

### 6. Collapse the two-balance confusion  ⟶ *noted (client/UX)*
Vex's recon flagged it: an onboarded agent has both module `agntcoin` and bank `stake`/`token` (from the dev-faucet bootstrap). Now that receive-auto-registers creates the `x/auth` account, the production flow is earn-first and never needs the bank tokens — the agent's canonical balance is its `agntcoin`. **Action:** surface `agntcoin` as the single balance in the client; keep the dev-faucet only as a sandbox convenience. (Fully dropping the bank bootstrap for the *self-faucet* path still needs the ante-deadlock fix — separate, deferred.)

### Mechanics learning (not a chain patch)
**Pay-on-test-pass** is the compute market's killer mechanic and structurally enforces TDD; **objective verification and reputation are complementary** (tests = fast path, reputation/k-of-n = backstop for what tests can't capture). Full write-up: [`compute-market-design.md`](compute-market-design.md). Implication for experiments: an iteration with *un-verifiable* work is what will finally force the reputation path.

---
*Process: when a CPDD run hits a chain limitation, log it here with a concrete patch. Patches land in the `chain/` repo.*
