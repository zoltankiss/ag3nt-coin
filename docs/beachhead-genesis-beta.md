# Beachhead Genesis Beta Plan

**Target version:** `0.4.0-beta.1`
**Goal:** validate a one-founder genesis and true fresh-agent onboarding path before `0.4.0` final.

## Genesis State

The beachhead genesis should be intentionally narrow:

- one founder/trust-root address
- founder is the initial anchor
- founder starts with `100` raw reputation/trust-root mass
- founder starts with `0` AGNT
- no friend accounts in genesis
- no premine
- empty gate/useful-work history

Friends and early agents should arrive after chain start. That is the point of the test: prove that a fresh `0 rep / 0 coin` agent can earn first working capital and then earn scoped trust.

## First Agents

After the chain is online:

1. Friend agent generates a keypair.
2. Friend registers on-chain.
3. Friend has `0 rep / 0 coin`.
4. Friend creates a small agntcoin protocol contribution artifact.
5. Founder can award capped protocol-minted AGNT for an accepted contribution,
   pinned to PR/commit/artifact/evidence hashes.
6. Friend answers protocol PR-review gates.
7. Coherent decoy answers mint tiny gate drip.
8. Founder reviews the evidence trail.
9. Founder vouches scoped reputation if the evidence supports it.

## Founder Vouch Decision

Founder vouches should be auditable attestations, not opaque reputation gifts.

A public decision artifact should include:

```json
{
  "type": "founder_reviewer_vouch",
  "target_agent": "agnt1...",
  "scope": "protocol-pr-review/security-low",
  "decision": "vouch",
  "weight": 12,
  "stake": 500,
  "evidence": [
    "gate:17",
    "gate:22",
    "pr-review:abc123"
  ],
  "rationale": "Correctly caught a hash-binding regression and rejected two seeded bad PRs.",
  "reviewer": "founder-address",
  "created_at": "2026-..."
}
```

The chain should eventually record:

- target agent
- scope
- weight/stake
- decision artifact URI
- decision artifact SHA-256
- evidence bundle URI
- evidence bundle SHA-256

For beta.1, the on-chain vouch primitive exists but scoped evidence artifacts are still a design/rollout layer. The beachhead simulation should still produce the off-chain artifacts and hash them so the next primitive can be forged from real traces.

Contribution awards are already on-chain in beta.1 as a narrow author-mint rail.
The award records:

- anchor/founder address
- recipient author address
- repo URL and optional PR URL
- commit SHA
- artifact URI and SHA-256
- evidence bundle SHA-256
- scope
- optional rationale hash
- capped minted amount

This proves the author half of proof-of-useful-work without claiming final
decentralized merge or emission governance.

## Artifact Availability

Full evidence files live off-chain. The chain should pin integrity, not store bulky review logs.

Availability rules for later hardening:

- moving an artifact requires an on-chain location update that preserves the same SHA-256
- changing artifact bytes creates a new artifact
- unavailable evidence should trigger a notice and cure window before slashing
- false or frivolous challenges must lose a cry-wolf bond

This prevents the system from becoming a low-cost accusation machine.

## Success Criteria

The beta beachhead succeeds if:

- the founder can start a chain with no premine
- two fresh agents join post-genesis as `0 rep / 0 coin`
- each attempts at least one small agntcoin contribution artifact
- at least one accepted contribution earns capped protocol-minted AGNT through
  a hash-pinned contribution award
- each earns first or additional AGNT through PR-review decoy gates
- decoy PR-review bundles are produced by agents and only beta-audited by the founder/anchor
- gate answers and rationales are recoverable as evidence
- founder produces public vouch decision artifacts
- founder vouches scoped reputation only where evidence supports it
- no live gate is treated as merge authority or major mint authority

## Non-Goals

- adversarial sybil resistance
- unattended open gate posting
- automatic reviewer reputation from gates
- app-specific QA subsidized by protocol mint
- decentralized contribution acceptance
- final `0.4.0` constants

Those are later validation rungs.
