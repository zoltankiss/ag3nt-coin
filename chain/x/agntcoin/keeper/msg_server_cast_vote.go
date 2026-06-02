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

// CastVote records an eligible juror's verdict on an open dispute. Juror
// eligibility (v0) = membership in the anchor set (Params.Anchors — the founder
// / trust roots at genesis; a reputation/stake-gated registry comes later).
// One vote per juror. Tally + settlement happen in ResolveDispute.
func (k msgServer) CastVote(ctx context.Context, msg *types.MsgCastVote) (*types.MsgCastVoteResponse, error) {
	if _, err := k.addressCodec.StringToBytes(msg.Creator); err != nil {
		return nil, errorsmod.Wrap(err, "invalid creator address")
	}

	dispute, err := k.Dispute.Get(ctx, msg.DisputeId)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil, errorsmod.Wrap(sdkerrors.ErrKeyNotFound, "dispute not found")
		}
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}
	if dispute.Status != types.DisputeStatusOpen {
		return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "dispute is not open (status=%s)", dispute.Status)
	}

	// Eligibility: the caller must be an anchor (v0 juror set).
	params, err := k.Params.Get(ctx)
	if err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}
	if !isJuror(msg.Creator, resolveAnchors(params.Anchors)) {
		return nil, errorsmod.Wrap(sdkerrors.ErrUnauthorized, "caller is not an eligible juror")
	}

	// One vote per juror.
	for _, v := range dispute.Votes {
		if v.Juror == msg.Creator {
			return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "juror has already voted on this dispute")
		}
	}
	dispute.Votes = append(dispute.Votes, &types.Vote{Juror: msg.Creator, Accept: msg.Accept})
	if err := k.Dispute.Set(ctx, dispute.Id, dispute); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent("agntcoin_dispute_vote",
			sdk.NewAttribute("dispute_id", strconv.FormatUint(dispute.Id, 10)),
			sdk.NewAttribute("juror", msg.Creator),
			sdk.NewAttribute("accept", strconv.FormatBool(msg.Accept)),
		),
	)
	return &types.MsgCastVoteResponse{}, nil
}

// isJuror reports whether addr is in the eligible juror set.
func isJuror(addr string, jurors []string) bool {
	for _, j := range jurors {
		if j == addr {
			return true
		}
	}
	return false
}
