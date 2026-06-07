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

func (q queryServer) ListContributionAward(ctx context.Context, req *types.QueryAllContributionAwardRequest) (*types.QueryAllContributionAwardResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	awards, pageRes, err := query.CollectionPaginate(
		ctx,
		q.k.ContributionAward,
		req.Pagination,
		func(_ uint64, value types.ContributionAward) (types.ContributionAward, error) {
			return value, nil
		},
	)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllContributionAwardResponse{ContributionAward: awards, Pagination: pageRes}, nil
}

func (q queryServer) GetContributionAward(ctx context.Context, req *types.QueryGetContributionAwardRequest) (*types.QueryGetContributionAwardResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	award, err := q.k.ContributionAward.Get(ctx, req.Id)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil, sdkerrors.ErrKeyNotFound
		}
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &types.QueryGetContributionAwardResponse{ContributionAward: award}, nil
}
