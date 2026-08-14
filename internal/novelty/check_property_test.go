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

	"pgregory.net/rapid"

	"go.nwlabs.dev/magneto/internal/models"
	"go.nwlabs.dev/magneto/internal/novelty"
)

// TestProperty_NoveltySubsetDetection verifies Property 5: Novelty check
// detects subset repetition.
//
// For any set of findings from round N and a set of findings from round N+1
// where every finding in round N+1 references a criterion and evidence already
// present in round N's findings, the Novelty Check SHALL return novel: false.
//
// **Validates: Requirements 6.2**.
func TestProperty_NoveltySubsetDetection(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate 1-5 findings for the "prior round" (round N).
		count := rapid.IntRange(1, 5).Draw(t, "finding_count")
		priorFindings := make([]models.ReviewFinding, count)

		for i := range count {
			priorFindings[i] = models.ReviewFinding{
				CriterionName: rapid.StringMatching(`[a-z]{3,15}-[a-z]{3,15}`).Draw(t, "criterion"),
				Score:         rapid.IntRange(1, 10).Draw(t, "score"),
				QuotedExcerpt: rapid.StringMatching(`[a-zA-Z0-9 ]{10,80}`).Draw(t, "excerpt"),
				ArtifactLocation: models.ArtifactLocation{
					FilePath:         rapid.StringMatching(`[a-z/]{5,20}\.md`).Draw(t, "path"),
					SectionReference: rapid.StringMatching(`[A-Z][a-z]{3,12}`).Draw(t, "section"),
				},
				Status:    models.StatusHypothesized,
				Reasoning: rapid.StringMatching(`[a-z ]{10,50}`).Draw(t, "reasoning"),
			}
		}

		// Draw a non-empty subset of priorFindings to form the "current round"
		// (round N+1). Each finding is selected by sampling an index from the
		// prior set.
		subsetSize := rapid.IntRange(1, count).Draw(t, "subset_size")

		currentFindings := make([]models.ReviewFinding, subsetSize)
		for i := range subsetSize {
			idx := rapid.IntRange(0, count-1).Draw(t, "idx")

			currentFindings[i] = priorFindings[idx]
		}

		// Run the novelty check with current as subset of prior.
		result := novelty.Check(&novelty.CheckInput{
			CurrentFindings: currentFindings,
			PriorFindings:   priorFindings,
		})

		// Property: every current finding already exists in prior, so the
		// result must be non-novel.
		if result.Novel {
			t.Fatalf(
				"expected Novel=false for subset of prior findings, got Novel=true with %d novel items",
				len(result.NovelItems),
			)
		}
	})
}
