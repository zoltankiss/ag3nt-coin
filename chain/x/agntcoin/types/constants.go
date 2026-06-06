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
