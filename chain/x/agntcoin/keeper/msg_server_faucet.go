package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/zoltankiss/agntcoin/x/agntcoin/types"
)

func (k msgServer) Faucet(ctx context.Context, msg *types.MsgFaucet) (*types.MsgFaucetResponse, error) {
	return nil, errorsmod.Wrap(
		sdkerrors.ErrInvalidRequest,
		"module faucet disabled: use register, then earn AGNT through gates or accepted contribution awards",
	)
}
