package keeper

import (
	"context"
	"os"
	"strconv"
	"strings"

	errorsmod "cosmossdk.io/errors"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/zoltankiss/agntcoin/x/agntcoin/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// resolveAnchors returns the reputation trust roots. The governed Params.Anchors
// is the source of truth; if it is empty, we fall back to a DEV/TEST source so a
// shared dev chain can be pointed at a per-run founder anchor without a genesis
// reinit or a gov proposal — production chains set Params.Anchors via governance.
// Dev source precedence: AGNTCOIN_ANCHORS env (comma-separated) → the file at
// AGNTCOIN_ANCHORS_FILE (default ~/.agntcoin/anchors.txt), comma/newline list.
// The file form is robust under a process supervisor (ignite): write the file
// and the next query reflects it, no restart.
func resolveAnchors(paramAnchors []string) []string {
	if len(paramAnchors) > 0 {
		return paramAnchors
	}
	if raw := strings.TrimSpace(os.Getenv("AGNTCOIN_ANCHORS")); raw != "" {
		return splitAnchors(raw)
	}
	path := strings.TrimSpace(os.Getenv("AGNTCOIN_ANCHORS_FILE"))
	if path == "" {
		if home, err := os.UserHomeDir(); err == nil {
			path = home + "/.agntcoin/anchors.txt"
		}
	}
	if path != "" {
		if b, err := os.ReadFile(path); err == nil {
			return splitAnchors(string(b))
		}
	}
	return nil
}

// splitAnchors parses a comma/whitespace/newline-separated address list.
func splitAnchors(raw string) []string {
	fields := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '\n' || r == '\r' || r == ' ' || r == '\t'
	})
	out := make([]string, 0, len(fields))
	for _, a := range fields {
		if a != "" {
			out = append(out, a)
		}
	}
	return out
}

func (q queryServer) Reputation(ctx context.Context, req *types.QueryReputationRequest) (*types.QueryReputationResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}
	if _, err := q.k.addressCodec.StringToBytes(req.Address); err != nil {
		return nil, errorsmod.Wrap(err, "invalid address")
	}

	// Vouch edges: every staked vouch.
	vouches := make([]types.Vouch, 0)
	if err := q.k.Vouch.Walk(ctx, nil, func(_ uint64, v types.Vouch) (bool, error) {
		vouches = append(vouches, v)
		return false, nil
	}); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}

	// Job edges: every released escrow is a completed, paid job (payer→payee).
	// A released escrow is on-chain proof that a real counterparty paid for and
	// accepted the work — Sybil-resistant by construction.
	jobs := make([]JobEdge, 0)
	if err := q.k.Escrow.Walk(ctx, nil, func(_ uint64, e types.Escrow) (bool, error) {
		if e.Status == types.EscrowStatusReleased {
			jobs = append(jobs, JobEdge{Payer: e.Payer, Payee: e.Payee, Amount: e.Amount})
		}
		return false, nil
	}); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}

	// Jury-endorsement edges: when a jury ACCEPTS a contested delivery, each
	// juror who voted accept has *endorsed* the worker on the merits. Modeled as
	// a directed edge juror→payee — mechanically identical to a job edge, so it
	// flows reputation. This fixes the it12 "money-not-standing" finding: a
	// worker hired by a buyer who is not anchor-rooted gets paid but earns no
	// standing, because the buyer→payee job edge carries ~no anchor-rooted rank.
	// A jury-accept by an anchor-juror is, by construction, an anchor-rooted
	// endorsement — so the worker finally accrues reputation for work an anchor
	// judged good. Derived purely from on-chain state (resolved-accept disputes
	// already record their jurors' votes + the escrow), so it adds no new
	// storage and stays deterministic for consensus.
	//
	// v0 safety note: under a *colluding* juror (v1, decentralized jurors) this
	// is a reputation-laundering vector — a corrupt juror could accept fake work
	// to launder rank to a Sybil. Defending that is exactly what jury-v1
	// (staking + slashing of incoherent jurors) is for; in v0 the only juror is
	// the honest anchor, so it is safe. (See BACKLOG: jury-v1.)
	if err := q.k.Dispute.Walk(ctx, nil, func(_ uint64, d types.Dispute) (bool, error) {
		if d.Resolution != types.DisputeResolutionAccept {
			return false, nil
		}
		e, err := q.k.Escrow.Get(ctx, d.EscrowId)
		if err != nil {
			// A resolved dispute always has its escrow; skip defensively rather
			// than fail the whole reputation query on an unexpected gap.
			return false, nil
		}
		for _, v := range d.Votes {
			if v.Accept {
				jobs = append(jobs, JobEdge{Payer: v.Juror, Payee: e.Payee, Amount: e.Amount})
			}
		}
		return false, nil
	}); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}

	// Anchors (trust roots) from params.
	params, err := q.k.Params.Get(ctx)
	if err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}

	scores := Reputation(vouches, jobs, resolveAnchors(params.Anchors))
	score, ok := scores[req.Address]
	if !ok {
		if _, err := q.k.Account.Get(ctx, req.Address); err != nil {
			return nil, status.Error(codes.NotFound, "address not in graph and not registered")
		}
		score = 0
	}
	return &types.QueryReputationResponse{Score: strconv.FormatFloat(score, 'f', -1, 64)}, nil
}
