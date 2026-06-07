package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zoltankiss/agntcoin/testutil/sample"
	"github.com/zoltankiss/agntcoin/x/agntcoin/keeper"
	"github.com/zoltankiss/agntcoin/x/agntcoin/types"
)

func setEmission(t *testing.T, f *fixture, e types.EmissionState) {
	t.Helper()
	require.NoError(t, f.keeper.EmissionState.Set(f.ctx, e))
}

func testEmission(maxSupply, divisor, scheduled uint64, startedAt int64) types.EmissionState {
	return types.EmissionState{
		MaxSupply:            maxSupply,
		EpochLengthSeconds:   100,
		EpochRewardDivisor:   divisor,
		GenesisTime:          startedAt,
		CurrentEpoch:         1,
		EpochStartedAt:       startedAt,
		EpochScheduled:       scheduled,
		EpochMined:           0,
		TotalMined:           0,
		TotalBurnedUnclaimed: 0,
	}
}

func TestEmissionContributionAwardConsumesEpochAndSupply(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	qs := keeper.NewQueryServerImpl(f.keeper)
	anchor := sample.AccAddress()
	author := sample.AccAddress()
	setJurors(t, f, anchor)
	seedAccount(t, f, anchor, 0)
	seedAccount(t, f, author, 0)
	setEmission(t, f, testEmission(100, 10, 10, 100))

	_, err := ms.AwardContribution(setBlockTime(f.ctx, 110), contributionAwardMsg(anchor, author, 4))
	require.NoError(t, err)
	require.Equal(t, uint64(4), balanceOf(t, f, author))

	resp, err := qs.Emission(f.ctx, &types.QueryEmissionRequest{})
	require.NoError(t, err)
	require.Equal(t, uint64(100), resp.Emission.MaxSupply)
	require.Equal(t, uint64(1), resp.Emission.CurrentEpoch)
	require.Equal(t, uint64(10), resp.Emission.EpochScheduled)
	require.Equal(t, uint64(4), resp.Emission.EpochMined)
	require.Equal(t, uint64(4), resp.Emission.TotalMined)
	require.Equal(t, uint64(0), resp.Emission.TotalBurnedUnclaimed)
	require.Equal(t, uint64(96), resp.RemainingScheduled)
	require.Equal(t, uint64(6), resp.RemainingEpoch)
}

func TestEmissionEpochRolloverBurnsUnclaimedScheduledReward(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	qs := keeper.NewQueryServerImpl(f.keeper)
	anchor := sample.AccAddress()
	author := sample.AccAddress()
	setJurors(t, f, anchor)
	seedAccount(t, f, anchor, 0)
	seedAccount(t, f, author, 0)
	setEmission(t, f, testEmission(100, 10, 10, 100))

	_, err := ms.AwardContribution(setBlockTime(f.ctx, 110), contributionAwardMsg(anchor, author, 4))
	require.NoError(t, err)

	next := contributionAwardMsg(anchor, author, 1)
	next.ArtifactSha256 = hexHash("different artifact")
	next.EvidenceSha256 = hexHash("different evidence")
	_, err = ms.AwardContribution(setBlockTime(f.ctx, 201), next)
	require.NoError(t, err)

	resp, err := qs.Emission(f.ctx, &types.QueryEmissionRequest{})
	require.NoError(t, err)
	require.Equal(t, uint64(2), resp.Emission.CurrentEpoch)
	require.Equal(t, int64(200), resp.Emission.EpochStartedAt)
	require.Equal(t, uint64(9), resp.Emission.EpochScheduled)
	require.Equal(t, uint64(1), resp.Emission.EpochMined)
	require.Equal(t, uint64(5), resp.Emission.TotalMined)
	require.Equal(t, uint64(6), resp.Emission.TotalBurnedUnclaimed)
	require.Equal(t, uint64(89), resp.RemainingScheduled)
	require.Equal(t, uint64(8), resp.RemainingEpoch)
}

func TestEmissionRejectsMintBeyondHardCap(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	anchor := sample.AccAddress()
	author := sample.AccAddress()
	setJurors(t, f, anchor)
	seedAccount(t, f, anchor, 0)
	seedAccount(t, f, author, 0)
	setEmission(t, f, testEmission(5, 1, 5, 100))

	_, err := ms.AwardContribution(setBlockTime(f.ctx, 110), contributionAwardMsg(anchor, author, 5))
	require.NoError(t, err)

	over := contributionAwardMsg(anchor, author, 1)
	over.ArtifactSha256 = hexHash("over cap artifact")
	over.EvidenceSha256 = hexHash("over cap evidence")
	_, err = ms.AwardContribution(setBlockTime(f.ctx, 111), over)
	require.ErrorContains(t, err, "mint exceeds remaining max supply")
	require.Equal(t, uint64(5), balanceOf(t, f, author))
}

func TestEmissionRejectsMintBeyondEpochReward(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	anchor := sample.AccAddress()
	author := sample.AccAddress()
	setJurors(t, f, anchor)
	seedAccount(t, f, anchor, 0)
	seedAccount(t, f, author, 0)
	setEmission(t, f, testEmission(100, 10, 3, 100))

	_, err := ms.AwardContribution(setBlockTime(f.ctx, 110), contributionAwardMsg(anchor, author, 4))
	require.ErrorContains(t, err, "mint exceeds remaining epoch reward")
	require.Equal(t, uint64(0), balanceOf(t, f, author))
}

func TestEmissionFaucetCannotBypassEpochCap(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	agent := sample.AccAddress()
	seedAccount(t, f, agent, 0)
	setEmission(t, f, testEmission(types.FaucetAmount*2, 1000, types.FaucetAmount-1, 100))

	_, err := ms.Faucet(setBlockTime(f.ctx, 110), &types.MsgFaucet{Creator: agent})
	require.ErrorContains(t, err, "mint exceeds remaining epoch reward")
	acct, err := f.keeper.Account.Get(f.ctx, agent)
	require.NoError(t, err)
	require.False(t, acct.FaucetClaimed)
	require.Equal(t, uint64(0), acct.Balance)
}
