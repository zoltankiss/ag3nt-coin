package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zoltankiss/agntcoin/x/agntcoin/keeper"
	"github.com/zoltankiss/agntcoin/x/agntcoin/types"
)

// Fair-exchange state machine:
//   locked    -> refund (payer, pre-deadline) | release | submit (payee)
//   submitted -> release | dispute (payer)    | auto-release (payee, post-deadline)
//   disputed  -> release (payer only; frozen, no auto-release)

func lockFresh(t *testing.T, f *fixture, ms types.MsgServer) uint64 {
	t.Helper()
	ctx := setBlockTime(f.ctx, 100)
	resp, err := ms.LockEscrow(ctx, &types.MsgLockEscrow{Creator: addrPayer, Payee: addrPayee, Amount: 400, Ref: "r", DisputeSeconds: 50})
	require.NoError(t, err)
	return resp.Id
}

// Submit blocks the payer from refunding (the labor-theft fix).
func TestSubmitBlocksRefund(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	seedAccount(t, f, addrPayer, 1000)
	seedAccount(t, f, addrPayee, 0)
	id := lockFresh(t, f, ms)

	_, err := ms.SubmitEscrow(setBlockTime(f.ctx, 110), &types.MsgSubmitEscrow{Creator: addrPayee, Id: id})
	require.NoError(t, err)
	esc, _ := f.keeper.Escrow.Get(f.ctx, id)
	require.Equal(t, types.EscrowStatusSubmitted, esc.Status)

	// Payer can no longer refund, even inside the original window (t=120 < 150).
	_, err = ms.RefundEscrow(setBlockTime(f.ctx, 120), &types.MsgRefundEscrow{Creator: addrPayer, Id: id})
	require.Error(t, err)
	require.Equal(t, uint64(600), balanceOf(t, f, addrPayer)) // funds still held
}

// Submit still allows the payer to release (pay).
func TestSubmitThenReleasePays(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	seedAccount(t, f, addrPayer, 1000)
	seedAccount(t, f, addrPayee, 5)
	id := lockFresh(t, f, ms)

	_, err := ms.SubmitEscrow(setBlockTime(f.ctx, 110), &types.MsgSubmitEscrow{Creator: addrPayee, Id: id})
	require.NoError(t, err)
	_, err = ms.ReleaseEscrow(setBlockTime(f.ctx, 120), &types.MsgReleaseEscrow{Creator: addrPayer, Id: id})
	require.NoError(t, err)
	require.Equal(t, uint64(405), balanceOf(t, f, addrPayee))
}

// A submitted escrow auto-releases to the payee after the deadline (a ghosting
// payer cannot trap delivered work).
func TestSubmittedAutoReleaseAfterDeadline(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	seedAccount(t, f, addrPayer, 1000)
	seedAccount(t, f, addrPayee, 0)
	id := lockFresh(t, f, ms)

	_, err := ms.SubmitEscrow(setBlockTime(f.ctx, 110), &types.MsgSubmitEscrow{Creator: addrPayee, Id: id})
	require.NoError(t, err)
	// payee self-releases after deadline (t=200 >= 150).
	_, err = ms.ReleaseEscrow(setBlockTime(f.ctx, 200), &types.MsgReleaseEscrow{Creator: addrPayee, Id: id})
	require.NoError(t, err)
	require.Equal(t, uint64(400), balanceOf(t, f, addrPayee))
}

func TestSubmitOnlyByPayeeOnLocked(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	seedAccount(t, f, addrPayer, 1000)
	seedAccount(t, f, addrPayee, 0)
	id := lockFresh(t, f, ms)

	// payer cannot submit
	_, err := ms.SubmitEscrow(f.ctx, &types.MsgSubmitEscrow{Creator: addrPayer, Id: id})
	require.Error(t, err)
	// submit, then a second submit fails (not locked anymore)
	_, err = ms.SubmitEscrow(f.ctx, &types.MsgSubmitEscrow{Creator: addrPayee, Id: id})
	require.NoError(t, err)
	_, err = ms.SubmitEscrow(f.ctx, &types.MsgSubmitEscrow{Creator: addrPayee, Id: id})
	require.Error(t, err)
}

// Dispute freezes a submitted escrow: no refund, no payee auto-release; only the
// payer can release (concede).
func TestDisputeFreezes(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	seedAccount(t, f, addrPayer, 1000)
	seedAccount(t, f, addrPayee, 0)
	id := lockFresh(t, f, ms)

	_, err := ms.SubmitEscrow(setBlockTime(f.ctx, 110), &types.MsgSubmitEscrow{Creator: addrPayee, Id: id})
	require.NoError(t, err)
	_, err = ms.DisputeEscrow(setBlockTime(f.ctx, 120), &types.MsgDisputeEscrow{Creator: addrPayer, Id: id})
	require.NoError(t, err)
	esc, _ := f.keeper.Escrow.Get(f.ctx, id)
	require.Equal(t, types.EscrowStatusDisputed, esc.Status)

	// payer cannot refund a disputed escrow
	_, err = ms.RefundEscrow(setBlockTime(f.ctx, 130), &types.MsgRefundEscrow{Creator: addrPayer, Id: id})
	require.Error(t, err)
	// payee cannot auto-release after the deadline (frozen)
	_, err = ms.ReleaseEscrow(setBlockTime(f.ctx, 200), &types.MsgReleaseEscrow{Creator: addrPayee, Id: id})
	require.Error(t, err)
	require.Equal(t, uint64(0), balanceOf(t, f, addrPayee))
	// payer may still concede-release
	_, err = ms.ReleaseEscrow(setBlockTime(f.ctx, 210), &types.MsgReleaseEscrow{Creator: addrPayer, Id: id})
	require.NoError(t, err)
	require.Equal(t, uint64(400), balanceOf(t, f, addrPayee))
}

// Lock auto-registers a brand-new payee so it can submit on its FIRST job
// (closes "first job unprotectable"). addrNewee starts non-existent.
func TestLockAutoRegistersPayeeForFirstJob(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	seedAccount(t, f, addrPayer, 1000)

	_, err := f.keeper.Account.Get(f.ctx, addrNewee) // does not exist yet
	require.Error(t, err)

	ctx := setBlockTime(f.ctx, 100)
	resp, err := ms.LockEscrow(ctx, &types.MsgLockEscrow{Creator: addrPayer, Payee: addrNewee, Amount: 400, Ref: "first", DisputeSeconds: 50})
	require.NoError(t, err)

	// payee now registered (zero balance — funds are in escrow, not credited)
	acc, err := f.keeper.Account.Get(f.ctx, addrNewee)
	require.NoError(t, err)
	require.True(t, acc.Registered)
	require.Equal(t, uint64(0), acc.Balance)

	// and the brand-new payee can SUBMIT — its first job is now protectable
	_, err = ms.SubmitEscrow(ctx, &types.MsgSubmitEscrow{Creator: addrNewee, Id: resp.Id})
	require.NoError(t, err)
	esc, _ := f.keeper.Escrow.Get(f.ctx, resp.Id)
	require.Equal(t, types.EscrowStatusSubmitted, esc.Status)
}

// Sub-minimum escrow to a brand-new payee is rejected (spam guard); payer not debited.
func TestLockToNewPayeeBelowMinRejected(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	seedAccount(t, f, addrPayer, 1000)
	require.Greater(t, types.MinNewAccountCredit, uint64(1))

	ctx := setBlockTime(f.ctx, 100)
	_, err := ms.LockEscrow(ctx, &types.MsgLockEscrow{Creator: addrPayer, Payee: addrNewee, Amount: types.MinNewAccountCredit - 1, Ref: "x", DisputeSeconds: 50})
	require.Error(t, err)
	require.Equal(t, uint64(1000), balanceOf(t, f, addrPayer)) // not debited
	_, err = f.keeper.Account.Get(f.ctx, addrNewee)
	require.Error(t, err) // payee not created
}

func TestDisputeOnlyOnSubmittedByPayer(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	seedAccount(t, f, addrPayer, 1000)
	seedAccount(t, f, addrPayee, 0)
	id := lockFresh(t, f, ms)

	// cannot dispute a locked (not yet submitted) escrow
	_, err := ms.DisputeEscrow(f.ctx, &types.MsgDisputeEscrow{Creator: addrPayer, Id: id})
	require.Error(t, err)

	_, err = ms.SubmitEscrow(f.ctx, &types.MsgSubmitEscrow{Creator: addrPayee, Id: id})
	require.NoError(t, err)
	// only the payer may dispute
	_, err = ms.DisputeEscrow(f.ctx, &types.MsgDisputeEscrow{Creator: addrPayee, Id: id})
	require.Error(t, err)
}
