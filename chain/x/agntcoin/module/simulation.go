package agntcoin

import (
	"math/rand"

	"github.com/cosmos/cosmos-sdk/types/module"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"

	agntcoinsimulation "github.com/zoltankiss/agntcoin/x/agntcoin/simulation"
	"github.com/zoltankiss/agntcoin/x/agntcoin/types"
)

// GenerateGenesisState creates a randomized GenState of the module.
func (AppModule) GenerateGenesisState(simState *module.SimulationState) {
	accs := make([]string, len(simState.Accounts))
	for i, acc := range simState.Accounts {
		accs[i] = acc.Address.String()
	}
	agntcoinGenesis := types.GenesisState{
		Params: types.DefaultParams(),
	}
	simState.GenState[types.ModuleName] = simState.Cdc.MustMarshalJSON(&agntcoinGenesis)
}

// RegisterStoreDecoder registers a decoder.
func (am AppModule) RegisterStoreDecoder(_ simtypes.StoreDecoderRegistry) {}

// WeightedOperations returns the all the gov module operations with their respective weights.
func (am AppModule) WeightedOperations(simState module.SimulationState) []simtypes.WeightedOperation {
	operations := make([]simtypes.WeightedOperation, 0)
	const (
		opWeightMsgRegister          = "op_weight_msg_agntcoin"
		defaultWeightMsgRegister int = 100
	)

	var weightMsgRegister int
	simState.AppParams.GetOrGenerate(opWeightMsgRegister, &weightMsgRegister, nil,
		func(_ *rand.Rand) {
			weightMsgRegister = defaultWeightMsgRegister
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgRegister,
		agntcoinsimulation.SimulateMsgRegister(am.authKeeper, am.bankKeeper, am.keeper, simState.TxConfig),
	))
	const (
		opWeightMsgFaucet          = "op_weight_msg_agntcoin"
		defaultWeightMsgFaucet int = 100
	)

	var weightMsgFaucet int
	simState.AppParams.GetOrGenerate(opWeightMsgFaucet, &weightMsgFaucet, nil,
		func(_ *rand.Rand) {
			weightMsgFaucet = defaultWeightMsgFaucet
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgFaucet,
		agntcoinsimulation.SimulateMsgFaucet(am.authKeeper, am.bankKeeper, am.keeper, simState.TxConfig),
	))
	const (
		opWeightMsgTransfer          = "op_weight_msg_agntcoin"
		defaultWeightMsgTransfer int = 100
	)

	var weightMsgTransfer int
	simState.AppParams.GetOrGenerate(opWeightMsgTransfer, &weightMsgTransfer, nil,
		func(_ *rand.Rand) {
			weightMsgTransfer = defaultWeightMsgTransfer
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgTransfer,
		agntcoinsimulation.SimulateMsgTransfer(am.authKeeper, am.bankKeeper, am.keeper, simState.TxConfig),
	))
	const (
		opWeightMsgCastvouch          = "op_weight_msg_agntcoin"
		defaultWeightMsgCastvouch int = 100
	)

	var weightMsgCastvouch int
	simState.AppParams.GetOrGenerate(opWeightMsgCastvouch, &weightMsgCastvouch, nil,
		func(_ *rand.Rand) {
			weightMsgCastvouch = defaultWeightMsgCastvouch
		},
	)
	operations = append(operations, simulation.NewWeightedOperation(
		weightMsgCastvouch,
		agntcoinsimulation.SimulateMsgCastvouch(am.authKeeper, am.bankKeeper, am.keeper, simState.TxConfig),
	))

	return operations
}

// ProposalMsgs returns msgs used for governance proposals for simulations.
func (am AppModule) ProposalMsgs(simState module.SimulationState) []simtypes.WeightedProposalMsg {
	return []simtypes.WeightedProposalMsg{}
}
