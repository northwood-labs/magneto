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

package novelty_test

import (
	"testing"

	"github.com/go-openapi/testify/assert"

	"go.nwlabs.dev/magneto/internal/models"
	"go.nwlabs.dev/magneto/internal/novelty"
)

func TestCheck(t *testing.T) {
	tests := []struct {
		name     string
		prior    []models.ReviewFinding
		current  []models.ReviewFinding
		expected bool
	}{
		{
			name: "identical findings non-novel",
			prior: []models.ReviewFinding{
				{
					CriterionName: "error-handling",
					Score:         4,
					QuotedExcerpt: "errors are silently ignored",
					ArtifactLocation: models.ArtifactLocation{
						FilePath:         "design.md",
						SectionReference: "Architecture",
					},
					Status:    models.StatusHypothesized,
					Reasoning: "no error propagation visible",
				},
			},
			current: []models.ReviewFinding{
				{
					CriterionName: "error-handling",
					Score:         4,
					QuotedExcerpt: "errors are silently ignored",
					ArtifactLocation: models.ArtifactLocation{
						FilePath:         "design.md",
						SectionReference: "Architecture",
					},
					Status:    models.StatusHypothesized,
					Reasoning: "no error propagation visible",
				},
			},
			expected: false,
		},
		{
			name: "subset non-novel",
			prior: []models.ReviewFinding{
				{
					CriterionName: "context-isolation",
					Score:         3,
					QuotedExcerpt: "shared state between agents",
					ArtifactLocation: models.ArtifactLocation{
						FilePath:         "design.md",
						SectionReference: "Components",
					},
					Status:    models.StatusHypothesized,
					Reasoning: "violates isolation requirement",
				},
				{
					CriterionName: "degradation-handling",
					Score:         5,
					QuotedExcerpt: "no fallback specified",
					ArtifactLocation: models.ArtifactLocation{
						FilePath:         "design.md",
						SectionReference: "Error handling",
					},
					Status:    models.StatusHypothesized,
					Reasoning: "missing degradation path",
				},
			},
			current: []models.ReviewFinding{
				{
					CriterionName: "context-isolation",
					Score:         3,
					QuotedExcerpt: "shared state between agents",
					ArtifactLocation: models.ArtifactLocation{
						FilePath:         "design.md",
						SectionReference: "Components",
					},
					Status:    models.StatusHypothesized,
					Reasoning: "violates isolation requirement",
				},
			},
			expected: false,
		},
		{
			name: "new criterion novel",
			prior: []models.ReviewFinding{
				{
					CriterionName: "error-handling",
					Score:         4,
					QuotedExcerpt: "errors are silently ignored",
					ArtifactLocation: models.ArtifactLocation{
						FilePath:         "design.md",
						SectionReference: "Architecture",
					},
					Status:    models.StatusHypothesized,
					Reasoning: "no error propagation visible",
				},
			},
			current: []models.ReviewFinding{
				{
					CriterionName: "security-boundaries",
					Score:         2,
					QuotedExcerpt: "no auth check on endpoint",
					ArtifactLocation: models.ArtifactLocation{
						FilePath:         "design.md",
						SectionReference: "API",
					},
					Status:    models.StatusHypothesized,
					Reasoning: "missing access control",
				},
			},
			expected: true,
		},
		{
			name: "same criterion with new evidence novel",
			prior: []models.ReviewFinding{
				{
					CriterionName: "error-handling",
					Score:         4,
					QuotedExcerpt: "errors are silently ignored",
					ArtifactLocation: models.ArtifactLocation{
						FilePath:         "design.md",
						SectionReference: "Architecture",
					},
					Status:    models.StatusHypothesized,
					Reasoning: "no error propagation visible",
				},
			},
			current: []models.ReviewFinding{
				{
					CriterionName: "error-handling",
					Score:         4,
					QuotedExcerpt: "panic recovery is not handled",
					ArtifactLocation: models.ArtifactLocation{
						FilePath:         "design.md",
						SectionReference: "Architecture",
					},
					Status:    models.StatusHypothesized,
					Reasoning: "unrecovered panics crash process",
				},
			},
			expected: true,
		},
		{
			name: "completely different findings novel",
			prior: []models.ReviewFinding{
				{
					CriterionName: "error-handling",
					Score:         4,
					QuotedExcerpt: "errors are silently ignored",
					ArtifactLocation: models.ArtifactLocation{
						FilePath:         "design.md",
						SectionReference: "Architecture",
					},
					Status:    models.StatusHypothesized,
					Reasoning: "no error propagation visible",
				},
			},
			current: []models.ReviewFinding{
				{
					CriterionName: "data-integrity",
					Score:         2,
					QuotedExcerpt: "no transaction boundaries",
					ArtifactLocation: models.ArtifactLocation{
						FilePath:         "design.md",
						SectionReference: "Data models",
					},
					Status:    models.StatusHypothesized,
					Reasoning: "partial writes possible",
				},
				{
					CriterionName: "observability",
					Score:         3,
					QuotedExcerpt: "no metrics or tracing",
					ArtifactLocation: models.ArtifactLocation{
						FilePath:         "design.md",
						SectionReference: "Operations",
					},
					Status:    models.StatusHypothesized,
					Reasoning: "blind to runtime behavior",
				},
			},
			expected: true,
		},
		{
			name: "empty current round non-novel",
			prior: []models.ReviewFinding{
				{
					CriterionName: "error-handling",
					Score:         4,
					QuotedExcerpt: "errors are silently ignored",
					ArtifactLocation: models.ArtifactLocation{
						FilePath:         "design.md",
						SectionReference: "Architecture",
					},
					Status:    models.StatusHypothesized,
					Reasoning: "no error propagation visible",
				},
			},
			current:  nil,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := novelty.Check(&novelty.CheckInput{
				CurrentFindings: tt.current,
				PriorFindings:   tt.prior,
			})

			assert.Equal(t, tt.expected, result.Novel)
		})
	}
}
