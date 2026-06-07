package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/zoltankiss/agntcoin/x/agntcoin/types"
)

func (k msgServer) Faucet(ctx context.Context, msg *types.MsgFaucet) (*types.MsgFaucetResponse, error) {
	creatorBytes, err := k.addressCodec.StringToBytes(msg.Creator)
	if err != nil {
		return nil, errorsmod.Wrap(err, "invalid creator address")
	}

	acct, err := k.Account.Get(ctx, msg.Creator)
	if err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "not registered")
	}
	if !acct.Registered {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "not registered")
	}
	if acct.FaucetClaimed {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "faucet already claimed")
	}

	if err := k.mintProtocolReward(ctx, msg.Creator, creatorBytes, types.FaucetAmount, "faucet"); err != nil {
		return nil, err
	}
	acct, err = k.Account.Get(ctx, msg.Creator)
	if err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}
	acct.FaucetClaimed = true
	if err := k.Account.Set(ctx, msg.Creator, acct); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}
	return &types.MsgFaucetResponse{}, nil
}
