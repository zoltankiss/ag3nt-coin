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

// ReleaseEscrow pays out a locked escrow to the payee. It is allowed if the
// caller is the payer (who may release at any time) OR the dispute deadline
// has passed (so a ghosting payer cannot trap the payee's funds).
func (k msgServer) ReleaseEscrow(ctx context.Context, msg *types.MsgReleaseEscrow) (*types.MsgReleaseEscrowResponse, error) {
	if _, err := k.addressCodec.StringToBytes(msg.Creator); err != nil {
		return nil, errorsmod.Wrap(err, "invalid creator address")
	}

	escrow, err := k.Escrow.Get(ctx, msg.Id)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil, errorsmod.Wrap(sdkerrors.ErrKeyNotFound, "escrow not found")
		}
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}
	if escrow.Status != types.EscrowStatusLocked {
		return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "escrow is not locked (status=%s)", escrow.Status)
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	now := sdkCtx.BlockTime().Unix()
	if msg.Creator != escrow.Payer && now < escrow.Deadline {
		return nil, errorsmod.Wrap(sdkerrors.ErrUnauthorized, "only the payer may release before the deadline")
	}

	payeeBytes, err := k.addressCodec.StringToBytes(escrow.Payee)
	if err != nil {
		return nil, errorsmod.Wrap(err, "invalid payee address")
	}
	// Credit the payee the full amount, auto-registering + creating an x/auth
	// account if the payee is new. Spam guard does not block legitimate escrow
	// payouts because LockEscrow already required amount > 0; the guard only
	// rejects sub-minimum credits to brand-new accounts.
	if err := k.checkNewAccountCredit(ctx, escrow.Payee, escrow.Amount); err != nil {
		return nil, err
	}
	if err := k.creditAccount(ctx, escrow.Payee, payeeBytes, escrow.Amount); err != nil {
		return nil, err
	}

	escrow.Status = types.EscrowStatusReleased
	if err := k.Escrow.Set(ctx, escrow.Id, escrow); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}

	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"agntcoin_escrow_released",
			sdk.NewAttribute("id", strconv.FormatUint(escrow.Id, 10)),
			sdk.NewAttribute("payer", escrow.Payer),
			sdk.NewAttribute("payee", escrow.Payee),
			sdk.NewAttribute("amount", strconv.FormatUint(escrow.Amount, 10)),
			sdk.NewAttribute("ref", escrow.Ref),
		),
	)

	return &types.MsgReleaseEscrowResponse{}, nil
}
