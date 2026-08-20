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

import "go.nwlabs.dev/magneto/internal/models"

const (
	// MaxConfirmerAttempts is the number of unsuccessful confirmation attempts
	// after which a high-impact finding becomes unconfirmed.
	MaxConfirmerAttempts = 3

	// GateAvailable indicates that the deterministic citation gate completed.
	GateAvailable GateAvailability = "available"

	// GateUnavailable indicates that the deterministic citation gate could not
	// complete for a finding.
	GateUnavailable GateAvailability = "unavailable"
)

type (
	// GateAvailability records whether the deterministic citation gate completed
	// for a finding.
	GateAvailability string

	// ApparentApprovalInput contains the complete criterion and degradation
	// state needed to determine whether an attack round is eligible.
	ApparentApprovalInput struct {
		ActiveCriteria []string
		Findings       []models.ReviewFinding
		Degradations   []models.DegradationEntry
	}

	// FindingStatusTransitionInput contains the deterministic gate availability
	// and finding state used to derive the next verification status.
	FindingStatusTransitionInput struct {
		GateAvailability GateAvailability
		Finding          models.ReviewFinding
	}
)

// IsHighImpactFinding reports whether a finding's severity and domains require
// independent confirmation. Criterion satisfaction is intentionally excluded.
func IsHighImpactFinding(finding *models.ReviewFinding) bool {
	if finding.FindingSeverity == models.SeverityCritical {
		return true
	}

	if finding.FindingSeverity != models.SeverityHigh {
		return false
	}

	return hasConfirmationDomain(finding.FindingDomains)
}

// IsGateValid reports whether a finding has a successful, correlated,
// deterministic schema and citation gate result.
func IsGateValid(finding *models.ReviewFinding) bool {
	if finding.Status == models.StatusUncheckedGateUnavail {
		return false
	}

	gateResult := finding.CitationGateResult
	if gateResult == nil {
		return false
	}

	return gateResult.SchemaValid && gateResult.CitationValid && gateResult.ProvenanceCorrelationID != ""
}

// IsConfirmerTarget reports whether a finding is eligible for Confirmer
// routing. Only gate-valid high-impact findings may be selected.
func IsConfirmerTarget(finding *models.ReviewFinding) bool {
	return IsGateValid(finding) && IsHighImpactFinding(finding)
}

// SelectConfirmerTargets returns the indexes of every finding eligible for
// Confirmer routing. It preserves the input ordering and does not mutate it.
func SelectConfirmerTargets(findings []models.ReviewFinding) []int {
	targets := make([]int, 0, len(findings))

	for index := range findings {
		if IsConfirmerTarget(&findings[index]) {
			targets = append(targets, index)
		}
	}

	return targets
}

// IsApparentApproval reports whether every active criterion has exactly one
// gate-valid finding with passing satisfaction and no required degradation.
func IsApparentApproval(input *ApparentApprovalInput) bool {
	if hasRequiredDegradation(input.Degradations) {
		return false
	}

	activeCriteria := make(map[string]struct{}, len(input.ActiveCriteria))
	for _, criterion := range input.ActiveCriteria {
		if criterion == "" {
			return false
		}

		if _, exists := activeCriteria[criterion]; exists {
			return false
		}

		activeCriteria[criterion] = struct{}{}
	}

	if len(activeCriteria) == 0 || len(input.Findings) != len(activeCriteria) {
		return false
	}

	seenCriteria := make(map[string]struct{}, len(input.Findings))
	for index := range input.Findings {
		finding := &input.Findings[index]
		if _, active := activeCriteria[finding.CriterionName]; !active {
			return false
		}

		if _, seen := seenCriteria[finding.CriterionName]; seen {
			return false
		}

		if finding.CriterionSatisfaction < PassingCriterionSatisfactionThreshold ||
			!IsGateValid(finding) {
			return false
		}

		seenCriteria[finding.CriterionName] = struct{}{}
	}

	return len(seenCriteria) == len(activeCriteria)
}

// TransitionFindingStatus applies deterministic gate and confirmation rules to
// a copy of a finding. It never confirms a finding without demonstration
// evidence from a successful Confirmer attempt.
func TransitionFindingStatus(input *FindingStatusTransitionInput) models.ReviewFinding {
	finding := input.Finding

	if input.GateAvailability != GateAvailable {
		finding.Status = models.StatusUncheckedGateUnavail

		return finding
	}

	if !IsGateValid(&finding) {
		finding.Status = models.StatusUnconfirmed

		return finding
	}

	if !IsConfirmerTarget(&finding) {
		finding.Status = models.StatusHypothesized

		return finding
	}

	demonstrationEvidence := demonstratedEvidence(&finding)
	if demonstrationEvidence != "" {
		finding.Status = models.StatusConfirmed
		finding.ConfirmerEvidence = demonstrationEvidence

		return finding
	}

	if hasThreeUnsuccessfulAttempts(&finding) {
		finding.Status = models.StatusUnconfirmed

		return finding
	}

	finding.Status = models.StatusHypothesized

	return finding
}

// hasConfirmationDomain reports whether a high-severity finding affects a
// domain that requires independent confirmation.
func hasConfirmationDomain(domains []models.FindingDomain) bool {
	for _, domain := range domains {
		if domain == models.DomainSecurity || domain == models.DomainCorrectness {
			return true
		}
	}

	return false
}

// hasRequiredDegradation reports whether a required workflow component failed.
func hasRequiredDegradation(entries []models.DegradationEntry) bool {
	for _, entry := range entries {
		if entry.Criticality == models.CriticalityRequired {
			return true
		}
	}

	return false
}

// demonstratedEvidence returns the first evidence value from a successful
// Confirmer attempt. Empty evidence cannot support confirmation.
func demonstratedEvidence(finding *models.ReviewFinding) string {
	for _, attempt := range finding.ConfirmerAttempts {
		if attempt.Demonstrated && attempt.DemonstrationEvidence != "" {
			return attempt.DemonstrationEvidence
		}
	}

	return ""
}

// hasThreeUnsuccessfulAttempts reports whether three or more completed
// Confirmer attempts ended without demonstrated evidence.
func hasThreeUnsuccessfulAttempts(finding *models.ReviewFinding) bool {
	if len(finding.ConfirmerAttempts) < MaxConfirmerAttempts {
		return false
	}

	return demonstratedEvidence(finding) == ""
}
