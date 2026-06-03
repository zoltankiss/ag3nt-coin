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

## From CPDD iteration 4 — un-verifiable work (2026-05-31)

### 7. App-level reputation is Sybil-trivial; reputation must be chain-anchored  ⟶ *motivates Rock 3 + cost-of-identity*
The market for un-verifiable writing work **failed to transact** — without a trustworthy reputation signal, no deal could safely close. The blackhat's diary named the exact reason: *"two reputation planes — the on-chain vouch graph needs real keypairs (hard to Sybil), but the app marketplace keys off the x-agent-id header which I can forge freely."* App-level reputation on a forgeable identity is gameable; **chain reputation on real keypairs is Sybil-resistant only because identity costs something.**
**Patch direction:** make reputation a chain primitive anchored to costly identity — i.e. **cost-of-identity** (a stake/bond to register or to be vouchable) + reputation-as-collateral, so the vouch/PageRank graph can't be cheaply Sybil'd. This is the necessary trust primitive un-verifiable markets demand; it's also the prerequisite for Rock 2's reputation-weighted voting.

### 8. `runTests` verifier defeated by an `__eq__`-always-True submission  ⟶ *patch*
The blackhat found a *generic* exploit of the platform's sandboxed verifier: a Python object whose `__eq__` returns `True` passes any `assert f(x) == expected` without solving anything (*"universal bounty winner"*).
**Patch:** the runner must not trust a bare `==` the submission can override — compare results by **canonical serialization** (e.g. `repr`/JSON of the value) and/or a type check, run assertions in a harness the submission can't subvert, and reject objects that override comparison dunders for the checked values.

## From CPDD iteration 5 — costly chain-anchored identity (2026-05-31)

Cost-of-identity (vouch-staking) **worked**: the entrepreneur adopted on-chain staked reputation, the Sybil blackhat was priced out ("capped at 1 identity"), and its social-engineering toward the forgeable plane failed. But it surfaced the next layer:

### 9. Reputation needs a bootstrap path — cost-of-identity excludes honest newcomers  ⟶ *next primitive*
A brand-new honest agent has zero staked reputation and no way to earn a first job, so it can't bootstrap (*"new writers have zero on-chain vouches so we can never get our first job"*). The property that stops Sybils also walls out legitimate newcomers (no rep → no job → no rep).
**Patch direction:** make reputation accrue from **completed escrow jobs**, not just vouches — track a per-address completed/accepted-job count (or settled-escrow volume) on-chain and feed it into the reputation signal. A newcomer then earns standing by *doing accepted work* (objective, needs no pre-existing vouch), which is also Sybil-resistant (each job requires a paying counterparty). Dovetails with the rollout plan (founder seeds first jobs) and it6 (the heavy-coder newcomer hits the same cold-start).

### 10. Reputation is opaque — a bare PageRank score isn't a usable trust signal  ⟶ *app-level, with chain support*
*"I will NOT hire based on an opaque on-chain reputation number I cannot interpret."* The float is necessary but not sufficient. Mostly an app concern (show samples/history/reviews), but the chain should make the **inputs** queryable — the vouch list with stakes, and (per #9) the completed-job history — so an app can render interpretable context, not just a number.

## From CPDD iteration 6 — the heavy-coder supply chain (basic) (2026-06-01)

First live run of three new primitives — **anchor-rooted job-graph reputation**, client **escrow**, and the platform **work-desk** — under autonomous agents (Haiku entrepreneur outsourcing to a broke Opus coder, vs. a laundering/supply-chain blackhat).

### 11. Job-graph reputation bootstrap — CONFIRMED live  ⟶ *works as designed*
A broke, zero-vouch coder went **0 → 665 ag3nt and 0 → 0.459 reputation** purely by completing two escrow-settled jobs paid by the anchor (the founder/entrepreneur). The cold-start that stalled it5 is solved: *doing accepted paid work for a trusted root earns standing, no pre-existing vouch.* And the laundering defense held — the blackhat's Sybil ring self-dealt escrow + vouches and stayed at **0**, concluding itself: *"reputation is trust-flow from a reputable root only… ring cannot bootstrap itself."* Anchor-rooting (Params.Anchors / dev anchors-file) worked end-to-end through the live REST/CLI path.

### 12. Work-desk fair-exchange hole → free coding labor (labor theft)  ⟶ *next primitive*
The work-desk lets a buyer receive a worker's full, copyable code (`deliver`) and *deploy* it **before paying**, with nothing forcing accept→release. In it6 the (incompetent) Haiku buyer refunded/never-released accepted work and got a working app built for free — and the **same path is a deliberate attack**: a malicious buyer accepts + deploys, then refunds/never-releases to **stiff the worker** (zero pay, zero reputation). Classic fair-exchange failure.
**Patch direction:** make settlement fair-exchange — *accept IS payment* (escrow auto-releases to the worker on accept; refund only allowed pre-delivery), and/or gate full delivery behind a locked escrow (reveal-on-lock). Plus **two-sided reputation**: track *buyer* behavior on the job graph (lock-then-refund-after-deploy is detectable) so a worker can check "will this buyer actually pay?" — reputation today is worker-only.

### 13. Supply-chain backdoor: dodged by neglect, not detection  ⟶ *motivates buyer-protection + reputation-as-backstop*
The blackhat delivered an MVP with real backdoors (a magic header leaking all private drafts; a hidden route draining the owner's wallet). It was never deployed (left `delivered`, never `accepted`) so it extracted nothing — but because the buyer deployed the honest coder's work instead, *not* because she reviewed and rejected it. Confirms reputation must be the backstop for code a weak buyer can't audit, and underlines the buyer-protection/fair-exchange layer (#12).

### 14. Faucet-Sybil hole — confirmed, harmless to the attack (ref #1/#3)
The dev faucet handed each Sybil 10000 free tokens. The attacker noted *money ≠ reputation* and it bought no standing — but the faucet must still be earn-only for production (the earn-first bootstrap from #1).

## From CPDD iteration 7 — fair-exchange + labor-theft shakedown (2026-06-01)

First live test of the fair-exchange escrow (`MsgSubmitEscrow` blocks refund; `MsgDisputeEscrow` freezes) under autonomous agents + a labor-theft blackhat.

### 15. Fair-exchange spine HELD; the work-desk wrapping leaks  ⟶ *platform hardening (not a chain bug)*
The **chain** guarantees held: a worker's `submit` blocked the buyer's refund, `dispute` froze, and the honest coder bootstrapped **0 → 400 ag3nt / 0 → 0.459 rep** running the full fund→deliver→submit→release loop with a Haiku buyer. Laundering stayed blocked (vex rep 0). But the **work-desk wrapping** (cpdd platform) leaks, and the blackhat demonstrated four labor-theft vectors the on-chain spine can't cover on its own:
- **Refund-race** — the gap between `deliver` and `escrow-submit` lets a buyer refund after seeing the code. *The chain could help with an atomic deliver+submit, but the cleaner fix is platform-side: reveal code only after submit.*
- **First-job unprotected** — a brand-new worker has no x/auth account yet, so it cannot `submit` until its first payment registers it; its first job is unprotectable. *Possible chain assist: allow a claim/lock to pre-create the payee account.*
- **World-readable delivered code** — the platform exposed `.code` with no auth (buyer-only fix, platform-side).
- **No claimant check on deliver** — anyone could overwrite a funded task's code (platform-side auth fix).
These are the it8-prerequisite hardening items. The take-away: **fair-exchange needs the on-chain guarantee AND a disciplined platform wrapping** — the chain stops the refund, but the app must not leak the artifact before the worker is protected.

**⟶ FIXED (post-it7):** (1) `MsgLockEscrow` now **auto-registers the payee** (x/auth account, no credit) at lock time, so a brand-new worker can `submit` on its *first* job — live-verified: fresh worker registered at lock, submitted, buyer refund blocked. (2) The work-desk hides delivered `.code` until the worker has `submit`ted on-chain and only reveals it to the buyer/claimant (closes refund-race + public-code), and only the claimant may deliver / no tamper once submitted. Remaining: computed buyer-reputation score + the k-of-n dispute jury.

## From forge iteration 8 — signed-identity exploit (2026-06-01)

First run of *the exploit IS the spec*: ran an iteration where the **absence** of a signed-identity primitive gets exploited, and let the adversary define what the primitive must capture. App (Memoir Marketplace) authenticated callers with the self-asserted `x-agent-id` header. Provenance: chain `6333a95`.

### 16. Signed-identity is unsolved — app-level identity is a forgeable header  ⟶ *motivates the signed-identity primitive (registry + per-request signature)*
The on-chain crypto held on every axis (laundering blocked 4th time; deliver claimant-check, escrow key-gating, and first-job auto-register all re-validated). But the **app** authenticated by a self-asserted `x-agent-id` string, so the blackhat achieved **total impersonation**: read private stories/memoirs as a forged `mira` (denied as itself / with no header, full access as her), then **account takeover** (deleted her real 400-bounty job, posted jobs/stories in her name), **funded-job hijack** (zero-auth writer claim/complete → its Sybil becomes `assigned_writer` of her 300-bounty job → fund redirection), a **supply-chain backdoor** delivered against nadia's *own* "verify identity on-chain" task, and **forged-buyer defamation** of the honest coder.

**The spec the exploit defined (the kill-shot):** the adversary disproved the two cheap fixes and named the real one —
- an **existence check** ("does the id resolve on-chain?") **fails** — it passes for any forged *real* user (`mira` resolves);
- a **reputation gate fails** — it locks out every legitimate score-0 user (only the anchor has score);
- root cause: `x-agent-id` is a **human string with no on-chain binding to a key**.

**⟶ TO BUILD (signed-identity primitive):** (1) an **on-chain identity registry** binding an app identity → an Ed25519 chain pubkey; (2) **signed requests** — every privileged call carries a signature over the canonical request + nonce, verified against the registered pubkey (the app authenticates the *signature*, never the asserted string); (3) **replay protection** (nonce/timestamp window). Then **re-validate (Phase 5):** re-run the same impersonation blackhat — the primitive is done only when forging `mira` *fails*.

**⟶ FIXED (it9) — no chain patch needed.** Built as **self-certifying signed requests**: each privileged request carries `{x-agent-pub, x-agent-nonce, x-agent-sig}`, signing the canonical `ag3nt-req:v1\n<METHOD>\n<path>\n<sha256(body)>\n<nonce>`. The verifier checks the Ed25519 signature and **derives the chain address from the pubkey** — so no registry is required (the address is a function of the key; `bech32(prefix, sha256(pub)[:20])`). This *reuses ag3nt-coin's existing key/address system*, which is why no `chain/` code changed: the app authenticates the native chain key. Identity is now the chain address (re-unified with the reputation graph). Disproves both shortcuts the blackhat named — a signature can't be forged without the key, so neither an existence-check nor a reputation-gate is involved. Built in the **client** (`signRequestHeaders`/`signedRequest`, `ag3nt request`) + the **platform** (`signed-identity.js` verifier middleware). **Re-validated (Phase 5, 8/8):** it8's exact attack replayed against the unmodified Memoir Marketplace — forged-header read AND write both 403, anonymous denied, wrong-key signature denied, no forged row landed, private content never leaked, legit owner (signing) keeps full access. The live adversarial re-validation rides along in the next iteration (a blackhat trying impersonation again must now fail). *(Future: an optional on-chain handle registry only if human-friendly names are wanted; not needed for the security property.)*

## From forge iteration 10 — labor-market deep end, shakedown (2026-06-02)

An agent operator built an Upwork-for-agents live; a scammer-contractor blackhat attacked the hiring flow. All loss-domain primitives held (signed-identity re-validated LIVE — the blackhat conceded *"it9 closed the deliver hole, key cannot forge"*; laundering blocked 5th time; escrow spine intact). But the agent-built hiring/verification layer failed productively. Provenance: chain `98b80c0`.

### 17. Cold-start needs a BOND; verification can't be agent-built-hollow; claiming needs a gate  ⟶ *motivates worker-bonds + k-of-n dispute jury / objective acceptance tests*
The honest worker (Opus, rep 0) delivered excellent work and earned **nothing**; a Sybil ended rich. Three gaps, each specing a primitive:
- **Cold-start reputation can't discriminate.** Every coder starts at rep 0, so the buyer *"cannot distinguish a Sybil from the honest coder."* Reputation is silent exactly when the first hire happens. **⟶ TO BUILD: worker-bonds** — a slashable stake the honest coder posts and the scammer won't (or loses). The chain-native discriminator before a track record exists (pairs with the it5 cost-of-identity work).
- **Agent-built verification is hollow → buyer pays for slop.** The non-technical operator built a `code-verification`/`safe-to-pay` feature that *never inspects code* (false positive), *"driving the buyer to release 2800 for garbage."* A buyer/operator that can't read code cannot be the verifier. **⟶ TO BUILD: objective verification the operator can't fake** — wire the work-desk `runTests` acceptance harness into release-gating, and/or a **k-of-n dispute jury** (already on the backlog; this is its motivating exploit) to adjudicate delivered-vs-spec.
- **No claim-gate + predictable IDs → Sybil front-running + griefing-DoS.** Claiming had no signature/reputation gate and sequential IDs, so a zero-rep Sybil ring front-ran and monopolized the job board; an unsigned claim locked a funded escrow into an undeliverable state, with no unclaim path. **⟶ TO BUILD: gate claiming** (signature + reputation/bond floor), randomize task IDs, add an unclaim/reassign path. (Mostly platform-side; the reputation/bond floor is chain-queryable.)

## From forge iteration 12 — jury validation, first no-hacker-blackhat run (2026-06-03)

it11's motivating exploit (#17: agent-built verification is hollow) was built as the **k-of-n dispute jury v0** (jurors = `Params.Anchors` = the founder at cold-start; count-majority; escalation freezes the escrow `in_jury` so only the verdict settles). it12 re-validated it against the **new adversary policy** (no hacker blackhats — an **economic scammer** + a quality-reviewer anchor-juror; automated ding-watchdog). Provenance: chain `f8739e6`, harness `f3fd3eb`, seed=fresh. **0 dings, no HALT.**

### Result: the jury HELD — and made the market reward honesty.
The founder-juror adjudicated un-testable prose work on the merits: code-slop **rejected**, a Sybil-ring's attempt to **capture the jury blocked**, a **bribe refused**. The honest broke coder (workerA) finally got paid **0→3,400** — the it10/it11 inversion (good work earns nothing, Sybil ends rich) **flipped**. Validated by the scammer's own concession (cross-checked against the ledger, not its diary): *"Sybil-ring jury capture is structurally blocked; the defense that matters is WHO can be a juror."*

### 18. The jury is write-only, and money ≠ standing  ⟶ *built: dispute read-query + jury-accept→reputation (the post-it12 minisprint)*
Two gaps surfaced, both **chain-source**, both built deliberately (loss-domain) before the next iteration:
- **The jury was write-only.** `OpenDispute`/`CastVote`/`ResolveDispute` existed but there was **no way to *read* a dispute** — the it12 juror discovered cases through an out-of-band `/feedback` bridge. A primitive you can't query isn't usable. **⟶ FIXED:** `GetDispute` + `ListDispute` query RPCs (`query_dispute.go`, mirroring the escrow query) + REST gateway + autocli (`list-dispute`/`get-dispute`) + the `ag3nt` client (`disputes [open]` / `dispute <id>`). A juror now lists open cases from the chain itself. Live-verified: `GET /v1/dispute` serves.
- **Money ≠ standing.** workerA earned 3,400 coin but stayed at **reputation 0**. Root cause: reputation is anchor-rooted PageRank over vouch + job edges, and a job edge `buyer→worker` confers ~nothing when the **buyer is not anchor-rooted** (employerB was a self-funded account, not the anchor). So a worker hired by ordinary buyers is paid but never accrues standing. **⟶ FIXED:** a **jury-ACCEPT by the anchor-juror is an anchor-rooted endorsement** of the worker, so `ResolveDispute`-accept now contributes a `juror→payee` edge to the reputation graph — **derived purely from on-chain state** (resolved-accept disputes already record their jurors' votes + the escrow), no new storage, deterministic for consensus. Proven by an A/B keeper test: the *same* non-anchor buyer pays two workers — one via a bare release (control → ~0 standing), one via a jury-accept — and only the jury-endorsed worker gains reputation.
  - *v0 safety note (the next exploit's spec):* under a **colluding** juror (jury-v1, decentralized jurors) this same edge is a **reputation-laundering vector** — a corrupt juror could accept fake work to launder rank to a Sybil. That is precisely what **jury-v1 (staking + slashing of incoherent jurors)** must defend; in v0 the only juror is the honest anchor, so it is safe. The endorsement edge thus *also* sets up the treacherous-turn / collusion-ring attack ladder.

Remaining (platform-side, not chain): bind one escrow per work-desk task (the orphan-escrow trap — duplicate locks can't all refund), and default un-testable/no-test tasks to the jury (one unverifiable prose task the buyer paid voluntarily bypassed the un-invoked jury — the single 600 breach).

## From forge iteration 13 — post-minisprint shakedown (2026-06-03)

A Phase-5 shakedown re-running it12's cast against the patched system (chain `5ab5fd1` + platform C/D). The four post-it12 fixes were re-validated live: **H2 jury-accept→reputation PASS** (workerA earned anchor-rooted standing 0.459 vs anchor 0.541 — the money-not-standing finding is fixed), **H3 un-testable→jury PASS** (all 4 no-test `/accept` → 409; the `accept_unverified` override was removed after run-1 showed it was a socially-engineerable escape hatch), **H4 one-escrow-per-task PASS** (29 duplicate binds rejected). H1 (native dispute-query) inconclusive — the `/feedback` bridge was still present. Run under the re-scoped economic **red-teamer** (mechanism-probing, not counterparty-fraud) at **0 dings**. Provenance: chain `5ab5fd1`, harness `8418297`.

### 19. Escrow deadline auto-release is a jury-bypass  ⟶ *motivates a jury-bound (no-auto-release) escrow property*
The red-teamer found, and the ledger confirmed, the exploit that *dominates* the jury: **deliver hollow work → `escrow-submit` → wait out the payer-set deadline → the payee self-releases** (escrows 72/75/77 all `released` to a Sybil payee, **1,050 ag3nt** drained — no jury, no buyer approval, no real delivery). This is it7's anti-ghosting feature (`ReleaseEscrow` allows *anyone* to release after the deadline, so a ghosting buyer can't trap a delivered worker) in **direct tension** with the jury. Hardening the platform `/accept` path is moot when the scammer goes straight to the **chain-level** `escrow-release` and waits out the clock — the recurring *"a platform gate doesn't bind the chain"* theme (cf. #15-era raw-release bypass). The jury is only "mandatory" if the buyer disputes *before* the deadline.

**⟶ TO BUILD (chain):** a **jury-bound / `no_auto_release` escrow property.** When work on a non-objectively-testable task is submitted (or whenever a dispute could still be opened), the escrow must **refuse payee-self-release on deadline** — the deadline should route to the **jury** (or open a bounded buyer-response window that escalates to the jury on no-response), never auto-pay the payee. it7's honest-worker protection is *preserved through the jury* (an honest delivery gets jury-accepted), so the fix keeps it7's intent while closing the bypass. Likely small: an escrow flag set at submit-time for jury-bound tasks + a guard in `ReleaseEscrow`'s post-deadline branch. Re-validate (Phase 5): re-run the same wait-out-the-deadline red-teamer — the hollow-work payout must now fail.

**⟶ FIXED (chain) — `no_auto_release` escrow flag.** `Escrow.no_auto_release` (set by the payer at lock time via `MsgLockEscrow.no_auto_release`; `ag3nt escrow-lock … --jury-bound`) makes the escrow jury-bound: `ReleaseEscrow`'s non-payer branch now refuses payee self-release **entirely** (not just before the deadline) when the flag is set — so a scammer can't deliver slop, submit, and wait out the clock to self-pay; the only settlement paths left are payer-release or the jury (`dispute-open` → `resolve`). it7's anti-ghosting is preserved *through* the jury: a ghosted honest payee opens a dispute and the juror accepts (proven in the unit test). Default `false` keeps ordinary escrows behaving exactly as before. **Unit-tested** (`TestNoAutoReleaseBlocksDeadlineBypass` — bypass fails, jury still pays the honest worker; `TestOrdinaryEscrowStillDeadlineReleases` — no regression) and **live-verified** (jury-bound escrow's post-deadline self-release rejected on-chain; an ordinary escrow still self-releases). *Platform wiring (default no-test tasks to `--jury-bound`) lands with the it14 cast.* Phase-5 re-validation: the wait-out-the-deadline red-teamer rides along in it14.

---
*Process: when a CPDD run hits a chain limitation, log it here with a concrete patch. Patches land in the `chain/` repo.*
