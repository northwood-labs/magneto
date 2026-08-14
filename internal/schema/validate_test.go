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

package schema_test

import (
	"testing"

	"github.com/go-openapi/testify/assert"

	"go.nwlabs.dev/magneto/internal/models"
	"go.nwlabs.dev/magneto/internal/schema"
)

// TestValidateFindingSchema exercises schema validation with table-driven tests
// covering valid findings and each specific validation failure.
func TestValidateFindingSchema(t *testing.T) {
	validFinding := &models.ReviewFinding{
		CriterionName: "Security Boundaries",
		Score:         7,
		QuotedExcerpt: "the system enforces structurally independent review",
		ArtifactLocation: models.ArtifactLocation{
			FilePath:         "design.md",
			SectionReference: "Overview",
		},
		Status:    models.StatusHypothesized,
		Reasoning: "criterion is satisfied with cited evidence",
	}

	tests := []struct {
		finding       *models.ReviewFinding
		name          string
		expectField   string
		expectMessage string
		expectErr     bool
	}{
		{
			name:      "valid finding passes",
			finding:   validFinding,
			expectErr: false,
		},
		{
			name: "missing criterion name",
			finding: &models.ReviewFinding{
				CriterionName: "",
				Score:         5,
				QuotedExcerpt: "some evidence text",
				ArtifactLocation: models.ArtifactLocation{
					FilePath:         "design.md",
					SectionReference: "Architecture",
				},
				Status:    models.StatusHypothesized,
				Reasoning: "reasoning text",
			},
			expectErr:     true,
			expectField:   "criterion_name",
			expectMessage: "criterion name is required",
		},
		{
			name: "score zero out of range",
			finding: &models.ReviewFinding{
				CriterionName: "Data Integrity",
				Score:         0,
				QuotedExcerpt: "some evidence text",
				ArtifactLocation: models.ArtifactLocation{
					FilePath:         "design.md",
					SectionReference: "Architecture",
				},
				Status:    models.StatusConfirmed,
				Reasoning: "reasoning text",
			},
			expectErr:     true,
			expectField:   "score",
			expectMessage: "score must be between 1 and 10",
		},
		{
			name: "score eleven out of range",
			finding: &models.ReviewFinding{
				CriterionName: "Data Integrity",
				Score:         11,
				QuotedExcerpt: "some evidence text",
				ArtifactLocation: models.ArtifactLocation{
					FilePath:         "design.md",
					SectionReference: "Architecture",
				},
				Status:    models.StatusConfirmed,
				Reasoning: "reasoning text",
			},
			expectErr:     true,
			expectField:   "score",
			expectMessage: "score must be between 1 and 10",
		},
		{
			name: "score negative out of range",
			finding: &models.ReviewFinding{
				CriterionName: "Data Integrity",
				Score:         -1,
				QuotedExcerpt: "some evidence text",
				ArtifactLocation: models.ArtifactLocation{
					FilePath:         "design.md",
					SectionReference: "Architecture",
				},
				Status:    models.StatusConfirmed,
				Reasoning: "reasoning text",
			},
			expectErr:     true,
			expectField:   "score",
			expectMessage: "score must be between 1 and 10",
		},
		{
			name: "empty quoted excerpt",
			finding: &models.ReviewFinding{
				CriterionName: "Context Isolation",
				Score:         3,
				QuotedExcerpt: "",
				ArtifactLocation: models.ArtifactLocation{
					FilePath:         "design.md",
					SectionReference: "Components",
				},
				Status:    models.StatusHypothesized,
				Reasoning: "reasoning text",
			},
			expectErr:     true,
			expectField:   "quoted_excerpt",
			expectMessage: "quoted excerpt is required",
		},
		{
			name: "missing file path",
			finding: &models.ReviewFinding{
				CriterionName: "Trigger Heuristic",
				Score:         5,
				QuotedExcerpt: "the artifact produces outputs",
				ArtifactLocation: models.ArtifactLocation{
					FilePath:         "",
					SectionReference: "Architecture",
				},
				Status:    models.StatusHypothesized,
				Reasoning: "reasoning text",
			},
			expectErr:     true,
			expectField:   "artifact_location.file_path",
			expectMessage: "artifact file path is required",
		},
		{
			name: "missing section reference",
			finding: &models.ReviewFinding{
				CriterionName: "Stopping Conditions",
				Score:         8,
				QuotedExcerpt: "hard round cap of 5",
				ArtifactLocation: models.ArtifactLocation{
					FilePath:         "design.md",
					SectionReference: "",
				},
				Status:    models.StatusConfirmed,
				Reasoning: "reasoning text",
			},
			expectErr:     true,
			expectField:   "artifact_location.section_reference",
			expectMessage: "artifact section reference is required",
		},
		{
			name: "invalid status value",
			finding: &models.ReviewFinding{
				CriterionName: "Advisory Only",
				Score:         6,
				QuotedExcerpt: "strictly advisory",
				ArtifactLocation: models.ArtifactLocation{
					FilePath:         "design.md",
					SectionReference: "Overview",
				},
				Status:    models.FindingStatus("invalid_status"),
				Reasoning: "reasoning text",
			},
			expectErr:     true,
			expectField:   "status",
			expectMessage: "status must be a valid FindingStatus value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var validationErr *schema.SchemaValidationError

			err := schema.ValidateFindingSchema(tt.finding)

			if !tt.expectErr {
				assert.NoError(t, err)

				return
			}

			assert.Error(t, err)
			assert.ErrorAs(t, err, &validationErr)

			if validationErr == nil {
				return
			}

			found := false

			for _, fe := range validationErr.Errors {
				if fe.Field == tt.expectField && fe.Message == tt.expectMessage {
					found = true

					break
				}
			}

			assert.True(
				t,
				found,
				"expected FieldError{Field: %q, Message: %q} in validation errors",
				tt.expectField,
				tt.expectMessage,
			)
		})
	}
}
