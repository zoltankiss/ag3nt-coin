package types

const (
	FaucetAmount   uint64 = 10_000
	MaxVouchWeight uint64 = 100

	// MinVouchStake is the minimum amount of ag3nt that must be locked behind a
	// vouch. This is the cost-of-identity floor: every edge in the reputation
	// graph costs real, locked stake, so a Sybil ring must lock real capital to
	// manufacture trust.
	MinVouchStake uint64 = 100

	// MinNewAccountCredit is the minimum amount that may be credited to a
	// brand-new (not yet existing) address. Transfers/escrow payouts that
	// would create a new account with less than this are rejected to bound
	// dust-spam account creation. Credits to addresses that already have an
	// Account record are unaffected.
	MinNewAccountCredit uint64 = 100

	// Escrow status values.
	EscrowStatusLocked    = "locked"
	EscrowStatusSubmitted = "submitted" // payee delivered → refund locked out
	EscrowStatusDisputed  = "disputed"  // payer contested submitted work → frozen
	EscrowStatusInJury    = "in_jury"   // escalated to a k-of-n jury → only a verdict settles it
	EscrowStatusReleased  = "released"
	EscrowStatusRefunded  = "refunded"
	// EscrowStatusFailAttested (verifier-v1): a verifier posted a staked
	// passed=false attestation. The payee's CONTEST window is open until
	// challenge_deadline (OpenDispute → jury; a proven false-fail slashes the
	// attester's stake to the payee). Only after the window passes uncontested
	// does RefundEscrow unlock for the payer — a fail attestation that unlocked
	// refund instantly would let a colluding buyer+verifier take delivered work
	// for free (the false-fail symmetry hole).
	EscrowStatusFailAttested = "fail_attested"
	// EscrowStatusChallenged (verifier-v1): the payer opened a post-release
	// fraud challenge inside the challenge window. The payout already happened
	// and is NOT clawed back from the payee — the jury verdict settles the
	// attestation-stakes instead (fraud => slashed to the payer, who is made
	// whole from the verifier's collateral; valid => challenger's dispute-bond
	// slashed to the griefed attester).
	EscrowStatusChallenged = "challenged"

	// ChallengeWindowSeconds (verifier-v1) bounds the post-settlement
	// adversarial window: after VerifiedRelease, how long the payer may open a
	// fraud challenge; after a fail attestation, how long the payee may contest
	// before refund unlocks. Deadline math uses BLOCK time at tx inclusion —
	// never a party-supplied timestamp (the chain can't trust anyone's clock).
	ChallengeWindowSeconds int64 = 72 * 3600

	// MinDisputeBond is the minimum slashable collateral an opener must post to
	// OpenDispute. Opening a dispute freezes the escrow + loads the jury, so a
	// FREE dispute is weaponizable (cry-wolf to stall payment; jury-DDoS to bury
	// real cases — it18 RED). Requiring a bond makes a frivolous dispute cost the
	// opener (slashed to the griefed counterparty on a lost verdict) and makes a
	// flood cost N bonds. Returned in full when the opener's dispute is upheld.
	MinDisputeBond uint64 = 100

	// MinJurorStake is the minimum slashable collateral a juror must post WITH a
	// vote (jury-v1, it20). The jury settles escrows AND slashes dispute-bonds, so
	// a free, penalty-free vote lets a colluding juror push fraudulent verdicts. A
	// per-vote stake makes voting cost: coherent-with-the-verdict jurors get it
	// back; a juror incoherent with the resolved verdict is slashed to the wronged
	// party (the minority-collusion deterrent). Majority collusion is out of scope
	// for v1 — it needs reputation-weighted/proper-scoring voting (it25).
	MinJurorStake uint64 = 100

	// Bond (slashable collateral) status values. ACTIVE resolves exactly once,
	// to RELEASED (collateral -> poster) or SLASHED (collateral -> beneficiary,
	// or burned) — and only ever by the bond's designated slasher.
	BondStatusActive   = "active"
	BondStatusReleased = "released"
	BondStatusSlashed  = "slashed"

	// Dispute (k-of-n jury) status + resolution values.
	DisputeStatusOpen       = "open"
	DisputeStatusResolved   = "resolved"
	DisputeResolutionAccept = "accept" // jury accepted the work → release to payee
	DisputeResolutionReject = "reject" // jury rejected the work → refund to payer

	// ---- gate-v1 (0.4.0-beta.1): protocol PR-review gates + earned faucet.

	// GateCommitWindowSeconds is how long after PostGate answers may be
	// committed. GateRevealWindowSeconds then bounds the reveal phase —
	// reveals open ONLY once commits close (anti-copying), and settlement
	// only once reveals close. Both bind to block time at tx inclusion.
	GateCommitWindowSeconds int64 = 300
	GateRevealWindowSeconds int64 = 300

	// MaxDripPerAnswer caps the minted drip per coherent gate answer. Tiny by
	// design: a gate answer is the earned-faucet granule (compare the old
	// faucet's one-shot 10,000) — first working capital is accumulated through
	// MANY calibrated protocol-review answers, so mass extraction costs mass
	// honest compute.
	MaxDripPerAnswer uint64 = 50

	// MaxGateAnswers bounds the answer set per gate (state growth + settle gas).
	MaxGateAnswers uint64 = 1000

	// Gate status values. OPEN settles exactly once into SETTLED.
	GateStatusOpen    = "open"
	GateStatusSettled = "settled"

	// MaxContributionAward caps founder-mediated author mints in the beta
	// proof-of-useful-work path. It is intentionally far below the old 10,000
	// faucet: author mining should be evidence-rich and incremental until
	// review sortition, duplicate checks, challenge bonds, and emission policy
	// are forged.
	MaxContributionAward uint64 = 500

	// MinScopedEvidenceReputation is the beta threshold for non-anchor issuers
	// to cast an unstaked scoped evidence vouch. Reputation scores are
	// normalized PageRank values, so 0.01 is intentionally high in small
	// networks while still allowing high-standing non-founders to participate.
	MinScopedEvidenceReputation float64 = 0.01
)
