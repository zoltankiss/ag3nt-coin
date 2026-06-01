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

// SubmitEscrow lets the PAYEE mark a locked escrow as delivered. Once submitted,
// the payer can no longer refund it (RefundEscrow only accepts "locked") — only
// release remains, or the payee self-releases after the deadline. This is the
// fair-exchange guarantee for the worker: it can reveal delivered work without
// the buyer clawing the payment back. The buyer's recourse against bad work is
// DisputeEscrow, not refund.
func (k msgServer) SubmitEscrow(ctx context.Context, msg *types.MsgSubmitEscrow) (*types.MsgSubmitEscrowResponse, error) {
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
	if msg.Creator != escrow.Payee {
		return nil, errorsmod.Wrap(sdkerrors.ErrUnauthorized, "only the payee may submit")
	}

	escrow.Status = types.EscrowStatusSubmitted
	if err := k.Escrow.Set(ctx, escrow.Id, escrow); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}

	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(
		sdk.NewEvent(
			"agntcoin_escrow_submitted",
			sdk.NewAttribute("id", strconv.FormatUint(escrow.Id, 10)),
			sdk.NewAttribute("payer", escrow.Payer),
			sdk.NewAttribute("payee", escrow.Payee),
			sdk.NewAttribute("amount", strconv.FormatUint(escrow.Amount, 10)),
			sdk.NewAttribute("ref", escrow.Ref),
		),
	)

	return &types.MsgSubmitEscrowResponse{}, nil
}
