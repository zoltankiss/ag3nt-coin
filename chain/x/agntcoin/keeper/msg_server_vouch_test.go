package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zoltankiss/agntcoin/x/agntcoin/keeper"
	"github.com/zoltankiss/agntcoin/x/agntcoin/types"
)

// vouchFor returns the caller's vouch for to_addr, or ok=false if none exists.
func vouchFor(t *testing.T, f *fixture, from, to string) (types.Vouch, bool) {
	t.Helper()
	var out types.Vouch
	found := false
	err := f.keeper.Vouch.Walk(f.ctx, nil, func(_ uint64, v types.Vouch) (bool, error) {
		if v.FromAddr == from && v.ToAddr == to {
			out = v
			found = true
			return true, nil
		}
		return false, nil
	})
	require.NoError(t, err)
	return out, found
}

func TestCastvouchDebitsAndLocksStake(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	seedAccount(t, f, addrPayer, 1000)
	seedAccount(t, f, addrPayee, 0)

	_, err := ms.Castvouch(f.ctx, &types.MsgCastvouch{
		Creator: addrPayer,
		ToAddr:  addrPayee,
		Weight:  50,
		Stake:   300,
	})
	require.NoError(t, err)

	// Voucher debited by the stake.
	require.Equal(t, uint64(700), balanceOf(t, f, addrPayer))

	// Stake is locked in the vouch record.
	v, ok := vouchFor(t, f, addrPayer, addrPayee)
	require.True(t, ok)
	require.Equal(t, uint64(300), v.Stake)
	require.Equal(t, uint64(50), v.Weight)
}

func TestCastvouchRejectsBelowMinStake(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	seedAccount(t, f, addrPayer, 1000)
	seedAccount(t, f, addrPayee, 0)

	_, err := ms.Castvouch(f.ctx, &types.MsgCastvouch{
		Creator: addrPayer,
		ToAddr:  addrPayee,
		Weight:  50,
		Stake:   types.MinVouchStake - 1,
	})
	require.Error(t, err)
	// No debit, no vouch created.
	require.Equal(t, uint64(1000), balanceOf(t, f, addrPayer))
	_, ok := vouchFor(t, f, addrPayer, addrPayee)
	require.False(t, ok)
}

func TestCastvouchRejectsInsufficientBalance(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	seedAccount(t, f, addrPayer, 200)
	seedAccount(t, f, addrPayee, 0)

	_, err := ms.Castvouch(f.ctx, &types.MsgCastvouch{
		Creator: addrPayer,
		ToAddr:  addrPayee,
		Weight:  50,
		Stake:   500,
	})
	require.Error(t, err)
	require.Equal(t, uint64(200), balanceOf(t, f, addrPayer))
	_, ok := vouchFor(t, f, addrPayer, addrPayee)
	require.False(t, ok)
}

func TestUnvouchReturnsStake(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	seedAccount(t, f, addrPayer, 1000)
	seedAccount(t, f, addrPayee, 0)

	_, err := ms.Castvouch(f.ctx, &types.MsgCastvouch{
		Creator: addrPayer, ToAddr: addrPayee, Weight: 50, Stake: 400,
	})
	require.NoError(t, err)
	require.Equal(t, uint64(600), balanceOf(t, f, addrPayer))

	_, err = ms.Unvouch(f.ctx, &types.MsgUnvouch{Creator: addrPayer, ToAddr: addrPayee})
	require.NoError(t, err)

	// Stake returned in full, vouch gone.
	require.Equal(t, uint64(1000), balanceOf(t, f, addrPayer))
	_, ok := vouchFor(t, f, addrPayer, addrPayee)
	require.False(t, ok)
}

func TestUnvouchRejectsMissingVouch(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	seedAccount(t, f, addrPayer, 1000)
	seedAccount(t, f, addrPayee, 0)

	_, err := ms.Unvouch(f.ctx, &types.MsgUnvouch{Creator: addrPayer, ToAddr: addrPayee})
	require.Error(t, err)
}

// Re-vouch (upsert) must return the old stake before debiting the new one, so
// the voucher is never double-charged.
func TestRevouchReturnsOldStakeNoDoubleCharge(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	seedAccount(t, f, addrPayer, 1000)
	seedAccount(t, f, addrPayee, 0)

	_, err := ms.Castvouch(f.ctx, &types.MsgCastvouch{
		Creator: addrPayer, ToAddr: addrPayee, Weight: 50, Stake: 300,
	})
	require.NoError(t, err)
	require.Equal(t, uint64(700), balanceOf(t, f, addrPayer))

	// Re-vouch with a larger stake: net delta is 500-300 = 200.
	_, err = ms.Castvouch(f.ctx, &types.MsgCastvouch{
		Creator: addrPayer, ToAddr: addrPayee, Weight: 80, Stake: 500,
	})
	require.NoError(t, err)
	// 1000 - 500 (not 1000 - 300 - 500).
	require.Equal(t, uint64(500), balanceOf(t, f, addrPayer))

	v, ok := vouchFor(t, f, addrPayer, addrPayee)
	require.True(t, ok)
	require.Equal(t, uint64(500), v.Stake)
	require.Equal(t, uint64(80), v.Weight)

	// Re-vouch with a smaller stake refunds the difference.
	_, err = ms.Castvouch(f.ctx, &types.MsgCastvouch{
		Creator: addrPayer, ToAddr: addrPayee, Weight: 10, Stake: 100,
	})
	require.NoError(t, err)
	// 1000 - 100.
	require.Equal(t, uint64(900), balanceOf(t, f, addrPayer))
}

func TestCastvouchRejectsSelfAndUnregistered(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	seedAccount(t, f, addrPayer, 1000)

	// self-vouch
	_, err := ms.Castvouch(f.ctx, &types.MsgCastvouch{Creator: addrPayer, ToAddr: addrPayer, Weight: 50, Stake: 200})
	require.Error(t, err)

	// vouchee not registered
	_, err = ms.Castvouch(f.ctx, &types.MsgCastvouch{Creator: addrPayer, ToAddr: addrOutside, Weight: 50, Stake: 200})
	require.Error(t, err)
	require.Equal(t, uint64(1000), balanceOf(t, f, addrPayer))
}

// A heavily-staked vouchee must outrank a lightly-staked one even at equal
// 1..100 weight — reputation now tracks locked stake, not the weight field.
func TestPageRankReflectsStakeNotWeight(t *testing.T) {
	// voucher "a" splits its trust: equal weight, but 9x the stake on b vs c.
	scores := keeper.PageRank([]types.Vouch{
		{FromAddr: "a", ToAddr: "b", Weight: 50, Stake: 900},
		{FromAddr: "a", ToAddr: "c", Weight: 50, Stake: 100},
	})
	require.Greater(t, scores["b"], scores["c"],
		"heavily-staked vouchee b should outrank lightly-staked c at equal weight")
}
