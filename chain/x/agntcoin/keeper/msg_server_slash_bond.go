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

// SlashBond punishes an active bond: the collateral goes to the named
// beneficiary (e.g. the buyer a ghosting worker stranded), or is burned if
// beneficiary is empty (a deliberate supply decrease). ONLY the bond's
// designated slasher may slash (I2); a bond resolves exactly once (I4).
func (k msgServer) SlashBond(ctx context.Context, msg *types.MsgSlashBond) (*types.MsgSlashBondResponse, error) {
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
		// Resolves once (I4): no double-slash / slash-after-release.
		return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "bond is not active (status=%s)", bond.Status)
	}
	if msg.Creator != bond.Slasher {
		// I2: only the designated slasher may punish.
		return nil, errorsmod.Wrap(sdkerrors.ErrUnauthorized, "only the bond's slasher may slash")
	}

	if msg.Beneficiary != "" {
		// Conservation (I3): exactly `amount` moves, once — to the beneficiary.
		beneficiaryBytes, err := k.addressCodec.StringToBytes(msg.Beneficiary)
		if err != nil {
			return nil, errorsmod.Wrap(err, "invalid beneficiary address")
		}
		// Dust-spam guard applies to brand-new beneficiaries, then credit
		// (auto-registering + creating an x/auth account if needed).
		if err := k.checkNewAccountCredit(ctx, msg.Beneficiary, bond.Amount); err != nil {
			return nil, err
		}
		if err := k.creditAccount(ctx, msg.Beneficiary, beneficiaryBytes, bond.Amount); err != nil {
			return nil, err
		}
	}
	// Empty beneficiary = burn: the collateral was already debited from the
	// poster at post time, so crediting no one is a deliberate supply decrease.

	bond.Status = types.BondStatusSlashed
	if err := k.Bond.Set(ctx, bond.Id, bond); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"agntcoin_bond_slashed",
			sdk.NewAttribute("id", strconv.FormatUint(bond.Id, 10)),
			sdk.NewAttribute("poster", bond.Poster),
			sdk.NewAttribute("amount", strconv.FormatUint(bond.Amount, 10)),
			sdk.NewAttribute("slasher", bond.Slasher),
			sdk.NewAttribute("beneficiary", msg.Beneficiary),
			sdk.NewAttribute("ref", bond.Ref),
		),
	)

	return &types.MsgSlashBondResponse{}, nil
}
