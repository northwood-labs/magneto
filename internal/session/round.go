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

package session

import (
	"go.nwlabs.dev/magneto/internal/models"
	"go.nwlabs.dev/magneto/internal/novelty"
)

const (
	// MaxRounds is the hard cap on review rounds per session.
	MaxRounds = 5

	// MaxFindingsPerRound is the maximum number of findings accepted in a
	// single round.
	MaxFindingsPerRound = 5

	// PassingScoreThreshold is the minimum score for a criterion to be
	// considered passing.
	PassingScoreThreshold = 7

	// StateActive indicates the review session is actively running rounds.
	StateActive RoundState = "active"

	// StateAttackRound indicates all criteria passed and the mandatory attack
	// round is in progress.
	StateAttackRound RoundState = "attack_round"

	// StateStopped indicates the session stopped due to novelty check failure.
	StateStopped RoundState = "stopped"

	// StateCapReached indicates the session hit the hard round cap.
	StateCapReached RoundState = "cap_reached"

	// StateApproved indicates the artifact passed the attack round and is
	// approved.
	StateApproved RoundState = "approved"

	// ReasonRoundCap indicates the session reached the maximum round limit.
	ReasonRoundCap StopReason = "round cap reached"

	// ReasonNoveltyFailed indicates the current round produced no novel
	// findings compared to prior rounds.
	ReasonNoveltyFailed StopReason = "novelty check failed"

	// ReasonApproved indicates the artifact passed the mandatory attack round
	// without new issues.
	ReasonApproved StopReason = "approved after attack round"

	// ReasonAttackFailed indicates the attack round surfaced new issues that
	// require further review.
	ReasonAttackFailed StopReason = "attack round surfaced new issues"
)

type (
	// RoundState represents the current state of the round manager.
	RoundState string

	// StopReason describes why the review session stopped.
	StopReason string

	// RoundManager tracks round progression and enforces stopping conditions
	// for a review session.
	RoundManager struct {
		state        RoundState
		stopReason   StopReason
		rounds       [][]models.ReviewFinding
		currentRound int
	}
)

// NewRoundManager creates a new round manager in active state with the first
// round ready.
func NewRoundManager() *RoundManager {
	return &RoundManager{
		rounds:       make([][]models.ReviewFinding, 0, MaxRounds),
		state:        StateActive,
		currentRound: 1,
	}
}

// CurrentRound returns the current round number (1-indexed).
func (rm *RoundManager) CurrentRound() int {
	return rm.currentRound
}

// State returns the current round manager state.
func (rm *RoundManager) State() RoundState {
	return rm.state
}

// StopReason returns the reason for stopping, if stopped.
func (rm *RoundManager) StopReason() StopReason {
	return rm.stopReason
}

// SubmitFindings submits findings for the current round, enforcing the max 5
// findings per round limit. Returns the accepted findings (truncated if
// necessary).
func (rm *RoundManager) SubmitFindings(findings []models.ReviewFinding) []models.ReviewFinding {
	accepted := findings
	if len(accepted) > MaxFindingsPerRound {
		accepted = accepted[:MaxFindingsPerRound]
	}

	rm.rounds = append(rm.rounds, accepted)

	return accepted
}

// AdvanceRound evaluates stopping conditions and advances to the next round.
// Returns the new state after advancing.
func (rm *RoundManager) AdvanceRound() RoundState {
	// Check round cap.
	if rm.currentRound >= MaxRounds {
		rm.state = StateCapReached
		rm.stopReason = ReasonRoundCap

		return rm.state
	}

	// Check novelty against all prior rounds.
	if len(rm.rounds) > 1 {
		currentFindings := rm.rounds[len(rm.rounds)-1]
		priorFindings := rm.collectPriorFindings()

		result := novelty.Check(&novelty.CheckInput{
			CurrentFindings: currentFindings,
			PriorFindings:   priorFindings,
		})

		if !result.Novel {
			rm.state = StateStopped
			rm.stopReason = ReasonNoveltyFailed

			return rm.state
		}
	}

	// Check if all criteria are passing.
	if rm.allCriteriaPassing() {
		rm.state = StateAttackRound

		return rm.state
	}

	// Continue to next round.
	rm.currentRound++

	rm.state = StateActive

	return rm.state
}

// SubmitAttackRound submits findings from the mandatory attack round. If new
// issues are found, returns to active state (if under cap). If no new issues,
// approves.
func (rm *RoundManager) SubmitAttackRound(findings []models.ReviewFinding) RoundState {
	accepted := findings
	if len(accepted) > MaxFindingsPerRound {
		accepted = accepted[:MaxFindingsPerRound]
	}

	if len(accepted) == 0 {
		rm.state = StateApproved
		rm.stopReason = ReasonApproved

		return rm.state
	}

	// New issues found — feed back into review cycle.
	rm.rounds = append(rm.rounds, accepted)
	rm.stopReason = ReasonAttackFailed

	// Check if we would exceed the round cap by continuing.
	if rm.currentRound >= MaxRounds {
		rm.state = StateCapReached
		rm.stopReason = ReasonRoundCap

		return rm.state
	}

	rm.currentRound++

	rm.state = StateActive

	return rm.state
}

// AllFindings returns all findings from all rounds.
func (rm *RoundManager) AllFindings() []models.ReviewFinding {
	total := 0
	for _, round := range rm.rounds {
		total += len(round)
	}

	all := make([]models.ReviewFinding, 0, total)
	for _, round := range rm.rounds {
		all = append(all, round...)
	}

	return all
}

// RoundsExecuted returns the total number of rounds executed.
func (rm *RoundManager) RoundsExecuted() int {
	return len(rm.rounds)
}

// collectPriorFindings returns all findings from rounds before the current
// (last) round.
func (rm *RoundManager) collectPriorFindings() []models.ReviewFinding {
	if len(rm.rounds) <= 1 {
		return nil
	}

	priorRounds := rm.rounds[:len(rm.rounds)-1]
	total := 0

	for _, round := range priorRounds {
		total += len(round)
	}

	prior := make([]models.ReviewFinding, 0, total)
	for _, round := range priorRounds {
		prior = append(prior, round...)
	}

	return prior
}

// allCriteriaPassing checks whether the most recent round's findings all have
// scores at or above the passing threshold.
func (rm *RoundManager) allCriteriaPassing() bool {
	if len(rm.rounds) == 0 {
		return false
	}

	currentFindings := rm.rounds[len(rm.rounds)-1]
	if len(currentFindings) == 0 {
		return false
	}

	for i := range currentFindings {
		if currentFindings[i].Score < PassingScoreThreshold {
			return false
		}
	}

	return true
}
