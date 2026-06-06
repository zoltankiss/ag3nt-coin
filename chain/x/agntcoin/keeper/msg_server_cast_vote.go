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

	// Voting requires a slashable JUROR-STAKE (jury-v1, it20): a free vote lets a
	// colluding juror push fraudulent verdicts at no cost. The juror must be
	// registered with sufficient balance; the stake is debited now and settled in
	// ResolveDispute (released if coherent with the verdict, slashed to the wronged
	// party if not). Protocol-held (slasher empty) — only ResolveDispute settles it.
	if msg.StakeAmount < types.MinJurorStake {
		return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "voting requires a juror-stake of at least %d", types.MinJurorStake)
	}
	juror, err := k.Account.Get(ctx, msg.Creator)
	if err != nil || !juror.Registered {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "juror not registered")
	}
	if juror.Balance < msg.StakeAmount {
		return nil, errorsmod.Wrap(sdkerrors.ErrInsufficientFunds, "insufficient balance for juror stake")
	}
	juror.Balance -= msg.StakeAmount
	if err := k.Account.Set(ctx, msg.Creator, juror); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}
	bondID, err := k.BondSeq.Next(ctx)
	if err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}
	if err := k.Bond.Set(ctx, bondID, types.Bond{
		Id:      bondID,
		Poster:  msg.Creator,
		Amount:  msg.StakeAmount,
		Purpose: "jurorstake:" + strconv.FormatUint(dispute.Id, 10),
		Slasher: "", // protocol-settled by ResolveDispute
		Status:  types.BondStatusActive,
		Ref:     strconv.FormatUint(dispute.Id, 10),
	}); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}

	dispute.Votes = append(dispute.Votes, &types.Vote{Juror: msg.Creator, Accept: msg.Accept, BondId: bondID})
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
