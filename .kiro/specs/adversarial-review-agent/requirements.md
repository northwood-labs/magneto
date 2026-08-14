# Requirements Document

## Introduction

Phase 1 of the Adversarial Review Agent: a minimal viable adversarial reviewer that operates at the spec/plan stage, integrated natively with Kiro IDE. The system enforces structurally independent review of design artifacts by running a context-isolated reviewer subagent between a spec's design phase and task generation. It is strictly advisory — no autonomous writes, no auto-merge — with the human retaining full control at all times.

The system is built on a verified mechanism: LLM self-preference correlates with self-recognition capability (Panickssery, Bowman & Feng, NeurIPS 2024), making same-context self-review structurally unreliable regardless of prompting. The fix is enforced separation — a different context evaluating the artifact on its own terms.

## Glossary

* **Reviewer_Subagent**: A Kiro subagent running in a fresh, isolated context with read-only tool access, tasked with adversarial review of a design artifact
* **Author**: The agent or human that produced the spec/plan/design artifact under review
* **Confirmer**: A secondary verification role that attempts to reproduce or verify high-severity claims before they are counted as confirmed
* **Citation_Gate**: A deterministic, non-LLM mechanism that validates every finding includes a literal quoted excerpt and location from the artifact under review
* **Steering_File**: A Kiro steering file (`.kiro/steering/`) containing the review rubric, known anti-patterns, and architecture constraints
* **Agent_Hook**: A Kiro hook that triggers automatically on specific events (e.g., PostTaskExec, PostFileSave)
* **MCP_Server**: A Model Context Protocol server providing deterministic, non-LLM tools (citation-gate checker, schema validators)
* **Review_Finding**: A structured output from the Reviewer_Subagent containing criterion, score, evidence citation, and confirmed/hypothesized status
* **Human_Escalation**: An explicit halt state where the system stops and asks a human to resolve a genuine judgment call
* **Review_Session**: A complete adversarial review cycle from trigger to final output (findings or human escalation)
* **Novelty_Check**: A qualitative assessment of whether a critique round surfaces new, concrete, codebase-specific failure modes versus repeating prior findings
* **Attack_Round**: A mandatory adversarial pass run after apparent agreement, specifically tasked with attacking the agreement itself
* **Blast_Radius**: The scope of impact if an artifact's design is wrong — measured by irreversibility, trust dependency, and affected systems

## Requirements

### Requirement 1: Context isolation

**User Story:** As a developer, I want the adversarial reviewer to operate in a completely separate context from the author, so that self-recognition bias cannot compromise review independence.

#### Acceptance criteria

1. WHEN the Reviewer_Subagent is instantiated, THE System SHALL provide a fresh context window containing none of the following Author-session content: the Author's reasoning or chain-of-thought, the Author's chat history, prior Review_Finding outputs from earlier Review_Sessions on the same artifact, or any self-assessment or explanation of intent produced by the Author
2. WHILE a Review_Session is active, THE Reviewer_Subagent SHALL have read-only tool access limited to file-reading and search tools (read_file, grep_search, list_directory, file_search) against the repository and artifact under review, with no access to file-writing, terminal execution, or communication tools
3. THE Reviewer_Subagent SHALL receive an explicit system-level instruction to default to rejection: findings of issues are the expected outcome, approval requires positive evidence of soundness across all rubric criteria
4. WHEN the Reviewer_Subagent is instantiated, THE System SHALL provide direct read access to the actual artifact files and all repository paths explicitly referenced within the artifact (identified by file path literals or directory references appearing in the artifact text), not summaries or descriptions of them
5. WHEN the Reviewer_Subagent is instantiated, THE System SHALL pass the artifact file paths and review rubric via the subagent's environmental_context parameter, constructed exclusively from artifact metadata and Steering_File content without incorporating any content from the Author's active session

### Requirement 2: Trigger heuristic

**User Story:** As a developer, I want adversarial review triggered selectively on high-impact artifacts, so that review effort is concentrated where it prevents the most costly errors without creating blocking overhead on reversible work.

#### Acceptance criteria

1. WHEN a spec's design phase completes AND the artifact produces outputs that downstream automated steps (task generation, code generation) will consume without independent verification, THE System SHALL classify the artifact as foundational and trigger a Review_Session
2. WHEN a spec's design phase completes AND the artifact's domain matches any category in the Blast_Radius domain list maintained in the Steering_File (default categories: auth, secrets, payments, data integrity, irreversible actions), THE System SHALL trigger a Review_Session
3. IF an artifact's scope is limited to a single file or configuration change revertable via one git revert AND the artifact will be reviewed by a human before any downstream automated step consumes it, THEN THE System SHALL skip adversarial review for that artifact
4. THE Agent_Hook SHALL fire on the PostTaskExec event after the design phase of a Kiro spec completes
5. IF the System cannot deterministically classify an artifact as triggering or skipping review (artifact partially matches foundational or Blast_Radius criteria), THEN THE System SHALL default to triggering a Review_Session and log the ambiguous classification for human review

### Requirement 3: Structured findings output

**User Story:** As a developer, I want review findings to be structured with criterion-level scoring and cited evidence, so that I can quickly assess severity and verify claims against the artifact myself.

#### Acceptance criteria

1. THE Reviewer_Subagent SHALL produce each Review_Finding with: criterion name, numeric score (1-10 scale where 1-3 indicates critical deficiency requiring rework, 4-6 indicates partial satisfaction with specific gaps, 7-9 indicates satisfied with minor observations, and 10 indicates fully satisfied with positive evidence), quoted evidence excerpt, artifact location (file path and line range or heading), and a status of confirmed or hypothesized
2. THE Reviewer_Subagent SHALL test every criterion defined in the active Steering_File rubric and report a score for each, skipping none
3. WHEN a Review_Finding references a specific claim about the artifact, THE Reviewer_Subagent SHALL include a literal quoted excerpt from the artifact (minimum 1 sentence, maximum 10 lines of source text) as evidence
4. THE System SHALL store Review_Session output as structured Markdown files under `.kiro/reviews/` using the naming convention `{spec-name}-{ISO-8601-date}-{sequence-number}.md` where sequence-number disambiguates multiple reviews of the same spec on the same date

### Requirement 4: Citation gate

**User Story:** As a developer, I want a deterministic validation layer that mechanically enforces evidence standards on review findings, so that unsupported claims are automatically downgraded regardless of how persuasive the reviewer's reasoning appears.

#### Acceptance criteria

1. WHEN the Reviewer_Subagent produces a Review_Finding, THE Citation_Gate SHALL verify that the finding includes a literal quoted excerpt and an artifact location specified as a file path and section reference
2. IF a Review_Finding lacks a citation (no quoted excerpt or no artifact location), THEN THE Citation_Gate SHALL downgrade that finding's status to "unconfirmed" automatically
3. THE Citation_Gate SHALL operate as a deterministic, non-LLM process implemented as an MCP_Server tool
4. THE Citation_Gate SHALL validate that the quoted excerpt exists as an exact string match within the cited file at or within the cited section of the artifact
5. IF the Citation_Gate performs verbatim validation and the quoted excerpt does not match any text within the cited section of the artifact, THEN THE Citation_Gate SHALL downgrade that finding's status to "unconfirmed" and include an indication that the citation failed verification

### Requirement 5: Confirmer verification for High-Severity claims

**User Story:** As a developer, I want high-severity claims (security, correctness) to be independently verified before being presented as confirmed, so that I do not waste time acting on hallucinated vulnerabilities.

#### Acceptance criteria

1. WHEN a Review_Finding has a severity score at or above a configurable threshold on the 1-10 scale (default: 8) AND concerns security or correctness, THE Confirmer SHALL attempt to reproduce the claimed defect using a separate context with no access to the Reviewer_Subagent's reasoning or intermediate outputs
2. WHILE the Confirmer is verifying a claim, THE System SHALL maintain the finding's status as "hypothesized" until reproduction succeeds or the verification attempt concludes
3. IF the Confirmer cannot reproduce the claimed defect within a configurable verification timeout (default: 3 attempts using available tools), THEN THE System SHALL mark the finding as "unconfirmed" and include the reproduction attempt details: what reproduction strategy was tried, what was observed, and why the defect could not be demonstrated
4. WHEN the Confirmer successfully reproduces a defect, THE System SHALL mark the finding as "confirmed" and attach the reproduction evidence (failing test, concrete counter-example, or exploit demonstration)
5. IF the Confirmer's verification attempt reaches the configured attempt limit without conclusive reproduction or conclusive failure, THEN THE System SHALL mark the finding as "unconfirmed (inconclusive)" and present it to the human with the partial verification details

### Requirement 6: Stopping conditions

**User Story:** As a developer, I want the review process to terminate predictably and not run indefinitely, while still catching the class of error where premature consensus is itself the failure mode.

#### Acceptance criteria

1. THE System SHALL enforce a hard round cap of 5 rounds on any Review_Session, terminating the session when the cap is reached regardless of convergence state
2. WHEN the Reviewer_Subagent's critique in a round surfaces no new concrete failure modes compared to prior rounds (Novelty_Check fails), THE System SHALL stop the review cycle
3. WHEN the Reviewer_Subagent scores all rubric criteria as passing with cited evidence in a round, THE System SHALL execute one mandatory Attack_Round specifically tasked with challenging that agreement before allowing final approval
4. IF the mandatory Attack_Round surfaces new issues, THEN THE System SHALL feed those issues back into the review cycle as a new round (subject to the hard round cap) rather than approving the artifact
5. WHEN a Review_Session exhausts its round cap without the Reviewer_Subagent approving the artifact, THE System SHALL report "not approved" as a valid terminal state with an explicit human override option
6. THE System SHALL cap the number of issues surfaced per review round to 5 findings, ranked by severity, to force prioritization and prevent open-ended accumulation

### Requirement 7: Human escalation

**User Story:** As a developer, I want the system to halt and ask me when it encounters genuine judgment calls rather than fabricating answers to keep moving, so that business and design decisions remain mine.

#### Acceptance criteria

1. WHEN the Reviewer_Subagent identifies a question that is objectively checkable against the repository, spec, or a deterministic tool, THE Reviewer_Subagent SHALL resolve the question itself using available tools and cite the evidence
2. WHEN the Reviewer_Subagent encounters a genuine business, product, or design judgment question that cannot be resolved from available evidence, THE System SHALL enter a Human_Escalation state for that criterion, halt further review progress on that criterion, and continue processing other criteria that do not depend on the unresolved question
3. WHILE in Human_Escalation state, THE System SHALL present the unresolved question with context (what was checked, what remains ambiguous, why it requires human judgment)
4. WHEN the human provides an answer to an escalated question, THE System SHALL record the answer, treat it as an established fact for the remainder of the Review_Session, re-evaluate the halted criterion using the answer, and resume review progress on any criteria that were blocked by the unresolved question
5. WHEN a Review_Session completes with one or more Human_Escalation events, THE System SHALL include each escalated question, the human's answer, and the criterion it unblocked in the Review_Session output stored under `.kiro/reviews/`

### Requirement 8: Steering file rubric

**User Story:** As a developer, I want the review rubric and known anti-patterns maintained as evolving Kiro steering files, so that the adversarial reviewer improves over time as I accumulate project-specific knowledge.

#### Acceptance criteria

1. THE System SHALL load the review rubric from one or more Steering_File entries in `.kiro/steering/` at the start of each Review_Session
2. THE Steering_File SHALL define named criteria, each with a description, scoring guidance (aligned to the 1-10 scale defined in Requirement 3), and at least one example of a pass condition and one example of a fail condition
3. WHEN a Review_Session starts, THE System SHALL evaluate each loaded rubric criterion for reachability against the current project state and flag any criterion that can never fire or has only one reachable value as a dead check in the Review_Session output
4. WHEN a new anti-pattern or architecture constraint is added to a Steering_File, THE System SHALL load and evaluate that criterion in the next Review_Session without requiring code changes or redeployment
5. IF no Steering_File containing rubric criteria exists in `.kiro/steering/` at Review_Session start, THEN THE System SHALL abort the Review_Session and notify the human that no rubric is available for review
6. IF a Steering_File entry is missing any required field (name, description, scoring guidance, or pass/fail examples), THEN THE System SHALL skip that criterion, log it as malformed, and continue the Review_Session with the remaining valid criteria

### Requirement 9: Advisory-Only operation

**User Story:** As a developer, I want the system to be strictly advisory with no autonomous write access, so that I retain full control over all changes until trust is calibrated.

#### Acceptance criteria

1. THE Reviewer_Subagent SHALL have no write access to repository files, spec files, or any project artifact during a Review_Session
2. THE System SHALL present findings and recommendations to the human without applying any changes automatically
3. WHEN the human explicitly overrides a review finding by issuing an override command, THE System SHALL require the human to provide a rationale statement (minimum 1 character), record the override decision with the human-provided rationale in the Review_Session output file under `.kiro/reviews/`, then allow the workflow to continue
4. THE System SHALL not block spec workflow progression (design to tasks) without the human issuing an explicit block-acceptance command; if no human response is received, the default behavior SHALL be to allow progression with a warning annotation that review was not resolved
5. IF the System has outstanding unresolved findings at the point of workflow progression AND the human has neither overridden nor accepted a block, THEN THE System SHALL allow progression, attach a "review-unresolved" annotation to the generated tasks, and log the unresolved findings for future reference

### Requirement 10: Operational resilience

**User Story:** As a developer, I want the review system to degrade gracefully when dependencies are unavailable, so that a broken review tool does not silently block my workflow or silently pass artifacts without review.

#### Acceptance criteria

1. IF the MCP_Server providing the Citation_Gate is unreachable or returns an error on invocation, THEN THE System SHALL log the failure (timestamp, component name, failure reason, affected artifact), mark all findings produced during that session as "unchecked (gate unavailable)," and present them to the human with that status
2. IF the Reviewer_Subagent model is unavailable at session start, THEN THE System SHALL notify the human inline in the IDE workflow that adversarial review was skipped for this artifact, log the reason (timestamp, component name, failure reason), and mark the artifact's review status as "skipped (reviewer unavailable)"
3. WHEN a dependency failure occurs during an in-progress Review_Session, THE System SHALL record what was substituted or skipped and why, complete remaining feasible review steps with degraded status annotations, and produce a final Review_Session output that explicitly lists all degraded components
4. THE System SHALL not mark an artifact's review status as "reviewed" or "approved" when any component of the review pipeline (Reviewer_Subagent, Citation_Gate, Steering_File loading) was degraded or unavailable; the stored Review_Session output SHALL carry a distinct terminal status indicating partial review
5. WHEN a Review_Session completes with any degraded component, THE System SHALL include in the stored Review_Session Markdown file a degradation summary listing each affected component, the failure mode, and which criteria were not fully evaluated

### Requirement 11: Kiro-Native integration

**User Story:** As a developer, I want the adversarial reviewer built on Kiro's native primitives (subagents, MCP, hooks, steering files, specs), so that it integrates seamlessly with my existing workflow without requiring external tooling or parallel systems.

#### Acceptance criteria

1. THE Reviewer_Subagent SHALL be implemented as a Kiro custom subagent with its own system prompt and tool permission configuration, where tool permissions explicitly exclude all file-write and repository-modification tools
2. THE Citation_Gate and Review_Finding schema validators SHALL be implemented as MCP_Server tools registered in the subagent's `mcpServers` configuration, where schema validators verify that each Review_Finding conforms to the required structure (criterion name, numeric score, quoted excerpt, artifact location, and status)
3. WHEN the design phase of a Kiro Spec completes, THE System SHALL trigger a Review_Session via a Kiro Agent_Hook configured for the PostTaskExec event at `.kiro/hooks/<id>.json`
4. THE System SHALL store all configuration, rubrics, and anti-patterns as Kiro Steering_File entries in `.kiro/steering/` that are auto-injected into review interactions by Kiro's native steering mechanism
5. THE System SHALL target Kiro Spec design artifacts as the Phase 1 review subject, operating between the design phase and task generation phase

### Requirement 12: Default-to-Reject stance

**User Story:** As a developer, I want the reviewer to default to rejection rather than approval, so that mediocre work is caught rather than waved through by a compliant review process.

#### Acceptance criteria

1. THE Reviewer_Subagent SHALL receive a system-level instruction that its default disposition is to request changes, and approval requires positive evidence of soundness across all rubric criteria
2. WHEN the Reviewer_Subagent cannot determine whether a criterion is satisfied from available evidence, THE Reviewer_Subagent SHALL score that criterion as failing (score of 1 on the 1-10 scale) rather than passing
3. THE Reviewer_Subagent SHALL not have access to the Author's self-assessment, reasoning, or explanation of intent — only the artifact itself and the repository it references
4. WHEN all rubric criteria receive a passing score (7 or above on the 1-10 scale) with cited evidence, THE Reviewer_Subagent SHALL approve the artifact only after the mandatory Attack_Round (Requirement 6, criterion 3) completes without surfacing new issues (as defined by Novelty_Check)
5. IF the mandatory Attack_Round surfaces new issues after all rubric criteria initially passed, THEN THE System SHALL revoke the pending approval, incorporate the new issues as additional Review_Findings, and return the Review_Session to the active critique state
