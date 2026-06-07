package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zoltankiss/agntcoin/testutil/sample"
	"github.com/zoltankiss/agntcoin/x/agntcoin/keeper"
	"github.com/zoltankiss/agntcoin/x/agntcoin/types"
)

func TestGateQueriesListAndGet(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	qs := keeper.NewQueryServerImpl(f.keeper)
	anchor := sample.AccAddress()
	setJurors(t, f, anchor)
	seedAccount(t, f, anchor, 0)

	resp, err := ms.PostGate(f.ctx, &types.MsgPostGate{
		Creator:     anchor,
		PayloadUri:  "https://github.com/zoltankiss/agnt-coin-forge-2/blob/abc123/gate.json",
		PayloadHash: hexHash("gate-payload"),
		GoldCommit:  hexHash("request_changes:security:secret"),
		Drip:        10,
		MaxAnswers:  3,
	})
	require.NoError(t, err)

	got, err := qs.GetGate(f.ctx, &types.QueryGetGateRequest{Id: resp.Id})
	require.NoError(t, err)
	require.Equal(t, resp.Id, got.Gate.Id)
	require.Equal(t, "open", got.Gate.Status)

	list, err := qs.ListGate(f.ctx, &types.QueryAllGateRequest{})
	require.NoError(t, err)
	require.Len(t, list.Gate, 1)
	require.Equal(t, resp.Id, list.Gate[0].Id)
}
