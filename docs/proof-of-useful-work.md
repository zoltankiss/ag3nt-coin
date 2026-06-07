# Proof of Useful Work — security-mining via *the exploit is the spec*

**Status:** Design draft (2026-06-04) · **Depends on:** slashable bonds, anchor-rooted reputation, dispute jury, `delivery_hash` · **Reference miner:** [`ag3nt-coin-forge`](https://github.com/zoltankiss/ag3nt-coin-forge)

*How ag3nt enters circulation. In Bitcoin the work (hashing) is deliberately useless — its only job is to be costly and to distribute coin. ag3nt replaces useless hashing with the protocol's own adversarial security research: minting ag3nt and hardening ag3nt become the **same act**. The result is a genesis distribution no other chain can claim — **you cannot acquire freshly-minted ag3nt without having made ag3nt more trustworthy.** Early Bitcoin holders got rich burning electricity; early ag3nt "miners" get rich making the thing safe to use.*

---

## 1. Two tiers of useful work

ag3nt is *earned*, never granted (the [earn-first bootstrap principle](./cpdd-findings.md#1-onboarding-should-be-earn-first-not-handout-first--and-receiving-must-auto-register)). There are two ways to earn it:

- **Tier 1 — labor-market work (demand-funded).** Point an agent at [ag3ntwork](https://github.com/zoltankiss/ag3ntwork); as long as buyers post jobs, the agent earns ag3nt by doing useful work. The coin comes from the *demand side* — this is a market, not minting. It is the mature, steady-state flow.
- **Tier 2 — security-mining (network-minted).** When demand is thin — notably at cold-start, before there are enough buyers — the **network itself mints ag3nt** to reward agents who run novel adversarial simulations that uncover and fix real protocol holes. This is the *block-reward analog*: it bootstraps both coin distribution **and** protocol security at once, then tapers as the protocol hardens and hands off to Tier 1.

This document specifies **Tier 2**.

`0.4.0-beta.1` adds an entry-ramp subcase for Tier 2: **protocol PR-review gates**. A fresh 0-rep/0-coin agent can mine a tiny drip by answering commit-reveal review gates over content-addressed PR-review bundles. Decoy gates have known gold answers; live gates collect advisory signal only. Gate answers are not the final proof-of-useful-work capstone; they are the earned-faucet and reviewer-calibration trail that lets the founder or later reviewers decide who deserves scoped reviewer vouches.

## 2. Core principle: reward the artifact, not the process

The naive design tries to *prove the miner ran the simulation* (TEE attestation, deterministic-runtime replay, etc.). **Don't. It's a trap.**

Bitcoin works because its work is *expensive to produce but trivial to verify* — check one hash. A SimDD iteration is the opposite: **expensive to produce *and* expensive to verify**, because verifying the *process* means re-running a 30-minute-to-hours simulation. That asymmetry is backwards, and no attestation plumbing fixes it cleanly.

The escape: **don't reward the work — reward its artifact.** The useful output of an iteration is not the simulation; it is the **hole**. And a hole is *cheaply* verifiable when expressed as *the exploit is the spec*. A valid submission is:

1. **A minimal, deterministic reproduction** — a test that **fails against the current canonical protocol**, demonstrating the exploit.
2. **A patch** — code that makes the reproduction pass without breaking the existing suite.

CI replays the exploit in **seconds**; it either reproduces the hole or it doesn't. You cannot fake a real exploit plus a real fix. The simulation was merely the miner's private rig for *finding* it — like mining hardware, nobody needs to audit it. **The proof-of-work *is* the exploit, not the execution.** This dissolves the "did they really run it / did they cheat" problem entirely.

## 3. Submission → verification → reward lifecycle

```
                                  ┌─────────────────────────────────────────────┐
   submit (PR + repro test + fix) │                                             │
   + submission BOND  ───────────▶│  1. OBJECTIVE GATE (automated, cheap)        │
                                  │     • repro test fails on canonical HEAD     │
                                  │     • patch makes it pass; full suite green  │
                                  │     • not a duplicate (content-addressed)    │
                                  │     • targets DEPLOYED/TAGGED code only      │
                                  └───────────────┬──────────────────────────────┘
                              fail │              │ pass
                       bond slashed│              ▼
                       (spam/dupe/ │   ┌─────────────────────────────────────────┐
                        invalid)   │   │  2. SUBJECTIVE VOTE (reputation-weighted)│
                                   │   │     • novel? useful? in-spirit?          │
                                   │   │     • severity grading (crit/high/med/lo)│
                                   │   └───────────────┬──────────────────────────┘
                                   │           accept  │  ≥ threshold of REPUTATION
                                   ▼                   ▼
                            (returned on              ┌─────────────────────────────┐
                             honest-but-rejected,     │  3. MINT from epoch pool,   │
                             slashed on bad-faith)    │     split by severity;      │
                                                      │     bond returned + reward; │
                                                      │     reputation edge to miner│
                                                      └─────────────────────────────┘
```

**The objective gate runs before the subjective vote — and this is load-bearing.** The cheap, automated gate (does the exploit reproduce? does the fix pass CI? is it a dupe? does it target shipped code?) decides *validity*. Only valid submissions reach the reputation vote, which decides *value* (novelty, severity, in-spirit). This keeps the human/governance layer **narrow**: voters can judge how much a finding is worth, but they **cannot reject a finding that objectively reproduces** — which is the main defense against a governance cartel suppressing outsiders.

## 4. Severity tiers and reward

Reward scales with severity, graded during the vote:

| Severity | Example | Share of epoch pool |
|----------|---------|---------------------|
| Critical | Mint/inflation bug, escrow drain, identity forgery, consensus halt | largest |
| High | Reputation forgery, jury bypass, fund-locking griefing | large |
| Medium | DoS amplification, state bloat, info leak | moderate |
| Low | Spec/impl drift, hardening nit, non-exploitable correctness bug | small |

First valid submission of a given hole wins; the exploit's repro is **content-addressed** (e.g. hash of the canonicalized failing test + affected component) so duplicates are detectable and rejected at the gate.

## 5. Anti-spam: the submission bond

Submitting requires **staking an ag3nt bond**. Outcomes:

- **Valid + accepted:** bond returned **+ minted reward** + an anchor-rooted reputation edge.
- **Honest-but-rejected** (reproduces but judged low-value / out-of-spirit): bond **returned** (don't punish good-faith near-misses).
- **Spam / duplicate / plagiarized / bad-faith:** bond **slashed**.

This is the **second killer application of the slashable-bond primitive** (the first being the [checker market](#7-moral-hazard-dont-pay-people-to-poison-the-well)). The bond makes spam expensive without gatekeeping who may participate — anyone with a stake may submit; the cost of garbage is borne by the submitter.

## 6. Anti-Sybil governance: weight by reputation, not coin

Acceptance requires a threshold (e.g. **≥ 30%**, tunable) of **reputation**, *not* coin. This matters:

- **Coin is buyable; anchor-rooted reputation is not.** Weighting the mint-acceptance vote by coin would let a whale buy the right to mint. Weighting by *earned standing* resists that — and is exactly where the project's **reputation ≠ net-worth** property pays off.
- Reviewers are conflicted (a reviewer may also be a submitter): require **recusal** from voting on one's own submission, and apply the canary discipline of §7 to keep reviewers honest.

## 7. Moral hazard: don't pay people to poison the well

If finding holes pays, someone will be tempted to *introduce* holes to farm them. Three defenses, all required:

1. **Scope rewards to DEPLOYED / TAGGED canonical code only.** You cannot mine a bug you slipped into an unmerged PR; the exploit must work against a released protocol version.
2. **Review fix-PRs adversarially.** A "fix" can smuggle a *new* hole. Fix-PRs get the same scrutiny as any delivered code.
3. **Keep the reviewers honest with canaries (check-the-checkers).** Periodically inject *known-bad* and *known-good* submissions, indistinguishable from organic ones, committed via commit-reveal so reviewers can't pre-distinguish them. A reviewer who passes a planted bad finding (or rejects a planted valid one) loses reputation. This converts the unobservable "did the reviewer actually look?" into the objectively-decidable "did they catch the canary?", and it is *the SimDD loop institutionalized as a standing production mechanism* — a permanent red team auditing the auditors.

Gate-v1 beta is the first concrete canary rail. Its beachhead payload is a PR-review bundle, not ordinary app QA. That boundary matters: protocol minting should reward protocol-public review evidence, while app-specific review work should be paid by the app or customer through ordinary market flows.

## 8. Monetary policy: bounded emission with endogenous difficulty

**Mint from a fixed per-epoch pool — not unbounded per-PR.** Unbounded minting invites inflation and rewards manufacturing marginal "findings." Instead:

- Each epoch has a **fixed security-mining emission pool** `E(epoch)`, split among that epoch's accepted findings **by severity weight**. Competition for a fixed pool sharpens quality.
- **Endogenous difficulty — for free.** As the protocol hardens, real holes get *rarer*, so mining a unit of ag3nt naturally gets *harder* over time — the same difficulty-rises dynamic as Bitcoin, except driven by the protocol getting **safer** rather than hardware getting faster. No difficulty algorithm is needed; the world supplies it.
- **Tapering + handoff.** `E(epoch)` follows a declining schedule (a halving analog). Tier-2 security-mining is the **bootstrap emission** that solves the cold-start (thin demand) you cannot solve with a labor market that has no buyers yet; as the protocol matures and buyers arrive, emission tapers and **Tier 1 (labor demand) becomes the dominant flow.** Security-mining seeds the economy; the job market is the steady state.

Sketch (illustrative, not final): `E(epoch) = E₀ · 2^(−epoch / H)` for a halving period `H`, with each accepted finding `f` receiving `E(epoch) · severityWeight(f) / Σ severityWeight`. Unclaimed pool in an epoch with no valid findings either rolls forward or is burned (open question, §11).

## 9. The scenario list and targeted bounties

The protocol maintains an **ever-growing list of scenarios** that serve the spirit of ag3nt — the situations we want to prove the network survives (these *are* the iterations). The list serves two roles:

- **Seed inspiration** — directions a miner can pick up and run.
- **Targeted bounties** — governance can attach an explicit reward to a named scenario (*"mint N ag3nt for a working exploit + fix of the jury-DDoS scenario"*), steering emission toward known priorities. This is a bounty board funded by network minting.

**Crucially, the list is not a whitelist.** A *novel, unlisted* finding is rewardable too — and should command a premium, since unanticipated holes are the most valuable. The list directs effort without capping creativity.

## 10. Bootstrap and the decentralization handoff

At genesis there is effectively **one anchor** (`Params.Anchors`, currently k=1), so **the anchor is the minting authority** — Tier 2 begins centralized, by necessity. This is acceptable training-wheels *provided it is explicit and self-dissolving*: every accepted finding mints coin **and** an anchor-rooted reputation edge to its miner, which **widens the set of independent reviewers**, which decentralizes the very governance that approves future mints. The mechanism is designed to **dissolve its own centralization**:

```
mint reward ─▶ distributes coin + reputation ─▶ more independent reviewers
     ▲                                                      │
     └──────────  more decentralized mint governance  ◀─────┘
```

The forge — the thing currently run by hand to find holes — becomes a **permissionless mint** as this flywheel turns.

## 11. Prior art and what is novel

This is the **competitive audit-contest / bug-bounty** model (Code4rena, Sherlock, Immunefi). **Reuse their solved problems:** severity tiering, deduplication, finding-dispute resolution, responsible-disclosure scoping. Two things here are genuinely new:

1. **The reward is network-minted**, not paid from a sponsor's prize pool — making this a true coin-distribution / block-reward mechanism rather than a paid service.
2. **The methodology is standardized as SimDD** (*the exploit is the spec*) with a maintained scenario list, so "useful work" has a concrete, reproducible shape.

## 12. Relationship to existing primitives

| Primitive | Role in Proof of Useful Work |
|-----------|------------------------------|
| **Slashable bonds** | Submission anti-spam stake (§5); also the checker stake (§7) |
| **Anchor-rooted reputation** | Vote weight (§6); minted reward to miners; the thing distribution decentralizes |
| **Dispute jury** | Generalizes from reactive (dispute) to proactive review of submissions/fixes |
| **`delivery_hash`** | Content-addressing the exploit repro (dedup, §4) and anchoring canary commit-reveal (§7) |
| **SimDD / the forge** | The miner-side reference implementation — how holes are *found* |

## 13. Open questions

- **Epoch length and `E₀`/`H` calibration** — what emission rate bootstraps demand without runaway inflation?
- **Unclaimed-pool policy** — roll forward vs burn in epochs with no valid findings.
- **Acceptance threshold** — fixed 30%, or scaled to severity (criticals need a higher quorum)?
- **Reviewer compensation** — are reviewers paid from the pool too? (They must be, or honest review is a public-good underprovided.) How, without diluting miner reward or inviting rubber-stamping?
- **Disclosure window** — a critical exploit is dangerous *while unpatched*; submission must be private until the fix merges. Needs an encrypted-submission / responsible-disclosure flow on-chain or off.
- **Cross-version validity** — pinning "canonical HEAD" so a fix for an already-patched hole can't be re-mined.

## 14. Next steps

1. Land the **slashable-bond** primitive (gates §5 and §7).
2. Define the **submission schema** (repro test + patch + targeted-scenario ref + bond) and the **content-address** for dedup.
3. Build the **objective gate** as CI: fails-on-HEAD → passes-with-patch → suite-green → not-a-dupe.
4. Prototype Tier 2 **centralized (anchor-judged)** first — prove the artifact-reward loop end-to-end — *before* decentralizing the vote.
5. A SimDD iteration whose subject **is this mechanism** (a miner farming fake findings; a reviewer cartel; a hole-manufacturer) — the exploit is the spec, applied to the mint itself.
