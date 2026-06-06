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

func (q queryServer) ListBond(ctx context.Context, req *types.QueryAllBondRequest) (*types.QueryAllBondResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	bonds, pageRes, err := query.CollectionPaginate(
		ctx,
		q.k.Bond,
		req.Pagination,
		func(_ uint64, value types.Bond) (types.Bond, error) {
			return value, nil
		},
	)

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllBondResponse{Bond: bonds, Pagination: pageRes}, nil
}

func (q queryServer) GetBond(ctx context.Context, req *types.QueryGetBondRequest) (*types.QueryGetBondResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	bond, err := q.k.Bond.Get(ctx, req.Id)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil, sdkerrors.ErrKeyNotFound
		}

		return nil, status.Error(codes.Internal, "internal error")
	}

	return &types.QueryGetBondResponse{Bond: bond}, nil
}
