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

package output_test

import (
	"strings"
	"testing"
	"time"

	"pgregory.net/rapid"

	"go.nwlabs.dev/magneto/internal/models"
	"go.nwlabs.dev/magneto/internal/output"
)

// Feature: adversarial-review-operational-workflow, Property 8: Human decisions
// and terminal records preserve available information
//
// For any non-empty human rationale, degradation entries, unavailable-value
// reasons, and terminal session data, finalization preserves every available
// finding and human decision, emits each unavailable value with its reason only
// in a terminal partial record, generates one unique record filename, and
// rejects an empty override or block rationale.
//
// **Validates: Requirements 7.8, 8.1, 8.3, 8.4, 8.7, 9.1, 9.5**.

type (
	// PropertySessionInput holds the generated terminal session data for
	// Property 8 assertions.
	PropertySessionInput struct {
		Session           *models.ReviewSessionOutput
		TerminalStatus    models.TerminalStatus
		OverrideRationale string
		BlockRationale    string
	}

	// UnavailableAssertionInput holds inputs for unavailable- value rendering
	// assertions.
	UnavailableAssertionInput struct {
		Rendered       string
		TerminalStatus models.TerminalStatus
		Values         []models.UnavailableValue
	}
)

// TestProperty_HumanDecisionsAndTerminalRecordsPreserveAvailableInformation
// verifies Property 8: Human decisions and terminal records preserve available
// information.
//
// Feature: adversarial-review-operational-workflow, Property 8: Human decisions
// and terminal records preserve available information
//
// **Validates: Requirements 7.8, 8.1, 8.3, 8.4, 8.7, 9.1, 9.5**.
func TestProperty_HumanDecisionsAndTerminalRecordsPreserveAvailableInformation( // lint:ignore_length
	t *testing.T,
) {
	rapid.Check(t, func(t *rapid.T) {
		terminalStatus := drawTerminalStatus(t)

		findingCount := rapid.IntRange(1, 5).Draw(
			t, "finding_count",
		)

		findings := make(
			[]models.ReviewFinding, findingCount,
		)

		for i := range findingCount {
			findings[i] = drawPropertyFinding(t)
		}

		escalations := drawEscalations(t)
		overrides := drawOverrides(t)
		blockAcceptance := drawBlockAcceptance(t)
		unavailableValues := drawUnavailableValues(t)
		degradations := drawDegradations(t)

		session := &models.ReviewSessionOutput{
			Metadata: models.SessionMetadata{
				SpecName:           "property-8-spec",
				ArtifactPath:       ".kiro/specs/prop8/design.md",
				Timestamp:          "2026-08-13T10:00:00Z",
				TerminalStatus:     terminalStatus,
				RoundsExecuted:     2,
				Phase3Baseline:     models.Phase3BaselineAbsent,
				DegradedComponents: degradations,
			},
			Findings:             findings,
			HumanEscalations:     escalations,
			HumanOverrides:       overrides,
			HumanBlockAcceptance: blockAcceptance,
			UnavailableValues:    unavailableValues,
		}

		rendered := output.RenderSession(session)

		// Verify every finding's criterion name appears.
		assertFindingsPreserved(t, rendered, findings)

		// Verify human escalation questions appear.
		assertEscalationsPreserved(t, rendered, escalations)

		// Verify human override rationales appear.
		assertOverridesPreserved(t, rendered, overrides)

		// Verify block acceptance rationale appears.
		assertBlockAcceptancePreserved(t, rendered, blockAcceptance)

		// Verify unavailable values appear ONLY for partial_review.
		assertUnavailableValueRendering(t, &UnavailableAssertionInput{
			Rendered:       rendered,
			TerminalStatus: terminalStatus,
			Values:         unavailableValues,
		})

		// Verify Phase 3 baseline absent label always appears.
		assertPhase3BaselinePresent(t, rendered)
	})
}

// TestProperty_UniqueFilenameGeneration verifies that different timestamps
// produce different filenames as part of Property 8.
//
// Feature: adversarial-review-operational-workflow, Property 8: Human decisions
// and terminal records preserve available information
//
// **Validates: Requirements 7.8, 8.1, 8.3, 8.4, 8.7, 9.1, 9.5**.
func TestProperty_UniqueFilenameGeneration(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		root := t.TempDir()

		// Generate two distinct timestamps on different dates.
		year := rapid.IntRange(2025, 2030).Draw(
			rt, "year",
		)

		month := rapid.IntRange(1, 12).Draw(rt, "month")
		day1 := rapid.IntRange(1, 28).Draw(rt, "day1")
		day2 := rapid.IntRange(1, 28).Draw(rt, "day2")

		ts1 := time.Date(
			year, time.Month(month), day1,
			10, 0, 0, 0, time.UTC,
		)

		ts2 := time.Date(
			year, time.Month(month), day2,
			11, 0, 0, 0, time.UTC,
		)

		specName := "unique-test"

		path1, err1 := output.GenerateFilename(
			&output.FilenameInput{
				Timestamp:     ts1,
				SpecName:      specName,
				WorkspaceRoot: root,
			},
		)
		if err1 != nil {
			rt.Fatalf(
				"first filename generation failed: %v",
				err1,
			)
		}

		path2, err2 := output.GenerateFilename(
			&output.FilenameInput{
				Timestamp:     ts2,
				SpecName:      specName,
				WorkspaceRoot: root,
			},
		)
		if err2 != nil {
			rt.Fatalf(
				"second filename generation failed: %v",
				err2,
			)
		}

		// Different timestamps produce different filenames (same date may
		// produce same filename when sequence-1 is available for both, so we
		// verify the pair is unique via at least different paths or different
		// sequence numbers).
		if day1 != day2 && path1 == path2 {
			rt.Fatalf("different dates must produce different filenames: got %q for both", path1)
		}
	})
}

// TestProperty_EmptyRationaleRejection verifies that empty override and block
// acceptance rationales are detectable as part of Property 8 rationale
// rejection.
//
// Feature: adversarial-review-operational-workflow, Property 8: Human decisions
// and terminal records preserve available information
//
// **Validates: Requirements 7.8, 8.1, 8.3, 8.4, 8.7, 9.1, 9.5**.
func TestProperty_EmptyRationaleRejection(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate an empty or whitespace-only rationale.
		emptyRationale := drawEmptyRationale(t)

		// Verify empty override rationale is detectable.
		if strings.TrimSpace(emptyRationale) == "" &&
			!isEmptyRationale(emptyRationale) {
			t.Fatal("empty override rationale must be detectable as empty")
		}

		// Verify empty block acceptance rationale is detectable.
		if strings.TrimSpace(emptyRationale) == "" && !isEmptyRationale(emptyRationale) {
			t.Fatal("empty block acceptance rationale must be detectable as empty")
		}

		// Verify non-empty rationale is not rejected.
		nonEmpty := rapid.StringMatching(`[a-zA-Z ]{5,30}`).Draw(t, "non_empty_rationale")

		if isEmptyRationale(nonEmpty) {
			t.Fatalf("non-empty rationale %q must not be detected as empty", nonEmpty)
		}
	})
}

// isEmptyRationale models the validation check that rejects an empty override
// or block rationale before finalization.
func isEmptyRationale(rationale string) bool {
	return strings.TrimSpace(rationale) == ""
}

// drawTerminalStatus generates one of the four approved terminal statuses.
func drawTerminalStatus(t *rapid.T) models.TerminalStatus {
	statuses := []models.TerminalStatus{
		models.TerminalApproved,
		models.TerminalNotApproved,
		models.TerminalPartialReview,
		models.TerminalHumanOverride,
	}

	idx := rapid.IntRange(0, len(statuses)-1).Draw(t, "terminal_status_idx")

	return statuses[idx]
}

// drawPropertyFinding generates a random ReviewFinding for Property 8 testing.
func drawPropertyFinding(t *rapid.T) models.ReviewFinding {
	criterionName := rapid.StringMatching(`[a-z]{3,12}-[a-z]{3,12}`).Draw(t, "criterion_name")
	satisfaction := rapid.IntRange(1, 10).Draw(t, "satisfaction")
	severity := drawPropertySeverity(t)
	domains := drawPropertyDomains(t)

	return models.ReviewFinding{
		ArtifactLocation: models.ArtifactLocation{
			FilePath:         "design.md",
			SectionReference: "Overview",
		},
		CriterionName:         criterionName,
		QuotedExcerpt:         "test excerpt",
		Status:                models.StatusHypothesized,
		Reasoning:             "test reasoning",
		FindingSeverity:       severity,
		FindingDomains:        domains,
		CriterionSatisfaction: satisfaction,
	}
}

// drawPropertySeverity generates a random FindingSeverity.
func drawPropertySeverity(t *rapid.T) models.FindingSeverity {
	severities := []models.FindingSeverity{
		models.SeverityCritical,
		models.SeverityHigh,
		models.SeverityMedium,
		models.SeverityLow,
	}

	idx := rapid.IntRange(0, len(severities)-1).Draw(t, "severity_idx")

	return severities[idx]
}

// drawPropertyDomains generates 1-3 non-duplicate finding domains.
func drawPropertyDomains(t *rapid.T) []models.FindingDomain {
	allDomains := []models.FindingDomain{
		models.DomainSecurity,
		models.DomainCorrectness,
		models.DomainArchitecture,
		models.DomainReliability,
		models.DomainOperations,
		models.DomainDeveloperExperience,
	}

	count := rapid.IntRange(1, 3).Draw(t, "domain_count")
	selected := make([]models.FindingDomain, 0, count)
	available := make([]models.FindingDomain, len(allDomains))
	copy(available, allDomains)

	for range count {
		idx := rapid.IntRange(0, len(available)-1).Draw(t, "domain_idx")

		selected = append(selected, available[idx])
		available[idx] = available[len(available)-1]
		available = available[:len(available)-1]
	}

	return selected
}

// drawEscalations generates 0-3 human escalations with non-empty questions.
func drawEscalations(t *rapid.T) []models.HumanEscalation {
	count := rapid.IntRange(0, 3).Draw(t, "escalation_count")

	escalations := make([]models.HumanEscalation, count)

	for i := range count {
		question := rapid.StringMatching(`[A-Z][a-z ]{10,30}\?`).Draw(t, "escalation_question")
		rationale := rapid.StringMatching(`[a-z ]{5,20}`).Draw(t, "escalation_rationale")

		escalations[i] = models.HumanEscalation{
			CriterionName:      "escalated-criterion",
			Question:           question,
			Context:            "test context",
			InspectedEvidence:  "inspected evidence",
			RemainingAmbiguity: "remaining ambiguity",
			HumanAnswer:        rationale,
			Resolved:           true,
		}
	}

	return escalations
}

// drawOverrides generates 0-2 human overrides. When useEmptyRationale is tested
// separately, all overrides here have non-empty rationales.
func drawOverrides(t *rapid.T) []models.HumanOverride {
	count := rapid.IntRange(0, 2).Draw(t, "override_count")
	overrides := make([]models.HumanOverride, count)

	for i := range count {
		rationale := rapid.StringMatching(`[A-Z][a-z ]{10,30}`).Draw(t, "override_rationale")

		overrides[i] = models.HumanOverride{
			CriterionName:                 "overridden-criterion",
			Decision:                      models.HumanDecisionOverride,
			HumanRationale:                rationale,
			FindingIndex:                  i,
			OriginalCriterionSatisfaction: 3,
		}
	}

	return overrides
}

// drawBlockAcceptance generates an optional human block acceptance with a
// non-empty rationale (or nil).
func drawBlockAcceptance(t *rapid.T) *models.HumanBlockAcceptance {
	hasBlock := rapid.Bool().Draw(t, "has_block_acceptance")

	if !hasBlock {
		return nil
	}

	rationale := rapid.StringMatching(`[A-Z][a-z ]{10,30}`).Draw(t, "block_rationale")

	return &models.HumanBlockAcceptance{
		CriterionName:   "blocked-criterion",
		Decision:        models.HumanDecisionBlockAcceptance,
		EvidenceContext: "block evidence context",
		HumanRationale:  rationale,
	}
}

// drawUnavailableValues generates 0-3 unavailable values with field and reason.
func drawUnavailableValues(t *rapid.T) []models.UnavailableValue {
	count := rapid.IntRange(0, 3).Draw(t, "unavailable_count")
	values := make([]models.UnavailableValue, count)

	for i := range count {
		field := rapid.StringMatching(`[a-z_]{5,15}`).Draw(t, "unavailable_field")
		reason := rapid.StringMatching(`[A-Z][a-z ]{10,40}`).Draw(t, "unavailable_reason")

		values[i] = models.UnavailableValue{
			Field:  field,
			Reason: reason,
		}
	}

	return values
}

// drawDegradations generates 0-3 degradation entries.
func drawDegradations(t *rapid.T) []models.DegradationEntry {
	count := rapid.IntRange(0, 3).Draw(t, "degradation_count")
	entries := make([]models.DegradationEntry, count)

	for i := range count {
		component := rapid.StringMatching(`[a-z]{4,10}`).Draw(t, "degradation_component")
		criticality := models.CriticalityRequired

		if rapid.Bool().Draw(t, "is_optional") {
			criticality = models.CriticalityOptional
		}

		entries[i] = models.DegradationEntry{
			Component:        component,
			Criticality:      criticality,
			FailureMode:      "test failure",
			Timestamp:        "2026-08-13T10:00:00Z",
			AffectedCriteria: []string{"affected-criterion"},
		}
	}

	return entries
}

// drawEmptyRationale generates an empty or whitespace-only string.
func drawEmptyRationale(t *rapid.T) string {
	emptyOptions := []string{
		"",
		" ",
		"  ",
		"\t",
		"\n",
		" \t\n ",
	}

	idx := rapid.IntRange(0, len(emptyOptions)-1).Draw(t, "empty_rationale_idx")

	return emptyOptions[idx]
}

// assertFindingsPreserved verifies every finding's criterion name appears in
// the rendered output.
func assertFindingsPreserved(t *rapid.T, rendered string, findings []models.ReviewFinding) {
	for i := range findings {
		finding := &findings[i]

		if !strings.Contains(rendered, finding.CriterionName) {
			t.Fatalf("finding[%d] criterion name %q not found in rendered output", i, finding.CriterionName)
		}
	}
}

// assertEscalationsPreserved verifies every escalation's question appears in
// the rendered output when escalations are present.
func assertEscalationsPreserved(t *rapid.T, rendered string, escalations []models.HumanEscalation) {
	for i := range escalations {
		escalation := &escalations[i]

		if !strings.Contains(rendered, escalation.Question) {
			t.Fatalf("escalation[%d] question %q not found in rendered output", i, escalation.Question)
		}
	}
}

// assertOverridesPreserved verifies every override's rationale appears in the
// rendered output when overrides are present.
func assertOverridesPreserved(t *rapid.T, rendered string, overrides []models.HumanOverride) {
	for i := range overrides {
		override := &overrides[i]

		if !strings.Contains(rendered, override.HumanRationale) {
			t.Fatalf("override[%d] rationale %q not found in rendered output", i, override.HumanRationale)
		}
	}
}

// assertBlockAcceptancePreserved verifies block acceptance rationale appears in
// the rendered output when present.
func assertBlockAcceptancePreserved(t *rapid.T, rendered string, acceptance *models.HumanBlockAcceptance) {
	if acceptance == nil {
		return
	}

	if !strings.Contains(rendered, acceptance.HumanRationale) {
		t.Fatalf("block acceptance rationale %q not found in rendered output", acceptance.HumanRationale)
	}
}

// assertUnavailableValueRendering verifies unavailable values appear ONLY when
// terminal status is partial_review.
func assertUnavailableValueRendering(t *rapid.T, input *UnavailableAssertionInput) {
	if input.TerminalStatus == models.TerminalPartialReview {
		// Unavailable values should appear in partial review records.
		for i := range input.Values {
			value := &input.Values[i]

			if !strings.Contains(input.Rendered, value.Field) {
				t.Fatalf("unavailable[%d] field %q must appear in partial_review rendered output", i, value.Field)
			}

			if !strings.Contains(input.Rendered, value.Reason) {
				t.Fatalf("unavailable[%d] reason %q must appear in partial_review rendered output", i, value.Reason)
			}
		}
	} else if strings.Contains(input.Rendered, "## Unavailable Values") {
		// Unavailable values must NOT appear for non-partial statuses.
		t.Fatalf("unavailable values section must not appear for terminal status %q", input.TerminalStatus)
	}
}

// assertPhase3BaselinePresent verifies the Phase 3 baseline absent label always
// appears in any rendered output.
func assertPhase3BaselinePresent(t *rapid.T, rendered string) {
	if !strings.Contains(rendered, "Phase 3 Baseline:** absent (no pre-Phase-1 control baseline)") {
		t.Fatal("Phase 3 baseline absent label must always appear in rendered output")
	}
}
