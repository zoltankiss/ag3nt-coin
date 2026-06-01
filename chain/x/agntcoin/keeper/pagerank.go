package keeper

import (
	"math"
	"sort"

	"github.com/zoltankiss/agntcoin/x/agntcoin/types"
)

// pagerankDamping is the standard PageRank damping factor.
const pagerankDamping = 0.85

// pagerankMaxIter caps power iteration so the query is bounded for consensus.
const pagerankMaxIter = 100

// pagerankTolerance is the L1 convergence threshold.
const pagerankTolerance = 1e-9

// PageRank computes weighted PageRank scores for the directed vouch graph.
//
// Input:
//   - vouches: every edge (from, to, weight). Weights are uint64 in [1..100].
//     The keeper deduplicates (from, to) pairs to the latest weight, so each
//     entry is treated as a unique edge.
//
// Output:
//   - map[address]float64 — one score per node. Scores sum to ~1.0 across all
//     nodes.
//
// The transition is weight-normalized: each node splits its influence across
// its out-edges in proportion to edge weight (not edge count). Dangling nodes
// (no out-edges) redistribute their rank uniformly each iteration. Node
// iteration order is sorted so results are deterministic for consensus.
func PageRank(vouches []types.Vouch) map[string]float64 {
	// Collect the node set from both endpoints; a vouched-for-only address
	// still gets a rank.
	nodeSet := map[string]struct{}{}
	for _, v := range vouches {
		nodeSet[v.FromAddr] = struct{}{}
		nodeSet[v.ToAddr] = struct{}{}
	}
	if len(nodeSet) == 0 {
		return map[string]float64{}
	}

	// Deterministic node ordering and index lookup.
	nodes := make([]string, 0, len(nodeSet))
	for n := range nodeSet {
		nodes = append(nodes, n)
	}
	sort.Strings(nodes)
	n := len(nodes)
	idx := make(map[string]int, n)
	for i, addr := range nodes {
		idx[addr] = i
	}

	// Total out-weight per node, used to weight-normalize out-edges.
	outWeight := make([]float64, n)
	for _, v := range vouches {
		// Self-loops are rejected by the module, but guard anyway: a
		// self-loop contributes no useful flow, so skip it.
		if v.FromAddr == v.ToAddr {
			continue
		}
		outWeight[idx[v.FromAddr]] += float64(v.Weight)
	}

	// Build the weight-normalized edge list: for each edge, the share of the
	// source's rank that flows to the target.
	type edge struct {
		from  int
		to    int
		share float64
	}
	edges := make([]edge, 0, len(vouches))
	for _, v := range vouches {
		if v.FromAddr == v.ToAddr {
			continue
		}
		f := idx[v.FromAddr]
		tw := outWeight[f]
		if tw == 0 {
			continue
		}
		edges = append(edges, edge{
			from:  f,
			to:    idx[v.ToAddr],
			share: float64(v.Weight) / tw,
		})
	}

	invN := 1.0 / float64(n)

	// Initialize r = 1/N for all nodes.
	r := make([]float64, n)
	for i := range r {
		r[i] = invN
	}

	next := make([]float64, n)
	for iter := 0; iter < pagerankMaxIter; iter++ {
		// Dangling mass: total rank held by nodes with no out-edges.
		var danglingMass float64
		for i := 0; i < n; i++ {
			if outWeight[i] == 0 {
				danglingMass += r[i]
			}
		}

		// Base term: teleport plus uniformly redistributed dangling mass,
		// then scaled by damping where appropriate.
		// r_{k+1} = (1-d)/N + d * (M^T r_k + danglingMass/N)
		base := (1.0-pagerankDamping)*invN + pagerankDamping*danglingMass*invN
		for i := range next {
			next[i] = base
		}

		// Weighted edge contributions: M^T r_k.
		for _, e := range edges {
			next[e.to] += pagerankDamping * e.share * r[e.from]
		}

		// L1 delta for convergence.
		var delta float64
		for i := 0; i < n; i++ {
			delta += math.Abs(next[i] - r[i])
		}

		r, next = next, r

		if delta < pagerankTolerance {
			break
		}
	}

	out := make(map[string]float64, n)
	for i, addr := range nodes {
		out[addr] = r[i]
	}
	return out
}
