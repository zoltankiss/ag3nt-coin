package keeper

import (
	"context"
	"strconv"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/zoltankiss/agntcoin/x/agntcoin/types"
)

// PostBond locks `amount` of the creator's balance as slashable collateral
// behind a claim/behavior (purpose/ref tag it). The named slasher — a neutral
// adjudicator, never the poster itself — is the ONLY address that may later
// release (refund the poster) or slash (pay a beneficiary / burn) the bond.
// Willingness to post slashable collateral is the cold-start sincerity
// signal: at reputation 0 it is what distinguishes an honest newcomer from a
// Sybil (it10 finding #17).
func (k msgServer) PostBond(ctx context.Context, msg *types.MsgPostBond) (*types.MsgPostBondResponse, error) {
	if _, err := k.addressCodec.StringToBytes(msg.Creator); err != nil {
		return nil, errorsmod.Wrap(err, "invalid creator address")
	}
	if _, err := k.addressCodec.StringToBytes(msg.Slasher); err != nil {
		return nil, errorsmod.Wrap(err, "invalid slasher address")
	}
	if msg.Amount == 0 {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "amount must be positive")
	}
	if msg.Creator == msg.Slasher {
		// A self-slashed bond is a refundable deposit, not a deterrent: the
		// poster could simply release itself. Reject it structurally (I1).
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "slasher cannot be the poster")
	}

	// Poster must be registered with sufficient balance.
	poster, err := k.Account.Get(ctx, msg.Creator)
	if err != nil || !poster.Registered {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "poster not registered")
	}
	if poster.Balance < msg.Amount {
		return nil, errorsmod.Wrap(sdkerrors.ErrInsufficientFunds, "insufficient balance")
	}

	// Debit the poster immediately; the collateral is now held by the protocol.
	poster.Balance -= msg.Amount
	if err := k.Account.Set(ctx, msg.Creator, poster); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}

	id, err := k.BondSeq.Next(ctx)
	if err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}
	bond := types.Bond{
		Id:      id,
		Poster:  msg.Creator,
		Amount:  msg.Amount,
		Purpose: msg.Purpose,
		Slasher: msg.Slasher,
		Status:  types.BondStatusActive,
		Ref:     msg.Ref,
	}
	if err := k.Bond.Set(ctx, id, bond); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"agntcoin_bond_posted",
			sdk.NewAttribute("id", strconv.FormatUint(id, 10)),
			sdk.NewAttribute("poster", msg.Creator),
			sdk.NewAttribute("amount", strconv.FormatUint(msg.Amount, 10)),
			sdk.NewAttribute("purpose", msg.Purpose),
			sdk.NewAttribute("slasher", msg.Slasher),
			sdk.NewAttribute("ref", msg.Ref),
		),
	)

	return &types.MsgPostBondResponse{Id: id}, nil
}
