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

var (
	VouchKey      = collections.NewPrefix("vouch/value/")
	VouchCountKey = collections.NewPrefix("vouch/count/")
)

var (
	EscrowKey      = collections.NewPrefix("escrow/value/")
	EscrowCountKey = collections.NewPrefix("escrow/count/")
)
