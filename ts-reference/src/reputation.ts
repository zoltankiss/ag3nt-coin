import type { AgentId, Vouch } from "./types.ts";

/**
 * Compute PageRank scores for the directed weighted vouch graph.
 *
 * Note: in this retired TS prototype the body is a uniform-score placeholder.
 * The production implementation lives in the Cosmos module at
 * chain/x/agntcoin/keeper/pagerank.go (anchor-rooted PageRank).
 *
 * Input:
 *   - `vouches`: array of edges, where `from` vouches for `to` with `weight` in [1..100].
 *     Multiple edges from a single `from` to different `to`s are independent.
 *     (state.ts already de-duplicates (from,to) pairs to the latest weight.)
 *
 * Output:
 *   - Map<AgentId, number>: PageRank score per agent. Scores should sum to 1
 *     (or to the number of agents — pick a convention and document it).
 *
 * Hints, not requirements:
 *   - Standard PageRank: power iteration on the column-stochastic transition matrix
 *     M, with damping d (typically 0.85): r_{k+1} = d * M * r_k + (1-d)/N * 1.
 *   - "Weighted" PageRank: when normalizing a node's out-edges, divide each edge's
 *     weight by the node's total out-weight (not by the count). That's the simplest
 *     way to honor the vouch weights.
 *   - Dangling nodes (agents with vouches in but none out): the standard trick is to
 *     redistribute their rank uniformly across all nodes each iteration. The vouch
 *     graph will have many dangling nodes early on — don't skip this.
 *   - Termination: iterate until L1 delta < epsilon (e.g. 1e-9) or max iterations
 *     (e.g. 100). Both are fine; pick what reads cleanly.
 *   - Agents that appear in `vouches` but not yet as senders need entries in the
 *     output too. Collect the full node set from both endpoints first.
 *
 * Suggested signature kept here so callers compile. Replace the body.
 */
export function pageRank(vouches: Vouch[]): Map<AgentId, number> {
  // Prototype placeholder: return uniform scores so the API doesn't crash.
  // The real anchor-rooted PageRank is implemented in the Go keeper.
  const nodes = new Set<AgentId>();
  for (const v of vouches) {
    nodes.add(v.from);
    nodes.add(v.to);
  }
  const out = new Map<AgentId, number>();
  if (nodes.size === 0) return out;
  const uniform = 1 / nodes.size;
  for (const n of nodes) out.set(n, uniform);
  return out;
}

export function reputationOf(vouches: Vouch[], pubkey: AgentId): number {
  return pageRank(vouches).get(pubkey) ?? 0;
}
