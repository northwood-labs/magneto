// Copyright 2026, Northwood Labs, LLC <license@northwood-labs.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package output

import (
	"fmt"
	"strings"

	"go.nwlabs.dev/magneto/internal/models"
)

// RenderSession produces a structured Markdown terminal review record. It
// preserves every available terminal value while emitting event and unavailable
// value sections only when their corresponding data is present.
func RenderSession(session *models.ReviewSessionOutput) string {
	var b strings.Builder

	renderHeader(&b, &session.Metadata, session.TerminalIdempotencyKey)
	renderSummary(&b, &session.Metadata)
	renderSelection(&b, &session.Metadata)
	renderLoadedRubricCriteria(&b, session.Metadata.LoadedRubricCriteria)
	renderFindings(&b, session.Findings)
	renderAttackRound(&b, session.AttackRoundResult)
	renderHumanEscalations(&b, session.HumanEscalations)
	renderHumanOverrides(&b, session.HumanOverrides)
	renderHumanBlockAcceptance(&b, session.HumanBlockAcceptance)
	renderDeadChecks(&b, session.DeadChecks)
	renderDegradationSummary(&b, session.Metadata.DegradedComponents)
	renderUnavailableValues(&b, session)

	return b.String()
}

// renderHeader writes the top-level heading and always-available metadata.
func renderHeader(b *strings.Builder, meta *models.SessionMetadata, idempotencyKey string) {
	fmt.Fprintf(b, "# Adversarial Review: %s\n\n", meta.SpecName)
	fmt.Fprintf(b, "**Session Timestamp:** %s\n", meta.Timestamp)
	fmt.Fprintf(b, "**Artifact:** %s\n", meta.ArtifactPath)
	fmt.Fprintf(b, "**Rounds:** %d of 5\n", meta.RoundsExecuted)
	fmt.Fprintf(b, "**Terminal Status:** %s\n", meta.TerminalStatus)
	fmt.Fprint(b, "**Phase 3 Baseline:** absent (no pre-Phase-1 control baseline)\n")

	if idempotencyKey != "" {
		fmt.Fprintf(b, "**Terminal Idempotency Key:** %s\n", idempotencyKey)
	}

	if meta.LegacyScoreMigrated {
		fmt.Fprint(b, "**Legacy Score Migration:** one or more findings used legacy score "+
			"field; values were mapped to canonical criterion_satisfaction\n")
	}
}

// renderSummary writes the terminal session summary section.
func renderSummary(b *strings.Builder, meta *models.SessionMetadata) {
	fmt.Fprint(b, "\n## Summary\n\n")
	fmt.Fprint(b, "| Field | Value |\n")
	fmt.Fprint(b, "|-------|-------|\n")
	fmt.Fprintf(b, "| Artifact | %s |\n", meta.ArtifactPath)
	fmt.Fprintf(b, "| Session timestamp | %s |\n", meta.Timestamp)
	fmt.Fprintf(b, "| Rounds | %d |\n", meta.RoundsExecuted)
	fmt.Fprintf(b, "| Terminal status | %s |\n", meta.TerminalStatus)
	fmt.Fprint(b, "| Phase 3 baseline | absent — no pre-Phase-1 control baseline |\n")

	if meta.TaskExecutionID != "" {
		fmt.Fprintf(b, "| Task execution ID | %s |\n", meta.TaskExecutionID)
	}

	if meta.SessionID != "" {
		fmt.Fprintf(b, "| Session ID | %s |\n", meta.SessionID)
	}

	if meta.LegacyScoreMigrated {
		fmt.Fprint(b, "| Legacy score migration | yes — mapped to criterion_satisfaction |\n")
	}
}

// renderSelection writes the deterministic selection metadata when it exists.
func renderSelection(b *strings.Builder, meta *models.SessionMetadata) {
	if !hasSelectionMetadata(meta) {
		return
	}

	fmt.Fprint(b, "\n## Selection\n\n")

	if meta.SelectionDecision != "" {
		fmt.Fprintf(b, "**Decision:** %s\n\n", meta.SelectionDecision)
	}

	if meta.SelectionReason != "" {
		fmt.Fprintf(b, "**Reason:** %s\n\n", meta.SelectionReason)
	}

	if meta.FoundationalArtifact {
		fmt.Fprint(b, "**Foundational Artifact:** yes\n\n")
	}

	if meta.SelectionAmbiguous {
		fmt.Fprint(b, "**Classification:** ambiguous\n\n")
	}

	if len(meta.TriggeredBlastRadiusDomains) > 0 {
		fmt.Fprintf(b, "**Triggered Blast-Radius Domains:** %s\n", strings.Join(meta.TriggeredBlastRadiusDomains, ", "))
	}
}

// hasSelectionMetadata reports whether a terminal record has selection data to
// render.
func hasSelectionMetadata(meta *models.SessionMetadata) bool {
	return meta.SelectionDecision != "" || meta.SelectionReason != "" || meta.FoundationalArtifact ||
		meta.SelectionAmbiguous || len(meta.TriggeredBlastRadiusDomains) > 0
}

// renderLoadedRubricCriteria writes the loaded criteria when the rubric was
// available during terminal review.
func renderLoadedRubricCriteria(b *strings.Builder, criteria []string) {
	if len(criteria) == 0 {
		return
	}

	fmt.Fprint(b, "\n## Loaded Rubric Criteria\n\n")

	for _, criterion := range criteria {
		fmt.Fprintf(b, "* %s\n", criterion)
	}
}

// renderFindings writes each finding with its canonical dimensions, evidence,
// deterministic gate result, and applicable confirmation outcome.
func renderFindings(b *strings.Builder, findings []models.ReviewFinding) {
	fmt.Fprint(b, "\n## Findings\n\n")

	if len(findings) == 0 {
		fmt.Fprint(b, "None\n")

		return
	}

	for index := range findings {
		finding := &findings[index]
		fmt.Fprintf(b, "### %s\n\n", finding.CriterionName)
		fmt.Fprintf(b, "**Criterion Satisfaction:** %d/10\n\n", finding.CriterionSatisfaction)
		fmt.Fprintf(b, "**Finding Severity:** %s\n\n", finding.FindingSeverity)
		fmt.Fprintf(b, "**Finding Domains:** %s\n\n", formatFindingDomains(finding.FindingDomains))
		fmt.Fprintf(b, "**Verification Status:** %s\n\n", finding.Status)
		fmt.Fprintf(b, "**Quoted Evidence:** %s\n\n", finding.QuotedExcerpt)
		fmt.Fprintf(
			b,
			"**Artifact Location:** %s § %s\n\n",
			finding.ArtifactLocation.FilePath,
			finding.ArtifactLocation.SectionReference,
		)
		fmt.Fprintf(b, "**Reasoning:** %s\n\n", finding.Reasoning)
		renderCitationGateResult(b, finding.CitationGateResult)
		renderConfirmation(b, finding)

		if index < len(findings)-1 {
			fmt.Fprint(b, "\n---\n\n")
		}
	}
}

// formatFindingDomains renders canonical finding domains in their stable enum
// order and preserves an explicit empty result for historical records.
func formatFindingDomains(domains []models.FindingDomain) string {
	canonical := models.CanonicalFindingDomains(domains)
	if len(canonical) == 0 {
		return "not recorded"
	}

	values := make([]string, len(canonical))
	for index, domain := range canonical {
		values[index] = string(domain)
	}

	return strings.Join(values, ", ")
}

// renderCitationGateResult writes the deterministic gate outcome associated
// with a canonical finding.
func renderCitationGateResult(b *strings.Builder, result *models.CitationGateResult) {
	fmt.Fprint(b, "**Citation Gate:**\n\n")

	if result == nil {
		fmt.Fprint(b, "* No deterministic gate result was recorded.\n")

		return
	}

	fmt.Fprintf(b, "* Schema valid: %t\n", result.SchemaValid)
	fmt.Fprintf(b, "* Citation valid: %t\n", result.CitationValid)

	if result.FailureReason != "" {
		fmt.Fprintf(b, "* Failure reason: %s\n", result.FailureReason)
	}

	if result.MatchedLines != nil {
		fmt.Fprintf(b, "* Matched lines: %d-%d\n", result.MatchedLines.Start, result.MatchedLines.End)
	}

	if result.ProvenanceCorrelationID != "" {
		fmt.Fprintf(b, "* Provenance correlation ID: %s\n", result.ProvenanceCorrelationID)
	}
}

// renderConfirmation writes concrete confirmation evidence and attempts only
// when they apply to the finding.
func renderConfirmation(b *strings.Builder, finding *models.ReviewFinding) {
	if finding.ConfirmerEvidence != "" {
		fmt.Fprintf(b, "\n**Confirmer Evidence:** %s\n", finding.ConfirmerEvidence)
	}

	if len(finding.ConfirmerAttempts) == 0 {
		return
	}

	fmt.Fprint(b, "\n**Confirmer Attempts:**\n\n")

	for _, attempt := range finding.ConfirmerAttempts {
		fmt.Fprintf(b, "* Attempt %d\n", attempt.AttemptNumber)
		fmt.Fprintf(b, "  * Strategy: %s\n", attempt.Strategy)
		fmt.Fprintf(b, "  * Observation: %s\n", attempt.Observation)
		fmt.Fprintf(b, "  * Demonstrated: %t\n", attempt.Demonstrated)

		if attempt.DemonstrationEvidence != "" {
			fmt.Fprintf(b, "  * Demonstration evidence: %s\n", attempt.DemonstrationEvidence)
		}
	}
}

// renderAttackRound writes the attack-round result when an attack round ran.
func renderAttackRound(b *strings.Builder, result *models.AttackRoundResult) {
	if result == nil {
		return
	}

	fmt.Fprint(b, "\n## Attack Round\n\n")

	if !result.NewIssuesFound || len(result.Issues) == 0 {
		fmt.Fprint(b, "**Result:** no new issues found\n")

		return
	}

	fmt.Fprintf(b, "**Result:** surfaced %d new issue(s)\n\n", len(result.Issues))

	for index := range result.Issues {
		issue := &result.Issues[index]
		fmt.Fprintf(
			b,
			"* **%s** (criterion satisfaction: %d/10; severity: %s; domains: %s): %s\n",
			issue.CriterionName,
			issue.CriterionSatisfaction,
			issue.FindingSeverity,
			formatFindingDomains(issue.FindingDomains),
			issue.Reasoning,
		)
	}
}

// renderHumanEscalations writes human escalation details only when an
// escalation was recorded.
func renderHumanEscalations(b *strings.Builder, escalations []models.HumanEscalation) {
	if len(escalations) == 0 {
		return
	}

	fmt.Fprint(b, "\n## Human Escalations\n\n")

	for _, escalation := range escalations {
		fmt.Fprintf(b, "### %s\n\n", escalation.CriterionName)
		fmt.Fprintf(b, "**Question:** %s\n\n", escalation.Question)

		if escalation.InspectedEvidence != "" {
			fmt.Fprintf(b, "**Inspected Evidence:** %s\n\n", escalation.InspectedEvidence)
		}

		if escalation.RemainingAmbiguity != "" {
			fmt.Fprintf(b, "**Remaining Ambiguity:** %s\n\n", escalation.RemainingAmbiguity)
		}

		if escalation.Context != "" {
			fmt.Fprintf(b, "**Evidence Context:** %s\n\n", escalation.Context)
		}

		if escalation.HumanAnswer != "" {
			fmt.Fprintf(b, "**Human Response:** %s\n\n", escalation.HumanAnswer)
		}

		fmt.Fprintf(b, "**Resolved:** %t\n", escalation.Resolved)
	}
}

// renderHumanOverrides writes override decisions only when they were recorded.
func renderHumanOverrides(b *strings.Builder, overrides []models.HumanOverride) {
	if len(overrides) == 0 {
		return
	}

	fmt.Fprint(b, "\n## Human Overrides\n\n")

	for _, override := range overrides {
		fmt.Fprintf(b, "### %s\n\n", override.CriterionName)
		fmt.Fprintf(b, "**Decision:** %s\n\n", override.Decision)
		fmt.Fprintf(b, "**Original Criterion Satisfaction:** %d/10\n\n", override.OriginalCriterionSatisfaction)
		fmt.Fprintf(b, "**Rationale:** %s\n", override.HumanRationale)
	}
}

// renderHumanBlockAcceptance writes an accepted progression block only when
// the corresponding human decision exists.
func renderHumanBlockAcceptance(b *strings.Builder, acceptance *models.HumanBlockAcceptance) {
	if acceptance == nil {
		return
	}

	fmt.Fprint(b, "\n## Human Block Acceptance\n\n")
	fmt.Fprintf(b, "**Criterion:** %s\n\n", acceptance.CriterionName)
	fmt.Fprintf(b, "**Decision:** %s\n\n", acceptance.Decision)
	fmt.Fprintf(b, "**Evidence Context:** %s\n\n", acceptance.EvidenceContext)
	fmt.Fprintf(b, "**Rationale:** %s\n", acceptance.HumanRationale)
}

// renderDeadChecks retains historical dead-check information when it exists.
func renderDeadChecks(b *strings.Builder, deadChecks []string) {
	if len(deadChecks) == 0 {
		return
	}

	fmt.Fprint(b, "\n## Dead Checks\n\n")

	for _, deadCheck := range deadChecks {
		fmt.Fprintf(b, "* %s\n", deadCheck)
	}
}

// renderDegradationSummary writes component criticality and all available
// degradation details.
func renderDegradationSummary(b *strings.Builder, entries []models.DegradationEntry) {
	fmt.Fprint(b, "\n## Degradation Summary\n\n")

	if len(entries) == 0 {
		fmt.Fprint(b, "No degradation events\n")

		return
	}

	for _, entry := range entries {
		fmt.Fprintf(b, "### %s\n\n", entry.Component)
		fmt.Fprintf(b, "**Criticality:** %s\n\n", entry.Criticality)
		fmt.Fprintf(b, "**Failure Mode:** %s\n\n", entry.FailureMode)
		fmt.Fprintf(b, "**Timestamp:** %s\n", entry.Timestamp)

		if len(entry.AffectedCriteria) > 0 {
			fmt.Fprintf(b, "\n**Affected Criteria:** %s\n", strings.Join(entry.AffectedCriteria, ", "))
		}

		if entry.UnavailableValueKey != "" {
			fmt.Fprintf(b, "\n**Unavailable Value Key:** %s\n", entry.UnavailableValueKey)
		}
	}
}

// renderUnavailableValues writes unavailable terminal data only for a partial
// review record, which is the only terminal status allowed to carry it.
func renderUnavailableValues(b *strings.Builder, session *models.ReviewSessionOutput) {
	if session.Metadata.TerminalStatus != models.TerminalPartialReview || len(session.UnavailableValues) == 0 {
		return
	}

	fmt.Fprint(b, "\n## Unavailable Values\n\n")

	for _, unavailable := range session.UnavailableValues {
		fmt.Fprintf(b, "* **%s:** %s\n", unavailable.Field, unavailable.Reason)
	}
}
