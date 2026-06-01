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

func (q queryServer) ListVouch(ctx context.Context, req *types.QueryAllVouchRequest) (*types.QueryAllVouchResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	vouchs, pageRes, err := query.CollectionPaginate(
		ctx,
		q.k.Vouch,
		req.Pagination,
		func(_ uint64, value types.Vouch) (types.Vouch, error) {
			return value, nil
		},
	)

	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllVouchResponse{Vouch: vouchs, Pagination: pageRes}, nil
}

func (q queryServer) GetVouch(ctx context.Context, req *types.QueryGetVouchRequest) (*types.QueryGetVouchResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	vouch, err := q.k.Vouch.Get(ctx, req.Id)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil, sdkerrors.ErrKeyNotFound
		}

		return nil, status.Error(codes.Internal, "internal error")
	}

	return &types.QueryGetVouchResponse{Vouch: vouch}, nil
}
