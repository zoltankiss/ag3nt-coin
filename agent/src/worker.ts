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
  const token = (await publicClient.readContract({
    address: escrow,
    abi: jobEscrowAbi,
    functionName: "token",
  })) as `0x${string}`;

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
    try {
      const hash = await wallet.writeContract({
        address: escrow,
        abi: jobEscrowAbi,
        functionName: "claimJob",
        args: [id, BOND_WEI],
      });
      await publicClient.waitForTransactionReceipt({ hash });
    } catch {
      log(`job #${id}: lost the claim race — moving on`);
      return;
    }

    const claimed = await getJob(id);
    if (claimed.worker.toLowerCase() !== me.toLowerCase()) {
      log(`job #${id}: another worker won the claim`);
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
  }
}

function log(m: string) {
  console.log(`[${HERMES_NAME}] ${m}`);
}
