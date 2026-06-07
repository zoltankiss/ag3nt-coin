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

// ReclaimAttestationBonds frees the attestation-stakes of an escrow whose
// adversarial window is over (verifier-v1). Permissionless trigger — the
// module has no end-blocker sweep, so unlocking collateral is a tx, same as
// ResolveDispute. Allowed when the escrow is in a settled/attested terminal
// state AND either no challenge window was ever opened (e.g. the payer
// released directly, conceding — no fraud claim is possible against a payout
// the payer authorized) or the window has passed uncontested. An escrow under
// jury control (in_jury / challenged) settles its stakes through the verdict
// instead — its status blocks this path until ResolveDispute restores a
// terminal one, by which point the stakes are already settled (non-active)
// and this becomes a no-op.
func (k msgServer) ReclaimAttestationBonds(ctx context.Context, msg *types.MsgReclaimAttestationBonds) (*types.MsgReclaimAttestationBondsResponse, error) {
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
	if len(escrow.Attestations) == 0 {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "escrow has no attestations")
	}
	switch escrow.Status {
	case types.EscrowStatusReleased, types.EscrowStatusRefunded, types.EscrowStatusFailAttested:
		// settled, or fail-attested with the payer yet to refund — the
		// verifier's stake must not depend on the buyer ever acting.
	default:
		return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "escrow's attestation stakes are not reclaimable (status=%s)", escrow.Status)
	}
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	if escrow.ChallengeDeadline > 0 && sdkCtx.BlockTime().Unix() < escrow.ChallengeDeadline {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "challenge window is still open")
	}

	purpose := "attestation:" + strconv.FormatUint(escrow.Id, 10)
	for _, a := range escrow.Attestations {
		bond, err := k.Bond.Get(ctx, a.BondId)
		if err != nil || bond.Status != types.BondStatusActive || bond.Purpose != purpose {
			continue // already settled by a jury verdict, or not ours
		}
		acct, err := k.Account.Get(ctx, bond.Poster)
		if err != nil {
			return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
		}
		acct.Balance += bond.Amount
		if err := k.Account.Set(ctx, bond.Poster, acct); err != nil {
			return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
		}
		bond.Status = types.BondStatusReleased
		if err := k.Bond.Set(ctx, bond.Id, bond); err != nil {
			return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
		}
		sdkCtx.EventManager().EmitEvent(
			sdk.NewEvent("agntcoin_attestation_bond_reclaimed",
				sdk.NewAttribute("escrow_id", strconv.FormatUint(escrow.Id, 10)),
				sdk.NewAttribute("verifier", a.Verifier),
				sdk.NewAttribute("bond_id", strconv.FormatUint(bond.Id, 10)),
				sdk.NewAttribute("amount", strconv.FormatUint(bond.Amount, 10)),
			),
		)
	}
	return &types.MsgReclaimAttestationBondsResponse{}, nil
}
