import { appendBlock } from "./chain.ts";
import { applyTx, validateTx } from "./state.ts";
import type { Keypair, State, Tx } from "./types.ts";

let queue: Promise<unknown> = Promise.resolve();

export function submit(state: State, node: Keypair, tx: Tx): Promise<{ height: number }> {
  const run = async () => {
    const err = await validateTx(state, tx);
    if (err) throw new Error(err);
    const block = await appendBlock([tx], node);
    applyTx(state, tx);
    return { height: block.height };
  };
  const next = queue.then(run, run);
  queue = next.catch(() => undefined);
  return next;
}
