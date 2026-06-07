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

// RefundEscrow returns a locked escrow's funds to the payer. It is only
// allowed for the payer and only within the dispute window (before the
// deadline) — this is the buyer's cancel/dispute window.
func (k msgServer) RefundEscrow(ctx context.Context, msg *types.MsgRefundEscrow) (*types.MsgRefundEscrowResponse, error) {
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
	if escrow.Status != types.EscrowStatusLocked && escrow.Status != types.EscrowStatusFailAttested {
		return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "escrow is not refundable (status=%s)", escrow.Status)
	}

	if msg.Creator != escrow.Payer {
		return nil, errorsmod.Wrap(sdkerrors.ErrUnauthorized, "only the payer may refund")
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	now := sdkCtx.BlockTime().Unix()
	if escrow.Status == types.EscrowStatusFailAttested {
		// Verifier-v1 fail path: a staked fail attestation justifies a refund
		// of submitted work, but ONLY after the payee's contest window passes
		// uncontested — instant refund-on-fail would let a colluding
		// buyer+verifier take delivered work for free (the false-fail
		// symmetry hole). The payee's recourse during the window is
		// OpenDispute: a jury that finds the work good releases the escrow
		// and slashes the false-fail attester's stake to the payee. The
		// original pre-deadline refund-window rule does not apply here — the
		// fail attestation is the verifier's signed justification, and the
		// contest window is the timing guard.
		if now < escrow.ChallengeDeadline {
			return nil, errorsmod.Wrap(sdkerrors.ErrUnauthorized, "fail-attested escrow: the payee's contest window is still open")
		}
	} else if now >= escrow.Deadline {
		return nil, errorsmod.Wrap(sdkerrors.ErrUnauthorized, "refund window has closed")
	}

	payerBytes, err := k.addressCodec.StringToBytes(escrow.Payer)
	if err != nil {
		return nil, errorsmod.Wrap(err, "invalid payer address")
	}
	// Credit the amount back to the payer. The payer already exists (it locked
	// the escrow), so the spam guard is a no-op here.
	if err := k.creditAccount(ctx, escrow.Payer, payerBytes, escrow.Amount); err != nil {
		return nil, err
	}

	escrow.Status = types.EscrowStatusRefunded
	if err := k.Escrow.Set(ctx, escrow.Id, escrow); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}

	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"agntcoin_escrow_refunded",
			sdk.NewAttribute("id", strconv.FormatUint(escrow.Id, 10)),
			sdk.NewAttribute("payer", escrow.Payer),
			sdk.NewAttribute("payee", escrow.Payee),
			sdk.NewAttribute("amount", strconv.FormatUint(escrow.Amount, 10)),
			sdk.NewAttribute("ref", escrow.Ref),
		),
	)

	return &types.MsgRefundEscrowResponse{}, nil
}
