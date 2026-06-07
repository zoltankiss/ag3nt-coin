package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zoltankiss/agntcoin/testutil/sample"
	"github.com/zoltankiss/agntcoin/x/agntcoin/keeper"
	"github.com/zoltankiss/agntcoin/x/agntcoin/types"
)

func scopedEvidenceMsg(issuer, recipient string) *types.MsgCastScopedEvidenceVouch {
	return &types.MsgCastScopedEvidenceVouch{
		Creator:        issuer,
		Recipient:      recipient,
		Scope:          "protocol-pr-review/security-low",
		Weight:         25,
		ArtifactUri:    "https://github.com/zoltankiss/agnt-coin-forge-2/blob/abc123/evidence.json",
		ArtifactSha256: hexHash("review-artifact"),
		EvidenceUri:    "https://github.com/zoltankiss/agnt-coin-forge-2/blob/abc123/rationale.md",
		EvidenceSha256: hexHash("review-rationale"),
		RationaleHash:  hexHash("founder says review was coherent"),
		ExpiresAt:      0,
	}
}

func TestCastScopedEvidenceVouchAllowsZeroCoinAnchor(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	anchor := sample.AccAddress()
	recipient := sample.AccAddress()
	setJurors(t, f, anchor)
	seedAccount(t, f, anchor, 0)
	seedAccount(t, f, recipient, 0)

	resp, err := ms.CastScopedEvidenceVouch(f.ctx, scopedEvidenceMsg(anchor, recipient))
	require.NoError(t, err)
	require.Equal(t, uint64(0), resp.Id)
	require.Equal(t, uint64(0), balanceOf(t, f, anchor), "scoped evidence vouch is reputation-backed, not coin-staked")

	vouch, err := f.keeper.ScopedEvidenceVouch.Get(f.ctx, resp.Id)
	require.NoError(t, err)
	require.Equal(t, anchor, vouch.Issuer)
	require.Equal(t, recipient, vouch.Recipient)
	require.Equal(t, "protocol-pr-review/security-low", vouch.Scope)
	require.Equal(t, uint64(25), vouch.Weight)
}

func TestCastScopedEvidenceVouchRejectsLowRepNonAnchor(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	anchor := sample.AccAddress()
	outsider := sample.AccAddress()
	recipient := sample.AccAddress()
	setJurors(t, f, anchor)
	seedAccount(t, f, outsider, 1000)
	seedAccount(t, f, recipient, 0)

	_, err := ms.CastScopedEvidenceVouch(f.ctx, scopedEvidenceMsg(outsider, recipient))
	require.Error(t, err)
}

func TestCastScopedEvidenceVouchRejectsBadEvidence(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	anchor := sample.AccAddress()
	recipient := sample.AccAddress()
	setJurors(t, f, anchor)
	seedAccount(t, f, anchor, 0)
	seedAccount(t, f, recipient, 0)

	bad := scopedEvidenceMsg(anchor, recipient)
	bad.EvidenceSha256 = "not-a-sha256"
	_, err := ms.CastScopedEvidenceVouch(f.ctx, bad)
	require.Error(t, err)

	blankScope := scopedEvidenceMsg(anchor, recipient)
	blankScope.Scope = ""
	_, err = ms.CastScopedEvidenceVouch(f.ctx, blankScope)
	require.Error(t, err)
}
