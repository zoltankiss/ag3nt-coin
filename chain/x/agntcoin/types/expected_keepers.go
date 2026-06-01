package types

import (
	"context"

	"cosmossdk.io/core/address"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// AuthKeeper defines the expected interface for the Auth module.
type AuthKeeper interface {
	AddressCodec() address.Codec
	GetAccount(context.Context, sdk.AccAddress) sdk.AccountI // only used for simulation
	// HasAccount reports whether an x/auth account already exists for the address.
	HasAccount(context.Context, sdk.AccAddress) bool
	// NewAccountWithAddress returns a new BaseAccount for the address (not yet persisted).
	NewAccountWithAddress(context.Context, sdk.AccAddress) sdk.AccountI
	// SetAccount persists an account in the auth store.
	SetAccount(context.Context, sdk.AccountI)
}

// BankKeeper defines the expected interface for the Bank module.
type BankKeeper interface {
	SpendableCoins(context.Context, sdk.AccAddress) sdk.Coins
	// Methods imported from bank should be defined here
}

// ParamSubspace defines the expected Subspace interface for parameters.
type ParamSubspace interface {
	Get(context.Context, []byte, interface{})
	Set(context.Context, []byte, interface{})
}
