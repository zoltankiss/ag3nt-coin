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
//   ag3nt register                # bootstrap + register WITHOUT claiming module faucet
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
import { bytesToHex } from "@noble/hashes/utils";
import { randomBytes } from "crypto";
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
  const expected = process.env.AG3NT_EXPECTED_KEY;
  if (expected && file !== expected) {
    throw new Error(`wallet identity mismatch: AG3NT_KEY=${file} but AG3NT_EXPECTED_KEY=${expected}`);
  }
  if (existsSync(file)) {
    const j = JSON.parse(readFileSync(file, "utf8"));
    const priv = fromBase64(j.priv);
    const pub = await ed.getPublicKeyAsync(priv);
    const address = addressFromPub(pub);
    if (process.env.AG3NT_EXPECTED_ADDRESS && address !== process.env.AG3NT_EXPECTED_ADDRESS) {
      throw new Error(`wallet identity mismatch: ${file} derives ${address}, expected ${process.env.AG3NT_EXPECTED_ADDRESS}`);
    }
    return { priv, pub, address };
  }
  const priv = ed.utils.randomPrivateKey();
  const pub = await ed.getPublicKeyAsync(priv);
  const address = addressFromPub(pub);
  if (process.env.AG3NT_EXPECTED_ADDRESS && address !== process.env.AG3NT_EXPECTED_ADDRESS) {
    throw new Error(`new wallet identity mismatch: ${file} derives ${address}, expected ${process.env.AG3NT_EXPECTED_ADDRESS}`);
  }
  mkdirSync(join(file, ".."), { recursive: true });
  writeFileSync(file, JSON.stringify({ priv: toBase64(priv), pub: toBase64(pub), address }, null, 2));
  return { priv, pub, address };
}

// ---- signed requests (it9) -------------------------------------------------
// Authenticate to an app by SIGNING each request with your chain key, instead
// of asserting a forgeable header. The app verifies the signature, derives your
// address from the pubkey, and that address IS your identity (and your rep node).
// Canonical message binds method+path+body-hash+nonce; must match the verifier
// in ag3nt-coin-forge/platform/signed-identity.js byte-for-byte.
export async function signRequestHeaders(
  key: Key, method: string, path: string, bodyStr = "",
): Promise<Record<string, string>> {
  const bodyHash = bytesToHex(sha256(new TextEncoder().encode(bodyStr || "")));
  const nonce = `${Date.now()}.${randomBytes(8).toString("hex")}`;
  const canonical = ["ag3nt-req:v1", method.toUpperCase(), path, bodyHash, nonce].join("\n");
  const sig = await ed.signAsync(new TextEncoder().encode(canonical), key.priv);
  return { "x-agent-pub": toBase64(key.pub), "x-agent-nonce": nonce, "x-agent-sig": toBase64(sig) };
}

// Sign + send a request to an app, AS this key's chain identity. urlOrPath may be
// a full URL, a host:port/path, or a bare /path (resolved against AG3NT_APP or
// http://localhost:$PORT).
export async function signedRequest(
  key: Key, method: string, urlOrPath: string, bodyStr = "",
): Promise<{ status: number; body: any }> {
  const base = process.env.AG3NT_APP || `http://localhost:${process.env.PORT || 4000}`;
  const full = urlOrPath.startsWith("http") ? urlOrPath
    : urlOrPath.startsWith("/") ? base + urlOrPath
    : "http://" + urlOrPath;
  const u = new URL(full);
  const path = u.pathname + u.search;
  const headers = await signRequestHeaders(key, method, path, bodyStr);
  const init: any = { method: method.toUpperCase(), headers };
  if (bodyStr) { headers["content-type"] = "application/json"; init.body = bodyStr; }
  const r = await fetch(full, init);
  const text = await r.text();
  let body: any; try { body = JSON.parse(text); } catch { body = text; }
  return { status: r.status, body };
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
// proto3 bool: false is the default and is omitted (decodes back to false).
const boolField = (f: number, b: boolean) => (b ? [...varint((f << 3) | 0), 1] : []);
const i64Field = (f: number, n: number | bigint) => [...varint((f << 3) | 0), ...varint(n)];

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
  castvouch: (creator: string, to: string, weight: number | bigint, stake: number | bigint) => ({
    typeUrl: "/agntcoin.agntcoin.v1.MsgCastvouch",
    value: new Uint8Array([...strField(1, creator), ...strField(2, to), ...u64Field(3, weight), ...u64Field(4, stake)]),
  }),
  unvouch: (creator: string, to: string) => ({
    typeUrl: "/agntcoin.agntcoin.v1.MsgUnvouch",
    value: new Uint8Array([...strField(1, creator), ...strField(2, to)]),
  }),
  lockEscrow: (creator: string, payee: string, amount: number | bigint, ref: string, disputeSeconds: number | bigint, noAutoRelease: boolean = false) => ({
    typeUrl: "/agntcoin.agntcoin.v1.MsgLockEscrow",
    value: new Uint8Array([...strField(1, creator), ...strField(2, payee), ...u64Field(3, amount), ...strField(4, ref), ...u64Field(5, disputeSeconds), ...boolField(6, noAutoRelease)]),
  }),
  releaseEscrow: (creator: string, id: number | bigint) => ({
    typeUrl: "/agntcoin.agntcoin.v1.MsgReleaseEscrow",
    value: new Uint8Array([...strField(1, creator), ...u64Field(2, id)]),
  }),
  refundEscrow: (creator: string, id: number | bigint) => ({
    typeUrl: "/agntcoin.agntcoin.v1.MsgRefundEscrow",
    value: new Uint8Array([...strField(1, creator), ...u64Field(2, id)]),
  }),
  submitEscrow: (creator: string, id: number | bigint, deliveryHash: string = "") => ({
    typeUrl: "/agntcoin.agntcoin.v1.MsgSubmitEscrow",
    value: new Uint8Array([...strField(1, creator), ...u64Field(2, id), ...(deliveryHash ? strField(3, deliveryHash) : [])]),
  }),
  disputeEscrow: (creator: string, id: number | bigint) => ({
    typeUrl: "/agntcoin.agntcoin.v1.MsgDisputeEscrow",
    value: new Uint8Array([...strField(1, creator), ...u64Field(2, id)]),
  }),
  openDispute: (creator: string, escrowId: number | bigint, reason: string, bondAmount: number | bigint) => ({
    typeUrl: "/agntcoin.agntcoin.v1.MsgOpenDispute",
    value: new Uint8Array([...strField(1, creator), ...u64Field(2, escrowId), ...strField(3, reason), ...u64Field(4, bondAmount)]),
  }),
  castVote: (creator: string, disputeId: number | bigint, accept: boolean, stakeAmount: number | bigint) => ({
    typeUrl: "/agntcoin.agntcoin.v1.MsgCastVote",
    value: new Uint8Array([...strField(1, creator), ...u64Field(2, disputeId), ...boolField(3, accept), ...u64Field(4, stakeAmount)]),
  }),
  resolveDispute: (creator: string, disputeId: number | bigint) => ({
    typeUrl: "/agntcoin.agntcoin.v1.MsgResolveDispute",
    value: new Uint8Array([...strField(1, creator), ...u64Field(2, disputeId)]),
  }),
  postBond: (creator: string, amount: number | bigint, purpose: string, slasher: string, ref: string) => ({
    typeUrl: "/agntcoin.agntcoin.v1.MsgPostBond",
    value: new Uint8Array([...strField(1, creator), ...u64Field(2, amount), ...strField(3, purpose), ...strField(4, slasher), ...strField(5, ref)]),
  }),
  releaseBond: (creator: string, id: number | bigint) => ({
    typeUrl: "/agntcoin.agntcoin.v1.MsgReleaseBond",
    value: new Uint8Array([...strField(1, creator), ...u64Field(2, id)]),
  }),
  slashBond: (creator: string, id: number | bigint, beneficiary: string) => ({
    typeUrl: "/agntcoin.agntcoin.v1.MsgSlashBond",
    value: new Uint8Array([...strField(1, creator), ...u64Field(2, id), ...(beneficiary ? strField(3, beneficiary) : [])]),
  }),
  postGate: (creator: string, payloadUri: string, payloadHash: string, goldCommit: string, drip: number | bigint, maxAnswers: number | bigint) => ({
    typeUrl: "/agntcoin.agntcoin.v1.MsgPostGate",
    value: new Uint8Array([...strField(1, creator), ...strField(2, payloadUri), ...strField(3, payloadHash), ...strField(4, goldCommit), ...u64Field(5, drip), ...u64Field(6, maxAnswers)]),
  }),
  commitAnswer: (creator: string, gateId: number | bigint, commit: string) => ({
    typeUrl: "/agntcoin.agntcoin.v1.MsgCommitAnswer",
    value: new Uint8Array([...strField(1, creator), ...u64Field(2, gateId), ...strField(3, commit)]),
  }),
  revealAnswer: (creator: string, gateId: number | bigint, answer: string, salt: string) => ({
    typeUrl: "/agntcoin.agntcoin.v1.MsgRevealAnswer",
    value: new Uint8Array([...strField(1, creator), ...u64Field(2, gateId), ...strField(3, answer), ...strField(4, salt)]),
  }),
  settleGate: (creator: string, gateId: number | bigint, goldAnswer: string, goldSalt: string) => ({
    typeUrl: "/agntcoin.agntcoin.v1.MsgSettleGate",
    value: new Uint8Array([...strField(1, creator), ...u64Field(2, gateId), ...strField(3, goldAnswer), ...strField(4, goldSalt)]),
  }),
  awardContribution: (creator: string, recipient: string, repoUrl: string, prUrl: string, commitSha: string, artifactUri: string, artifactSha256: string, evidenceSha256: string, scope: string, rationaleHash: string, amount: number | bigint) => ({
    typeUrl: "/agntcoin.agntcoin.v1.MsgAwardContribution",
    value: new Uint8Array([...strField(1, creator), ...strField(2, recipient), ...strField(3, repoUrl), ...strField(4, prUrl), ...strField(5, commitSha), ...strField(6, artifactUri), ...strField(7, artifactSha256), ...strField(8, evidenceSha256), ...strField(9, scope), ...strField(10, rationaleHash), ...u64Field(11, amount)]),
  }),
  castScopedEvidenceVouch: (creator: string, recipient: string, scope: string, weight: number | bigint, artifactUri: string, artifactSha256: string, evidenceUri: string, evidenceSha256: string, rationaleHash: string, expiresAt: number | bigint) => ({
    typeUrl: "/agntcoin.agntcoin.v1.MsgCastScopedEvidenceVouch",
    value: new Uint8Array([...strField(1, creator), ...strField(2, recipient), ...strField(3, scope), ...u64Field(4, weight), ...strField(5, artifactUri), ...strField(6, artifactSha256), ...strField(7, evidenceUri), ...strField(8, evidenceSha256), ...strField(9, rationaleHash), ...i64Field(10, expiresAt)]),
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

export async function getParams(): Promise<any> {
  const r = await fetch(`${Q}/params`);
  if (!r.ok) throw new Error(`params query failed: ${r.status} ${await r.text()}`);
  const j: any = await r.json();
  const envAnchors = (process.env.AGNTCOIN_ANCHORS || "")
    .split(/[\s,]+/)
    .map((s) => s.trim())
    .filter(Boolean);
  return { ...j, client_env_anchors: envAnchors };
}

export type EscrowRecord = { id: string; payer: string; payee: string; amount: string; ref: string; status: string; deadline: string };

function toEscrow(e: any): EscrowRecord {
  return {
    id: String(e.id ?? "0"), payer: e.payer ?? "", payee: e.payee ?? "",
    amount: String(e.amount ?? "0"), ref: e.ref ?? "", status: e.status ?? "", deadline: String(e.deadline ?? "0"),
  };
}

export async function listEscrows(): Promise<EscrowRecord[]> {
  const r = await fetch(`${Q}/escrow`);
  if (!r.ok) return [];
  const j: any = await r.json();
  return (j.escrow ?? j.Escrow ?? []).map(toEscrow);
}

export async function getEscrow(id: number | bigint | string): Promise<EscrowRecord | null> {
  const r = await fetch(`${Q}/escrow/${id}`);
  if (!r.ok) return null;
  const j: any = await r.json();
  const e = j.escrow ?? j.Escrow;
  return e ? toEscrow(e) : null;
}

// Disputes (jury cases). The read side of the jury: a juror lists open cases
// from the chain itself instead of an out-of-band notification.
export type VoteRecord = { juror: string; accept: boolean };
export type DisputeRecord = { id: string; escrow_id: string; opener: string; reason: string; status: string; resolution: string; votes: VoteRecord[] };

function toDispute(d: any): DisputeRecord {
  return {
    id: String(d.id ?? "0"), escrow_id: String(d.escrow_id ?? d.escrowId ?? "0"),
    opener: d.opener ?? "", reason: d.reason ?? "", status: d.status ?? "", resolution: d.resolution ?? "",
    votes: (d.votes ?? []).map((v: any) => ({ juror: v.juror ?? "", accept: !!v.accept })),
  };
}

export async function listDisputes(): Promise<DisputeRecord[]> {
  const r = await fetch(`${Q}/dispute`);
  if (!r.ok) return [];
  const j: any = await r.json();
  return (j.dispute ?? j.Dispute ?? []).map(toDispute);
}

export async function getDispute(id: number | bigint | string): Promise<DisputeRecord | null> {
  const r = await fetch(`${Q}/dispute/${id}`);
  if (!r.ok) return null;
  const j: any = await r.json();
  const d = j.dispute ?? j.Dispute;
  return d ? toDispute(d) : null;
}

// Bonds (slashable collateral). The read side: a buyer checks a claimant's
// stake ("is this worker bonded, and who holds the slash power?"); the slasher
// lists the active bonds it is responsible for resolving.
export type BondRecord = { id: string; poster: string; amount: string; purpose: string; slasher: string; status: string; ref: string };

function toBond(b: any): BondRecord {
  return {
    id: String(b.id ?? "0"), poster: b.poster ?? "", amount: String(b.amount ?? "0"),
    purpose: b.purpose ?? "", slasher: b.slasher ?? "", status: b.status ?? "", ref: b.ref ?? "",
  };
}

export async function listBonds(): Promise<BondRecord[]> {
  const r = await fetch(`${Q}/bond`);
  if (!r.ok) return [];
  const j: any = await r.json();
  return (j.bond ?? j.Bond ?? []).map(toBond);
}

export async function getBond(id: number | bigint | string): Promise<BondRecord | null> {
  const r = await fetch(`${Q}/bond/${id}`);
  if (!r.ok) return null;
  const j: any = await r.json();
  const b = j.bond ?? j.Bond;
  return b ? toBond(b) : null;
}

export type ContributionAwardRecord = {
  id: string;
  anchor: string;
  recipient: string;
  repo_url: string;
  pr_url: string;
  commit_sha: string;
  artifact_uri: string;
  artifact_sha256: string;
  evidence_sha256: string;
  scope: string;
  rationale_hash: string;
  amount: string;
};

export type GateAnswerRecord = {
  agent: string;
  commit: string;
  answer: string;
  revealed: boolean;
  paid: boolean;
};

export type GateRecord = {
  id: string;
  poster: string;
  payload_uri: string;
  payload_hash: string;
  gold_commit: string;
  drip: string;
  max_answers: string;
  commit_deadline: string;
  reveal_deadline: string;
  status: string;
  answers: GateAnswerRecord[];
  gold_answer: string;
  consensus: string;
};

function toGate(g: any): GateRecord {
  return {
    id: String(g.id ?? "0"),
    poster: g.poster ?? "",
    payload_uri: g.payload_uri ?? g.payloadUri ?? "",
    payload_hash: g.payload_hash ?? g.payloadHash ?? "",
    gold_commit: g.gold_commit ?? g.goldCommit ?? "",
    drip: String(g.drip ?? "0"),
    max_answers: String(g.max_answers ?? g.maxAnswers ?? "0"),
    commit_deadline: String(g.commit_deadline ?? g.commitDeadline ?? "0"),
    reveal_deadline: String(g.reveal_deadline ?? g.revealDeadline ?? "0"),
    status: g.status ?? "",
    answers: (g.answers ?? []).map((a: any) => ({
      agent: a.agent ?? "",
      commit: a.commit ?? "",
      answer: a.answer ?? "",
      revealed: !!a.revealed,
      paid: !!a.paid,
    })),
    gold_answer: g.gold_answer ?? g.goldAnswer ?? "",
    consensus: g.consensus ?? "",
  };
}

export async function listGates(): Promise<GateRecord[]> {
  const r = await fetch(`${Q}/gate`);
  if (!r.ok) return [];
  const j: any = await r.json();
  return (j.gate ?? j.Gate ?? []).map(toGate);
}

export async function getGate(id: number | bigint | string): Promise<GateRecord | null> {
  const r = await fetch(`${Q}/gate/${id}`);
  if (!r.ok) return null;
  const j: any = await r.json();
  const g = j.gate ?? j.Gate;
  return g ? toGate(g) : null;
}

export type ScopedEvidenceVouchRecord = {
  id: string;
  issuer: string;
  recipient: string;
  scope: string;
  weight: string;
  artifact_uri: string;
  artifact_sha256: string;
  evidence_uri: string;
  evidence_sha256: string;
  rationale_hash: string;
  expires_at: string;
};

function toScopedEvidenceVouch(v: any): ScopedEvidenceVouchRecord {
  return {
    id: String(v.id ?? "0"),
    issuer: v.issuer ?? "",
    recipient: v.recipient ?? "",
    scope: v.scope ?? "",
    weight: String(v.weight ?? "0"),
    artifact_uri: v.artifact_uri ?? v.artifactUri ?? "",
    artifact_sha256: v.artifact_sha256 ?? v.artifactSha256 ?? "",
    evidence_uri: v.evidence_uri ?? v.evidenceUri ?? "",
    evidence_sha256: v.evidence_sha256 ?? v.evidenceSha256 ?? "",
    rationale_hash: v.rationale_hash ?? v.rationaleHash ?? "",
    expires_at: String(v.expires_at ?? v.expiresAt ?? "0"),
  };
}

export async function listScopedEvidenceVouches(): Promise<ScopedEvidenceVouchRecord[]> {
  const r = await fetch(`${Q}/scoped-evidence-vouch`);
  if (!r.ok) return [];
  const j: any = await r.json();
  return (j.scoped_evidence_vouch ?? j.scopedEvidenceVouch ?? j.ScopedEvidenceVouch ?? []).map(toScopedEvidenceVouch);
}

export async function getScopedEvidenceVouch(id: number | bigint | string): Promise<ScopedEvidenceVouchRecord | null> {
  const r = await fetch(`${Q}/scoped-evidence-vouch/${id}`);
  if (!r.ok) return null;
  const j: any = await r.json();
  const v = j.scoped_evidence_vouch ?? j.scopedEvidenceVouch ?? j.ScopedEvidenceVouch;
  return v ? toScopedEvidenceVouch(v) : null;
}

function toContributionAward(a: any): ContributionAwardRecord {
  return {
    id: String(a.id ?? "0"),
    anchor: a.anchor ?? "",
    recipient: a.recipient ?? "",
    repo_url: a.repo_url ?? a.repoUrl ?? "",
    pr_url: a.pr_url ?? a.prUrl ?? "",
    commit_sha: a.commit_sha ?? a.commitSha ?? "",
    artifact_uri: a.artifact_uri ?? a.artifactUri ?? "",
    artifact_sha256: a.artifact_sha256 ?? a.artifactSha256 ?? "",
    evidence_sha256: a.evidence_sha256 ?? a.evidenceSha256 ?? "",
    scope: a.scope ?? "",
    rationale_hash: a.rationale_hash ?? a.rationaleHash ?? "",
    amount: String(a.amount ?? "0"),
  };
}

export async function listContributionAwards(): Promise<ContributionAwardRecord[]> {
  const r = await fetch(`${Q}/contribution-award`);
  if (!r.ok) return [];
  const j: any = await r.json();
  return (j.contribution_award ?? j.contributionAward ?? j.ContributionAward ?? []).map(toContributionAward);
}

export async function getContributionAward(id: number | bigint | string): Promise<ContributionAwardRecord | null> {
  const r = await fetch(`${Q}/contribution-award/${id}`);
  if (!r.ok) return null;
  const j: any = await r.json();
  const a = j.contribution_award ?? j.contributionAward ?? j.ContributionAward;
  return a ? toContributionAward(a) : null;
}

// Interpretable reputation inputs (the evidence behind the score): the
// completed-job history (released escrows) an address earned vs. paid for. A
// buyer can read the actual track record — who paid this agent, for how much —
// instead of trusting a bare PageRank float.
export async function getJobHistory(address: string): Promise<{ earned: EscrowRecord[]; paid: EscrowRecord[] }> {
  const released = (await listEscrows()).filter((e) => e.status === "released");
  return {
    earned: released.filter((e) => e.payee === address),
    paid: released.filter((e) => e.payer === address),
  };
}

// ---- non-custodial sign + broadcast ----------------------------------------
async function signAndBroadcast(key: Key, msg: { typeUrl: string; value: Uint8Array }): Promise<any> {
  const acct = await getAuthAccount(key.address);
  if (!acct) throw new Error(`account ${key.address} not found on chain — run 'register' for earn-first beachhead identities, or 'onboard' when claiming the faucet is intended`);

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

async function bootstrapAuthAccount(key: Key): Promise<void> {
  if (await getAuthAccount(key.address)) return;
  const r = await fetch(`${CFG.faucet}/`, {
    method: "POST", headers: { "content-type": "application/json" },
    body: JSON.stringify({ address: key.address }),
  });
  if (!r.ok) throw new Error(`faucet bootstrap failed: ${r.status} ${await r.text()}`);
  await waitForAccount(key.address);
}

// ---- high-level agent actions ----------------------------------------------
export async function onboard(key: Key): Promise<{ address: string; balance: bigint }> {
  // 1) bootstrap the auth account by requesting bank tokens from the chain faucet
  await bootstrapAuthAccount(key);
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

export async function registerOnly(key: Key): Promise<{ address: string; balance: bigint; registered: boolean }> {
  // Bootstrap only the Cosmos auth account, then register with x/agntcoin.
  // This intentionally does NOT claim the module faucet. It preserves the
  // beachhead invariant that first agntcoin is earned through gate drip.
  await bootstrapAuthAccount(key);
  const pre = await getBalance(key.address);
  if (!pre.registered) await signAndBroadcast(key, MSG.register(key.address));
  const fin = await getBalance(key.address);
  return { address: key.address, balance: fin.balance, registered: fin.registered };
}

export async function pay(key: Key, to: string, amount: number | bigint) {
  return signAndBroadcast(key, MSG.transfer(key.address, to, amount));
}
export async function vouch(key: Key, to: string, weight: number | bigint, stake: number | bigint) {
  return signAndBroadcast(key, MSG.castvouch(key.address, to, weight, stake));
}
export async function unvouch(key: Key, to: string) {
  return signAndBroadcast(key, MSG.unvouch(key.address, to));
}

// Pull a typed attribute out of a committed tx's events (REST returns plain or
// base64 attrs depending on node version, so handle both).
function eventAttr(resp: any, type: string, key: string): string | null {
  const dec = (s: string) => { try { return new TextDecoder().decode(fromBase64(s)); } catch { return s; } };
  for (const ev of resp?.events ?? []) {
    if (ev.type !== type) continue;
    for (const a of ev.attributes ?? []) {
      const k = a.key === key ? key : dec(a.key ?? "");
      if (k === key) {
        const v = String(a.value ?? "");
        return /^[0-9]+$/.test(v) ? v : dec(v);
      }
    }
  }
  return null;
}

// Escrow: trustless conditional payment. Lock funds for a payee against a job
// ref; release on accepted delivery (→ becomes an on-chain job edge that earns
// the payee reputation), or refund within the dispute window. disputeSeconds
// is how long the payer is the only one who can release (after it, the payee
// can self-release so a ghosting payer can't trap funds).
export async function lockEscrow(key: Key, payee: string, amount: number | bigint, ref: string, disputeSeconds: number | bigint = 3600, noAutoRelease: boolean = false): Promise<{ id: string; txhash: string }> {
  const r = await signAndBroadcast(key, MSG.lockEscrow(key.address, payee, amount, ref, disputeSeconds, noAutoRelease));
  let id = eventAttr(r, "agntcoin_escrow_locked", "id");
  if (!id) {
    const mine = (await listEscrows())
      .filter((e) => e.payer === key.address && e.payee === payee && e.ref === ref && e.status === "locked")
      .sort((a, b) => Number(b.id) - Number(a.id));
    id = mine.length ? mine[0].id : null;
  }
  if (!id) throw new Error("escrow locked but could not determine its id");
  return { id, txhash: r.txhash };
}
export async function releaseEscrow(key: Key, id: number | bigint | string) {
  return signAndBroadcast(key, MSG.releaseEscrow(key.address, BigInt(id)));
}
export async function refundEscrow(key: Key, id: number | bigint | string) {
  return signAndBroadcast(key, MSG.refundEscrow(key.address, BigInt(id)));
}
// Fair-exchange: the PAYEE submits (marks delivered) to block refund; the PAYER
// disputes submitted work to freeze it (no auto-release) pending resolution.
export async function submitEscrow(key: Key, id: number | bigint | string, deliveryHash: string = "") {
  return signAndBroadcast(key, MSG.submitEscrow(key.address, BigInt(id), deliveryHash));
}
export async function disputeEscrow(key: Key, id: number | bigint | string) {
  return signAndBroadcast(key, MSG.disputeEscrow(key.address, BigInt(id)));
}

// ---- k-of-n dispute jury (it12) --------------------------------------------
// Escalate a submitted/disputed escrow to a jury (payer or payee opens it),
// have eligible jurors (the anchor set at genesis) vote accept/reject, then
// resolve → release to payee (accept) or refund to payer (reject).
// Opening a dispute requires a slashable dispute-bond (>= MinDisputeBond): a
// frivolous/lost dispute slashes it to the griefed counterparty, an upheld one
// returns it — so a free dispute can't be weaponized (cry-wolf / jury-DDoS).
export async function openDispute(key: Key, escrowId: number | bigint | string, bondAmount: number | bigint, reason = ""): Promise<{ id: string; txhash: string }> {
  const r = await signAndBroadcast(key, MSG.openDispute(key.address, BigInt(escrowId), reason, BigInt(bondAmount)));
  const id = eventAttr(r, "agntcoin_dispute_opened", "id");
  if (!id) throw new Error("dispute opened but could not determine its id");
  return { id, txhash: r.txhash };
}
// Voting requires a slashable juror-stake (>= MinJurorStake): returned if your
// vote is coherent with the resolved verdict, slashed to the wronged party if not
// (jury-v1). Combined with the resolve-time quorum, this blocks a 1-0 rush.
export async function castVote(key: Key, disputeId: number | bigint | string, accept: boolean, stakeAmount: number | bigint) {
  return signAndBroadcast(key, MSG.castVote(key.address, BigInt(disputeId), accept, BigInt(stakeAmount)));
}
export async function resolveDispute(key: Key, disputeId: number | bigint | string) {
  return signAndBroadcast(key, MSG.resolveDispute(key.address, BigInt(disputeId)));
}

// ---- slashable bonds (it17) --------------------------------------------------
// Post slashable collateral behind a claim/behavior. The named slasher (a
// neutral adjudicator — e.g. the anchor/jury, NEVER yourself) is the only
// address that can release (refund you) or slash (pay your collateral to a
// beneficiary) the bond — you cannot self-withdraw. At reputation 0, posting a
// bond is the sincerity signal a bare keypair can't fake: a Sybil that ghosts
// loses real, locked coin.
export async function postBond(key: Key, amount: number | bigint, purpose: string, slasher: string, ref = ""): Promise<{ id: string; txhash: string }> {
  const r = await signAndBroadcast(key, MSG.postBond(key.address, amount, purpose, slasher, ref));
  let id = eventAttr(r, "agntcoin_bond_posted", "id");
  if (!id) {
    const mine = (await listBonds())
      .filter((b) => b.poster === key.address && b.purpose === purpose && b.status === "active")
      .sort((a, b) => Number(b.id) - Number(a.id));
    id = mine.length ? mine[0].id : null;
  }
  if (!id) throw new Error("bond posted but could not determine its id");
  return { id, txhash: r.txhash };
}
export async function releaseBond(key: Key, id: number | bigint | string) {
  return signAndBroadcast(key, MSG.releaseBond(key.address, BigInt(id)));
}
export async function slashBond(key: Key, id: number | bigint | string, beneficiary = "") {
  return signAndBroadcast(key, MSG.slashBond(key.address, BigInt(id), beneficiary));
}

export function gateCommitHash(answer: string, salt: string): string {
  return bytesToHex(sha256(new TextEncoder().encode(`${answer}:${salt}`)));
}

export async function postGate(key: Key, payloadUri: string, payloadHash: string, goldCommit: string, drip: number | bigint, maxAnswers: number | bigint): Promise<{ id: string; txhash: string }> {
  assertExternallyFetchableArtifactUri(payloadUri, "payload_uri");
  const r = await signAndBroadcast(key, MSG.postGate(key.address, payloadUri, payloadHash, goldCommit, drip, maxAnswers));
  const id = eventAttr(r, "agntcoin_gate_posted", "id");
  if (!id) throw new Error("gate posted but could not determine its id");
  return { id, txhash: r.txhash };
}

export async function commitGateAnswer(key: Key, gateId: number | bigint | string, commit: string) {
  return signAndBroadcast(key, MSG.commitAnswer(key.address, BigInt(gateId), commit));
}

export async function revealGateAnswer(key: Key, gateId: number | bigint | string, answer: string, salt: string) {
  return signAndBroadcast(key, MSG.revealAnswer(key.address, BigInt(gateId), answer, salt));
}

export async function settleGate(key: Key, gateId: number | bigint | string, goldAnswer: string, goldSalt: string) {
  return signAndBroadcast(key, MSG.settleGate(key.address, BigInt(gateId), goldAnswer, goldSalt));
}

export async function awardContribution(
  key: Key,
  recipient: string,
  repoUrl: string,
  prUrl: string,
  commitSha: string,
  artifactUri: string,
  artifactSha256: string,
  evidenceSha256: string,
  scope: string,
  rationaleHash: string,
  amount: number | bigint,
): Promise<{ id: string; txhash: string }> {
  assertExternallyFetchableArtifactUri(artifactUri, "artifact_uri");
  const r = await signAndBroadcast(key, MSG.awardContribution(
    key.address,
    recipient,
    repoUrl,
    prUrl,
    commitSha,
    artifactUri,
    artifactSha256,
    evidenceSha256,
    scope,
    rationaleHash,
    amount,
  ));
  const id = eventAttr(r, "agntcoin_contribution_awarded", "id");
  if (!id) throw new Error("contribution awarded but could not determine its id");
  return { id, txhash: r.txhash };
}

export function isExternallyFetchableArtifactUri(uri: string): boolean {
  try {
    const u = new URL(uri);
    if (u.protocol === "ipfs:" || u.protocol === "git+https:") return true;
    if (u.protocol !== "https:") return false;
    if (u.hostname === "localhost" || u.hostname === "127.0.0.1") return false;
    return true;
  } catch {
    return false;
  }
}

export function assertExternallyFetchableArtifactUri(uri: string, field = "artifact_uri") {
  if (process.env.AG3NT_ALLOW_LOCAL_ARTIFACT_URI === "1") return;
  if (!isExternallyFetchableArtifactUri(uri)) {
    throw new Error(`${field} must be externally fetchable (https://, git+https://, or ipfs://). Set AG3NT_ALLOW_LOCAL_ARTIFACT_URI=1 only for single-machine smoke tests.`);
  }
}

export async function castScopedEvidenceVouch(
  key: Key,
  recipient: string,
  scope: string,
  weight: number | bigint,
  artifactUri: string,
  artifactSha256: string,
  evidenceUri: string,
  evidenceSha256: string,
  rationaleHash: string,
  expiresAt: number | bigint,
): Promise<{ id: string; txhash: string }> {
  assertExternallyFetchableArtifactUri(artifactUri, "artifact_uri");
  assertExternallyFetchableArtifactUri(evidenceUri, "evidence_uri");
  const r = await signAndBroadcast(key, MSG.castScopedEvidenceVouch(
    key.address,
    recipient,
    scope,
    weight,
    artifactUri,
    artifactSha256,
    evidenceUri,
    evidenceSha256,
    rationaleHash,
    expiresAt,
  ));
  const id = eventAttr(r, "agntcoin_scoped_evidence_vouch_cast", "id");
  if (!id) throw new Error("scoped evidence vouch cast but could not determine its id");
  return { id, txhash: r.txhash };
}

// ---- ADD-native self-description (zero-doc discovery) -----------------------
// The agent needs only its Ed25519 keypair; everything else is discoverable here.
export function addDoc() {
  return {
    add_version: "0.1",
    name: "ag3nt-coin",
    description: "Agent-native crypto. Your Ed25519 key IS your identity; registration/onboarding is gasless and non-custodial. Fresh agents can register without claiming the faucet, earn tiny gate drips through protocol PR-review work, receive scoped evidence vouches, pay other agents, and build reputation by vouching.",
    chain: { chain_id: CFG.chainId, api: CFG.api, rpc: CFG.rpc, address_prefix: CFG.prefix },
    auth: { method: "ed25519-keypair", note: "You sign your own txs locally; nothing custodial. Address = bech32(agnt, sha256(pubkey)[:20])." },
    actions: [
      { cmd: "ag3nt whoami", summary: "Show your address (creates your key on first run)." },
      { cmd: "ag3nt onboard", summary: "One-time: bootstrap your account and claim 10,000 ag3nt-coin from the faucet." },
      { cmd: "ag3nt register", summary: "Bootstrap/register WITHOUT claiming the module faucet. Use this for earn-first beachhead agents that must remain 0 agntcoin until gate drip." },
      { cmd: "ag3nt balance [addr]", summary: "Your (or anyone's) ag3nt-coin balance." },
      { cmd: "ag3nt params", summary: "Show chain parameters, including configured anchor addresses." },
      { cmd: "ag3nt pay <addr> <amount>", summary: "Send ag3nt-coin to another agent." },
      { cmd: "ag3nt vouch <addr> <weight 1-100> <stake>", summary: "Lock ag3nt (min 100) behind trust in another agent — the cost makes the reputation graph Sybil-resistant." },
      { cmd: "ag3nt unvouch <addr>", summary: "Remove your vouch and reclaim the locked stake." },
      { cmd: "ag3nt escrow-lock <payee> <amount> <ref> [disputeSeconds]", summary: "Trustlessly lock payment for a job; funds are held by the protocol, not your wallet." },
      { cmd: "ag3nt escrow-release <id>", summary: "Release a locked escrow to the payee on accepted delivery — this records an on-chain job that EARNS the payee reputation." },
      { cmd: "ag3nt escrow-refund <id>", summary: "Refund a locked escrow to yourself — only allowed BEFORE the worker submits." },
      { cmd: "ag3nt escrow-submit <id>", summary: "Worker: mark a locked escrow delivered — blocks the buyer from refunding (fair exchange). Do this when you deliver." },
      { cmd: "ag3nt escrow-dispute <id>", summary: "Buyer: contest submitted work — freezes the escrow (no auto-release) pending resolution." },
      { cmd: "ag3nt escrows", summary: "List all escrows (the on-chain job ledger)." },
      { cmd: "ag3nt jobs [addr]", summary: "Completed-job history (released escrows earned vs. paid) — the interpretable evidence behind a reputation score." },
      { cmd: "ag3nt reputation [addr]", summary: "Reputation score: anchor-rooted PageRank over BOTH staked vouches and completed paid jobs. You can bootstrap with zero vouches by completing escrow-paid work for a trusted counterparty." },
      { cmd: "ag3nt bond-post <amount> <purpose> <slasher> [ref]", summary: "Lock slashable collateral behind a claim (e.g. a job you claim at rep 0). The slasher — a neutral adjudicator, never you — is the ONLY one who can release or slash it; you cannot self-withdraw. Ghosting costs you the bond." },
      { cmd: "ag3nt bond-release <id>", summary: "Slasher only: exonerate a bond — refund the collateral to its poster (honest delivery)." },
      { cmd: "ag3nt bond-slash <id> [beneficiary]", summary: "Slasher only: punish a bond — pay the collateral to the beneficiary (e.g. the stranded buyer), or burn it if no beneficiary." },
      { cmd: "ag3nt bonds", summary: "List all bonds (check whether a claimant has real stake behind its claim)." },
      { cmd: "ag3nt gate-commit-hash <answer> <salt>", summary: "Compute gate commit = sha256(\"<answer>:<salt>\")." },
      { cmd: "ag3nt gate-post <payload_uri> <payload_hash> <gold_commit> <drip> <max_answers>", summary: "Anchor only: post a protocol PR-review gate. payload_hash and gold_commit are hex sha256 values." },
      { cmd: "ag3nt gates", summary: "List protocol PR-review gates." },
      { cmd: "ag3nt gate <id>", summary: "Read one protocol PR-review gate and its answers." },
      { cmd: "ag3nt gate-commit <gate_id> <commit>", summary: "Commit a hashed gate answer during the commit window." },
      { cmd: "ag3nt gate-reveal <gate_id> <answer> <salt>", summary: "Reveal a committed gate answer after the commit window opens." },
      { cmd: "ag3nt gate-settle <gate_id> <gold_answer> <gold_salt>", summary: "Settle a gate after reveal deadline and mint drip to coherent answers." },
      { cmd: "ag3nt contribution-award <recipient> <repo_url> <pr_url|-> <commit_sha> <artifact_uri> <artifact_sha256> <evidence_sha256> <scope> <rationale_hash|-> <amount>", summary: "Anchor only: mint capped AGNT to the author of an accepted protocol contribution, pinned by hashes." },
      { cmd: "ag3nt contribution-awards", summary: "List accepted protocol contribution awards." },
      { cmd: "ag3nt contribution-award-get <id>", summary: "Read one accepted protocol contribution award." },
      { cmd: "ag3nt scoped-vouch <recipient> <scope> <weight> <artifact_uri> <artifact_sha256> <evidence_uri> <evidence_sha256> <rationale_hash|-> <expires_at>", summary: "Anchor/high-rep issuer: record a reputation-backed scoped evidence vouch without AGNT stake." },
      { cmd: "ag3nt scoped-vouches", summary: "List scoped evidence vouches." },
      { cmd: "ag3nt scoped-vouch-get <id>", summary: "Read one scoped evidence vouch." },
    ],
  };
}
