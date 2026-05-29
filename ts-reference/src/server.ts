import { getBalance, getReputation } from "./routes/agents.ts";
import { discovery } from "./routes/discovery.ts";
import { submitTx } from "./routes/submit.ts";
import type { Keypair, State, Tx } from "./types.ts";

async function parseTx(req: Request, type: Tx["type"]): Promise<Tx> {
  const body = (await req.json()) as Record<string, unknown>;
  return { type, ...body } as Tx;
}

export function createServer(state: State, node: Keypair, port: number) {
  return Bun.serve({
    port,
    async fetch(req) {
      const url = new URL(req.url);
      const { pathname } = url;

      if (req.method === "GET" && pathname === "/.well-known/add.json") return discovery();
      if (req.method === "GET" && pathname === "/health") return Response.json({ ok: true });

      const balMatch = pathname.match(/^\/agents\/([^/]+)\/balance$/);
      if (req.method === "GET" && balMatch) return getBalance(state, decodeURIComponent(balMatch[1]!));

      const repMatch = pathname.match(/^\/agents\/([^/]+)\/reputation$/);
      if (req.method === "GET" && repMatch) return getReputation(state, decodeURIComponent(repMatch[1]!));

      if (req.method === "POST" && pathname === "/agents") return submitTx(state, node, await parseTx(req, "register"));
      if (req.method === "POST" && pathname === "/faucet") return submitTx(state, node, await parseTx(req, "faucet"));
      if (req.method === "POST" && pathname === "/transfers") return submitTx(state, node, await parseTx(req, "transfer"));
      if (req.method === "POST" && pathname === "/vouches") return submitTx(state, node, await parseTx(req, "vouch"));

      return Response.json({ error: "not found" }, { status: 404 });
    },
  });
}
