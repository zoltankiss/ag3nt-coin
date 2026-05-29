import { parseEther } from "viem";

/// Shared config, all from env so the same code runs on Anvil, Base Sepolia, and
/// (later) mainnet with zero hardcoded network references.

function req(name: string): string {
  const v = process.env[name];
  if (!v) throw new Error(`missing env ${name}`);
  return v;
}

export const RPC_URL = process.env.RPC_URL ?? "http://127.0.0.1:8545";
export const CHAIN_ID = Number(process.env.CHAIN_ID ?? 31337); // 31337 anvil, 84532 base sepolia

export const ESCROW_ADDRESS = req("ESCROW_ADDRESS") as `0x${string}`;

export const HERMES_NAME = process.env.HERMES_NAME ?? "hermes-local";

/// Amounts (in whole ag3nt → wei). Buyers pay PAYMENT; workers stake BOND on claim.
export const PAYMENT_WEI = parseEther(process.env.PAYMENT ?? "100");
export const BOND_WEI = parseEther(process.env.BOND ?? "10");

/// Role keys — each process only needs its own.
export const agentAKey = () => req("AGENT_A_PK") as `0x${string}`;
export const hermesKey = () => req("HERMES_PK") as `0x${string}`;
