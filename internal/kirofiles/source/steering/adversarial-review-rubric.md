---
inclusion: manual
---

# Adversarial Review Rubric

## Scoring guidance

Each criterion is scored 1-10 where:

* 1-3: Critical failure, artifact must be revised before proceeding
* 4-6: Significant concerns, revision recommended
* 7-8: Acceptable with minor observations
* 9-10: Exemplary, no concerns

## Criteria

### Context isolation

The reviewer subagent receives only the artifact and rubric, with no access to the author session context.

* Pass example: Design document reviewed with fresh context window containing only the artifact
* Fail example: Reviewer references prior conversation history or author's stated intent

### Citation evidence

Every finding must cite verbatim text from the artifact.

* Pass example: Finding quotes "the system uses shared mutable state" from section Architecture
* Fail example: Finding paraphrases the artifact without a direct quote

### Structural independence

The reviewer cannot modify the artifact or communicate with the author agent.

* Pass example: Reviewer reports findings to orchestrator, author receives summary only
* Fail example: Reviewer edits the design document directly

### Blast-radius awareness

Findings are prioritized based on the downstream impact domain.

* Pass example: Auth-related finding is escalated for human review
* Fail example: Typo in README is treated with same severity as secrets exposure

### Novelty per round

Each review round must surface new failure modes, not repeat prior findings.

* Pass example: Round 2 identifies a new race condition not found in Round 1
* Fail example: Round 3 repeats the same "missing error handling" finding from Round 1
