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
	// QUORUM (jury-v1, it20): a real majority of the eligible juror set must have
	// voted before a dispute can resolve — so a single colluding juror cannot vote
	// "accept" and immediately resolve a 1-0 verdict before honest jurors weigh in
	// (the rush-resolve hole the it20 RED targets). quorum = floor(n/2)+1 of the n
	// eligible jurors (a strict majority of the whole set, not just of votes cast).
	params, err := k.Params.Get(ctx)
	if err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}
	nJurors := len(resolveAnchors(params.Anchors))
	quorum := nJurors/2 + 1
	if len(dispute.Votes) < quorum {
		return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "quorum not reached: %d of %d eligible jurors voted (need %d)", len(dispute.Votes), nJurors, quorum)
	}
	if accept == reject {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "no majority yet — need a strict accept/reject majority to resolve")
	}

	escrow, err := k.Escrow.Get(ctx, dispute.EscrowId)
	if err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}
	// challenged (verifier-v1) = a post-release fraud claim against the
	// attesters: the payout already happened and is NEVER clawed back from the
	// payee — the verdict settles the BONDS (the attester's stake insured the
	// bounty; a fraud verdict pays the payer from it). in_jury = the
	// pre-settlement case where the verdict settles the escrow money itself.
	challenged := escrow.Status == types.EscrowStatusChallenged
	if escrow.Status != types.EscrowStatusInJury && !challenged {
		return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "escrow is not under jury control (status=%s)", escrow.Status)
	}

	var resolution string
	if challenged {
		// No escrow money moves: accept = the release was valid; reject = the
		// release was fraudulent, and the payer is made whole below from the
		// slashed attestation-stakes (each >= the bounty by the coverage
		// rule), while the payee keeps the payout — that's precisely what the
		// verifier's collateral insured. Either way the escrow returns to its
		// terminal released state.
		if accept > reject {
			resolution = types.DisputeResolutionAccept
		} else {
			resolution = types.DisputeResolutionReject
		}
		escrow.Status = types.EscrowStatusReleased
	} else if accept > reject {
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

	// Settle the opener's DISPUTE-BOND by the verdict (it19). The opener's dispute
	// is UPHELD iff the verdict went their way — payer-opener wants a reject
	// (refund), payee-opener wants an accept (release). Upheld → the bond returns
	// to the opener; not-upheld (frivolous/cry-wolf) → the bond is slashed to the
	// griefed counterparty. Protocol-settled here (the bond's slasher is empty, so
	// no external SlashBond/ReleaseBond can touch it); guarded by a purpose match.
	if bond, err := k.Bond.Get(ctx, dispute.BondId); err == nil &&
		bond.Status == types.BondStatusActive && bond.Purpose == "dispute:"+strconv.FormatUint(dispute.Id, 10) {
		openerUpheld := (dispute.Opener == escrow.Payer && resolution == types.DisputeResolutionReject) ||
			(dispute.Opener == escrow.Payee && resolution == types.DisputeResolutionAccept)
		recipient := dispute.Opener // upheld → refund the opener
		if openerUpheld {
			bond.Status = types.BondStatusReleased
		} else {
			// frivolous/lost → slash to the counterparty the opener griefed
			if dispute.Opener == escrow.Payer {
				recipient = escrow.Payee
			} else {
				recipient = escrow.Payer
			}
			if challenged {
				// A lost post-release fraud challenge griefed the ATTESTER —
				// the party the claim accused and whose stake it froze — not
				// the payee (verifier-v1; cry-wolf has a price and the
				// harassed party is compensated). v1: first pass-attester.
				for _, a := range escrow.Attestations {
					if a.Passed {
						recipient = a.Verifier
						break
					}
				}
			}
			bond.Status = types.BondStatusSlashed
		}
		acct, err := k.Account.Get(ctx, recipient)
		if err != nil {
			return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
		}
		acct.Balance += bond.Amount
		if err := k.Account.Set(ctx, recipient, acct); err != nil {
			return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
		}
		if err := k.Bond.Set(ctx, bond.Id, bond); err != nil {
			return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
		}
		sdkCtx0 := sdk.UnwrapSDKContext(ctx)
		sdkCtx0.EventManager().EmitEvent(
			sdk.NewEvent("agntcoin_dispute_bond_settled",
				sdk.NewAttribute("dispute_id", strconv.FormatUint(dispute.Id, 10)),
				sdk.NewAttribute("bond_id", strconv.FormatUint(bond.Id, 10)),
				sdk.NewAttribute("status", bond.Status),
				sdk.NewAttribute("recipient", recipient),
				sdk.NewAttribute("amount", strconv.FormatUint(bond.Amount, 10)),
			),
		)
	}

	// Settle each juror's JUROR-STAKE by coherence with the resolved verdict
	// (jury-v1, it20). A juror who voted WITH the resolution gets its stake back; a
	// juror who voted AGAINST it (incoherent — the minority-collusion signature) is
	// slashed, the stake credited to the party the verdict favored (whom the
	// incoherent vote tried to harm). Protocol-settled; guarded by the jurorstake
	// purpose. v1 scope: this punishes a MINORITY colluder caught by an honest
	// majority; a colluding MAJORITY would invert it — deferred to reputation-
	// weighted / proper-scoring voting (it25).
	verdictAccept := resolution == types.DisputeResolutionAccept
	winner := escrow.Payer
	if verdictAccept {
		winner = escrow.Payee
	}
	for _, v := range dispute.Votes {
		bond, err := k.Bond.Get(ctx, v.BondId)
		if err != nil || bond.Status != types.BondStatusActive || bond.Purpose != "jurorstake:"+strconv.FormatUint(dispute.Id, 10) {
			continue
		}
		recipient := v.Juror // coherent → stake returned to the juror
		if v.Accept == verdictAccept {
			bond.Status = types.BondStatusReleased
		} else {
			recipient = winner // incoherent → slashed to the party it tried to wrong
			bond.Status = types.BondStatusSlashed
		}
		acct, err := k.Account.Get(ctx, recipient)
		if err != nil {
			return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
		}
		acct.Balance += bond.Amount
		if err := k.Account.Set(ctx, recipient, acct); err != nil {
			return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
		}
		if err := k.Bond.Set(ctx, bond.Id, bond); err != nil {
			return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
		}
		sdkCtxJ := sdk.UnwrapSDKContext(ctx)
		sdkCtxJ.EventManager().EmitEvent(
			sdk.NewEvent("agntcoin_juror_stake_settled",
				sdk.NewAttribute("dispute_id", strconv.FormatUint(dispute.Id, 10)),
				sdk.NewAttribute("juror", v.Juror),
				sdk.NewAttribute("bond_id", strconv.FormatUint(bond.Id, 10)),
				sdk.NewAttribute("status", bond.Status),
				sdk.NewAttribute("recipient", recipient),
				sdk.NewAttribute("amount", strconv.FormatUint(bond.Amount, 10)),
			),
		)
	}

	// Settle each verifier's ATTESTATION-STAKE by coherence with the verdict
	// (verifier-v1) — the same shape as juror stakes, because the attester is
	// just a juror who ruled FIRST and at higher stakes. accept (work good /
	// release valid) vindicates pass attestations and convicts fail ones;
	// reject vindicates fail attestations and convicts pass ones. A convicted
	// stake is slashed to the party the lie tried to wrong (`winner`): a
	// false-pass tried to drain the payer, a false-fail tried to stiff the
	// payee. Coverage rule (stake >= bounty) means a fraud verdict makes the
	// payer whole here even though the payout itself stays with the payee.
	// Coherent stakes are released. One provable lie costs the verifier more
	// than honest verification of many deals earns — that's the deterrent.
	attPurpose := "attestation:" + strconv.FormatUint(escrow.Id, 10)
	for _, a := range escrow.Attestations {
		bond, err := k.Bond.Get(ctx, a.BondId)
		if err != nil || bond.Status != types.BondStatusActive || bond.Purpose != attPurpose {
			continue
		}
		recipient := a.Verifier // coherent → stake returned to the verifier
		if a.Passed == verdictAccept {
			bond.Status = types.BondStatusReleased
		} else {
			recipient = winner // incoherent → slashed to the wronged party
			bond.Status = types.BondStatusSlashed
		}
		acct, err := k.Account.Get(ctx, recipient)
		if err != nil {
			return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
		}
		acct.Balance += bond.Amount
		if err := k.Account.Set(ctx, recipient, acct); err != nil {
			return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
		}
		if err := k.Bond.Set(ctx, bond.Id, bond); err != nil {
			return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
		}
		sdkCtxA := sdk.UnwrapSDKContext(ctx)
		sdkCtxA.EventManager().EmitEvent(
			sdk.NewEvent("agntcoin_attestation_bond_settled",
				sdk.NewAttribute("dispute_id", strconv.FormatUint(dispute.Id, 10)),
				sdk.NewAttribute("escrow_id", strconv.FormatUint(escrow.Id, 10)),
				sdk.NewAttribute("verifier", a.Verifier),
				sdk.NewAttribute("bond_id", strconv.FormatUint(bond.Id, 10)),
				sdk.NewAttribute("status", bond.Status),
				sdk.NewAttribute("recipient", recipient),
				sdk.NewAttribute("amount", strconv.FormatUint(bond.Amount, 10)),
			),
		)
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
