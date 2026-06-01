package keeper

import (
	"context"
	"strconv"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/zoltankiss/agntcoin/x/agntcoin/types"
)

// Castvouch records (or updates) the caller's vouch for to_addr and LOCKS a
// stake of ag3nt behind it. The staked coin is debited from the voucher's
// balance and held in the Vouch record; it is returned when the vouch is
// removed (Unvouch) or replaced (re-vouch). Reputation is derived from the
// locked stake, making the trust graph costly to Sybil.
func (k msgServer) Castvouch(ctx context.Context, msg *types.MsgCastvouch) (*types.MsgCastvouchResponse, error) {
	if _, err := k.addressCodec.StringToBytes(msg.Creator); err != nil {
		return nil, errorsmod.Wrap(err, "invalid creator address")
	}
	if _, err := k.addressCodec.StringToBytes(msg.ToAddr); err != nil {
		return nil, errorsmod.Wrap(err, "invalid vouchee address")
	}
	if msg.Creator == msg.ToAddr {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "cannot vouch for self")
	}
	if msg.Weight < 1 || msg.Weight > types.MaxVouchWeight {
		return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "weight must be 1..%d", types.MaxVouchWeight)
	}
	if msg.Stake < types.MinVouchStake {
		return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "stake must be at least %d", types.MinVouchStake)
	}

	from, err := k.Account.Get(ctx, msg.Creator)
	if err != nil || !from.Registered {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "voucher not registered")
	}
	if to, err := k.Account.Get(ctx, msg.ToAddr); err != nil || !to.Registered {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "vouchee not registered")
	}

	// Find an existing vouch for (from, to) so we can upsert.
	var existingID uint64
	var oldStake uint64
	found := false
	err = k.Vouch.Walk(ctx, nil, func(id uint64, v types.Vouch) (bool, error) {
		if v.FromAddr == msg.Creator && v.ToAddr == msg.ToAddr {
			existingID = id
			oldStake = v.Stake
			found = true
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}

	// On re-vouch, return the previously-locked stake first so we never
	// double-charge the voucher. The net delta is (newStake - oldStake).
	if found {
		from.Balance += oldStake
	}

	// The voucher must be able to cover the new stake (after any refund of the
	// old one).
	if from.Balance < msg.Stake {
		return nil, errorsmod.Wrap(sdkerrors.ErrInsufficientFunds, "insufficient balance to lock vouch stake")
	}
	// Debit the new stake; it is now held in the Vouch record.
	from.Balance -= msg.Stake
	if err := k.Account.Set(ctx, msg.Creator, from); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}

	var entry types.Vouch
	if found {
		entry = types.Vouch{Id: existingID, FromAddr: msg.Creator, ToAddr: msg.ToAddr, Weight: msg.Weight, Stake: msg.Stake}
		if err := k.Vouch.Set(ctx, existingID, entry); err != nil {
			return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
		}
	} else {
		id, err := k.VouchSeq.Next(ctx)
		if err != nil {
			return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
		}
		entry = types.Vouch{Id: id, FromAddr: msg.Creator, ToAddr: msg.ToAddr, Weight: msg.Weight, Stake: msg.Stake}
		if err := k.Vouch.Set(ctx, id, entry); err != nil {
			return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
		}
	}

	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(
		sdk.NewEvent(
			"agntcoin_vouch_cast",
			sdk.NewAttribute("id", strconv.FormatUint(entry.Id, 10)),
			sdk.NewAttribute("from", msg.Creator),
			sdk.NewAttribute("to", msg.ToAddr),
			sdk.NewAttribute("weight", strconv.FormatUint(msg.Weight, 10)),
			sdk.NewAttribute("stake", strconv.FormatUint(msg.Stake, 10)),
		),
	)

	return &types.MsgCastvouchResponse{}, nil
}
