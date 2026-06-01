package types

const (
	FaucetAmount   uint64 = 10_000
	MaxVouchWeight uint64 = 100

	// MinNewAccountCredit is the minimum amount that may be credited to a
	// brand-new (not yet existing) address. Transfers/escrow payouts that
	// would create a new account with less than this are rejected to bound
	// dust-spam account creation. Credits to addresses that already have an
	// Account record are unaffected.
	MinNewAccountCredit uint64 = 100

	// Escrow status values.
	EscrowStatusLocked   = "locked"
	EscrowStatusReleased = "released"
	EscrowStatusRefunded = "refunded"
)
