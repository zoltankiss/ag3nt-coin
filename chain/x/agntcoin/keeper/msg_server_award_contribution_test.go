package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zoltankiss/agntcoin/testutil/sample"
	"github.com/zoltankiss/agntcoin/x/agntcoin/keeper"
	"github.com/zoltankiss/agntcoin/x/agntcoin/types"
)

func contributionAwardMsg(anchor, recipient string, amount uint64) *types.MsgAwardContribution {
	return &types.MsgAwardContribution{
		Creator:        anchor,
		Recipient:      recipient,
		RepoUrl:        "https://github.com/zoltankiss/agntcoin",
		PrUrl:          "https://github.com/zoltankiss/agntcoin/pull/404",
		CommitSha:      "0123456789abcdef0123456789abcdef01234567",
		ArtifactUri:    "https://github.com/zoltankiss/agntcoin/pull/404",
		ArtifactSha256: hexHash("accepted-protocol-patch-artifact"),
		EvidenceSha256: hexHash("founder-review-evidence-bundle"),
		Scope:          "protocol-contribution/docs-test-low",
		RationaleHash:  hexHash("accepted because it fixes beachhead CLI docs"),
		Amount:         amount,
	}
}

func TestAwardContributionMintsAcceptedPatchReward(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	anchor := sample.AccAddress()
	author := sample.AccAddress()
	setJurors(t, f, anchor)
	seedAccount(t, f, anchor, 0)
	seedAccount(t, f, author, 0)

	resp, err := ms.AwardContribution(f.ctx, contributionAwardMsg(anchor, author, 100))
	require.NoError(t, err)
	require.Equal(t, uint64(0), resp.Id)
	require.Equal(t, uint64(0), balanceOf(t, f, anchor), "author mint is protocol-issued, not debited from the founder")
	require.Equal(t, uint64(100), balanceOf(t, f, author))

	award, err := f.keeper.ContributionAward.Get(f.ctx, resp.Id)
	require.NoError(t, err)
	require.Equal(t, anchor, award.Anchor)
	require.Equal(t, author, award.Recipient)
	require.Equal(t, uint64(100), award.Amount)
	require.Equal(t, "protocol-contribution/docs-test-low", award.Scope)
}

func TestAwardContributionRejectsNonAnchor(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	anchor := sample.AccAddress()
	outsider := sample.AccAddress()
	author := sample.AccAddress()
	setJurors(t, f, anchor)
	seedAccount(t, f, outsider, 1000)
	seedAccount(t, f, author, 0)

	_, err := ms.AwardContribution(f.ctx, contributionAwardMsg(outsider, author, 100))
	require.Error(t, err)
	require.Equal(t, uint64(0), balanceOf(t, f, author))
}

func TestAwardContributionRejectsDuplicateArtifact(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	anchor := sample.AccAddress()
	author := sample.AccAddress()
	setJurors(t, f, anchor)
	seedAccount(t, f, anchor, 0)
	seedAccount(t, f, author, 0)

	msg := contributionAwardMsg(anchor, author, 100)
	_, err := ms.AwardContribution(f.ctx, msg)
	require.NoError(t, err)

	dup := contributionAwardMsg(anchor, author, 100)
	dup.CommitSha = "abcdef0123456789abcdef0123456789abcdef01"
	_, err = ms.AwardContribution(f.ctx, dup)
	require.Error(t, err)
	require.Equal(t, uint64(100), balanceOf(t, f, author), "duplicate artifact must not mint twice")
}

func TestAwardContributionRejectsBadHashAndCap(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	anchor := sample.AccAddress()
	author := sample.AccAddress()
	setJurors(t, f, anchor)
	seedAccount(t, f, anchor, 0)
	seedAccount(t, f, author, 0)

	badHash := contributionAwardMsg(anchor, author, 100)
	badHash.ArtifactSha256 = "not-a-sha256"
	_, err := ms.AwardContribution(f.ctx, badHash)
	require.Error(t, err)

	tooLarge := contributionAwardMsg(anchor, author, types.MaxContributionAward+1)
	tooLarge.ArtifactSha256 = hexHash("different-artifact")
	_, err = ms.AwardContribution(f.ctx, tooLarge)
	require.Error(t, err)

	require.Equal(t, uint64(0), balanceOf(t, f, author))
}
