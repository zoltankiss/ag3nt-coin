package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zoltankiss/agntcoin/x/agntcoin/keeper"
	"github.com/zoltankiss/agntcoin/x/agntcoin/types"
)

// The bond tests reuse the fixed bech32 test addresses from
// msg_server_escrow_test.go in bond roles:
//   addrPayer   -> the poster (e.g. a worker staking a worker-bond)
//   addrPayee   -> the slasher (the neutral adjudicator / anchor)
//   addrNewee   -> the beneficiary on slash (e.g. the stranded buyer)
//   addrOutside -> an unauthorized third party

func TestPostBondLocksCollateral(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	seedAccount(t, f, addrPayer, 100)

	resp, err := ms.PostBond(f.ctx, &types.MsgPostBond{
		Creator: addrPayer,
		Amount:  50,
		Purpose: "workerbond:task-1",
		Slasher: addrPayee,
		Ref:     "task-1",
	})
	require.NoError(t, err)
	require.Equal(t, uint64(0), resp.Id)

	// Poster debited immediately; the collateral is protocol-held.
	require.Equal(t, uint64(50), balanceOf(t, f, addrPayer))

	bond, err := f.keeper.Bond.Get(f.ctx, resp.Id)
	require.NoError(t, err)
	require.Equal(t, types.BondStatusActive, bond.Status)
	require.Equal(t, addrPayer, bond.Poster)
	require.Equal(t, uint64(50), bond.Amount)
	require.Equal(t, addrPayee, bond.Slasher)
	require.Equal(t, "workerbond:task-1", bond.Purpose)

	// Bad input is rejected: zero amount, self-slasher, insufficient balance.
	_, err = ms.PostBond(f.ctx, &types.MsgPostBond{Creator: addrPayer, Amount: 0, Slasher: addrPayee})
	require.Error(t, err)
	_, err = ms.PostBond(f.ctx, &types.MsgPostBond{Creator: addrPayer, Amount: 10, Slasher: addrPayer})
	require.Error(t, err)
	_, err = ms.PostBond(f.ctx, &types.MsgPostBond{Creator: addrPayer, Amount: 1000, Slasher: addrPayee})
	require.Error(t, err)
}

func TestReleaseBondRefundsPoster(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	seedAccount(t, f, addrPayer, 100)

	resp, err := ms.PostBond(f.ctx, &types.MsgPostBond{
		Creator: addrPayer, Amount: 50, Purpose: "workerbond:task-1", Slasher: addrPayee, Ref: "task-1",
	})
	require.NoError(t, err)

	// The slasher exonerates: the poster is made whole.
	_, err = ms.ReleaseBond(f.ctx, &types.MsgReleaseBond{Creator: addrPayee, Id: resp.Id})
	require.NoError(t, err)
	require.Equal(t, uint64(100), balanceOf(t, f, addrPayer))

	bond, err := f.keeper.Bond.Get(f.ctx, resp.Id)
	require.NoError(t, err)
	require.Equal(t, types.BondStatusReleased, bond.Status)

	// Resolves once (I4): a released bond cannot be slashed or re-released.
	_, err = ms.SlashBond(f.ctx, &types.MsgSlashBond{Creator: addrPayee, Id: resp.Id, Beneficiary: addrNewee})
	require.Error(t, err)
	_, err = ms.ReleaseBond(f.ctx, &types.MsgReleaseBond{Creator: addrPayee, Id: resp.Id})
	require.Error(t, err)
	require.Equal(t, uint64(100), balanceOf(t, f, addrPayer))
}

func TestSlashBondPaysBeneficiary(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	seedAccount(t, f, addrPayer, 100)
	seedAccount(t, f, addrNewee, 7)

	resp, err := ms.PostBond(f.ctx, &types.MsgPostBond{
		Creator: addrPayer, Amount: 50, Purpose: "workerbond:task-1", Slasher: addrPayee, Ref: "task-1",
	})
	require.NoError(t, err)

	// The slasher punishes: the beneficiary is credited, the poster is NOT refunded.
	_, err = ms.SlashBond(f.ctx, &types.MsgSlashBond{Creator: addrPayee, Id: resp.Id, Beneficiary: addrNewee})
	require.NoError(t, err)
	require.Equal(t, uint64(50), balanceOf(t, f, addrPayer))
	require.Equal(t, uint64(57), balanceOf(t, f, addrNewee))

	bond, err := f.keeper.Bond.Get(f.ctx, resp.Id)
	require.NoError(t, err)
	require.Equal(t, types.BondStatusSlashed, bond.Status)

	// Conservation (I3): poster's loss == beneficiary's gain == amount, moved once.
	// Resolves once (I4): no double-slash / release-after-slash.
	_, err = ms.SlashBond(f.ctx, &types.MsgSlashBond{Creator: addrPayee, Id: resp.Id, Beneficiary: addrNewee})
	require.Error(t, err)
	_, err = ms.ReleaseBond(f.ctx, &types.MsgReleaseBond{Creator: addrPayee, Id: resp.Id})
	require.Error(t, err)
	require.Equal(t, uint64(50), balanceOf(t, f, addrPayer))
	require.Equal(t, uint64(57), balanceOf(t, f, addrNewee))
}

func TestPosterCannotSelfRelease(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	seedAccount(t, f, addrPayer, 100)

	resp, err := ms.PostBond(f.ctx, &types.MsgPostBond{
		Creator: addrPayer, Amount: 50, Purpose: "workerbond:task-1", Slasher: addrPayee, Ref: "task-1",
	})
	require.NoError(t, err)

	// I1 — the deterrent: the poster's own ReleaseBond is rejected and the
	// collateral stays locked. Without this a bond is just a refundable
	// deposit a Sybil could post and withdraw at will.
	_, err = ms.ReleaseBond(f.ctx, &types.MsgReleaseBond{Creator: addrPayer, Id: resp.Id})
	require.Error(t, err)
	require.Equal(t, uint64(50), balanceOf(t, f, addrPayer))

	bond, err := f.keeper.Bond.Get(f.ctx, resp.Id)
	require.NoError(t, err)
	require.Equal(t, types.BondStatusActive, bond.Status)
}

func TestSlashByUnauthorizedRejected(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	seedAccount(t, f, addrPayer, 100)
	seedAccount(t, f, addrOutside, 100)

	resp, err := ms.PostBond(f.ctx, &types.MsgPostBond{
		Creator: addrPayer, Amount: 50, Purpose: "workerbond:task-1", Slasher: addrPayee, Ref: "task-1",
	})
	require.NoError(t, err)

	// I2: neither an outsider nor the poster itself may slash; nothing moves.
	_, err = ms.SlashBond(f.ctx, &types.MsgSlashBond{Creator: addrOutside, Id: resp.Id, Beneficiary: addrOutside})
	require.Error(t, err)
	_, err = ms.SlashBond(f.ctx, &types.MsgSlashBond{Creator: addrPayer, Id: resp.Id, Beneficiary: addrPayer})
	require.Error(t, err)

	require.Equal(t, uint64(50), balanceOf(t, f, addrPayer))
	require.Equal(t, uint64(100), balanceOf(t, f, addrOutside))
	bond, err := f.keeper.Bond.Get(f.ctx, resp.Id)
	require.NoError(t, err)
	require.Equal(t, types.BondStatusActive, bond.Status)
}
