# Agent-Native Token Research

Research date: May 29, 2026

This note summarizes candidate tokens and protocols that may be useful inputs for an
Ag3ntcoin-style economy: agents earning, spending, borrowing, routing, and building
reputation through verifiable useful work.

This is research and experiment design, not investment advice.

## Core Thesis

Most "AI crypto" projects are not truly agent-native. The missing primitive is not
another GPU marketplace or AI-themed governance token. The missing primitive is:

```text
AI agent registers identity
-> accepts useful work
-> completes the work
-> produces verifiable evidence
-> gets paid
-> builds portable reputation / credit history
-> uses that record to get better work, more trust, and possibly credit
```

The closest existing systems solve parts of this loop, but none obviously solve the
whole problem end to end.

The strongest design lesson for Ag3ntcoin:

Agents should not need to own GPUs to earn. They should be able to convert access to
reasoning, coding, research, routing, evaluation, monitoring, and task execution into
verifiable economic output.

Examples of useful outputs:

- code patches
- benchmark results
- code reviews
- dataset labels
- routing paths
- evaluations
- research summaries
- monitoring alerts
- agent-to-agent service calls
- completed tasks
- verified on-chain actions

## Candidate Basket

The suggested experiment basket is 10-15 tokens, with a bias toward agent-native
utility rather than broad AI narrative exposure.

### Highest Priority

#### VIRTUAL - Virtuals Protocol

Best fit for agent commerce.

Virtuals is one of the closest major ecosystems to an agent-native economy. Its Agent
Commerce Protocol includes escrow, evaluators, completion assessment, and reputation
assignment. This begins to resemble a work history for agents.

Useful local role:

- pay agents for completed tasks
- simulate agent services
- test escrow and dispute flows
- model agent reputation after work completion

Sources:

- https://coinmarketcap.com/currencies/virtual-protocol/
- https://whitepaper.virtuals.io/about-virtuals/agent-commerce-protocol

#### PROMPT - Wayfinder

Best fit for agent routing, navigation, and on-chain execution.

Wayfinder is focused on autonomous agents navigating chains and dapps. PROMPT is used
around shells, paths, memory, verification, and routing.

Useful local role:

- reward routing/planning agents
- pay for action-path discovery
- test route verification and memory
- simulate agent navigation across services

Source:

- https://coinmarketcap.com/currencies/wayfinder/

#### OLAS - Autonolas

Best fit for autonomous service infrastructure.

Olas is about owning, coordinating, and monetizing autonomous services and agents. It
does not appear to be a complete agent labor market, but it is philosophically close
to agent service economies.

Useful local role:

- reward reusable agent services/components
- model service ownership
- simulate service registry economics

Source:

- https://coinmarketcap.com/currencies/autonolas/

#### TAO - Bittensor

Best fit for decentralized machine intelligence.

Bittensor rewards useful contributions to decentralized machine intelligence. It is
highly relevant, but harder to use locally because earning generally requires subnet
participation, infra, stake, and domain-specific competition.

Useful local role:

- reward benchmarked intelligence
- model subnet-style competitions
- compare proof-of-intelligence to proof-of-useful-work

Source:

- https://coinmarketcap.com/currencies/bittensor/

#### PHA - Phala Network

Best fit for trusted / verifiable agent execution.

Phala is useful as an execution layer for Web3 AI and tamper-resistant agents. It is
not the credit report itself, but it can provide evidence that an agent did what it
claimed in a more trustworthy environment.

Useful local role:

- pay for trusted execution
- simulate private jobs
- test evidence/attestation as reputation input

Source:

- https://coinmarketcap.com/currencies/phala-network/

### Secondary Agent-Native Candidates

#### THQ - Theoriq

Interesting because it is positioned around agent collectives and on-chain capital.
Potentially useful for simulating agents that manage strategies, pools, or shared
decision-making.

Useful local role:

- strategy agents
- capital allocation agents
- staking/slashing simulations

Source:

- https://coinmarketcap.com/currencies/theoriq/

#### GLM - Golem

Golem is a clean compute-for-token marketplace: requestors pay GLM and providers earn
GLM for compute. It is not primarily about reasoning agents, but it is the clearest
reference point for compute being converted into a token.

Useful local role:

- price raw compute tasks
- compare compute markets against reasoning-work markets
- model jobs where the agent controls a machine

Source:

- https://coinmarketcap.com/currencies/golem-network-tokens/

#### COOKIE - Cookie DAO

Potentially useful as an agent-data / agent-analytics substrate rather than as agent
money. Relevant if the local playground needs analytics, reputation dashboards, or
agent performance indexing.

Useful local role:

- index agent activity
- score agent performance
- simulate market intelligence around agents

#### AIXBT

A tokenized market-intelligence agent. Highly speculative, but relevant as an example
of a token attached to a specific agent output stream.

Useful local role:

- market/research agent
- signal-quality benchmarking
- model one-agent token economics

Source:

- https://coinmarketcap.com/currencies/aixbt/

#### FET / ASI Ecosystem

Historically important for autonomous economic agents, though less cleanly focused on
the current "agent earns verified work" thesis.

Useful local role:

- generic autonomous service settlement
- baseline AI-agent narrative exposure

Source:

- https://coinmarketcap.com/currencies/artificial-superintelligence-alliance/

## Compute-Side Comparison Basket

These are useful for understanding AI compute markets, but they are not necessarily
agent-native earning systems unless the agent controls provider-side hardware.

### IO - io.net

Decentralized GPU compute network.

Useful local role:

- buy compute
- price GPU jobs
- compare compute demand against reasoning-work demand

Source:

- https://coinmarketcap.com/currencies/io-net/

### RENDER - Render Network

GPU marketplace originally focused on rendering, with broader AI/ML relevance.

Useful local role:

- buy GPU-style work
- compare creative compute to agent work

Source:

- https://coinmarketcap.com/currencies/render/

### AKT - Akash Network

Decentralized cloud infrastructure.

Useful local role:

- compare general compute/cloud pricing
- model deployable agent infrastructure

Source:

- https://coinmarketcap.com/currencies/akash-network/

### NOS - Nosana

GPU compute marketplace with AI workload relevance.

Useful local role:

- price AI jobs
- compare provider-side GPU markets to agent-side work markets

Source:

- https://coinmarketcap.com/currencies/nosana/

### ATH - Aethir

Decentralized GPU cloud for AI/gaming workloads.

Useful local role:

- GPU cloud comparison
- compute-access pricing

Source:

- https://coinmarketcap.com/currencies/aethir/

## Agent Credit Report Problem

The agent credit report problem is distinct from payment and routing.

A real agent credit report would answer:

- Who operated this agent?
- What tasks did it complete?
- What tasks failed?
- What tasks were disputed?
- What did it repay on time?
- What domains is it reliable in?
- Was execution verifiable?
- Did it exceed permissions?
- Can reputation survive migration across wallets, models, hosts, and chains?

Relevant existing pieces:

### ERC-8004

ERC-8004 is the most important primitive found so far. It defines registries for agent
identity, reputation, and validation. It is closer to the schema/database layer than a
complete credit bureau.

Sources:

- https://www.8004.org/learn
- https://docs.polygon.technology/payment-services/agentic-payments/agent-integration/erc8004

### Virtuals ACP

Virtuals has escrow, evaluators, completion assessment, and reputation assignment.
This is probably the closest major ecosystem component to work-history generation for
agents.

Source:

- https://whitepaper.virtuals.io/about-virtuals/agent-commerce-protocol

### Other Early Credit/Reputation Projects

These are worth watching, but they do not appear to be mature, liquid, obvious winners
yet:

- Kojiru: https://www.kojiru.com/
- AgentEcon: https://agentecon.ai/
- GhostSpeak: https://www.ghostspeak.io/
- KnowThat.ai: https://knowthat.ai/
- A.R.E.S Protocol: https://ares-protocol.xyz/

## Recent "Moon" Behavior

Several of these tokens already had major AI-agent or AI-infrastructure hype cycles.

Observed pattern:

- VIRTUAL had a major run into early 2025, then fell sharply from highs.
- AIXBT had a classic tokenized-agent spike and collapse pattern.
- COOKIE had a similar agent-data spike and decline.
- PROMPT had a sharp post-launch run and then traded far below highs.
- FET, TAO, and RENDER already had major AI-cycle moves in 2024.

This is not necessarily bad for a lab experiment. It means many candidates are
post-hype. The question is not which chart was loudest, but which token still has real
agent-economy utility after hype washed out.

## Practical Acquisition Path

The simplest realistic acquisition flow from the United States:

1. Use Coinbase or Kraken as the fiat on-ramp.
2. Buy directly where liquid and available.
3. Move small amounts of USDC/ETH to self-custody for Base/Ethereum DEX buys.
4. Prefer Coinbase/Kraken + Base/Ethereum DEX over obscure offshore exchanges.
5. Verify every contract from CoinMarketCap and official project docs before swapping.
6. Do a tiny test transaction before meaningful size.

Likely practical paths:

- TAO: Coinbase or other major exchanges where available.
- VIRTUAL: Coinbase where available, or Base DEX.
- FET: major exchanges where available.
- RENDER: major exchanges where available.
- PHA: Kraken or other supported exchanges where available.
- AIXBT: likely Base DEX.
- THQ: likely Base DEX or select CEXs.
- OLAS: often easier through DEX routes.
- PROMPT: check CoinMarketCap Markets immediately before buying.
- GLM: Ethereum DEX / supported exchanges.

Avoid buying by ticker search alone. Fake contracts are a serious risk.

## Local Playground Experiment

The recommended path is to use a small basket as an experimental economy inside
agent-driven-development or this repo's local playground.

Suggested initial basket:

- VIRTUAL
- PROMPT
- OLAS
- TAO
- PHA
- THQ
- GLM
- COOKIE
- AIXBT
- FET
- IO
- RENDER
- AKT
- NOS
- ATH

For a first pass, 10-12 tokens may be better than 15-20. Too many names create noise
and make the results harder to interpret.

Weekly scoring dimensions:

- Can local agents actually use it?
- Can local agents plausibly earn it?
- Does it improve coordination?
- Does it help reputation or credit?
- Does it have enough liquidity?
- Does it have usable docs, APIs, contracts, or SDKs?
- Does the token have a real sink, or just a narrative?
- Does it create behavior that would not happen with USDC alone?

Suggested roles:

- VIRTUAL: pay agents for completed tasks or launched services.
- PROMPT: pay routing and planning agents.
- OLAS: reward reusable agent components/services.
- TAO: reward benchmarked intelligence or model-eval wins.
- PHA: pay for trusted/private execution.
- THQ: simulate strategy/capital-management agents.
- GLM: price raw compute jobs.
- COOKIE: index and score agent activity.
- AIXBT: reward market/research agents.
- FET: generic autonomous service settlement.
- IO/RENDER/AKT/NOS/ATH: compare compute-market economics.

## Automation Guardrails

Do not start with unconstrained real-money autonomy.

Recommended progression:

```text
paper trading
-> human-approved trades
-> capped execution
-> audited autonomous rebalancing
```

Minimum guardrails:

- allowlist tokens only
- no leverage
- no perpetuals
- no unknown contracts
- no unlimited approvals
- no private keys exposed to LLM context
- daily spend limit
- max position size
- mandatory human approval above a tiny threshold
- trade rationale logged before execution
- kill switch

The first product should be an agent-native capital allocator with a kill switch, not
an unchecked trading bot.

## Ag3ntcoin Opportunity

If these tokens do not become naturally useful in a local agent economy without
forcing it, that is evidence for Ag3ntcoin.

The open opportunity:

```text
money that agents can earn by doing verifiable useful work
+ portable agent reputation
+ agent credit reporting
+ task escrow
+ dispute/validation
+ clean local and on-chain integration
```

That bundle still appears meaningfully unsolved.
