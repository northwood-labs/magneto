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

// buildFullSession constructs a ReviewSessionOutput with all sections
// populated for testing purposes.
func buildFullSession() *models.ReviewSessionOutput {
	return &models.ReviewSessionOutput{
		Metadata: models.SessionMetadata{
			SpecName:       "auth-redesign",
			ArtifactPath:   ".kiro/specs/auth-redesign/design.md",
			Timestamp:      "2026-08-13",
			TerminalStatus: models.TerminalNotApproved,
			DegradedComponents: []models.DegradationEntry{
				{
					Component:   "confirmer",
					FailureMode: "model unavailable",
					Timestamp:   "2026-08-13T10:05:00Z",
					AffectedCriteria: []string{
						"security-boundaries",
					},
				},
			},
			RoundsExecuted: 3,
		},
		Findings: []models.ReviewFinding{
			{
				CriterionName: "error-handling",
				Score:         4,
				QuotedExcerpt: "errors are logged but not returned",
				ArtifactLocation: models.ArtifactLocation{
					FilePath:         "design.md",
					SectionReference: "Error Handling",
				},
				Status:    models.StatusConfirmed,
				Reasoning: "Missing error propagation to caller",
			},
		},
		HumanEscalations: []models.HumanEscalation{
			{
				CriterionName: "data-integrity",
				Question:      "Is eventual consistency acceptable?",
				Context:       "Design does not specify consistency model",
				HumanAnswer:   "Yes, eventual is fine",
				Resolved:      true,
			},
		},
		HumanOverrides: []models.HumanOverride{
			{
				CriterionName:  "performance",
				FindingIndex:   0,
				OriginalScore:  3,
				HumanRationale: "Acceptable for MVP scope",
			},
		},
		DeadChecks: []string{
			"payment-processing (no payment logic present)",
		},
		AttackRoundResult: &models.AttackRoundResult{
			NewIssuesFound: true,
			Issues: []models.ReviewFinding{
				{
					CriterionName: "auth-bypass",
					Score:         2,
					Reasoning:     "Token validation skipped on retry path",
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
	assert.Contains(t, result, "## Findings")
	assert.Contains(t, result, "## Attack Round")
	assert.Contains(t, result, "## Human Escalations")
	assert.Contains(t, result, "## Human Overrides")
	assert.Contains(t, result, "## Dead Checks")
	assert.Contains(t, result, "## Degradation Summary")

	assert.Contains(t, result, "error-handling")
	assert.Contains(t, result, "Score: 4/10")
	assert.Contains(t, result, "errors are logged but not returned")

	assert.Contains(t, result, "data-integrity")
	assert.Contains(t, result, "Is eventual consistency acceptable?")
	assert.Contains(t, result, "Yes, eventual is fine")

	assert.Contains(t, result, "performance")
	assert.Contains(t, result, "Acceptable for MVP scope")

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
	assert.Contains(t, result, "## Human Escalations")
	assert.Contains(t, result, "## Human Overrides")
	assert.Contains(t, result, "## Dead Checks")
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
								CriterionName: "race-condition",
								Score:         3,
								Reasoning:     "Shared mutable state without locking",
							},
							{
								CriterionName: "input-validation",
								Score:         5,
								Reasoning:     "Missing boundary check on array index",
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
				assert.Contains(t, result, "No new issues found")
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
				CriterionName: "test-criterion",
				Score:         5,
				QuotedExcerpt: "some excerpt",
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
	assert.Equal(t, 7, h2Count, "should have 7 H2 sections")
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
