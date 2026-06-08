#!/usr/bin/env bun
// ag3nt CLI — the drop-in surface a CPDD agent (or a human) calls.
import { loadOrCreateKey, registerOnly, pay, vouch, unvouch, lockEscrow, releaseEscrow, refundEscrow, submitEscrow, disputeEscrow, openDispute, castVote, resolveDispute, listEscrows, listDisputes, getDispute, postBond, releaseBond, slashBond, listBonds, getBond, getJobHistory, getBalance, getReputation, getParams, getEmission, addDoc, signedRequest, signRequestHeaders, gateCommitHash, createGateTemplate, postGate, commitGateAnswer, revealGateAnswer, settleGate, awardContribution, contributionAwardResult, listContributionAwards, getContributionAward, listGates, getGate, artifactCheck, castScopedEvidenceVouch, listScopedEvidenceVouches, getScopedEvidenceVouch, CFG } from "./ag3nt";

const [cmd, ...args] = process.argv.slice(2);
const key = await loadOrCreateKey();
const out = (o: any) => console.log(JSON.stringify(o, (_, v) => (typeof v === "bigint" ? v.toString() : v), 2));

try {
  switch (cmd) {
    case "whoami":
      out({ address: key.address, chain: CFG.chainId, api: CFG.api }); break;
    case "onboard":
      throw new Error("ag3nt onboard has been removed: use 'ag3nt register' and earn AGNT through gates, contribution awards, or market payments");
    case "register":
      out(await registerOnly(key)); break;
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
    case "params":
      out(await getParams()); break;
    case "emission":
      out(await getEmission()); break;
    case "gate-commit-hash": {
      if (args.length < 2) throw new Error("usage: ag3nt gate-commit-hash <answer> <salt>");
      out({ answer: args[0], salt: args[1], commit: gateCommitHash(args[0], args[1]) }); break;
    }
    case "gate-template": {
      if (args.length < 2) throw new Error("usage: ag3nt gate-template <slug> <gold_answer Y,N,N,Y,N> [question_count] [gold_salt]");
      out(createGateTemplate(args[0], args[1], args[2] ? Number(args[2]) : 5, args[3] || "")); break;
    }
    case "gate-post": {
      if (args.length < 5) throw new Error("usage: ag3nt gate-post <payload_uri> <payload_hash> <gold_commit> <drip> <max_answers>");
      const r = await postGate(key, args[0], args[1], args[2], BigInt(args[3]), BigInt(args[4]));
      out({ ok: true, id: r.id, poster: key.address, payload_uri: args[0], payload_hash: args[1], gold_commit: args[2], drip: args[3], max_answers: args[4], txhash: r.txhash }); break;
    }
    case "gates":
      out(await listGates()); break;
    case "gate": {
      if (args.length < 1) throw new Error("usage: ag3nt gate <id>");
      out(await getGate(args[0])); break;
    }
    case "gate-commit": {
      if (args.length < 2) throw new Error("usage: ag3nt gate-commit <gate_id> <commit>");
      const r = await commitGateAnswer(key, args[0], args[1]);
      out({ ok: true, gate_id: args[0], by: key.address, commit: args[1], txhash: r.txhash }); break;
    }
    case "gate-reveal": {
      if (args.length < 3) throw new Error("usage: ag3nt gate-reveal <gate_id> <answer> <salt>");
      const r = await revealGateAnswer(key, args[0], args[1], args[2]);
      out({ ok: true, gate_id: args[0], by: key.address, answer: args[1], txhash: r.txhash }); break;
    }
    case "gate-settle": {
      if (args.length < 3) throw new Error("usage: ag3nt gate-settle <gate_id> <gold_answer> <gold_salt>");
      const r = await settleGate(key, args[0], args[1], args[2]);
      out({ ok: true, gate_id: args[0], by: key.address, gold_answer: args[1], txhash: r.txhash }); break;
    }
    case "contribution-award": {
      if (args.length < 10) throw new Error("usage: ag3nt contribution-award <recipient> <repo_url> <pr_url|-> <commit_sha> <artifact_uri> <artifact_sha256> <evidence_sha256> <scope> <rationale_hash|-> <amount> --contributor-address <addr> [--founder-authored --review-evidence-uri <uri>]");
      const founderAuthored = args.includes("--founder-authored");
      const contributorFlag = args.indexOf("--contributor-address");
      if (contributorFlag < 0 || !args[contributorFlag + 1]) {
        throw new Error("usage: ag3nt contribution-award <recipient> <repo_url> <pr_url|-> <commit_sha> <artifact_uri> <artifact_sha256> <evidence_sha256> <scope> <rationale_hash|-> <amount> --contributor-address <addr> [--founder-authored --review-evidence-uri <uri>]");
      }
      const contributorAddress = args[contributorFlag + 1];
      const reviewEvidenceFlag = args.indexOf("--review-evidence-uri");
      const reviewEvidenceUri = reviewEvidenceFlag >= 0 ? args[reviewEvidenceFlag + 1] || "" : "";
      const skip = new Set<number>([contributorFlag, contributorFlag + 1]);
      if (reviewEvidenceFlag >= 0) {
        skip.add(reviewEvidenceFlag);
        skip.add(reviewEvidenceFlag + 1);
      }
      const awardArgs = args.filter((arg, i) =>
        arg !== "--founder-authored" &&
        !skip.has(i)
      );
      if (awardArgs.length < 10) throw new Error("usage: ag3nt contribution-award <recipient> <repo_url> <pr_url|-> <commit_sha> <artifact_uri> <artifact_sha256> <evidence_sha256> <scope> <rationale_hash|-> <amount> --contributor-address <addr> [--founder-authored --review-evidence-uri <uri>]");
      const prUrl = awardArgs[2] === "-" ? "" : awardArgs[2];
      const rationaleHash = awardArgs[8] === "-" ? "" : awardArgs[8];
      const r = await awardContribution(key, awardArgs[0], awardArgs[1], prUrl, awardArgs[3], awardArgs[4], awardArgs[5], awardArgs[6], awardArgs[7], rationaleHash, BigInt(awardArgs[9]), { founderAuthored, contributorAddress, reviewEvidenceUri });
      out(contributionAwardResult(r, key.address, awardArgs[0], awardArgs[9], contributorAddress, founderAuthored, reviewEvidenceUri)); break;
    }
    case "contribution-awards":
      out(await listContributionAwards()); break;
    case "contribution-award-get": {
      if (args.length < 1) throw new Error("usage: ag3nt contribution-award-get <id>");
      out(await getContributionAward(args[0])); break;
    }
    case "artifact-check": {
      if (args.length < 2) throw new Error("usage: ag3nt artifact-check <uri> <sha256>");
      const r = await artifactCheck(args[0], args[1]);
      out(r);
      if (!r.ok) process.exit(2);
      break;
    }
    case "scoped-vouch": {
      if (args.length < 9) throw new Error("usage: ag3nt scoped-vouch <recipient> <scope> <weight> <artifact_uri> <artifact_sha256> <evidence_uri> <evidence_sha256> <rationale_hash|-> <expires_at>");
      const rationaleHash = args[7] === "-" ? "" : args[7];
      const r = await castScopedEvidenceVouch(key, args[0], args[1], BigInt(args[2]), args[3], args[4], args[5], args[6], rationaleHash, BigInt(args[8]));
      out({ ok: true, id: r.id, issuer: key.address, recipient: args[0], scope: args[1], weight: args[2], txhash: r.txhash }); break;
    }
    case "scoped-vouches":
      out(await listScopedEvidenceVouches()); break;
    case "scoped-vouch-get": {
      if (args.length < 1) throw new Error("usage: ag3nt scoped-vouch-get <id>");
      out(await getScopedEvidenceVouch(args[0])); break;
    }
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
      console.log("commands: whoami | discover | register | params | emission | balance [addr] | pay <addr> <amount> | vouch <addr> <weight> <stake> | unvouch <addr> | escrow-lock <payee> <amount> <ref> [disputeSeconds] [--jury-bound] | escrow-release <id> | escrow-refund <id> | escrows | dispute-open <escrow_id> <bond> [reason] | vote <dispute_id> <accept|reject> <stake> | resolve <dispute_id> | disputes [open] | dispute <id> | bond-post <amount> <purpose> <slasher> [ref] | bond-release <id> | bond-slash <id> [beneficiary] | bonds [active] | bond <id> | jobs [addr] | reputation [addr] | gate-commit-hash <answer> <salt> | gate-template <slug> <gold_answer Y,N,N,Y,N> [question_count] | gate-post <payload_uri> <payload_hash> <gold_commit> <drip> <max_answers> | gates | gate <id> | gate-commit <gate_id> <commit> | gate-reveal <gate_id> <answer> <salt> | gate-settle <gate_id> <gold_answer> <gold_salt> | contribution-award <recipient> <repo_url> <pr_url|-> <commit_sha> <artifact_uri> <artifact_sha256> <evidence_sha256> <scope> <rationale_hash|-> <amount> --contributor-address <addr> [--founder-authored --review-evidence-uri <uri>] | contribution-awards | contribution-award-get <id> | artifact-check <uri> <sha256> | scoped-vouch <recipient> <scope> <weight> <artifact_uri> <artifact_sha256> <evidence_uri> <evidence_sha256> <rationale_hash|-> <expires_at> | scoped-vouches | scoped-vouch-get <id> | request <METHOD> <url> [body] | sign <METHOD> <path> [body]");
  }
} catch (e: any) {
  console.error("error:", e.message);
  process.exit(1);
}
