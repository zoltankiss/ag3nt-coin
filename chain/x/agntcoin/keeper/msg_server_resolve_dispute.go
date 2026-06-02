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

// ResolveDispute tallies an open dispute's juror votes by COUNT majority and
// settles the underlying escrow: accept (more accept than reject) releases the
// escrow to the payee; reject refunds the payer. Permissionless trigger —
// anyone may resolve once a strict majority exists. (v0: count majority among
// anchor jurors. Stake-weighting, an explicit quorum, juror fees, and slashing
// of incoherent/no-show jurors are v1 — they only matter under adversarial
// pressure, which is validated post-CVP.)
func (k msgServer) ResolveDispute(ctx context.Context, msg *types.MsgResolveDispute) (*types.MsgResolveDisputeResponse, error) {
	if _, err := k.addressCodec.StringToBytes(msg.Creator); err != nil {
		return nil, errorsmod.Wrap(err, "invalid creator address")
	}

	dispute, err := k.Dispute.Get(ctx, msg.DisputeId)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil, errorsmod.Wrap(sdkerrors.ErrKeyNotFound, "dispute not found")
		}
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}
	if dispute.Status != types.DisputeStatusOpen {
		return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "dispute is not open (status=%s)", dispute.Status)
	}

	var accept, reject int
	for _, v := range dispute.Votes {
		if v.Accept {
			accept++
		} else {
			reject++
		}
	}
	if accept == reject {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "no majority yet — need a strict accept/reject majority to resolve")
	}

	escrow, err := k.Escrow.Get(ctx, dispute.EscrowId)
	if err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}
	if escrow.Status != types.EscrowStatusInJury {
		return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "escrow is not under jury control (status=%s)", escrow.Status)
	}

	var resolution string
	if accept > reject {
		// Accept: release the escrow to the payee (auto-registering a new payee).
		payeeBytes, err := k.addressCodec.StringToBytes(escrow.Payee)
		if err != nil {
			return nil, errorsmod.Wrap(err, "invalid payee address")
		}
		if err := k.checkNewAccountCredit(ctx, escrow.Payee, escrow.Amount); err != nil {
			return nil, err
		}
		if err := k.creditAccount(ctx, escrow.Payee, payeeBytes, escrow.Amount); err != nil {
			return nil, err
		}
		escrow.Status = types.EscrowStatusReleased
		resolution = types.DisputeResolutionAccept
	} else {
		// Reject: refund the escrow to the payer (the payer already exists).
		payerBytes, err := k.addressCodec.StringToBytes(escrow.Payer)
		if err != nil {
			return nil, errorsmod.Wrap(err, "invalid payer address")
		}
		if err := k.creditAccount(ctx, escrow.Payer, payerBytes, escrow.Amount); err != nil {
			return nil, err
		}
		escrow.Status = types.EscrowStatusRefunded
		resolution = types.DisputeResolutionReject
	}
	if err := k.Escrow.Set(ctx, escrow.Id, escrow); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}

	dispute.Status = types.DisputeStatusResolved
	dispute.Resolution = resolution
	if err := k.Dispute.Set(ctx, dispute.Id, dispute); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent("agntcoin_dispute_resolved",
			sdk.NewAttribute("id", strconv.FormatUint(dispute.Id, 10)),
			sdk.NewAttribute("escrow_id", strconv.FormatUint(escrow.Id, 10)),
			sdk.NewAttribute("resolution", resolution),
			sdk.NewAttribute("accept_votes", strconv.Itoa(accept)),
			sdk.NewAttribute("reject_votes", strconv.Itoa(reject)),
		),
	)
	return &types.MsgResolveDisputeResponse{Resolution: resolution}, nil
}
