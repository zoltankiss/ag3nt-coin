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
				{
					RpcMethod: "ListDispute",
					Use:       "list-dispute",
					Short:     "List all jury disputes",
				},
				{
					RpcMethod:      "GetDispute",
					Use:            "get-dispute [id]",
					Short:          "Gets a jury dispute by id",
					Alias:          []string{"show-dispute"},
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "id"}},
				},
				{
					RpcMethod: "ListBond",
					Use:       "list-bond",
					Short:     "List all bonds (slashable collateral)",
				},
				{
					RpcMethod:      "GetBond",
					Use:            "get-bond [id]",
					Short:          "Gets a bond by id",
					Alias:          []string{"show-bond"},
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "id"}},
				},
				{
					RpcMethod:      "GetContributionAward",
					Use:            "get-contribution-award [id]",
					Short:          "Gets an accepted protocol contribution award by id",
					Alias:          []string{"show-contribution-award"},
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "id"}},
				},
				{
					RpcMethod: "ListContributionAward",
					Use:       "list-contribution-award",
					Short:     "List accepted protocol contribution awards",
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
					Use:            "castvouch [to-addr] [weight] [stake]",
					Short:          "Send a castvouch tx (locks stake behind the vouch)",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "to_addr"}, {ProtoField: "weight"}, {ProtoField: "stake"}},
				},
				{
					RpcMethod:      "Unvouch",
					Use:            "unvouch [to-addr]",
					Short:          "Remove your vouch for an address and reclaim the locked stake",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "to_addr"}},
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
				{
					RpcMethod:      "SubmitEscrow",
					Use:            "submit-escrow [id]",
					Short:          "Payee: mark an escrow delivered (blocks refund)",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "id"}},
				},
				{
					RpcMethod:      "DisputeEscrow",
					Use:            "dispute-escrow [id]",
					Short:          "Payer: contest submitted work (freeze, no auto-release)",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "id"}},
				},
				{
					RpcMethod:      "PostBond",
					Use:            "post-bond [amount] [purpose] [slasher] [ref]",
					Short:          "Lock slashable collateral behind a claim (only the slasher can release/slash it)",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "amount"}, {ProtoField: "purpose"}, {ProtoField: "slasher"}, {ProtoField: "ref"}},
				},
				{
					RpcMethod:      "ReleaseBond",
					Use:            "release-bond [id]",
					Short:          "Slasher: exonerate a bond (refund the poster)",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "id"}},
				},
				{
					RpcMethod:      "SlashBond",
					Use:            "slash-bond [id] [beneficiary]",
					Short:          "Slasher: punish a bond (collateral to beneficiary, or burn if empty)",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "id"}, {ProtoField: "beneficiary"}},
				},
				{
					RpcMethod:      "PostGate",
					Use:            "post-gate [payload-uri] [payload-hash] [gold-commit] [drip] [max-answers]",
					Short:          "Anchor: post a protocol PR-review gate",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "payload_uri"}, {ProtoField: "payload_hash"}, {ProtoField: "gold_commit"}, {ProtoField: "drip"}, {ProtoField: "max_answers"}},
				},
				{
					RpcMethod:      "CommitAnswer",
					Use:            "commit-answer [gate-id] [commit]",
					Short:          "Commit a hashed gate answer",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "gate_id"}, {ProtoField: "commit"}},
				},
				{
					RpcMethod:      "RevealAnswer",
					Use:            "reveal-answer [gate-id] [answer] [salt]",
					Short:          "Reveal a committed gate answer",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "gate_id"}, {ProtoField: "answer"}, {ProtoField: "salt"}},
				},
				{
					RpcMethod:      "SettleGate",
					Use:            "settle-gate [gate-id] [gold-answer] [gold-salt]",
					Short:          "Settle a gate and mint drip to coherent answers",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "gate_id"}, {ProtoField: "gold_answer"}, {ProtoField: "gold_salt"}},
				},
				{
					RpcMethod:      "AwardContribution",
					Use:            "award-contribution [recipient] [repo-url] [pr-url] [commit-sha] [artifact-uri] [artifact-sha256] [evidence-sha256] [scope] [rationale-hash] [amount]",
					Short:          "Anchor: mint capped AGNT to an accepted protocol contribution author",
					PositionalArgs: []*autocliv1.PositionalArgDescriptor{{ProtoField: "recipient"}, {ProtoField: "repo_url"}, {ProtoField: "pr_url"}, {ProtoField: "commit_sha"}, {ProtoField: "artifact_uri"}, {ProtoField: "artifact_sha256"}, {ProtoField: "evidence_sha256"}, {ProtoField: "scope"}, {ProtoField: "rationale_hash"}, {ProtoField: "amount"}},
				},
			},
		},
	}
}
