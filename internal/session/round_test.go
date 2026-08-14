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
	"fmt"
	"testing"

	"github.com/go-openapi/testify/assert"

	"go.nwlabs.dev/magneto/internal/models"
	"go.nwlabs.dev/magneto/internal/session"
)

// uniqueFindings generates n unique findings with distinct criterion names and
// excerpts for round r.
func uniqueFindings(r, n int) []models.ReviewFinding {
	findings := make([]models.ReviewFinding, 0, n)

	for i := range n {
		findings = append(findings, models.ReviewFinding{
			CriterionName: fmt.Sprintf("criterion-r%d-%d", r, i),
			Score:         5,
			QuotedExcerpt: fmt.Sprintf("evidence for round %d finding %d", r, i),
			ArtifactLocation: models.ArtifactLocation{
				FilePath:         "design.md",
				SectionReference: "Overview",
			},
			Status:    models.StatusHypothesized,
			Reasoning: "test reasoning",
		})
	}

	return findings
}

// passingFindings generates n unique findings with scores at or above the
// passing threshold (7).
func passingFindings(r, n int) []models.ReviewFinding {
	findings := make([]models.ReviewFinding, 0, n)

	for i := range n {
		findings = append(findings, models.ReviewFinding{
			CriterionName: fmt.Sprintf("passing-r%d-%d", r, i),
			Score:         8,
			QuotedExcerpt: fmt.Sprintf("passing evidence for round %d finding %d", r, i),
			ArtifactLocation: models.ArtifactLocation{
				FilePath:         "design.md",
				SectionReference: "Architecture",
			},
			Status:    models.StatusConfirmed,
			Reasoning: "satisfies criterion",
		})
	}

	return findings
}

func TestRoundManager(t *testing.T) {
	tests := []struct {
		name          string
		run           func(t *testing.T)
		expectedState session.RoundState
	}{
		{
			name: "round cap at 5",
			run: func(t *testing.T) {
				t.Helper()

				rm := session.NewRoundManager()

				// Submit unique findings for 5 rounds and advance after each.
				for r := range 5 {
					rm.SubmitFindings(uniqueFindings(r, 3))

					state := rm.AdvanceRound()

					if r < 4 {
						assert.Equal(t, session.StateActive, state, "round %d should remain active", r+1)
					} else {
						assert.Equal(t, session.StateCapReached, state, "round 5 should hit cap")
					}
				}

				assert.Equal(t, session.StateCapReached, rm.State())
			},
		},
		{
			name: "attack round triggered when all criteria >= 7",
			run: func(t *testing.T) {
				t.Helper()

				rm := session.NewRoundManager()

				rm.SubmitFindings(passingFindings(1, 3))

				state := rm.AdvanceRound()

				assert.Equal(t, session.StateAttackRound, state)
				assert.Equal(t, session.StateAttackRound, rm.State())
			},
		},
		{
			name: "attack round with new issues returns to cycle",
			run: func(t *testing.T) {
				t.Helper()

				rm := session.NewRoundManager()

				// First round passes all criteria.
				rm.SubmitFindings(passingFindings(1, 3))
				rm.AdvanceRound()

				assert.Equal(t, session.StateAttackRound, rm.State())

				// Attack round surfaces new issues.
				attackFindings := []models.ReviewFinding{
					{
						CriterionName: "attack-issue-1",
						Score:         4,
						QuotedExcerpt: "attack evidence alpha",
						ArtifactLocation: models.ArtifactLocation{
							FilePath:         "design.md",
							SectionReference: "Security",
						},
						Status:    models.StatusHypothesized,
						Reasoning: "found during attack",
					},
				}

				state := rm.SubmitAttackRound(attackFindings)

				assert.Equal(t, session.StateActive, state)
				assert.Equal(t, session.StateActive, rm.State())
			},
		},
		{
			name: "attack round with no issues approves",
			run: func(t *testing.T) {
				t.Helper()

				rm := session.NewRoundManager()

				rm.SubmitFindings(passingFindings(1, 3))
				rm.AdvanceRound()

				assert.Equal(t, session.StateAttackRound, rm.State())

				// Attack round finds nothing new.
				state := rm.SubmitAttackRound(nil)

				assert.Equal(t, session.StateApproved, state)
				assert.Equal(t, session.StateApproved, rm.State())
			},
		},
		{
			name: "novelty check stops loop",
			run: func(t *testing.T) {
				t.Helper()

				rm := session.NewRoundManager()

				// Round 1: submit findings.
				duplicatedFindings := []models.ReviewFinding{
					{
						CriterionName: "duplicate-criterion",
						Score:         4,
						QuotedExcerpt: "same evidence each time",
						ArtifactLocation: models.ArtifactLocation{
							FilePath:         "design.md",
							SectionReference: "Overview",
						},
						Status:    models.StatusHypothesized,
						Reasoning: "reasoning",
					},
				}

				rm.SubmitFindings(duplicatedFindings)

				state := rm.AdvanceRound()

				// First round always advances (no prior findings to compare).
				assert.Equal(t, session.StateActive, state)

				// Round 2: submit identical findings.
				rm.SubmitFindings(duplicatedFindings)

				state = rm.AdvanceRound()

				assert.Equal(t, session.StateStopped, state)
				assert.Equal(t, session.ReasonNoveltyFailed, rm.StopReason())
			},
		},
		{
			name: "findings capped at 5 per round",
			run: func(t *testing.T) {
				t.Helper()

				rm := session.NewRoundManager()

				// Submit 8 findings.
				findings := uniqueFindings(1, 8)
				accepted := rm.SubmitFindings(findings)

				assert.Len(t, accepted, 5)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.run(t)
		})
	}
}
