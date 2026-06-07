package types

import (
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

func RegisterInterfaces(registrar codectypes.InterfaceRegistry) {
	registrar.RegisterImplementations((*sdk.Msg)(nil),
		&MsgCastvouch{},
	)

	registrar.RegisterImplementations((*sdk.Msg)(nil),
		&MsgUnvouch{},
	)

	registrar.RegisterImplementations((*sdk.Msg)(nil),
		&MsgLockEscrow{},
	)

	registrar.RegisterImplementations((*sdk.Msg)(nil),
		&MsgReleaseEscrow{},
	)

	registrar.RegisterImplementations((*sdk.Msg)(nil),
		&MsgRefundEscrow{},
	)

	registrar.RegisterImplementations((*sdk.Msg)(nil),
		&MsgSubmitEscrow{},
	)

	registrar.RegisterImplementations((*sdk.Msg)(nil),
		&MsgDisputeEscrow{},
	)

	registrar.RegisterImplementations((*sdk.Msg)(nil),
		&MsgTransfer{},
	)

	registrar.RegisterImplementations((*sdk.Msg)(nil),
		&MsgFaucet{},
	)

	registrar.RegisterImplementations((*sdk.Msg)(nil),
		&MsgRegister{},
	)

	registrar.RegisterImplementations((*sdk.Msg)(nil),
		&MsgUpdateParams{},
	)

	registrar.RegisterImplementations((*sdk.Msg)(nil),
		&MsgOpenDispute{},
	)

	registrar.RegisterImplementations((*sdk.Msg)(nil),
		&MsgCastVote{},
	)

	registrar.RegisterImplementations((*sdk.Msg)(nil),
		&MsgResolveDispute{},
	)

	registrar.RegisterImplementations((*sdk.Msg)(nil),
		&MsgPostBond{},
	)

	registrar.RegisterImplementations((*sdk.Msg)(nil),
		&MsgReleaseBond{},
	)

	registrar.RegisterImplementations((*sdk.Msg)(nil),
		&MsgSlashBond{},
	)

	registrar.RegisterImplementations((*sdk.Msg)(nil),
		&MsgAttestEscrow{},
	)

	registrar.RegisterImplementations((*sdk.Msg)(nil),
		&MsgVerifiedRelease{},
	)

	registrar.RegisterImplementations((*sdk.Msg)(nil),
		&MsgReclaimAttestationBonds{},
	)
	msgservice.RegisterMsgServiceDesc(registrar, &_Msg_serviceDesc)
}
