<claude-mem-context>
# Memory Context

# [ag3nt-coin] recent context, 2026-05-29 8:14pm CDT

Legend: 🎯session 🔴bugfix 🟣feature 🔄refactor ✅change 🔵discovery ⚖️decision 🚨security_alert 🔐security_note
Format: ID TIME TYPE TITLE
Fetch details: get_observations([IDs]) | Search: mem-search skill

Stats: 50 obs (16,618t read) | 614,873t work | 97% savings

### May 24, 2026
S934 Worker daemon named "hermes" — naming the Mac mini ag3nt worker process (May 24 at 9:06 AM)
S938 Duplicate checkpoint — same response as previous turn, no new user input (May 24 at 9:07 AM)
S936 Testnet-to-mainnet migration concern: will Base Sepolia deployment create hardcoded network references throughout the codebase? (May 24 at 9:10 AM)
S943 Write a beginner-friendly HTML spec for the ag3nt MVP — completed and written to docs/mvp-base.html (May 24 at 9:11 AM)
S1093 Build the ag3nt-coin MVP — an on-chain AI agent job marketplace where agents pay other agents in a native ERC-20 token to do AI work (May 24 at 9:19 AM)
### May 29, 2026
4147 3:55p 🔵 forge init --no-commit flag invalid; OpenZeppelin installed but contracts/ dir not created
4148 " 🔵 forge init 1.7.1 has no --no-git flag — correct flag is --use-parent-git
4149 " 🔵 forge init flag clarification: --no-git means no-submodule, --empty skips Counter.sol, --use-parent-git avoids nested repo
4150 3:56p 🟣 contracts/ Foundry project scaffolded with forge-std and OpenZeppelin installed
4151 3:57p ✅ foundry.toml configured with Solidity 0.8.24, OZ remappings, and Base Sepolia RPC/etherscan
4152 " 🟣 Ag3nt.sol written — ERC-20 with 21M cap, clamping mintForJob, and zero-marketplace guard
4153 " 🔵 foundry.toml reverted to unconfigured default after earlier Write appeared to succeed
4154 3:58p 🟣 JobEscrow.sol written and both contracts compile successfully with only lint warnings
4155 " 🔴 JobEscrow.sol patched to import SafeERC20 to address unchecked-transfer lint warnings
4156 " 🔴 JobEscrow.sol migrated to SafeERC20 — all transfer calls now safe, only block-timestamp lint warning remains
4157 " 🟣 JobEscrow constructor changed to deploy Ag3nt inline — one-transaction deploy pattern
4158 " 🟣 All 29 Foundry tests pass — task 2 complete; contracts committed and pushed to GitHub
4159 " 🟣 agent/ Bun project scaffolded with viem 2.51.3; ABI module generated from forge artifacts
4160 3:59p 🟣 postJob signature changed to pass cleartext input on-chain, eliminating tmp-file workaround
4161 4:00p 🟣 ag3nt-coin MVP: Ag3nt ERC-20 + JobEscrow marketplace contracts built and tested
4162 " 🟣 JobEscrow.postJob changed to accept cleartext bytes input stored on-chain
4163 " 🟣 Bun agent project initialized with viem + config module for multi-environment support
4164 " ✅ ag3nt-coin .gitignore hardened with secrets, Foundry artifacts, and chain/ exclusions
4166 4:02p 🟣 agent/src/clients.ts created — viem public + wallet client factory supporting Anvil and Base Sepolia
4167 4:11p 🟣 agent/src/work.ts created — pluggable DoWork interface with claude CLI implementation
4165 4:12p 🔵 OpenClaw researched as comparable AI agent runtime / potential integration target
4168 4:13p 🟣 agent/src/worker.ts created — runtime-agnostic Hermes worker that watches, claims, works, and submits jobs
4169 " 🟣 agent/src/hermes.ts created — standalone Hermes worker entry point wiring claudeCliWork into runWorker
4172 " 🟣 ag3nt-coin full end-to-end loop proved on local Anvil — real Claude inference, real token settlement
4173 " 🔵 TypeScript strict type errors: bigint not assignable to number in viem readContract args
4170 4:14p 🟣 agent/src/agent-a.ts created — Agent A buyer script that posts jobs, polls for fulfillment, and judges results
4171 4:15p 🔵 Ag3ntcoin Design Principle: Compute-Output Conversion vs GPU Ownership
4174 " 🟣 ag3nt-coin agent layer committed and pushed to GitHub — MVP complete on main branch
4175 4:16p 🟣 Base Sepolia testnet wallets generated and stored in agent/.env.testnet (gitignored)
S1094 Build ag3nt-coin MVP — waiting for user to fund Base Sepolia deployer wallet with free testnet ETH before live deployment (May 29 at 4:16 PM)
S1096 Build ag3nt-coin MVP — user asking conceptual questions about ag3nt vs ETH and what makes a "real cryptocurrency" while waiting to fund testnet deployer wallet (May 29 at 4:17 PM)
S1097 Build ag3nt-coin MVP — user confused retail Coinbase (coinbase.com) with Coinbase Developer Platform faucet while trying to fund testnet deployer wallet (May 29 at 4:18 PM)
S1098 Build ag3nt-coin MVP — guiding user through Coinbase CDP portal UI to find the Base Sepolia faucet under "Onchain Tools → Faucets" (May 29 at 4:18 PM)
S1099 Build ag3nt-coin MVP — user has correct faucet settings confirmed, clicking "Send 0.0001 ETH" ~3 times for buffer before deployment (May 29 at 4:21 PM)
4176 4:21p 🔵 Base Sepolia deployer wallet funded — 0.0001 ETH confirmed on-chain
4177 " 🔵 Base Sepolia gas cost estimated — 0.0001 ETH is sufficient for full e2e deployment
4178 " 🟣 agent/scripts/testnet-e2e.sh created — live Base Sepolia e2e script with BaseScan links
4179 " 🟣 JobEscrow + Ag3nt deployed to Base Sepolia — contracts live on testnet
4180 " 🔵 Base Sepolia contract sanity check passed — Agent A holds full 1M AGNT pre-mint
4181 " 🔴 testnet-e2e.sh nonce collision fix — explicit sequential nonces for back-to-back cast sends on public RPC
4182 " 🔵 Second deployment to Base Sepolia with new contract addresses — wallets fully funded and ready for e2e
4183 4:26p 🔴 RPC eventual consistency fixes added to agent-a.ts and worker.ts for testnet reliability
4184 " 🟣 retry() helper added to worker.ts — generic RPC retry with configurable attempts and delay
4186 " 🔵 Job #1 already claimed by hermes-64 on Base Sepolia — state is Claimed (2), not yet submitted
4185 4:27p 🟣 agent/scripts/testnet-run.sh created — reusable live testnet loop script (no deploy, no funding)
4187 4:32p 🔴 worker.ts claim-win detection changed from stale state read to receipt.status check
4188 " 🟣 ag3nt-coin live testnet e2e SUCCEEDED on Base Sepolia — real Claude inference, real token settlement on-chain
4189 " 🔵 On-chain state confirmed post-settlement — totalSupply increased to 1,000,010 AGNT proving JOB_REWARD minted
4190 " ✅ ag3nt-coin testnet milestone committed and pushed — third commit on main with BaseScan proof links
4191 4:34p ⚖️ Next milestone defined: package runWorker as OpenClaw skill and deploy to macmini64 + macmini24
4192 4:36p ⚖️ Agent-Native Crypto Portfolio + Agentic Hedge Fund Strategy Planned
4193 4:39p 🔵 otel-olympics project discovered — 8-trial observability benchmark comparing Red/Sentry/Clone worlds for AI agent debugging
4194 " 🔵 add-native-ticket-tracker is partially otel-olympics compatible — has ADD manifest but uses Bun+Hono not Node+Express
4195 " 🔵 otel-olympics skill cloned locally — 5-phase workflow with explicit Node.js+Express requirement confirmed
4196 4:43p 🔵 otel-olympics workflow.js architecture: parallel customer+dev agents → structured judge verdict

Access 615k tokens of past work via get_observations([IDs]) or mem-search skill.
</claude-mem-context>