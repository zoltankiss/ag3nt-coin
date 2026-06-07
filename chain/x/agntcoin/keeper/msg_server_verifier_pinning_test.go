package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zoltankiss/agntcoin/testutil/sample"
	"github.com/zoltankiss/agntcoin/x/agntcoin/keeper"
	"github.com/zoltankiss/agntcoin/x/agntcoin/types"
)

// verifier-v1 hardening (0.3.x): four scenario fixes argued through after the
// initial release.
//
//  1. FAIR EXCHANGE IS NOT OPTIONAL — a verifier-bound escrow without a
//     delivery_hash + key_hash commitment silently downgrades to custodial
//     trust (and, worse, lets a seller + honest verifier collect payment via
//     VerifiedRelease while withholding the artifact from the buyer — the
//     attestation is TRUE, so the fraud challenge can't help). Submit rejects
//     verifier-bound deliveries without both commitments.
//  2. PINNED PROCEDURE IS NOT OPTIONAL — an empty acceptance_hash IS the
//     unpinned-procedure hole. Lock rejects verifier-bound escrows without it.
//     And the attestation must RESTATE what it judged (delivery_hash,
//     key_hash, acceptance_hash): the chain rejects any mismatch, so a
//     verdict can never be re-bound to a procedure or artifact the verifier
//     did not check, and the stored attestation is self-contained evidence.
//  3. NO OVERBOOKING — each attestation locks its own fresh stake, so one
//     bankroll can never back two concurrent bounties (characterization).
//  4. EVIDENCE BUNDLE — the escrow record alone (artifact_uri + hashes +
//     attestations + block timestamps) lets a juror verify what was
//     delivered, against what procedure, and when — never trusting a mutable
//     app DB row.
//
// A hash that cannot be re-checked is not a commitment: every supplied hash
// must be a well-formed hex sha256 (64 hex chars) or the tx fails.

// 1+2 (lock side) — the acceptance procedure must be pinned, verifiably,
// before any work starts.
func TestVerifierBoundLockRequiresPinnedAcceptance(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	verifier := sample.AccAddress()
	seedAccount(t, f, addrPayer, 1000)
	seedAccount(t, f, addrPayee, 0)

	lc := setBlockTime(f.ctx, 100)
	base := func() *types.MsgLockEscrow {
		return &types.MsgLockEscrow{
			Creator: addrPayer, Payee: addrPayee, Amount: 400, Ref: "deal",
			DisputeSeconds: 50, VerifierAddrs: []string{verifier}, VerifierQuorum: 1,
		}
	}

	m := base() // acceptance_hash empty = the unpinned-procedure hole
	_, err := ms.LockEscrow(lc, m)
	require.Error(t, err, "verifier-bound lock without acceptance_hash must fail")

	m = base()
	m.AcceptanceHash = "feedc0de" // not a sha256 — no juror can ever re-check it
	_, err = ms.LockEscrow(lc, m)
	require.Error(t, err, "malformed acceptance_hash must fail")

	m = base()
	m.AcceptanceHash = acceptHash
	resp, err := ms.LockEscrow(lc, m)
	require.NoError(t, err)
	esc, _ := f.keeper.Escrow.Get(f.ctx, resp.Id)
	require.Equal(t, acceptHash, esc.AcceptanceHash)
}

// 1 (submit side) — a verifier-bound delivery must commit BOTH the artifact
// bytes and the decryption key; otherwise the only tx that pays the seller
// would no longer be the tx that hands the buyer the key.
func TestVerifierBoundSubmitRequiresFairExchangeCommitments(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	verifier := sample.AccAddress()
	seedAccount(t, f, addrPayer, 5000)
	seedAccount(t, f, addrPayee, 0)

	id := lockVerified(t, f, ms, 1, verifier)
	sub := func(delivery, key string) error {
		_, err := ms.SubmitEscrow(setBlockTime(f.ctx, 110), &types.MsgSubmitEscrow{
			Creator: addrPayee, Id: id, DeliveryHash: delivery, KeyHash: key,
		})
		return err
	}

	require.Error(t, sub("", ""), "key-less + hash-less verifier delivery must fail")
	require.Error(t, sub(deliveryHash, ""), "missing key_hash must fail")
	require.Error(t, sub("", hexHash(artifactKey)), "missing delivery_hash must fail")
	require.Error(t, sub("feedc0de", hexHash(artifactKey)), "malformed delivery_hash must fail")
	require.Error(t, sub(deliveryHash, "zz"), "malformed key_hash must fail")
	require.NoError(t, sub(deliveryHash, hexHash(artifactKey)))

	esc, _ := f.keeper.Escrow.Get(f.ctx, id)
	require.Equal(t, int64(110), esc.SubmittedAt, "submit must pin block time for the evidence timeline")
}

// Ordinary (non-verifier) escrows keep their optionality — but a hash that IS
// supplied must still be re-checkable.
func TestOrdinaryEscrowHashesStayOptionalButValidated(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	seedAccount(t, f, addrPayer, 1000)
	seedAccount(t, f, addrPayee, 0)

	id := lockFresh(t, f, ms)
	_, err := ms.SubmitEscrow(setBlockTime(f.ctx, 110), &types.MsgSubmitEscrow{
		Creator: addrPayee, Id: id, DeliveryHash: "feedc0de",
	})
	require.Error(t, err, "a malformed hash is worse than none: it looks like evidence and verifies nothing")

	_, err = ms.SubmitEscrow(setBlockTime(f.ctx, 110), &types.MsgSubmitEscrow{Creator: addrPayee, Id: id})
	require.NoError(t, err, "hash-less ordinary submit stays legal")
}

// 2 (attest side) — the verdict is bound to WHAT was judged. An attestation
// naming different tests, different bytes, or a different key commitment than
// the escrow record cannot land.
func TestAttestationBoundToCommitments(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	verifier := sample.AccAddress()
	seedAccount(t, f, addrPayer, 1000)
	seedAccount(t, f, addrPayee, 0)
	seedAccount(t, f, verifier, 1000)

	id := lockVerified(t, f, ms, 1, verifier)
	submitEncrypted(t, f, ms, id)

	att := func(delivery, key, acceptance string) error {
		_, err := ms.AttestEscrow(setBlockTime(f.ctx, 115), &types.MsgAttestEscrow{
			Creator: verifier, EscrowId: id, Passed: true, StakeAmount: 400,
			DeliveryHash: delivery, KeyHash: key, AcceptanceHash: acceptance,
		})
		return err
	}

	other := hexHash("something-else-entirely")
	require.Error(t, att("", "", ""), "an attestation that names nothing attests nothing")
	require.Error(t, att(other, hexHash(artifactKey), acceptHash), "different bytes than committed")
	require.Error(t, att(deliveryHash, other, acceptHash), "different key commitment than committed")
	require.Error(t, att(deliveryHash, hexHash(artifactKey), other), "different acceptance procedure than pinned")
	require.NoError(t, att(deliveryHash, hexHash(artifactKey), acceptHash))

	// The stored attestation is self-contained evidence: verdict + the three
	// commitments it was made against + when.
	esc, _ := f.keeper.Escrow.Get(f.ctx, id)
	require.Len(t, esc.Attestations, 1)
	a := esc.Attestations[0]
	require.Equal(t, deliveryHash, a.DeliveryHash)
	require.Equal(t, hexHash(artifactKey), a.KeyHash)
	require.Equal(t, acceptHash, a.AcceptanceHash)
	require.Equal(t, int64(115), a.AttestedAt)
}

// 2 (release side, belt-and-braces) — even if a future code path ever stored
// an attestation whose pins drifted from the escrow's commitments, it must
// not count toward the release quorum. Exercised by direct state injection
// (no public msg can produce this state today — that is the point).
func TestVerifiedReleaseIgnoresMismatchedAttestations(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	verifier := sample.AccAddress()
	seedAccount(t, f, addrPayer, 1000)
	seedAccount(t, f, addrPayee, 0)
	seedAccount(t, f, verifier, 1000)

	id := lockVerified(t, f, ms, 1, verifier)
	submitEncrypted(t, f, ms, id)

	esc, _ := f.keeper.Escrow.Get(f.ctx, id)
	esc.Attestations = append(esc.Attestations, &types.Attestation{
		Verifier: verifier, Passed: true, BondId: 999,
		DeliveryHash:   hexHash("not-what-was-submitted"), // judged the wrong bytes
		KeyHash:        esc.KeyHash,
		AcceptanceHash: esc.AcceptanceHash,
		AttestedAt:     115,
	})
	require.NoError(t, f.keeper.Escrow.Set(f.ctx, id, esc))

	_, err := ms.VerifiedRelease(setBlockTime(f.ctx, 120), &types.MsgVerifiedRelease{
		Creator: addrPayee, EscrowId: id, Key: artifactKey,
	})
	require.Error(t, err, "an attestation over different bytes is not a release authority")
}

// 3 — one bankroll cannot back two concurrent bounties: every attestation
// locks its own fresh stake, so coverage is encumbered per deal, by
// construction (the overbooking case the create-time-spot-check design would
// have raced on).
func TestVerifierCannotOverbookStakeAcrossEscrows(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	verifier := sample.AccAddress()
	seedAccount(t, f, addrPayer, 2000)
	seedAccount(t, f, addrPayee, 0)
	seedAccount(t, f, verifier, 700) // covers one 400-bounty, not two

	idA := lockVerified(t, f, ms, 1, verifier)
	idB := lockVerified(t, f, ms, 1, verifier)
	submitEncrypted(t, f, ms, idA)
	submitEncrypted(t, f, ms, idB)

	_, err := ms.AttestEscrow(setBlockTime(f.ctx, 115), attestMsg(verifier, idA, true, 400))
	require.NoError(t, err)
	require.Equal(t, uint64(300), balanceOf(t, f, verifier), "stake encumbered immediately")

	_, err = ms.AttestEscrow(setBlockTime(f.ctx, 116), attestMsg(verifier, idB, true, 400))
	require.Error(t, err, "the same funds cannot back a second concurrent bounty")
}

// 4 — the on-chain escrow record alone is the evidence bundle: artifact
// reference + every commitment + every attestation + the full timeline. A
// juror needs nothing from the app's database, and tampered artifact bytes
// are rejected by re-hashing against the committed delivery_hash.
func TestEscrowIsSelfContainedEvidenceBundle(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	verifier := sample.AccAddress()
	seedAccount(t, f, addrPayer, 1000)
	seedAccount(t, f, addrPayee, 0)
	seedAccount(t, f, verifier, 1000)

	id := lockVerified(t, f, ms, 1, verifier)

	const uri = "ipfs://bafybeigdyrzt5sfp7udm7hu76uh7y26nf3efuylqabf3oclgtqy55fbzdi"
	_, err := ms.SubmitEscrow(setBlockTime(f.ctx, 110), &types.MsgSubmitEscrow{
		Creator: addrPayee, Id: id,
		DeliveryHash: hexHash(artifactCiphertext), KeyHash: hexHash(artifactKey),
		ArtifactUri: uri,
	})
	require.NoError(t, err)
	_, err = ms.AttestEscrow(setBlockTime(f.ctx, 115), attestMsg(verifier, id, true, 400))
	require.NoError(t, err)
	_, err = ms.VerifiedRelease(setBlockTime(f.ctx, 120), &types.MsgVerifiedRelease{
		Creator: addrPayee, EscrowId: id, Key: artifactKey,
	})
	require.NoError(t, err)

	// The bundle, straight off the chain (QueryEscrow serves this record):
	esc, err := f.keeper.Escrow.Get(f.ctx, id)
	require.NoError(t, err)
	require.Equal(t, uri, esc.ArtifactUri, "where to fetch the ciphertext")
	require.Equal(t, hexHash(artifactCiphertext), esc.DeliveryHash, "what the bytes must hash to")
	require.Equal(t, hexHash(artifactKey), esc.KeyHash, "what the revealed key must hash to")
	require.Equal(t, acceptHash, esc.AcceptanceHash, "which procedure was pinned")
	require.Equal(t, int64(110), esc.SubmittedAt)
	require.Equal(t, int64(120), esc.ReleasedAt)
	require.Equal(t, int64(120+types.ChallengeWindowSeconds), esc.ChallengeDeadline)
	require.Len(t, esc.Attestations, 1)
	a := esc.Attestations[0]
	require.Equal(t, verifier, a.Verifier)
	require.True(t, a.Passed)
	require.Equal(t, int64(115), a.AttestedAt)
	require.Equal(t, esc.DeliveryHash, a.DeliveryHash, "the verdict names the exact bytes it judged")
	require.Equal(t, esc.AcceptanceHash, a.AcceptanceHash, "…and the exact procedure it ran")

	// The juror's mechanical check: fetch bytes from artifact_uri, re-hash,
	// compare. The genuine ciphertext verifies; a mutated app-DB copy does not.
	require.Equal(t, esc.DeliveryHash, hexHash(artifactCiphertext), "genuine artifact bytes verify")
	require.NotEqual(t, esc.DeliveryHash, hexHash(artifactCiphertext+" (tampered)"), "tampered bytes are rejected")
}
