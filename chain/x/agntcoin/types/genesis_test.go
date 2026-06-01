package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zoltankiss/agntcoin/x/agntcoin/types"
)

func TestGenesisState_Validate(t *testing.T) {
	tests := []struct {
		desc     string
		genState *types.GenesisState
		valid    bool
	}{
		{
			desc:     "default is valid",
			genState: types.DefaultGenesis(),
			valid:    true,
		},
		{
			desc:     "valid genesis state",
			genState: &types.GenesisState{AccountMap: []types.Account{{Address: "0"}, {Address: "1"}}, VouchList: []types.Vouch{{Id: 0}, {Id: 1}}, VouchCount: 2}, valid: true,
		}, {
			desc: "duplicated account",
			genState: &types.GenesisState{
				AccountMap: []types.Account{
					{
						Address: "0",
					},
					{
						Address: "0",
					},
				},
				VouchList: []types.Vouch{{Id: 0}, {Id: 1}}, VouchCount: 2,
			}, valid: false,
		}, {
			desc: "duplicated vouch",
			genState: &types.GenesisState{
				VouchList: []types.Vouch{
					{
						Id: 0,
					},
					{
						Id: 0,
					},
				},
			},
			valid: false,
		}, {
			desc: "invalid vouch count",
			genState: &types.GenesisState{
				VouchList: []types.Vouch{
					{
						Id: 1,
					},
				},
				VouchCount: 0,
			},
			valid: false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.genState.Validate()
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
