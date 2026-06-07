package keeper_test

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/zoltankiss/agntcoin/testutil/sample"
	"github.com/zoltankiss/agntcoin/x/agntcoin/keeper"
	"github.com/zoltankiss/agntcoin/x/agntcoin/types"
)

// gate-v1 (0.4.0): reCAPTCHA-style verification gates + the earned faucet.
// The scenarios these tests script:
//
//	G1  decoy gate happy path — a 0-coin agent's first working capital is
//	    MINTED for a calibrated answer (the earned faucet / first PoUW rail)
//	G2  commit-reveal discipline — every ordering / forgery hole
//	G3  live gate plurality consensus — majority paid, minority not, tie voids
//	G4  posting guards — anchor-only emission, drip/answer caps, hash formats
//	G5  guesser micro-sim — an uninformed constant policy earns a fraction of
//	    an honest answerer across a decoy stream (the anti-sybil arithmetic)
//
// Design notes under test: decoy and live gates are BYTE-IDENTICAL on-chain
// until settlement (gold_commit hides which); reveals only open after commits
// close, so copying is impossible by construction; coherence pays, presence
// does not.

// gateCommit reproduces the chain's commit canonicalization:
// hex sha256("<answer>:<salt>").
func gateCommit(answer, salt string) string {
	sum := sha256.Sum256([]byte(answer + ":" + salt))
	return hex.EncodeToString(sum[:])
}

// postGate posts a gate from `anchor` at block time `at` and returns its id.
func postGate(t *testing.T, f *fixture, ms types.MsgServer, anchor string, at int64, goldCommit string, drip, maxAnswers uint64) uint64 {
	t.Helper()
	resp, err := ms.PostGate(setBlockTime(f.ctx, at), &types.MsgPostGate{
		Creator:     anchor,
		PayloadUri:  "ipfs://bafygatepayload",
		PayloadHash: hexHash("gate-task-content"),
		GoldCommit:  goldCommit,
		Drip:        drip,
		MaxAnswers:  maxAnswers,
	})
	require.NoError(t, err)
	return resp.Id
}

// G1 — Decoy gate, the earned faucet: three 0-coin agents answer; the two
// coherent with the precommitted gold verdict are MINTED the drip (no payer
// debited anywhere); the incoherent one earns nothing.
func TestGateDecoyEarnedDripHappyPath(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	anchor := sample.AccAddress()
	setJurors(t, f, anchor)
	seedAccount(t, f, anchor, 1000)
	a1, a2, a3 := sample.AccAddress(), sample.AccAddress(), sample.AccAddress()
	for _, a := range []string{a1, a2, a3} {
		seedAccount(t, f, a, 0) // registered, ZERO balance — the entry ramp
	}

	const gold, goldSalt = "fail:item2", "s3cret-gold-salt"
	id := postGate(t, f, ms, anchor, 100, gateCommit(gold, goldSalt), 10, 100)

	g, err := f.keeper.Gate.Get(f.ctx, id)
	require.NoError(t, err)
	require.Equal(t, types.GateStatusOpen, g.Status)
	require.Equal(t, int64(100+types.GateCommitWindowSeconds), g.CommitDeadline)
	require.Equal(t, g.CommitDeadline+types.GateRevealWindowSeconds, g.RevealDeadline)
	// Pre-settlement a decoy is indistinguishable from a live gate: the only
	// verdict-bearing field is an opaque hash commitment.
	require.Empty(t, g.GoldAnswer)
	require.Empty(t, g.Consensus)

	// Commit window: hashed answers only — nothing to copy.
	for i, c := range []struct{ agent, answer, salt string }{
		{a1, gold, "salt-1"}, {a2, gold, "salt-2"}, {a3, "pass", "salt-3"},
	} {
		_, err := ms.CommitAnswer(setBlockTime(f.ctx, 200+int64(i)), &types.MsgCommitAnswer{
			Creator: c.agent, GateId: id, Commit: gateCommit(c.answer, c.salt),
		})
		require.NoError(t, err)
	}

	// Reveal window (commits are closed).
	rt := int64(100) + types.GateCommitWindowSeconds + 10
	for i, c := range []struct{ agent, answer, salt string }{
		{a1, gold, "salt-1"}, {a2, gold, "salt-2"}, {a3, "pass", "salt-3"},
	} {
		_, err := ms.RevealAnswer(setBlockTime(f.ctx, rt+int64(i)), &types.MsgRevealAnswer{
			Creator: c.agent, GateId: id, Answer: c.answer, Salt: c.salt,
		})
		require.NoError(t, err)
	}

	// Settle: gold preimage revealed; coherent answers minted, incoherent not.
	st := int64(100) + types.GateCommitWindowSeconds + types.GateRevealWindowSeconds + 1
	resp, err := ms.SettleGate(setBlockTime(f.ctx, st), &types.MsgSettleGate{
		Creator: anchor, GateId: id, GoldAnswer: gold, GoldSalt: goldSalt,
	})
	require.NoError(t, err)
	require.Equal(t, gold, resp.Consensus)

	require.Equal(t, uint64(10), balanceOf(t, f, a1), "coherent answer mints the drip — first coin from labor, not a faucet")
	require.Equal(t, uint64(10), balanceOf(t, f, a2))
	require.Equal(t, uint64(0), balanceOf(t, f, a3), "incoherent answer earns nothing")
	require.Equal(t, uint64(1000), balanceOf(t, f, anchor), "drip is MINTED, not debited from the poster")

	g, _ = f.keeper.Gate.Get(f.ctx, id)
	require.Equal(t, types.GateStatusSettled, g.Status)
	require.Equal(t, gold, g.GoldAnswer)
	require.Equal(t, gold, g.Consensus)
	paid := 0
	for _, a := range g.Answers {
		if a.Paid {
			paid++
		}
	}
	require.Equal(t, 2, paid)
}

// G2 — Commit-reveal discipline: every ordering and forgery hole the design
// closes by construction.
func TestGateCommitRevealDiscipline(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	anchor := sample.AccAddress()
	setJurors(t, f, anchor)
	seedAccount(t, f, anchor, 1000)
	agent1, agent2 := sample.AccAddress(), sample.AccAddress()
	seedAccount(t, f, agent1, 0)
	seedAccount(t, f, agent2, 0)
	stranger := sample.AccAddress() // never registered

	const gold, goldSalt = "pass", "gsalt"
	id := postGate(t, f, ms, anchor, 100, gateCommit(gold, goldSalt), 5, 100)
	commitDeadline := int64(100) + types.GateCommitWindowSeconds
	revealDeadline := commitDeadline + types.GateRevealWindowSeconds

	// The poster cannot answer its own gate (self-mining).
	_, err := ms.CommitAnswer(setBlockTime(f.ctx, 200), &types.MsgCommitAnswer{Creator: anchor, GateId: id, Commit: gateCommit("pass", "x")})
	require.Error(t, err, "poster self-answer")
	// Unregistered agents cannot answer (identity is free but must exist).
	_, err = ms.CommitAnswer(setBlockTime(f.ctx, 200), &types.MsgCommitAnswer{Creator: stranger, GateId: id, Commit: gateCommit("pass", "x")})
	require.Error(t, err, "unregistered agent")
	// Malformed commitment is not a commitment.
	_, err = ms.CommitAnswer(setBlockTime(f.ctx, 200), &types.MsgCommitAnswer{Creator: agent1, GateId: id, Commit: "feedc0de"})
	require.Error(t, err, "malformed commit hash")

	// A reveal during the commit window must fail — otherwise later
	// committers could copy it (the entire point of commit-reveal).
	_, err = ms.CommitAnswer(setBlockTime(f.ctx, 200), &types.MsgCommitAnswer{Creator: agent1, GateId: id, Commit: gateCommit("pass", "s1")})
	require.NoError(t, err)
	_, err = ms.RevealAnswer(setBlockTime(f.ctx, 210), &types.MsgRevealAnswer{Creator: agent1, GateId: id, Answer: "pass", Salt: "s1"})
	require.Error(t, err, "reveal while commits are still open")

	// One commit per agent.
	_, err = ms.CommitAnswer(setBlockTime(f.ctx, 220), &types.MsgCommitAnswer{Creator: agent1, GateId: id, Commit: gateCommit("fail:1", "s9")})
	require.Error(t, err, "double commit")

	// Commit after the commit deadline fails.
	_, err = ms.CommitAnswer(setBlockTime(f.ctx, commitDeadline+1), &types.MsgCommitAnswer{Creator: agent2, GateId: id, Commit: gateCommit("pass", "s2")})
	require.Error(t, err, "commit after deadline")

	// Reveal forgery: wrong salt, wrong answer, no commit at all.
	rt := commitDeadline + 10
	_, err = ms.RevealAnswer(setBlockTime(f.ctx, rt), &types.MsgRevealAnswer{Creator: agent1, GateId: id, Answer: "pass", Salt: "WRONG"})
	require.Error(t, err, "salt mismatch")
	_, err = ms.RevealAnswer(setBlockTime(f.ctx, rt), &types.MsgRevealAnswer{Creator: agent1, GateId: id, Answer: "fail:1", Salt: "s1"})
	require.Error(t, err, "answer mismatch")
	_, err = ms.RevealAnswer(setBlockTime(f.ctx, rt), &types.MsgRevealAnswer{Creator: agent2, GateId: id, Answer: "pass", Salt: "s2"})
	require.Error(t, err, "reveal without commit")

	// Settlement gates: too early, wrong gold preimage.
	_, err = ms.SettleGate(setBlockTime(f.ctx, rt), &types.MsgSettleGate{Creator: anchor, GateId: id, GoldAnswer: gold, GoldSalt: goldSalt})
	require.Error(t, err, "settle before reveal deadline")
	_, err = ms.RevealAnswer(setBlockTime(f.ctx, rt+1), &types.MsgRevealAnswer{Creator: agent1, GateId: id, Answer: "pass", Salt: "s1"})
	require.NoError(t, err)
	_, err = ms.RevealAnswer(setBlockTime(f.ctx, rt+2), &types.MsgRevealAnswer{Creator: agent1, GateId: id, Answer: "pass", Salt: "s1"})
	require.Error(t, err, "double reveal")
	_, err = ms.RevealAnswer(setBlockTime(f.ctx, revealDeadline+1), &types.MsgRevealAnswer{Creator: agent1, GateId: id, Answer: "pass", Salt: "s1"})
	require.Error(t, err, "reveal after reveal deadline")
	_, err = ms.SettleGate(setBlockTime(f.ctx, revealDeadline+1), &types.MsgSettleGate{Creator: anchor, GateId: id, GoldAnswer: "fail:1", GoldSalt: "nope"})
	require.Error(t, err, "wrong gold preimage")

	// Settle once, then never again; the settled gate accepts nothing.
	_, err = ms.SettleGate(setBlockTime(f.ctx, revealDeadline+1), &types.MsgSettleGate{Creator: anchor, GateId: id, GoldAnswer: gold, GoldSalt: goldSalt})
	require.NoError(t, err)
	require.Equal(t, uint64(5), balanceOf(t, f, agent1))
	_, err = ms.SettleGate(setBlockTime(f.ctx, revealDeadline+2), &types.MsgSettleGate{Creator: anchor, GateId: id, GoldAnswer: gold, GoldSalt: goldSalt})
	require.Error(t, err, "double settle")
	_, err = ms.CommitAnswer(setBlockTime(f.ctx, revealDeadline+2), &types.MsgCommitAnswer{Creator: agent2, GateId: id, Commit: gateCommit("pass", "s2")})
	require.Error(t, err, "commit on settled gate")
}

// G3 — Live gate: no known verdict exists; the strict-plurality revealed
// answer is the consensus and pays. A tie pays nobody (no coin flip, no
// poster discretion).
func TestLiveGateConsensusPlurality(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	anchor := sample.AccAddress()
	setJurors(t, f, anchor)
	seedAccount(t, f, anchor, 1000)
	agents := make([]string, 5)
	for i := range agents {
		agents[i] = sample.AccAddress()
		seedAccount(t, f, agents[i], 0)
	}

	// gold_commit over the EMPTY answer marks a live gate — but only its salt
	// hash is on-chain, so answerers cannot distinguish it from a decoy.
	id := postGate(t, f, ms, anchor, 100, gateCommit("", "live-salt"), 10, 100)

	answers := []string{"pass", "pass", "pass", "fail:item1", "fail:item1"}
	for i, a := range agents {
		_, err := ms.CommitAnswer(setBlockTime(f.ctx, 200+int64(i)), &types.MsgCommitAnswer{
			Creator: a, GateId: id, Commit: gateCommit(answers[i], fmt.Sprintf("s%d", i)),
		})
		require.NoError(t, err)
	}
	rt := int64(100) + types.GateCommitWindowSeconds + 1
	for i, a := range agents {
		_, err := ms.RevealAnswer(setBlockTime(f.ctx, rt+int64(i)), &types.MsgRevealAnswer{
			Creator: a, GateId: id, Answer: answers[i], Salt: fmt.Sprintf("s%d", i),
		})
		require.NoError(t, err)
	}
	st := int64(100) + types.GateCommitWindowSeconds + types.GateRevealWindowSeconds + 1
	resp, err := ms.SettleGate(setBlockTime(f.ctx, st), &types.MsgSettleGate{
		Creator: anchor, GateId: id, GoldAnswer: "", GoldSalt: "live-salt",
	})
	require.NoError(t, err)
	require.Equal(t, "pass", resp.Consensus)
	for i, a := range agents {
		want := uint64(0)
		if answers[i] == "pass" {
			want = 10
		}
		require.Equal(t, want, balanceOf(t, f, a), "agent %d", i)
	}

	// Tie: 1-1 split → consensus empty, nobody paid.
	id2 := postGate(t, f, ms, anchor, 50_000, gateCommit("", "live-salt-2"), 10, 100)
	_, err = ms.CommitAnswer(setBlockTime(f.ctx, 50_100), &types.MsgCommitAnswer{Creator: agents[0], GateId: id2, Commit: gateCommit("pass", "t0")})
	require.NoError(t, err)
	_, err = ms.CommitAnswer(setBlockTime(f.ctx, 50_100), &types.MsgCommitAnswer{Creator: agents[1], GateId: id2, Commit: gateCommit("fail:item1", "t1")})
	require.NoError(t, err)
	rt2 := int64(50_000) + types.GateCommitWindowSeconds + 1
	_, err = ms.RevealAnswer(setBlockTime(f.ctx, rt2), &types.MsgRevealAnswer{Creator: agents[0], GateId: id2, Answer: "pass", Salt: "t0"})
	require.NoError(t, err)
	_, err = ms.RevealAnswer(setBlockTime(f.ctx, rt2), &types.MsgRevealAnswer{Creator: agents[1], GateId: id2, Answer: "fail:item1", Salt: "t1"})
	require.NoError(t, err)
	st2 := int64(50_000) + types.GateCommitWindowSeconds + types.GateRevealWindowSeconds + 1
	resp2, err := ms.SettleGate(setBlockTime(f.ctx, st2), &types.MsgSettleGate{Creator: anchor, GateId: id2, GoldAnswer: "", GoldSalt: "live-salt-2"})
	require.NoError(t, err)
	require.Equal(t, "", resp2.Consensus, "a tie has no consensus")
	require.Equal(t, uint64(10), balanceOf(t, f, agents[0]), "balance unchanged from gate 1")
	require.Equal(t, uint64(10), balanceOf(t, f, agents[1]), "tie mints nothing")
}

// G4 — Posting guards: emission authority and caps.
func TestGatePostingGuards(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	anchor := sample.AccAddress()
	outsider := sample.AccAddress()
	setJurors(t, f, anchor)
	seedAccount(t, f, anchor, 1000)
	seedAccount(t, f, outsider, 1000)
	agent1, agent2 := sample.AccAddress(), sample.AccAddress()
	seedAccount(t, f, agent1, 0)
	seedAccount(t, f, agent2, 0)

	base := func() *types.MsgPostGate {
		return &types.MsgPostGate{
			Creator: anchor, PayloadUri: "ipfs://x", PayloadHash: hexHash("p"),
			GoldCommit: gateCommit("pass", "s"), Drip: 10, MaxAnswers: 10,
		}
	}
	lc := setBlockTime(f.ctx, 100)

	m := base()
	m.Creator = outsider
	_, err := ms.PostGate(lc, m)
	require.Error(t, err, "drip is minted supply — only anchors may post gates in v1")

	m = base()
	m.Drip = 0
	_, err = ms.PostGate(lc, m)
	require.Error(t, err, "zero drip")
	m = base()
	m.Drip = types.MaxDripPerAnswer + 1
	_, err = ms.PostGate(lc, m)
	require.Error(t, err, "drip above the emission cap")
	m = base()
	m.MaxAnswers = 0
	_, err = ms.PostGate(lc, m)
	require.Error(t, err, "zero max_answers")
	m = base()
	m.MaxAnswers = types.MaxGateAnswers + 1
	_, err = ms.PostGate(lc, m)
	require.Error(t, err, "max_answers above the cap")
	m = base()
	m.PayloadHash = "feedc0de"
	_, err = ms.PostGate(lc, m)
	require.Error(t, err, "malformed payload_hash")
	m = base()
	m.GoldCommit = "nope"
	_, err = ms.PostGate(lc, m)
	require.Error(t, err, "malformed gold_commit")

	// max_answers is enforced at commit time.
	id := postGate(t, f, ms, anchor, 100, gateCommit("pass", "s"), 10, 1)
	_, err = ms.CommitAnswer(setBlockTime(f.ctx, 200), &types.MsgCommitAnswer{Creator: agent1, GateId: id, Commit: gateCommit("pass", "a")})
	require.NoError(t, err)
	_, err = ms.CommitAnswer(setBlockTime(f.ctx, 201), &types.MsgCommitAnswer{Creator: agent2, GateId: id, Commit: gateCommit("pass", "b")})
	require.Error(t, err, "gate is full")
}

// G5 — The anti-sybil arithmetic, as a deterministic micro-sim: across a
// decoy stream with a k=4 answer space, an honest answerer banks the full
// drip while an uninformed constant policy ("always pass" — the lazy
// rubber-stamp sybil) only collects when the decoy happens to agree.
// Coherence pays; presence does not.
func TestGuesserEarnsLessThanHonestAcrossDecoyStream(t *testing.T) {
	f := initFixture(t)
	ms := keeper.NewMsgServerImpl(f.keeper)
	anchor := sample.AccAddress()
	setJurors(t, f, anchor)
	seedAccount(t, f, anchor, 10_000)
	honest, guesser := sample.AccAddress(), sample.AccAddress()
	seedAccount(t, f, honest, 0)
	seedAccount(t, f, guesser, 0)

	space := []string{"pass", "fail:1", "fail:2", "fail:3"}
	const drip = uint64(10)
	const n = 12

	for i := 0; i < n; i++ {
		t0 := int64(100 + i*20_000) // each gate gets its own non-overlapping windows
		gold := space[i%len(space)]
		salt := fmt.Sprintf("gold-salt-%d", i)
		id := postGate(t, f, ms, anchor, t0, gateCommit(gold, salt), drip, 10)

		// Honest agent did the verification work off-chain → answers gold.
		// Guesser always answers "pass" without looking.
		_, err := ms.CommitAnswer(setBlockTime(f.ctx, t0+10), &types.MsgCommitAnswer{Creator: honest, GateId: id, Commit: gateCommit(gold, fmt.Sprintf("h%d", i))})
		require.NoError(t, err)
		_, err = ms.CommitAnswer(setBlockTime(f.ctx, t0+11), &types.MsgCommitAnswer{Creator: guesser, GateId: id, Commit: gateCommit("pass", fmt.Sprintf("g%d", i))})
		require.NoError(t, err)

		rt := t0 + types.GateCommitWindowSeconds + 1
		_, err = ms.RevealAnswer(setBlockTime(f.ctx, rt), &types.MsgRevealAnswer{Creator: honest, GateId: id, Answer: gold, Salt: fmt.Sprintf("h%d", i)})
		require.NoError(t, err)
		_, err = ms.RevealAnswer(setBlockTime(f.ctx, rt+1), &types.MsgRevealAnswer{Creator: guesser, GateId: id, Answer: "pass", Salt: fmt.Sprintf("g%d", i)})
		require.NoError(t, err)

		st := t0 + types.GateCommitWindowSeconds + types.GateRevealWindowSeconds + 1
		_, err = ms.SettleGate(setBlockTime(f.ctx, st), &types.MsgSettleGate{Creator: anchor, GateId: id, GoldAnswer: gold, GoldSalt: salt})
		require.NoError(t, err)
	}

	honestEarned := balanceOf(t, f, honest)
	guesserEarned := balanceOf(t, f, guesser)
	require.Equal(t, drip*n, honestEarned, "honest verification banks every drip")
	require.Equal(t, drip*n/uint64(len(space)), guesserEarned, "constant policy only matches 1/k of decoys")
	require.Less(t, guesserEarned*3, honestEarned, "guessing earns under a third of honest work at k=4")
}
