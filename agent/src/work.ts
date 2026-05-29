/// The "do the work" step for the worker, pluggable so the chain logic in
/// worker.ts stays runtime-agnostic. This implementation shells out to the
/// local `claude` CLI in headless mode. The OpenClaw skill will provide its own
/// DoWork that uses OpenClaw's Claude access instead — same signature.

export type DoWork = (body: string) => Promise<string>;

const PROMPT = (body: string) =>
  `Generate a concise ticket title (max 60 chars) for this task description. ` +
  `Return ONLY the title, no quotes, no prefix.\n\n${body}`;

/// Mirrors add-native-ticket-tracker's generateTitle prompt for fidelity, so a
/// job fulfilled here is indistinguishable from the in-process version it replaces.
export const claudeCliWork: DoWork = async (body) => {
  try {
    const proc = Bun.spawn(["claude", "-p", PROMPT(body)], {
      stdout: "pipe",
      stderr: "pipe",
      signal: AbortSignal.timeout(120_000),
    });
    const out = await new Response(proc.stdout).text();
    const code = await proc.exited;
    const title = out
      .trim()
      .split("\n")
      .map((s) => s.trim())
      .filter(Boolean)[0] ?? "";
    if (code === 0 && title) return stripQuotes(title).slice(0, 60);
  } catch {
    // fall through to the deterministic fallback
  }
  return truncate(body, 60);
};

function stripQuotes(s: string): string {
  return s.replace(/^["'`]+|["'`]+$/g, "").trim();
}

function truncate(text: string, max: number): string {
  const first = text.split("\n")[0].trim();
  if (first.length <= max) return first;
  const cut = first.slice(0, max);
  const sp = cut.lastIndexOf(" ");
  return (sp > max * 0.5 ? cut.slice(0, sp) : cut) + "...";
}
