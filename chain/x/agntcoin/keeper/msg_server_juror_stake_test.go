package keeper_test

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zoltankiss/agntcoin/testutil/sample"
	"github.com/zoltankiss/agntcoin/x/agntcoin/keeper"
	"github.com/zoltankiss/agntcoin/x/agntcoin/types"
)

// jury-v1 (it20): voting requires a slashable juror-stake, resolving requires a
// real quorum (no 1-0 rush), and a juror incoherent with the resolved verdict is
// slashed to the wronged party. These are the chain-level GREEN of the it20
// collusion RED.

// openDisputed locks payer->payee 400, submits it, and opens a jury dispute with
// the required dispute-bond — the open state from which jurors vote.
func openDisputed(t *testing.T, f *fixture, ms types.MsgServer) uint64 {
	t.Helper()
	seedAccount(t, f, addrPayer, 2000) // 400 escrow + 100 dispute-bond + slack
	eid := submittedEscrow(t, f, ms)
	od, err := ms.OpenDispute(f.ctx, &types.MsgOpenDispute{Creator: addrPayer, EscrowId: eid, Reason: "rule please", BondAmount: 100})
	require.NoError(t, err)
	return od.Id
}

// Voting requires a stake of at least MinJurorStake, and it is debited + locked.
func TestVotingRequiresJurorStake(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	juror := sample.AccAddress()
	setJurors(t, f, juror)
	seedAccount(t, f, juror, 1000)
	did := openDisputed(t, f, ms)

	// No stake / sub-minimum stake → rejected (a free vote is no longer possible).
	_, err := ms.CastVote(f.ctx, &types.MsgCastVote{Creator: juror, DisputeId: did, Accept: true})
	require.Error(t, err)
	_, err = ms.CastVote(f.ctx, &types.MsgCastVote{Creator: juror, DisputeId: did, Accept: true, StakeAmount: types.MinJurorStake - 1})
	require.Error(t, err)

	// A sufficient stake is debited + recorded on the vote as a juror-stake bond.
	before := balanceOf(t, f, juror)
	_, err = ms.CastVote(f.ctx, &types.MsgCastVote{Creator: juror, DisputeId: did, Accept: true, StakeAmount: 150})
	require.NoError(t, err)
	require.Equal(t, before-150, balanceOf(t, f, juror))

	d, _ := f.keeper.Dispute.Get(f.ctx, did)
	require.Len(t, d.Votes, 1)
	b, err := f.keeper.Bond.Get(f.ctx, d.Votes[0].BondId)
	require.NoError(t, err)
	require.Equal(t, types.BondStatusActive, b.Status)
	require.Equal(t, "jurorstake:"+strconv.FormatUint(did, 10), b.Purpose)
	require.Equal(t, uint64(150), b.Amount)
}

// Quorum: with 3 eligible jurors, a single juror cannot vote + resolve a 1-0
// verdict — resolution requires a majority of the eligible set (2). This closes
// the rush-resolve capture the it20 RED targets.
func TestQuorumBlocksSingleJurorRush(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	jA, jB, jC := sample.AccAddress(), sample.AccAddress(), sample.AccAddress()
	setJurors(t, f, jA, jB, jC)
	for _, j := range []string{jA, jB, jC} {
		seedAccount(t, f, j, 1000)
	}
	did := openDisputed(t, f, ms)

	// jC alone votes + tries to rush-resolve 1-0 → blocked (quorum is 2 of 3).
	_, err := ms.CastVote(f.ctx, &types.MsgCastVote{Creator: jC, DisputeId: did, Accept: true, StakeAmount: 100})
	require.NoError(t, err)
	_, err = ms.ResolveDispute(f.ctx, &types.MsgResolveDispute{Creator: jC, DisputeId: did})
	require.Error(t, err, "a single juror must not be able to resolve before quorum")

	// A second juror votes → quorum reached → resolves.
	_, err = ms.CastVote(f.ctx, &types.MsgCastVote{Creator: jA, DisputeId: did, Accept: true, StakeAmount: 100})
	require.NoError(t, err)
	_, err = ms.ResolveDispute(f.ctx, &types.MsgResolveDispute{Creator: jA, DisputeId: did})
	require.NoError(t, err)
}

// Collusion GREEN: 2 honest jurors accept good work; 1 colluding juror votes
// against the verdict. The verdict is accept (2-1); the coherent jurors get their
// stakes back, the incoherent (colluding) juror is SLASHED to the wronged party.
func TestIncoherentJurorStakeSlashed(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	jA, jB, jC := sample.AccAddress(), sample.AccAddress(), sample.AccAddress()
	setJurors(t, f, jA, jB, jC)
	for _, j := range []string{jA, jB, jC} {
		seedAccount(t, f, j, 1000)
	}
	did := openDisputed(t, f, ms)

	_, err := ms.CastVote(f.ctx, &types.MsgCastVote{Creator: jA, DisputeId: did, Accept: true, StakeAmount: 100})  // honest
	require.NoError(t, err)
	_, err = ms.CastVote(f.ctx, &types.MsgCastVote{Creator: jB, DisputeId: did, Accept: true, StakeAmount: 100})   // honest
	require.NoError(t, err)
	_, err = ms.CastVote(f.ctx, &types.MsgCastVote{Creator: jC, DisputeId: did, Accept: false, StakeAmount: 100})  // colluder, against truth
	require.NoError(t, err)

	payeeBefore := balanceOf(t, f, addrPayee)
	rd, err := ms.ResolveDispute(f.ctx, &types.MsgResolveDispute{Creator: jA, DisputeId: did})
	require.NoError(t, err)
	require.Equal(t, types.DisputeResolutionAccept, rd.Resolution) // 2 accept > 1 reject

	// Coherent jurors made whole (stake returned); the colluder lost its stake.
	require.Equal(t, uint64(1000), balanceOf(t, f, jA))
	require.Equal(t, uint64(1000), balanceOf(t, f, jB))
	require.Equal(t, uint64(900), balanceOf(t, f, jC), "colluding juror's stake must be slashed")

	d, _ := f.keeper.Dispute.Get(f.ctx, did)
	st := map[string]string{}
	for _, v := range d.Votes {
		b, _ := f.keeper.Bond.Get(f.ctx, v.BondId)
		st[v.Juror] = b.Status
	}
	require.Equal(t, types.BondStatusReleased, st[jA])
	require.Equal(t, types.BondStatusReleased, st[jB])
	require.Equal(t, types.BondStatusSlashed, st[jC])

	// The payee (the verdict favored release to it) ends up with three credits:
	// the 400 escrow, the payer-opener's 100 dispute-bond (its dispute was rejected,
	// it17/it19), and the colluding juror's 100 slashed juror-stake — 600 total.
	require.Equal(t, payeeBefore+600, balanceOf(t, f, addrPayee))
}
