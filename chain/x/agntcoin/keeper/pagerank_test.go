package keeper

import (
	"math"
	"testing"

	"github.com/zoltankiss/agntcoin/x/agntcoin/types"
)

const prEpsilon = 1e-6

func sumScores(scores map[string]float64) float64 {
	var s float64
	for _, v := range scores {
		s += v
	}
	return s
}

func TestPageRankEmptyGraph(t *testing.T) {
	scores := PageRank(nil)
	if len(scores) != 0 {
		t.Fatalf("expected empty map for empty graph, got %d entries", len(scores))
	}

	scores = PageRank([]types.Vouch{})
	if len(scores) != 0 {
		t.Fatalf("expected empty map for empty slice, got %d entries", len(scores))
	}
}

func TestPageRankSingleNodeDangling(t *testing.T) {
	// One node that is only vouched-for (the from-node has no other edges).
	// Two nodes total here: a -> b, b dangling. Just assert it doesn't crash
	// and sums to ~1.
	scores := PageRank([]types.Vouch{
		{FromAddr: "a", ToAddr: "b", Stake: 10},
	})
	if got := sumScores(scores); math.Abs(got-1.0) > prEpsilon {
		t.Fatalf("scores should sum to ~1.0, got %v", got)
	}
	if len(scores) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(scores))
	}
	// b receives a's vouch, so b should outrank a.
	if scores["b"] <= scores["a"] {
		t.Fatalf("vouched-for node b (%v) should outrank voucher a (%v)", scores["b"], scores["a"])
	}
}

func TestPageRankTwoNodeMutualSymmetric(t *testing.T) {
	scores := PageRank([]types.Vouch{
		{FromAddr: "a", ToAddr: "b", Stake: 50},
		{FromAddr: "b", ToAddr: "a", Stake: 50},
	})
	if got := sumScores(scores); math.Abs(got-1.0) > prEpsilon {
		t.Fatalf("scores should sum to ~1.0, got %v", got)
	}
	if math.Abs(scores["a"]-scores["b"]) > prEpsilon {
		t.Fatalf("mutual vouch should be symmetric: a=%v b=%v", scores["a"], scores["b"])
	}
	if math.Abs(scores["a"]-0.5) > prEpsilon {
		t.Fatalf("each node should be ~0.5, got a=%v", scores["a"])
	}
}

func TestPageRankStarHubHighest(t *testing.T) {
	// Spokes all vouch for the hub; hub vouches back to nobody (dangling).
	vouches := []types.Vouch{
		{FromAddr: "s1", ToAddr: "hub", Stake: 10},
		{FromAddr: "s2", ToAddr: "hub", Stake: 10},
		{FromAddr: "s3", ToAddr: "hub", Stake: 10},
		{FromAddr: "s4", ToAddr: "hub", Stake: 10},
	}
	scores := PageRank(vouches)

	if got := sumScores(scores); math.Abs(got-1.0) > prEpsilon {
		t.Fatalf("scores should sum to ~1.0, got %v", got)
	}
	for _, spoke := range []string{"s1", "s2", "s3", "s4"} {
		if scores["hub"] <= scores[spoke] {
			t.Fatalf("hub (%v) should outrank spoke %s (%v)", scores["hub"], spoke, scores[spoke])
		}
	}
	// Spokes are symmetric.
	for _, spoke := range []string{"s2", "s3", "s4"} {
		if math.Abs(scores["s1"]-scores[spoke]) > prEpsilon {
			t.Fatalf("spokes should be symmetric: s1=%v %s=%v", scores["s1"], spoke, scores[spoke])
		}
	}
}

func TestPageRankWeightedSplit(t *testing.T) {
	// a splits its vouch: heavily toward b, lightly toward c.
	// b should end up ranked higher than c.
	scores := PageRank([]types.Vouch{
		{FromAddr: "a", ToAddr: "b", Stake: 90},
		{FromAddr: "a", ToAddr: "c", Stake: 10},
	})
	if got := sumScores(scores); math.Abs(got-1.0) > prEpsilon {
		t.Fatalf("scores should sum to ~1.0, got %v", got)
	}
	if scores["b"] <= scores["c"] {
		t.Fatalf("heavily-weighted target b (%v) should outrank c (%v)", scores["b"], scores["c"])
	}
}

func TestPageRankDeterministic(t *testing.T) {
	vouches := []types.Vouch{
		{FromAddr: "z", ToAddr: "a", Stake: 7},
		{FromAddr: "a", ToAddr: "m", Stake: 3},
		{FromAddr: "m", ToAddr: "z", Stake: 11},
	}
	first := PageRank(vouches)
	for i := 0; i < 5; i++ {
		again := PageRank(vouches)
		for k, v := range first {
			if again[k] != v {
				t.Fatalf("non-deterministic result for %s: %v vs %v", k, v, again[k])
			}
		}
	}
}
