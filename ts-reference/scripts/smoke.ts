import { canonicalize, generateKeypair, sign } from "../src/crypto.ts";
import type { Tx } from "../src/types.ts";

const BASE = process.env.BASE ?? "http://localhost:3041";

async function post(path: string, body: unknown): Promise<unknown> {
  const r = await fetch(`${BASE}${path}`, {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify(body),
  });
  const j = await r.json();
  if (!r.ok) throw new Error(`${path} → ${r.status}: ${JSON.stringify(j)}`);
  return j;
}

async function get(path: string): Promise<unknown> {
  const r = await fetch(`${BASE}${path}`);
  return r.json();
}

async function signTx<T extends Omit<Tx, "sig">>(priv: string, draft: T): Promise<T & { sig: string }> {
  const sig = await sign(priv, canonicalize({ ...draft, sig: "" }));
  return { ...draft, sig };
}

const alice = await generateKeypair();
const bob = await generateKeypair();
const carol = await generateKeypair();

console.log("alice:", alice.pub.slice(0, 16) + "...");
console.log("bob:  ", bob.pub.slice(0, 16) + "...");
console.log("carol:", carol.pub.slice(0, 16) + "...");

for (const [name, kp] of [["alice", alice], ["bob", bob], ["carol", carol]] as const) {
  const tx = await signTx(kp.priv, { type: "register" as const, pubkey: kp.pub, nonce: 0 as const });
  console.log(`register ${name}:`, await post("/agents", tx));
}

for (const [name, kp] of [["alice", alice], ["bob", bob], ["carol", carol]] as const) {
  const tx = await signTx(kp.priv, { type: "faucet" as const, pubkey: kp.pub, nonce: 1 });
  console.log(`faucet ${name}:`, await post("/faucet", tx));
}

const xfer = await signTx(alice.priv, {
  type: "transfer" as const,
  from: alice.pub,
  to: bob.pub,
  amount: 3000,
  nonce: 2,
});
console.log("alice→bob 3000:", await post("/transfers", xfer));

const vouches: { from: typeof alice; to: typeof bob; w: number }[] = [
  { from: alice, to: bob, w: 80 },
  { from: alice, to: carol, w: 20 },
  { from: bob, to: carol, w: 90 },
  { from: carol, to: bob, w: 50 },
];

let nonce = new Map<string, number>([[alice.pub, 3], [bob.pub, 2], [carol.pub, 2]]);
for (const v of vouches) {
  const n = nonce.get(v.from.pub)!;
  const tx = await signTx(v.from.priv, {
    type: "vouch" as const,
    from: v.from.pub,
    to: v.to.pub,
    weight: v.w,
    nonce: n,
  });
  await post("/vouches", tx);
  nonce.set(v.from.pub, n + 1);
}

console.log("\nbalances:");
for (const [name, kp] of [["alice", alice], ["bob", bob], ["carol", carol]] as const) {
  console.log(` ${name}:`, await get(`/agents/${encodeURIComponent(kp.pub)}/balance`));
}

console.log("\nreputation (uniform until pageRank is implemented):");
for (const [name, kp] of [["alice", alice], ["bob", bob], ["carol", carol]] as const) {
  console.log(` ${name}:`, await get(`/agents/${encodeURIComponent(kp.pub)}/reputation`));
}
