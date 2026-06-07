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

func (q queryServer) ListScopedEvidenceVouch(ctx context.Context, req *types.QueryAllScopedEvidenceVouchRequest) (*types.QueryAllScopedEvidenceVouchResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	vouches, pageRes, err := query.CollectionPaginate(
		ctx,
		q.k.ScopedEvidenceVouch,
		req.Pagination,
		func(_ uint64, value types.ScopedEvidenceVouch) (types.ScopedEvidenceVouch, error) {
			return value, nil
		},
	)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllScopedEvidenceVouchResponse{ScopedEvidenceVouch: vouches, Pagination: pageRes}, nil
}

func (q queryServer) GetScopedEvidenceVouch(ctx context.Context, req *types.QueryGetScopedEvidenceVouchRequest) (*types.QueryGetScopedEvidenceVouchResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	vouch, err := q.k.ScopedEvidenceVouch.Get(ctx, req.Id)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil, sdkerrors.ErrKeyNotFound
		}
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &types.QueryGetScopedEvidenceVouchResponse{ScopedEvidenceVouch: vouch}, nil
}
