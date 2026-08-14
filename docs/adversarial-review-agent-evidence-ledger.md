# Evidence Ledger — Adversarial Review Agent Research Inputs

This is the source-of-truth appendix for `adversarial-review-agent-plan.md`. It records what was actually fed into that plan, at what epistemic tier, what I independently verified versus took on faith, and where sources contradict each other. The plan document cites back to this ledger rather than re-litigating source quality inline. Compiled 2026-08-12.

**How to read the tiers:**

* **Tier 1 — Peer-reviewed research.** Independently confirmed to exist, at a real venue, with public code where claimed.
* **Tier 2 — Practitioner report (N=1), no commercial motive.** Anecdotal, single case, self-reported, but no product being sold.
* **Tier 3 — Vendor / content-marketing.** Has a commercial incentive (selling a product, driving traffic to one) to present the pattern favorably.

Nothing in this corpus is Tier 0 (independently replicated, multi-study consensus). That ceiling matters — see the audit at the end of this document.

## Input 1 — "Debate, deliberate, decide (d3)"

**Tier 1.** Verified: real arXiv preprint ([2410.04663](https://arxiv.org/abs/2410.04663), Oct 2024), accepted as a long paper at **EACL 2026**, public code at [github.com/abirharrasse/D3-Judge](https://github.com/abirharrasse/D3-Judge). Authors: Harrasse, Bandi, Bandi. This is the only source in the set with genuine independent peer review. I did not verify this until the audit pass at the end of the conversation — see the audit note below; it held up, but the gap in when I checked it is itself a finding.

**Core claim:** A courtroom-structured multi-agent framework (Advocates → Judge → Jury) for LLM-as-judge evaluation, with two protocols — MORE (parallel one-round, cheap) and SAMRE (iterative multi-round with budgeted stopping, expensive but deeper) — benchmarked against ChatEval, PRD, PandaLM, and a single-judge baseline across MT-Bench, AlignBench, and AUTO-J, measured by accuracy and Cohen's κ against human judgment.

**Key results:** D3-SAMRE reached 86.3% accuracy on MT-Bench vs. 72.5% single-judge / 78.2% ChatEval. Ensemble size (k=1→5 jurors) was the single largest driver of accuracy gain in their own ablation (+8.8%), dwarfing persona-diversity effects (+3.8%, and only significant on subjective tasks — near-zero on objective ones like coding/math). Anonymization measurably reduces positional and self-enhancement bias. Budgeted stopping cut SAMRE token cost 40% with no meaningful accuracy loss (58% of debates converged by round 2).

**What it doesn't cover:** Never executes code — pure text judging of two finished candidate answers. No mechanism for reviewing a single artifact against a spec (only pairwise comparison). Its own Limitations section admits performance is capped by the backbone model's capability.

## Input 2 — "Ralph for claude code"

**Tier 2** (open-source project, single maintainer, no commercial product). Not independently verified beyond reading the README as provided — this is documentation of the tool's own design, not a third-party evaluation of its effectiveness.

**Core claim:** An orchestration harness for autonomous Claude Code development loops — dual-condition exit gate (heuristic completion indicators AND explicit `EXIT_SIGNAL`), circuit breaker for stuck-loop detection, rate/token budgets, session continuity, Docker/E2B sandboxing, GitHub issue lifecycle integration, sequential (non-concurrent) queue processing.

**What it doesn't cover:** No adversarial multi-role structure at all — this is a harness for a _single_ implementer agent. Its exit gate is a **self-report**: the same agent that did the work also emits the `EXIT_SIGNAL` that says the work is done, checked only by pattern-matching heuristics on that agent's own output. No independent verification role exists in this design.

## Input 3 — "Adversarial planning for spec driven development"

**Tier 2** (personal blog post, no product, references an unelaborated side project "Orka-reasoning"). Zero metrics, zero reproducible experiment — pure first-person account.

**Core claim:** A two-role "Planner + Architect" pattern applied at the spec/plan stage, before any code exists. The Architect's job is to find why the plan will fail; the standard against a spec/plan is _falsifiability_ — every step should be attackable by "if this condition doesn't hold, the step is wrong." Names the underlying failure mode precisely: planner LLMs are compliant by default, filling gaps with plausible assumptions rather than pushing back.

**Notable internal tension:** The piece praises the Architect for being maximally unpleasant/paranoid in its opening sections, then later in the same piece warns explicitly about "corrosive doubt" — the adversarial agent eroding the _human's_ confidence and causing paralysis, independent of whether its critiques are individually valid. It never fully reconciles "be as adversarial as possible" with "bound the pressure or it becomes corrosive." Its stopping heuristic — stop when critique starts repeating rather than surfacing new concrete failure modes, filtered through "does this point to a likely failure mode in this codebase, in this release, under these constraints" — is one of the sharper, more actionable ideas in the whole corpus despite the tier.

## Input 4 — "agent-plan-review-loop"

**Tier 2** (open-source project, single developer, no commercial product; `github.com/execute25/agent-plan-review-loop`). Not independently verified beyond the text provided.

**Core claim:** Author → Reviewer loop where every artifact (plan, review, questions, decisions, diffs) is stored as a Markdown file, and the Reviewer runs as a **completely fresh process** with zero access to the Author's conversation/reasoning — only the plan file, the real repository, and its own instructions. Reviewer is instructed to default to `CHANGES_REQUESTED` and approve only if genuinely sound. Complexity-tiered model routing (Haiku classifies task tier; Sonnet/Opus assigned per tier). Explicit human-escalation state (`NEEDS_ANSWERS`) for genuine business-judgment gaps the agent should not guess at. Implementation isolated via git worktree; deploy gated on a pluggable validation command with automatic rollback.

**Load-bearing design flaw, flagged and not resolved by the source itself:** the routing table inverts reviewer strength for its highest-complexity tier — T2 (complex refactors) has Opus authoring but only **Sonnet** reviewing, purely for cost reasons, compensated only by a higher iteration cap (6 vs. 3). This contradicts the general pattern elsewhere in the corpus (see Contradiction #2 below) that review rigor should scale _up_, not down, with stakes. Also: single reviewer only, no ensemble — sits opposite D3's strongest empirical finding (ensemble size as the dominant accuracy lever).

## Input 5 — "AgentDesk"

**Tier 3 — vendor marketing** for a commercial product (MCP server + hosted REST API). The "How This Compares" table is self-authored, with the vendor's own approach landing in the best cell on every axis, no methodology shown.

**Core claim:** Dual independent reviewers, each prompted adversarially, with mandatory consensus (both must pass; either can veto). A **deterministic, non-LLM evidence-citation gate**: every checklist item must include a literal quoted excerpt from the output as evidence; missing citation → automatic fail on that item; >30% of items lacking evidence → whole review capped at score 50. Fresh API call per reviewer, distinct system prompts, no shared conversation history.

**What doesn't hold up:** The foundational causal claim the entire product is built on — "LLM self-review has a systematic leniency bias" — is asserted from what the author "kept finding," with **zero quantification**, the least evidenced version of a claim that Input 1 (D3) and Input 8 (via a verified citation) both measure rigorously elsewhere in this same corpus. The citation-gating mechanism itself is sound and worth keeping regardless of the vendor framing around it.

## Input 6 — "Rotating the hostile seat" (hexisteme, 2026-07-17)

**Tier 2** (personal notes/blog, N=1, no commercial product). Unusually self-aware about its own limits — the source's own FAQ states outright: "That is not a universal rate — it is one system at one point in its life... treat it as one data point, not a formula."

**Core claim:** Three roles (Question / Defend / Attack) rotated through three reviewer groups (the author's own agent fleet, their main coding assistant, and an external multi-model verification pool including non-LLM tools — a symbolic-math engine, a theorem prover) across all six possible role×group permutations, fixed before round one so no participant can angle for a friendlier seat later. Six rounds against eight review targets produced seven confirmed defects. Explicitly runs at least one attack round _after_ apparent multi-party consensus has formed — this is what caught a fix (F6) that three rounds of agreement across different groups had wrongly accepted. Append-only logging of every round, including refuted accusations. Designed degrade path when the external verification tooling itself went down three layers deep in round one, and the review proceeded anyway rather than stalling.

**Honest cost accounting:** 18 individual passes (6 rounds × 3 seats) over 8 targets for one design review — explicitly called "theater, not rigor" if applied to anything small/reversible. Selection heuristic offered: spend the expensive rotation on code other automated decisions will trust unchecked (invariants, scoring gates, calibration ledgers); skip it on anything a human reviews before it matters.

**What I flagged as not fully supported by its own evidence:** the source demonstrates genuine cross-vendor/cross-model-family review in practice, but the specific defects it caught (self-consistency laundering, threshold disagreements, silent overwrites) are generic verification-design failures that plausibly could have been caught by same-vendor context-isolated review too (per Inputs 4, 5, 8, 10). Cross-vendor is shown as _practiced_, not proven _necessary_.

## Input 7 — "Five adversarial reviews told me my study was measuring a fiction" (Study 015, cloudflare OS boundary)

**Tier 2** (research notes, N=1, no commercial product). The single most epistemically disciplined source in the corpus: publishes five _consecutive rejections_ of its own artifact as the actual result, refuses to claim what its machinery doesn't prove, and states plainly that this is the point, not a failure.

**Core claim:** A falsification study on whether a third party can prove, from retained records alone, which authorization decision caused which external action in a pinned platform (Cloudflare OS). Five rounds of **cross-vendor** adversarial review (a different vendor's model, read-only, against the pinned source tree) never let the study freeze. Caught, among other things: a scenario built around a platform configuration that didn't actually exist in the pinned implementation; a "self-consistency" failure where a component checked downstream state against its own prior commitment rather than an independent source; a circular "historical witness" problem where the evidence for a claim came from the same store being examined; a fatal count-vs-identity confusion (1 approved action, 1 external effect, counts matched, but the effect was caused by a _different_ invocation — "count matching is not identity matching"); a "digest-shaped string" mistaken for actual cryptographic evidence; a field name (`appliedAt`) that didn't mean what it appeared to mean; a scripted edit that silently failed while its commit message claimed success.

**The load-bearing, cross-cutting finding:** "**Verify the reviewer.**" Every load-bearing adversarial finding was independently reproduced against source before being trusted — "most reviewer findings held... sometimes the verification produced a better repair than the reviewer proposed." The adversarial reviewer's own output is not self-certifying just because it's well-argued.

## Input 8 — "Adversarial code review: Why the maker shouldn't grade the checker" (Augment code, ani galstian)

**Tier 3 — vendor marketing**, but the most citation-dense source in the corpus, which I spot-checked directly:

| Claim                                                                                                                                     | Status                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  |
|-------------------------------------------------------------------------------------------------------------------------------------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| Panickssery, Bowman & Feng, "LLM Evaluators Recognize and Favor Their Own Generations," NeurIPS 2024                                      | **Confirmed real.** [arXiv 2404.13076](https://arxiv.org/abs/2404.13076). GPT-4/Llama-2 can distinguish their own output; fine-tuning to increase self-recognition correlates with stronger self-preference, even on pairs humans rated equal quality. This is the strongest primary evidence in the whole corpus for the core "self-review is unreliable" claim — better than D3's self-enhancement-rate metric because it's a dedicated study of the mechanism.                                                                       |
| SWR-Bench, "multi-agent architecture ≠ guaranteed stronger review"                                                                        | **Confirmed real, and important.** [arXiv 2509.01494](https://arxiv.org/html/2509.01494v2), FSE 2026, 1000 verified GitHub PRs. Actual finding: refined _prompting_ (not multi-agent architecture) was the single most effective strategy; a _multi-review aggregation_ strategy boosted F1 by up to 43.67%; a two-agent debate baseline (CR-Agent) underperformed a well-prompted single baseline. **This is the one independently-sourced data point in the entire corpus that cuts against the whole thesis** — see the audit below. |
| arXiv 2604.01993 cited for "self-verification produced lower accuracy than external verification"                                         | **Mismatched.** The ID is real but resolves to a different paper ("SAFE: An LLM-as-Verifier Framework for Evidence-Grounded Multi-Hop Reasoning," about external step-level verification improving multi-hop QA accuracy by 8.8pp) — not a general self-vs-external verification head-to-head as characterized. Either mis-cited or the wrong ID was pasted in. Do not cite this specific pairing.                                                                                                                                      |
| libfuse CVE campaign (CVE-2026-33150, CVE-2026-33179; "Codex found 3 bugs Claude-family missed, caught issues in 3 of 19 proposed fixes") | **Not independently corroborated.** Every source repeating these specific numbers traces back to Augment's own content; no independent CVE-database or security-researcher confirmation found.                                                                                                                                                                                                                                                                                                                                          |
| "CVCP protocol" cross-verification                                                                                                        | **Not independently corroborated** — no source found outside this same guide.                                                                                                                                                                                                                                                                                                                                                                                                                                                           |

**Transferable mechanisms that stand regardless of the citation issues:** maker-checker's real institutional lineage (banking dual-control, the Clark-Wilson integrity model, NIST AC-5 separation of duties); a mechanistic (not just correlational) explanation for self-preference — it tracks self-recognition, which tracks familiarity (measured via perplexity), and prompting alone doesn't fix it but authorship-masking does; a concrete five-dimension maker/checker separation checklist (context, system prompt, model, tools, output format); risk-based verifier-budget routing by blast radius, not diff size; an advisory-then-critical-only-gate rollout sequence.

## Input 9 — "Building a secure code review agent" (Hungrysoul, medium, aug 2025)

**Tier 2** (personal blog, informal/hype tone, no product). The only source that admits a real weakness — "it occasionally hallucinates vulnerabilities that don't exist" — without quantifying it anywhere in the piece, despite headlining "56 vulnerabilities found" as an unambiguous win.

**Core claim:** A functional pipeline (not adversarial in the debate/maker-checker sense) — Project Analysis → Chunking → Reasoning (hypothesize) → **Confirmer** (attempt to construct a working exploit) → Reporting, run by a Coordinator — using GPT-5/o4-mini via LangGraph. Found 56 vulnerabilities in a 200K+ LOC production Django app for $38.72 / 41M tokens / 47 minutes ($0.69/bug at this scale; "$100 to $1000+" for larger reviews).

**The one standout mechanism:** the Confirmer Agent — findings aren't reported until a separate role tries to actually reproduce/exploit them, not just reason persuasively about them. This is the clearest automated instantiation anywhere in the corpus of Input 7's "verify the reviewer" principle, applied _within_ a single pipeline rather than left for a human to do afterward.

**The gap:** no false-positive rate, no count of how many "hypothesized" findings the Confirmer actually failed to confirm, no precision measurement at all — the piece doesn't hold its own headline claim to the same evidentiary standard the Confirmer Agent enforces on individual findings.

## Input 10 — "How to set up automated code review with multiple AI agents"

**Tier 3, probable** (no byline, generic "guide" format, heavy internal cross-linking to other similarly-templated guide pages — structurally consistent with Input 8's site, though not confirmed to be the identical vendor). No research citations to fact-check; makes two concrete, checkable claims, both spot-checked:

* "OpenAI Codex plugin for Claude Code" — **confirmed real**, shipped March 30, 2026, explicitly marketed for cross-provider review with nearly identical framing to this guide's own argument. First shipped, first-party confirmation in the corpus that genuine cross-provider adversarial tooling exists as a product. ([AIToolly](https://aitoolly.com/ai-news/article/2026-07-05-openai-launches-codex-plugin-for-claude-code-to-streamline-code-review-and-task-delegation))
* "Mirrors how enterprise teams like Stripe structure their AI coding harnesses" — **real but a loose analogy.** Stripe's "minions" system (1,300 PRs/week) is real and well-documented, but its actual review mechanism leans on deterministic gates, test coverage, synthetic e2e tests, and blue-green rollback — not specifically an adversarial cross-model debate. Directionally honest, not a precise match. ([MindStudio](https://www.mindstudio.ai/blog/what-is-harness-engineering-beyond-prompt-context-engineering))

**Most operationally detailed source in the corpus.** Precise handoff discipline (diff + original requirements only — explicitly excluding both the builder's explanation _and_ historical review comments, since the latter biases toward "what got approved before"). Concrete anti-pattern list (don't let the builder explain itself; don't reuse a session across PRs; don't treat all findings as equal severity; don't skip the rubric). Frames inter-reviewer _disagreement_ as signal, not noise to be averaged away. Proposes an actual ongoing calibration loop — periodically checking validator output against what human reviewers agreed with, treating validator accuracy as something to measure and improve over time, which is the only source-side answer anywhere in the corpus to the false-positive gap Input 9 left open. Its own FAQ concedes cross-provider is a "helps, not required" enhancement over context isolation alone. Sharpest single line in the corpus, directly citing the general form of Input 7's finding without seeming to know it: **tests are not ground truth** — "a test can pass and still be wrong if it was written alongside a buggy implementation"; the spec, not the test suite, must be what the validator checks against.

## Corpus-Wide bias and contradiction audit

> [!TIP]
> Conducted after all ten inputs were collected, at the requester's explicit instruction to not treat provided input as automatically valid.

### Composition

1 peer-reviewed paper (Tier 1), 6 anecdotal single-practitioner reports (Tier 2), 3 vendor/content-marketing pieces (Tier 3). **Every single source describes adversarial review positively.** No source in this set reports it as a net loss for its own use case. This is structural, not incidental: anecdotal sources report on systems their authors built and are motivated to show working; vendor sources are selling something. The only independently-sourced data point anywhere in the corpus that cuts against the whole thesis is the SWR-Bench finding buried inside Input 8 (multi-agent debate underperforming a well-prompted single baseline) — real, verified, and easy to lose among nine other sources pointing the other way.

### Direct contradictions (not just differences of emphasis)

1. **Ensemble size is unresolved across the corpus.** D3: ensemble (k=1→5) is the dominant lever, +8.8%. Input 4: one reviewer. Input 5: two, mandatory dual-veto. Input 6: three, rotated. Input 8: one by default, two only on flagged high-risk paths. No source reconciles its choice against D3's own finding.

2. **Reviewer capability should scale with stakes — except in Input 4, which does the opposite.** Input 4's own routing table gives its highest-complexity tier (T2) a _weaker_ reviewer than author (Sonnet reviewing Opus), for cost reasons, compensated only by more iterations. This directly contradicts Input 8's explicit "add cross-family review on high-risk paths" and Input 6's "spend the rotation on code other automated decisions will trust unchecked" — both argue for _more_ scrutiny where stakes are highest, not less.

3. **Cross-provider: necessary or merely beneficial?** Sources that get adversarial lift from role/context separation alone, same backbone model: D3, Input 3, Input 4. Sources using genuine cross-vendor pools (6, 7) had their specific catches traceable to generic verification-design flaws, not obviously requiring vendor diversity. Input 10's own FAQ: "No, but it helps." The one source arguing cross-provider is load-bearing rather than optional — Input 8's libfuse claim — is also the one specific claim that could not be independently corroborated. **Net read: the corpus supports "context isolation is necessary" far more strongly than "cross-provider is necessary."**

4. **Convergence/consensus as a stopping condition is directly falsified.** Four designs (D3-SAMRE, Ralph, Input 4, Input 5) treat agreement as sufficient grounds to stop. Input 6 shows a concrete case (F6) where three rounds of independent agreement were wrong, caught only by a fourth round specifically assigned to attack the already-agreed fix.

5. **Is the reviewer's own verdict self-certifying?** Five designs (1, 4, 5, 8, 9) terminate at the adversarial layer's rendered verdict. Inputs 6 and 7 — the two most epistemically careful sources — both independently concluded the reviewer's own claims need a further check (fabrication catches, source reproduction).

6. **Tests as ground truth, directly contradicted.** Ralph's `GATE_CMD` and Input 4's deploy gate both use test/build pass as the primary automated safety signal. Input 10 explicitly argues this is insufficient — tests co-authored with a buggy implementation aren't independent evidence; the spec must be the actual ground truth.

7. **How much adversarial pressure is correct is unresolved even within a single source.** Input 3 praises maximal unpleasantness early, then warns about corrosive doubt later, without reconciling the two. Input 9's tone treats maximal paranoia as an unambiguous good with no counterbalance at all. Inputs 6 and 10 both build in structural limits (role rotation; capped issue counts, calibration).

### Self-referential irony

Input 5 argues against unverified self-review while presenting its own core claim with zero verification. Input 8 argues most directly against trusting confident, well-sourced-looking claims — and contains a mismatched citation plus vendor-internal claims that don't independently corroborate. Input 9 builds a mechanism (the Confirmer Agent) specifically to force claims to be proven before being reported, then doesn't apply that same standard to its own headline number.

### What the whole corpus is silent on

* No controlled study anywhere compares multi-agent adversarial review against a single well-prompted thorough review as a baseline, except the unfavorable SWR-Bench data point.

* The "corrosive doubt" cost to the human operator (Input 3) is asserted from introspection, never measured, anywhere in this corpus — despite being possibly the most important constraint for a system whose explicit job is to challenge a specific person directly.

* Only D3's own Limitations section grapples with what happens when the backbone model isn't capable enough to be a useful adversary in the first place; no practitioner or vendor source addresses this at all.

* Cost figures are reported in incompatible units across sources (per-evaluation, per-bug, per-scan, per-PR) and cannot be combined into a single expected-cost figure.
