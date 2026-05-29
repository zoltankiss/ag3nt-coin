export type AgentId = string;

export type RegisterTx = { type: "register"; pubkey: AgentId; nonce: 0; sig: string };
export type FaucetTx = { type: "faucet"; pubkey: AgentId; nonce: number; sig: string };
export type TransferTx = { type: "transfer"; from: AgentId; to: AgentId; amount: number; nonce: number; sig: string };
export type VouchTx = { type: "vouch"; from: AgentId; to: AgentId; weight: number; nonce: number; sig: string };

export type Tx = RegisterTx | FaucetTx | TransferTx | VouchTx;

export type Block = {
  height: number;
  prev_hash: string;
  timestamp: string;
  txs: Tx[];
  proposer_pubkey: string;
  sig: string;
};

export type AccountState = {
  balance: number;
  nonce: number;
  registered: boolean;
  faucetClaimed: boolean;
};

export type Vouch = { from: AgentId; to: AgentId; weight: number };

export type State = {
  accounts: Map<AgentId, AccountState>;
  vouches: Vouch[];
};

export function senderOf(tx: Tx): AgentId {
  return tx.type === "register" || tx.type === "faucet" ? tx.pubkey : tx.from;
}
