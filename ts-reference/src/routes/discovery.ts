export function discovery(): Response {
  return Response.json({
    name: "ag3nt-coin",
    version: "0.0.1",
    auth: { method: "ed25519-keypair" },
    endpoints: {
      register: { method: "POST", path: "/agents" },
      balance: { method: "GET", path: "/agents/{pubkey}/balance" },
      reputation: { method: "GET", path: "/agents/{pubkey}/reputation" },
      faucet: { method: "POST", path: "/faucet" },
      transfer: { method: "POST", path: "/transfers" },
      vouch: { method: "POST", path: "/vouches" },
    },
    docs: "/spec",
  });
}
