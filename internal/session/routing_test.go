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

	"github.com/go-openapi/testify/assert"

	"go.nwlabs.dev/magneto/internal/models"
	"go.nwlabs.dev/magneto/internal/session"
)

func TestSelectConfirmerTargets(t *testing.T) {
	findings := []models.ReviewFinding{
		routingFinding(models.SeverityCritical, []models.FindingDomain{models.DomainArchitecture}),
		routingFinding(models.SeverityHigh, []models.FindingDomain{models.DomainSecurity}),
		routingFinding(models.SeverityHigh, []models.FindingDomain{models.DomainCorrectness}),
		routingFinding(models.SeverityHigh, []models.FindingDomain{models.DomainArchitecture}),
		routingFinding(models.SeverityMedium, []models.FindingDomain{models.DomainSecurity}),
		invalidRoutingFinding(models.SeverityCritical, []models.FindingDomain{models.DomainSecurity}),
	}

	targets := session.SelectConfirmerTargets(findings)

	assert.Equal(t, []int{0, 1, 2}, targets)
}

func TestTransitionFindingStatus(t *testing.T) {
	tests := []struct {
		name             string
		input            *session.FindingStatusTransitionInput
		expectedStatus   models.FindingStatus
		expectedEvidence string
	}{
		{
			name: "unavailable gate remains unchecked",
			input: &session.FindingStatusTransitionInput{
				GateAvailability: session.GateUnavailable,
				Finding: routingFinding(
					models.SeverityCritical,
					[]models.FindingDomain{models.DomainSecurity},
				),
			},
			expectedStatus: models.StatusUncheckedGateUnavail,
		},
		{
			name: "invalid gate becomes unconfirmed",
			input: &session.FindingStatusTransitionInput{
				GateAvailability: session.GateAvailable,
				Finding: invalidRoutingFinding(
					models.SeverityCritical,
					[]models.FindingDomain{models.DomainSecurity},
				),
			},
			expectedStatus: models.StatusUnconfirmed,
		},
		{
			name: "valid non-target remains hypothesized",
			input: &session.FindingStatusTransitionInput{
				GateAvailability: session.GateAvailable,
				Finding:          routingFinding(models.SeverityMedium, []models.FindingDomain{models.DomainSecurity}),
			},
			expectedStatus: models.StatusHypothesized,
		},
		{
			name: "demonstration evidence confirms high impact target",
			input: &session.FindingStatusTransitionInput{
				GateAvailability: session.GateAvailable,
				Finding: routingFindingWithAttempts([]models.ConfirmerAttempt{{
					AttemptNumber:         1,
					Demonstrated:          true,
					DemonstrationEvidence: "a reproducible exploit",
				}}),
			},
			expectedStatus:   models.StatusConfirmed,
			expectedEvidence: "a reproducible exploit",
		},
		{
			name: "fewer than three unsuccessful attempts remain hypothesized",
			input: &session.FindingStatusTransitionInput{
				GateAvailability: session.GateAvailable,
				Finding: routingFindingWithAttempts([]models.ConfirmerAttempt{
					{AttemptNumber: 1},
					{AttemptNumber: 2},
				}),
			},
			expectedStatus: models.StatusHypothesized,
		},
		{
			name: "three unsuccessful attempts become unconfirmed",
			input: &session.FindingStatusTransitionInput{
				GateAvailability: session.GateAvailable,
				Finding: routingFindingWithAttempts([]models.ConfirmerAttempt{
					{AttemptNumber: 1},
					{AttemptNumber: 2},
					{AttemptNumber: 3},
				}),
			},
			expectedStatus: models.StatusUnconfirmed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := session.TransitionFindingStatus(tt.input)

			assert.Equal(t, tt.expectedStatus, result.Status)
			assert.Equal(t, tt.expectedEvidence, result.ConfirmerEvidence)
		})
	}
}

func TestIsApparentApproval(t *testing.T) {
	validFinding := routingFinding(models.SeverityLow, []models.FindingDomain{models.DomainArchitecture})

	validFinding.CriterionSatisfaction = session.PassingCriterionSatisfactionThreshold

	tests := []struct {
		input    *session.ApparentApprovalInput
		name     string
		expected bool
	}{
		{
			name: "all active criteria are gate valid and passing",
			input: &session.ApparentApprovalInput{
				ActiveCriteria: []string{"criterion"},
				Findings:       []models.ReviewFinding{validFinding},
			},
			expected: true,
		},
		{
			name: "required degradation excludes approval",
			input: &session.ApparentApprovalInput{
				ActiveCriteria: []string{"criterion"},
				Findings:       []models.ReviewFinding{validFinding},
				Degradations: []models.DegradationEntry{{
					Criticality: models.CriticalityRequired,
				}},
			},
			expected: false,
		},
		{
			name: "optional degradation does not exclude approval",
			input: &session.ApparentApprovalInput{
				ActiveCriteria: []string{"criterion"},
				Findings:       []models.ReviewFinding{validFinding},
				Degradations: []models.DegradationEntry{{
					Criticality: models.CriticalityOptional,
				}},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, session.IsApparentApproval(tt.input))
		})
	}
}

func routingFinding(severity models.FindingSeverity, domains []models.FindingDomain) models.ReviewFinding {
	return models.ReviewFinding{
		CriterionName:         "criterion",
		CriterionSatisfaction: 4,
		FindingSeverity:       severity,
		FindingDomains:        domains,
		Status:                models.StatusHypothesized,
		CitationGateResult: &models.CitationGateResult{
			SchemaValid:             true,
			CitationValid:           true,
			ProvenanceCorrelationID: "gate-result",
		},
	}
}

func invalidRoutingFinding(severity models.FindingSeverity, domains []models.FindingDomain) models.ReviewFinding {
	finding := routingFinding(severity, domains)

	finding.CitationGateResult = nil

	return finding
}

func routingFindingWithAttempts(attempts []models.ConfirmerAttempt) models.ReviewFinding {
	finding := routingFinding(models.SeverityHigh, []models.FindingDomain{models.DomainSecurity})

	finding.ConfirmerAttempts = attempts

	return finding
}
