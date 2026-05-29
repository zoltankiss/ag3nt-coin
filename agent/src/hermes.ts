/// Standalone worker runner — used to prove the loop on local Anvil and as a
/// fallback deploy. On the macminis the worker ships as an OpenClaw skill that
/// calls runWorker() with OpenClaw's own Claude; this entry just injects the
/// local `claude` CLI. The name "hermes" is cosmetic — see HERMES_NAME.
import { runWorker } from "./worker";
import { claudeCliWork } from "./work";

runWorker(claudeCliWork).catch((e) => {
  console.error("[worker] fatal:", e);
  process.exit(1);
});
