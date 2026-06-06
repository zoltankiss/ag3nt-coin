package keeper_test

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zoltankiss/agntcoin/testutil/sample"
	"github.com/zoltankiss/agntcoin/x/agntcoin/keeper"
	"github.com/zoltankiss/agntcoin/x/agntcoin/types"
)

// The dispute-bond (v0.1.0) makes OpenDispute cost the opener: a frivolous/lost
// verdict slashes the bond to the griefed counterparty; an upheld one returns it.
// This is the it19 GREEN of the it18 cry-wolf + jury-DDoS RED.

// OpenDispute requires a bond of at least MinDisputeBond, debits it, and pins a
// verifiable escrow ref on it (it17 #22) — a free dispute is no longer possible.
func TestOpenDisputeRequiresBond(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	juror := sample.AccAddress()
	setJurors(t, f, juror)
	seedAccount(t, f, juror, 1000) // jury-v1: fund the juror-stake
	seedAccount(t, f, addrPayer, 1000)
	seedAccount(t, f, addrPayee, 0)
	eid := submittedEscrow(t, f, ms)

	// No bond → rejected. Sub-minimum bond → rejected. (The free-dispute hole closed.)
	_, err := ms.OpenDispute(f.ctx, &types.MsgOpenDispute{Creator: addrPayer, EscrowId: eid, Reason: "x"})
	require.Error(t, err)
	_, err = ms.OpenDispute(f.ctx, &types.MsgOpenDispute{Creator: addrPayer, EscrowId: eid, Reason: "x", BondAmount: types.MinDisputeBond - 1})
	require.Error(t, err)

	// A sufficient bond is debited and the dispute records it; the bond pins the escrow.
	before := balanceOf(t, f, addrPayer)
	od, err := ms.OpenDispute(f.ctx, &types.MsgOpenDispute{Creator: addrPayer, EscrowId: eid, Reason: "rule please", BondAmount: 150})
	require.NoError(t, err)
	require.Equal(t, before-150, balanceOf(t, f, addrPayer))

	d, err := f.keeper.Dispute.Get(f.ctx, od.Id)
	require.NoError(t, err)
	b, err := f.keeper.Bond.Get(f.ctx, d.BondId)
	require.NoError(t, err)
	require.Equal(t, types.BondStatusActive, b.Status)
	require.Equal(t, addrPayer, b.Poster)
	require.Equal(t, uint64(150), b.Amount)
	require.Equal(t, "dispute:"+strconv.FormatUint(od.Id, 10), b.Purpose)
	require.Equal(t, strconv.FormatUint(eid, 10), b.Ref) // verifiable escrow ref
	require.Equal(t, "", b.Slasher)                      // protocol-settled only
}

// CRY-WOLF: a buyer disputes good work; the jury accepts (work was fine); the
// buyer's dispute is NOT upheld → its bond is slashed to the griefed worker.
func TestDisputeBondSlashedToGriefedWorkerOnFrivolous(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	juror := sample.AccAddress()
	setJurors(t, f, juror)
	seedAccount(t, f, juror, 1000) // jury-v1: fund the juror-stake
	seedAccount(t, f, addrPayer, 1000)
	seedAccount(t, f, addrPayee, 0)
	eid := submittedEscrow(t, f, ms) // locks 400 payer→payee, submitted

	od, err := ms.OpenDispute(f.ctx, &types.MsgOpenDispute{Creator: addrPayer, EscrowId: eid, Reason: "cry wolf", BondAmount: 200})
	require.NoError(t, err)
	_, err = ms.CastVote(f.ctx, &types.MsgCastVote{Creator: juror, DisputeId: od.Id, Accept: true, StakeAmount: 100}) // work was good
	require.NoError(t, err)
	_, err = ms.ResolveDispute(f.ctx, &types.MsgResolveDispute{Creator: juror, DisputeId: od.Id})
	require.NoError(t, err)

	// Worker is made whole: 400 escrow + 200 slashed cry-wolf bond = 600.
	require.Equal(t, uint64(600), balanceOf(t, f, addrPayee))
	// The cry-wolf buyer ate the loss: 1000 - 400 locked - 200 bond (slashed, NOT returned) = 400.
	require.Equal(t, uint64(400), balanceOf(t, f, addrPayer))

	d, _ := f.keeper.Dispute.Get(f.ctx, od.Id)
	b, _ := f.keeper.Bond.Get(f.ctx, d.BondId)
	require.Equal(t, types.BondStatusSlashed, b.Status)
}

// UPHELD: a buyer disputes genuine slop; the jury rejects (refund the buyer);
// the buyer's dispute IS upheld → its bond returns. Honest disputing is free.
func TestDisputeBondReturnedWhenUpheld(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	juror := sample.AccAddress()
	setJurors(t, f, juror)
	seedAccount(t, f, juror, 1000) // jury-v1: fund the juror-stake
	seedAccount(t, f, addrPayer, 1000)
	seedAccount(t, f, addrPayee, 0)
	eid := submittedEscrow(t, f, ms)

	od, err := ms.OpenDispute(f.ctx, &types.MsgOpenDispute{Creator: addrPayer, EscrowId: eid, Reason: "genuine slop", BondAmount: 200})
	require.NoError(t, err)
	_, err = ms.CastVote(f.ctx, &types.MsgCastVote{Creator: juror, DisputeId: od.Id, Accept: false, StakeAmount: 100}) // slop → reject
	require.NoError(t, err)
	_, err = ms.ResolveDispute(f.ctx, &types.MsgResolveDispute{Creator: juror, DisputeId: od.Id})
	require.NoError(t, err)

	// Buyer's dispute upheld: 1000 - 400 locked + 400 refunded + 200 bond returned = 1200... no:
	// 1000 start, -400 lock, -200 bond, +400 refund, +200 bond return = 1000.
	require.Equal(t, uint64(1000), balanceOf(t, f, addrPayer))
	require.Equal(t, uint64(0), balanceOf(t, f, addrPayee))

	d, _ := f.keeper.Dispute.Get(f.ctx, od.Id)
	b, _ := f.keeper.Bond.Get(f.ctx, d.BondId)
	require.Equal(t, types.BondStatusReleased, b.Status)
}

// The protocol-held dispute-bond cannot be settled by an external SlashBond/
// ReleaseBond (its slasher is empty) — only ResolveDispute settles it.
func TestDisputeBondNotExternallySettleable(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	juror := sample.AccAddress()
	setJurors(t, f, juror)
	seedAccount(t, f, juror, 1000) // jury-v1: fund the juror-stake
	seedAccount(t, f, addrPayer, 1000)
	seedAccount(t, f, addrPayee, 0)
	seedAccount(t, f, addrOutside, 1000)
	eid := submittedEscrow(t, f, ms)

	od, err := ms.OpenDispute(f.ctx, &types.MsgOpenDispute{Creator: addrPayer, EscrowId: eid, Reason: "x", BondAmount: 200})
	require.NoError(t, err)
	d, _ := f.keeper.Dispute.Get(f.ctx, od.Id)

	// Nobody — not the opener, not an outsider — can release or slash it directly.
	_, err = ms.ReleaseBond(f.ctx, &types.MsgReleaseBond{Creator: addrPayer, Id: d.BondId})
	require.Error(t, err)
	_, err = ms.SlashBond(f.ctx, &types.MsgSlashBond{Creator: addrOutside, Id: d.BondId, Beneficiary: addrOutside})
	require.Error(t, err)
	b, _ := f.keeper.Bond.Get(f.ctx, d.BondId)
	require.Equal(t, types.BondStatusActive, b.Status) // still active; only ResolveDispute settles it
}
