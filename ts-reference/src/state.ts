import { FAUCET_AMOUNT, MAX_VOUCH_WEIGHT } from "./config.ts";
import { canonicalize, verify } from "./crypto.ts";
import { iterateBlocks } from "./chain.ts";
import type { AccountState, State, Tx } from "./types.ts";
import { senderOf } from "./types.ts";

export function emptyState(): State {
  return { accounts: new Map(), vouches: [] };
}

function getOrInit(state: State, id: string): AccountState {
  let a = state.accounts.get(id);
  if (!a) {
    a = { balance: 0, nonce: 0, registered: false, faucetClaimed: false };
    state.accounts.set(id, a);
  }
  return a;
}

export function canonicalForSig(tx: Tx): string {
  return canonicalize({ ...tx, sig: "" });
}

export async function validateTx(state: State, tx: Tx): Promise<string | null> {
  const sender = senderOf(tx);
  const sigOk = await verify(sender, canonicalForSig(tx), tx.sig);
  if (!sigOk) return "bad signature";

  const acct = state.accounts.get(sender);

  if (tx.type === "register") {
    if (tx.nonce !== 0) return "register nonce must be 0";
    if (acct?.registered) return "already registered";
    return null;
  }

  if (!acct?.registered) return "sender not registered";
  if (tx.nonce !== acct.nonce) return `bad nonce: expected ${acct.nonce}, got ${tx.nonce}`;

  if (tx.type === "faucet") {
    if (acct.faucetClaimed) return "faucet already claimed";
    return null;
  }

  if (tx.type === "transfer") {
    if (tx.amount <= 0) return "amount must be positive";
    if (tx.amount > acct.balance) return "insufficient balance";
    const to = state.accounts.get(tx.to);
    if (!to?.registered) return "recipient not registered";
    return null;
  }

  if (tx.type === "vouch") {
    if (tx.weight < 1 || tx.weight > MAX_VOUCH_WEIGHT) return `weight must be 1..${MAX_VOUCH_WEIGHT}`;
    if (tx.from === tx.to) return "cannot vouch for self";
    const to = state.accounts.get(tx.to);
    if (!to?.registered) return "vouchee not registered";
    return null;
  }

  return "unknown tx type";
}

export function applyTx(state: State, tx: Tx): void {
  const sender = senderOf(tx);
  const acct = getOrInit(state, sender);

  switch (tx.type) {
    case "register":
      acct.registered = true;
      acct.nonce = 1;
      return;

    case "faucet":
      acct.balance += FAUCET_AMOUNT;
      acct.faucetClaimed = true;
      acct.nonce += 1;
      return;

    case "transfer": {
      const to = getOrInit(state, tx.to);
      acct.balance -= tx.amount;
      to.balance += tx.amount;
      acct.nonce += 1;
      return;
    }

    case "vouch": {
      const idx = state.vouches.findIndex((v) => v.from === tx.from && v.to === tx.to);
      const entry = { from: tx.from, to: tx.to, weight: tx.weight };
      if (idx >= 0) state.vouches[idx] = entry;
      else state.vouches.push(entry);
      acct.nonce += 1;
      return;
    }
  }
}

export async function replay(): Promise<State> {
  const state = emptyState();
  for await (const block of iterateBlocks()) {
    for (const tx of block.txs) applyTx(state, tx);
  }
  return state;
}
