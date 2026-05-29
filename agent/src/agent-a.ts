/// Agent A — the buyer. Stands in for add-native-ticket-tracker's generateTitle:
/// instead of calling Claude in-process, it posts a "title this ticket" job to
/// the marketplace, waits for a worker to fulfill it, judges the result, and
/// settles (release on accept, dispute on reject). One-shot script.
import { hexToString, stringToHex, formatEther, decodeEventLog, maxUint256, type Log } from "viem";
import { publicClient, walletFor } from "./clients";
import { ESCROW_ADDRESS, agentAKey, PAYMENT_WEI } from "./config";
import { jobEscrowAbi, ag3ntAbi } from "./abi";

const STATE = ["None", "Posted", "Claimed", "Submitted", "Released", "Disputed", "Cancelled", "Expired"];
const SUBMITTED = 3;

const DEFAULT_BODY =
  "User reports the app logs them out every time they switch browser tabs on " +
  "Safari. Only happens on macOS, started after the latest deploy. Repro: log " +
  "in, open a second tab, come back — session is gone.";

async function main() {
  const body = process.argv.slice(2).join(" ").trim() || DEFAULT_BODY;
  const wallet = walletFor(agentAKey());
  const me = wallet.account.address;
  const escrow = ESCROW_ADDRESS;
  const token = (await read("token")) as `0x${string}`;

  log(`buyer = ${me}`);
  log(`ticket body: "${body}"`);

  await approveIfNeeded(wallet, token, escrow, me);

  log(`posting job, locking ${formatEther(PAYMENT_WEI)} AGNT in escrow…`);
  const postHash = await wallet.writeContract({
    address: escrow,
    abi: jobEscrowAbi,
    functionName: "postJob",
    args: [PAYMENT_WEI, stringToHex(body)],
  });
  const receipt = await publicClient.waitForTransactionReceipt({ hash: postHash });
  const id = jobIdFromLogs(receipt.logs);
  log(`posted job #${id}. waiting for a worker to claim + fulfill…`);

  // Poll until a worker submits (claim race + real Claude work take time).
  const deadline = Date.now() + 240_000;
  let job = await getJob(id);
  while (Date.now() < deadline && Number(job.state) < SUBMITTED) {
    await sleep(2_000);
    job = await getJob(id);
  }
  const st = Number(job.state);
  if (st !== SUBMITTED) {
    log(`gave up — job is in state ${STATE[st] ?? st} (no submission in time)`);
    process.exit(1);
  }

  const title = hexToString(job.result);
  log(`worker ${job.worker} submitted: "${title}"`);

  // Judge. Verification is cheap for titles: non-empty and a sane length.
  const accept = title.trim().length > 0 && title.length <= 80;
  if (accept) {
    log("result looks good → releasing payment");
    await send("release", [id]);
    log(`✅ settled. worker paid ${formatEther(job.payment)} AGNT + a minted reward.`);
  } else {
    log("result looks wrong → disputing (payment refunded)");
    await send("dispute", [id]);
    log("⚠️ disputed.");
  }

  const bal = (await publicClient.readContract({
    address: token,
    abi: ag3ntAbi,
    functionName: "balanceOf",
    args: [job.worker],
  })) as bigint;
  log(`worker balance now: ${formatEther(bal)} AGNT`);

  // helpers bound to this run -------------------------------------------------
  async function send(fn: "release" | "dispute", args: unknown[]) {
    const h = await wallet.writeContract({ address: escrow, abi: jobEscrowAbi, functionName: fn, args });
    await publicClient.waitForTransactionReceipt({ hash: h });
  }
}

async function read(fn: string, args: unknown[] = []) {
  return publicClient.readContract({ address: ESCROW_ADDRESS, abi: jobEscrowAbi, functionName: fn as never, args: args as never });
}
async function getJob(id: bigint) {
  return (await read("getJob", [id])) as {
    worker: `0x${string}`;
    payment: bigint;
    state: number;
    result: `0x${string}`;
  };
}
async function approveIfNeeded(
  wallet: ReturnType<typeof walletFor>,
  token: `0x${string}`,
  escrow: `0x${string}`,
  me: `0x${string}`,
) {
  const allowance = (await publicClient.readContract({
    address: token,
    abi: ag3ntAbi,
    functionName: "allowance",
    args: [me, escrow],
  })) as bigint;
  if (allowance < PAYMENT_WEI) {
    log("approving escrow to spend ag3nt…");
    const h = await wallet.writeContract({ address: token, abi: ag3ntAbi, functionName: "approve", args: [escrow, maxUint256] });
    await publicClient.waitForTransactionReceipt({ hash: h });
  }
}
function jobIdFromLogs(logs: Log[]): bigint {
  for (const l of logs) {
    try {
      const d = decodeEventLog({ abi: jobEscrowAbi, data: l.data, topics: l.topics });
      if (d.eventName === "JobPosted") return (d.args as { id: bigint }).id;
    } catch {
      // not our event
    }
  }
  throw new Error("no JobPosted event in receipt");
}
const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));
function log(m: string) {
  console.log(`[agent-a] ${m}`);
}

main().catch((e) => {
  console.error("[agent-a] fatal:", e);
  process.exit(1);
});
