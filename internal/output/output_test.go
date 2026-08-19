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
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-openapi/testify/assert"
	"github.com/go-openapi/testify/require"

	"go.nwlabs.dev/magneto/internal/models"
	"go.nwlabs.dev/magneto/internal/output"
)

// buildFullSession constructs a ReviewSessionOutput with all sections populated
// for testing purposes.
func buildFullSession() *models.ReviewSessionOutput {
	return &models.ReviewSessionOutput{
		Metadata: models.SessionMetadata{
			SpecName:                    "auth-redesign",
			ArtifactPath:                ".kiro/specs/auth-redesign/design.md",
			Timestamp:                   "2026-08-13",
			TerminalStatus:              models.TerminalNotApproved,
			TaskExecutionID:             "task-42",
			SessionID:                   "session-7",
			SelectionDecision:           models.SelectionSelected,
			SelectionReason:             "Foundational artifact changed.",
			FoundationalArtifact:        true,
			TriggeredBlastRadiusDomains: []string{"auth", "secrets"},
			LoadedRubricCriteria:        []string{"error-handling", "data-integrity"},
			Phase3Baseline:              models.Phase3BaselineAbsent,
			RoundsExecuted:              3,
			DegradedComponents: []models.DegradationEntry{
				{
					Component:           "confirmer",
					Criticality:         models.CriticalityRequired,
					FailureMode:         "model unavailable",
					Timestamp:           "2026-08-13T10:05:00Z",
					UnavailableValueKey: "confirmer_attempts",
					AffectedCriteria:    []string{"security-boundaries"},
				},
			},
		},
		Findings: []models.ReviewFinding{
			{
				CriterionName:         "error-handling",
				CriterionSatisfaction: 4,
				FindingSeverity:       models.SeverityHigh,
				FindingDomains:        []models.FindingDomain{models.DomainCorrectness, models.DomainSecurity},
				QuotedExcerpt:         "errors are logged but not returned",
				ArtifactLocation: models.ArtifactLocation{
					FilePath:         "design.md",
					SectionReference: "Error Handling",
				},
				Status:            models.StatusConfirmed,
				Reasoning:         "Missing error propagation to caller",
				ConfirmerEvidence: "A failing caller test demonstrates the missing propagation.",
				ConfirmerAttempts: []models.ConfirmerAttempt{
					{
						AttemptNumber:         1,
						Strategy:              "Exercise the error path.",
						Observation:           "The caller receives a nil error.",
						Demonstrated:          true,
						DemonstrationEvidence: "The failing test assertion is reproducible.",
					},
				},
				CitationGateResult: &models.CitationGateResult{
					SchemaValid:             true,
					CitationValid:           true,
					ProvenanceCorrelationID: "citation-1",
					MatchedLines:            &models.CitationMatchedLines{Start: 12, End: 14},
				},
			},
		},
		HumanEscalations: []models.HumanEscalation{
			{
				CriterionName:      "data-integrity",
				Question:           "Is eventual consistency acceptable?",
				Context:            "Design does not specify consistency model",
				InspectedEvidence:  "The data model section omits a consistency guarantee.",
				RemainingAmbiguity: "The product tolerance is not documented.",
				HumanAnswer:        "Yes, eventual is fine",
				Resolved:           true,
			},
		},
		HumanOverrides: []models.HumanOverride{
			{
				CriterionName:                 "performance",
				Decision:                      models.HumanDecisionOverride,
				FindingIndex:                  0,
				OriginalCriterionSatisfaction: 3,
				HumanRationale:                "Acceptable for MVP scope",
			},
		},
		HumanBlockAcceptance: &models.HumanBlockAcceptance{
			CriterionName:   "security-boundaries",
			Decision:        models.HumanDecisionBlockAcceptance,
			EvidenceContext: "The remaining claim is unresolved.",
			HumanRationale:  "The human accepts the advisory block.",
		},
		DeadChecks: []string{
			"payment-processing (no payment logic present)",
		},
		AttackRoundResult: &models.AttackRoundResult{
			NewIssuesFound: true,
			Issues: []models.ReviewFinding{
				{
					CriterionName:         "auth-bypass",
					CriterionSatisfaction: 2,
					FindingSeverity:       models.SeverityCritical,
					FindingDomains:        []models.FindingDomain{models.DomainSecurity},
					Reasoning:             "Token validation skipped on retry path",
				},
			},
		},
	}
}

func TestRenderSessionFullOutput(t *testing.T) {
	session := buildFullSession()
	result := output.RenderSession(session)

	assert.Contains(t, result, "# Adversarial Review: auth-redesign")
	assert.Contains(t, result, "## Summary")
	assert.Contains(t, result, "## Selection")
	assert.Contains(t, result, "## Loaded Rubric Criteria")
	assert.Contains(t, result, "## Findings")
	assert.Contains(t, result, "## Attack Round")
	assert.Contains(t, result, "## Human Escalations")
	assert.Contains(t, result, "## Human Overrides")
	assert.Contains(t, result, "## Human Block Acceptance")
	assert.Contains(t, result, "## Dead Checks")
	assert.Contains(t, result, "## Degradation Summary")
	assert.Contains(t, result, "Phase 3 Baseline:** absent (no pre-Phase-1 control baseline)")
	assert.Contains(t, result, "Foundational Artifact:** yes")
	assert.Contains(t, result, "Triggered Blast-Radius Domains:** auth, secrets")

	assert.Contains(t, result, "error-handling")
	assert.Contains(t, result, "Criterion Satisfaction:** 4/10")
	assert.Contains(t, result, "Finding Severity:** high")
	assert.Contains(t, result, "Finding Domains:** security, correctness")
	assert.Contains(t, result, "Schema valid: true")
	assert.Contains(t, result, "Matched lines: 12-14")
	assert.Contains(t, result, "Confirmer Evidence")
	assert.Contains(t, result, "Attempt 1")
	assert.Contains(t, result, "errors are logged but not returned")

	assert.Contains(t, result, "data-integrity")
	assert.Contains(t, result, "Is eventual consistency acceptable?")
	assert.Contains(t, result, "Yes, eventual is fine")

	assert.Contains(t, result, "performance")
	assert.Contains(t, result, "Decision:** override")
	assert.Contains(t, result, "Acceptable for MVP scope")
	assert.Contains(t, result, "security-boundaries")
	assert.Contains(t, result, "block_acceptance")
	assert.Contains(t, result, "The human accepts the advisory block.")

	assert.Contains(t, result, "payment-processing (no payment logic present)")

	assert.Contains(t, result, "auth-bypass")
	assert.Contains(t, result, "Token validation skipped on retry path")

	assert.Contains(t, result, "confirmer")
	assert.Contains(t, result, "model unavailable")
}

func TestRenderSessionEmptySession(t *testing.T) {
	session := &models.ReviewSessionOutput{
		Metadata: models.SessionMetadata{
			SpecName:       "simple-feature",
			ArtifactPath:   ".kiro/specs/simple-feature/design.md",
			Timestamp:      "2026-08-13",
			TerminalStatus: models.TerminalApproved,
			RoundsExecuted: 1,
		},
		Findings: nil,
	}

	result := output.RenderSession(session)

	assert.Contains(t, result, "## Findings")
	assert.Contains(t, result, "None")
	assert.NotContains(t, result, "## Human Escalations")
	assert.NotContains(t, result, "## Human Overrides")
	assert.NotContains(t, result, "## Human Block Acceptance")
	assert.NotContains(t, result, "## Dead Checks")
	assert.NotContains(t, result, "## Unavailable Values")
	assert.Contains(t, result, "No degradation events")
}

func TestRenderSessionAttackRound(t *testing.T) {
	tests := []struct {
		fn   func(t *testing.T)
		name string
	}{
		{
			name: "new issues listed",
			fn: func(t *testing.T) {
				t.Helper()

				session := &models.ReviewSessionOutput{
					Metadata: models.SessionMetadata{
						SpecName:       "attack-test",
						ArtifactPath:   "design.md",
						Timestamp:      "2026-08-13",
						TerminalStatus: models.TerminalNotApproved,
						RoundsExecuted: 2,
					},
					AttackRoundResult: &models.AttackRoundResult{
						NewIssuesFound: true,
						Issues: []models.ReviewFinding{
							{
								CriterionName:         "race-condition",
								CriterionSatisfaction: 3,
								Reasoning:             "Shared mutable state without locking",
							},
							{
								CriterionName:         "input-validation",
								CriterionSatisfaction: 5,
								Reasoning:             "Missing boundary check on array index",
							},
						},
					},
				}

				result := output.RenderSession(session)

				assert.Contains(t, result, "## Attack Round")
				assert.Contains(t, result, "2 new issue(s)")
				assert.Contains(t, result, "race-condition")
				assert.Contains(t, result, "Shared mutable state without locking")
				assert.Contains(t, result, "input-validation")
			},
		},
		{
			name: "no issues found",
			fn: func(t *testing.T) {
				t.Helper()

				session := &models.ReviewSessionOutput{
					Metadata: models.SessionMetadata{
						SpecName:       "clean-spec",
						ArtifactPath:   "design.md",
						Timestamp:      "2026-08-13",
						TerminalStatus: models.TerminalApproved,
						RoundsExecuted: 2,
					},
					AttackRoundResult: &models.AttackRoundResult{
						NewIssuesFound: false,
						Issues:         nil,
					},
				}

				result := output.RenderSession(session)

				assert.Contains(t, result, "## Attack Round")
				assert.Contains(t, result, "no new issues found")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fn(t)
		})
	}
}

func TestRenderSessionHeadingStructure(t *testing.T) {
	session := &models.ReviewSessionOutput{
		Metadata: models.SessionMetadata{
			SpecName:       "heading-test",
			ArtifactPath:   "design.md",
			Timestamp:      "2026-08-13",
			TerminalStatus: models.TerminalNotApproved,
			RoundsExecuted: 1,
		},
		Findings: []models.ReviewFinding{
			{
				CriterionName:         "test-criterion",
				CriterionSatisfaction: 5,
				QuotedExcerpt:         "some excerpt",
				ArtifactLocation: models.ArtifactLocation{
					FilePath:         "design.md",
					SectionReference: "Overview",
				},
				Status:    models.StatusHypothesized,
				Reasoning: "Partial coverage",
			},
		},
	}

	result := output.RenderSession(session)

	lines := strings.Split(result, "\n")
	h1Count := 0
	h2Count := 0
	h3Count := 0

	for _, line := range lines {
		if strings.HasPrefix(line, "# ") &&
			!strings.HasPrefix(line, "## ") {
			h1Count++
		}

		if strings.HasPrefix(line, "## ") &&
			!strings.HasPrefix(line, "### ") {
			h2Count++
		}

		if strings.HasPrefix(line, "### ") {
			h3Count++
		}
	}

	assert.Equal(t, 1, h1Count, "should have exactly one H1")
	assert.Equal(t, 3, h2Count, "should have summary, findings, and degradation sections")
	assert.Equal(t, 1, h3Count, "should have 1 H3 for finding")
}

func TestGenerateFilename(t *testing.T) {
	tests := []struct {
		fn   func(t *testing.T)
		name string
	}{
		{
			name: "first review on a date produces sequence 1",
			fn: func(t *testing.T) {
				t.Helper()

				root := t.TempDir()
				ts := time.Date(2026, 8, 13, 10, 0, 0, 0, time.UTC)

				result, err := output.GenerateFilename(&output.FilenameInput{
					Timestamp:     ts,
					SpecName:      "auth-redesign",
					WorkspaceRoot: root,
				})

				require.NoError(t, err)

				expected := filepath.Join(root, ".kiro", "reviews", "auth-redesign-2026-08-13-1.md")
				assert.Equal(t, expected, result)
			},
		},
		{
			name: "multiple reviews on same date increment sequence",
			fn: func(t *testing.T) {
				t.Helper()

				root := t.TempDir()
				ts := time.Date(2026, 8, 13, 10, 0, 0, 0, time.UTC)

				reviewDir := filepath.Join(root, ".kiro", "reviews")
				mkdirErr := os.MkdirAll(reviewDir, 0o0755) // lint:allow_755
				require.NoError(t, mkdirErr)

				firstFile := filepath.Join(reviewDir, "auth-redesign-2026-08-13-1.md")
				writeErr := os.WriteFile(firstFile, []byte("existing"), 0o0666) // lint:allow_666
				require.NoError(t, writeErr)

				result, err := output.GenerateFilename(&output.FilenameInput{
					Timestamp:     ts,
					SpecName:      "auth-redesign",
					WorkspaceRoot: root,
				})

				require.NoError(t, err)

				expected := filepath.Join(reviewDir, "auth-redesign-2026-08-13-2.md")
				assert.Equal(t, expected, result)
			},
		},
		{
			name: "creates reviews directory if missing",
			fn: func(t *testing.T) {
				t.Helper()

				root := t.TempDir()
				ts := time.Date(2026, 8, 13, 10, 0, 0, 0, time.UTC)

				_, err := output.GenerateFilename(&output.FilenameInput{
					Timestamp:     ts,
					SpecName:      "new-spec",
					WorkspaceRoot: root,
				})

				require.NoError(t, err)

				reviewDir := filepath.Join(root, ".kiro", "reviews")
				info, statErr := os.Stat(reviewDir)
				require.NoError(t, statErr)
				assert.True(t, info.IsDir())
			},
		},
		{
			name: "third review on same date gets sequence 3",
			fn: func(t *testing.T) {
				t.Helper()

				root := t.TempDir()
				ts := time.Date(2026, 8, 13, 10, 0, 0, 0, time.UTC)

				reviewDir := filepath.Join(root, ".kiro", "reviews")
				mkdirErr := os.MkdirAll(reviewDir, 0o0755) // lint:allow_755
				require.NoError(t, mkdirErr)

				for _, seq := range []string{"1", "2"} {
					name := "my-spec-2026-08-13-" + seq + ".md"
					writeErr := os.WriteFile(
						filepath.Join(reviewDir, name),
						[]byte("existing"),
						0o0666, // lint:allow_666
					)
					require.NoError(t, writeErr)
				}

				result, err := output.GenerateFilename(&output.FilenameInput{
					Timestamp:     ts,
					SpecName:      "my-spec",
					WorkspaceRoot: root,
				})

				require.NoError(t, err)

				expected := filepath.Join(reviewDir, "my-spec-2026-08-13-3.md")
				assert.Equal(t, expected, result)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fn(t)
		})
	}
}

func TestPersistSessionIsIdempotent(t *testing.T) {
	root := t.TempDir()
	session := buildFullSession()

	session.Metadata.Timestamp = "2026-08-13T10:00:00Z"
	session.TerminalIdempotencyKey = "task-1-session-1"

	first, firstErr := output.PersistSession(&output.PersistSessionInput{
		Session:       session,
		WorkspaceRoot: root,
	})
	require.NoError(t, firstErr)
	assert.True(t, first.Created)

	second, secondErr := output.PersistSession(&output.PersistSessionInput{
		Session:       session,
		WorkspaceRoot: root,
	})
	require.NoError(t, secondErr)
	assert.False(t, second.Created)
	assert.Equal(t, first.RecordPath, second.RecordPath)

	recordEntries, readDirErr := os.ReadDir(filepath.Join(root, ".kiro", "reviews"))
	require.NoError(t, readDirErr)
	assert.Len(t, recordEntries, 1)

	expectedPath := filepath.Join(root, ".kiro", "reviews", "auth-redesign-2026-08-13-1.md")
	assert.Equal(t, expectedPath, first.RecordPath)

	record, readErr := os.ReadFile(first.RecordPath)
	require.NoError(t, readErr)
	assert.Contains(t, string(record), "**Terminal Idempotency Key:** task-1-session-1")
}

func TestRenderSessionPartialReviewIncludesUnavailableValues(t *testing.T) {
	session := &models.ReviewSessionOutput{
		Metadata: models.SessionMetadata{
			SpecName:       "partial-review",
			ArtifactPath:   ".kiro/specs/partial-review/design.md",
			Timestamp:      "2026-08-13T10:00:00Z",
			TerminalStatus: models.TerminalPartialReview,
			DegradedComponents: []models.DegradationEntry{
				{
					Component:           "citation-gate",
					Criticality:         models.CriticalityRequired,
					FailureMode:         "service unavailable",
					Timestamp:           "2026-08-13T10:00:00Z",
					UnavailableValueKey: "citation_gate_result",
					AffectedCriteria:    []string{"path-containment"},
				},
			},
		},
		UnavailableValues: []models.UnavailableValue{
			{
				Field:  "citation_gate_result",
				Reason: "Citation gate was unavailable for path-containment.",
			},
		},
	}

	result := output.RenderSession(session)

	assert.Contains(t, result, "## Unavailable Values")
	assert.Contains(t, result, "citation_gate_result")
	assert.Contains(t, result, "Citation gate was unavailable for path-containment.")
	assert.Contains(t, result, "Criticality:** required")
	assert.Contains(t, result, "Unavailable Value Key:** citation_gate_result")
}

func TestRenderSessionPartialReviewPreservesFindingsAndDegradations(t *testing.T) {
	session := buildFullSession()

	session.Metadata.TerminalStatus = models.TerminalPartialReview
	session.Metadata.DegradedComponents = append(session.Metadata.DegradedComponents, models.DegradationEntry{
		Component:           "rubric-loader",
		Criticality:         models.CriticalityOptional,
		FailureMode:         "criterion metadata unavailable",
		Timestamp:           "2026-08-13T10:06:00Z",
		UnavailableValueKey: "rubric_metadata",
		AffectedCriteria:    []string{"operations"},
	})
	session.Findings = append(session.Findings, models.ReviewFinding{
		CriterionName:         "deployment-safety",
		CriterionSatisfaction: 6,
		FindingSeverity:       models.SeverityMedium,
		FindingDomains:        []models.FindingDomain{models.DomainReliability, models.DomainSecurity},
		QuotedExcerpt:         "deployments require manual rollback",
		ArtifactLocation: models.ArtifactLocation{
			FilePath:         "design.md",
			SectionReference: "Deployment",
		},
		Status:    models.StatusUnconfirmed,
		Reasoning: "The rollback procedure lacks an automated verification step.",
		CitationGateResult: &models.CitationGateResult{
			SchemaValid:             true,
			CitationValid:           false,
			FailureReason:           "quoted excerpt does not match the cited section",
			ProvenanceCorrelationID: "citation-2",
		},
	})
	session.UnavailableValues = []models.UnavailableValue{
		{
			Field:  "rubric_metadata",
			Reason: "The optional rubric metadata source was unavailable.",
		},
	}

	first := output.RenderSession(session)
	second := output.RenderSession(session)

	assert.Equal(t, first, second)
	assert.Contains(t, first, "### deployment-safety")
	assert.Contains(t, first, "**Criterion Satisfaction:** 6/10")
	assert.Contains(t, first, "**Finding Severity:** medium")
	assert.Contains(t, first, "**Finding Domains:** security, reliability")
	assert.Contains(t, first, "**Verification Status:** unconfirmed")
	assert.Contains(t, first, "**Quoted Evidence:** deployments require manual rollback")
	assert.Contains(t, first, "**Artifact Location:** design.md § Deployment")
	assert.Contains(t, first, "The rollback procedure lacks an automated verification step.")
	assert.Contains(t, first, "* Citation valid: false")
	assert.Contains(t, first, "* Failure reason: quoted excerpt does not match the cited section")
	assert.Contains(t, first, "* Provenance correlation ID: citation-2")
	assert.Contains(t, first, "## Unavailable Values")
	assert.Contains(t, first, "rubric_metadata")
	assert.Contains(t, first, "The optional rubric metadata source was unavailable.")
	assert.Contains(t, first, "### confirmer")
	assert.Contains(t, first, "### rubric-loader")
	assert.Contains(t, first, "**Criticality:** optional")
	assert.Contains(t, first, "**Failure Mode:** criterion metadata unavailable")
	assert.Less(t, strings.Index(first, "### confirmer"), strings.Index(first, "### rubric-loader"))
}

func TestRenderSessionApprovedRecordOmitsUnavailableAndHumanEvents(t *testing.T) {
	session := &models.ReviewSessionOutput{
		Metadata: models.SessionMetadata{
			SpecName:       "approved-record",
			ArtifactPath:   ".kiro/specs/approved-record/design.md",
			Timestamp:      "2026-08-13T10:00:00Z",
			TerminalStatus: models.TerminalApproved,
			RoundsExecuted: 1,
		},
		UnavailableValues: []models.UnavailableValue{
			{
				Field:  "ignored_value",
				Reason: "Unavailable values belong only to terminal partial reviews.",
			},
		},
	}

	result := output.RenderSession(session)

	assert.Contains(t, result, "**Phase 3 Baseline:** absent (no pre-Phase-1 control baseline)")
	assert.NotContains(t, result, "## Unavailable Values")
	assert.NotContains(t, result, "## Human Escalations")
	assert.NotContains(t, result, "## Human Overrides")
	assert.NotContains(t, result, "## Human Block Acceptance")
}

func TestPersistSessionRejectsInterimSession(t *testing.T) {
	result, persistErr := output.PersistSession(&output.PersistSessionInput{
		Session: &models.ReviewSessionOutput{
			Metadata: models.SessionMetadata{
				TerminalStatus: "reviewing",
			},
			TerminalIdempotencyKey: "interim-session",
		},
		WorkspaceRoot: t.TempDir(),
	})

	assert.Nil(t, result)
	assert.ErrorIs(t, persistErr, output.ErrOutputTerminalStatus)
}

// --- Compatibility boundary tests (Task 5.2) ---.

// TestCompatibility_RecordNamingConvention verifies that the output file
// naming follows the {spec-name}-{ISO-8601-date}-{sequence-number}.md pattern
// used by historical records.
func TestCompatibility_RecordNamingConvention(t *testing.T) {
	root := t.TempDir()
	ts := time.Date(2026, 1, 15, 14, 30, 0, 0, time.UTC)

	result, err := output.GenerateFilename(&output.FilenameInput{
		Timestamp:     ts,
		SpecName:      "legacy-compat",
		WorkspaceRoot: root,
	})

	require.NoError(t, err)

	// The generated filename must use ISO-8601 date format with a sequence
	// number suffix.
	expected := filepath.Join(root, ".kiro", "reviews", "legacy-compat-2026-01-15-1.md")
	assert.Equal(t, expected, result)
}

// TestCompatibility_Phase3BaselineLabeling verifies that every terminal record
// always labels the Phase 3 baseline as absent regardless of status.
func TestCompatibility_Phase3BaselineLabeling(t *testing.T) {
	statuses := []models.TerminalStatus{
		models.TerminalApproved,
		models.TerminalNotApproved,
		models.TerminalPartialReview,
		models.TerminalHumanOverride,
	}

	for _, status := range statuses {
		t.Run(string(status), func(t *testing.T) {
			session := &models.ReviewSessionOutput{
				Metadata: models.SessionMetadata{
					SpecName:       "phase3-test",
					ArtifactPath:   "design.md",
					Timestamp:      "2026-01-15",
					TerminalStatus: status,
					RoundsExecuted: 1,
				},
			}

			result := output.RenderSession(session)

			assert.Contains(
				t,
				result,
				"Phase 3 Baseline:** absent (no pre-Phase-1 control baseline)",
			)
		})
	}
}

// TestCompatibility_LegacyScoreMigrationMarkerPresent verifies that the
// rendered record includes a migration marker when LegacyScoreMigrated is set.
func TestCompatibility_LegacyScoreMigrationMarkerPresent(t *testing.T) {
	session := &models.ReviewSessionOutput{
		Metadata: models.SessionMetadata{
			SpecName:            "migrated-record",
			ArtifactPath:        "design.md",
			Timestamp:           "2026-01-15",
			TerminalStatus:      models.TerminalNotApproved,
			LegacyScoreMigrated: true,
			RoundsExecuted:      1,
		},
	}

	result := output.RenderSession(session)

	assert.Contains(t, result, "Legacy Score Migration")
	assert.Contains(t, result, "mapped to canonical criterion_satisfaction")
	assert.Contains(t, result, "Legacy score migration")
}

// TestCompatibility_NoMigrationMarkerWhenNotMigrated verifies that the rendered
// record does NOT include a migration marker when no legacy score was received.
func TestCompatibility_NoMigrationMarkerWhenNotMigrated(t *testing.T) {
	session := &models.ReviewSessionOutput{
		Metadata: models.SessionMetadata{
			SpecName:            "canonical-record",
			ArtifactPath:        "design.md",
			Timestamp:           "2026-01-15",
			TerminalStatus:      models.TerminalApproved,
			LegacyScoreMigrated: false,
			RoundsExecuted:      2,
		},
	}

	result := output.RenderSession(session)

	assert.NotContains(t, result, "Legacy Score Migration")
	assert.NotContains(t, result, "Legacy score migration")
}

// TestCompatibility_HumanEventOmissionWhenAbsent verifies that human
// escalation, override, and block acceptance sections are omitted when no
// corresponding events exist. This preserves historical record readability.
func TestCompatibility_HumanEventOmissionWhenAbsent(t *testing.T) {
	session := &models.ReviewSessionOutput{
		Metadata: models.SessionMetadata{
			SpecName:       "no-events",
			ArtifactPath:   "design.md",
			Timestamp:      "2026-01-15",
			TerminalStatus: models.TerminalApproved,
			RoundsExecuted: 1,
		},
		Findings: []models.ReviewFinding{
			{
				CriterionName:         "stable-criterion",
				CriterionSatisfaction: 8,
				FindingSeverity:       models.SeverityLow,
				FindingDomains:        []models.FindingDomain{models.DomainArchitecture},
				QuotedExcerpt:         "evidence text",
				ArtifactLocation: models.ArtifactLocation{
					FilePath:         "design.md",
					SectionReference: "Overview",
				},
				Status:    models.StatusHypothesized,
				Reasoning: "adequate coverage",
			},
		},
	}

	result := output.RenderSession(session)

	assert.NotContains(t, result, "## Human Escalations")
	assert.NotContains(t, result, "## Human Overrides")
	assert.NotContains(t, result, "## Human Block Acceptance")
}

// TestCompatibility_PartialUnavailableValueInclusion verifies that unavailable
// value reasons are included only for terminal partial_review records.
func TestCompatibility_PartialUnavailableValueInclusion(t *testing.T) {
	session := &models.ReviewSessionOutput{
		Metadata: models.SessionMetadata{
			SpecName:       "partial-compat",
			ArtifactPath:   "design.md",
			Timestamp:      "2026-01-15",
			TerminalStatus: models.TerminalPartialReview,
			RoundsExecuted: 1,
			DegradedComponents: []models.DegradationEntry{
				{
					Component:           "reviewer",
					Criticality:         models.CriticalityRequired,
					FailureMode:         "unavailable",
					Timestamp:           "2026-01-15T10:00:00Z",
					UnavailableValueKey: "reviewer_findings",
					AffectedCriteria:    []string{"error-handling"},
				},
			},
		},
		UnavailableValues: []models.UnavailableValue{
			{
				Field:  "reviewer_findings",
				Reason: "The reviewer was unavailable for error-handling.",
			},
		},
	}

	result := output.RenderSession(session)

	assert.Contains(t, result, "## Unavailable Values")
	assert.Contains(t, result, "reviewer_findings")
	assert.Contains(t, result, "The reviewer was unavailable for error-handling.")
}

// TestCompatibility_CompleteFindingPreservation verifies that all canonical
// finding dimensions are preserved in rendered output: criterion satisfaction,
// severity, domains, status, evidence, location, reasoning, and gate result.
func TestCompatibility_CompleteFindingPreservation(t *testing.T) {
	session := &models.ReviewSessionOutput{
		Metadata: models.SessionMetadata{
			SpecName:       "finding-compat",
			ArtifactPath:   "design.md",
			Timestamp:      "2026-01-15",
			TerminalStatus: models.TerminalNotApproved,
			RoundsExecuted: 2,
		},
		Findings: []models.ReviewFinding{
			{
				CriterionName:         "path-containment",
				CriterionSatisfaction: 3,
				FindingSeverity:       models.SeverityCritical,
				FindingDomains:        []models.FindingDomain{models.DomainSecurity, models.DomainCorrectness},
				QuotedExcerpt:         "resolve paths under workspace",
				ArtifactLocation: models.ArtifactLocation{
					FilePath:         "design.md",
					SectionReference: "Security",
				},
				Status:    models.StatusConfirmed,
				Reasoning: "Symlinks escape containment boundary",
				CitationGateResult: &models.CitationGateResult{
					SchemaValid:             true,
					CitationValid:           true,
					ProvenanceCorrelationID: "prov-compat-1",
					MatchedLines:            &models.CitationMatchedLines{Start: 5, End: 7},
				},
				ConfirmerEvidence: "Demonstrated via symlink traversal test.",
			},
		},
	}

	result := output.RenderSession(session)

	// All canonical dimensions must be preserved in the rendered output.
	assert.Contains(t, result, "### path-containment")
	assert.Contains(t, result, "**Criterion Satisfaction:** 3/10")
	assert.Contains(t, result, "**Finding Severity:** critical")
	assert.Contains(t, result, "**Finding Domains:** security, correctness")
	assert.Contains(t, result, "**Verification Status:** confirmed")
	assert.Contains(t, result, "**Quoted Evidence:** resolve paths under workspace")
	assert.Contains(t, result, "**Artifact Location:** design.md § Security")
	assert.Contains(t, result, "Symlinks escape containment boundary")
	assert.Contains(t, result, "Schema valid: true")
	assert.Contains(t, result, "Citation valid: true")
	assert.Contains(t, result, "Provenance correlation ID: prov-compat-1")
	assert.Contains(t, result, "Matched lines: 5-7")
	assert.Contains(t, result, "Demonstrated via symlink traversal test.")
}

// TestCompatibility_StableDegradationRendering verifies that degradation
// entries render with all available fields in a stable order.
func TestCompatibility_StableDegradationRendering(t *testing.T) {
	session := &models.ReviewSessionOutput{
		Metadata: models.SessionMetadata{
			SpecName:       "degradation-compat",
			ArtifactPath:   "design.md",
			Timestamp:      "2026-01-15",
			TerminalStatus: models.TerminalPartialReview,
			RoundsExecuted: 1,
			DegradedComponents: []models.DegradationEntry{
				{
					Component:           "citation-gate",
					Criticality:         models.CriticalityRequired,
					FailureMode:         "timeout",
					Timestamp:           "2026-01-15T09:00:00Z",
					UnavailableValueKey: "citation_result",
					AffectedCriteria:    []string{"auth-check", "input-validation"},
				},
				{
					Component:        "rubric-loader",
					Criticality:      models.CriticalityOptional,
					FailureMode:      "metadata missing",
					Timestamp:        "2026-01-15T09:01:00Z",
					AffectedCriteria: []string{"operations"},
				},
			},
		},
	}

	first := output.RenderSession(session)
	second := output.RenderSession(session)

	// Rendering must be stable across invocations.
	assert.Equal(t, first, second)

	// Required degradation renders with all fields.
	assert.Contains(t, first, "### citation-gate")
	assert.Contains(t, first, "**Criticality:** required")
	assert.Contains(t, first, "**Failure Mode:** timeout")
	assert.Contains(t, first, "**Timestamp:** 2026-01-15T09:00:00Z")
	assert.Contains(t, first, "**Affected Criteria:** auth-check, input-validation")
	assert.Contains(t, first, "**Unavailable Value Key:** citation_result")

	// Optional degradation renders correctly.
	assert.Contains(t, first, "### rubric-loader")
	assert.Contains(t, first, "**Criticality:** optional")
	assert.Contains(t, first, "**Failure Mode:** metadata missing")

	// Degradation entries render in input order.
	assert.Less(
		t,
		strings.Index(first, "### citation-gate"),
		strings.Index(first, "### rubric-loader"),
	)
}
