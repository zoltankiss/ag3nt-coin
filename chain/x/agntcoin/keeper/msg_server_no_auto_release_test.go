package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zoltankiss/agntcoin/testutil/sample"
	"github.com/zoltankiss/agntcoin/x/agntcoin/keeper"
	"github.com/zoltankiss/agntcoin/x/agntcoin/types"
)

// lockNoAutoReleaseSubmitted locks a 400 escrow payer→payee with no_auto_release
// set, then submits it — the jury-bound, delivered-and-contested state a scammer
// would try to drain by waiting out the deadline.
func lockNoAutoReleaseSubmitted(t *testing.T, f *fixture, ms types.MsgServer) uint64 {
	t.Helper()
	resp, err := ms.LockEscrow(setBlockTime(f.ctx, 100), &types.MsgLockEscrow{
		Creator: addrPayer, Payee: addrPayee, Amount: 400, Ref: "r", DisputeSeconds: 50, NoAutoRelease: true,
	})
	require.NoError(t, err)
	_, err = ms.SubmitEscrow(setBlockTime(f.ctx, 110), &types.MsgSubmitEscrow{Creator: addrPayee, Id: resp.Id})
	require.NoError(t, err)
	return resp.Id
}

// it13 finding #19: a jury-bound (no_auto_release) escrow closes the deadline
// self-release jury-bypass — a scammer cannot deliver slop, submit, and wait out
// the deadline to self-pay, dodging the jury. The honest-worker protection is
// preserved THROUGH the jury: the payee opens a dispute and an honest delivery is
// jury-accepted (slop would be rejected and the buyer refunded).
func TestNoAutoReleaseBlocksDeadlineBypass(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	juror := sample.AccAddress()
	setJurors(t, f, juror)
	seedAccount(t, f, juror, 1000) // jury-v1: fund the juror-stake
	seedAccount(t, f, addrPayer, 1000)
	seedAccount(t, f, addrPayee, 100) // funds the payee's dispute-bond (returned when upheld)

	eid := lockNoAutoReleaseSubmitted(t, f, ms)

	// Past the deadline (t=200 > lock@100 + 50), the payee attempts the #19 bypass:
	// self-release its own submitted escrow. On a jury-bound escrow it must FAIL.
	_, err := ms.ReleaseEscrow(setBlockTime(f.ctx, 200), &types.MsgReleaseEscrow{Creator: addrPayee, Id: eid})
	require.Error(t, err)
	require.Equal(t, uint64(100), balanceOf(t, f, addrPayee)) // still just the seed — not paid by waiting out the clock

	// Honest-worker recourse preserved: the payee opens a dispute (posting a 100
	// bond), the anchor-juror reviews and accepts, and resolution pays the payee —
	// via the jury, not a bypass. The dispute is upheld, so the bond returns.
	od, err := ms.OpenDispute(setBlockTime(f.ctx, 210), &types.MsgOpenDispute{Creator: addrPayee, EscrowId: eid, Reason: "buyer ghosted; please rule on the merits", BondAmount: 100})
	require.NoError(t, err)
	_, err = ms.CastVote(f.ctx, &types.MsgCastVote{Creator: juror, DisputeId: od.Id, Accept: true, StakeAmount: 100})
	require.NoError(t, err)
	_, err = ms.ResolveDispute(f.ctx, &types.MsgResolveDispute{Creator: addrPayee, DisputeId: od.Id})
	require.NoError(t, err)
	require.Equal(t, uint64(500), balanceOf(t, f, addrPayee)) // 100 seed + 400 escrow (bond posted then returned)
}

// Control: an ORDINARY escrow (no_auto_release=false) still auto-releases to the
// payee past the deadline — the it7 anti-ghosting protection is unregressed.
func TestOrdinaryEscrowStillDeadlineReleases(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	seedAccount(t, f, addrPayer, 1000)
	seedAccount(t, f, addrPayee, 0)

	id := lockFresh(t, f, ms) // no_auto_release defaults false
	_, err := ms.SubmitEscrow(setBlockTime(f.ctx, 110), &types.MsgSubmitEscrow{Creator: addrPayee, Id: id})
	require.NoError(t, err)
	// Past the deadline the payee may self-release — allowed for ordinary escrows.
	_, err = ms.ReleaseEscrow(setBlockTime(f.ctx, 200), &types.MsgReleaseEscrow{Creator: addrPayee, Id: id})
	require.NoError(t, err)
	require.Equal(t, uint64(400), balanceOf(t, f, addrPayee))
}
