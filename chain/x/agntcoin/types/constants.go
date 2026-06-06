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
)
