package keeper

import (
	"context"

	"github.com/zoltankiss/agntcoin/x/agntcoin/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (q queryServer) Emission(ctx context.Context, req *types.QueryEmissionRequest) (*types.QueryEmissionResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	emission, err := q.k.peekEmissionState(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &types.QueryEmissionResponse{
		Emission:           emission,
		RemainingScheduled: types.RemainingScheduled(emission),
		RemainingEpoch:     types.RemainingEpoch(emission),
	}, nil
}
