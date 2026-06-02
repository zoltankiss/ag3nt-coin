package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zoltankiss/agntcoin/testutil/sample"
	"github.com/zoltankiss/agntcoin/x/agntcoin/keeper"
	"github.com/zoltankiss/agntcoin/x/agntcoin/types"
)

// setJurors points the v0 juror set (= Params.Anchors, the founder/trust roots)
// at the given addresses.
func setJurors(t *testing.T, f *fixture, jurors ...string) {
	t.Helper()
	require.NoError(t, f.keeper.Params.Set(f.ctx, types.Params{Anchors: jurors}))
}

// submittedEscrow locks a 400 escrow payer→payee and submits it — the delivered,
// contested state from which a jury case is opened.
func submittedEscrow(t *testing.T, f *fixture, ms types.MsgServer) uint64 {
	t.Helper()
	id := lockFresh(t, f, ms)
	_, err := ms.SubmitEscrow(setBlockTime(f.ctx, 110), &types.MsgSubmitEscrow{Creator: addrPayee, Id: id})
	require.NoError(t, err)
	return id
}

// Jury ACCEPT: a majority-accept verdict releases the escrow to the payee — the
// recourse the it11 honest worker lacked when a buyer's broken tests blocked
// genuinely good (but un-objectively-testable) work.
func TestJuryAcceptReleasesToPayee(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	juror := sample.AccAddress()
	setJurors(t, f, juror)
	seedAccount(t, f, addrPayer, 1000)
	seedAccount(t, f, addrPayee, 0)
	eid := submittedEscrow(t, f, ms)

	// Buyer escalates to the jury; the escrow freezes under jury control.
	od, err := ms.OpenDispute(f.ctx, &types.MsgOpenDispute{Creator: addrPayer, EscrowId: eid, Reason: "can't verify; want a ruling"})
	require.NoError(t, err)
	esc, _ := f.keeper.Escrow.Get(f.ctx, eid)
	require.Equal(t, types.EscrowStatusInJury, esc.Status)

	// Neither side can unilaterally settle while it is in_jury.
	_, err = ms.ReleaseEscrow(setBlockTime(f.ctx, 120), &types.MsgReleaseEscrow{Creator: addrPayer, Id: eid})
	require.Error(t, err)

	// Juror votes accept; resolve releases to the payee.
	_, err = ms.CastVote(f.ctx, &types.MsgCastVote{Creator: juror, DisputeId: od.Id, Accept: true})
	require.NoError(t, err)
	rd, err := ms.ResolveDispute(f.ctx, &types.MsgResolveDispute{Creator: addrPayer, DisputeId: od.Id})
	require.NoError(t, err)
	require.Equal(t, types.DisputeResolutionAccept, rd.Resolution)
	require.Equal(t, uint64(400), balanceOf(t, f, addrPayee))
	esc, _ = f.keeper.Escrow.Get(f.ctx, eid)
	require.Equal(t, types.EscrowStatusReleased, esc.Status)
}

// Jury REJECT: a majority-reject verdict refunds the escrow to the payer.
func TestJuryRejectRefundsToPayer(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	juror := sample.AccAddress()
	setJurors(t, f, juror)
	seedAccount(t, f, addrPayer, 1000) // 1000 - 400 locked = 600 held after lock
	seedAccount(t, f, addrPayee, 0)
	eid := submittedEscrow(t, f, ms)

	od, err := ms.OpenDispute(f.ctx, &types.MsgOpenDispute{Creator: addrPayer, EscrowId: eid, Reason: "slop"})
	require.NoError(t, err)
	_, err = ms.CastVote(f.ctx, &types.MsgCastVote{Creator: juror, DisputeId: od.Id, Accept: false})
	require.NoError(t, err)
	rd, err := ms.ResolveDispute(f.ctx, &types.MsgResolveDispute{Creator: addrPayer, DisputeId: od.Id})
	require.NoError(t, err)
	require.Equal(t, types.DisputeResolutionReject, rd.Resolution)
	require.Equal(t, uint64(1000), balanceOf(t, f, addrPayer)) // 600 + 400 refunded
	require.Equal(t, uint64(0), balanceOf(t, f, addrPayee))
	esc, _ := f.keeper.Escrow.Get(f.ctx, eid)
	require.Equal(t, types.EscrowStatusRefunded, esc.Status)
}

// Guards: a non-juror cannot vote, a juror cannot double-vote, and a dispute
// with no strict majority cannot resolve.
func TestJuryGuards(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	juror := sample.AccAddress()
	setJurors(t, f, juror)
	seedAccount(t, f, addrPayer, 1000)
	seedAccount(t, f, addrPayee, 0)
	eid := submittedEscrow(t, f, ms)
	od, err := ms.OpenDispute(f.ctx, &types.MsgOpenDispute{Creator: addrPayee, EscrowId: eid, Reason: "rule please"})
	require.NoError(t, err)

	// Non-juror (the payer) cannot vote.
	_, err = ms.CastVote(f.ctx, &types.MsgCastVote{Creator: addrPayer, DisputeId: od.Id, Accept: true})
	require.Error(t, err)

	// No votes yet → cannot resolve.
	_, err = ms.ResolveDispute(f.ctx, &types.MsgResolveDispute{Creator: addrPayer, DisputeId: od.Id})
	require.Error(t, err)

	// Juror votes once; a second vote by the same juror is rejected.
	_, err = ms.CastVote(f.ctx, &types.MsgCastVote{Creator: juror, DisputeId: od.Id, Accept: true})
	require.NoError(t, err)
	_, err = ms.CastVote(f.ctx, &types.MsgCastVote{Creator: juror, DisputeId: od.Id, Accept: false})
	require.Error(t, err)
}
