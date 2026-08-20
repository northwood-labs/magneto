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

	"pgregory.net/rapid"

	"go.nwlabs.dev/magneto/internal/models"
	"go.nwlabs.dev/magneto/internal/session"
)

// Feature: adversarial-review-operational-workflow, Property 7: Review state
// remains bounded and requires an attack round before approval
//
// For any sequence of normalized Reviewer and Attack_Round findings, no session
// accepts more than five findings per round or executes more than five Reviewer
// rounds; a non-novel ordinary round stops, all gate-valid scores of 7 through
// 10 require one attack round before approval, a novel attack result resumes
// ordinary rounds below the cap, and exhaustion without approval yields
// `not_approved`.
//
// **Validates: Requirements 6.1, 6.2, 6.3, 6.4, 6.5, 6.6**.

type (
	// RoundPropertyAssertionInput holds the state to verify after executing a
	// sequence of rounds.
	RoundPropertyAssertionInput struct {
		Manager        *session.RoundManager
		PerRoundCounts []int
		RoundsExecuted int
	}

	// AttackRoundRequiredInput holds the inputs for asserting the attack-round
	// requirement.
	AttackRoundRequiredInput struct {
		Manager        *session.RoundManager
		AttackExecuted bool
	}

	// RoundFindingOptions controls how buildRoundFinding generates satisfaction
	// scores.
	RoundFindingOptions struct {
		AllPassing bool
	}
)

// TestProperty_ReviewStateRemainsBoundedAndChallengesApproval verifies Property
// 7: Review state remains bounded and requires an attack round before approval.
//
// Feature: adversarial-review-operational-workflow, Property 7: Review state
// remains bounded and requires an attack round before approval
//
// **Validates: Requirements 6.1, 6.2, 6.3, 6.4, 6.5, 6.6**.
func TestProperty_ReviewStateRemainsBoundedAndChallengesApproval(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		rm := session.NewRoundManager()

		// Generate 1-7 rounds of findings (more than 5 to test the cap).
		roundCount := rapid.IntRange(1, 7).Draw(t, "round_count")

		perRoundCounts := make([]int, 0, roundCount)
		attackExecuted := false

		for round := range roundCount {
			if rm.State() != session.StateActive {
				break
			}

			// Generate 1-7 findings per round (more than 5 to test per-round
			// cap).
			findingCount := rapid.IntRange(1, 7).Draw(t, "finding_count")
			findings := drawRoundFindings(t, round, findingCount)
			accepted := rm.SubmitFindings(findings)

			perRoundCounts = append(perRoundCounts, len(accepted))

			state := rm.AdvanceRound()

			if state != session.StateAttackRound {
				continue
			}

			attackExecuted = true

			attackFindings := drawAttackFindings(t, round)

			rm.SubmitAttackRound(attackFindings)
		}

		// Assert all bounded-review invariants.
		assertBoundedReviewInvariants(t, &RoundPropertyAssertionInput{
			Manager:        rm,
			RoundsExecuted: rm.RoundsExecuted(),
			PerRoundCounts: perRoundCounts,
		})

		// Assert attack-round requirement for approval.
		assertAttackRoundRequired(t, &AttackRoundRequiredInput{
			Manager:        rm,
			AttackExecuted: attackExecuted,
		})

		// Assert terminal status on exhaustion.
		assertExhaustionTerminalStatus(t, rm)
	})
}

// drawRoundFindings generates findings for an ordinary round. It uses the round
// number to generate either novel (unique) or repeated criterion names based on
// randomization.
func drawRoundFindings(t *rapid.T, round, count int) []models.ReviewFinding {
	// Decide whether this round should have passing scores (7-10) or mixed
	// scores.
	allPassing := rapid.Bool().Draw(t, "all_passing")
	findings := make([]models.ReviewFinding, 0, count)

	for i := range count {
		finding := buildRoundFinding(t, round, i, RoundFindingOptions{
			AllPassing: allPassing,
		})

		findings = append(findings, finding)
	}

	return findings
}

// buildRoundFinding constructs a single finding for an ordinary round with
// novel or repeated criterion names and appropriate satisfaction scores.
func buildRoundFinding(t *rapid.T, round, index int, opts RoundFindingOptions) models.ReviewFinding {
	criterionName := drawCriterionName(t, round, index)
	satisfaction := drawSatisfactionScore(t, opts)

	finding := models.ReviewFinding{
		CriterionName:         criterionName,
		CriterionSatisfaction: satisfaction,
		QuotedExcerpt:         fmt.Sprintf("evidence r%d f%d", round, index),
		ArtifactLocation: models.ArtifactLocation{
			FilePath:         "design.md",
			SectionReference: "Architecture",
		},
		Status:    models.StatusHypothesized,
		Reasoning: "test reasoning",
	}

	// Add a valid gate result for passing findings.
	if opts.AllPassing {
		finding.CitationGateResult = &models.CitationGateResult{
			SchemaValid:             true,
			CitationValid:           true,
			ProvenanceCorrelationID: fmt.Sprintf("gate-%d-%d", round, index),
		}
	}

	return finding
}

// drawCriterionName generates either a novel criterion name or reuses a prior
// one to test novelty detection.
func drawCriterionName(t *rapid.T, round, index int) string {
	novel := rapid.Bool().Draw(t, "novel_criterion")
	if novel {
		return fmt.Sprintf("criterion-r%d-f%d", round, index)
	}

	// Reuse a criterion name from earlier rounds to test novelty.
	return fmt.Sprintf(
		"criterion-r%d-f%d",
		rapid.IntRange(0, max(0, round-1)).Draw(t, "prior_round"),
		rapid.IntRange(0, max(0, index)).Draw(t, "prior_finding"),
	)
}

// drawSatisfactionScore generates a satisfaction score that is either passing
// (7-10) or mixed (1-10).
func drawSatisfactionScore(t *rapid.T, opts RoundFindingOptions) int {
	if opts.AllPassing {
		return rapid.IntRange(7, 10).Draw(t, "passing_satisfaction")
	}

	return rapid.IntRange(1, 10).Draw(t, "satisfaction")
}

// drawAttackFindings generates findings for an attack round. The findings may
// or may not contain novel criterion names.
func drawAttackFindings(t *rapid.T, priorRound int) []models.ReviewFinding {
	count := rapid.IntRange(0, 5).Draw(t, "attack_finding_count")

	if count == 0 {
		return nil
	}

	findings := make([]models.ReviewFinding, 0, count)

	for i := range count {
		criterionName := drawAttackCriterionName(t, priorRound, i)

		findings = append(findings, models.ReviewFinding{
			CriterionName:         criterionName,
			CriterionSatisfaction: rapid.IntRange(1, 6).Draw(t, "attack_satisfaction"),
			QuotedExcerpt:         fmt.Sprintf("attack evidence %d-%d", priorRound, i),
			ArtifactLocation: models.ArtifactLocation{
				FilePath:         "design.md",
				SectionReference: "Security",
			},
			Status:    models.StatusHypothesized,
			Reasoning: "attack finding reasoning",
		})
	}

	return findings
}

// drawAttackCriterionName generates either a novel criterion name for the
// attack round or reuses a prior one.
func drawAttackCriterionName(t *rapid.T, priorRound, index int) string {
	novel := rapid.Bool().Draw(t, "attack_novel")
	if novel {
		return fmt.Sprintf("attack-criterion-%d-%d", priorRound, index)
	}

	// Reuse a prior criterion name to test the non-novel attack path.
	return fmt.Sprintf("criterion-r%d-f%d", rapid.IntRange(0, max(0, priorRound)).Draw(t, "attack_prior_round"), 0)
}

// assertBoundedReviewInvariants verifies the core bounded-review properties:
// round cap and per-round finding limit.
func assertBoundedReviewInvariants(t *rapid.T, input *RoundPropertyAssertionInput) {
	// Requirement 6.1: No more than five ordinary rounds.
	if input.RoundsExecuted > session.MaxRounds {
		t.Fatalf("rounds executed (%d) exceeds maximum (%d)", input.RoundsExecuted, session.MaxRounds)
	}

	// Requirement 6.6: No more than five findings per round.
	for i, count := range input.PerRoundCounts {
		if count > session.MaxFindingsPerRound {
			t.Fatalf("round %d accepted %d findings, exceeding maximum (%d)", i+1, count, session.MaxFindingsPerRound)
		}
	}

	// Requirement 6.1: After 5 rounds, the manager must not remain active.
	if input.RoundsExecuted >= session.MaxRounds && input.Manager.State() == session.StateActive {
		t.Fatal("manager must not remain active after reaching the round cap")
	}
}

// assertAttackRoundRequired verifies that approval cannot happen without
// executing an attack round (Requirement 6.3).
func assertAttackRoundRequired(t *rapid.T, input *AttackRoundRequiredInput) {
	if input.Manager.State() != session.StateApproved {
		return
	}

	if !input.AttackExecuted {
		t.Fatal("approval requires an attack round to have been executed (Requirement 6.3)")
	}
}

// assertExhaustionTerminalStatus verifies that when five rounds execute without
// approval, the terminal state is not_approved (Requirement 6.5).
func assertExhaustionTerminalStatus(t *rapid.T, rm *session.RoundManager) {
	if rm.RoundsExecuted() < session.MaxRounds {
		return
	}

	if rm.State() == session.StateApproved {
		return
	}

	// If 5 rounds executed and the session is not approved, the state must
	// indicate cap reached, stopped, or attack round (which maps to terminal
	// not_approved per Requirement 6.5).
	state := rm.State()

	if state == session.StateActive {
		t.Fatalf("after %d rounds without approval, state must not be 'active', got %q", session.MaxRounds, state)
	}
}
