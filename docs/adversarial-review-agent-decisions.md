# Adversarial Review Agent — Decision Log

Lightweight ADR-style record. Policy in effect: **low-risk, reversible choices are decided here and documented; anything higher-risk or not easily reversible gets presented explicitly rather than guessed.** Companion to `adversarial-review-agent-plan.md` (design) and `adversarial-review-agent-evidence-ledger.md` (source evidence). Compiled 2026-08-12.

Each entry: the question, what was considered, the decision, why, and — since these are "best guesses to make progress," not final commitments — what it costs to reverse if the guess is wrong.

## Scope & sequencing

### D-001: Which SDLC point does phase 1 target?

* **Options:** pre-code/spec review · post-hoc PR/diff review · gating an autonomous dev loop's self-reported completion.

* **Decision:** Pre-code, at the spec/plan stage.

* **Why:** Cheapest point to catch problems (moves ambiguity upstream), and maps directly onto Kiro's own Spec artifact — no new artifact type needed.

* **Reversibility:** High. Adding a second intervention point later doesn't undo this one; they're additive, not exclusive.

### D-010: Is an unattended autonomous dev loop (Ralph-style) in scope?

* **Decision:** Assume **no** for now. Phase 4 (loop-gating) stays deferred and conditional, not built.

* **Why:** Nothing in the research record confirms this is part of your workflow. Building a gate for a loop that may not exist is effort spent on an unconfirmed premise.

* **Reversibility:** High. If it turns out you do run something like this, Phase 4 activates without needing to undo anything already built — it was never started.

### D-013: How much of the four-part purpose (design, architecture, security, DX) does phase 1 cover?

* **Decision:** Phase 1's rubric centers on design/architecture review, matching the spec-stage intervention point. Security and DX criteria are present in the rubric but not built as dedicated, specialized passes yet.

* **Why:** Architecture review is what a spec-stage reviewer naturally does. Security-specific review benefits from the code-grounded, exploit-confirming pattern (Input 9's Confirmer) that fits post-hoc PR review better than pre-code specs. DX is inherently harder to falsify at spec-time than at usage-time.

* **Reversibility:** High. Expanding the rubric, or adding a dedicated security pass in Phase 2, doesn't require undoing Phase 1.

## Architecture mechanics

### D-002: Starting ensemble size

* **Decision:** One adversarial reviewer, not a jury.

* **Why:** D3's evidence that ensemble size is the dominant accuracy lever was never measured against an _already context-isolated_ single reviewer — it was measured against same-context self-review. That gain may not transfer once isolation (the better-evidenced lever) is already in place. Guessing "more reviewers" without that baseline is guessing twice.

* **Reversibility:** High. Adding a second reviewer is a config change, not a rebuild — see Phase 3 in the plan, where this gets settled with your own measured data instead of the corpus's anecdotes.

### D-003: Provider strategy for v1

* **Decision:** Single provider, matching your existing Claude/Kiro environment. Cross-provider deferred to a measured Phase 3 experiment.

* **Why:** Context isolation is strongly evidenced as necessary; cross-provider diversity is evidenced as "helps" at best, and the one source claiming it's load-bearing (the libfuse claim) is the one specific claim in the whole corpus I couldn't independently verify.

* **Reversibility:** High. A real, working precedent for bridging providers already exists (the OpenAI Codex plugin for Claude Code) — adding a second provider later is additive infrastructure, not a rearchitecture.

### D-004: Stopping-condition model

* **Decision:** Hybrid — quantitative convergence or hard budget/round cap (whichever comes first), plus a qualitative novelty check (stop when critique repeats rather than surfaces new concrete issues), plus one mandatory attack round _after_ apparent agreement, before final approval.

* **Why:** No single source's stopping rule survived the bias audit intact — four designs in the corpus use "consensus = done," which a documented case study directly falsified.

* **Reversibility:** High. Pure logic/prompt configuration, no external dependency.

### D-005: Include the "verify-the-reviewer" layer starting phase 1?

* **Decision:** Yes — citation-gating (every finding needs a quoted, located excerpt) and Confirmer-style reproduction for high-severity claims, from day one, not added later.

* **Why:** This is the layer most practitioner-tier designs in the corpus skip, and it's cheap relative to the rest of the system (a prompt-level requirement plus, for Confirmer, an execution step you likely need anyway per your own Core Premise #9).

* **Reversibility:** High. Can be loosened or tightened per finding-severity threshold at any time.

### D-006: Include human-escalation halt state starting phase 1?

* **Decision:** Yes.

* **Why:** Directly operationalizes your own Core Premise #1. Costs little to include from the start; expensive to retrofit the discipline of "distinguish checkable-from-judgment-call" onto a system that was never designed to make that distinction.

* **Reversibility:** High, though the _habit_ of designing every check to know which bucket it's in is easier to build in from the start than bolt on later — this is the one auto-decision where "reverse later" is technically true but practically annoying.

### D-007: Default rule for when review gets triggered

* **Decision:** Blast-radius / foundational-trust heuristic — invoke on anything foundational (other automated decisions will trust its correctness unchecked) or high blast-radius (auth, secrets, payments, irreversible actions), skip anything a human reviews before it matters or that costs one revert if wrong.

* **Why:** Synthesizes the two most concrete selection heuristics in the corpus (Inputs 6 and 8) into one rule; both independently warn that over-triggering gets the system routed around.

* **Reversibility:** High. Tunable threshold, not a structural commitment.

## Kiro-specific

### D-008: Build on kiro's native primitives, not a parallel system

* **Decision:** Use Kiro Subagents for context isolation, MCP for the deterministic-tool layer, Agent Hooks for triggering, Steering files for the rubric, and Specs as the Phase 1 target artifact.

* **Why:** You already required Kiro-ecosystem integration as a hard constraint; this maps the plan onto Kiro's actual, currently-verified capabilities rather than an assumed abstraction layer.

* **Reversibility:** Medium — this is the one "low-risk" entry closest to the line. Reversing it means moving off Kiro's native model entirely, which would be real rework. It's classified low-risk here specifically _because_ it was independently verified against current docs this session, not guessed — the risk isn't in this decision, it's in the two unverified sub-facts logged below.

### D-009: Findings/artifact storage format

* **Decision:** Structured, human-readable Markdown files under a `.kiro/`-convention-matching directory — not a database, not a proprietary format.

* **Why:** Matches the pattern nearly every credible source in the corpus converged on independently (Ralph's `.ralph/`, agent-plan-review-loop's Markdown artifacts, Kiro's own `.kiro/steering/`) and keeps the audit trail (Core Premise #4/#8) inspectable without tooling.

* **Reversibility:** High early, lower once real review history accumulates in this format — same logic as any schema choice. Worth locking in now while the cost of being wrong is still just "rename some files."

## Safety posture

### D-011: Starting intensity of adversarial pressure

* **Decision:** Conservative default — capped issue counts per review, hard round limits, explicit human override always available (you can always say "I accept this risk, proceed").

* **Why:** This is the single least-evidenced dimension in the whole corpus (asserted once, never measured — "corrosive doubt," Input 3). Starting conservative and increasing pressure later is cheap; starting aggressive and souring your trust in the system early is not.

* **Reversibility:** High as a parameter, but the _experience_ of the first few uses shapes whether you keep using it at all — bias the guess toward the reversible failure mode (too gentle) over the costly one (you stop trusting it).

### D-012: Write/fix authority in phase 1

* **Decision:** Strictly advisory. No autonomous Fixer, no auto-write, no auto-merge. A human applies every change until trust is calibrated.

* **Why:** Directly matches the corpus's explicit, repeated rollout guidance (Input 8: advisory before blocking; Input 4: worktree isolation before any write; Input 10: Fixer is a separate, later-granted role) — this isn't a guess, it's the one place the practitioner-tier sources most consistently agree with each other.

* **Reversibility:** High, and the direction of travel (advisory → write access, once calibrated) is itself the corpus's own recommended sequencing, not just a convenient default.

## Verification backlog (facts to check, not decisions to make)

These gate later phases but aren't choices — they're things to go confirm before spending real effort, not things anyone needs to decide:

* **Does Kiro's custom-subagent system support different model providers, or only different models within one provider's catalog?** Determines whether D-003's Phase 3 cross-provider experiment is natively buildable in Kiro or needs an external bridge. Check before scoping Phase 3.

* **What does Kiro's existing native "per-file code review" feature (referenced in its changelog) actually cover?** Check before starting Phase 2 — may already do part of the job, and duplicating it would be wasted effort.

## Decided by explicit request (not a low-risk guess)

### D-000: How does phase 1 actually get built?

* **Options considered:** (a) build lean and custom, directly against Kiro's native primitives; (b) fork/adapt agent-plan-review-loop's fresh-process, file-artifact pattern and port it to Kiro; (c) integrate packaged tools (Kiro's native per-file review, and/or an MCP-based product like AgentDesk) with custom glue only for the gaps.

* **Decision:** Build lean and custom.

* **Why:** Phase 1 is intentionally small — a single reviewer, single provider, advisory-only, per D-002/D-003/D-012. At that scope, taking on a third-party dependency costs more in integration and trust-surface risk than it saves in effort. Option (b)'s skeleton is shaped for Claude Code CLI, not Kiro, so real porting work is required either way, and it would silently inherit that source's own flagged weaknesses (single reviewer, no verify-the-reviewer layer) unless actively designed out — no net savings once corrected. Option (c) leans on two unverified quantities: Kiro's native review feature's actual scope (logged in the verification backlog), and AgentDesk's core leniency-bias claim, which was the single least-evidenced claim in the entire ten-source corpus (see ledger, Input 5).

* **Reversibility:** This is the one item in this pass that is genuinely not cheap to reverse — real engineering time lands on top of this choice. It was presented explicitly rather than guessed for that reason.

* **Revisit when:** Phase 2+ scope grows past what a lean custom build comfortably covers, or the verification backlog resolves in favor of an existing tool actually covering meaningful ground.

## Follow-on operational workflow decisions

### D-014: How does phase 1 route findings to the confirmer?

* **Decision:** Phase 1 includes a Confirmer only for high-impact security or correctness findings. A finding is high-impact when its explicit severity is `critical` or `high` and its explicit domain includes `security` or `correctness`.

* **Why:** Criterion satisfaction answers whether a rubric criterion is satisfied; severity and domain answer the independent impact question. Routing on a criterion score, including the prior `score >= 8` proposal, conflates those measures and can route low-impact findings or miss high-impact findings.

* **Reversibility:** High. Severity labels and eligible domains can be calibrated after observing review records without changing the isolation, citation, or advisory invariants.

* **Supersedes:** The Phase 1 roadmap statement that deferred Confirmer to Phase 2 and the design's `score >= 8` routing condition are stale. The follow-on operational workflow requirements are the authoritative Phase 1 routing policy; reconcile the roadmap and design during their next approved update.

### D-015: Is phase 0 baseline collection required before phase 1?

* **Decision:** No. Phase 0 is intentionally skipped.

* **Why:** The operational workflow proceeds without delaying Phase 1 for retrospective baseline collection. Any Phase 3 evaluation lacks a pre-Phase-1 control baseline and must state that limitation rather than presenting causal comparisons as measured.

* **Reversibility:** Low for historical comparison: a pre-Phase-1 control cannot be reconstructed after Phase 1 begins. Future measurement can still establish prospective operational baselines.
