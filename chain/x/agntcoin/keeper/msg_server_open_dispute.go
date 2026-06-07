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

// OpenDispute opens a k-of-n jury case against a submitted or disputed escrow
// whose acceptability cannot be settled by an objective test (or is contested).
// Either the payer or the payee may open it. The escrow moves to "in_jury",
// which freezes it: neither side can unilaterally release/refund — only a jury
// verdict (ResolveDispute) settles it. This is the recourse the it11 honest
// worker lacked when a non-technical buyer's broken tests blocked good work.
func (k msgServer) OpenDispute(ctx context.Context, msg *types.MsgOpenDispute) (*types.MsgOpenDisputeResponse, error) {
	if _, err := k.addressCodec.StringToBytes(msg.Creator); err != nil {
		return nil, errorsmod.Wrap(err, "invalid creator address")
	}

	escrow, err := k.Escrow.Get(ctx, msg.EscrowId)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil, errorsmod.Wrap(sdkerrors.ErrKeyNotFound, "escrow not found")
		}
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}
	if msg.Creator != escrow.Payer && msg.Creator != escrow.Payee {
		return nil, errorsmod.Wrap(sdkerrors.ErrUnauthorized, "only the escrow payer or payee may open a dispute")
	}
	sdkCtxNow := sdk.UnwrapSDKContext(ctx).BlockTime().Unix()
	// frozenStatus is what the escrow moves to while the jury sits. in_jury
	// settles the escrow money by verdict; challenged (post-release fraud
	// claim, verifier-v1) settles only the attestation/dispute bonds — the
	// payout already happened and is never clawed back from the payee.
	frozenStatus := types.EscrowStatusInJury
	switch escrow.Status {
	case types.EscrowStatusSubmitted, types.EscrowStatusDisputed:
		// the pre-settlement jury case (it13+): either side may open.
	case types.EscrowStatusFailAttested:
		// Verifier-v1 CONTEST: only the payee may dispute a staked fail
		// attestation, and only inside the contest window (after it the fail
		// stands by default and refund unlocks). A jury-accept releases the
		// escrow to the payee AND slashes the false-fail attester's stake to
		// the payee (the false-fail symmetry of the fraud challenge).
		if msg.Creator != escrow.Payee {
			return nil, errorsmod.Wrap(sdkerrors.ErrUnauthorized, "only the payee may contest a fail attestation")
		}
		if sdkCtxNow >= escrow.ChallengeDeadline {
			return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "the contest window has closed")
		}
	case types.EscrowStatusReleased:
		// Verifier-v1 post-release FRAUD CHALLENGE: only against a
		// verifier-triggered release (attestations + an open challenge
		// window), only by the payer. A payer-goodwill or jury release is
		// final — there is no attester collateral behind it to claim from.
		if msg.Creator != escrow.Payer {
			return nil, errorsmod.Wrap(sdkerrors.ErrUnauthorized, "only the payer may challenge a verified release")
		}
		if len(escrow.Attestations) == 0 || escrow.ChallengeDeadline == 0 {
			return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "released escrow has no challengeable attestations")
		}
		if sdkCtxNow >= escrow.ChallengeDeadline {
			return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "the challenge window has closed")
		}
		frozenStatus = types.EscrowStatusChallenged
	default:
		return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "escrow is not disputable (status=%s)", escrow.Status)
	}

	// Opening a dispute requires a slashable DISPUTE-BOND (it18 RED → it19): a free
	// dispute is weaponizable (cry-wolf to stall payment; jury-DDoS to bury real
	// cases), so the opener must post collateral that a lost/frivolous verdict
	// slashes to the griefed counterparty (and an upheld one returns). The opener
	// must be registered with sufficient balance; the bond is debited now and
	// settled in ResolveDispute.
	if msg.BondAmount < types.MinDisputeBond {
		return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "opening a dispute requires a bond of at least %d", types.MinDisputeBond)
	}
	opener, err := k.Account.Get(ctx, msg.Creator)
	if err != nil || !opener.Registered {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "opener not registered")
	}
	if opener.Balance < msg.BondAmount {
		return nil, errorsmod.Wrap(sdkerrors.ErrInsufficientFunds, "insufficient balance for dispute bond")
	}

	id, err := k.DisputeSeq.Next(ctx)
	if err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}

	// Debit the opener + lock the dispute-bond (protocol-held; slasher empty so
	// only ResolveDispute can settle it — never an external SlashBond/ReleaseBond).
	opener.Balance -= msg.BondAmount
	if err := k.Account.Set(ctx, msg.Creator, opener); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}
	bondID, err := k.BondSeq.Next(ctx)
	if err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}
	bond := types.Bond{
		Id:      bondID,
		Poster:  msg.Creator,
		Amount:  msg.BondAmount,
		Purpose: "dispute:" + strconv.FormatUint(id, 10),
		Slasher: "", // protocol-settled by ResolveDispute; no external slasher
		Status:  types.BondStatusActive,
		Ref:     strconv.FormatUint(msg.EscrowId, 10), // verifiable escrow ref (it17 #22)
	}
	if err := k.Bond.Set(ctx, bondID, bond); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}

	dispute := types.Dispute{
		Id:       id,
		EscrowId: msg.EscrowId,
		Opener:   msg.Creator,
		Reason:   msg.Reason,
		Status:   types.DisputeStatusOpen,
		BondId:   bondID,
	}
	if err := k.Dispute.Set(ctx, id, dispute); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}

	// Freeze the escrow under jury control: only ResolveDispute can settle it now.
	escrow.Status = frozenStatus
	if err := k.Escrow.Set(ctx, escrow.Id, escrow); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent("agntcoin_dispute_opened",
			sdk.NewAttribute("id", strconv.FormatUint(id, 10)),
			sdk.NewAttribute("escrow_id", strconv.FormatUint(msg.EscrowId, 10)),
			sdk.NewAttribute("opener", msg.Creator),
		),
	)
	return &types.MsgOpenDisputeResponse{Id: id}, nil
}
