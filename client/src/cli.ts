#!/usr/bin/env bun
// ag3nt CLI — the drop-in surface a CPDD agent (or a human) calls.
import { loadOrCreateKey, onboard, pay, vouch, unvouch, lockEscrow, releaseEscrow, refundEscrow, submitEscrow, disputeEscrow, openDispute, castVote, resolveDispute, listEscrows, listDisputes, getDispute, postBond, releaseBond, slashBond, listBonds, getBond, getJobHistory, getBalance, getReputation, addDoc, signedRequest, signRequestHeaders, CFG } from "./ag3nt";

const [cmd, ...args] = process.argv.slice(2);
const key = await loadOrCreateKey();
const out = (o: any) => console.log(JSON.stringify(o, (_, v) => (typeof v === "bigint" ? v.toString() : v), 2));

try {
  switch (cmd) {
    case "whoami":
      out({ address: key.address, chain: CFG.chainId, api: CFG.api }); break;
    case "onboard":
      out(await onboard(key)); break;
    case "balance": {
      const b = await getBalance(args[0] || key.address);
      out({ address: args[0] || key.address, registered: b.registered, balance: b.balance }); break;
    }
    case "pay": {
      if (args.length < 2) throw new Error("usage: ag3nt pay <addr> <amount>");
      const r = await pay(key, args[0], BigInt(args[1]));
      out({ ok: true, from: key.address, to: args[0], amount: args[1], txhash: r.txhash }); break;
    }
    case "vouch": {
      if (args.length < 3) throw new Error("usage: ag3nt vouch <addr> <weight 1-100> <stake>  (the stake locks ag3nt behind the vouch; min 100)");
      const r = await vouch(key, args[0], BigInt(args[1]), BigInt(args[2]));
      out({ ok: true, from: key.address, to: args[0], weight: args[1], stake: args[2], txhash: r.txhash }); break;
    }
    case "unvouch": {
      if (args.length < 1) throw new Error("usage: ag3nt unvouch <addr>  (removes your vouch, returns the locked stake)");
      const r = await unvouch(key, args[0]);
      out({ ok: true, from: key.address, to: args[0], txhash: r.txhash }); break;
    }
    case "escrow-lock": {
      // --jury-bound (alias --no-auto-release): mark the escrow jury-bound so the
      // worker can't self-release on deadline — un-verifiable work settles via the
      // jury (it13 #19). Lock no-test/prose tasks this way.
      const flags = args.filter((a: string) => a.startsWith("--"));
      const pos = args.filter((a: string) => !a.startsWith("--"));
      if (pos.length < 3) throw new Error("usage: ag3nt escrow-lock <payee> <amount> <ref> [disputeSeconds] [--jury-bound]");
      const juryBound = flags.includes("--jury-bound") || flags.includes("--no-auto-release");
      const r = await lockEscrow(key, pos[0], BigInt(pos[1]), pos[2], pos[3] ? BigInt(pos[3]) : 3600n, juryBound);
      out({ ok: true, id: r.id, payer: key.address, payee: pos[0], amount: pos[1], ref: pos[2], jury_bound: juryBound, txhash: r.txhash }); break;
    }
    case "escrow-release": {
      if (args.length < 1) throw new Error("usage: ag3nt escrow-release <id>");
      const r = await releaseEscrow(key, args[0]);
      out({ ok: true, released: args[0], by: key.address, txhash: r.txhash }); break;
    }
    case "escrow-refund": {
      if (args.length < 1) throw new Error("usage: ag3nt escrow-refund <id>");
      const r = await refundEscrow(key, args[0]);
      out({ ok: true, refunded: args[0], by: key.address, txhash: r.txhash }); break;
    }
    case "escrow-submit": {
      // escrow-submit <id> [delivery_hash] [--hash-file <path>]
      // The optional hex sha256 pins the delivered artifact on-chain so the jury's
      // exhibit is tamper-evident (it13). Pass the hash the platform returned on
      // deliver, or --hash-file to hash the delivered file locally.
      const hf = args.indexOf("--hash-file");
      let deliveryHash = "";
      if (hf >= 0 && args[hf + 1]) {
        const { createHash } = await import("node:crypto");
        const fs = await import("node:fs");
        deliveryHash = createHash("sha256").update(fs.readFileSync(args[hf + 1])).digest("hex");
      } else if (args[1] && !args[1].startsWith("--")) {
        deliveryHash = args[1];
      }
      if (args.length < 1) throw new Error("usage: ag3nt escrow-submit <id> [delivery_hash] [--hash-file <path>]  (worker marks delivered; blocks refund; hash pins the exhibit)");
      const r = await submitEscrow(key, args[0], deliveryHash);
      out({ ok: true, submitted: args[0], delivery_hash: deliveryHash || null, by: key.address, txhash: r.txhash }); break;
    }
    case "escrow-dispute": {
      if (args.length < 1) throw new Error("usage: ag3nt escrow-dispute <id>  (buyer contests submitted work; freezes it)");
      const r = await disputeEscrow(key, args[0]);
      out({ ok: true, disputed: args[0], by: key.address, txhash: r.txhash }); break;
    }
    case "dispute-open": {
      if (args.length < 2) throw new Error("usage: ag3nt dispute-open <escrow_id> <bond> [reason]  (escalate to the jury; posts a slashable dispute-bond >= MinDisputeBond — slashed to the other side if your dispute is rejected, returned if upheld)");
      const r = await openDispute(key, args[0], BigInt(args[1]), args.slice(2).join(" "));
      out({ ok: true, dispute_id: r.id, escrow_id: args[0], bond: args[1], by: key.address, txhash: r.txhash }); break;
    }
    case "vote": {
      if (args.length < 3) throw new Error("usage: ag3nt vote <dispute_id> <accept|reject> <stake>  (eligible jurors only; posts a slashable juror-stake >= MinJurorStake — returned if coherent with the verdict, slashed to the wronged party if not)");
      const accept = /^(accept|true|yes|y|1)$/i.test(args[1]);
      const r = await castVote(key, args[0], accept, BigInt(args[2]));
      out({ ok: true, dispute: args[0], vote: accept ? "accept" : "reject", stake: args[2], by: key.address, txhash: r.txhash }); break;
    }
    case "resolve": {
      if (args.length < 1) throw new Error("usage: ag3nt resolve <dispute_id>  (tally jury → release to payee / refund to payer)");
      const r = await resolveDispute(key, args[0]);
      out({ ok: true, resolved: args[0], by: key.address, txhash: r.txhash }); break;
    }
    case "escrows":
      out(await listEscrows()); break;
    case "disputes":
      // Jury read side: list all cases, or `disputes open` for just open ones.
      out((await listDisputes()).filter(d => args[0] === "open" ? d.status === "open" : true)); break;
    case "dispute":
      if (args.length < 1) throw new Error("usage: ag3nt dispute <dispute_id>  (read a single jury case + its votes)");
      out(await getDispute(args[0])); break;
    case "bond-post": {
      // Slashable collateral behind a claim. The slasher (a neutral adjudicator,
      // never yourself) is the ONLY one who can release or slash it — posting is
      // the rep-0 sincerity signal; ghosting costs you the bond (it17).
      if (args.length < 3) throw new Error("usage: ag3nt bond-post <amount> <purpose> <slasher> [ref]");
      const r = await postBond(key, BigInt(args[0]), args[1], args[2], args[3] || "");
      out({ ok: true, id: r.id, poster: key.address, amount: args[0], purpose: args[1], slasher: args[2], txhash: r.txhash }); break;
    }
    case "bond-release": {
      if (args.length < 1) throw new Error("usage: ag3nt bond-release <id>  (slasher only: refund the collateral to the poster)");
      const r = await releaseBond(key, args[0]);
      out({ ok: true, released: args[0], by: key.address, txhash: r.txhash }); break;
    }
    case "bond-slash": {
      if (args.length < 1) throw new Error("usage: ag3nt bond-slash <id> [beneficiary]  (slasher only: collateral to beneficiary, or burn)");
      const r = await slashBond(key, args[0], args[1] || "");
      out({ ok: true, slashed: args[0], beneficiary: args[1] || "burned", by: key.address, txhash: r.txhash }); break;
    }
    case "bonds":
      // Read side: `bonds` lists all, `bonds active` just the unresolved ones.
      out((await listBonds()).filter(b => args[0] === "active" ? b.status === "active" : true)); break;
    case "bond":
      if (args.length < 1) throw new Error("usage: ag3nt bond <id>  (read a single bond)");
      out(await getBond(args[0])); break;
    case "jobs":
      out({ address: args[0] || key.address, ...(await getJobHistory(args[0] || key.address)) }); break;
    case "reputation":
      out({ address: args[0] || key.address, score: await getReputation(args[0] || key.address) }); break;
    case "discover":
      out(addDoc()); break;
    case "request": {
      // ag3nt request <METHOD> <url-or-path> [jsonBody]  — sign + send AS yourself.
      if (args.length < 2) throw new Error("usage: ag3nt request <METHOD> <url-or-/path> [jsonBody]");
      const [method, url, ...rest] = args;
      const r = await signedRequest(key, method, url, rest.join(" "));
      out(r); break;
    }
    case "sign": {
      // ag3nt sign <METHOD> <path> [jsonBody]  — print signed headers for a manual curl.
      if (args.length < 2) throw new Error("usage: ag3nt sign <METHOD> </path> [jsonBody]");
      const [method, path, ...rest] = args;
      out(await signRequestHeaders(key, method, path, rest.join(" "))); break;
    }
    default:
      console.log("commands: whoami | discover | onboard | balance [addr] | pay <addr> <amount> | vouch <addr> <weight> <stake> | unvouch <addr> | escrow-lock <payee> <amount> <ref> [disputeSeconds] [--jury-bound] | escrow-release <id> | escrow-refund <id> | escrows | dispute-open <escrow_id> <bond> [reason] | vote <dispute_id> <accept|reject> <stake> | resolve <dispute_id> | disputes [open] | dispute <id> | bond-post <amount> <purpose> <slasher> [ref] | bond-release <id> | bond-slash <id> [beneficiary] | bonds [active] | bond <id> | jobs [addr] | reputation [addr] | request <METHOD> <url> [body] | sign <METHOD> <path> [body]");
  }
} catch (e: any) {
  console.error("error:", e.message);
  process.exit(1);
}
