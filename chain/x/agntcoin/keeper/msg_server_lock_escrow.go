package keeper

import (
	"context"
	"strconv"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/zoltankiss/agntcoin/x/agntcoin/types"
)

// LockEscrow locks funds from the creator (payer) into a new protocol-held
// escrow. The funds are debited immediately and recorded as the escrow's
// amount until the escrow is released to the payee or refunded to the payer.
func (k msgServer) LockEscrow(ctx context.Context, msg *types.MsgLockEscrow) (*types.MsgLockEscrowResponse, error) {
	if _, err := k.addressCodec.StringToBytes(msg.Creator); err != nil {
		return nil, errorsmod.Wrap(err, "invalid creator address")
	}
	if _, err := k.addressCodec.StringToBytes(msg.Payee); err != nil {
		return nil, errorsmod.Wrap(err, "invalid payee address")
	}
	if msg.Amount == 0 {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "amount must be positive")
	}
	if msg.Creator == msg.Payee {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "payee cannot be the payer")
	}

	// Payer must be registered with sufficient balance.
	payer, err := k.Account.Get(ctx, msg.Creator)
	if err != nil || !payer.Registered {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "payer not registered")
	}
	if payer.Balance < msg.Amount {
		return nil, errorsmod.Wrap(sdkerrors.ErrInsufficientFunds, "insufficient balance")
	}

	// Auto-register the payee at lock time (x/auth account, no balance credit) so
	// a brand-new worker can escrow-submit on its very first job — closing the
	// "first job is unprotectable" hole. Done BEFORE debiting so a rejected
	// sub-minimum escrow to a brand-new payee leaves no partial state.
	payeeBytes, err := k.addressCodec.StringToBytes(msg.Payee)
	if err != nil {
		return nil, errorsmod.Wrap(err, "invalid payee address")
	}
	if err := k.ensureRegistered(ctx, msg.Payee, payeeBytes, msg.Amount); err != nil {
		return nil, err
	}

	// Debit the payer immediately; funds are now held by the protocol.
	payer.Balance -= msg.Amount
	if err := k.Account.Set(ctx, msg.Creator, payer); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	deadline := sdkCtx.BlockTime().Unix() + int64(msg.DisputeSeconds)

	id, err := k.EscrowSeq.Next(ctx)
	if err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}
	escrow := types.Escrow{
		Id:            id,
		Payer:         msg.Creator,
		Payee:         msg.Payee,
		Amount:        msg.Amount,
		Ref:           msg.Ref,
		Status:        types.EscrowStatusLocked,
		Deadline:      deadline,
		NoAutoRelease: msg.NoAutoRelease,
	}
	if err := k.Escrow.Set(ctx, id, escrow); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}

	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"agntcoin_escrow_locked",
			sdk.NewAttribute("id", strconv.FormatUint(id, 10)),
			sdk.NewAttribute("payer", msg.Creator),
			sdk.NewAttribute("payee", msg.Payee),
			sdk.NewAttribute("amount", strconv.FormatUint(msg.Amount, 10)),
			sdk.NewAttribute("ref", msg.Ref),
			sdk.NewAttribute("deadline", strconv.FormatInt(deadline, 10)),
		),
	)

	return &types.MsgLockEscrowResponse{Id: id}, nil
}
