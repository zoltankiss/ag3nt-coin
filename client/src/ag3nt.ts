// ===========================================================================
// ag3nt — ADD-native, NON-CUSTODIAL Ed25519 client for ag3nt-coin.
// ---------------------------------------------------------------------------
// Drop-in for CPDD agents (claude -p + bash): an agent that has nothing but an
// Ed25519 keypair can onboard (get coins) and pay/vouch — with zero knowledge
// of Cosmos. The agent holds its own key and signs its own txs; nothing here is
// custodial. The agent's Ed25519 key IS its chain identity (address derived
// from the pubkey), so it's the same identity it uses across the ADD economy.
//
//   ag3nt whoami                  # show my address (creates a key on first run)
//   ag3nt onboard                 # bootstrap + register + claim faucet
//   ag3nt balance [addr]          # ag3nt-coin balance
//   ag3nt pay <addr> <amount>     # send ag3nt-coin to another agent
//   ag3nt vouch <addr> <weight>   # stake 1..100 trust in another agent (reputation)
//   ag3nt reputation [addr]       # reputation score (PageRank over the vouch graph)
//
// Per-agent identity: set AG3NT_KEY=/path/to/key.json (each CPDD agent its own).
// ===========================================================================
import * as ed from "@noble/ed25519";
import { sha512 } from "@noble/hashes/sha512";
import { sha256 } from "@noble/hashes/sha256";
import { toBech32, toBase64, fromBase64 } from "@cosmjs/encoding";
import { TxBody, AuthInfo, SignerInfo, Fee, SignDoc, TxRaw } from "cosmjs-types/cosmos/tx/v1beta1/tx";
import { SignMode } from "cosmjs-types/cosmos/tx/signing/v1beta1/signing";
import { PubKey as Ed25519PubKey } from "cosmjs-types/cosmos/crypto/ed25519/keys";
import { homedir } from "os";
import { join } from "path";
import { mkdirSync, existsSync, readFileSync, writeFileSync } from "fs";

// @noble/ed25519 v2 needs a sha512 impl for sync ops; we use async, but set it anyway.
ed.etc.sha512Sync = (...m: Uint8Array[]) => sha512(ed.etc.concatBytes(...m));

// ---- config (env-overridable; defaults = local ignite dev chain) ----------
export const CFG = {
  api: process.env.AG3NT_API || "http://localhost:1317",
  rpc: process.env.AG3NT_RPC || "http://localhost:26657",
  faucet: process.env.AG3NT_FAUCET || "http://localhost:4500",
  chainId: process.env.AG3NT_CHAIN_ID || "agntcoin",
  prefix: process.env.AG3NT_PREFIX || "agnt",
  denom: process.env.AG3NT_DENOM || "token", // bank denom used to bootstrap the auth account
  module: "agntcoin",
};
const Q = `${CFG.api}/zoltankiss/agntcoin/agntcoin/v1`; // generated module REST base

// ---- identity --------------------------------------------------------------
export type Key = { priv: Uint8Array; pub: Uint8Array; address: string };

function addressFromPub(pub: Uint8Array): string {
  // Cosmos ed25519 address = first 20 bytes of sha256(pubkey)
  return toBech32(CFG.prefix, sha256(pub).slice(0, 20));
}

export async function loadOrCreateKey(path?: string): Promise<Key> {
  const file = path || process.env.AG3NT_KEY || join(homedir(), ".ag3nt", "key.json");
  if (existsSync(file)) {
    const j = JSON.parse(readFileSync(file, "utf8"));
    const priv = fromBase64(j.priv);
    const pub = await ed.getPublicKeyAsync(priv);
    return { priv, pub, address: addressFromPub(pub) };
  }
  const priv = ed.utils.randomPrivateKey();
  const pub = await ed.getPublicKeyAsync(priv);
  const address = addressFromPub(pub);
  mkdirSync(join(file, ".."), { recursive: true });
  writeFileSync(file, JSON.stringify({ priv: toBase64(priv), pub: toBase64(pub), address }, null, 2));
  return { priv, pub, address };
}

// ---- minimal protobuf encoders for the 4 module Msgs -----------------------
function varint(nIn: number | bigint): number[] {
  let n = BigInt(nIn); const out: number[] = [];
  while (n > 0x7fn) { out.push(Number((n & 0x7fn) | 0x80n)); n >>= 7n; }
  out.push(Number(n)); return out;
}
const strField = (f: number, s: string) => {
  const b = new TextEncoder().encode(s);
  return [...varint((f << 3) | 2), ...varint(b.length), ...b];
};
const u64Field = (f: number, n: number | bigint) => [...varint((f << 3) | 0), ...varint(n)];

const MSG = {
  register: (creator: string) => ({
    typeUrl: "/agntcoin.agntcoin.v1.MsgRegister",
    value: new Uint8Array(strField(1, creator)),
  }),
  faucet: (creator: string) => ({
    typeUrl: "/agntcoin.agntcoin.v1.MsgFaucet",
    value: new Uint8Array(strField(1, creator)),
  }),
  transfer: (creator: string, to: string, amount: number | bigint) => ({
    typeUrl: "/agntcoin.agntcoin.v1.MsgTransfer",
    value: new Uint8Array([...strField(1, creator), ...strField(2, to), ...u64Field(3, amount)]),
  }),
  castvouch: (creator: string, to: string, weight: number | bigint) => ({
    typeUrl: "/agntcoin.agntcoin.v1.MsgCastvouch",
    value: new Uint8Array([...strField(1, creator), ...strField(2, to), ...u64Field(3, weight)]),
  }),
};

// ---- queries ---------------------------------------------------------------
async function getAuthAccount(address: string): Promise<{ accountNumber: bigint; sequence: bigint } | null> {
  const r = await fetch(`${CFG.api}/cosmos/auth/v1beta1/accounts/${address}`);
  if (!r.ok) return null;
  const j: any = await r.json();
  const a = j.account;
  if (!a) return null;
  return { accountNumber: BigInt(a.account_number ?? 0), sequence: BigInt(a.sequence ?? 0) };
}

export async function getBalance(address: string): Promise<{ registered: boolean; balance: bigint }> {
  const r = await fetch(`${Q}/account/${address}`);
  if (!r.ok) return { registered: false, balance: 0n };
  const j: any = await r.json();
  const acct = j.account ?? j.Account ?? j;
  return { registered: !!acct, balance: BigInt(acct?.balance ?? 0) };
}

export async function getReputation(address: string): Promise<string> {
  const r = await fetch(`${Q}/reputation/${address}`);
  if (!r.ok) return "0";
  const j: any = await r.json();
  return String(j.score ?? j.Score ?? "0");
}

// ---- non-custodial sign + broadcast ----------------------------------------
async function signAndBroadcast(key: Key, msg: { typeUrl: string; value: Uint8Array }): Promise<any> {
  const acct = await getAuthAccount(key.address);
  if (!acct) throw new Error(`account ${key.address} not found on chain — run 'onboard' first`);

  const bodyBytes = TxBody.encode(TxBody.fromPartial({ messages: [msg], memo: "" })).finish();
  const pubAny = {
    typeUrl: "/cosmos.crypto.ed25519.PubKey",
    value: Ed25519PubKey.encode({ key: key.pub }).finish(),
  };
  const authInfoBytes = AuthInfo.encode(AuthInfo.fromPartial({
    signerInfos: [SignerInfo.fromPartial({
      publicKey: pubAny,
      modeInfo: { single: { mode: SignMode.SIGN_MODE_DIRECT } },
      sequence: acct.sequence,
    })],
    fee: Fee.fromPartial({ amount: [], gasLimit: 400000n }), // gasless: chain min-gas-price = 0
  })).finish();

  const signBytes = SignDoc.encode(SignDoc.fromPartial({
    bodyBytes, authInfoBytes, chainId: CFG.chainId, accountNumber: acct.accountNumber,
  })).finish();
  const sig = await ed.signAsync(signBytes, key.priv);

  const txBytes = TxRaw.encode(TxRaw.fromPartial({ bodyBytes, authInfoBytes, signatures: [sig] })).finish();
  const res = await fetch(`${CFG.api}/cosmos/tx/v1beta1/txs`, {
    method: "POST", headers: { "content-type": "application/json" },
    body: JSON.stringify({ tx_bytes: toBase64(txBytes), mode: "BROADCAST_MODE_SYNC" }),
  });
  const j: any = await res.json();
  const tx = j.tx_response ?? j;
  if (tx?.code && tx.code !== 0) throw new Error(`mempool rejected tx (code ${tx.code}): ${tx.raw_log}`);
  // SYNC returns after CheckTx; poll until the tx is in a block so state + sequence advance,
  // and so we surface real in-block (DeliverTx) execution errors.
  const hash = tx.txhash;
  for (let i = 0; i < 20; i++) {
    await new Promise((r) => setTimeout(r, 800));
    const q = await fetch(`${CFG.api}/cosmos/tx/v1beta1/txs/${hash}`);
    if (!q.ok) continue;
    const committed = ((await q.json()) as any).tx_response;
    if (!committed) continue;
    if (committed.code && committed.code !== 0)
      throw new Error(`tx failed in block (code ${committed.code}): ${committed.raw_log}`);
    return committed;
  }
  throw new Error(`tx ${hash} not committed within timeout`);
}

async function waitForAccount(address: string, tries = 30): Promise<void> {
  for (let i = 0; i < tries; i++) {
    if (await getAuthAccount(address)) return;
    await new Promise((r) => setTimeout(r, 1000));
  }
  throw new Error("timed out waiting for account creation via faucet");
}

// ---- high-level agent actions ----------------------------------------------
export async function onboard(key: Key): Promise<{ address: string; balance: bigint }> {
  // 1) bootstrap the auth account by requesting bank tokens from the chain faucet
  if (!(await getAuthAccount(key.address))) {
    const r = await fetch(`${CFG.faucet}/`, {
      method: "POST", headers: { "content-type": "application/json" },
      body: JSON.stringify({ address: key.address }),
    });
    if (!r.ok) throw new Error(`faucet bootstrap failed: ${r.status} ${await r.text()}`);
    await waitForAccount(key.address);
  }
  // 2) register with the agntcoin module (idempotent-ish; skip if already registered)
  const pre = await getBalance(key.address);
  if (!pre.registered) await signAndBroadcast(key, MSG.register(key.address));
  // 3) claim the one-shot module faucet (10,000 ag3nt-coin)
  const post = await getBalance(key.address);
  if (post.balance === 0n) {
    try { await signAndBroadcast(key, MSG.faucet(key.address)); } catch (e) { /* already claimed */ }
  }
  const fin = await getBalance(key.address);
  return { address: key.address, balance: fin.balance };
}

export async function pay(key: Key, to: string, amount: number | bigint) {
  return signAndBroadcast(key, MSG.transfer(key.address, to, amount));
}
export async function vouch(key: Key, to: string, weight: number | bigint) {
  return signAndBroadcast(key, MSG.castvouch(key.address, to, weight));
}

// ---- ADD-native self-description (zero-doc discovery) -----------------------
// The agent needs only its Ed25519 keypair; everything else is discoverable here.
export function addDoc() {
  return {
    add_version: "0.1",
    name: "ag3nt-coin",
    description: "Agent-native crypto. Your Ed25519 key IS your identity; onboarding is gasless and non-custodial. Acquire coin via the faucet, pay other agents, and build reputation by vouching.",
    chain: { chain_id: CFG.chainId, api: CFG.api, rpc: CFG.rpc, address_prefix: CFG.prefix },
    auth: { method: "ed25519-keypair", note: "You sign your own txs locally; nothing custodial. Address = bech32(agnt, sha256(pubkey)[:20])." },
    actions: [
      { cmd: "ag3nt whoami", summary: "Show your address (creates your key on first run)." },
      { cmd: "ag3nt onboard", summary: "One-time: bootstrap your account and claim 10,000 ag3nt-coin from the faucet." },
      { cmd: "ag3nt balance [addr]", summary: "Your (or anyone's) ag3nt-coin balance." },
      { cmd: "ag3nt pay <addr> <amount>", summary: "Send ag3nt-coin to another agent." },
      { cmd: "ag3nt vouch <addr> <weight 1-100>", summary: "Stake trust in another agent; feeds the reputation (PageRank) graph." },
      { cmd: "ag3nt reputation [addr]", summary: "Reputation score (weighted PageRank over the vouch graph)." },
    ],
  };
}
