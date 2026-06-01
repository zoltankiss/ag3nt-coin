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
		&MsgLockEscrow{},
	)

	registrar.RegisterImplementations((*sdk.Msg)(nil),
		&MsgReleaseEscrow{},
	)

	registrar.RegisterImplementations((*sdk.Msg)(nil),
		&MsgRefundEscrow{},
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
	msgservice.RegisterMsgServiceDesc(registrar, &_Msg_serviceDesc)
}
