import { existsSync, readdirSync } from "node:fs";
import { mkdir, writeFile } from "node:fs/promises";
import { resolve } from "node:path";
import { CHAIN_DIR } from "./config.ts";
import { canonicalize, sha256Hex, sign } from "./crypto.ts";
import type { Block, Tx, Keypair } from "./types.ts";

const ZERO_HASH = "0".repeat(64);

function blockPath(height: number): string {
  return resolve(CHAIN_DIR, `${String(height).padStart(8, "0")}.json`);
}

export async function ensureChainDir(): Promise<void> {
  if (!existsSync(CHAIN_DIR)) await mkdir(CHAIN_DIR, { recursive: true });
}

export function listHeights(): number[] {
  if (!existsSync(CHAIN_DIR)) return [];
  return readdirSync(CHAIN_DIR)
    .filter((f) => f.endsWith(".json"))
    .map((f) => Number(f.replace(".json", "")))
    .sort((a, b) => a - b);
}

export async function readBlock(height: number): Promise<Block> {
  return (await Bun.file(blockPath(height)).json()) as Block;
}

export async function* iterateBlocks(): AsyncGenerator<Block> {
  for (const h of listHeights()) yield await readBlock(h);
}

export function hashBlock(b: Block): string {
  return sha256Hex(canonicalize({ ...b, sig: "" }));
}

export async function writeGenesis(proposer: Keypair): Promise<Block> {
  await ensureChainDir();
  if (listHeights().length > 0) return await readBlock(0);
  const unsigned: Block = {
    height: 0,
    prev_hash: ZERO_HASH,
    timestamp: new Date().toISOString(),
    txs: [],
    proposer_pubkey: proposer.pub,
    sig: "",
  };
  const sig = await sign(proposer.priv, canonicalize(unsigned));
  const signed: Block = { ...unsigned, sig };
  await writeFile(blockPath(0), JSON.stringify(signed, null, 2));
  return signed;
}

export async function appendBlock(txs: Tx[], proposer: Keypair): Promise<Block> {
  const heights = listHeights();
  const prevHeight = heights[heights.length - 1] ?? -1;
  const prevBlock = await readBlock(prevHeight);
  const prevHash = hashBlock(prevBlock);

  const unsigned: Block = {
    height: prevHeight + 1,
    prev_hash: prevHash,
    timestamp: new Date().toISOString(),
    txs,
    proposer_pubkey: proposer.pub,
    sig: "",
  };
  const sig = await sign(proposer.priv, canonicalize(unsigned));
  const signed: Block = { ...unsigned, sig };
  await writeFile(blockPath(signed.height), JSON.stringify(signed, null, 2));
  return signed;
}
