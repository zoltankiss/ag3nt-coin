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

// JobEdge is a completed, paid job: payer paid payee `amount` ag3nt for accepted
// work (a released escrow). It is a directed trust edge payer→payee — being paid
// for accepted work confers reputation, just like being vouched for, except a
// job edge is backed by spent (not merely locked) ag3nt and a real counterparty
// who chose to pay.
type JobEdge struct {
	Payer  string
	Payee  string
	Amount uint64
}

// repEdge is the unified internal edge: `weight` is the ag3nt a node has
// committed toward another (vouch stake or job payment), all in the same units.
type repEdge struct {
	from   string
	to     string
	weight float64
}

// PageRank computes staked-trust PageRank over the vouch graph with uniform
// teleport (no anchors). Retained for callers/tests that only have vouches.
func PageRank(vouches []types.Vouch) map[string]float64 {
	return Reputation(vouches, nil, nil)
}

// Reputation computes anchor-rooted (personalized) PageRank over the combined
// trust graph: vouch edges (share ∝ locked stake) and completed-job edges
// (share ∝ amount paid). Both are denominated in ag3nt, so a node's outgoing
// influence is split across everyone it has committed ag3nt to — whether by
// locking a vouch stake or by paying for accepted work. This is what lets an
// honest newcomer bootstrap: it earns reputation by *doing accepted work*, with
// no pre-existing vouch.
//
// anchors is the personalization (teleport) set — the trust roots (seeded with
// the founder). The (1-d) teleport mass and all dangling mass flow to the
// anchors, so reputation *originates* at the roots and reaches a node only to
// the extent it is connected, via vouches or jobs, back to an anchor. A Sybil
// ring with no anchor-rooted inflow scores ~0; a reputation-laundering ring can
// only pass along the bounded, per-hop-decaying rank one of its members earned
// from a real anchor-rooted job. If anchors is empty (or none are present in
// the graph), teleport is uniform — classic PageRank — for backward
// compatibility.
//
// Output: map[address]float64; scores sum to ~1.0. Node order is sorted so the
// result is deterministic for consensus.
func Reputation(vouches []types.Vouch, jobs []JobEdge, anchors []string) map[string]float64 {
	// Unify vouches and jobs into weighted directed edges.
	edges := make([]repEdge, 0, len(vouches)+len(jobs))
	for _, v := range vouches {
		// Self-loops contribute no useful flow; zero-stake edges carry none.
		if v.FromAddr == v.ToAddr || v.Stake == 0 {
			continue
		}
		edges = append(edges, repEdge{from: v.FromAddr, to: v.ToAddr, weight: float64(v.Stake)})
	}
	for _, j := range jobs {
		if j.Payer == j.Payee || j.Amount == 0 {
			continue
		}
		edges = append(edges, repEdge{from: j.Payer, to: j.Payee, weight: float64(j.Amount)})
	}

	// Node set from both endpoints plus the anchors, so teleport always has a
	// node to land on even if an anchor has no edges yet.
	nodeSet := map[string]struct{}{}
	for _, e := range edges {
		nodeSet[e.from] = struct{}{}
		nodeSet[e.to] = struct{}{}
	}
	for _, a := range anchors {
		if a != "" {
			nodeSet[a] = struct{}{}
		}
	}
	if len(nodeSet) == 0 {
		return map[string]float64{}
	}

	// Deterministic node ordering and index lookup.
	nodes := make([]string, 0, len(nodeSet))
	for nd := range nodeSet {
		nodes = append(nodes, nd)
	}
	sort.Strings(nodes)
	n := len(nodes)
	idx := make(map[string]int, n)
	for i, addr := range nodes {
		idx[addr] = i
	}

	// Total out-weight per node, used to normalize out-edges.
	outWeight := make([]float64, n)
	for _, e := range edges {
		outWeight[idx[e.from]] += e.weight
	}

	// Normalized edges: the share of the source's rank that flows to the target.
	type fedge struct {
		from  int
		to    int
		share float64
	}
	fedges := make([]fedge, 0, len(edges))
	for _, e := range edges {
		f := idx[e.from]
		tw := outWeight[f]
		if tw == 0 {
			continue
		}
		fedges = append(fedges, fedge{from: f, to: idx[e.to], share: e.weight / tw})
	}

	// Personalization vector p: mass concentrated on the anchors, or uniform if
	// there are no anchors in the graph.
	p := make([]float64, n)
	anchorIdx := make([]int, 0, len(anchors))
	for _, a := range anchors {
		if i, ok := idx[a]; ok {
			anchorIdx = append(anchorIdx, i)
		}
	}
	if len(anchorIdx) == 0 {
		invN := 1.0 / float64(n)
		for i := range p {
			p[i] = invN
		}
	} else {
		share := 1.0 / float64(len(anchorIdx))
		for _, i := range anchorIdx {
			p[i] = share
		}
	}

	// Initialize r = p (mass starts at the teleport distribution).
	r := make([]float64, n)
	copy(r, p)

	next := make([]float64, n)
	for iter := 0; iter < pagerankMaxIter; iter++ {
		// Dangling mass: rank held by nodes with no out-edges.
		var danglingMass float64
		for i := 0; i < n; i++ {
			if outWeight[i] == 0 {
				danglingMass += r[i]
			}
		}

		// Base term: teleport plus dangling mass, both redistributed to the
		// personalization vector p (not uniformly) so reputation stays rooted at
		// the anchors. r_{k+1} = (1-d)*p + d*(M^T r_k + danglingMass*p).
		for i := range next {
			next[i] = (1.0-pagerankDamping)*p[i] + pagerankDamping*danglingMass*p[i]
		}

		// Weighted edge contributions: M^T r_k.
		for _, e := range fedges {
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
