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

func (q queryServer) ListEscrow(ctx context.Context, req *types.QueryAllEscrowRequest) (*types.QueryAllEscrowResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	escrows, pageRes, err := query.CollectionPaginate(
		ctx,
		q.k.Escrow,
		req.Pagination,
		func(_ uint64, value types.Escrow) (types.Escrow, error) {
			return value, nil
		},
	)

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllEscrowResponse{Escrow: escrows, Pagination: pageRes}, nil
}

func (q queryServer) GetEscrow(ctx context.Context, req *types.QueryGetEscrowRequest) (*types.QueryGetEscrowResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	escrow, err := q.k.Escrow.Get(ctx, req.Id)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil, sdkerrors.ErrKeyNotFound
		}

		return nil, status.Error(codes.Internal, "internal error")
	}

	return &types.QueryGetEscrowResponse{Escrow: escrow}, nil
}
