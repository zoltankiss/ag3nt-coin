package keeper

import (
	"math"
	"os"
	"testing"

	"github.com/zoltankiss/agntcoin/x/agntcoin/types"
)

// TestResolveAnchors: Params win; env is the first dev fallback; parsing accepts
// comma/whitespace separation.
func TestResolveAnchors(t *testing.T) {
	t.Setenv("AGNTCOIN_ANCHORS", "agnt1aaa, agnt1bbb")
	t.Setenv("AGNTCOIN_ANCHORS_FILE", "/nonexistent")

	// Params take precedence over the dev fallback.
	if got := resolveAnchors([]string{"agnt1param"}); len(got) != 1 || got[0] != "agnt1param" {
		t.Fatalf("params should win, got %v", got)
	}
	// Empty params → env fallback, parsed and trimmed.
	got := resolveAnchors(nil)
	if len(got) != 2 || got[0] != "agnt1aaa" || got[1] != "agnt1bbb" {
		t.Fatalf("env fallback parse failed: %v", got)
	}
	// No params, no env, no file → nil.
	os.Unsetenv("AGNTCOIN_ANCHORS")
	if got := resolveAnchors(nil); got != nil {
		t.Fatalf("expected nil with no source, got %v", got)
	}
}

// These tests pin the two properties the job-graph + anchor-rooting design
// exists to provide:
//   1. Bootstrap: an honest newcomer with NO vouches earns reputation purely by
//      completing a paid job for an anchor-connected counterparty.
//   2. Sybil/laundering resistance: reputation originates at the anchors, so a
//      disconnected ring scores ~0, and a ring that earns one real job can only
//      pass along bounded, decaying, dilutable reputation.

const founder = "founder"

// TestReputationNewcomerBootstrap: a newcomer earns standing from a single
// founder-paid job, with zero vouches. This is the cold-start fix.
func TestReputationNewcomerBootstrap(t *testing.T) {
	jobs := []JobEdge{{Payer: founder, Payee: "newcomer", Amount: 1000}}
	scores := Reputation(nil, jobs, []string{founder})

	if got := sumScores(scores); math.Abs(got-1.0) > prEpsilon {
		t.Fatalf("scores should sum to ~1.0, got %v", got)
	}
	if scores["newcomer"] <= 0 {
		t.Fatalf("newcomer should earn positive reputation from a founder job, got %v", scores["newcomer"])
	}
	// A registered stranger with no edges is not in the graph at all → no
	// reputation conferred by merely existing.
	if _, ok := scores["stranger"]; ok {
		t.Fatalf("a node with no edges should not appear in the graph")
	}
}

// TestReputationDisconnectedSybilStarved: a Sybil ring with no path back to an
// anchor accrues essentially zero reputation, no matter how much it vouches for
// and pays itself.
func TestReputationDisconnectedSybilStarved(t *testing.T) {
	// Honest, anchor-rooted side.
	jobs := []JobEdge{
		{Payer: founder, Payee: "alice", Amount: 1000},
		// Disconnected Sybil ring trading reputation among itself.
		{Payer: "s0", Payee: "s1", Amount: 100000},
		{Payer: "s1", Payee: "s2", Amount: 100000},
		{Payer: "s2", Payee: "s0", Amount: 100000},
	}
	vouches := []types.Vouch{
		{FromAddr: "s0", ToAddr: "s1", Stake: 100000},
		{FromAddr: "s1", ToAddr: "s2", Stake: 100000},
		{FromAddr: "s2", ToAddr: "s0", Stake: 100000},
	}
	scores := Reputation(vouches, jobs, []string{founder})

	for _, s := range []string{"s0", "s1", "s2"} {
		if scores[s] > prEpsilon {
			t.Fatalf("disconnected Sybil %s should score ~0, got %v", s, scores[s])
		}
		if scores[s] >= scores["alice"] {
			t.Fatalf("Sybil %s (%v) must not outrank anchor-rooted alice (%v)", s, scores[s], scores["alice"])
		}
	}
}

// TestReputationLaunderingBounded: a ring that earns ONE real founder job and
// then self-deals can pass reputation to its Sybils, but only bounded, decayed,
// and dilutable amounts — never matching a directly-earned reputation.
func TestReputationLaunderingBounded(t *testing.T) {
	// s0 earns a real job from the founder, then launders to s1, s2.
	launderTo2 := Reputation(nil, []JobEdge{
		{Payer: founder, Payee: "s0", Amount: 1000},
		{Payer: "s0", Payee: "s1", Amount: 1000},
		{Payer: "s0", Payee: "s2", Amount: 1000},
	}, []string{founder})

	// (a) Laundering passes *some* reputation (we don't claim it's impossible).
	if launderTo2["s1"] <= 0 {
		t.Fatalf("expected some laundered reputation to s1, got %v", launderTo2["s1"])
	}
	// (b) Per-hop decay: a laundered Sybil ranks below the member that earned it.
	if launderTo2["s1"] >= launderTo2["s0"] {
		t.Fatalf("laundered s1 (%v) must rank below earner s0 (%v)", launderTo2["s1"], launderTo2["s0"])
	}

	// (c) Dilution: spreading the same earned reputation across MORE Sybils
	// gives each less.
	launderTo6 := Reputation(nil, []JobEdge{
		{Payer: founder, Payee: "s0", Amount: 1000},
		{Payer: "s0", Payee: "s1", Amount: 1000},
		{Payer: "s0", Payee: "s2", Amount: 1000},
		{Payer: "s0", Payee: "s3", Amount: 1000},
		{Payer: "s0", Payee: "s4", Amount: 1000},
		{Payer: "s0", Payee: "s5", Amount: 1000},
		{Payer: "s0", Payee: "s6", Amount: 1000},
	}, []string{founder})
	if launderTo6["s1"] >= launderTo2["s1"] {
		t.Fatalf("diluting across 6 Sybils should lower each (%v) vs 2 Sybils (%v)", launderTo6["s1"], launderTo2["s1"])
	}

	// (d) A laundered Sybil never beats a newcomer who did the SAME direct
	// founder job. Honest direct work dominates laundered work.
	honest := Reputation(nil, []JobEdge{
		{Payer: founder, Payee: "alice", Amount: 1000},
	}, []string{founder})
	if launderTo2["s1"] >= honest["alice"] {
		t.Fatalf("laundered s1 (%v) must not beat directly-paid newcomer alice (%v)", launderTo2["s1"], honest["alice"])
	}
}

// TestReputationVouchAndJobCombine: vouch stake and job payment are both ag3nt,
// so a node accrues from both kinds of edge in the same units.
func TestReputationVouchAndJobCombine(t *testing.T) {
	// founder vouches for v and pays w a job; both should outrank an unrelated
	// anchor-disconnected node.
	vouches := []types.Vouch{{FromAddr: founder, ToAddr: "v", Stake: 500}}
	jobs := []JobEdge{{Payer: founder, Payee: "w", Amount: 500}}
	scores := Reputation(vouches, jobs, []string{founder})

	if scores["v"] <= 0 || scores["w"] <= 0 {
		t.Fatalf("both vouched (v=%v) and paid (w=%v) nodes should have reputation", scores["v"], scores["w"])
	}
	// Equal commitment (500 stake vs 500 paid) split evenly from the same source
	// → symmetric.
	if math.Abs(scores["v"]-scores["w"]) > prEpsilon {
		t.Fatalf("equal vouch-stake and job-amount from the same source should be symmetric: v=%v w=%v", scores["v"], scores["w"])
	}
}

// TestReputationEmptyAnchorsUniform: with no anchors, Reputation reduces to the
// classic uniform-teleport PageRank (PageRank() is the wrapper).
func TestReputationEmptyAnchorsUniform(t *testing.T) {
	vouches := []types.Vouch{
		{FromAddr: "a", ToAddr: "b", Stake: 50},
		{FromAddr: "b", ToAddr: "a", Stake: 50},
	}
	viaReputation := Reputation(vouches, nil, nil)
	viaPageRank := PageRank(vouches)
	for k, v := range viaPageRank {
		if math.Abs(viaReputation[k]-v) > prEpsilon {
			t.Fatalf("empty-anchor Reputation must equal PageRank for %s: %v vs %v", k, viaReputation[k], v)
		}
	}
}
