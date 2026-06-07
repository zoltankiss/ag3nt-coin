package keeper_test

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zoltankiss/agntcoin/testutil/sample"
	"github.com/zoltankiss/agntcoin/x/agntcoin/keeper"
	"github.com/zoltankiss/agntcoin/x/agntcoin/types"
)

// verifier-v1 (0.3.0): bonded optimistic attestation + atomic key reveal.
// These tests script the adversarial scenarios the design was argued through:
//
//	S1  happy path with a NON-COOPERATIVE buyer (the iteration's core question)
//	S2  fail attestation gates refund behind the payee's contest window
//	S3  lying verifier (false pass) slashed via post-release fraud challenge
//	S4  ghosting verifier — the jury rescues the seller
//	S5  frivolous post-release challenge — challenger's bond goes to the verifier
//	S7  false-fail collusion — contest slashes the verifier's stake to the payee
//	H1  the verification method is opaque to the chain (LLM-judge acceptance spec)
//
// (S6 — the flaky/non-hermetic acceptance procedure — is deliberately absent:
// it is excluded at deal-formation time by pinning the judge/environment in
// acceptance_hash, not litigated on-chain. There is nothing chain-side to test.)

func hexHash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

const (
	artifactKey        = "decryption-key-K-7f3a"
	artifactCiphertext = "ciphertext-of-the-deliverable-v1"
)

// Real hex-sha256 commitments: the chain rejects anything a juror could never
// re-check by hashing actual bytes (a malformed "hash" only looks like evidence).
var (
	acceptHash   = hexHash("pytest-suite+hermetic-env-spec-v1") // precommitment; CONTENT opaque to the chain
	deliveryHash = hexHash(artifactCiphertext)
)

// attestMsg builds a verdict that RESTATES the standard test deal's
// commitments — delivery_hash, key_hash, acceptance_hash — as the chain
// requires (an attestation that names different bytes or a different
// procedure than the escrow record is rejected).
func attestMsg(creator string, id uint64, passed bool, stake uint64) *types.MsgAttestEscrow {
	return &types.MsgAttestEscrow{
		Creator: creator, EscrowId: id, Passed: passed, StakeAmount: stake,
		DeliveryHash: deliveryHash, KeyHash: hexHash(artifactKey), AcceptanceHash: acceptHash,
	}
}

// lockVerified locks a 400 payer→payee escrow bound to the given verifier set.
func lockVerified(t *testing.T, f *fixture, ms types.MsgServer, quorum uint64, verifiers ...string) uint64 {
	t.Helper()
	resp, err := ms.LockEscrow(setBlockTime(f.ctx, 100), &types.MsgLockEscrow{
		Creator:        addrPayer,
		Payee:          addrPayee,
		Amount:         400,
		Ref:            "deal-1",
		DisputeSeconds: 50,
		VerifierAddrs:  verifiers,
		VerifierQuorum: quorum,
		AcceptanceHash: acceptHash,
	})
	require.NoError(t, err)
	return resp.Id
}

// submitEncrypted: the payee commits the ciphertext hash AND the key hash —
// the fair-exchange commitments VerifiedRelease later checks.
func submitEncrypted(t *testing.T, f *fixture, ms types.MsgServer, id uint64) {
	t.Helper()
	_, err := ms.SubmitEscrow(setBlockTime(f.ctx, 110), &types.MsgSubmitEscrow{
		Creator:      addrPayee,
		Id:           id,
		DeliveryHash: deliveryHash,
		KeyHash:      hexHash(artifactKey),
	})
	require.NoError(t, err)
}

// S1 — Happy path, buyer locks and then goes silent forever. The seller is
// paid without the buyer's goodwill, the key is revealed atomically with the
// payment, and the verifier's stake comes back once the window passes.
func TestVerifiedReleaseHappyPathNonCooperativeBuyer(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	verifier := sample.AccAddress()
	seedAccount(t, f, addrPayer, 1000)
	seedAccount(t, f, addrPayee, 0)
	seedAccount(t, f, verifier, 1000)

	id := lockVerified(t, f, ms, 1, verifier)

	// The clock-bypass kill: a verifier-bound escrow is ALWAYS no_auto_release,
	// even though the payer didn't ask for it.
	esc, _ := f.keeper.Escrow.Get(f.ctx, id)
	require.True(t, esc.NoAutoRelease, "verifier-bound escrow must force no_auto_release")
	require.Equal(t, acceptHash, esc.AcceptanceHash)

	// Buyer is now silent forever. Seller delivers (ciphertext + key commitment).
	submitEncrypted(t, f, ms, id)

	// Verifier runs the precommitted acceptance procedure off-chain, attests
	// pass with a stake covering the bounty. Stake debited + locked.
	ar, err := ms.AttestEscrow(setBlockTime(f.ctx, 115), attestMsg(verifier, id, true, 400))
	require.NoError(t, err)
	require.Equal(t, uint64(600), balanceOf(t, f, verifier))

	// Anyone (here: the seller) triggers settlement with the key preimage.
	// Payment + key reveal are one tx; the challenge window opens.
	_, err = ms.VerifiedRelease(setBlockTime(f.ctx, 120), &types.MsgVerifiedRelease{
		Creator: addrPayee, EscrowId: id, Key: artifactKey,
	})
	require.NoError(t, err)
	require.Equal(t, uint64(400), balanceOf(t, f, addrPayee), "seller paid without any buyer action after funding")

	esc, _ = f.keeper.Escrow.Get(f.ctx, id)
	require.Equal(t, types.EscrowStatusReleased, esc.Status)
	require.Equal(t, int64(120+types.ChallengeWindowSeconds), esc.ChallengeDeadline)

	// The stake stays locked while the buyer could still challenge…
	_, err = ms.ReclaimAttestationBonds(setBlockTime(f.ctx, 130), &types.MsgReclaimAttestationBonds{Creator: verifier, EscrowId: id})
	require.Error(t, err, "stake must stay locked inside the challenge window")

	// …and is freed once the window passes uncontested.
	_, err = ms.ReclaimAttestationBonds(setBlockTime(f.ctx, esc.ChallengeDeadline+1), &types.MsgReclaimAttestationBonds{Creator: verifier, EscrowId: id})
	require.NoError(t, err)
	require.Equal(t, uint64(1000), balanceOf(t, f, verifier), "honest verifier made whole")
	b, _ := f.keeper.Bond.Get(f.ctx, ar.BondId)
	require.Equal(t, types.BondStatusReleased, b.Status)
}

// S2 — Fail attestation: refund is JUSTIFIED but not INSTANT. The payee's
// contest window must pass uncontested first (the false-fail+instant-refund
// hole), then the payer recovers the funds and the verifier its stake.
func TestFailAttestationGatesRefundBehindContestWindow(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	verifier := sample.AccAddress()
	seedAccount(t, f, addrPayer, 1000)
	seedAccount(t, f, addrPayee, 0)
	seedAccount(t, f, verifier, 1000)

	id := lockVerified(t, f, ms, 1, verifier)
	submitEncrypted(t, f, ms, id)

	// 2/3 tests fail → staked fail verdict. Escrow enters the contest window.
	_, err := ms.AttestEscrow(setBlockTime(f.ctx, 115), attestMsg(verifier, id, false, 400))
	require.NoError(t, err)
	esc, _ := f.keeper.Escrow.Get(f.ctx, id)
	require.Equal(t, types.EscrowStatusFailAttested, esc.Status)
	deadline := esc.ChallengeDeadline
	require.Equal(t, int64(115+types.ChallengeWindowSeconds), deadline)

	// No instant clawback: refund is blocked while the payee could contest.
	_, err = ms.RefundEscrow(setBlockTime(f.ctx, 120), &types.MsgRefundEscrow{Creator: addrPayer, Id: id})
	require.Error(t, err, "refund must wait out the payee's contest window")

	// A pass-quorum release is also impossible — the fail froze the pass path.
	_, err = ms.VerifiedRelease(setBlockTime(f.ctx, 120), &types.MsgVerifiedRelease{Creator: addrPayee, EscrowId: id, Key: artifactKey})
	require.Error(t, err)

	// Window passes uncontested → the fail stands by default → refund unlocks
	// (note: well past the escrow's ORIGINAL deadline — the staked fail verdict
	// is the justification, the contest window the timing guard).
	_, err = ms.RefundEscrow(setBlockTime(f.ctx, deadline+1), &types.MsgRefundEscrow{Creator: addrPayer, Id: id})
	require.NoError(t, err)
	require.Equal(t, uint64(1000), balanceOf(t, f, addrPayer))

	// The uncontested fail-attester reclaims its stake.
	_, err = ms.ReclaimAttestationBonds(setBlockTime(f.ctx, deadline+1), &types.MsgReclaimAttestationBonds{Creator: verifier, EscrowId: id})
	require.NoError(t, err)
	require.Equal(t, uint64(1000), balanceOf(t, f, verifier))
}

// S3 — The scenario that justifies the whole design: verifier colludes with
// the seller and signs pass over garbage. Release fires (optimistic), the
// buyer challenges inside the window, the jury convicts — and the buyer is
// made whole FROM THE VERIFIER'S SLASHED STAKE while the seller keeps the
// payout (that is exactly what the stake insured).
func TestLyingVerifierSlashedToBuyerViaChallenge(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	verifier := sample.AccAddress()
	juror := sample.AccAddress()
	setJurors(t, f, juror)
	seedAccount(t, f, addrPayer, 1000)
	seedAccount(t, f, addrPayee, 0)
	seedAccount(t, f, verifier, 1000)
	seedAccount(t, f, juror, 1000)

	id := lockVerified(t, f, ms, 1, verifier)
	submitEncrypted(t, f, ms, id)
	_, err := ms.AttestEscrow(setBlockTime(f.ctx, 115), attestMsg(verifier, id, true, 400)) // the lie
	require.NoError(t, err)
	_, err = ms.VerifiedRelease(setBlockTime(f.ctx, 120), &types.MsgVerifiedRelease{Creator: addrPayee, EscrowId: id, Key: artifactKey})
	require.NoError(t, err)

	// Buyer decrypts, re-runs the pinned acceptance procedure: fail,
	// reproducibly. Opens the bonded fraud challenge inside the window.
	od, err := ms.OpenDispute(setBlockTime(f.ctx, 130), &types.MsgOpenDispute{
		Creator: addrPayer, EscrowId: id, Reason: "attested pass; tests fail reproducibly", BondAmount: 100,
	})
	require.NoError(t, err)
	esc, _ := f.keeper.Escrow.Get(f.ctx, id)
	require.Equal(t, types.EscrowStatusChallenged, esc.Status)

	// The attester's stake is now jury-bound — no quiet reclaim.
	_, err = ms.ReclaimAttestationBonds(setBlockTime(f.ctx, esc.ChallengeDeadline+1), &types.MsgReclaimAttestationBonds{Creator: verifier, EscrowId: id})
	require.Error(t, err)

	// Juror re-runs the pinned procedure (mechanical) → fraud. Reject.
	_, err = ms.CastVote(f.ctx, &types.MsgCastVote{Creator: juror, DisputeId: od.Id, Accept: false, StakeAmount: 100})
	require.NoError(t, err)
	rd, err := ms.ResolveDispute(f.ctx, &types.MsgResolveDispute{Creator: addrPayer, DisputeId: od.Id})
	require.NoError(t, err)
	require.Equal(t, types.DisputeResolutionReject, rd.Resolution)

	// Final tally — buyer −0, seller +400 (kept; insured), verifier −400.
	require.Equal(t, uint64(1000), balanceOf(t, f, addrPayer), "buyer made whole from the verifier's stake + bond returned")
	require.Equal(t, uint64(400), balanceOf(t, f, addrPayee), "payout is never clawed back from the payee")
	require.Equal(t, uint64(600), balanceOf(t, f, verifier), "one provable lie costs the verifier its stake")
	require.Equal(t, uint64(1000), balanceOf(t, f, juror), "coherent juror made whole")

	esc, _ = f.keeper.Escrow.Get(f.ctx, id)
	require.Equal(t, types.EscrowStatusReleased, esc.Status, "escrow stays settled; only the bonds moved")
}

// S4 — Verifier ghosts: no attestation ever lands. The clock path is dead
// (no_auto_release forced), so the seller's recourse is the bonded jury,
// which accepts honest work and pays out.
func TestGhostingVerifierJuryRescuesSeller(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	verifier := sample.AccAddress()
	juror := sample.AccAddress()
	setJurors(t, f, juror)
	seedAccount(t, f, addrPayer, 1000)
	seedAccount(t, f, addrPayee, 100) // funds the dispute-bond
	seedAccount(t, f, juror, 1000)

	id := lockVerified(t, f, ms, 1, verifier)
	submitEncrypted(t, f, ms, id)

	// Silence. The seller cannot wait out the clock (that hole is closed)…
	_, err := ms.ReleaseEscrow(setBlockTime(f.ctx, 200), &types.MsgReleaseEscrow{Creator: addrPayee, Id: id})
	require.Error(t, err, "verifier-bound escrow must never release by the clock")
	// …and cannot release on a quorum that never formed.
	_, err = ms.VerifiedRelease(setBlockTime(f.ctx, 200), &types.MsgVerifiedRelease{Creator: addrPayee, EscrowId: id, Key: artifactKey})
	require.Error(t, err)

	// Recourse: the bonded jury. Honest work → accept → paid, bond returned.
	od, err := ms.OpenDispute(setBlockTime(f.ctx, 200), &types.MsgOpenDispute{
		Creator: addrPayee, EscrowId: id, Reason: "verifier ghosted; key attached as evidence", BondAmount: 100,
	})
	require.NoError(t, err)
	_, err = ms.CastVote(f.ctx, &types.MsgCastVote{Creator: juror, DisputeId: od.Id, Accept: true, StakeAmount: 100})
	require.NoError(t, err)
	_, err = ms.ResolveDispute(f.ctx, &types.MsgResolveDispute{Creator: addrPayee, DisputeId: od.Id})
	require.NoError(t, err)
	require.Equal(t, uint64(500), balanceOf(t, f, addrPayee), "100 seed - 100 bond + 400 escrow + 100 bond returned")
}

// S5 — Everything was honest; the buyer challenges anyway hoping to claw
// back. The jury upholds the release: the verifier's stake comes home AND the
// challenger's dispute-bond is slashed to the verifier — the griefed party,
// compensated for the harassment. Cry-wolf has a price.
func TestFrivolousChallengeSlashedToVerifier(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	verifier := sample.AccAddress()
	juror := sample.AccAddress()
	setJurors(t, f, juror)
	seedAccount(t, f, addrPayer, 1000)
	seedAccount(t, f, addrPayee, 0)
	seedAccount(t, f, verifier, 1000)
	seedAccount(t, f, juror, 1000)

	id := lockVerified(t, f, ms, 1, verifier)
	submitEncrypted(t, f, ms, id)
	_, err := ms.AttestEscrow(setBlockTime(f.ctx, 115), attestMsg(verifier, id, true, 400)) // honest
	require.NoError(t, err)
	_, err = ms.VerifiedRelease(setBlockTime(f.ctx, 120), &types.MsgVerifiedRelease{Creator: addrPayee, EscrowId: id, Key: artifactKey})
	require.NoError(t, err)

	od, err := ms.OpenDispute(setBlockTime(f.ctx, 130), &types.MsgOpenDispute{
		Creator: addrPayer, EscrowId: id, Reason: "buyer's remorse", BondAmount: 100,
	})
	require.NoError(t, err)
	_, err = ms.CastVote(f.ctx, &types.MsgCastVote{Creator: juror, DisputeId: od.Id, Accept: true, StakeAmount: 100})
	require.NoError(t, err)
	rd, err := ms.ResolveDispute(f.ctx, &types.MsgResolveDispute{Creator: addrPayer, DisputeId: od.Id})
	require.NoError(t, err)
	require.Equal(t, types.DisputeResolutionAccept, rd.Resolution)

	require.Equal(t, uint64(1100), balanceOf(t, f, verifier), "stake returned + the challenger's slashed bond")
	require.Equal(t, uint64(500), balanceOf(t, f, addrPayer), "600 after lock - 100 bond, slashed")
	require.Equal(t, uint64(400), balanceOf(t, f, addrPayee))
	esc, _ := f.keeper.Escrow.Get(f.ctx, id)
	require.Equal(t, types.EscrowStatusReleased, esc.Status)
}

// S7 — The hole the six original scenarios missed: buyer+verifier collude on
// a FALSE FAIL against correct work. The payee contests inside the window;
// the jury finds the work good → escrow releases to the payee AND the
// false-fail stake is slashed to the payee — symmetric with S3.
func TestFalseFailContestSlashesVerifierToSeller(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	verifier := sample.AccAddress()
	juror := sample.AccAddress()
	setJurors(t, f, juror)
	seedAccount(t, f, addrPayer, 1000)
	seedAccount(t, f, addrPayee, 100) // funds the contest bond
	seedAccount(t, f, verifier, 1000)
	seedAccount(t, f, juror, 1000)

	id := lockVerified(t, f, ms, 1, verifier)
	submitEncrypted(t, f, ms, id)
	_, err := ms.AttestEscrow(setBlockTime(f.ctx, 115), attestMsg(verifier, id, false, 400)) // the false fail
	require.NoError(t, err)

	// Payee contests inside the window (a payer cannot — it's not their window).
	_, err = ms.OpenDispute(setBlockTime(f.ctx, 120), &types.MsgOpenDispute{
		Creator: addrPayer, EscrowId: id, Reason: "x", BondAmount: 100,
	})
	require.Error(t, err, "only the payee may contest a fail attestation")
	od, err := ms.OpenDispute(setBlockTime(f.ctx, 120), &types.MsgOpenDispute{
		Creator: addrPayee, EscrowId: id, Reason: "work passes the pinned tests; fail is fabricated", BondAmount: 100,
	})
	require.NoError(t, err)

	// While in jury, the colluding buyer cannot refund.
	_, err = ms.RefundEscrow(setBlockTime(f.ctx, 125), &types.MsgRefundEscrow{Creator: addrPayer, Id: id})
	require.Error(t, err)

	// Jury re-runs the pinned procedure: work is good → accept.
	_, err = ms.CastVote(f.ctx, &types.MsgCastVote{Creator: juror, DisputeId: od.Id, Accept: true, StakeAmount: 100})
	require.NoError(t, err)
	rd, err := ms.ResolveDispute(f.ctx, &types.MsgResolveDispute{Creator: addrPayee, DisputeId: od.Id})
	require.NoError(t, err)
	require.Equal(t, types.DisputeResolutionAccept, rd.Resolution)

	// Payee: 100 - 100 bond + 400 escrow + 400 slashed stake + 100 bond = 900.
	require.Equal(t, uint64(900), balanceOf(t, f, addrPayee), "escrow + the false-fail stake + contest bond returned")
	require.Equal(t, uint64(600), balanceOf(t, f, verifier), "false fail costs the stake, symmetric with false pass")
	require.Equal(t, uint64(600), balanceOf(t, f, addrPayer), "colluding buyer got nothing back")
}

// H1 — The chain never learns HOW the verifier verified: an LLM-judge
// acceptance spec (model + rubric + sampling + threshold, hashed into
// acceptance_hash) flows through the exact same path as a pytest suite.
// "Expensive to create, cheap to verify" only changes the verifier's
// off-chain bill, not one byte of chain state.
func TestVerificationMethodIsOpaqueToTheChain(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	verifier := sample.AccAddress()
	seedAccount(t, f, addrPayer, 1000)
	seedAccount(t, f, addrPayee, 0)
	seedAccount(t, f, verifier, 1000)

	judgeSpec := `{"judge":"claude-haiku-4-5","temperature":0,"samples":5,` +
		`"pass_threshold":"4/5","challenge_threshold":"<=2/5",` +
		`"rubric":"essay covers X,Y,Z; no fabricated citations"}`

	resp, err := ms.LockEscrow(setBlockTime(f.ctx, 100), &types.MsgLockEscrow{
		Creator: addrPayer, Payee: addrPayee, Amount: 400, Ref: "essay-deal",
		DisputeSeconds: 50, VerifierAddrs: []string{verifier}, VerifierQuorum: 1,
		AcceptanceHash: hexHash(judgeSpec), // pin the JUDGE, not a test suite
	})
	require.NoError(t, err)
	id := resp.Id
	submitEncrypted(t, f, ms, id)

	// Verifier ran haiku 5x at temp 0 off-chain: 5/5 over threshold → pass.
	// The verdict restates the JUDGE spec hash, same as it would a test suite.
	_, err = ms.AttestEscrow(setBlockTime(f.ctx, 115), &types.MsgAttestEscrow{
		Creator: verifier, EscrowId: id, Passed: true, StakeAmount: 400,
		DeliveryHash: deliveryHash, KeyHash: hexHash(artifactKey), AcceptanceHash: hexHash(judgeSpec),
	})
	require.NoError(t, err)
	_, err = ms.VerifiedRelease(setBlockTime(f.ctx, 120), &types.MsgVerifiedRelease{Creator: addrPayee, EscrowId: id, Key: artifactKey})
	require.NoError(t, err)
	require.Equal(t, uint64(400), balanceOf(t, f, addrPayee))

	esc, _ := f.keeper.Escrow.Get(f.ctx, id)
	require.Equal(t, hexHash(judgeSpec), esc.AcceptanceHash, "any juror can check the rubric was the precommitted one")
}

// m-of-n: with a 2-of-3 set, one pass attestation is not a release authority;
// the second is. Quorum was a day-one parameter, not a migration.
func TestQuorumTwoOfThree(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	v1, v2, v3 := sample.AccAddress(), sample.AccAddress(), sample.AccAddress()
	seedAccount(t, f, addrPayer, 1000)
	seedAccount(t, f, addrPayee, 0)
	for _, v := range []string{v1, v2, v3} {
		seedAccount(t, f, v, 1000)
	}

	id := lockVerified(t, f, ms, 2, v1, v2, v3)
	submitEncrypted(t, f, ms, id)

	_, err := ms.AttestEscrow(setBlockTime(f.ctx, 115), attestMsg(v1, id, true, 400))
	require.NoError(t, err)
	_, err = ms.VerifiedRelease(setBlockTime(f.ctx, 116), &types.MsgVerifiedRelease{Creator: addrPayee, EscrowId: id, Key: artifactKey})
	require.Error(t, err, "1 of 2 required pass attestations must not release")

	_, err = ms.AttestEscrow(setBlockTime(f.ctx, 117), attestMsg(v2, id, true, 400))
	require.NoError(t, err)
	_, err = ms.VerifiedRelease(setBlockTime(f.ctx, 118), &types.MsgVerifiedRelease{Creator: addrPayee, EscrowId: id, Key: artifactKey})
	require.NoError(t, err)
	require.Equal(t, uint64(400), balanceOf(t, f, addrPayee))
}

// Guard rails: every smaller hole the design review called out.
func TestVerifierGuards(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	verifier := sample.AccAddress()
	outsider := sample.AccAddress()
	seedAccount(t, f, addrPayer, 2000)
	seedAccount(t, f, addrPayee, 0)
	seedAccount(t, f, verifier, 1000)
	seedAccount(t, f, outsider, 1000)

	// Lock-time validation.
	lc := setBlockTime(f.ctx, 100)
	_, err := ms.LockEscrow(lc, &types.MsgLockEscrow{Creator: addrPayer, Payee: addrPayee, Amount: 400, DisputeSeconds: 50,
		VerifierAddrs: []string{verifier}, VerifierQuorum: 0})
	require.Error(t, err, "quorum 0 with verifiers")
	_, err = ms.LockEscrow(lc, &types.MsgLockEscrow{Creator: addrPayer, Payee: addrPayee, Amount: 400, DisputeSeconds: 50,
		VerifierAddrs: []string{verifier}, VerifierQuorum: 2})
	require.Error(t, err, "quorum above set size")
	_, err = ms.LockEscrow(lc, &types.MsgLockEscrow{Creator: addrPayer, Payee: addrPayee, Amount: 400, DisputeSeconds: 50,
		VerifierAddrs: []string{addrPayee}, VerifierQuorum: 1})
	require.Error(t, err, "payee cannot verify its own work")
	_, err = ms.LockEscrow(lc, &types.MsgLockEscrow{Creator: addrPayer, Payee: addrPayee, Amount: 400, DisputeSeconds: 50,
		VerifierAddrs: []string{verifier, verifier}, VerifierQuorum: 2})
	require.Error(t, err, "duplicate verifier")
	_, err = ms.LockEscrow(lc, &types.MsgLockEscrow{Creator: addrPayer, Payee: addrPayee, Amount: 400, DisputeSeconds: 50,
		VerifierQuorum: 1})
	require.Error(t, err, "quorum without verifiers")

	id := lockVerified(t, f, ms, 1, verifier)

	// Attestation requires a submitted escrow…
	_, err = ms.AttestEscrow(setBlockTime(f.ctx, 105), attestMsg(verifier, id, true, 400))
	require.Error(t, err, "nothing to judge before submit")
	submitEncrypted(t, f, ms, id)

	// …set membership, full bond coverage, and one verdict per verifier.
	_, err = ms.AttestEscrow(setBlockTime(f.ctx, 115), attestMsg(outsider, id, true, 400))
	require.Error(t, err, "outsider cannot attest")
	_, err = ms.AttestEscrow(setBlockTime(f.ctx, 115), attestMsg(verifier, id, true, 399))
	require.Error(t, err, "stake below the bounty is theater, not coverage")
	_, err = ms.AttestEscrow(setBlockTime(f.ctx, 115), attestMsg(verifier, id, true, 400))
	require.NoError(t, err)
	_, err = ms.AttestEscrow(setBlockTime(f.ctx, 116), attestMsg(verifier, id, false, 400))
	require.Error(t, err, "no second verdict from the same verifier")

	// The free pre-release freeze is closed for verifier-bound escrows.
	_, err = ms.DisputeEscrow(setBlockTime(f.ctx, 116), &types.MsgDisputeEscrow{Creator: addrPayer, Id: id})
	require.Error(t, err, "bond-less freeze would stall verified release for free")

	// Fair exchange: a wrong key can never trigger settlement.
	_, err = ms.VerifiedRelease(setBlockTime(f.ctx, 120), &types.MsgVerifiedRelease{Creator: addrPayee, EscrowId: id, Key: "not-the-key"})
	require.Error(t, err, "wrong key preimage")
	_, err = ms.VerifiedRelease(setBlockTime(f.ctx, 120), &types.MsgVerifiedRelease{Creator: addrPayee, EscrowId: id, Key: artifactKey})
	require.NoError(t, err)

	// Challenge discipline: payee can't challenge, the window is hard, and a
	// second settlement is impossible.
	esc, _ := f.keeper.Escrow.Get(f.ctx, id)
	_, err = ms.OpenDispute(setBlockTime(f.ctx, 130), &types.MsgOpenDispute{Creator: addrPayee, EscrowId: id, Reason: "x", BondAmount: 100})
	require.Error(t, err, "only the payer may challenge a verified release")
	_, err = ms.OpenDispute(setBlockTime(f.ctx, esc.ChallengeDeadline+1), &types.MsgOpenDispute{Creator: addrPayer, EscrowId: id, Reason: "x", BondAmount: 100})
	require.Error(t, err, "challenge after the window")
	_, err = ms.VerifiedRelease(setBlockTime(f.ctx, 130), &types.MsgVerifiedRelease{Creator: addrPayee, EscrowId: id, Key: artifactKey})
	require.Error(t, err, "double release")

	// A payer-goodwill release leaves nothing to challenge — there is no
	// attester collateral behind it.
	id2 := lockVerified(t, f, ms, 1, verifier)
	_, err = ms.ReleaseEscrow(setBlockTime(f.ctx, 120), &types.MsgReleaseEscrow{Creator: addrPayer, Id: id2})
	require.NoError(t, err, "the payer may always release directly")
	_, err = ms.OpenDispute(setBlockTime(f.ctx, 125), &types.MsgOpenDispute{Creator: addrPayer, EscrowId: id2, Reason: "x", BondAmount: 100})
	require.Error(t, err, "no attestations -> nothing to claim fraud against")

	// A verifier-bound delivery MUST commit the fair exchange. Key-less /
	// hash-less submits are rejected outright: otherwise a seller + truthful
	// verifier could collect via VerifiedRelease while withholding the
	// artifact from the buyer — the attestation would be TRUE, so the fraud
	// challenge could never make the buyer whole.
	id3 := lockVerified(t, f, ms, 1, verifier)
	_, err = ms.SubmitEscrow(setBlockTime(f.ctx, 110), &types.MsgSubmitEscrow{Creator: addrPayee, Id: id3}) // no commitments
	require.Error(t, err, "verifier-bound submit without delivery_hash+key_hash")
}
