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

// TestProperty_RoundCapNeverExceeded verifies Property 6: Round cap is never
// exceeded.
//
// For any sequence of review round transitions, the total number of rounds
// executed SHALL never exceed 5, regardless of novelty check results, attack
// round results, or human escalation events.
//
// **Validates: Requirements 6.1**.
func TestProperty_RoundCapNeverExceeded(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		rm := session.NewRoundManager()

		// Generate a random number of rounds to attempt (1-10), exceeding the
		// cap to verify enforcement.
		rounds := rapid.IntRange(1, 10).Draw(t, "rounds")

		for i := range rounds {
			// Stop if the session is no longer active.
			if rm.State() != session.StateActive {
				break
			}

			// Generate 1-5 findings for this round.
			findingCount := rapid.IntRange(1, 5).Draw(t, "finding_count")
			findings := make([]models.ReviewFinding, findingCount)

			for j := range findingCount {
				findings[j] = models.ReviewFinding{
					CriterionName: rapid.StringMatching(`[a-z]{3,12}-[a-z]{3,12}`).Draw(t, "criterion"),
					Score:         rapid.IntRange(1, 6).Draw(t, "score"),
					QuotedExcerpt: rapid.StringMatching(`[a-zA-Z0-9 ]{10,60}`).Draw(t, "excerpt"),
					ArtifactLocation: models.ArtifactLocation{
						FilePath:         rapid.StringMatching(`[a-z]{3,10}/[a-z]{3,10}\.md`).Draw(t, "path"),
						SectionReference: rapid.StringMatching(`[A-Z][a-z]{3,12}`).Draw(t, "section"),
					},
					Status:    models.StatusHypothesized,
					Reasoning: rapid.StringMatching(`[a-z ]{10,40}`).Draw(t, "reasoning"),
				}
			}

			_ = rm.SubmitFindings(findings)
			_ = i

			rm.AdvanceRound()
		}

		// Property: total rounds executed must never exceed the maximum.
		if rm.RoundsExecuted() > session.MaxRounds {
			t.Fatalf(
				"round cap violated: RoundsExecuted()=%d exceeds MaxRounds=%d",
				rm.RoundsExecuted(),
				session.MaxRounds,
			)
		}

		// Property: current round number must never exceed the maximum.
		if rm.CurrentRound() > session.MaxRounds {
			t.Fatalf("round cap violated: CurrentRound()=%d exceeds MaxRounds=%d", rm.CurrentRound(), session.MaxRounds)
		}
	})
}
