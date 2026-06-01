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
