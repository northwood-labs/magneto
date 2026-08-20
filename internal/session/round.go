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
	// MaxRounds is the hard cap on ordinary reviewer rounds per session.
	MaxRounds = 5

	// MaxFindingsPerRound is the maximum number of findings accepted in a
	// single ordinary or attack round.
	MaxFindingsPerRound = 5

	// PassingCriterionSatisfactionThreshold is the minimum satisfaction for a
	// criterion to be considered passing.
	PassingCriterionSatisfactionThreshold = 7

	// StateActive indicates the review session is actively running ordinary
	// reviewer rounds.
	StateActive RoundState = "active"

	// StateAttackRound indicates all criteria passed and the mandatory attack
	// round is in progress.
	StateAttackRound RoundState = "attack_round"

	// StateStopped indicates the session stopped due to novelty check failure.
	StateStopped RoundState = "stopped"

	// StateCapReached indicates the session hit the hard ordinary-round cap.
	StateCapReached RoundState = "cap_reached"

	// StateApproved indicates the artifact passed the mandatory attack round
	// and is approved.
	StateApproved RoundState = "approved"

	// ReasonRoundCap indicates the session reached the maximum ordinary-round
	// limit before approval.
	ReasonRoundCap StopReason = "round cap reached"

	// ReasonNoveltyFailed indicates the current ordinary round produced no
	// novel findings compared to prior rounds.
	ReasonNoveltyFailed StopReason = "novelty check failed"

	// ReasonApproved indicates the artifact passed the mandatory attack round
	// without novel issues.
	ReasonApproved StopReason = "approved after attack round"

	// ReasonAttackFailed indicates the attack round surfaced novel issues that
	// require further ordinary review.
	ReasonAttackFailed StopReason = "attack round surfaced new issues"
)

type (
	// RoundState represents the current state of the round manager.
	RoundState string

	// StopReason describes why the review session stopped.
	StopReason string

	// RoundManager tracks ordinary and attack round progression while
	// preserving finding-level evidence from both kinds of round.
	RoundManager struct {
		state          RoundState
		stopReason     StopReason
		attackFindings [][]models.ReviewFinding
		rounds         [][]models.ReviewFinding
		currentRound   int
	}
)

// NewRoundManager creates a new round manager in active state with the first
// ordinary reviewer round ready.
func NewRoundManager() *RoundManager {
	return &RoundManager{
		attackFindings: make([][]models.ReviewFinding, 0, 1),
		rounds:         make([][]models.ReviewFinding, 0, MaxRounds),
		state:          StateActive,
		currentRound:   1,
	}
}

// CurrentRound returns the current ordinary reviewer round number (1-indexed).
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

// SubmitFindings submits findings for the current ordinary reviewer round,
// enforcing the maximum per-round limit. It returns accepted findings.
func (rm *RoundManager) SubmitFindings(findings []models.ReviewFinding) []models.ReviewFinding {
	accepted := limitFindings(findings)

	rm.rounds = append(rm.rounds, accepted)

	return accepted
}

// AdvanceRound evaluates ordinary-round stopping conditions and advances to the
// next round or the mandatory attack round.
func (rm *RoundManager) AdvanceRound() RoundState {
	if rm.allCriteriaPassing() {
		rm.state = StateAttackRound

		return rm.state
	}

	if rm.currentRound >= MaxRounds {
		rm.state = StateCapReached
		rm.stopReason = ReasonRoundCap

		return rm.state
	}

	if rm.currentRoundHasNoNovelFindings() {
		rm.state = StateStopped
		rm.stopReason = ReasonNoveltyFailed

		return rm.state
	}

	rm.currentRound++

	rm.state = StateActive

	return rm.state
}

// SubmitAttackRound submits findings from the mandatory attack round. Novel
// findings return the session to ordinary review when the cap permits it; no
// novel findings make the session eligible for approval.
func (rm *RoundManager) SubmitAttackRound(findings []models.ReviewFinding) RoundState {
	accepted := limitFindings(findings)
	priorFindings := rm.AllFindings()

	rm.attackFindings = append(rm.attackFindings, accepted)

	if !hasNovelFindings(accepted, priorFindings) {
		rm.state = StateApproved
		rm.stopReason = ReasonApproved

		return rm.state
	}

	rm.stopReason = ReasonAttackFailed
	if rm.currentRound >= MaxRounds {
		rm.state = StateCapReached
		rm.stopReason = ReasonRoundCap

		return rm.state
	}

	rm.currentRound++

	rm.state = StateActive

	return rm.state
}

// AllFindings returns all criterion-level findings from ordinary and attack
// rounds in execution order.
func (rm *RoundManager) AllFindings() []models.ReviewFinding {
	total := countFindings(rm.rounds) + countFindings(rm.attackFindings)
	all := make([]models.ReviewFinding, 0, total)

	for _, round := range rm.rounds {
		all = append(all, round...)
	}

	for _, round := range rm.attackFindings {
		all = append(all, round...)
	}

	return all
}

// RoundsExecuted returns the total number of ordinary reviewer rounds
// executed. Attack rounds do not consume the ordinary-round limit.
func (rm *RoundManager) RoundsExecuted() int {
	return len(rm.rounds)
}

// collectPriorFindings returns all ordinary findings before the current round.
func (rm *RoundManager) collectPriorFindings() []models.ReviewFinding {
	if len(rm.rounds) <= 1 {
		return nil
	}

	priorRounds := rm.rounds[:len(rm.rounds)-1]
	prior := make([]models.ReviewFinding, 0, countFindings(priorRounds))

	for _, round := range priorRounds {
		prior = append(prior, round...)
	}

	return prior
}

// allCriteriaPassing reports whether each current criterion meets the
// satisfaction threshold and has a deterministic, correlated gate result.
func (rm *RoundManager) allCriteriaPassing() bool {
	if len(rm.rounds) == 0 {
		return false
	}

	currentFindings := rm.rounds[len(rm.rounds)-1]
	if len(currentFindings) == 0 {
		return false
	}

	for index := range currentFindings {
		finding := &currentFindings[index]
		if finding.CriterionSatisfaction < PassingCriterionSatisfactionThreshold || !IsGateValid(finding) {
			return false
		}
	}

	return true
}

// currentRoundHasNoNovelFindings reports whether the current ordinary round
// repeats only failure modes already recorded by ordinary rounds.
func (rm *RoundManager) currentRoundHasNoNovelFindings() bool {
	if len(rm.rounds) <= 1 {
		return false
	}

	return !hasNovelFindings(rm.rounds[len(rm.rounds)-1], rm.collectPriorFindings())
}

// hasNovelFindings reports whether the current round contains a failure mode
// that was not recorded in prior rounds.
func hasNovelFindings(current, prior []models.ReviewFinding) bool {
	if len(current) == 0 {
		return false
	}

	result := novelty.Check(&novelty.CheckInput{
		CurrentFindings: current,
		PriorFindings:   prior,
	})

	return result.Novel
}

// countFindings returns the number of findings contained in the supplied
// rounds.
func countFindings(rounds [][]models.ReviewFinding) int {
	total := 0
	for _, round := range rounds {
		total += len(round)
	}

	return total
}

// limitFindings returns at most the permitted number of findings from a round.
func limitFindings(findings []models.ReviewFinding) []models.ReviewFinding {
	if len(findings) <= MaxFindingsPerRound {
		return findings
	}

	return findings[:MaxFindingsPerRound]
}
