package keeper

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"strconv"
	"strings"

	"cosmossdk.io/collections"
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/zoltankiss/agntcoin/x/agntcoin/types"
)

// gate-v1 (0.4.0-beta.1): protocol PR-review gates + the earned faucet.
//
// A gate streams one slice of protocol-public review work to the agent pool.
// For the beachhead, the payload is a content-addressed PR-review bundle
// (repo/base/head/diff/test-evidence/options/invariant scope). Agents COMMIT
// hashed answers while the commit window is open, REVEAL them once it closes
// (so an early reveal can never be copied into a later commit — the
// anti-free-riding property is structural, not policed), and a coherent answer
// MINTS a tiny drip at settlement. This is the chain's first PoUW mint rail and
// the replacement path for the sybil-exploitable one-shot faucet: a 0-coin
// 0-rep agent's first working capital is EARNED through calibrated protocol
// review labor.
//
// Decoys vs live gates: gold_commit = sha256("<gold_answer>:<salt>") — a
// known verdict for decoys, the empty answer for live gates. The two are
// byte-identical on-chain until settlement, so "answer honestly always" is
// the only coherent policy for an answerer. Settlement pays exact gold
// matches (decoy) or the strict-plurality answer (live); ties pay nobody. In
// beta.1, live gates are advisory review signal only. They must not become PR
// merge authority or escrow settlement authority until rep-weighted/sortitioned
// review and vesting/slash are forged.

// gateAnswerCommit is the chain's commit canonicalization for gate answers
// and gold verdicts: hex sha256("<answer>:<salt>").
func gateAnswerCommit(answer, salt string) string {
	sum := sha256.Sum256([]byte(answer + ":" + salt))
	return hex.EncodeToString(sum[:])
}

func gateWindowSeconds(envName string, fallback int64) int64 {
	raw := strings.TrimSpace(os.Getenv(envName))
	if raw == "" {
		return fallback
	}
	n, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}

// PostGate opens a gate. ANCHOR-only in v1: the drip is minted supply, so
// emission authority stays with the trust roots until the no-self-mining
// inequality (drip per deal < fees + capital cost) is enforced in state.
func (k msgServer) PostGate(ctx context.Context, msg *types.MsgPostGate) (*types.MsgPostGateResponse, error) {
	if _, err := k.addressCodec.StringToBytes(msg.Creator); err != nil {
		return nil, errorsmod.Wrap(err, "invalid creator address")
	}
	params, err := k.Params.Get(ctx)
	if err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}
	if !isJuror(msg.Creator, resolveAnchors(params.Anchors)) {
		return nil, errorsmod.Wrap(sdkerrors.ErrUnauthorized, "only an anchor may post gates (the drip is minted supply)")
	}
	// The gate's task content is pinned by hash (same evidence discipline as
	// escrow delivery_hash); the gold commitment must be re-checkable.
	if !isHexSHA256(msg.PayloadHash) {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "payload_hash must be a hex sha256")
	}
	if !isHexSHA256(msg.GoldCommit) {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "gold_commit must be a hex sha256")
	}
	if msg.Drip == 0 || msg.Drip > types.MaxDripPerAnswer {
		return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "drip must be 1..%d (the per-answer emission cap)", types.MaxDripPerAnswer)
	}
	if msg.MaxAnswers == 0 || msg.MaxAnswers > types.MaxGateAnswers {
		return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "max_answers must be 1..%d", types.MaxGateAnswers)
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	now := sdkCtx.BlockTime().Unix()
	commitWindow := gateWindowSeconds("AGNT_GATE_COMMIT_WINDOW_SECONDS", types.GateCommitWindowSeconds)
	revealWindow := gateWindowSeconds("AGNT_GATE_REVEAL_WINDOW_SECONDS", types.GateRevealWindowSeconds)
	id, err := k.GateSeq.Next(ctx)
	if err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}
	gate := types.Gate{
		Id:             id,
		Poster:         msg.Creator,
		PayloadUri:     msg.PayloadUri,
		PayloadHash:    msg.PayloadHash,
		GoldCommit:     msg.GoldCommit,
		Drip:           msg.Drip,
		MaxAnswers:     msg.MaxAnswers,
		CommitDeadline: now + commitWindow,
		RevealDeadline: now + commitWindow + revealWindow,
		Status:         types.GateStatusOpen,
	}
	if err := k.Gate.Set(ctx, id, gate); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}

	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent("agntcoin_gate_posted",
			sdk.NewAttribute("id", strconv.FormatUint(id, 10)),
			sdk.NewAttribute("poster", msg.Creator),
			sdk.NewAttribute("payload_hash", msg.PayloadHash),
			sdk.NewAttribute("drip", strconv.FormatUint(msg.Drip, 10)),
			sdk.NewAttribute("commit_deadline", strconv.FormatInt(gate.CommitDeadline, 10)),
			sdk.NewAttribute("reveal_deadline", strconv.FormatInt(gate.RevealDeadline, 10)),
		),
	)
	return &types.MsgPostGateResponse{Id: id}, nil
}

// CommitAnswer posts a hashed answer during the commit window. Free for a
// 0-balance agent — this is the entry ramp; the anti-guesser teeth are k-ary
// answers + decoy density + coherence-only pay, not an entry fee.
func (k msgServer) CommitAnswer(ctx context.Context, msg *types.MsgCommitAnswer) (*types.MsgCommitAnswerResponse, error) {
	if _, err := k.addressCodec.StringToBytes(msg.Creator); err != nil {
		return nil, errorsmod.Wrap(err, "invalid creator address")
	}
	gate, err := k.Gate.Get(ctx, msg.GateId)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil, errorsmod.Wrap(sdkerrors.ErrKeyNotFound, "gate not found")
		}
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}
	if gate.Status != types.GateStatusOpen {
		return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "gate is not open (status=%s)", gate.Status)
	}
	now := sdk.UnwrapSDKContext(ctx).BlockTime().Unix()
	if now >= gate.CommitDeadline {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "commit window has closed")
	}
	// No self-mining: the poster precommitted the verdict — answering its own
	// gate would be printing drip against its own salt.
	if msg.Creator == gate.Poster {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "the poster cannot answer its own gate")
	}
	acct, err := k.Account.Get(ctx, msg.Creator)
	if err != nil || !acct.Registered {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "answerer not registered")
	}
	if !isHexSHA256(msg.Commit) {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "commit must be a hex sha256")
	}
	if uint64(len(gate.Answers)) >= gate.MaxAnswers {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "gate is full")
	}
	for _, a := range gate.Answers {
		if a.Agent == msg.Creator {
			return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "agent has already committed to this gate")
		}
	}

	gate.Answers = append(gate.Answers, &types.GateAnswer{Agent: msg.Creator, Commit: msg.Commit})
	if err := k.Gate.Set(ctx, gate.Id, gate); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}
	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(
		sdk.NewEvent("agntcoin_gate_answer_committed",
			sdk.NewAttribute("gate_id", strconv.FormatUint(gate.Id, 10)),
			sdk.NewAttribute("agent", msg.Creator),
		),
	)
	return &types.MsgCommitAnswerResponse{}, nil
}

// RevealAnswer opens a committed answer. Valid ONLY between the commit
// deadline (so commits are closed — nothing left to copy into) and the reveal
// deadline. The preimage must reproduce the agent's own commit exactly.
func (k msgServer) RevealAnswer(ctx context.Context, msg *types.MsgRevealAnswer) (*types.MsgRevealAnswerResponse, error) {
	if _, err := k.addressCodec.StringToBytes(msg.Creator); err != nil {
		return nil, errorsmod.Wrap(err, "invalid creator address")
	}
	gate, err := k.Gate.Get(ctx, msg.GateId)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil, errorsmod.Wrap(sdkerrors.ErrKeyNotFound, "gate not found")
		}
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}
	if gate.Status != types.GateStatusOpen {
		return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "gate is not open (status=%s)", gate.Status)
	}
	now := sdk.UnwrapSDKContext(ctx).BlockTime().Unix()
	if now < gate.CommitDeadline {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "reveal window has not opened (commits are still open — an early reveal would be copyable)")
	}
	if now >= gate.RevealDeadline {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "reveal window has closed")
	}
	if msg.Answer == "" {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "answer must be non-empty (the empty answer is the live-gate gold sentinel)")
	}

	for i, a := range gate.Answers {
		if a.Agent != msg.Creator {
			continue
		}
		if a.Revealed {
			return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "answer already revealed")
		}
		if !strings.EqualFold(gateAnswerCommit(msg.Answer, msg.Salt), a.Commit) {
			return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "answer+salt does not match the commit")
		}
		gate.Answers[i].Answer = msg.Answer
		gate.Answers[i].Revealed = true
		if err := k.Gate.Set(ctx, gate.Id, gate); err != nil {
			return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
		}
		sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(
			sdk.NewEvent("agntcoin_gate_answer_revealed",
				sdk.NewAttribute("gate_id", strconv.FormatUint(gate.Id, 10)),
				sdk.NewAttribute("agent", msg.Creator),
				sdk.NewAttribute("answer", msg.Answer),
			),
		)
		return &types.MsgRevealAnswerResponse{}, nil
	}
	return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "no commit from this agent on this gate")
}

// SettleGate settles once, at/after the reveal deadline, against the revealed
// gold preimage. Non-empty gold => decoy (exact matches paid); empty gold =>
// live (strict-plurality answer paid; tie pays nobody). The drip is MINTED —
// credited without debiting anyone — which is exactly what makes this the
// emission rail, and exactly why posting is anchor-gated in v1.
func (k msgServer) SettleGate(ctx context.Context, msg *types.MsgSettleGate) (*types.MsgSettleGateResponse, error) {
	if _, err := k.addressCodec.StringToBytes(msg.Creator); err != nil {
		return nil, errorsmod.Wrap(err, "invalid creator address")
	}
	gate, err := k.Gate.Get(ctx, msg.GateId)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return nil, errorsmod.Wrap(sdkerrors.ErrKeyNotFound, "gate not found")
		}
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}
	if gate.Status != types.GateStatusOpen {
		return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "gate is not open (status=%s)", gate.Status)
	}
	now := sdk.UnwrapSDKContext(ctx).BlockTime().Unix()
	if now < gate.RevealDeadline {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "reveal window has not closed")
	}
	if !strings.EqualFold(gateAnswerCommit(msg.GoldAnswer, msg.GoldSalt), gate.GoldCommit) {
		return nil, errorsmod.Wrap(sdkerrors.ErrInvalidRequest, "gold preimage does not match the gate's gold_commit")
	}

	// Winning answer: the precommitted gold verdict (decoy), or the strict
	// plurality of revealed answers (live). A tie has no consensus — gates
	// must never pay on a coin flip or the poster's discretion.
	winning := msg.GoldAnswer
	if winning == "" {
		counts := map[string]int{}
		for _, a := range gate.Answers {
			if a.Revealed {
				counts[a.Answer]++
			}
		}
		best, bestCount, tied := "", 0, false
		for ans, c := range counts {
			switch {
			case c > bestCount:
				best, bestCount, tied = ans, c, false
			case c == bestCount:
				tied = true
			}
		}
		if !tied && bestCount > 0 {
			winning = best
		}
	}

	// Mint the drip to every coherent revealed answer. Answerers were required
	// to be registered at commit time, so the new-account dust guard never
	// blocks an earned drip.
	var minted uint64
	if winning != "" {
		for i, a := range gate.Answers {
			if !a.Revealed || a.Answer != winning {
				continue
			}
			agentBytes, err := k.addressCodec.StringToBytes(a.Agent)
			if err != nil {
				return nil, errorsmod.Wrap(err, "invalid answerer address")
			}
			if err := k.mintProtocolReward(ctx, a.Agent, agentBytes, gate.Drip, "gate"); err != nil {
				return nil, err
			}
			gate.Answers[i].Paid = true
			minted += gate.Drip
		}
	}

	gate.Status = types.GateStatusSettled
	gate.GoldAnswer = msg.GoldAnswer
	gate.Consensus = winning
	if err := k.Gate.Set(ctx, gate.Id, gate); err != nil {
		return nil, errorsmod.Wrap(sdkerrors.ErrIO, err.Error())
	}

	sdk.UnwrapSDKContext(ctx).EventManager().EmitEvent(
		sdk.NewEvent("agntcoin_gate_settled",
			sdk.NewAttribute("gate_id", strconv.FormatUint(gate.Id, 10)),
			sdk.NewAttribute("kind", map[bool]string{true: "decoy", false: "live"}[msg.GoldAnswer != ""]),
			sdk.NewAttribute("consensus", winning),
			sdk.NewAttribute("minted", strconv.FormatUint(minted, 10)),
		),
	)
	return &types.MsgSettleGateResponse{Consensus: winning}, nil
}
