import { submit } from "../producer.ts";
import type { Keypair, State, Tx } from "../types.ts";

export async function submitTx(state: State, node: Keypair, tx: Tx): Promise<Response> {
  try {
    const { height } = await submit(state, node, tx);
    return Response.json({ ok: true, height });
  } catch (e) {
    return Response.json({ error: (e as Error).message }, { status: 400 });
  }
}
