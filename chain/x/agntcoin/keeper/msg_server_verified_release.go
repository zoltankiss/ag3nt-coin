package keeper

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/zoltankiss/agntcoin/x/agntcoin/types"
)

// VerifiedRelease settles a verifier-attested escrow WITHOUT the payer
// (verifier-v1): the third release authority after "payer" and "jury". Anyone
// may submit it — the staked attestations recorded on the escrow are the
// authority, not the submitter (same philosophy as anyone-can-release after
// the deadline on ordinary escrows).
//
// Requirements: a quorum of distinct pass-attestations, and — if the payee
// committed a key_hash at submit — the key preimage. The payee is paid and
// the key is emitted in the release event in ONE tx: the buyer can only ever
// read the plaintext in the same state transition that pays the seller
// (fair exchange enforced by the chain, not by platform behavior).
//
// Release is OPTIMISTIC and instant — the seller never waits on the buyer.
// The buyer's protection is the challenge window this opens: until
// escrow.challenge_deadline the payer may OpenDispute a fraud claim, and a
// jury-confirmed fraud is recovered from the attesters' stakes (each >= the
// bounty, so the buyer is always made whole). The payout itself is never
// clawed back from the payee — the verifier, the only party who can cause the
// bad outcome, is the one whose capital is at risk.
func (k msgServer) VerifiedRelease(ctx context.Context, msg *types.MsgVerifiedRelease) (*types.MsgVerifiedReleaseResponse, error) {
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
	if len(escrow.VerifierAddrs) == 0 {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "escrow is not verifier-bound")
	}
	if escrow.Status != types.EscrowStatusSubmitted {
		return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "escrow is not submitted (status=%s)", escrow.Status)
	}

	// Quorum: count distinct pass verdicts from the named set. AttestEscrow
	// already enforced membership, uniqueness, per-attestation stake >=
	// amount, AND that the verdict restates the escrow's commitments — the
	// match is re-checked here (belt-and-braces) so that even if a future
	// code path ever stored an attestation whose pins drifted from the
	// escrow's delivery_hash/key_hash/acceptance_hash, it could never count
	// as a release authority.
	var passed uint64
	for _, a := range escrow.Attestations {
		if a.Passed &&
			strings.EqualFold(a.DeliveryHash, escrow.DeliveryHash) &&
			strings.EqualFold(a.KeyHash, escrow.KeyHash) &&
			strings.EqualFold(a.AcceptanceHash, escrow.AcceptanceHash) {
			passed++
		}
	}
	if passed < escrow.VerifierQuorum {
		return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "verifier quorum not met: %d of %d required pass attestations", passed, escrow.VerifierQuorum)
	}

	// Fair exchange: the key preimage must match the payee's submit-time
	// commitment. If no key was committed, none may be presented (an
	// uncommitted "reveal" would be theater).
	if escrow.KeyHash != "" {
		sum := sha256.Sum256([]byte(msg.Key))
		if !strings.EqualFold(hex.EncodeToString(sum[:]), escrow.KeyHash) {
			return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "key does not match the committed key_hash")
		}
	} else if msg.Key != "" {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "no key_hash was committed for this escrow")
	}

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

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	escrow.Status = types.EscrowStatusReleased
	// Evidence timeline: when settlement happened, by block time.
	escrow.ReleasedAt = sdkCtx.BlockTime().Unix()
	// Open the payer's fraud-challenge window (block time, never a
	// party-supplied timestamp). The attestation stakes stay locked until it
	// passes uncontested (ReclaimAttestationBonds) or a jury settles them.
	escrow.ChallengeDeadline = sdkCtx.BlockTime().Unix() + types.ChallengeWindowSeconds
	if err := k.Escrow.Set(ctx, escrow.Id, escrow); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}

	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent("agntcoin_escrow_verified_release",
			sdk.NewAttribute("id", strconv.FormatUint(escrow.Id, 10)),
			sdk.NewAttribute("payer", escrow.Payer),
			sdk.NewAttribute("payee", escrow.Payee),
			sdk.NewAttribute("amount", strconv.FormatUint(escrow.Amount, 10)),
			sdk.NewAttribute("ref", escrow.Ref),
			// THE atomic reveal: the decryption key becomes public in the same
			// tx that pays the seller. The buyer reads it here.
			sdk.NewAttribute("key", msg.Key),
			sdk.NewAttribute("pass_attestations", strconv.FormatUint(passed, 10)),
			sdk.NewAttribute("challenge_deadline", strconv.FormatInt(escrow.ChallengeDeadline, 10)),
		),
	)
	return &types.MsgVerifiedReleaseResponse{}, nil
}
