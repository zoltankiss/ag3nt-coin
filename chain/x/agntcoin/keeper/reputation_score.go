package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/zoltankiss/agntcoin/x/agntcoin/types"
)

func (k Keeper) reputationScores(ctx context.Context) (map[string]float64, error) {
	vouches := make([]types.Vouch, 0)
	if err := k.Vouch.Walk(ctx, nil, func(_ uint64, v types.Vouch) (bool, error) {
		vouches = append(vouches, v)
		return false, nil
	}); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}

	jobs := make([]JobEdge, 0)
	if err := k.Escrow.Walk(ctx, nil, func(_ uint64, e types.Escrow) (bool, error) {
		if e.Status == types.EscrowStatusReleased {
			jobs = append(jobs, JobEdge{Payer: e.Payer, Payee: e.Payee, Amount: e.Amount})
		}
		return false, nil
	}); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}

	if err := k.Dispute.Walk(ctx, nil, func(_ uint64, d types.Dispute) (bool, error) {
		if d.Resolution != types.DisputeResolutionAccept {
			return false, nil
		}
		e, err := k.Escrow.Get(ctx, d.EscrowId)
		if err != nil {
			return false, nil
		}
		for _, v := range d.Votes {
			if v.Accept {
				jobs = append(jobs, JobEdge{Payer: v.Juror, Payee: e.Payee, Amount: e.Amount})
			}
		}
		return false, nil
	}); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}

	params, err := k.Params.Get(ctx)
	if err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}
	return Reputation(vouches, jobs, resolveAnchors(params.Anchors)), nil
}

func (k Keeper) reputationScore(ctx context.Context, address string) (float64, error) {
	scores, err := k.reputationScores(ctx)
	if err != nil {
		return 0, err
	}
	return scores[address], nil
}
