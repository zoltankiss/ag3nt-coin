package keeper_test

import (
	"context"
	"strconv"
	"testing"

	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/zoltankiss/agntcoin/x/agntcoin/keeper"
	"github.com/zoltankiss/agntcoin/x/agntcoin/types"
)

func createNAccount(keeper keeper.Keeper, ctx context.Context, n int) []types.Account {
	items := make([]types.Account, n)
	for i := range items {
		items[i].Address = strconv.Itoa(i)
		items[i].Balance = uint64(i)
		items[i].Nonce = uint64(i)
		items[i].Registered = true
		items[i].FaucetClaimed = true
		_ = keeper.Account.Set(ctx, items[i].Address, items[i])
	}
	return items
}

func TestAccountQuerySingle(t *testing.T) {
	f := initFixture(t)
	qs := keeper.NewQueryServerImpl(f.keeper)
	msgs := createNAccount(f.keeper, f.ctx, 2)
	tests := []struct {
		desc     string
		request  *types.QueryGetAccountRequest
		response *types.QueryGetAccountResponse
		err      error
	}{
		{
			desc: "First",
			request: &types.QueryGetAccountRequest{
				Address: msgs[0].Address,
			},
			response: &types.QueryGetAccountResponse{Account: msgs[0]},
		},
		{
			desc: "Second",
			request: &types.QueryGetAccountRequest{
				Address: msgs[1].Address,
			},
			response: &types.QueryGetAccountResponse{Account: msgs[1]},
		},
		{
			desc: "KeyNotFound",
			request: &types.QueryGetAccountRequest{
				Address: strconv.Itoa(100000),
			},
			err: status.Error(codes.NotFound, "not found"),
		},
		{
			desc: "InvalidRequest",
			err:  status.Error(codes.InvalidArgument, "invalid request"),
		},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			response, err := qs.GetAccount(f.ctx, tc.request)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err)
			} else {
				require.NoError(t, err)
				require.EqualExportedValues(t, tc.response, response)
			}
		})
	}
}

func TestAccountQueryPaginated(t *testing.T) {
	f := initFixture(t)
	qs := keeper.NewQueryServerImpl(f.keeper)
	msgs := createNAccount(f.keeper, f.ctx, 5)

	request := func(next []byte, offset, limit uint64, total bool) *types.QueryAllAccountRequest {
		return &types.QueryAllAccountRequest{
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
			resp, err := qs.ListAccount(f.ctx, request(nil, uint64(i), uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.Account), step)
			require.Subset(t, msgs, resp.Account)
		}
	})
	t.Run("ByKey", func(t *testing.T) {
		step := 2
		var next []byte
		for i := 0; i < len(msgs); i += step {
			resp, err := qs.ListAccount(f.ctx, request(next, 0, uint64(step), false))
			require.NoError(t, err)
			require.LessOrEqual(t, len(resp.Account), step)
			require.Subset(t, msgs, resp.Account)
			next = resp.Pagination.NextKey
		}
	})
	t.Run("Total", func(t *testing.T) {
		resp, err := qs.ListAccount(f.ctx, request(nil, 0, 0, true))
		require.NoError(t, err)
		require.Equal(t, len(msgs), int(resp.Pagination.Total))
		require.EqualExportedValues(t, msgs, resp.Account)
	})
	t.Run("InvalidRequest", func(t *testing.T) {
		_, err := qs.ListAccount(f.ctx, nil)
		require.ErrorIs(t, err, status.Error(codes.InvalidArgument, "invalid request"))
	})
}
