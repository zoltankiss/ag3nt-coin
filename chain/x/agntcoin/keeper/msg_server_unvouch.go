package keeper

import (
	"context"
	"strconv"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/zoltankiss/agntcoin/x/agntcoin/types"
)

// Unvouch removes the caller's vouch for to_addr and returns the locked stake
// to the caller's balance. It rejects if no such vouch exists.
func (k msgServer) Unvouch(ctx context.Context, msg *types.MsgUnvouch) (*types.MsgUnvouchResponse, error) {
	if _, err := k.addressCodec.StringToBytes(msg.Creator); err != nil {
		return nil, errorsmod.Wrap(err, "invalid creator address")
	}
	if _, err := k.addressCodec.StringToBytes(msg.ToAddr); err != nil {
		return nil, errorsmod.Wrap(err, "invalid vouchee address")
	}

	// Locate the caller's vouch for to_addr.
	var existingID uint64
	var stake uint64
	found := false
	err := k.Vouch.Walk(ctx, nil, func(id uint64, v types.Vouch) (bool, error) {
		if v.FromAddr == msg.Creator && v.ToAddr == msg.ToAddr {
			existingID = id
			stake = v.Stake
			found = true
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}
	if !found {
		return nil, errorsmod.Wrap(sdkerrors.ErrKeyNotFound, "no vouch from caller for that address")
	}

	// Return the locked stake to the voucher. The voucher already has an
	// Account record (they staked it), so credit it directly.
	from, err := k.Account.Get(ctx, msg.Creator)
	if err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}
	from.Balance += stake
	if err := k.Account.Set(ctx, msg.Creator, from); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}

	if err := k.Vouch.Remove(ctx, existingID); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}

	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(
		sdk.NewEvent(
			"agntcoin_vouch_removed",
			sdk.NewAttribute("id", strconv.FormatUint(existingID, 10)),
			sdk.NewAttribute("from", msg.Creator),
			sdk.NewAttribute("to", msg.ToAddr),
			sdk.NewAttribute("stake", strconv.FormatUint(stake, 10)),
		),
	)

	return &types.MsgUnvouchResponse{}, nil
}
