package agntcoin

import (
	autocliv1 "cosmossdk.io/api/cosmos/autocli/v1"

	"github.com/zoltankiss/agntcoin/x/agntcoin/types"
)

// AutoCLIOptions implements the autocli.HasAutoCLIConfig interface.
func (am AppModule) AutoCLIOptions() *autocliv1.ModuleOptions {
	return &autocliv1.ModuleOptions{
		Query: &autocliv1.ServiceCommandDescriptor{
			Service: types.Query_serviceDesc.ServiceName,
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "Params",
					Use:       "params",
					Short:     "Shows the parameters of the module",
				},
				{
					RpcMethod: "ListAccount",
					Use:       "list-account",
					Short:     "List all account",
				},
				{
					RpcMethod:      "GetAccount",
					Use:            "get-account [id]",
					Short:          "Gets a account",
					Alias:          []string{"show-account"},
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "address"}},
				},
				{
					RpcMethod: "ListVouch",
					Use:       "list-vouch",
					Short:     "List all vouch",
				},
				{
					RpcMethod:      "GetVouch",
					Use:            "get-vouch [id]",
					Short:          "Gets a vouch by id",
					Alias:          []string{"show-vouch"},
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "id"}},
				},
				{
					RpcMethod:      "Reputation",
					Use:            "reputation [address]",
					Short:          "Query reputation",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "address"}},
				},
				{
					RpcMethod: "ListEscrow",
					Use:       "list-escrow",
					Short:     "List all escrow",
				},
				{
					RpcMethod:      "GetEscrow",
					Use:            "get-escrow [id]",
					Short:          "Gets an escrow by id",
					Alias:          []string{"show-escrow"},
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "id"}},
				},
			},
		},
		Tx: &autocliv1.ServiceCommandDescriptor{
			Service:              types.Msg_serviceDesc.ServiceName,
			EnhanceCustomCommand: true, // only required if you want to use the custom command
			RpcCommandOptions: []*autocliv1.RpcCommandOptions{
				{
					RpcMethod: "UpdateParams",
					Skip:      true, // skipped because authority gated
				},
				{
					RpcMethod:      "Register",
					Use:            "register ",
					Short:          "Send a register tx",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{},
				},
				{
					RpcMethod:      "Faucet",
					Use:            "faucet ",
					Short:          "Send a faucet tx",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{},
				},
				{
					RpcMethod:      "Transfer",
					Use:            "transfer [to-addr] [amount]",
					Short:          "Send a transfer tx",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "to_addr"}, {ProtoField: "amount"}},
				},
				{
					RpcMethod:      "Castvouch",
					Use:            "castvouch [to-addr] [weight]",
					Short:          "Send a castvouch tx",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "to_addr"}, {ProtoField: "weight"}},
				},
				{
					RpcMethod:      "LockEscrow",
					Use:            "lock-escrow [payee] [amount] [ref] [dispute-seconds]",
					Short:          "Lock funds into a new escrow",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "payee"}, {ProtoField: "amount"}, {ProtoField: "ref"}, {ProtoField: "dispute_seconds"}},
				},
				{
					RpcMethod:      "ReleaseEscrow",
					Use:            "release-escrow [id]",
					Short:          "Release an escrow to the payee",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "id"}},
				},
				{
					RpcMethod:      "RefundEscrow",
					Use:            "refund-escrow [id]",
					Short:          "Refund an escrow to the payer",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "id"}},
				},
			},
		},
	}
}
