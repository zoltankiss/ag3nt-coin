package keeper_test

import (
	"context"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/zoltankiss/agntcoin/x/agntcoin/keeper"
	"github.com/zoltankiss/agntcoin/x/agntcoin/types"
)

// fixed valid bech32 addresses for the escrow tests (derived from constant
// 20-byte arrays so they pass address-codec validation).
const (
	addrPayer   = "cosmos1qyqszqgpqyqszqgpqyqszqgpqyqszqgpjnp7du"
	addrPayee   = "cosmos1qgpqyqszqgpqyqszqgpqyqszqgpqyqszrh8mx2"
	addrNewee   = "cosmos1qvpsxqcrqvpsxqcrqvpsxqcrqvpsxqcrz8x6vt"
	addrOutside = "cosmos1qszqgpqyqszqgpqyqszqgpqyqszqgpqyzhplth"
)

// setBlockTime returns a context whose block time is `t`.
func setBlockTime(ctx context.Context, sec int64) context.Context {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	return sdkCtx.WithBlockTime(time.Unix(sec, 0))
}

// seedAccount creates a registered account with the given balance.
func seedAccount(t *testing.T, f *fixture, addr string, balance uint64) {
	t.Helper()
	err := f.keeper.Account.Set(f.ctx, addr, types.Account{
		Address:    addr,
		Balance:    balance,
		Registered: true,
	})
	require.NoError(t, err)
}

func balanceOf(t *testing.T, f *fixture, addr string) uint64 {
	t.Helper()
	acc, err := f.keeper.Account.Get(f.ctx, addr)
	require.NoError(t, err)
	return acc.Balance
}

func TestLockEscrowDebitsPayer(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	seedAccount(t, f, addrPayer, 1000)

	ctx := setBlockTime(f.ctx, 100)
	resp, err := ms.LockEscrow(ctx, &types.MsgLockEscrow{
		Creator:        addrPayer,
		Payee:          addrPayee,
		Amount:         400,
		Ref:            "job-1",
		DisputeSeconds: 50,
	})
	require.NoError(t, err)
	require.Equal(t, uint64(0), resp.Id)

	// Payer debited immediately.
	require.Equal(t, uint64(600), balanceOf(t, f, addrPayer))

	esc, err := f.keeper.Escrow.Get(f.ctx, resp.Id)
	require.NoError(t, err)
	require.Equal(t, types.EscrowStatusLocked, esc.Status)
	require.Equal(t, uint64(400), esc.Amount)
	require.Equal(t, "job-1", esc.Ref)
	require.Equal(t, int64(150), esc.Deadline)
}

func TestLockEscrowRejectsBadInput(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	seedAccount(t, f, addrPayer, 100)
	ctx := setBlockTime(f.ctx, 100)

	// zero amount
	_, err := ms.LockEscrow(ctx, &types.MsgLockEscrow{Creator: addrPayer, Payee: addrPayee, Amount: 0, DisputeSeconds: 10})
	require.Error(t, err)

	// payee == payer
	_, err = ms.LockEscrow(ctx, &types.MsgLockEscrow{Creator: addrPayer, Payee: addrPayer, Amount: 10, DisputeSeconds: 10})
	require.Error(t, err)

	// insufficient balance
	_, err = ms.LockEscrow(ctx, &types.MsgLockEscrow{Creator: addrPayer, Payee: addrPayee, Amount: 1000, DisputeSeconds: 10})
	require.Error(t, err)
}

func TestReleaseEscrowPaysExistingPayee(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	seedAccount(t, f, addrPayer, 1000)
	seedAccount(t, f, addrPayee, 5)

	ctx := setBlockTime(f.ctx, 100)
	resp, err := ms.LockEscrow(ctx, &types.MsgLockEscrow{Creator: addrPayer, Payee: addrPayee, Amount: 400, Ref: "r", DisputeSeconds: 50})
	require.NoError(t, err)

	// payer releases before deadline.
	_, err = ms.ReleaseEscrow(ctx, &types.MsgReleaseEscrow{Creator: addrPayer, Id: resp.Id})
	require.NoError(t, err)

	require.Equal(t, uint64(405), balanceOf(t, f, addrPayee))
	esc, _ := f.keeper.Escrow.Get(f.ctx, resp.Id)
	require.Equal(t, types.EscrowStatusReleased, esc.Status)
}

func TestReleaseEscrowAutoRegistersNewPayee(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	seedAccount(t, f, addrPayer, 1000)

	ctx := setBlockTime(f.ctx, 100)
	resp, err := ms.LockEscrow(ctx, &types.MsgLockEscrow{Creator: addrPayer, Payee: addrNewee, Amount: 400, Ref: "r", DisputeSeconds: 50})
	require.NoError(t, err)

	// payee is now auto-registered at LOCK time (zero balance — funds in escrow).
	acc0, err := f.keeper.Account.Get(f.ctx, addrNewee)
	require.NoError(t, err)
	require.True(t, acc0.Registered)
	require.Equal(t, uint64(0), acc0.Balance)

	_, err = ms.ReleaseEscrow(ctx, &types.MsgReleaseEscrow{Creator: addrPayer, Id: resp.Id})
	require.NoError(t, err)

	// release credits the (already-registered) payee.
	acc, err := f.keeper.Account.Get(f.ctx, addrNewee)
	require.NoError(t, err)
	require.True(t, acc.Registered)
	require.Equal(t, uint64(400), acc.Balance)
}

func TestRefundEscrowReturnsToPayerInWindow(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	seedAccount(t, f, addrPayer, 1000)

	ctx := setBlockTime(f.ctx, 100)
	resp, err := ms.LockEscrow(ctx, &types.MsgLockEscrow{Creator: addrPayer, Payee: addrPayee, Amount: 400, Ref: "r", DisputeSeconds: 50})
	require.NoError(t, err)
	require.Equal(t, uint64(600), balanceOf(t, f, addrPayer))

	// refund before deadline (t=120 < 150).
	ctx2 := setBlockTime(f.ctx, 120)
	_, err = ms.RefundEscrow(ctx2, &types.MsgRefundEscrow{Creator: addrPayer, Id: resp.Id})
	require.NoError(t, err)

	require.Equal(t, uint64(1000), balanceOf(t, f, addrPayer))
	esc, _ := f.keeper.Escrow.Get(f.ctx, resp.Id)
	require.Equal(t, types.EscrowStatusRefunded, esc.Status)
}

func TestReleaseAfterDeadlineByNonPayer(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	seedAccount(t, f, addrPayer, 1000)
	seedAccount(t, f, addrPayee, 0)

	ctx := setBlockTime(f.ctx, 100)
	resp, err := ms.LockEscrow(ctx, &types.MsgLockEscrow{Creator: addrPayer, Payee: addrPayee, Amount: 400, Ref: "r", DisputeSeconds: 50})
	require.NoError(t, err)

	// Before deadline a non-payer cannot release.
	ctxEarly := setBlockTime(f.ctx, 120)
	_, err = ms.ReleaseEscrow(ctxEarly, &types.MsgReleaseEscrow{Creator: addrPayee, Id: resp.Id})
	require.Error(t, err)

	// After deadline (t=200 >= 150) a non-payer (e.g. the payee) can release.
	ctxLate := setBlockTime(f.ctx, 200)
	_, err = ms.ReleaseEscrow(ctxLate, &types.MsgReleaseEscrow{Creator: addrPayee, Id: resp.Id})
	require.NoError(t, err)
	require.Equal(t, uint64(400), balanceOf(t, f, addrPayee))
}

func TestRefundAfterDeadlineRejected(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	seedAccount(t, f, addrPayer, 1000)

	ctx := setBlockTime(f.ctx, 100)
	resp, err := ms.LockEscrow(ctx, &types.MsgLockEscrow{Creator: addrPayer, Payee: addrPayee, Amount: 400, Ref: "r", DisputeSeconds: 50})
	require.NoError(t, err)

	// refund after deadline (t=200 >= 150) is rejected.
	ctxLate := setBlockTime(f.ctx, 200)
	_, err = ms.RefundEscrow(ctxLate, &types.MsgRefundEscrow{Creator: addrPayer, Id: resp.Id})
	require.Error(t, err)

	// funds still held (payer not credited back).
	require.Equal(t, uint64(600), balanceOf(t, f, addrPayer))
}

func TestRefundByNonPayerRejected(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	seedAccount(t, f, addrPayer, 1000)

	ctx := setBlockTime(f.ctx, 100)
	resp, err := ms.LockEscrow(ctx, &types.MsgLockEscrow{Creator: addrPayer, Payee: addrPayee, Amount: 400, Ref: "r", DisputeSeconds: 50})
	require.NoError(t, err)

	ctx2 := setBlockTime(f.ctx, 120)
	_, err = ms.RefundEscrow(ctx2, &types.MsgRefundEscrow{Creator: addrOutside, Id: resp.Id})
	require.Error(t, err)
}

func TestDoubleReleaseRejected(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	seedAccount(t, f, addrPayer, 1000)
	seedAccount(t, f, addrPayee, 0)

	ctx := setBlockTime(f.ctx, 100)
	resp, err := ms.LockEscrow(ctx, &types.MsgLockEscrow{Creator: addrPayer, Payee: addrPayee, Amount: 400, Ref: "r", DisputeSeconds: 50})
	require.NoError(t, err)

	_, err = ms.ReleaseEscrow(ctx, &types.MsgReleaseEscrow{Creator: addrPayer, Id: resp.Id})
	require.NoError(t, err)

	// second release must fail (not locked anymore).
	_, err = ms.ReleaseEscrow(ctx, &types.MsgReleaseEscrow{Creator: addrPayer, Id: resp.Id})
	require.Error(t, err)
}

func TestTransferSpamGuardRejectsDustToNewAccount(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	seedAccount(t, f, addrPayer, 1000)

	require.Greater(t, types.MinNewAccountCredit, uint64(1), "guard test requires min > 1")

	// Dust (positive but below the minimum) to a brand-new address is rejected.
	_, err := ms.Transfer(f.ctx, &types.MsgTransfer{
		Creator: addrPayer,
		ToAddr:  addrNewee,
		Amount:  types.MinNewAccountCredit - 1,
	})
	require.Error(t, err)
	// Payer must NOT have been debited.
	require.Equal(t, uint64(1000), balanceOf(t, f, addrPayer))
	// New account must NOT have been created.
	_, err = f.keeper.Account.Get(f.ctx, addrNewee)
	require.Error(t, err)
}

func TestTransferAtMinimumToNewAccountSucceeds(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	seedAccount(t, f, addrPayer, 1000)

	_, err := ms.Transfer(f.ctx, &types.MsgTransfer{
		Creator: addrPayer,
		ToAddr:  addrNewee,
		Amount:  types.MinNewAccountCredit,
	})
	require.NoError(t, err)
	acc, err := f.keeper.Account.Get(f.ctx, addrNewee)
	require.NoError(t, err)
	require.True(t, acc.Registered)
	require.Equal(t, types.MinNewAccountCredit, acc.Balance)
}

func TestTransferDustToExistingAccountAllowed(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	seedAccount(t, f, addrPayer, 1000)
	seedAccount(t, f, addrPayee, 0) // already exists

	// Dust to an EXISTING account is fine (guard only applies to new accounts).
	_, err := ms.Transfer(f.ctx, &types.MsgTransfer{
		Creator: addrPayer,
		ToAddr:  addrPayee,
		Amount:  1,
	})
	require.NoError(t, err)
	require.Equal(t, uint64(1), balanceOf(t, f, addrPayee))
}
