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

// ListDispute returns all jury cases (paginated). This is what makes the jury
// legible from the chain itself: a juror lists cases and filters for status
// "open" instead of depending on an out-of-band notification (the it12 jury
// reached jurors via a /feedback bridge because disputes were write-only).
func (q queryServer) ListDispute(ctx context.Context, req *types.QueryAllDisputeRequest) (*types.QueryAllDisputeResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	disputes, pageRes, err := query.CollectionPaginate(
		ctx,
		q.k.Dispute,
		req.Pagination,
		func(_ uint64, value types.Dispute) (types.Dispute, error) {
			return value, nil
		},
	)
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &types.QueryAllDisputeResponse{Dispute: disputes, Pagination: pageRes}, nil
}

// GetDispute returns a single jury case by id (votes included).
func (q queryServer) GetDispute(ctx context.Context, req *types.QueryGetDisputeRequest) (*types.QueryGetDisputeResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	dispute, err := q.k.Dispute.Get(ctx, req.Id)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil, sdkerrors.ErrKeyNotFound
		}
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &types.QueryGetDisputeResponse{Dispute: dispute}, nil
}
