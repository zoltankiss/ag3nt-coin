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
	EscrowStatusReleased  = "released"
	EscrowStatusRefunded  = "refunded"
)
