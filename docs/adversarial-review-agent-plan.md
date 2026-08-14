# Adversarial Review Agent — Design & Build Plan

**Status:** Planning synthesis, no code written. **Companion document:** `adversarial-review-agent-evidence-ledger.md` (per-source detail, verification notes, full bias/contradiction audit — this plan cites back to it rather than re-arguing source quality inline). Compiled 2026-08-12.

## 0. How to read this document

This plan is a synthesis of ten inputs (one peer-reviewed paper, six anecdotal practitioner reports, three vendor/content pieces — see the ledger for the breakdown) plus independent verification work done during the research phase, plus a deliberate bias-and-contradiction audit run against the full corpus at the requester's instruction. It is opinionated where the evidence supports a decision, and explicit about uncertainty where it doesn't. Where a recommendation rests on weak or contested evidence, that's stated plainly rather than smoothed over — that's the whole point of the exercise this document is itself an instance of.

## 1. Purpose

An adversarial review agent to challenge the user's software design, system architecture, security posture, and developer experience decisions — operating as a structurally independent second party, not a second opinion from the same reasoning process that produced the work. This is a planning document, not an implementation. It must ultimately integrate with **Kiro IDE**.

## 2. Core premises

These operate on two levels simultaneously, per explicit direction: they governed how this document was produced, and they are requirements the built agent must satisfy.

1. **Don't assume. Don't hide confusion. Surface tradeoffs.** → §4 (audit findings stated up front, not buried), §5.8 (explicit human-escalation state for genuine judgment calls the agent must not guess at).

2. **Define success criteria. Loop until verified.** → §5.5 (stopping conditions), the falsifiability standard from Input 3 (a review step must be able to say "if X doesn't hold, this is wrong," not just render an impression).

3. **Score each criterion honestly on a 1–10 scale.** → §5.1–5.7 assume structured, criterion-level scoring (D3's rubric model), not a single holistic verdict.

4. **Provide specific, actionable feedback for any failures.** → §5.6 (deterministic evidence-citation requirement — file, line, quoted evidence, not prose impressions).

5. **Do NOT be generous. Do NOT talk yourself into approving mediocre work. When in doubt, fail it.** → §5.1's default-to-reject stance is the load-bearing design decision of the entire system; this is not a tone instruction, it's an architectural one (see below).

6. **Test EVERY criterion in the contract. Do not skip any.** → §5.7 (periodic audit for dead/unreachable checks — a check that can never fire is not defense in depth, it's decoration).

7. **Specific details: file paths, line numbers, exact messages, expected vs. actual.** → same mechanism as #4.

8. **When code is involved, run the code. Do not just assume it works.** → §5.6's Confirmer pattern — a claim is "hypothesized" until something actually reproduces it, never "confirmed" on reasoning alone.

9. **Check edge cases, not just the happy path.** → part of the structured rubric in §5.3.

## 3. Problem statement

A standard coding agent that both writes and reviews its own work is not doing independent review — it's re-deriving the same conclusion from the same starting assumptions. This isn't a tone or effort problem fixable by better prompting. **Input 8's citation of Panickssery, Bowman & Feng (NeurIPS 2024, independently verified — see ledger) gives this a real mechanism, not just an intuition:** LLM self-preference correlates with self-*recognition* capability, which tracks output familiarity (measurable via perplexity) — and it persists even on outputs human raters judge as equal quality. Instructing a model to "be critical of your own work" doesn't reliably overcome this; masking authorship does. This is the single best-evidenced claim in the whole research corpus, and it's the foundation everything else in this plan is built on.

Every other input in the corpus — from a formally benchmarked paper (D3) down to informal blog posts — converges on the same practical conclusion from a different angle: an autonomous loop that self-certifies its own completion (Ralph), a plan reviewed by the same reasoning that produced it, a PR approved by the model that wrote it — all reproduce this failure mode. The consistent fix proposed across nearly every source, independent of tier or motive, is **structural separation**: a different context, ideally enforced by tooling rather than instruction, evaluating the artifact on its own terms.

## 4. What the evidence actually supports (read this before the architecture)

Full audit in the ledger; the load-bearing points, stated plainly:

* **Context isolation is strongly, convergently evidenced as necessary.** Independent isolation mechanisms appear in D3 (anonymization), Input 4 (fresh process), Input 5 (fresh API call, no shared history), Input 7 (fresh process against a pinned repo), Input 8 (mechanistic explanation via self-recognition), and Input 10 (explicit clean-handoff protocol). No source contradicts this.

* **Cross-provider diversity is not proven necessary — only "helps."** The one source arguing it's load-bearing (Input 8's libfuse claim) is also the one specific claim in the corpus that could not be independently corroborated. Sources with genuine cross-vendor practice (6, 7) didn't isolate that variable from context isolation, which they also had. Treat this as an open empirical question for your own system, not a settled requirement — see §6.

* **"Convergence/consensus = done" is directly falsified**, not just debatable. Input 6 documents a specific case where three independent rounds of agreement on a fix were wrong, caught only by a mandatory fourth round assigned purely to attack the already-agreed fix. Four of the ten designs in this corpus (D3-SAMRE, Ralph, Input 4, Input 5) use a stopping rule vulnerable to exactly this.

* **The adversarial reviewer's own output is not self-certifying.** The two most epistemically careful sources in the corpus (6, 7) both independently concluded the reviewer's claims need a further check — and Input 7 demonstrates a case where source-reproduction produced a _better_ fix than the reviewer itself proposed. Five of the ten designs stop at the reviewer's rendered verdict with no such layer.

* **The single independently-sourced data point that cuts against the whole thesis**: SWR-Bench (verified real, FSE 2026) found a multi-agent debate baseline underperforming a well-prompted single-agent baseline, and found prompt quality mattered more than architecture. This document's recommendations do not ignore this — see the phased, measure-before-you-scale approach in §10.

* **No source in the corpus measures the human cost of being challenged** (Input 3's "corrosive doubt") — it's asserted from introspection once, nowhere quantified, and it's arguably the most consequential unknown for a system whose stated job is to challenge a specific person directly, not gate an anonymous CI pipeline. This document treats it as an unresolved risk requiring your own calibration, not a solved problem — see §5.10.

## 5. Recommended architecture

### 5.1 foundational pattern: Maker-checker with _enforced_, not instructed, isolation

The checker runs in a separate context (subagent, fresh process, or fresh API call — mechanism depends on host tooling, see §7) with:

* No access to the author's reasoning, chat history, or prior review comments (Input 10: prior review comments bias the _next_ author draft toward "what got approved before," not toward correctness — exclude these from the author's context too, not just the checker's).

* Read-only tool access until a review is complete; a separate **Fixer** role, not the reviewer itself, gets write access to apply corrections (Inputs 8, 10).

* An explicit, literal instruction to default to rejection: _"You did not write this. Your job is to find problems. Default to changes-requested; approve only if genuinely sound."_ (near-verbatim from Input 4 — this is Core Premise #5/#6 operationalized as a system prompt, not a hope.)

* Real access to the artifact under review, not just its description — the reviewing agent must be able to read the actual repository, not reason over a plan's prose alone (Input 4's core insight: hallucinated file paths and function signatures are only catchable by an agent that can check).

### 5.2 roles

| Role                                           | Job                                                                                                                                                                    | Notes                                             |
|------------------------------------------------|------------------------------------------------------------------------------------------------------------------------------------------------------------------------|---------------------------------------------------|
| **Author**                                     | Produces the spec, plan, or diff                                                                                                                                       | Never certifies its own output                    |
| **Adversarial Reviewer**                       | Fresh context, read-only, tries to disprove the artifact                                                                                                               | Start with one (see §5.4); default-to-reject      |
| **Confirmer** (conditional)                    | For claims of a specific defect (security, correctness), attempts to actually reproduce/exploit it before the finding counts as "confirmed" rather than "hypothesized" | Directly implements Core Premise #9; from Input 9 |
| **Arbiter** (conditional, only if >1 reviewer) | Consolidates findings, explicitly surfaces _disagreement_ between reviewers as signal, not something to average away                                                   | Input 10                                          |
| **Fixer**                                      | Applies corrections; separate write authority from the Reviewer                                                                                                        | Inputs 8, 10                                      |
| **Human checkpoint**                           | Resolves anything flagged as a genuine judgment call                                                                                                                   | See §5.8                                          |

### 5.3 where in the workflow this operates first

Three intervention points appear across the corpus: pre-code (spec/plan — Inputs 3, 4), in-loop (gating an autonomous dev loop's self-reported completion — the Ralph/D3 combination flagged early in this research), and post-hoc (PR/diff review — Inputs 5, 8, 10, 9).

**Recommendation: start pre-code, at the spec/plan stage.** This is the highest-leverage, cheapest point (Input 3: "move ambiguity upstream, when it's still cheap"), and — see §7 — it maps almost exactly onto Kiro's own Spec artifacts, which already separate "thinking" from "doing." Add post-hoc PR-level review as a second phase once the pattern is validated (§10). **In-loop gating of an autonomous dev loop is explicitly deferred**, not because it's a bad idea — replacing a self-reported `EXIT_SIGNAL` with an independent adversarial verdict is one of the sharpest architectural insights this research surfaced — but because nothing in the input record confirms whether an unattended autonomous loop (Ralph-style) is actually part of your workflow. Building that gate before confirming the loop exists to gate would be solving a problem you may not have. Flagged as an open question in §11.

### 5.4 ensemble size: Start at one, don't assume d3's numbers transfer

D3's finding that ensemble size (k=1→5) was the dominant accuracy lever is the best empirical result in the corpus — but D3 never tested it against an _already context-isolated_ single reviewer. D3's own baseline the whole time was same-context self-review; its ensemble gain is measured relative to that weaker baseline, not relative to a single isolated adversarial reviewer. **This corpus cannot tell you how much of that gain survives once you've already fixed the bigger, better-evidenced lever (isolation).** Start with one reviewer. Instrument its actual catch rate and false-negative rate on your own work (§10, Phase 3). Add a second reviewer only if measured evidence — not the corpus's anecdotes — says a single isolated reviewer is missing things a second independent pass would catch.

### 5.5 stopping conditions — Hybrid, not single-signal

No individual source's stopping rule survives the audit unmodified. Combine:

* **Quantitative convergence or hard budget cap**, whichever comes first (D3, Ralph) — never open-ended.

* **Qualitative novelty check**: stop when critique stops surfacing new, concrete, codebase-specific failure modes and starts repeating (Input 3) — this is a real signal distinct from score convergence.

* **Mandatory post-agreement attack round**: after the reviewer and author appear to agree a fix is sound, run one more adversarial pass whose only job is to attack that agreement, before final approval (Input 6 — this is the direct corrective to the falsified "consensus = done" assumption in §4).

* **A stop condition that is allowed to be "never approved."** Per Input 7: five consecutive rejections is a valid, informative terminal state for a study, not a bug to route around. Your system needs an explicit human override — "I accept this risk and am proceeding anyway" — because the alternative is either the system silently getting bypassed (Input 8's warning) or the human never shipping (Input 3's corrosive doubt).

### 5.6 verifying the reviewer

Two independent sources (6, 7) converged on this as necessary and no design in the corpus's practitioner tier does it. For any finding above a defined severity threshold:

* **Require a Confirmer-style reproduction** where the claim is checkable (Input 9) — a security or correctness claim is "hypothesized" until something actually reproduces it (a working exploit, a failing test that demonstrates the defect, a concrete counter-example), never "confirmed" on reasoning quality alone.

* **Require a deterministic, non-LLM evidence-citation gate** for everything else (Input 5): every finding must include a literal quoted excerpt and location from the artifact under review. No citation → the finding is downgraded to unconfirmed automatically, not silently dropped or silently trusted. This directly operationalizes Core Premise #4 and #8 as a mechanical rule, not a hope that the model complies.

* **Periodically audit the rubric itself** for dead checks — a criterion that can never actually fire, or a field with only one reachable value, creates the illusion of coverage without providing any (Input 7).

### 5.7 Non-LLM deterministic layer

Treat this as first-class, not an afterthought — it's the cheapest and most reliable class of check available, and it doesn't share any LLM's blind spots by construction:

* Citation-gating (§5.6).

* Dead-check audits (§5.6).

* Where a claim is mathematically or structurally checkable rather than a judgment call, prefer a deterministic tool over LLM reasoning — a symbolic/formal check, a type check, a schema validator, an actual test run (Input 6's symbolic-math engine confirming a scoring formula is a working example of this class).

### 5.8 human escalation

Explicit halt-and-ask state (Input 4's `NEEDS_ANSWERS` pattern), distinguishing:

* **Objectively checkable against the repo, spec, or a deterministic tool** → the agent resolves this itself, cites evidence, moves on.

* **Genuine business/product/design judgment** → the agent must halt and ask, never fabricate an answer to keep moving. This is Core Premise #1 as a hard state transition, not a suggestion.

### 5.9 when to invoke this at all

Synthesizing Input 6's "will other automated decisions trust this unchecked" with Input 8's "blast radius" into one combined rule: invoke full adversarial review when **either** condition holds —

* The artifact is foundational — other code, decisions, or automated processes will trust its correctness without re-checking (type invariants, security boundaries, scoring/gating logic, anything a calibration loop depends on), **or**

* Blast radius is high regardless of size (auth, secrets, payment paths, data integrity, irreversible actions).

Skip it — explicitly, not by neglect — for anything a human reviews before it matters and anything where a single revert is the entire cost of being wrong. Running the full process on trivial, reversible work is "theater, not rigor" (Input 6's own words), and per Input 8, over-applying a blocking gate is how teams end up routing around it entirely.

### 5.10 mitigating corrosive doubt (weakest-evidenced part of this plan — Flagged as such)

This is asserted once in the corpus, from introspection, never measured. Treat everything below as a starting hypothesis to calibrate against your own experience, not a validated design:

* Mechanical stop conditions independent of whether the critique still _feels_ valid (§5.5's hard cap) — the point is that the boundary is not emotional.

* Cap the number of issues surfaced per review to force prioritization rather than open-ended piling-on (Input 10).

* Explicitly frame "not yet approved" as sometimes correct and informative (§5.5), paired with a real override so you retain the final call on shipping with accepted, explicit risk.

### 5.11 operational resilience

Any reviewer dependency — a model, a provider, a tool — will eventually be unavailable. Design a degrade path in from the start: substitute, log what was substituted and why, keep going (Input 6). A review pipeline that only works when every dependency is up isn't resilient, it's untested.

## 6. Provider strategy — Does this need multiple agent providers?

**Direct answer: not proven necessary, plausibly beneficial, worth testing empirically rather than assuming.**

|                            | Same-provider, context-isolated                                                      | Cross-provider (e.g., Claude + Codex)                                                                                                                                                                  |
|----------------------------|--------------------------------------------------------------------------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **Evidence for necessity** | Strong — this is the load-bearing, convergent finding across most of the corpus (§4) | Weak — best single evidence (Input 8's libfuse claim) unverified; genuine cross-vendor practice (6, 7) didn't isolate this variable from isolation itself                                              |
| **Mechanism**              | Attacks self-recognition/self-preference directly (verified: Panickssery et al.)     | Adds genuinely different failure-mode distributions on top of isolation — plausible, not measured in this corpus                                                                                       |
| **Cost/complexity**        | Lower — one set of credentials, one integration surface, one context to reason about | Higher — separate auth, separate rate limits/cost tracking, a bridge/plugin layer between ecosystems                                                                                                   |
| **Precedent**              | Native fit for most current agent-hosting IDEs' subagent models (see §7)             | Real, shipped precedent exists (OpenAI's Codex plugin for Claude Code, confirmed real — ledger, Input 10) — the pattern is viable, just not proven necessary                                           |
| **Recommendation**         | **Build v1 here.**                                                                   | **Add as a Phase-3 controlled experiment** (§10) on a sample of high-risk reviews only, to measure whether it catches anything the isolated same-provider reviewer didn't — decide with your own data. |

## 7. Kiro integration plan

Verified against current Kiro documentation (not assumed from stale training knowledge — see search results in this session). Kiro's architecture maps onto this plan unusually cleanly:

* **Subagents** run with their own independent context window and (for custom subagents) their own system prompt and tool permissions — this _is_ the context-isolation mechanism §5.1 requires, natively available rather than something to build. A custom Reviewer subagent with read-only tools is a first-class Kiro concept, not a workaround.

* **MCP** is natively supported in custom agent configuration (`mcpServers` field, per-agent tool allow-listing). This is the natural home for the non-LLM deterministic layer (§5.7) — a citation-gate checker, a symbolic/formal tool, or a Confirmer-style exploit-reproduction tool can all be wired in as MCP servers rather than bespoke integrations, consistent with the broader pattern (Inputs 5 and 10 both converged on MCP as the emerging standard integration surface independently of Kiro).

* **Agent Hooks** (triggered on file save, before/after tool execution, before/after a spec task executes) are the natural gating mechanism for §5.9's "when to invoke this" rule — e.g., trigger adversarial review automatically after a spec's design phase completes, rather than on every keystroke.

* **Steering files** (`.kiro/steering/`, auto-injected into every interaction, project-wide) are the natural home for the review rubric, known anti-patterns, and architecture constraints that Input 10 recommends accumulating over time ("things to add: known anti-patterns, architecture constraints, common bugs from your history").

* **Specs** (requirements → design → tasks, Kiro's own structured planning layer) are the single best-fit target for §5.3's recommended starting point — pre-code adversarial review. Kiro already separates "thinking" from "doing" architecturally; this plan's Phase-1 recommendation is to put the adversarial Reviewer subagent between a Spec's design phase and its task-generation phase, not to invent a new artifact type.

**Two things this plan does not yet know and flags rather than guesses at:**

* Whether Kiro's custom subagents support genuinely different model _providers_ per agent, or only different models within one provider's catalog — this determines whether §6's Phase-3 cross-provider experiment is natively buildable in Kiro or requires an external bridge (à la the OpenAI Codex plugin pattern). Needs a direct check against current Kiro docs before that phase is scoped.

* Kiro's own changelog references a native "per-file code review" feature already shipped. This needs direct investigation before building anything — it may already cover part of §5.3's post-hoc phase, and duplicating it would be wasted work.

## 8. Trade-offs vs. A standard coding agent

| Dimension                                  | Standard coding agent                           | This system                                                                                                                                                   |
|--------------------------------------------|-------------------------------------------------|---------------------------------------------------------------------------------------------------------------------------------------------------------------|
| **Cost**                                   | One pass                                        | Multiple passes minimum (author + reviewer, possibly + confirmer + fixer); see §9 — no reliable multiplier exists across the corpus's incompatible cost units |
| **Latency**                                | Single response                                 | Review round(s) add real wall-clock time before you get a usable artifact                                                                                     |
| **Correctness on self-review blind spots** | Weak by verified mechanism (Panickssery et al.) | Directly targeted by design — this is the entire reason to build it                                                                                           |
| **Noise / false positives**                | N/A                                             | Real, unmeasured risk (Input 9's own gap) — requires the calibration loop in §10 or it erodes trust and gets routed around (Input 8)                          |
| **Psychological cost to the user**         | None beyond normal review fatigue               | Unmeasured, real risk unique to this pattern (§5.10) — the one dimension the corpus is least equipped to answer                                               |
| **Operational complexity**                 | Low                                             | Higher — more roles, more failure points, requires a designed degrade path (§5.11)                                                                            |
| **Auditability**                           | Depends on discipline                           | Structurally better if built per §5.6 — findings carry evidence citations and confirmed/hypothesized status by design                                         |

The case for building this at all rests on §3's verified mechanism (self-review is measurably unreliable) plus your own stated goal (you want to be challenged, not just assisted). The case for caution rests on §4's SWR-Bench counter-evidence and the unmeasured corrosive-doubt risk — this is not a strictly-better upgrade over a standard agent, it's a different tool with real costs that only pays off when applied selectively (§5.9), not universally.

## 9. Cost model caveats

Do not budget off any single number in the source corpus — they are reported in incompatible units (D3: ~$0.31–00.47 per pairwise evaluation; Input 9: $0.69/bug, scaling to "$100–$1000+" per full codebase scan; Input 8: "varies by diff size and model," no figure given). None of these can be combined into a single expected-cost figure for your system. **Instrument your own cost per review from day one of Phase 1** (§10) instead of estimating from these sources.

## 10. Phased build roadmap

* **Phase 0 — Baseline.** Before building anything, informally track how often your current standard coding agent's output has a defect that only surfaces later, for a few weeks. This is the control group the entire research corpus is missing (§4) — without it, you can't tell whether Phase 1 actually helped.

* **Phase 1 — Minimal viable adversarial reviewer, spec-stage only.** One context-isolated Reviewer subagent (§5.1, §5.4), same provider, sitting between a Kiro Spec's design phase and task generation (§7). Citation-gate (§5.6) for every finding. Human escalation state (§5.8) for judgment calls. Hard budget/round cap (§5.5). No ensemble, no cross-provider, no Confirmer yet — get the core isolation mechanism working and trusted first.

* **Phase 2 — Extend to post-hoc PR/diff review**, gated by §5.9's selection rule (blast radius / foundational-trust), not applied to every diff. Add the Confirmer role (§5.6) for security/correctness claims specifically.

* **Phase 3 — Measure, then decide.** False-positive rate, catch rate against the Phase 0 baseline, cost per review in your own units, and an honest self-check on whether you're shipping less (corrosive doubt, §5.10) rather than shipping better. Only with this data should you decide whether ensemble size >1 (§5.4) or a cross-provider second pass (§6) is worth its added cost — as a controlled experiment on a sample, not a wholesale switch.

* **Phase 4 — Conditional: loop-gating.** Only if an unattended autonomous dev loop (Ralph-style) is actually part of your workflow — confirm this first (§11) — replace its self-reported completion signal with this system's independent verdict, per the architectural insight from early in this research: an autonomous loop shouldn't get to grade its own homework any more than a single coding agent should.

## 11. Decisions and remaining open question

Every item originally listed here as an open question has been triaged in `adversarial-review-agent-decisions.md` under a stated policy: low-risk, reversible choices are decided and documented there; anything higher-risk or hard to reverse is presented explicitly rather than guessed. That triage resolved intervention-point sequencing, ensemble starting size, provider strategy, stopping conditions, the verify-the-reviewer layer, human escalation, the review-trigger heuristic, Kiro integration approach, artifact storage format, autonomous-loop scope, adversarial-pressure intensity, and write/fix authority — thirteen decisions in total, each with rationale and an explicit reversibility note.

Two items remain as **facts to verify, not decisions to make** (Kiro's cross-provider subagent support; the scope of Kiro's existing native per-file review feature) — logged in that document's verification backlog, gating Phase 2/3 rather than Phase 1.

The one item that genuinely required your decision rather than a guess — **how Phase 1 actually gets built** — is now resolved: build lean and custom, directly against Kiro's native primitives, rather than forking an existing OSS pattern or integrating packaged third-party tools. See D-000 in that document for the full rationale.

_See `adversarial-review-agent-evidence-ledger.md` for full source-by-source detail, verification notes, and the complete bias/contradiction audit this plan is built on. See `adversarial-review-agent-decisions.md` for the full decision log._
