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

// OpenDispute opens a k-of-n jury case against a submitted or disputed escrow
// whose acceptability cannot be settled by an objective test (or is contested).
// Either the payer or the payee may open it. The escrow moves to "in_jury",
// which freezes it: neither side can unilaterally release/refund — only a jury
// verdict (ResolveDispute) settles it. This is the recourse the it11 honest
// worker lacked when a non-technical buyer's broken tests blocked good work.
func (k msgServer) OpenDispute(ctx context.Context, msg *types.MsgOpenDispute) (*types.MsgOpenDisputeResponse, error) {
	if _, err := k.addressCodec.StringToBytes(msg.Creator); err != nil {
		return nil, errorsmod.Wrap(err, "invalid creator address")
	}

	escrow, err := k.Escrow.Get(ctx, msg.EscrowId)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil, errorsmod.Wrap(sdkerrors.ErrKeyNotFound, "escrow not found")
		}
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}
	if escrow.Status != types.EscrowStatusSubmitted && escrow.Status != types.EscrowStatusDisputed {
		return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "escrow must be submitted or disputed to open a jury case (status=%s)", escrow.Status)
	}
	if msg.Creator != escrow.Payer && msg.Creator != escrow.Payee {
		return nil, errorsmod.Wrap(sdkerrors.ErrUnauthorized, "only the escrow payer or payee may open a dispute")
	}

	id, err := k.DisputeSeq.Next(ctx)
	if err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}
	dispute := types.Dispute{
		Id:       id,
		EscrowId: msg.EscrowId,
		Opener:   msg.Creator,
		Reason:   msg.Reason,
		Status:   types.DisputeStatusOpen,
	}
	if err := k.Dispute.Set(ctx, id, dispute); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}

	// Freeze the escrow under jury control: only ResolveDispute can settle it now.
	escrow.Status = types.EscrowStatusInJury
	if err := k.Escrow.Set(ctx, escrow.Id, escrow); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent("agntcoin_dispute_opened",
			sdk.NewAttribute("id", strconv.FormatUint(id, 10)),
			sdk.NewAttribute("escrow_id", strconv.FormatUint(msg.EscrowId, 10)),
			sdk.NewAttribute("opener", msg.Creator),
		),
	)
	return &types.MsgOpenDisputeResponse{Id: id}, nil
}
