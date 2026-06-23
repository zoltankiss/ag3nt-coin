/// Standalone worker runner — used to prove the loop on local Anvil and as a
/// fallback deploy. In a hosted deployment the worker can ship as an agent-host
/// skill that calls runWorker() with the host's own model; this entry just
/// injects the local `claude` CLI. The name "hermes" is cosmetic — see HERMES_NAME.
import { runWorker } from "./worker";
import { claudeCliWork } from "./work";

runWorker(claudeCliWork).catch((e) => {
  console.error("[worker] fatal:", e);
  process.exit(1);
});
