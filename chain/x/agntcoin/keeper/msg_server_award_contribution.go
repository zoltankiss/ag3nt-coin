package keeper

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/zoltankiss/agntcoin/x/agntcoin/types"
)

// AwardContribution is the beta author-mint rail for proof-of-useful-work.
//
// In 0.4.0-beta.1 this is intentionally founder/anchor-mediated: an anchor
// accepts a concrete protocol contribution, pins public evidence by hash, and
// mints a capped amount to the author. This gives the genesis beachhead both
// halves of the useful-work loop (author + reviewer) without pretending the
// final decentralized acceptance process exists yet.
func (k msgServer) AwardContribution(ctx context.Context, msg *types.MsgAwardContribution) (*types.MsgAwardContributionResponse, error) {
	if _, err := k.addressCodec.StringToBytes(msg.Creator); err != nil {
		return nil, errorsmod.Wrap(err, "invalid creator address")
	}
	recipientBytes, err := k.addressCodec.StringToBytes(msg.Recipient)
	if err != nil {
		return nil, errorsmod.Wrap(err, "invalid recipient address")
	}
	if _, err := k.addressCodec.StringToBytes(msg.Contributor); err != nil {
		return nil, errorsmod.Wrap(err, "invalid contributor address")
	}

	params, err := k.Params.Get(ctx)
	if err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}
	if !isJuror(msg.Creator, resolveAnchors(params.Anchors)) {
		return nil, errorsmod.Wrap(sdkerrors.ErrUnauthorized, "only an anchor may award protocol contribution mint in beta")
	}
	if msg.Amount == 0 || msg.Amount > types.MaxContributionAward {
		return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "amount must be 1..%d", types.MaxContributionAward)
	}
	if msg.Recipient != msg.Contributor {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "recipient must match contributor")
	}
	if msg.FounderAuthored && msg.Recipient != msg.Creator {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "founder_authored requires recipient to match anchor")
	}
	if msg.Recipient == msg.Creator && !msg.FounderAuthored {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "anchor self-awards require founder_authored=true")
	}
	if msg.FounderAuthored && strings.TrimSpace(msg.ReviewEvidenceUri) == "" {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "founder-authored awards require review_evidence_uri")
	}
	if strings.TrimSpace(msg.RepoUrl) == "" {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "repo_url is required")
	}
	if strings.TrimSpace(msg.CommitSha) == "" {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "commit_sha is required")
	}
	if strings.TrimSpace(msg.ArtifactUri) == "" {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "artifact_uri is required")
	}
	if strings.TrimSpace(msg.Scope) == "" {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "scope is required")
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

	var duplicate bool
	err = k.ContributionAward.Walk(ctx, nil, func(_ uint64, a types.ContributionAward) (bool, error) {
		if strings.EqualFold(a.ArtifactSha256, msg.ArtifactSha256) {
			duplicate = true
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil, errorsmod.Wrap(sdkerrors.ErrKeyNotFound, err.Error())
		}
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}
	if duplicate {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "artifact_sha256 has already been awarded")
	}
	if err := k.checkNewAccountCredit(ctx, msg.Recipient, msg.Amount); err != nil {
		return nil, err
	}

	id, err := k.ContributionAwardSeq.Next(ctx)
	if err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}

	award := types.ContributionAward{
		Id:                id,
		Anchor:            msg.Creator,
		Recipient:         msg.Recipient,
		RepoUrl:           msg.RepoUrl,
		PrUrl:             msg.PrUrl,
		CommitSha:         msg.CommitSha,
		ArtifactUri:       msg.ArtifactUri,
		ArtifactSha256:    msg.ArtifactSha256,
		EvidenceSha256:    msg.EvidenceSha256,
		Scope:             msg.Scope,
		RationaleHash:     msg.RationaleHash,
		Amount:            msg.Amount,
		Contributor:       msg.Contributor,
		FounderAuthored:   msg.FounderAuthored,
		ReviewEvidenceUri: msg.ReviewEvidenceUri,
	}
	if err := k.ContributionAward.Set(ctx, id, award); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}

	if err := k.mintProtocolReward(ctx, msg.Recipient, recipientBytes, msg.Amount, "contribution_award"); err != nil {
		return nil, err
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent("agntcoin_contribution_awarded",
			sdk.NewAttribute("id", strconv.FormatUint(id, 10)),
			sdk.NewAttribute("anchor", msg.Creator),
			sdk.NewAttribute("recipient", msg.Recipient),
			sdk.NewAttribute("contributor", msg.Contributor),
			sdk.NewAttribute("founder_authored", strconv.FormatBool(msg.FounderAuthored)),
			sdk.NewAttribute("repo_url", msg.RepoUrl),
			sdk.NewAttribute("commit_sha", msg.CommitSha),
			sdk.NewAttribute("artifact_sha256", msg.ArtifactSha256),
			sdk.NewAttribute("evidence_sha256", msg.EvidenceSha256),
			sdk.NewAttribute("review_evidence_uri", msg.ReviewEvidenceUri),
			sdk.NewAttribute("scope", msg.Scope),
			sdk.NewAttribute("amount", strconv.FormatUint(msg.Amount, 10)),
		),
	)

	return &types.MsgAwardContributionResponse{Id: id}, nil
}
