package keeper

import (
	"context"
	"errors"
	"strconv"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/zoltankiss/agntcoin/x/agntcoin/types"
)

// DisputeEscrow lets the PAYER contest a SUBMITTED escrow — delivered work it
// judges bad. It freezes the escrow: no refund, and no auto-release after the
// deadline (the payee cannot self-release a disputed escrow). The payer may
// still release (concede); otherwise the funds stay frozen pending resolution
// by a future k-of-n jury. The deadlock — neither side gets the funds — plus
// the visibility of the dispute (buyer reputation) deters abuse by either side:
// a worker won't submit slop it can't get paid for, and a buyer won't dispute
// good work it would have to forfeit.
func (k msgServer) DisputeEscrow(ctx context.Context, msg *types.MsgDisputeEscrow) (*types.MsgDisputeEscrowResponse, error) {
	if _, err := k.addressCodec.StringToBytes(msg.Creator); err != nil {
		return nil, errorsmod.Wrap(err, "invalid creator address")
	}

	escrow, err := k.Escrow.Get(ctx, msg.Id)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil, errorsmod.Wrap(sdkerrors.ErrKeyNotFound, "escrow not found")
		}
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}
	if escrow.Status != types.EscrowStatusSubmitted {
		return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "only submitted escrows can be disputed (status=%s)", escrow.Status)
	}
	if msg.Creator != escrow.Payer {
		return nil, errorsmod.Wrap(sdkerrors.ErrUnauthorized, "only the payer may dispute")
	}
	// Verifier-bound escrows (verifier-v1) deny the payer this FREE freeze: it
	// would let a buyer costlessly stall the verifier-triggered release the
	// primitive exists to provide (and strand the attester's stake). The
	// buyer's pre-release escalation is the bonded OpenDispute; its
	// post-release protection is the bonded fraud challenge.
	if len(escrow.VerifierAddrs) > 0 {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "verifier-bound escrow: use OpenDispute (bonded) — the free freeze would stall verified release")
	}

	escrow.Status = types.EscrowStatusDisputed
	if err := k.Escrow.Set(ctx, escrow.Id, escrow); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}

	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(
		sdk.NewEvent(
			"agntcoin_escrow_disputed",
			sdk.NewAttribute("id", strconv.FormatUint(escrow.Id, 10)),
			sdk.NewAttribute("payer", escrow.Payer),
			sdk.NewAttribute("payee", escrow.Payee),
			sdk.NewAttribute("amount", strconv.FormatUint(escrow.Amount, 10)),
			sdk.NewAttribute("ref", escrow.Ref),
		),
	)

	return &types.MsgDisputeEscrowResponse{}, nil
}
