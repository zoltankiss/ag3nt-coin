package keeper

import (
	"context"
	"errors"

	"cosmossdk.io/collections"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/zoltankiss/agntcoin/x/agntcoin/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (q queryServer) ListGate(ctx context.Context, req *types.QueryAllGateRequest) (*types.QueryAllGateResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	gates, pageRes, err := query.CollectionPaginate(
		ctx,
		q.k.Gate,
		req.Pagination,
		func(_ uint64, value types.Gate) (types.Gate, error) {
			return value, nil
		},
	)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllGateResponse{Gate: gates, Pagination: pageRes}, nil
}

func (q queryServer) GetGate(ctx context.Context, req *types.QueryGetGateRequest) (*types.QueryGetGateResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	gate, err := q.k.Gate.Get(ctx, req.Id)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil, sdkerrors.ErrKeyNotFound
		}
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &types.QueryGetGateResponse{Gate: gate}, nil
}
