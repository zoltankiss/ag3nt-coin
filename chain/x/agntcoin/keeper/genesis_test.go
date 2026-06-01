package keeper_test

import (
	"testing"

	"github.com/zoltankiss/agntcoin/x/agntcoin/types"

	"github.com/stretchr/testify/require"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params:     types.DefaultParams(),
		AccountMap: []types.Account{{Address: "0"}, {Address: "1"}}, VouchList: []types.Vouch{{Id: 0}, {Id: 1}},
		VouchCount: 2,
	}
	f := initFixture(t)
	err := f.keeper.InitGenesis(f.ctx, genesisState)
	require.NoError(t, err)
	got, err := f.keeper.ExportGenesis(f.ctx)
	require.NoError(t, err)
	require.NotNil(t, got)

	require.EqualExportedValues(t, genesisState.Params, got.Params)
	require.EqualExportedValues(t, genesisState.AccountMap, got.AccountMap)
	require.EqualExportedValues(t, genesisState.VouchList, got.VouchList)
	require.Equal(t, genesisState.VouchCount, got.VouchCount)

}
