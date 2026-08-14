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

// RenderSession produces a structured Markdown review output from the
// session data. The output includes all review sections: summary,
// findings, attack round, human escalations, human overrides, dead
// checks, and degradation summary.
func RenderSession(session *models.ReviewSessionOutput) string {
	var b strings.Builder

	renderHeader(&b, &session.Metadata)
	renderSummary(&b, &session.Metadata)
	renderFindings(&b, session.Findings)
	renderAttackRound(&b, session.AttackRoundResult)
	renderHumanEscalations(&b, session.HumanEscalations)
	renderHumanOverrides(&b, session.HumanOverrides)
	renderDeadChecks(&b, session.DeadChecks)
	renderDegradationSummary(&b, session.Metadata.DegradedComponents)

	return b.String()
}

// renderHeader writes the top-level heading and metadata table.
func renderHeader(b *strings.Builder, meta *models.SessionMetadata) {
	fmt.Fprintf(b, "# Adversarial Review: %s\n\n", meta.SpecName)
	fmt.Fprintf(b, "**Date:** %s\n", meta.Timestamp)
	fmt.Fprintf(b, "**Artifact:** %s\n", meta.ArtifactPath)
	fmt.Fprintf(b, "**Rounds:** %d of 5\n", meta.RoundsExecuted)
	fmt.Fprintf(b, "**Terminal Status:** %s\n", string(meta.TerminalStatus))
}

// renderSummary writes the summary section with a metadata table.
func renderSummary(b *strings.Builder, meta *models.SessionMetadata) {
	fmt.Fprint(b, "\n## Summary\n\n")
	fmt.Fprint(b, "| Field | Value |\n")
	fmt.Fprint(b, "|-------|-------|\n")
	fmt.Fprintf(b, "| Artifact | %s |\n", meta.ArtifactPath)
	fmt.Fprintf(b, "| Date | %s |\n", meta.Timestamp)
	fmt.Fprintf(b, "| Rounds | %d |\n", meta.RoundsExecuted)
	fmt.Fprintf(b, "| Status | %s |\n", string(meta.TerminalStatus))
}

// renderFindings writes each finding as a subsection under Findings.
func renderFindings(b *strings.Builder, findings []models.ReviewFinding) {
	fmt.Fprint(b, "\n## Findings\n\n")

	if len(findings) == 0 {
		fmt.Fprint(b, "None\n")

		return
	}

	for i, f := range findings {
		fmt.Fprintf(b, "### %s (Score: %d/10) — %s\n\n", f.CriterionName, f.Score, string(f.Status))

		fmt.Fprintf(b, "**Evidence:** %s\n\n", f.QuotedExcerpt)
		fmt.Fprintf(b, "**Location:** %s § %s\n\n", f.ArtifactLocation.FilePath, f.ArtifactLocation.SectionReference)
		fmt.Fprintf(b, "**Reasoning:** %s\n", f.Reasoning)

		if i < len(findings)-1 {
			fmt.Fprint(b, "\n---\n\n")
		}
	}
}

// renderAttackRound writes the attack round section.
func renderAttackRound(b *strings.Builder, result *models.AttackRoundResult) {
	fmt.Fprint(b, "\n## Attack Round\n\n")

	if result == nil {
		fmt.Fprint(b, "No attack round executed\n")

		return
	}

	if !result.NewIssuesFound || len(result.Issues) == 0 {
		fmt.Fprint(b, "No new issues found\n")

		return
	}

	fmt.Fprintf(b, "Attack round surfaced %d new issue(s):\n\n", len(result.Issues))

	for _, issue := range result.Issues {
		fmt.Fprintf(b, "* **%s** (Score: %d/10): %s\n", issue.CriterionName, issue.Score, issue.Reasoning)
	}
}

// renderHumanEscalations writes the human escalations section.
func renderHumanEscalations(b *strings.Builder, escalations []models.HumanEscalation) {
	fmt.Fprint(b, "\n## Human Escalations\n\n")

	if len(escalations) == 0 {
		fmt.Fprint(b, "None\n")

		return
	}

	for _, e := range escalations {
		fmt.Fprintf(b, "* **%s:** %s\n", e.CriterionName, e.Question)

		if e.HumanAnswer != "" {
			fmt.Fprintf(b, "  * **Answer:** %s\n", e.HumanAnswer)
		}
	}
}

// renderHumanOverrides writes the human overrides section.
func renderHumanOverrides(b *strings.Builder, overrides []models.HumanOverride) {
	fmt.Fprint(b, "\n## Human Overrides\n\n")

	if len(overrides) == 0 {
		fmt.Fprint(b, "None\n")

		return
	}

	for _, o := range overrides {
		fmt.Fprintf(b, "* **%s** (original score: %d/10): %s\n", o.CriterionName, o.OriginalScore, o.HumanRationale)
	}
}

// renderDeadChecks writes the dead checks section.
func renderDeadChecks(b *strings.Builder, deadChecks []string) {
	fmt.Fprint(b, "\n## Dead Checks\n\n")

	if len(deadChecks) == 0 {
		fmt.Fprint(b, "None\n")

		return
	}

	for _, dc := range deadChecks {
		fmt.Fprintf(b, "* %s\n", dc)
	}
}

// renderDegradationSummary writes the degradation summary section.
func renderDegradationSummary(b *strings.Builder, entries []models.DegradationEntry) {
	fmt.Fprint(b, "\n## Degradation Summary\n\n")

	if len(entries) == 0 {
		fmt.Fprint(b, "No degradation events\n")

		return
	}

	for _, e := range entries {
		fmt.Fprintf(b, "* **%s** — %s (at %s)\n", e.Component, e.FailureMode, e.Timestamp)

		if len(e.AffectedCriteria) > 0 {
			fmt.Fprintf(b, "  * Affected criteria: %s\n", strings.Join(e.AffectedCriteria, ", "))
		}
	}
}
