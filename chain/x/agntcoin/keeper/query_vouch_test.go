package keeper_test

import (
	"context"
	"strconv"
	"testing"

	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/zoltankiss/agntcoin/x/agntcoin/keeper"
	"github.com/zoltankiss/agntcoin/x/agntcoin/types"
)

func createNVouch(keeper keeper.Keeper, ctx context.Context, n int) []types.Vouch {
	items := make([]types.Vouch, n)
	for i := range items {
		iu := uint64(i)
		items[i].Id = iu
		items[i].FromAddr = strconv.Itoa(i)
		items[i].ToAddr = strconv.Itoa(i)
		items[i].Weight = uint64(i)
		_ = keeper.Vouch.Set(ctx, iu, items[i])
		_ = keeper.VouchSeq.Set(ctx, iu)
	}
	return items
}

func TestVouchQuerySingle(t *testing.T) {
	f := initFixture(t)
	qs := keeper.NewQueryServerImpl(f.keeper)
	msgs := createNVouch(f.keeper, f.ctx, 2)
	tests := []struct {
		desc     string
		request  *types.QueryGetVouchRequest
		response *types.QueryGetVouchResponse
		err      error
	}{
		{
			desc:     "First",
			request:  &types.QueryGetVouchRequest{Id: msgs[0].Id},
			response: &types.QueryGetVouchResponse{Vouch: msgs[0]},
		},
		{
			desc:     "Second",
			request:  &types.QueryGetVouchRequest{Id: msgs[1].Id},
			response: &types.QueryGetVouchResponse{Vouch: msgs[1]},
		},
		{
			desc:    "KeyNotFound",
			request: &types.QueryGetVouchRequest{Id: uint64(len(msgs))},
			err:     sdkerrors.ErrKeyNotFound,
		},
		{
			desc: "InvalidRequest",
			err:  status.Error(codes.InvalidArgument, "invalid request"),
		},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			response, err := qs.GetVouch(f.ctx, tc.request)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
			} else {
				require.NoError(t, err)
				require.EqualExportedValues(t, tc.response, response)
			}
		})
	}
}

func TestVouchQueryPaginated(t *testing.T) {
	f := initFixture(t)
	qs := keeper.NewQueryServerImpl(f.keeper)
	msgs := createNVouch(f.keeper, f.ctx, 5)

	request := func(next []byte, offset, limit uint64, total bool) *types.QueryAllVouchRequest {
		return &types.QueryAllVouchRequest{
			Pagination: &query.PageRequest{
				Key:        next,
				Offset:     offset,
				Limit:      limit,
				CountTotal: total,
			},
		}
	}
	t.Run("ByOffset", func(t *testing.T) {
		step := 2
		for i := 0; i < len(msgs); i += step {
			resp, err := qs.ListVouch(f.ctx, request(nil, uint64(i), uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.Vouch), step)
			require.Subset(t, msgs, resp.Vouch)
		}
	})
	t.Run("ByKey", func(t *testing.T) {
		step := 2
		var next []byte
		for i := 0; i < len(msgs); i += step {
			resp, err := qs.ListVouch(f.ctx, request(next, 0, uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.Vouch), step)
			require.Subset(t, msgs, resp.Vouch)
			next = resp.Pagination.NextKey
		}
	})
	t.Run("Total", func(t *testing.T) {
		resp, err := qs.ListVouch(f.ctx, request(nil, 0, 0, true))
		require.NoError(t, err)
		require.Equal(t, len(msgs), int(resp.Pagination.Total))
		require.EqualExportedValues(t, msgs, resp.Vouch)
	})
	t.Run("InvalidRequest", func(t *testing.T) {
		_, err := qs.ListVouch(f.ctx, nil)
		require.ErrorIs(t, err, status.Error(codes.InvalidArgument, "invalid request"))
	})
}
