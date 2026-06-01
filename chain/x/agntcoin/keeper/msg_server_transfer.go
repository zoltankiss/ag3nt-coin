package keeper

import (
	"context"
	"strconv"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/zoltankiss/agntcoin/x/agntcoin/types"
)

func (k msgServer) Transfer(ctx context.Context, msg *types.MsgTransfer) (*types.MsgTransferResponse, error) {
	if _, err := k.addressCodec.StringToBytes(msg.Creator); err != nil {
		return nil, errorsmod.Wrap(err, "invalid creator address")
	}
	toBytes, err := k.addressCodec.StringToBytes(msg.ToAddr)
	if err != nil {
		return nil, errorsmod.Wrap(err, "invalid recipient address")
	}
	if msg.Amount == 0 {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "amount must be positive")
	}
	if msg.Creator == msg.ToAddr {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "cannot transfer to self")
	}

	// Sender-side checks are unchanged: sender must be registered with funds.
	from, err := k.Account.Get(ctx, msg.Creator)
	if err != nil || !from.Registered {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "sender not registered")
	}
	if from.Balance < msg.Amount {
		return nil, errorsmod.Wrap(sdkerrors.ErrInsufficientFunds, "insufficient balance")
	}

	// PATCH 3: reject dust to a brand-new account before mutating any state.
	if err := k.checkNewAccountCredit(ctx, msg.ToAddr, msg.Amount); err != nil {
		return nil, err
	}

	// Debit the sender first.
	from.Balance -= msg.Amount
	if err := k.Account.Set(ctx, msg.Creator, from); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}

	// CHANGE 1: auto-register the recipient (and create their x/auth account)
	// instead of erroring. The shared creditAccount helper also enforces the
	// new-account spam guard (PATCH 3).
	if err := k.creditAccount(ctx, msg.ToAddr, toBytes, msg.Amount); err != nil {
		return nil, err
	}

	// CHANGE 2: emit a typed transfer event for auditability/queryability.
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"agntcoin_transfer",
			sdk.NewAttribute("from", msg.Creator),
			sdk.NewAttribute("to", msg.ToAddr),
			sdk.NewAttribute("amount", strconv.FormatUint(msg.Amount, 10)),
			sdk.NewAttribute("memo", msg.Memo),
		),
	)

	return &types.MsgTransferResponse{}, nil
}
