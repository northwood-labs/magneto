---
inclusion: manual
---

# Adversarial Review Operational Protocol

## Overview

This protocol defines the Kiro coordinator role for the Adversarial_Review_Operational_Workflow. The coordinator owns transient session state, invokes context-isolated Reviewer and Confirmer subagents, validates findings through deterministic Magneto MCP tools, manages human escalation, and finalizes a terminal review record. It is advisory-only: it never modifies the Design_Artifact, blocks task progression autonomously, or grants mutable capabilities to review roles.

## Pre-task snapshot

Before a Kiro design task starts, the coordinator retains an in-memory SHA-256 digest keyed by the task execution ID and the canonical Design_Artifact path. The digest is computed from the exact pre-task bytes concatenated with the artifact path. It is never written to the Spec, task, or review directories.

At `PostTaskExec`, the coordinator reads the current artifact bytes through the read-only workspace facility and compares the resulting digest against the matching pre-task snapshot:

* Equal digests produce `review-skipped` with reason `unchanged`.
* A missing snapshot entry starts one ambiguous session and records `missing-snapshot` as the ambiguity reason.
* A changed artifact proceeds to selection logic.

## Selection logic

After snapshot comparison establishes a change, the coordinator classifies the artifact:

* **Eligible** — the artifact is foundational or identifies one or more blast-radius domains (`auth`, `secrets`, `payments`, `data-integrity`, `irreversible-actions`). Start exactly one review session.
* **Ineligible** — the trigger classifier deterministically rules the artifact out. Attach `review-skipped` with the classifier reason.
* **Ambiguous** — the classifier cannot decide, design metadata is malformed, or classification evidence is unavailable. Start exactly one review session and record the ambiguous classification in the review record.
* **Selection conflict** — a stored selection reason says unchanged but comparison shows changed. Start exactly one review session and record the conflicting reason.

The ambiguous fallback ensures that uncertain artifacts receive review rather than silent skipping. Every selection decision results in exactly zero or exactly one session; multiple sessions for a single `PostTaskExec` event are never created.

## Read-only capability manifest

Each Reviewer and Confirmer invocation receives an allowlisted capability manifest. The coordinator enforces these boundaries per invocation rather than relying on prompt instruction alone.

### Allowed capabilities

* Read the Design_Artifact at the canonical path
* Read explicitly referenced repository paths declared by the coordinator
* Search the repository using read-only search tools
* Invoke Magneto MCP tools (`validate_citation`, `validate_findings_batch`, `validate_finding_schema`)

### Forbidden capabilities

* Write, edit, or create any file
* Execute shell commands
* Access network resources
* Perform deployment operations
* Mutate source control (commit, push, branch, tag)
* Access or manage secrets
* Invoke task execution or modify task state
* Read author chat history, author reasoning, or author self-assessments

A subagent request for an unlisted path is rejected by the coordinator and recorded as unavailable evidence or degradation as appropriate.

## Reviewer invocation

The coordinator creates one Reviewer invocation per round. The Reviewer receives:

* Canonical artifact path
* Active rubric content
* Round number (1 through 5)
* Allowed repository paths
* Opaque prior-failure fingerprints (criterion name, canonical satisfaction, normalized evidence hash)

The Reviewer does not receive:

* Author chat history or reasoning
* Author self-assessments
* Prior finding reasoning or intermediate Reviewer outputs
* Human answers unrelated to the active criterion
* Confirmer output or demonstration evidence
* Mutable capabilities of any kind

The Reviewer returns at most five candidate findings sorted by severity (`critical`, `high`, `medium`, `low`) and zero or more human escalation proposals. Candidate findings that assert `confirmed`, `citation_valid`, a gate result, or an undeclared field are rejected by the coordinator.

## Confirmer selection

The coordinator selects a Confirmer whose identity differs from the author identity. This is a hard requirement enforced at invocation time; no Confirmer may share the author identity regardless of other conditions.

### Target selection

The coordinator selects every gate-valid high-impact finding for Confirmer invocation. A finding is high-impact when:

* Finding_Severity is `critical` (regardless of domain), or
* Finding_Severity is `high` and Finding_Domain includes `security` or `correctness`

Findings that are not high-impact, have invalid gate results, or have `unchecked (gate unavailable)` status are excluded from Confirmer routing.

### Confirmer input

Each Confirmer invocation receives only:

* Claim text
* Criterion name
* Artifact path
* Section reference
* Quoted excerpt
* Finding_Severity
* Finding_Domain values
* Attempt number (1 through 3)
* Allowed repository paths

It excludes Reviewer reasoning, intermediate output, other findings, author context, and all mutable capabilities.

### Confirmer outcomes

* A demonstrated counter-example, logical contradiction, failing test, or exploit sets status to `confirmed` and ends evaluation.
* No demonstration before final determination retains `hypothesized`.
* Three attempts without demonstration sets status to `unconfirmed`.
* Attempt-detail persistence failure does not change the `confirmed` or final `unconfirmed` status.

## Citation gate validation

The coordinator passes normalized findings through deterministic Magneto MCP validation in this order:

1. `validate_finding_schema` for each candidate — validates required fields, enum values, satisfaction range, domain set, and proposed status.
2. `validate_findings_batch` for schema-valid candidates — validates quoted excerpts against cited file locations using whitespace-normalized matching.

The coordinator rejects any candidate that proposes its own validation results. No Reviewer or Confirmer output can assert `citation_valid`, `confirmed`, schema validity, or gate provenance. Only Magneto tool responses correlated to the current session and finding index establish validation state.

Gate failures set the finding to `unconfirmed`, record the validation failure, and block the finding from Confirmer routing and approval evaluation until validation succeeds in a subsequent round.

When the Citation_Gate is unavailable (transport or component failure), affected findings receive status `unchecked (gate unavailable)` and are presented to the human. This constitutes a required-component failure that prevents `approved` terminal status.

## Round management

The coordinator enforces these bounds:

* Maximum 5 ordinary Reviewer rounds per session
* Maximum 5 findings per round, sorted by severity
* Novelty check between rounds — a round with no new failure mode (according to the novelty detector) ends ordinary rounds
* One mandatory attack round before approval — when every active criterion has gate-valid satisfaction of 7 through 10, the coordinator executes one attack round
* A novel attack finding returns the session to ordinary rounds if fewer than 5 rounds have executed
* No novel attack finding makes approval eligible (subject to required-component and human-decision rules)
* Exhaustion of 5 rounds without an approved result produces terminal status `not_approved`

## Human escalation

The coordinator escalates to the human when the Reviewer identifies a question that the Design_Artifact, repository evidence, and deterministic tools cannot resolve.

### When to escalate

* Evidence required to evaluate a criterion is unavailable from the artifact, repository, and all deterministic tools
* The Reviewer explicitly identifies the question as unresolvable from available sources

### What to present

The coordinator presents:

* The specific question requiring human input
* Evidence the Reviewer inspected while attempting resolution
* Remaining ambiguity that prevents determination
* The affected criterion name

### Resume behavior

* Independent criteria continue evaluation during the escalation
* When the human supplies an answer, the coordinator records it, re-evaluates the affected criterion, and resumes dependent review work
* When unresolved findings remain at design-to-task progression and the human has not accepted a block, progression continues with a `review-unresolved` annotation
* When the human accepts a block, the coordinator prevents progression
* When the human overrides a finding, the coordinator records the decision and a non-empty rationale

## Terminal finalization

When all review work and applicable human decisions are complete, the coordinator invokes `finalize_review_session` on the Magneto MCP server with the terminal session data.

### Terminal status precedence

The finalizer applies this precedence:

1. A valid human override with non-empty rationale produces `human_override`
2. A required-component degradation produces `partial_review`
3. Attack-round success (no novel findings) with all criteria gate-valid at 7-10 produces `approved`
4. All other cases produce `not_approved`

### Idempotency

The finalization uses an idempotency key composed from task execution ID and terminal-session ID. A retry with the same key returns the original record path without creating a second record.

### Record content

The terminal record includes all available metadata, findings, gate results, confirmation attempts, selection metadata, attack-round results, degradation entries, and human events. Unavailable values are rendered with reasons only in `partial_review` records. The record always labels Phase 3 analysis as lacking a pre-Phase-1 control baseline.

## Non-blocking annotations

All annotations produced by this workflow are advisory and never block the Kiro task workflow:

* `review-skipped` — the artifact was unchanged or deterministically ineligible; recorded with path and reason
* `review-trigger-failed` — session start, annotation persistence, or record persistence failed; the human is notified and the workflow continues
* `review-unresolved` — unresolved findings remain at progression and the human has not accepted a block; progression continues with explicit risk acknowledgment

These annotations inform the human without autonomously preventing task execution, deployment, or progression decisions.

## Degradation handling

### Required components

Failure of a required component prevents the session from reaching `approved` terminal status:

* Pre-task snapshot comparison (ambiguous fallback available)
* Rubric loading
* Reviewer invocation and read access
* Citation_Gate schema and citation validation
* Confirmer invocation for each gate-valid high-impact target

### Optional components

Failure of an optional component is recorded but does not prevent approval evaluation:

* Confirmer attempt-detail persistence
* Phase 3 baseline metadata and comparative analytics

### Failure behavior

* A required failure without human override finalizes as `partial_review` with available results and identified unavailable values
* An optional failure alone preserves normal approval evaluation and records the failure
* A human override of any degraded condition finalizes as `human_override`, preserves all degradation entries, and requires non-empty rationale
* A required failure before review can start emits `review-trigger-failed`, records the skip reason when persistence is available, and allows the workflow to continue with a warning annotation
