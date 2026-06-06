package keeper_test

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zoltankiss/agntcoin/testutil/sample"
	"github.com/zoltankiss/agntcoin/x/agntcoin/keeper"
	"github.com/zoltankiss/agntcoin/x/agntcoin/types"
)

// repScore queries a node's reputation through the query server and parses it.
func repScore(t *testing.T, f *fixture, qs types.QueryServer, addr string) float64 {
	t.Helper()
	resp, err := qs.Reputation(f.ctx, &types.QueryReputationRequest{Address: addr})
	require.NoError(t, err)
	v, err := strconv.ParseFloat(resp.Score, 64)
	require.NoError(t, err)
	return v
}

// Money-not-standing fix (it12): a jury-ACCEPT by the anchor-juror confers
// anchor-rooted reputation on the worker — the recognition a paid-but-unvouched
// worker otherwise never earns when its buyer is not anchor-connected. A/B
// within one test: the SAME non-anchor buyer pays two workers — one by a normal
// release (control), one via a jury-accept — and only the jury-endorsed worker
// gains standing.
func TestJuryAcceptConfersAnchorRootedReputation(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	qs := keeper.NewQueryServerImpl(f.keeper)

	// The anchor/trust-root is the juror — NOT the buyer (addrPayer), who is an
	// ordinary funded account with no anchor connection.
	juror := sample.AccAddress()
	setJurors(t, f, juror)
	seedAccount(t, f, addrPayer, 2000)

	// CONTROL: buyer pays worker2 by a normal release (no jury). worker2 is paid
	// but, because the buyer isn't anchor-rooted, earns ~no standing.
	worker2 := sample.AccAddress()
	lc := setBlockTime(f.ctx, 100)
	lr, err := ms.LockEscrow(lc, &types.MsgLockEscrow{Creator: addrPayer, Payee: worker2, Amount: 400, Ref: "control", DisputeSeconds: 50})
	require.NoError(t, err)
	_, err = ms.ReleaseEscrow(lc, &types.MsgReleaseEscrow{Creator: addrPayer, Id: lr.Id})
	require.NoError(t, err)

	// JURY PATH: buyer hires addrPayee, who delivers; buyer escalates; the
	// anchor-juror accepts on the merits.
	eid := submittedEscrow(t, f, ms)
	od, err := ms.OpenDispute(f.ctx, &types.MsgOpenDispute{Creator: addrPayer, EscrowId: eid, Reason: "can't verify; rule please", BondAmount: 100})
	require.NoError(t, err)
	_, err = ms.CastVote(f.ctx, &types.MsgCastVote{Creator: juror, DisputeId: od.Id, Accept: true})
	require.NoError(t, err)
	_, err = ms.ResolveDispute(f.ctx, &types.MsgResolveDispute{Creator: addrPayer, DisputeId: od.Id})
	require.NoError(t, err)

	// Both workers got PAID the same 400 escrow; addrPayee additionally collects
	// the buyer's 100 dispute-bond, slashed to it because the buyer cry-wolfed
	// (disputed good work the jury then accepted) — so 500 total.
	require.Equal(t, uint64(400), balanceOf(t, f, worker2))
	require.Equal(t, uint64(500), balanceOf(t, f, addrPayee))

	// But only the jury-endorsed worker earns STANDING: the anchor-juror's
	// accept is an anchor-rooted endorsement; the control worker's buyer-only
	// release carries ~no anchor-rooted rank.
	juryRep := repScore(t, f, qs, addrPayee)
	controlRep := repScore(t, f, qs, worker2)
	require.Greater(t, juryRep, 0.0, "jury-accepted worker must earn anchor-rooted reputation")
	require.Greater(t, juryRep, controlRep*10, "jury endorsement must dominate a non-anchor buyer's bare release")
}

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
	od, err := ms.OpenDispute(f.ctx, &types.MsgOpenDispute{Creator: addrPayer, EscrowId: eid, Reason: "can't verify; want a ruling", BondAmount: 100})
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
	// 400 escrow + 100 dispute-bond slashed to the payee (payer cry-wolfed: jury accepted).
	require.Equal(t, uint64(500), balanceOf(t, f, addrPayee))
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

	od, err := ms.OpenDispute(f.ctx, &types.MsgOpenDispute{Creator: addrPayer, EscrowId: eid, Reason: "slop", BondAmount: 100})
	require.NoError(t, err)
	_, err = ms.CastVote(f.ctx, &types.MsgCastVote{Creator: juror, DisputeId: od.Id, Accept: false})
	require.NoError(t, err)
	rd, err := ms.ResolveDispute(f.ctx, &types.MsgResolveDispute{Creator: addrPayer, DisputeId: od.Id})
	require.NoError(t, err)
	require.Equal(t, types.DisputeResolutionReject, rd.Resolution)
	// 600 held + 400 refunded; the 100 dispute-bond round-trips (payer's dispute upheld → released back).
	require.Equal(t, uint64(1000), balanceOf(t, f, addrPayer))
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
	seedAccount(t, f, addrPayee, 200) // funds the payee's dispute-bond
	eid := submittedEscrow(t, f, ms)
	od, err := ms.OpenDispute(f.ctx, &types.MsgOpenDispute{Creator: addrPayee, EscrowId: eid, Reason: "rule please", BondAmount: 100})
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
