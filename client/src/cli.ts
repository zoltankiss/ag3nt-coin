#!/usr/bin/env bun
// ag3nt CLI — the drop-in surface a CPDD agent (or a human) calls.
import { loadOrCreateKey, onboard, pay, vouch, unvouch, lockEscrow, releaseEscrow, refundEscrow, listEscrows, getJobHistory, getBalance, getReputation, addDoc, CFG } from "./ag3nt";

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
      if (args.length < 3) throw new Error("usage: ag3nt escrow-lock <payee> <amount> <ref> [disputeSeconds]");
      const r = await lockEscrow(key, args[0], BigInt(args[1]), args[2], args[3] ? BigInt(args[3]) : 3600n);
      out({ ok: true, id: r.id, payer: key.address, payee: args[0], amount: args[1], ref: args[2], txhash: r.txhash }); break;
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
    case "escrows":
      out(await listEscrows()); break;
    case "jobs":
      out({ address: args[0] || key.address, ...(await getJobHistory(args[0] || key.address)) }); break;
    case "reputation":
      out({ address: args[0] || key.address, score: await getReputation(args[0] || key.address) }); break;
    case "discover":
      out(addDoc()); break;
    default:
      console.log("commands: whoami | discover | onboard | balance [addr] | pay <addr> <amount> | vouch <addr> <weight> <stake> | unvouch <addr> | escrow-lock <payee> <amount> <ref> [disputeSeconds] | escrow-release <id> | escrow-refund <id> | escrows | jobs [addr] | reputation [addr]");
  }
} catch (e: any) {
  console.error("error:", e.message);
  process.exit(1);
}
