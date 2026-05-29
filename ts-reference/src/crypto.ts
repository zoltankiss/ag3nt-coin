import * as ed from "@noble/ed25519";
import { sha512 } from "@noble/hashes/sha512";
import { existsSync } from "node:fs";
import { mkdir, writeFile } from "node:fs/promises";
import { dirname } from "node:path";
import { NODE_KEY_PATH } from "./config.ts";

ed.etc.sha512Sync = (...m) => sha512(ed.etc.concatBytes(...m));

export type Keypair = { pub: string; priv: string };

const b64 = {
  enc: (b: Uint8Array) => Buffer.from(b).toString("base64"),
  dec: (s: string) => new Uint8Array(Buffer.from(s, "base64")),
};

export function canonicalize(obj: unknown): string {
  if (obj === null || typeof obj !== "object") return JSON.stringify(obj);
  if (Array.isArray(obj)) return "[" + obj.map(canonicalize).join(",") + "]";
  const o = obj as Record<string, unknown>;
  const keys = Object.keys(o).sort();
  return "{" + keys.map((k) => JSON.stringify(k) + ":" + canonicalize(o[k])).join(",") + "}";
}

export function sha256Hex(bytes: Uint8Array | string): string {
  const h = new Bun.CryptoHasher("sha256");
  h.update(bytes);
  return h.digest("hex");
}

export async function generateKeypair(): Promise<Keypair> {
  const priv = ed.utils.randomPrivateKey();
  const pub = await ed.getPublicKeyAsync(priv);
  return { pub: b64.enc(pub), priv: b64.enc(priv) };
}

export async function sign(privB64: string, msg: string): Promise<string> {
  const sig = await ed.signAsync(new TextEncoder().encode(msg), b64.dec(privB64));
  return b64.enc(sig);
}

export async function verify(pubB64: string, msg: string, sigB64: string): Promise<boolean> {
  try {
    return await ed.verifyAsync(b64.dec(sigB64), new TextEncoder().encode(msg), b64.dec(pubB64));
  } catch {
    return false;
  }
}

export async function loadOrGenerateNodeKey(): Promise<Keypair> {
  if (existsSync(NODE_KEY_PATH)) {
    const f = Bun.file(NODE_KEY_PATH);
    return (await f.json()) as Keypair;
  }
  await mkdir(dirname(NODE_KEY_PATH), { recursive: true });
  const kp = await generateKeypair();
  await writeFile(NODE_KEY_PATH, JSON.stringify(kp, null, 2), { mode: 0o600 });
  return kp;
}
