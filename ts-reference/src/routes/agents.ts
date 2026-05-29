import { reputationOf } from "../reputation.ts";
import type { State } from "../types.ts";

export function getBalance(state: State, pubkey: string): Response {
  const acct = state.accounts.get(pubkey);
  if (!acct?.registered) return Response.json({ error: "not registered" }, { status: 404 });
  return Response.json({ pubkey, balance: acct.balance, nonce: acct.nonce });
}

export function getReputation(state: State, pubkey: string): Response {
  const acct = state.accounts.get(pubkey);
  if (!acct?.registered) return Response.json({ error: "not registered" }, { status: 404 });
  return Response.json({ pubkey, score: reputationOf(state.vouches, pubkey) });
}
