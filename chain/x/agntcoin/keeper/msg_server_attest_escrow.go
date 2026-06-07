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

// AttestEscrow records a named verifier's staked pass/fail verdict on a
// SUBMITTED verifier-bound escrow (verifier-v1). The verifier's own tx
// signature is the attestation authority — the chain never parses a detached
// "canonical JSON" signature; proto tx signing is already deterministic bytes.
//
// Every attestation locks a per-attestation ATTESTATION-STAKE >= the escrow
// amount (the bond-coverage rule): the stake is what makes the optimistic
// release trustworthy-by-economics — a jury-proven false attestation is
// slashed to the wronged party, so one provable lie costs the verifier more
// than honest verification of many deals earns. Per-attestation (not a shared
// standing bond) so N concurrent deals lock N stakes and exposure can never
// silently exceed collateral.
//
// passed=true counts toward the VerifiedRelease quorum. passed=false moves
// the escrow to fail_attested and opens the payee's CONTEST window: refund
// stays locked until the window passes uncontested, so a colluding
// buyer+verifier can't false-fail + instant-refund delivered work (the
// scenario-7 symmetry hole). The payee contests via OpenDispute; a jury that
// finds the work good slashes this stake to the payee.
func (k msgServer) AttestEscrow(ctx context.Context, msg *types.MsgAttestEscrow) (*types.MsgAttestEscrowResponse, error) {
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
	// Only a SUBMITTED escrow is attestable: before submit there is no
	// committed delivery to judge; after settlement (or under jury control)
	// the verdict no longer has a release to authorize. v1 limitation: the
	// first fail attestation moves the escrow off "submitted", freezing the
	// pass path — with m-of-n sets a single rogue fail blocks release until
	// the contest window runs (documented, acceptable at quorum 1-of-1).
	if escrow.Status != types.EscrowStatusSubmitted {
		return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "escrow is not submitted (status=%s)", escrow.Status)
	}
	if !isVerifier(msg.Creator, escrow.VerifierAddrs) {
		return nil, errorsmod.Wrap(sdkerrors.ErrUnauthorized, "caller is not in the escrow's verifier set")
	}
	for _, a := range escrow.Attestations {
		if a.Verifier == msg.Creator {
			return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "verifier has already attested this escrow")
		}
	}

	// Bond coverage: this attestation alone must be able to make the wronged
	// party whole, so the stake must cover the full bounty. Chain-enforced
	// here (not spot-checked at release) — the deterrent is arithmetic, not
	// theater.
	if msg.StakeAmount < escrow.Amount {
		return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "attestation stake must be at least the escrow amount (%d)", escrow.Amount)
	}
	verifier, err := k.Account.Get(ctx, msg.Creator)
	if err != nil || !verifier.Registered {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "verifier not registered")
	}
	if verifier.Balance < msg.StakeAmount {
		return nil, errorsmod.Wrap(sdkerrors.ErrInsufficientFunds, "insufficient balance for attestation stake")
	}
	verifier.Balance -= msg.StakeAmount
	if err := k.Account.Set(ctx, msg.Creator, verifier); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}
	bondID, err := k.BondSeq.Next(ctx)
	if err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}
	if err := k.Bond.Set(ctx, bondID, types.Bond{
		Id:      bondID,
		Poster:  msg.Creator,
		Amount:  msg.StakeAmount,
		Purpose: "attestation:" + strconv.FormatUint(escrow.Id, 10),
		Slasher: "", // protocol-settled: ResolveDispute (challenged/contested) or ReclaimAttestationBonds
		Status:  types.BondStatusActive,
		Ref:     strconv.FormatUint(escrow.Id, 10),
	}); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}

	escrow.Attestations = append(escrow.Attestations, &types.Attestation{
		Verifier: msg.Creator,
		Passed:   msg.Passed,
		BondId:   bondID,
	})

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	if !msg.Passed {
		// Fail verdict: open the payee's contest window. Refund unlocks only
		// once it passes uncontested (RefundEscrow checks this), so the fail
		// attestation justifies the refund WITHOUT enabling instant clawback.
		// Deadline = BLOCK time + window — never the verifier's wall clock.
		escrow.Status = types.EscrowStatusFailAttested
		escrow.ChallengeDeadline = sdkCtx.BlockTime().Unix() + types.ChallengeWindowSeconds
	}
	if err := k.Escrow.Set(ctx, escrow.Id, escrow); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}

	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent("agntcoin_escrow_attested",
			sdk.NewAttribute("escrow_id", strconv.FormatUint(escrow.Id, 10)),
			sdk.NewAttribute("verifier", msg.Creator),
			sdk.NewAttribute("passed", strconv.FormatBool(msg.Passed)),
			sdk.NewAttribute("bond_id", strconv.FormatUint(bondID, 10)),
			sdk.NewAttribute("stake", strconv.FormatUint(msg.StakeAmount, 10)),
		),
	)
	return &types.MsgAttestEscrowResponse{BondId: bondID}, nil
}

// isVerifier reports whether addr is in the escrow's named verifier set.
func isVerifier(addr string, verifiers []string) bool {
	for _, v := range verifiers {
		if v == addr {
			return true
		}
	}
	return false
}
