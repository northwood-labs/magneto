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

	"pgregory.net/rapid"

	"go.nwlabs.dev/magneto/internal/models"
	"go.nwlabs.dev/magneto/internal/schema"
)

// TestProperty_FindingSchemaRejectsIncomplete verifies Property 4: Finding
// schema validation rejects incomplete findings.
//
// For any object missing one or more required ReviewFinding fields
// (CriterionName, Score, QuotedExcerpt, ArtifactLocation, Status),
// ValidateFindingSchema SHALL return an error identifying the missing field(s).
//
// **Validates: Requirements 3.1**.
func TestProperty_FindingSchemaRejectsIncomplete(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a valid finding with all required fields populated.
		finding := &models.ReviewFinding{
			CriterionName: rapid.StringMatching(`[a-z]{3,20}`).Draw(t, "criterion"),
			Score:         rapid.IntRange(1, 10).Draw(t, "score"),
			QuotedExcerpt: rapid.StringMatching(`[a-zA-Z0-9 ]{10,100}`).Draw(t, "excerpt"),
			ArtifactLocation: models.ArtifactLocation{
				FilePath:         rapid.StringMatching(`[a-z/]{5,30}\.md`).Draw(t, "path"),
				SectionReference: rapid.StringMatching(`[A-Z][a-z]{3,15}`).Draw(t, "section"),
			},
			Status:    models.StatusHypothesized,
			Reasoning: rapid.StringMatching(`[a-zA-Z0-9 ]{10,200}`).Draw(t, "reasoning"),
		}

		// Randomly remove one required field to create an invalid
		// finding.
		fieldToRemove := rapid.IntRange(0, 4).Draw(t, "field_index")

		switch fieldToRemove {
		case 0:
			finding.CriterionName = ""
		case 1:
			finding.Score = 0 // Invalid: score must be 1-10.
		case 2:
			finding.QuotedExcerpt = ""
		case 3:
			finding.ArtifactLocation.FilePath = ""
		default:
			finding.ArtifactLocation.SectionReference = ""
		}

		// Schema validation must reject the incomplete finding.
		err := schema.ValidateFindingSchema(finding)
		if err == nil {
			t.Fatalf("expected validation error for missing field %d, got nil", fieldToRemove)
		}
	})
}
