package types

import "cosmossdk.io/collections"

const (
	// ModuleName defines the module name
	ModuleName = "agntcoin"

	// StoreKey defines the primary module store key
	StoreKey = ModuleName

	// GovModuleName duplicates the gov module's name to avoid a dependency with x/gov.
	// It should be synced with the gov module's name if it is ever changed.
	// See: https://github.com/cosmos/cosmos-sdk/blob/v0.52.0-beta.2/x/gov/types/keys.go#L9
	GovModuleName = "gov"
)

// ParamsKey is the prefix to retrieve all Params
var ParamsKey = collections.NewPrefix("p_agntcoin")

// EmissionStateKey stores the singleton monetary-policy state.
var EmissionStateKey = collections.NewPrefix("emission/state")

var (
	VouchKey      = collections.NewPrefix("vouch/value/")
	VouchCountKey = collections.NewPrefix("vouch/count/")
)

var (
	EscrowKey      = collections.NewPrefix("escrow/value/")
	EscrowCountKey = collections.NewPrefix("escrow/count/")
)

var (
	DisputeKey      = collections.NewPrefix("dispute/value/")
	DisputeCountKey = collections.NewPrefix("dispute/count/")
)

var (
	BondKey      = collections.NewPrefix("bond/value/")
	BondCountKey = collections.NewPrefix("bond/count/")
)

var (
	GateKey      = collections.NewPrefix("gate/value/")
	GateCountKey = collections.NewPrefix("gate/count/")
)

var (
	ContributionAwardKey      = collections.NewPrefix("contribution_award/value/")
	ContributionAwardCountKey = collections.NewPrefix("contribution_award/count/")
)

var (
	ScopedEvidenceVouchKey      = collections.NewPrefix("scoped_evidence_vouch/value/")
	ScopedEvidenceVouchCountKey = collections.NewPrefix("scoped_evidence_vouch/count/")
)
