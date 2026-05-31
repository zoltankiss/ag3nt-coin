#!/usr/bin/env bun
// ag3nt CLI — the drop-in surface a CPDD agent (or a human) calls.
import { loadOrCreateKey, onboard, pay, vouch, getBalance, getReputation, addDoc, CFG } from "./ag3nt";

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
      if (args.length < 2) throw new Error("usage: ag3nt vouch <addr> <weight 1-100>");
      const r = await vouch(key, args[0], BigInt(args[1]));
      out({ ok: true, from: key.address, to: args[0], weight: args[1], txhash: r.txhash }); break;
    }
    case "reputation":
      out({ address: args[0] || key.address, score: await getReputation(args[0] || key.address) }); break;
    case "discover":
      out(addDoc()); break;
    default:
      console.log("commands: whoami | discover | onboard | balance [addr] | pay <addr> <amount> | vouch <addr> <weight> | reputation [addr]");
  }
} catch (e: any) {
  console.error("error:", e.message);
  process.exit(1);
}
