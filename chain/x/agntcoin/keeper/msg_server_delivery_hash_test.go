package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zoltankiss/agntcoin/x/agntcoin/keeper"
	"github.com/zoltankiss/agntcoin/x/agntcoin/types"
)

// The payee pins the delivered artifact's hash on-chain at submit, so the exhibit
// the jury later judges is tamper-evident — a serving layer can't silently alter
// the delivery without the re-hash mismatching this commitment. The commitment is
// set once at submit and persists unchanged through to settlement.
func TestSubmitPinsDeliveryHash(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	seedAccount(t, f, addrPayer, 1000)
	seedAccount(t, f, addrPayee, 0)

	id := lockFresh(t, f, ms)
	const h = "9f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08" // hex sha256
	_, err := ms.SubmitEscrow(setBlockTime(f.ctx, 110), &types.MsgSubmitEscrow{Creator: addrPayee, Id: id, DeliveryHash: h})
	require.NoError(t, err)

	esc, err := f.keeper.Escrow.Get(f.ctx, id)
	require.NoError(t, err)
	require.Equal(t, h, esc.DeliveryHash)

	// The commitment survives release unchanged (auditable after settlement).
	_, err = ms.ReleaseEscrow(setBlockTime(f.ctx, 120), &types.MsgReleaseEscrow{Creator: addrPayer, Id: id})
	require.NoError(t, err)
	esc, _ = f.keeper.Escrow.Get(f.ctx, id)
	require.Equal(t, h, esc.DeliveryHash)
	require.Equal(t, types.EscrowStatusReleased, esc.Status)
}

// An ordinary submit with no hash leaves the commitment empty (backward-compatible).
func TestSubmitWithoutHashIsEmpty(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	seedAccount(t, f, addrPayer, 1000)
	seedAccount(t, f, addrPayee, 0)

	id := lockFresh(t, f, ms)
	_, err := ms.SubmitEscrow(setBlockTime(f.ctx, 110), &types.MsgSubmitEscrow{Creator: addrPayee, Id: id})
	require.NoError(t, err)
	esc, _ := f.keeper.Escrow.Get(f.ctx, id)
	require.Equal(t, "", esc.DeliveryHash)
}
