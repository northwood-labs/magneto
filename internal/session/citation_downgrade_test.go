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

package session_test

import (
	"testing"

	"github.com/go-openapi/testify/assert"

	"go.nwlabs.dev/magneto/internal/models"
	"go.nwlabs.dev/magneto/internal/session"
)

func TestDowngradeUncitedFindings(t *testing.T) {
	tests := []struct {
		name           string
		input          *session.DowngradeInput
		expectedStatus []models.FindingStatus
	}{
		{
			name: "missing excerpt downgrades to unconfirmed",
			input: &session.DowngradeInput{
				Findings: []models.ReviewFinding{
					{
						CriterionName: "error-handling",
						Score:         4,
						QuotedExcerpt: "",
						ArtifactLocation: models.ArtifactLocation{
							FilePath:         "design.md",
							SectionReference: "Architecture",
						},
						Status:    models.StatusHypothesized,
						Reasoning: "missing excerpt",
					},
				},
				ValidationResults: nil,
			},
			expectedStatus: []models.FindingStatus{
				models.StatusUnconfirmed,
			},
		},
		{
			name: "missing file path downgrades to unconfirmed",
			input: &session.DowngradeInput{
				Findings: []models.ReviewFinding{
					{
						CriterionName: "security-check",
						Score:         3,
						QuotedExcerpt: "the system enforces isolation",
						ArtifactLocation: models.ArtifactLocation{
							FilePath:         "",
							SectionReference: "Overview",
						},
						Status:    models.StatusConfirmed,
						Reasoning: "missing file path",
					},
				},
				ValidationResults: nil,
			},
			expectedStatus: []models.FindingStatus{
				models.StatusUnconfirmed,
			},
		},
		{
			name: "missing section reference downgrades to unconfirmed",
			input: &session.DowngradeInput{
				Findings: []models.ReviewFinding{
					{
						CriterionName: "completeness",
						Score:         5,
						QuotedExcerpt: "all criteria must be tested",
						ArtifactLocation: models.ArtifactLocation{
							FilePath:         "requirements.md",
							SectionReference: "",
						},
						Status:    models.StatusHypothesized,
						Reasoning: "missing section reference",
					},
				},
				ValidationResults: nil,
			},
			expectedStatus: []models.FindingStatus{
				models.StatusUnconfirmed,
			},
		},
		{
			name: "failed verbatim match downgrades to unconfirmed",
			input: &session.DowngradeInput{
				Findings: []models.ReviewFinding{
					{
						CriterionName: "correctness",
						Score:         6,
						QuotedExcerpt: "cited text here",
						ArtifactLocation: models.ArtifactLocation{
							FilePath:         "design.md",
							SectionReference: "Components",
						},
						Status:    models.StatusConfirmed,
						Reasoning: "verbatim match failed",
					},
				},
				ValidationResults: []session.CitationValidationResult{
					{
						FindingIndex:  0,
						CitationValid: false,
						FailureReason: "quoted excerpt not found",
					},
				},
			},
			expectedStatus: []models.FindingStatus{
				models.StatusUnconfirmed,
			},
		},
		{
			name: "valid citation preserves original status",
			input: &session.DowngradeInput{
				Findings: []models.ReviewFinding{
					{
						CriterionName: "isolation",
						Score:         8,
						QuotedExcerpt: "fresh context window",
						ArtifactLocation: models.ArtifactLocation{
							FilePath:         "design.md",
							SectionReference: "Overview",
						},
						Status:    models.StatusConfirmed,
						Reasoning: "valid citation",
					},
				},
				ValidationResults: []session.CitationValidationResult{
					{
						FindingIndex:  0,
						CitationValid: true,
					},
				},
			},
			expectedStatus: []models.FindingStatus{
				models.StatusConfirmed,
			},
		},
		{
			name: "original findings not mutated",
			input: &session.DowngradeInput{
				Findings: []models.ReviewFinding{
					{
						CriterionName: "blast-radius",
						Score:         2,
						QuotedExcerpt: "",
						ArtifactLocation: models.ArtifactLocation{
							FilePath:         "design.md",
							SectionReference: "Trigger",
						},
						Status:    models.StatusConfirmed,
						Reasoning: "will be downgraded",
					},
				},
				ValidationResults: nil,
			},
			expectedStatus: []models.FindingStatus{
				models.StatusUnconfirmed,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture original statuses for mutation check.
			originalStatuses := make(
				[]models.FindingStatus,
				len(tt.input.Findings),
			)
			for i, f := range tt.input.Findings {
				originalStatuses[i] = f.Status
			}

			result := session.DowngradeUncitedFindings(tt.input)

			// Verify expected output statuses.
			for i, expected := range tt.expectedStatus {
				assert.Equal(t, expected, result[i].Status, "finding[%d] status mismatch", i)
			}

			// Verify input slice was not mutated.
			for i, f := range tt.input.Findings {
				assert.Equal(t, originalStatuses[i], f.Status, "input finding[%d] was mutated", i)
			}
		})
	}
}
