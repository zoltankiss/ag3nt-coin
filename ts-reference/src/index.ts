import { existsSync } from "node:fs";
import { mkdir } from "node:fs/promises";
import { writeGenesis } from "./chain.ts";
import { DATA_DIR, PORT } from "./config.ts";
import { loadOrGenerateNodeKey } from "./crypto.ts";
import { createServer } from "./server.ts";
import { replay } from "./state.ts";

if (!existsSync(DATA_DIR)) await mkdir(DATA_DIR, { recursive: true });
const node = await loadOrGenerateNodeKey();
const genesis = await writeGenesis(node);
const state = await replay();

console.log(`ag3nt-coin node`);
console.log(`  pubkey:  ${node.pub}`);
console.log(`  genesis: height ${genesis.height} at ${genesis.timestamp}`);
console.log(`  accounts replayed: ${state.accounts.size}`);
console.log(`  vouches replayed:  ${state.vouches.length}`);

createServer(state, node, PORT);
console.log(`  listening on http://localhost:${PORT}`);
