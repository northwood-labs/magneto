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

	"pgregory.net/rapid"

	"go.nwlabs.dev/magneto/internal/models"
	"go.nwlabs.dev/magneto/internal/session"
)

// TestProperty_UncitedFindingsAlwaysDowngraded verifies Property 8: Uncited
// findings are always downgraded.
//
// For any ReviewFinding that fails citation validation (verbatim match failed),
// the finding's status SHALL be set to "unconfirmed" in the final output
// regardless of the reviewer's original status assignment.
//
// **Validates: Requirements 4.2, 4.5**.
func TestProperty_UncitedFindingsAlwaysDowngraded(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate 1-5 findings with non-unconfirmed statuses.
		count := rapid.IntRange(1, 5).Draw(t, "finding_count")

		nonUnconfirmedStatuses := []models.FindingStatus{
			models.StatusConfirmed,
			models.StatusHypothesized,
			models.StatusUncheckedGateUnavail,
		}

		findings := make([]models.ReviewFinding, count)
		validationResults := make([]session.CitationValidationResult, count)

		for i := range count {
			statusIdx := rapid.IntRange(0, len(nonUnconfirmedStatuses)-1).Draw(t, "status_idx")

			findings[i] = models.ReviewFinding{
				CriterionName:         rapid.StringMatching(`[a-z]{3,12}-[a-z]{3,12}`).Draw(t, "criterion"),
				CriterionSatisfaction: rapid.IntRange(1, 10).Draw(t, "score"),
				QuotedExcerpt:         rapid.StringMatching(`[a-zA-Z0-9 ]{10,60}`).Draw(t, "excerpt"),
				ArtifactLocation: models.ArtifactLocation{
					FilePath:         rapid.StringMatching(`[a-z]{3,10}/[a-z]{3,10}\.md`).Draw(t, "path"),
					SectionReference: rapid.StringMatching(`[A-Z][a-z]{3,12}`).Draw(t, "section"),
				},
				Status:    nonUnconfirmedStatuses[statusIdx],
				Reasoning: rapid.StringMatching(`[a-z ]{10,40}`).Draw(t, "reasoning"),
			}

			// All validation results indicate citation invalid.
			validationResults[i] = session.CitationValidationResult{
				FindingIndex:  i,
				CitationValid: false,
				FailureReason: "quoted excerpt not found within cited section",
			}
		}

		// Execute the downgrade.
		result := session.DowngradeUncitedFindings(
			&session.DowngradeInput{
				Findings:          findings,
				ValidationResults: validationResults,
			},
		)

		// Property: every finding must have status "unconfirmed".
		for i, f := range result {
			if f.Status != models.StatusUnconfirmed {
				t.Fatalf(
					"finding[%d] status=%q, want %q (original status=%q)",
					i,
					f.Status,
					models.StatusUnconfirmed,
					findings[i].Status,
				)
			}
		}
	})
}
