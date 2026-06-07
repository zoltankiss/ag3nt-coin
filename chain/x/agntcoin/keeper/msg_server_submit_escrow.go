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

// SubmitEscrow lets the PAYEE mark a locked escrow as delivered. Once submitted,
// the payer can no longer refund it (RefundEscrow only accepts "locked") — only
// release remains, or the payee self-releases after the deadline. This is the
// fair-exchange guarantee for the worker: it can reveal delivered work without
// the buyer clawing the payment back. The buyer's recourse against bad work is
// DisputeEscrow, not refund.
func (k msgServer) SubmitEscrow(ctx context.Context, msg *types.MsgSubmitEscrow) (*types.MsgSubmitEscrowResponse, error) {
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
	if escrow.Status != types.EscrowStatusLocked {
		return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "escrow is not locked (status=%s)", escrow.Status)
	}
	if msg.Creator != escrow.Payee {
		return nil, errorsmod.Wrap(sdkerrors.ErrUnauthorized, "only the payee may submit")
	}

	// A verifier-bound delivery MUST commit the fair exchange — both the
	// artifact bytes (delivery_hash) and the decryption key (key_hash).
	// Without them the primitive silently degrades to custodial trust, and
	// worse: a seller + truthful verifier could collect via VerifiedRelease
	// while withholding the artifact from the buyer — the attestation would
	// be TRUE, so the fraud challenge could never make the buyer whole. With
	// them, the only tx that pays the seller is the tx that hands the buyer
	// the key. Ordinary escrows keep both optional, but any supplied hash
	// must be re-checkable (hex sha256) or it is theater, not evidence.
	verifierBound := len(escrow.VerifierAddrs) > 0
	if verifierBound && (!isHexSHA256(msg.DeliveryHash) || !isHexSHA256(msg.KeyHash)) {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "verifier-bound submit requires delivery_hash and key_hash (hex sha256)")
	}
	if msg.DeliveryHash != "" && !isHexSHA256(msg.DeliveryHash) {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "delivery_hash must be a hex sha256")
	}
	if msg.KeyHash != "" && !isHexSHA256(msg.KeyHash) {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "key_hash must be a hex sha256")
	}

	escrow.Status = types.EscrowStatusSubmitted
	// Pin the delivered artifact on-chain (hex sha256) so the exhibit the jury
	// later judges is tamper-evident: anyone can re-hash what the serving layer
	// presents and verify it against this commitment. Set once, at submit.
	escrow.DeliveryHash = msg.DeliveryHash
	// Pin the artifact's decryption key (hex sha256 of the key, verifier-v1
	// fair exchange): the payee publishes the CIPHERTEXT anywhere and commits
	// h(key) here; VerifiedRelease must present the preimage and emits it in
	// the release event — so payment and key reveal are one atomic tx. Set
	// once, at submit.
	escrow.KeyHash = msg.KeyHash
	// Evidence availability: the canonical, content-addressable place to fetch
	// the ciphertext from — jurors re-hash bytes from HERE against
	// delivery_hash, never trusting a mutable app DB row. The chain can't
	// guarantee availability, only pin the reference beside the hash.
	escrow.ArtifactUri = msg.ArtifactUri
	// Evidence timeline: block time at tx inclusion, never a party's clock.
	escrow.SubmittedAt = sdk.UnwrapSDKContext(ctx).BlockTime().Unix()
	if err := k.Escrow.Set(ctx, escrow.Id, escrow); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}

	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(
		sdk.NewEvent(
			"agntcoin_escrow_submitted",
			sdk.NewAttribute("id", strconv.FormatUint(escrow.Id, 10)),
			sdk.NewAttribute("payer", escrow.Payer),
			sdk.NewAttribute("payee", escrow.Payee),
			sdk.NewAttribute("amount", strconv.FormatUint(escrow.Amount, 10)),
			sdk.NewAttribute("ref", escrow.Ref),
			sdk.NewAttribute("delivery_hash", escrow.DeliveryHash),
		),
	)

	return &types.MsgSubmitEscrowResponse{}, nil
}
