import { hexToString, stringToHex, keccak256, formatEther, maxUint256 } from "viem";
import { publicClient, walletFor } from "./clients";
import { ESCROW_ADDRESS, hermesKey, HERMES_NAME, BOND_WEI } from "./config";
import { jobEscrowAbi, ag3ntAbi } from "./abi";
import type { DoWork } from "./work";

const POSTED = 1; // JobEscrow.State.Posted

/// Runtime-agnostic worker. Watches the escrow for open jobs, races to claim
/// them, fulfills the work via the injected `doWork`, and submits the result.
/// Crash-safe and idempotent: all decisions are re-derived from on-chain state,
/// so a restart never double-claims or loses a job. The same function powers the
/// standalone dev runner (hermes.ts) and, later, the OpenClaw skill.
export async function runWorker(doWork: DoWork): Promise<void> {
  const wallet = walletFor(hermesKey());
  const me = wallet.account.address;
  const escrow = ESCROW_ADDRESS;
  // Retry: a freshly-deployed contract can be invisible for a few seconds on an
  // eventually-consistent public RPC.
  const token = (await retry(() =>
    publicClient.readContract({ address: escrow, abi: jobEscrowAbi, functionName: "token" }),
  )) as `0x${string}`;

  log(`worker = ${me}`);
  log(`escrow = ${escrow}`);
  log(`token  = ${token}`);
  await ensureApproval(wallet, token, escrow, me);

  const seen = new Set<bigint>(); // jobs we've already started handling

  async function handle(id: bigint, body: string) {
    if (seen.has(id)) return;
    seen.add(id);

    const job = await getJob(id);
    if (Number(job.state) !== POSTED) return; // already taken/closed

    log(`job #${id}: claiming (bond ${formatEther(BOND_WEI)} AGNT)…`);
    // Win/lose is decided by the claim tx's receipt status — not by a follow-up
    // state read (which can be stale on an eventually-consistent RPC and give a
    // false "lost the race"). If claimJob reverted, someone beat us to it.
    let won = false;
    try {
      const hash = await wallet.writeContract({
        address: escrow,
        abi: jobEscrowAbi,
        functionName: "claimJob",
        args: [id, BOND_WEI],
      });
      const receipt = await publicClient.waitForTransactionReceipt({ hash });
      won = receipt.status === "success";
    } catch {
      won = false;
    }
    if (!won) {
      log(`job #${id}: lost the claim race — moving on`);
      return;
    }

    log(`job #${id}: won. working…`);
    const title = (await doWork(body)).slice(0, 120);
    const result = stringToHex(title);
    const resultHash = keccak256(result);

    const sh = await wallet.writeContract({
      address: escrow,
      abi: jobEscrowAbi,
      functionName: "submitResult",
      args: [id, resultHash, result],
    });
    await publicClient.waitForTransactionReceipt({ hash: sh });
    log(`job #${id}: submitted → "${title}"`);
  }

  // Boot scan: pick up any jobs that are already open (missed while we were down).
  const next = (await publicClient.readContract({
    address: escrow,
    abi: jobEscrowAbi,
    functionName: "nextJobId",
  })) as bigint;
  for (let id = 1n; id <= next; id++) {
    const job = await getJob(id);
    if (Number(job.state) === POSTED) void handle(id, hexToString(job.input));
  }

  // Live watch.
  publicClient.watchContractEvent({
    address: escrow,
    abi: jobEscrowAbi,
    eventName: "JobPosted",
    onLogs: (logs) => {
      for (const l of logs) {
        const { id, input } = l.args as { id: bigint; input: `0x${string}` };
        void handle(id, hexToString(input));
      }
    },
  });

  log("watching for jobs… (ctrl-c to stop)");
  await new Promise<never>(() => {}); // run forever
}

async function getJob(id: bigint) {
  return (await publicClient.readContract({
    address: ESCROW_ADDRESS,
    abi: jobEscrowAbi,
    functionName: "getJob",
    args: [id],
  })) as {
    buyer: `0x${string}`;
    worker: `0x${string}`;
    payment: bigint;
    workerBond: bigint;
    state: number;
    input: `0x${string}`;
    result: `0x${string}`;
  };
}

async function ensureApproval(
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
  if (allowance < BOND_WEI * 1_000n) {
    log("approving escrow to spend ag3nt for bonds…");
    const hash = await wallet.writeContract({
      address: token,
      abi: ag3ntAbi,
      functionName: "approve",
      args: [escrow, maxUint256],
    });
    await publicClient.waitForTransactionReceipt({ hash });
    // Wait until the allowance is visible before we try to claim against it.
    for (let i = 0; i < 20; i++) {
      const a = (await publicClient.readContract({
        address: token,
        abi: ag3ntAbi,
        functionName: "allowance",
        args: [me, escrow],
      })) as bigint;
      if (a >= BOND_WEI) break;
      await new Promise((r) => setTimeout(r, 1_000));
    }
  }
}

/// Retry a read against an eventually-consistent RPC.
async function retry<T>(fn: () => Promise<T>, tries = 12, delayMs = 1_000): Promise<T> {
  let last: unknown;
  for (let i = 0; i < tries; i++) {
    try {
      return await fn();
    } catch (e) {
      last = e;
      await new Promise((r) => setTimeout(r, delayMs));
    }
  }
  throw last;
}

function log(m: string) {
  console.log(`[${HERMES_NAME}] ${m}`);
}
