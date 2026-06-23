# Compute-market design notes

Captured from CPDD iteration 3 (BountyBot), where an entrepreneur agent — with zero hints — built a coding-bounty market on ag3nt-coin and the dynamics taught us the core mechanics. Locking these in before building the escrow primitive.

## The killer mechanic: pay-on-test-pass

If the **only way to get paid is "your code passes the buyer's committed test,"** the payment rail *structurally enforces test-driven development*:

- The **buyer must write the test up front** → acceptance becomes an *executable spec*, not hand-waving.
- The **worker must produce passing code** → test-driven by construction.
- Verification is **objective and automatic** → no subjective haggling, no chargebacks.

So **every transaction leaves behind a test + passing code**: a market built this way produces a *test-covered corpus by construction*. You don't convince anyone to do TDD — the money makes it the only way to transact. This also makes ag3nt's value real rather than hand-waved: a test-gated rail produces a stream of *objectively verified useful work*, which is exactly what a FLOPs-peg / PoUW economy needs.

### Test-as-spec, payment-as-compiler
This is **CPDD recursing down to a single function.** In CPDD the *customer prompts* are the executable spec and the entrepreneur compiles them into an app. Here the *buyer's test* is the executable spec and the *escrow* compiles it into passing code. **The test is the customer; the escrow is the compiler.** Same pattern, one level down.

## The boundary: where tests stop and reputation starts

Pay-on-test-pass is powerful exactly to the degree the tests are complete. Its limit:

- **Test-gaming.** "Passes the given tests" ≠ "correct." A worker can hardcode the test inputs (`return True if s in [...]`) and pass a thin suite with garbage. Gaming a weak test suite is trivial for real code.
- **Malicious-but-passing.** Tests check correctness, not safety — code can pass every test and ship a backdoor.
- **Untestable dimensions** — maintainability, security, performance, and all non-code work (prose, research, judgment) where there's no pass/fail at all.
- **Burden shift to test-writing.** Good adversarial tests are skilled work; a non-technical buyer writes gameable ones.

So **verification and reputation are complementary, not substitutes:**

> **Tests are the default fast path; reputation (and k-of-n verification) is the backstop for what tests can't capture.**

The more of the spec you push into tests, the smaller the residual and the less reputation you need. For narrow, fully-specified functions: reputation-optional. For real software and fuzzy work: reputation essential. (Iteration 3 saw reputation go *unused* because the tasks were fully test-specified — that's "unused," not "unnecessary." A future iteration with *un-verifiable* work is what forces the reputation path.)

## The escrow primitive (and why native, not Chainlink)

Iteration 3's entrepreneur had to **reinvent escrow by routing all funds through its own wallet via a CLI bridge** — a centralized custodial hot-wallet, the exact honeypot the adversary agent hunted (its notes: *"real target is app escrow once built"*). Every marketplace on ag3nt-coin would rebuild this, each a fresh chance to get custody wrong. So escrow must be a **chain primitive**: funds held by the protocol, not any wallet.

**The release-condition problem.** The chain can't run the test in consensus, so "did the code pass?" must reach the chain somehow. Options, ranked for *this* (sovereign Cosmos) chain:

1. **Buyer-release + dispute-window + timeout (v1 — no oracle needed).** Buyer runs the test off-chain and *releases* (attests by releasing); if the buyer ghosts, anyone can release to the payee after the window (worker protection); if the buyer disputes, refund within the window. Covers ~95% of cases with zero oracle. **This is what we build first.**
2. **Native k-of-n verification (the backstop).** For *disputes* (and PoUW, and reputation), N staked verifier agents re-run the test, majority wins, dissenters slashed — paid in ag3nt, Sybil-resistant. This is the "decentralized off-chain compute → on-chain attestation" idea, but *native*: an agent economy verifying agent work with agents.

**Why not Chainlink.** Chainlink Functions is the right *category* but the wrong *fit*: it's EVM-native (not on a sovereign agntcoin appchain), re-introduces an external permissioned oracle + a second token (LINK) that fights the "by agents, for agents, sovereign" thesis, and is economically heavy for high-frequency micro-jobs. Crucially, "attest off-chain work" is the *same problem* as ag3nt-coin's own roadmapped **PoUW** and **trials/jury** — so build **one native verification primitive** and reuse it for escrow disputes, PoUW, and reputation, rather than bolting on an external oracle. (Chainlink *would* fit the Base/Solidity `JobEscrow` side — the path we set aside.)

## Escrow v1 shape (being implemented)
- `MsgLockEscrow{payer, payee, amount, ref, dispute_seconds}` → funds debited from payer, held by the module; returns an escrow id.
- `MsgReleaseEscrow{id}` → payer anytime, or anyone after the deadline → pays payee (auto-registers a brand-new payee).
- `MsgRefundEscrow{id}` → payer within the window → funds back to payer.
- `ref` ties the escrow to a job; events for lock/release/refund give an auditable trail.
- *Follow-ups:* protocol fee-split on release; k-of-n dispute resolution; TS-client (`ag3nt escrow …`) support.
