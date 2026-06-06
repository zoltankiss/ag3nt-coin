package keeper

import (
	"context"
	"errors"
	"strconv"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/zoltankiss/agntcoin/x/agntcoin/types"
)

// ReleaseBond exonerates an active bond: the collateral returns to the
// poster. ONLY the bond's designated slasher may release — in particular the
// poster can never release its own bond (no self-withdraw, I1). That single
// rejection is what makes a bond a deterrent instead of a refundable deposit.
func (k msgServer) ReleaseBond(ctx context.Context, msg *types.MsgReleaseBond) (*types.MsgReleaseBondResponse, error) {
	if _, err := k.addressCodec.StringToBytes(msg.Creator); err != nil {
		return nil, errorsmod.Wrap(err, "invalid creator address")
	}

	bond, err := k.Bond.Get(ctx, msg.Id)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil, errorsmod.Wrap(sdkerrors.ErrKeyNotFound, "bond not found")
		}
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}
	if bond.Status != types.BondStatusActive {
		// Resolves once (I4): no release-after-slash / double-release.
		return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "bond is not active (status=%s)", bond.Status)
	}
	if msg.Creator != bond.Slasher {
		// I1/I2: the poster cannot self-withdraw; nobody but the slasher resolves.
		return nil, errorsmod.Wrap(sdkerrors.ErrUnauthorized, "only the bond's slasher may release (the poster cannot self-withdraw)")
	}

	// Refund the collateral to the poster. The poster's account necessarily
	// exists (it was debited at post time), so this is a plain credit.
	poster, err := k.Account.Get(ctx, bond.Poster)
	if err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}
	poster.Balance += bond.Amount
	if err := k.Account.Set(ctx, bond.Poster, poster); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}

	bond.Status = types.BondStatusReleased
	if err := k.Bond.Set(ctx, bond.Id, bond); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"agntcoin_bond_released",
			sdk.NewAttribute("id", strconv.FormatUint(bond.Id, 10)),
			sdk.NewAttribute("poster", bond.Poster),
			sdk.NewAttribute("amount", strconv.FormatUint(bond.Amount, 10)),
			sdk.NewAttribute("slasher", bond.Slasher),
			sdk.NewAttribute("ref", bond.Ref),
		),
	)

	return &types.MsgReleaseBondResponse{}, nil
}
