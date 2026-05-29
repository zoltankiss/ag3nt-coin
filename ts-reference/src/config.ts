import { resolve } from "node:path";

export const DATA_DIR = resolve(process.env.DATA_DIR ?? "./data");
export const CHAIN_DIR = resolve(DATA_DIR, "chain");
export const NODE_KEY_PATH = resolve(DATA_DIR, "node.key.json");

export const PORT = Number(process.env.PORT ?? 3041);

export const COIN = 1_000_000;
export const FAUCET_AMOUNT = 10_000;
export const MAX_VOUCH_WEIGHT = 100;
