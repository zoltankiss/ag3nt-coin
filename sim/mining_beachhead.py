#!/usr/bin/env python3
"""mining_beachhead.py — gate-v1 parameter Monte Carlo (the "mining beachhead").

Rung 2 of the validation ladder (rung 1 = keeper integration tests, rung 3 =
end-to-end runs with real LLM agents). This sweeps gate-v1's economic
constants over populations of pure-policy agents to find the viable region
BEFORE the constants harden into genesis values.

Policies (no LLMs — policies are probability tables):
  honest   — does the verification work; answers the true verdict with
             accuracy ACC (0.95: a representative cheap-verifier accuracy
             from earlier smoke tests),
             pays COST_HONEST microcoin-equivalent compute per answer
  stamper  — the lazy sybil: always answers "pass", ~zero cost
  guesser  — uniform random over the k-ary answer space, ~zero cost
  (copier is not a policy: commit-reveal makes copying structurally
   impossible — that's chain code, pinned by integration test G2)

Gate model (mirrors x/agntcoin gate-v1 exactly):
  - decoy w.p. DECOY_RATIO (gold verdict precommitted), else live (plurality)
  - latent truth: "pass" w.p. G_PASS, else one of k-1 fail variants
  - decoy pays exact gold matches; live pays strict-plurality (tie pays none)
  - drip minted per coherent answer

Outputs: results JSON + a findings table on stdout; the HTML report is
generated alongside by the caller.
"""

import json
import random
import sys
from collections import Counter

SEED = 1337
POP = 200              # total agent population
ANSWERERS_PER_GATE = 11  # self-selected sample answering each gate (odd)
GATES = 2000           # gates per config
ACC = 0.95             # honest accuracy (measured, smoke test 2026-06-07)
G_PASS = 0.4           # latent P(truth == "pass")
COST_HONEST = 2        # microcoin-equiv compute per honest answer
COST_LAZY = 0          # stampers/guessers don't look at the payload
DRIP = 10              # microcoin minted per coherent answer (cap is 50)

K_SPACE = [2, 4, 8]
DECOY_RATIOS = [0.1, 0.3, 0.5, 0.7]
SYBIL_SHARES = [0.2, 0.5, 0.8]  # stampers+guessers as a fraction of POP


def run_config(k, decoy_ratio, sybil_share, rng):
    answers_space = ["pass"] + [f"fail:{i}" for i in range(1, k)]
    n_sybil = int(POP * sybil_share)
    n_stamper = n_sybil // 2
    n_guesser = n_sybil - n_stamper
    n_honest = POP - n_sybil
    # population roster: policy per agent index
    roster = (["honest"] * n_honest) + (["stamper"] * n_stamper) + (["guesser"] * n_guesser)

    earned = Counter()   # policy -> microcoin minted
    cost = Counter()     # policy -> compute spent
    answers_given = Counter()
    live_total, live_correct = 0, 0
    emission = 0

    for _ in range(GATES):
        # latent truth for this gate's payload
        truth = "pass" if rng.random() < G_PASS else rng.choice(answers_space[1:])
        is_decoy = rng.random() < decoy_ratio

        panel = rng.sample(range(POP), ANSWERERS_PER_GATE)
        gate_answers = []  # (policy, answer)
        for idx in panel:
            pol = roster[idx]
            if pol == "honest":
                ans = truth if rng.random() < ACC else rng.choice([a for a in answers_space if a != truth])
                cost[pol] += COST_HONEST
            elif pol == "stamper":
                ans = "pass"
                cost[pol] += COST_LAZY
            else:  # guesser
                ans = rng.choice(answers_space)
                cost[pol] += COST_LAZY
            gate_answers.append((pol, ans))
            answers_given[pol] += 1

        if is_decoy:
            winning = truth  # the precommitted gold verdict
        else:
            counts = Counter(a for _, a in gate_answers)
            top = counts.most_common()
            if len(top) > 1 and top[0][1] == top[1][1]:
                winning = None  # tie pays nobody
            else:
                winning = top[0][0]
            live_total += 1
            if winning == truth:
                live_correct += 1

        if winning is not None:
            for pol, ans in gate_answers:
                if ans == winning:
                    earned[pol] += DRIP
                    emission += DRIP

    def per_answer(c, pol):
        n = answers_given[pol]
        return (c[pol] / n) if n else 0.0

    dishonest_earned = earned["stamper"] + earned["guesser"]
    return {
        "k": k,
        "decoy_ratio": decoy_ratio,
        "sybil_share": sybil_share,
        "emission_per_1k_gates": round(emission * 1000 / GATES, 1),
        "dishonest_emission_share": round(dishonest_earned / emission, 3) if emission else 0.0,
        "honest_net_per_answer": round(per_answer(earned, "honest") - per_answer(cost, "honest"), 3),
        "stamper_gross_per_answer": round(per_answer(earned, "stamper"), 3),
        "guesser_gross_per_answer": round(per_answer(earned, "guesser"), 3),
        "live_integrity": round(live_correct / live_total, 3) if live_total else None,
        # gates an honest agent must answer to afford the MinDisputeBond (100)
        "honest_gates_to_first_bond": (
            round(100 / (per_answer(earned, "honest") - per_answer(cost, "honest")), 1)
            if per_answer(earned, "honest") > per_answer(cost, "honest") else None
        ),
    }


def viable(r):
    """The genesis bar: honest profitable, dishonest marginal, live gates honest."""
    return (
        r["honest_net_per_answer"] > 0
        and r["dishonest_emission_share"] < 0.25
        and (r["live_integrity"] is None or r["live_integrity"] > 0.90)
    )


def main():
    rng = random.Random(SEED)
    results = []
    for k in K_SPACE:
        for dr in DECOY_RATIOS:
            for ss in SYBIL_SHARES:
                r = run_config(k, dr, ss, rng)
                r["viable"] = viable(r)
                results.append(r)

    out = {
        "params": {
            "seed": SEED, "pop": POP, "answerers_per_gate": ANSWERERS_PER_GATE,
            "gates_per_config": GATES, "honest_accuracy": ACC, "g_pass": G_PASS,
            "cost_honest": COST_HONEST, "drip": DRIP,
        },
        "results": results,
    }
    with open(sys.argv[1] if len(sys.argv) > 1 else "mining_beachhead_results.json", "w") as f:
        json.dump(out, f, indent=1)

    hdr = f"{'k':>2} {'decoy':>6} {'sybil':>6} | {'hon net/ans':>11} {'stamp/ans':>9} {'guess/ans':>9} {'dish share':>10} {'live integ':>10} {'->bond':>7} {'VIABLE':>7}"
    print(hdr)
    print("-" * len(hdr))
    for r in results:
        li = f"{r['live_integrity']:.2f}" if r["live_integrity"] is not None else "  -"
        gb = f"{r['honest_gates_to_first_bond']:.0f}" if r["honest_gates_to_first_bond"] else "  inf"
        print(f"{r['k']:>2} {r['decoy_ratio']:>6} {r['sybil_share']:>6} | "
              f"{r['honest_net_per_answer']:>11} {r['stamper_gross_per_answer']:>9} "
              f"{r['guesser_gross_per_answer']:>9} {r['dishonest_emission_share']:>10} "
              f"{li:>10} {gb:>7} {'YES' if r['viable'] else 'no':>7}")


if __name__ == "__main__":
    main()
