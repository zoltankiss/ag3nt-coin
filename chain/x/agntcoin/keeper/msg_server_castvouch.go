package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/zoltankiss/agntcoin/x/agntcoin/types"
)

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

	if from, err := k.Account.Get(ctx, msg.Creator); err != nil || !from.Registered {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "voucher not registered")
	}
	if to, err := k.Account.Get(ctx, msg.ToAddr); err != nil || !to.Registered {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "vouchee not registered")
	}

	var existingID uint64
	found := false
	err := k.Vouch.Walk(ctx, nil, func(id uint64, v types.Vouch) (bool, error) {
		if v.FromAddr == msg.Creator && v.ToAddr == msg.ToAddr {
			existingID = id
			found = true
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}

	var entry types.Vouch
	if found {
		entry = types.Vouch{Id: existingID, FromAddr: msg.Creator, ToAddr: msg.ToAddr, Weight: msg.Weight}
		if err := k.Vouch.Set(ctx, existingID, entry); err != nil {
			return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
		}
	} else {
		id, err := k.VouchSeq.Next(ctx)
		if err != nil {
			return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
		}
		entry = types.Vouch{Id: id, FromAddr: msg.Creator, ToAddr: msg.ToAddr, Weight: msg.Weight}
		if err := k.Vouch.Set(ctx, id, entry); err != nil {
			return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
		}
	}
	return &types.MsgCastvouchResponse{}, nil
}
