package keeper

import (
	"context"
	"strconv"
	"strings"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/zoltankiss/agntcoin/x/agntcoin/types"
)

func (k msgServer) CastScopedEvidenceVouch(ctx context.Context, msg *types.MsgCastScopedEvidenceVouch) (*types.MsgCastScopedEvidenceVouchResponse, error) {
	if _, err := k.addressCodec.StringToBytes(msg.Creator); err != nil {
		return nil, errorsmod.Wrap(err, "invalid creator address")
	}
	if _, err := k.addressCodec.StringToBytes(msg.Recipient); err != nil {
		return nil, errorsmod.Wrap(err, "invalid recipient address")
	}
	params, err := k.Params.Get(ctx)
	if err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}
	issuerIsAnchor := isJuror(msg.Creator, resolveAnchors(params.Anchors))
	if !issuerIsAnchor {
		score, err := k.Keeper.reputationScore(ctx, msg.Creator)
		if err != nil {
			return nil, err
		}
		if score < types.MinScopedEvidenceReputation {
			return nil, errorsmod.Wrapf(sdkerrors.ErrUnauthorized, "issuer reputation %.8f below scoped evidence threshold %.8f", score, types.MinScopedEvidenceReputation)
		}
	}
	if strings.TrimSpace(msg.Scope) == "" {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "scope is required")
	}
	if msg.Weight == 0 || msg.Weight > types.MaxVouchWeight {
		return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "weight must be 1..%d", types.MaxVouchWeight)
	}
	if strings.TrimSpace(msg.ArtifactUri) == "" {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "artifact_uri is required")
	}
	if strings.TrimSpace(msg.EvidenceUri) == "" {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "evidence_uri is required")
	}
	if !isHexSHA256(msg.ArtifactSha256) {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "artifact_sha256 must be a hex sha256")
	}
	if !isHexSHA256(msg.EvidenceSha256) {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "evidence_sha256 must be a hex sha256")
	}
	if msg.RationaleHash != "" && !isHexSHA256(msg.RationaleHash) {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "rationale_hash must be a hex sha256 when provided")
	}

	id, err := k.ScopedEvidenceVouchSeq.Next(ctx)
	if err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}
	vouch := types.ScopedEvidenceVouch{
		Id:             id,
		Issuer:         msg.Creator,
		Recipient:      msg.Recipient,
		Scope:          msg.Scope,
		Weight:         msg.Weight,
		ArtifactUri:    msg.ArtifactUri,
		ArtifactSha256: msg.ArtifactSha256,
		EvidenceUri:    msg.EvidenceUri,
		EvidenceSha256: msg.EvidenceSha256,
		RationaleHash:  msg.RationaleHash,
		ExpiresAt:      msg.ExpiresAt,
	}
	if err := k.ScopedEvidenceVouch.Set(ctx, id, vouch); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}

	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(
		sdk.NewEvent("agntcoin_scoped_evidence_vouch_cast",
			sdk.NewAttribute("id", strconv.FormatUint(id, 10)),
			sdk.NewAttribute("issuer", msg.Creator),
			sdk.NewAttribute("recipient", msg.Recipient),
			sdk.NewAttribute("scope", msg.Scope),
			sdk.NewAttribute("weight", strconv.FormatUint(msg.Weight, 10)),
			sdk.NewAttribute("artifact_sha256", msg.ArtifactSha256),
			sdk.NewAttribute("evidence_sha256", msg.EvidenceSha256),
		),
	)
	return &types.MsgCastScopedEvidenceVouchResponse{Id: id}, nil
}
