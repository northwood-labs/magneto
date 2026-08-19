# Requirements Document

## Introduction

The Adversarial_Review_Operational_Workflow makes the existing Phase 1 adversarial review components operate as one Kiro-native, end-to-end workflow for changed Kiro Spec design artifacts. The workflow is advisory-only: it selects eligible design artifacts at `PostTaskExec`, performs context-isolated read-only review, validates findings deterministically, conditionally verifies high-impact security and correctness claims, records an auditable result, and allows the human to continue with explicit risk.

**Scope decisions:** Phase 1 includes a Confirmer only for high-impact security or correctness findings. Criterion satisfaction is distinct from impact: a rubric score records criterion satisfaction and does not determine Confirmer routing. Phase 0 baseline collection is intentionally skipped. Consequently, future Phase 3 evaluation has no pre-Phase-1 control baseline and must label that limitation in any comparison of review outcomes.

## Glossary

* **Adversarial_Review_Operational_Workflow**: The Kiro-native process that orchestrates review of an eligible Kiro Spec design artifact.
* **Design_Artifact**: The `design.md` document for a Kiro Spec.
* **PostTaskExec_Hook**: The Kiro hook event emitted after a spec task completes.
* **Trigger_Classifier**: The deterministic component that decides whether a changed Design_Artifact is eligible for review.
* **Foundational_Artifact**: A Design_Artifact whose outputs downstream automated steps consume without independent verification.
* **Blast_Radius_Domain**: One of `auth`, `secrets`, `payments`, `data-integrity`, or `irreversible-actions`.
* **Reviewer**: A context-isolated Kiro subagent that reads an eligible Design_Artifact and repository evidence to produce Review_Findings.
* **Confirmer**: A context-isolated Kiro subagent that independently attempts to demonstrate a selected Review_Finding.
* **Review_Finding**: A structured observation about one rubric criterion with a criterion satisfaction score, severity, domain, evidence citation, and verification status.
* **Criterion_Satisfaction**: An integer from 1 through 10 that records how fully a Review_Finding's rubric criterion is satisfied.
* **Finding_Severity**: Exactly one impact value: `critical`, `high`, `medium`, or `low`.
* **Finding_Domain**: One or more category values selected from `security`, `correctness`, `architecture`, `reliability`, `operations`, or `developer-experience`.
* **High_Impact_Finding**: A Review_Finding with Finding_Severity `critical`, regardless of Finding_Domain, or Finding_Severity `high` with a Finding_Domain of `security` or `correctness`.
* **Citation_Gate**: The deterministic, non-LLM validator that checks a Review_Finding's schema, quoted excerpt, file path, and cited artifact location.
* **Review_Session**: One complete workflow execution for one eligible Design_Artifact.
* **Novelty_Check**: The deterministic comparison that identifies whether a review round contains a concrete failure mode not already recorded in that Review_Session.
* **Attack_Round**: The mandatory Reviewer pass that challenges an apparent approval before the Review_Session can be approved.
* **Human_Escalation**: A paused criterion requiring a question that repository, artifact, and deterministic-tool evidence cannot resolve.
* **Required_Component**: A workflow component whose failure prevents a Review_Session from establishing a complete review result.
* **Optional_Component**: A workflow component whose failure is recorded but does not by itself prevent a complete review result.
* **Degraded_Session**: A Review_Session in which a required workflow component was unavailable, malformed, or failed.
* **Review_Record**: The persisted Markdown output containing a Review_Session's inputs, results, decisions, and degradation information.

## Requirements

### Requirement 1: Gate changed design artifacts

**User Story:** As a developer, I want the workflow to select changed, consequential design artifacts at the Kiro design checkpoint, so that advisory review effort is focused where downstream mistakes cost the most.

#### Acceptance criteria

1. WHEN the PostTaskExec_Hook reports completion of a Kiro Spec design task, THE Adversarial_Review_Operational_Workflow SHALL compare the Design_Artifact with the version observed before that task.
2. WHEN the PostTaskExec_Hook reports a changed Design_Artifact that is a Foundational_Artifact or identifies one or more Blast_Radius_Domains, THE Adversarial_Review_Operational_Workflow SHALL start exactly one Review_Session.
3. IF the Trigger_Classifier cannot classify a changed Design_Artifact as eligible or ineligible, THEN THE Adversarial_Review_Operational_Workflow SHALL start exactly one Review_Session and record the ambiguous classification in the Review_Record.
4. IF comparison establishes that a Design_Artifact is unchanged, THEN THE Adversarial_Review_Operational_Workflow SHALL skip the Review_Session and record the selection reason.
5. IF the Trigger_Classifier classifies a changed Design_Artifact as ineligible, THEN THE Adversarial_Review_Operational_Workflow SHALL skip the Review_Session and record the selection reason.
6. IF Review_Session start or ambiguous-classification persistence fails, THEN THE Adversarial_Review_Operational_Workflow SHALL notify the human, record the trigger failure when persistence is available, and allow the Kiro Spec workflow to continue with a `review-trigger-failed` warning annotation.
7. WHEN a recorded unchanged selection reason conflicts with comparison that establishes a changed Design_Artifact, THE Adversarial_Review_Operational_Workflow SHALL start exactly one Review_Session and record the conflicting selection reason.

### Requirement 2: Isolate read-only review roles

**User Story:** As a developer, I want independent, read-only review roles, so that reviewer conclusions are based on the artifact and repository evidence rather than author context or mutable project state.

#### Acceptance criteria

1. WHEN the Adversarial_Review_Operational_Workflow starts a Review_Session, THE Reviewer SHALL receive a fresh context that excludes Author chat history, Author reasoning, Author self-assessments, and prior Review_Finding reasoning.
2. WHILE a Review_Session is active, THE Reviewer SHALL access only the Design_Artifact, explicitly referenced repository paths, active rubric content, and read-only file and search tools.
3. WHEN the Adversarial_Review_Operational_Workflow invokes a Confirmer, THE Confirmer SHALL receive a fresh context that excludes Reviewer reasoning and intermediate Reviewer outputs.
4. WHEN the Adversarial_Review_Operational_Workflow selects a Confirmer, THE Adversarial_Review_Operational_Workflow SHALL select a Confirmer whose identity differs from the Author identity.
5. WHILE the Confirmer is active, THE Confirmer SHALL use read-only repository access and the claim statement, artifact location, severity, and domain values needed to attempt independent verification.
6. WHEN the Reviewer cannot establish a rubric criterion from available evidence, THE Reviewer SHALL record the criterion as unsatisfied with Criterion_Satisfaction `1` and cite the missing evidence.
7. WHEN clear available evidence establishes that a rubric criterion is satisfied, THE Reviewer SHALL record the criterion as satisfied with Criterion_Satisfaction from 2 through 10.

### Requirement 3: Record satisfaction separately from impact

**User Story:** As a developer, I want the workflow to distinguish rubric satisfaction from finding impact, so that high-impact verification is not distorted by a criterion score.

#### Acceptance criteria

1. WHEN the Reviewer submits a Criterion_Satisfaction outside the range from 1 through 10, THE Adversarial_Review_Operational_Workflow SHALL clamp the value to the nearest bound before storing the Review_Finding.
2. THE Reviewer SHALL record exactly one Criterion_Satisfaction integer from 1 through 10 for every active rubric criterion in a Review_Session.
3. THE Reviewer SHALL record exactly one Finding_Severity value of `critical`, `high`, `medium`, or `low` for every Review_Finding.
4. THE Reviewer SHALL record one or more Finding_Domain values selected from `security`, `correctness`, `architecture`, `reliability`, `operations`, or `developer-experience` for every Review_Finding.
5. WHEN the Adversarial_Review_Operational_Workflow selects Confirmer targets, THE Adversarial_Review_Operational_Workflow SHALL select every Review_Finding with Finding_Severity `critical` and every Review_Finding with Finding_Severity `high` and a Finding_Domain of `security` or `correctness`.
6. WHEN the Adversarial_Review_Operational_Workflow reports Criterion_Satisfaction, THE Adversarial_Review_Operational_Workflow SHALL determine apparent approval eligibility from Criterion_Satisfaction and Citation_Gate validation.

### Requirement 4: Validate findings deterministically

**User Story:** As a developer, I want each finding validated by deterministic schema and citation checks, so that unsupported claims are visible and cannot acquire confirmed status through persuasive prose.

#### Acceptance criteria

1. WHEN the Reviewer produces a Review_Finding, THE Citation_Gate SHALL validate that the Review_Finding contains a criterion name, Criterion_Satisfaction, Finding_Severity, Finding_Domain, quoted excerpt, artifact location, status, and reasoning.
2. WHEN the Citation_Gate validates a quoted excerpt, THE Citation_Gate SHALL complete the established whitespace normalization before verifying that the excerpt occurs within the cited file and cited section.
3. IF the Citation_Gate detects a missing required field, absent citation, invalid citation location, or unmatched quoted excerpt, THEN THE Citation_Gate SHALL set the Review_Finding status to `unconfirmed`, record the validation failure, and block the Review_Finding from Confirmer routing and approval evaluation until validation succeeds.
4. WHEN a Review_Finding passes Citation_Gate validation, THE Adversarial_Review_Operational_Workflow SHALL block the Review_Finding from receiving `confirmed` status until applicable Confirmer verification succeeds.
5. THE Citation_Gate SHALL perform schema and citation validation without LLM inference.
6. IF the Citation_Gate detects LLM-derived validation, THEN THE Citation_Gate SHALL discard the LLM-derived validation, record the detection, and continue using only deterministic non-LLM validation.

### Requirement 5: Confirm high-impact claims

**User Story:** As a developer, I want only high-impact security or correctness claims independently tested before confirmation, so that verification effort is proportional to risk without treating rubric scores as severity.

#### Acceptance criteria

1. WHEN a Review_Finding is a High_Impact_Finding and passes Citation_Gate validation, THE Adversarial_Review_Operational_Workflow SHALL invoke the Confirmer for that Review_Finding.
2. WHILE the Confirmer evaluates a High_Impact_Finding without demonstration evidence, THE Adversarial_Review_Operational_Workflow SHALL retain the `hypothesized` status for that Review_Finding.
3. WHEN the Confirmer demonstrates a concrete counter-example, logical contradiction, failing test, or exploit that establishes the High_Impact_Finding, THE Adversarial_Review_Operational_Workflow SHALL set the Review_Finding status to `confirmed`, attach the demonstration evidence, and end Confirmer evaluation for that Review_Finding.
4. WHEN Confirmer evaluation ends without demonstration evidence and before final determination, THE Adversarial_Review_Operational_Workflow SHALL retain the `hypothesized` status for the High_Impact_Finding.
5. WHEN the Confirmer completes three verification attempts and demonstration evidence exists for the High_Impact_Finding, THE Adversarial_Review_Operational_Workflow SHALL retain the `confirmed` status even when attempt-detail persistence fails.
6. IF the Confirmer completes three verification attempts without demonstration evidence for the High_Impact_Finding, THEN THE Adversarial_Review_Operational_Workflow SHALL set the Review_Finding status to `unconfirmed` even when attempt-detail persistence fails.
7. WHEN attempt-detail persistence succeeds for an unconfirmed High_Impact_Finding, THE Adversarial_Review_Operational_Workflow SHALL record each attempt strategy and observation.
8. WHEN a Review_Finding is not a High_Impact_Finding, THE Adversarial_Review_Operational_Workflow SHALL retain the Citation_Gate result without invoking the Confirmer.

### Requirement 6: Bound review rounds and challenge agreement

**User Story:** As a developer, I want review iterations to end predictably while challenging premature agreement, so that the workflow catches novel problems without becoming open-ended.

#### Acceptance criteria

1. THE Adversarial_Review_Operational_Workflow SHALL execute no more than five Reviewer rounds in one Review_Session.
2. WHEN a completed Reviewer round contains no Review_Finding with a failure mode that is new according to the Novelty_Check, THE Adversarial_Review_Operational_Workflow SHALL end ordinary Reviewer rounds.
3. WHEN every active rubric criterion has a Criterion_Satisfaction of 7 through 10 and each supporting Review_Finding passes Citation_Gate validation, THE Adversarial_Review_Operational_Workflow SHALL execute one Attack_Round before recording approval.
4. IF the Attack_Round produces a Review_Finding with a new failure mode, THEN THE Adversarial_Review_Operational_Workflow SHALL continue ordinary Reviewer rounds when fewer than five Reviewer rounds have executed.
5. WHEN five Reviewer rounds execute without an approved result, THE Adversarial_Review_Operational_Workflow SHALL record terminal status `not_approved`.
6. THE Reviewer SHALL record no more than five Review_Findings in each Reviewer round, ordered from highest to lowest Finding_Severity.

### Requirement 7: Escalate unresolved questions and remain advisory

**User Story:** As a developer, I want the workflow to ask me about questions that available evidence cannot resolve while leaving all changes and progression decisions under my control, so that it informs rather than autonomously governs my work.

#### Acceptance criteria

1. WHEN the Reviewer identifies a question resolvable from the Design_Artifact, repository evidence, or a deterministic tool, THE Reviewer SHALL resolve the question and cite the evidence.
2. WHEN the Reviewer identifies a question unavailable from Design_Artifact, repository, and deterministic-tool evidence, THE Adversarial_Review_Operational_Workflow SHALL create a Human_Escalation for the affected criterion and continue independent criteria.
3. WHILE a Human_Escalation remains unresolved, THE Adversarial_Review_Operational_Workflow SHALL present the question, inspected evidence, remaining ambiguity, and affected criterion to the human.
4. WHEN the human supplies an answer to a Human_Escalation, THE Adversarial_Review_Operational_Workflow SHALL record the answer, re-evaluate the affected criterion, and resume dependent review work.
5. THE Adversarial_Review_Operational_Workflow SHALL present findings without modifying repository files, Spec artifacts, task artifacts, or review configuration.
6. WHEN unresolved findings remain at design-to-task progression and the human accepts a block, THE Adversarial_Review_Operational_Workflow SHALL prevent progression.
7. WHEN unresolved findings remain at design-to-task progression and the human has not accepted a block, THE Adversarial_Review_Operational_Workflow SHALL allow progression with a `review-unresolved` annotation.
8. WHEN the human overrides a finding or accepts a block, THE Adversarial_Review_Operational_Workflow SHALL record the human decision and a non-empty rationale in the Review_Record.

### Requirement 8: Persist an auditable review record

**User Story:** As a developer, I want durable review records, so that I can inspect what ran, what evidence supported each result, and what decisions were accepted over time.

#### Acceptance criteria

1. WHEN a Review_Session reaches a terminal status, THE Adversarial_Review_Operational_Workflow SHALL persist one Review_Record under `.kiro/reviews/` named `{spec-name}-{ISO-8601-date}-{sequence-number}.md` regardless of the executed Reviewer round count.
2. THE Review_Record SHALL include the Design_Artifact path, session timestamp, triggered Blast_Radius_Domains or Foundational_Artifact classification, loaded rubric criteria, executed round count, terminal status, and Attack_Round result when the corresponding values are available.
3. THE Review_Record SHALL include every available Review_Finding with Criterion_Satisfaction, Finding_Severity, Finding_Domain, verification status, quoted evidence, artifact location, Citation_Gate result, and Confirmer evidence or attempts when applicable.
4. WHEN a Review_Session includes a Human_Escalation, override, or block acceptance, THE Review_Record SHALL include the criterion, question or decision, evidence context, human response, and rationale.
5. WHEN a Review_Session contains no Human_Escalation, override, or block acceptance, THE Review_Record SHALL omit the corresponding event fields.
6. WHEN the Adversarial_Review_Operational_Workflow records evaluation outcomes for Phase 3 analysis, THE Adversarial_Review_Operational_Workflow SHALL label the absence of a pre-Phase-1 control baseline.
7. WHEN a Review_Session reaches terminal status `partial_review` and an otherwise required Review_Record value is unavailable, THE Adversarial_Review_Operational_Workflow SHALL persist the available Review_Record values and identify each unavailable value with its reason.

### Requirement 9: Surface degraded review status

**User Story:** As a developer, I want dependency failures represented explicitly, so that unavailable review components neither silently block the workflow nor create a false approval.

#### Acceptance criteria

1. IF the Reviewer, Confirmer, Citation_Gate, or rubric loading component is unavailable or fails during a Review_Session, THEN THE Adversarial_Review_Operational_Workflow SHALL create a Degraded_Session and record the component, failure mode, timestamp, and affected criteria when Review_Record persistence is available.
2. WHEN access controls prevent a Reviewer from accessing a Design_Artifact, THE Adversarial_Review_Operational_Workflow SHALL create a Degraded_Session, record incomplete evaluation for the affected criteria, and continue the available review work.
3. WHEN the Citation_Gate is unavailable, THE Adversarial_Review_Operational_Workflow SHALL set each affected Review_Finding status to `unchecked (gate unavailable)` and present the affected Review_Findings to the human.
4. WHEN a Degraded_Session reaches a terminal status without a human override, THE Adversarial_Review_Operational_Workflow SHALL record terminal status `partial_review` and persist the available results.
5. WHEN a human overrides any Degraded_Session condition, including Reviewer unavailability, THE Adversarial_Review_Operational_Workflow SHALL record terminal status `human_override`, preserve the Degraded_Session information, record an auditable non-empty rationale, and allow the Kiro Spec workflow to continue.
6. WHILE a Required_Component failure occurs in a Review_Session, THE Adversarial_Review_Operational_Workflow SHALL not record terminal status `approved` regardless of Review_Record persistence.
7. WHEN an Optional_Component failure occurs without a Required_Component failure, THE Adversarial_Review_Operational_Workflow SHALL continue approval evaluation and record the optional component failure.
8. WHEN a Required_Component fails before review can start, THE Adversarial_Review_Operational_Workflow SHALL notify the human, record the skip reason when Review_Record persistence is available, and allow the Kiro Spec workflow to continue with a warning annotation.
