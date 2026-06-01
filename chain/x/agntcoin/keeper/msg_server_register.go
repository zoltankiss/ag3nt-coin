package keeper

import (
	"context"
	"errors"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/zoltankiss/agntcoin/x/agntcoin/types"
)

func (k msgServer) Register(ctx context.Context, msg *types.MsgRegister) (*types.MsgRegisterResponse, error) {
	if _, err := k.addressCodec.StringToBytes(msg.Creator); err != nil {
		return nil, errorsmod.Wrap(err, "invalid creator address")
	}

	existing, err := k.Account.Get(ctx, msg.Creator)
	if err != nil && !errors.Is(err, collections.ErrNotFound) {
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}
	if err == nil && existing.Registered {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "already registered")
	}

	acct := types.Account{
		Address:       msg.Creator,
		Balance:       0,
		Nonce:         0,
		Registered:    true,
		FaucetClaimed: false,
	}
	if err := k.Account.Set(ctx, msg.Creator, acct); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}
	return &types.MsgRegisterResponse{}, nil
}
