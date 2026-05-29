import { createPublicClient, createWalletClient, http, defineChain } from "viem";
import { privateKeyToAccount } from "viem/accounts";
import { baseSepolia } from "viem/chains";
import { RPC_URL, CHAIN_ID } from "./config";

/// Local Anvil chain (CHAIN_ID 31337). For Base Sepolia (84532) we use viem's
/// built-in chain def but still route RPC through our env URL.
const anvil = defineChain({
  id: 31337,
  name: "anvil",
  nativeCurrency: { name: "Ether", symbol: "ETH", decimals: 18 },
  rpcUrls: { default: { http: [RPC_URL] } },
});

export const chain = CHAIN_ID === 84532 ? baseSepolia : anvil;

export const publicClient = createPublicClient({
  chain,
  transport: http(RPC_URL),
  pollingInterval: 1_000, // snappy local demo; events poll every 1s
});

export function walletFor(pk: `0x${string}`) {
  return createWalletClient({
    account: privateKeyToAccount(pk),
    chain,
    transport: http(RPC_URL),
  });
}
