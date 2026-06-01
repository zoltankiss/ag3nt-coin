package keeper

import (
	"context"
	"strconv"

	errorsmod "cosmossdk.io/errors"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/zoltankiss/agntcoin/x/agntcoin/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (q queryServer) Reputation(ctx context.Context, req *types.QueryReputationRequest) (*types.QueryReputationResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	if _, err := q.k.addressCodec.StringToBytes(req.Address); err != nil {
		return nil, errorsmod.Wrap(err, "invalid address")
	}

	vouches := make([]types.Vouch, 0)
	err := q.k.Vouch.Walk(ctx, nil, func(_ uint64, v types.Vouch) (bool, error) {
		vouches = append(vouches, v)
		return false, nil
	})
	if err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}

	scores := PageRank(vouches)
	score, ok := scores[req.Address]
	if !ok {
		if _, err := q.k.Account.Get(ctx, req.Address); err != nil {
			return nil, status.Error(codes.NotFound, "address not in graph and not registered")
		}
		score = 0
	}
	return &types.QueryReputationResponse{Score: strconv.FormatFloat(score, 'f', -1, 64)}, nil
}
