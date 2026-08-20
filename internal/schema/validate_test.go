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
	"encoding/json"
	"testing"

	"github.com/go-openapi/testify/assert"
	"github.com/go-openapi/testify/require"

	"go.nwlabs.dev/magneto/internal/models"
	"go.nwlabs.dev/magneto/internal/schema"
)

type validationCase struct {
	name          string
	mutate        func(*models.ReviewFinding)
	expectField   string
	expectMessage string
	expectScore   int
	expectErr     bool
}

// TestValidateFindingSchema exercises schema normalization and validation for
// canonical proposed findings.
func TestValidateFindingSchema(t *testing.T) {
	tests := []validationCase{
		{
			name:        "valid finding passes",
			expectScore: 7,
		},
		{
			name: "satisfaction below range is clamped",
			mutate: func(finding *models.ReviewFinding) {
				finding.CriterionSatisfaction = 0
			},
			expectScore: 1,
		},
		{
			name: "satisfaction above range is clamped",
			mutate: func(finding *models.ReviewFinding) {
				finding.CriterionSatisfaction = 11
			},
			expectScore: 10,
		},
		{
			name: "missing criterion name",
			mutate: func(finding *models.ReviewFinding) {
				finding.CriterionName = ""
			},
			expectErr:     true,
			expectField:   "criterion_name",
			expectMessage: "criterion name is required",
		},
		{
			name: "invalid severity",
			mutate: func(finding *models.ReviewFinding) {
				finding.FindingSeverity = models.FindingSeverity("invalid")
			},
			expectErr:     true,
			expectField:   "finding_severity",
			expectMessage: "finding severity must be critical, high, medium, or low",
		},
		{
			name: "missing finding domains",
			mutate: func(finding *models.ReviewFinding) {
				finding.FindingDomains = nil
			},
			expectErr:     true,
			expectField:   "finding_domains",
			expectMessage: "at least one finding domain is required",
		},
		{
			name: "duplicate finding domain",
			mutate: func(finding *models.ReviewFinding) {
				finding.FindingDomains = []models.FindingDomain{
					models.DomainSecurity,
					models.DomainSecurity,
				}
			},
			expectErr:     true,
			expectField:   "finding_domains",
			expectMessage: "finding domain \"security\" is duplicated",
		},
		{
			name: "empty quoted excerpt",
			mutate: func(finding *models.ReviewFinding) {
				finding.QuotedExcerpt = ""
			},
			expectErr:     true,
			expectField:   "quoted_excerpt",
			expectMessage: "quoted excerpt is required",
		},
		{
			name: "missing file path",
			mutate: func(finding *models.ReviewFinding) {
				finding.ArtifactLocation.FilePath = ""
			},
			expectErr:     true,
			expectField:   "artifact_location.file_path",
			expectMessage: "artifact file path is required",
		},
		{
			name: "missing section reference",
			mutate: func(finding *models.ReviewFinding) {
				finding.ArtifactLocation.SectionReference = ""
			},
			expectErr:     true,
			expectField:   "artifact_location.section_reference",
			expectMessage: "artifact section reference is required",
		},
		{
			name: "confirmed status is not accepted from a proposal",
			mutate: func(finding *models.ReviewFinding) {
				finding.Status = models.StatusConfirmed
			},
			expectErr:     true,
			expectField:   "status",
			expectMessage: "proposed status must be hypothesized",
		},
		{
			name: "request-provided gate result is rejected",
			mutate: func(finding *models.ReviewFinding) {
				finding.CitationGateResult = &models.CitationGateResult{}
			},
			expectErr:     true,
			expectField:   "citation_gate_result",
			expectMessage: "gate assertions are not accepted from requests",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			finding := validFinding()
			if tt.mutate != nil {
				tt.mutate(finding)
			}

			var validationErr *schema.SchemaValidationError

			err := schema.ValidateFindingSchema(finding)

			if !tt.expectErr {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectScore, finding.CriterionSatisfaction)

				return
			}

			assert.Error(t, err)
			assert.ErrorAs(t, err, &validationErr)

			if validationErr == nil {
				return
			}

			found := false

			for _, fieldErr := range validationErr.Errors {
				if fieldErr.Field == tt.expectField && fieldErr.Message == tt.expectMessage {
					found = true

					break
				}
			}

			assert.True(
				t,
				found,
				"expected FieldError{Field: %q, Message: %q} in validation errors",
				tt.expectField,
				tt.expectMessage,
			)
		})
	}
}

// TestDecodeAndNormalizeFinding verifies canonical migration and proposal
// validation at the JSON adapter boundary.
func TestDecodeAndNormalizeFinding(t *testing.T) {
	tests := []struct {
		name          string
		mutate        func(map[string]any)
		expectField   string
		expectMessage string
		expectScore   int
		expectErr     bool
	}{
		{
			name: "legacy score decodes when canonical field is absent",
			mutate: func(payload map[string]any) {
				delete(payload, "criterion_satisfaction")

				payload["score"] = 5
			},
			expectScore: 5,
		},
		{
			name: "canonical satisfaction takes precedence over legacy score",
			mutate: func(payload map[string]any) {
				payload["criterion_satisfaction"] = 6
				payload["score"] = 2
			},
			expectScore: 6,
		},
		{
			name: "satisfaction below lower bound is clamped",
			mutate: func(payload map[string]any) {
				payload["criterion_satisfaction"] = 0
			},
			expectScore: 1,
		},
		{
			name: "satisfaction above upper bound is clamped",
			mutate: func(payload map[string]any) {
				payload["criterion_satisfaction"] = 11
			},
			expectScore: 10,
		},
		{
			name: "missing satisfaction is rejected without a legacy score",
			mutate: func(payload map[string]any) {
				delete(payload, "criterion_satisfaction")
			},
			expectErr:     true,
			expectField:   "criterion_satisfaction",
			expectMessage: "criterion satisfaction is required",
		},
		{
			name: "invalid severity is rejected",
			mutate: func(payload map[string]any) {
				payload["finding_severity"] = "urgent"
			},
			expectErr:     true,
			expectField:   "finding_severity",
			expectMessage: "finding severity must be critical, high, medium, or low",
		},
		{
			name: "invalid domain is rejected",
			mutate: func(payload map[string]any) {
				payload["finding_domains"] = []string{"unsupported"}
			},
			expectErr:     true,
			expectField:   "finding_domains",
			expectMessage: "finding domain \"unsupported\" is not valid",
		},
		{
			name: "empty domain set is rejected",
			mutate: func(payload map[string]any) {
				payload["finding_domains"] = nil
			},
			expectErr:     true,
			expectField:   "finding_domains",
			expectMessage: "at least one finding domain is required",
		},
		{
			name: "duplicate domains are rejected",
			mutate: func(payload map[string]any) {
				payload["finding_domains"] = []string{"security", "security"}
			},
			expectErr:     true,
			expectField:   "finding_domains",
			expectMessage: "finding domain \"security\" is duplicated",
		},
		{
			name: "confirmed status is rejected from a proposal",
			mutate: func(payload map[string]any) {
				payload["status"] = models.StatusConfirmed
			},
			expectErr:     true,
			expectField:   "status",
			expectMessage: "proposed status must be hypothesized",
		},
		{
			name: "unconfirmed status is rejected from a proposal",
			mutate: func(payload map[string]any) {
				payload["status"] = models.StatusUnconfirmed
			},
			expectErr:     true,
			expectField:   "status",
			expectMessage: "proposed status must be hypothesized",
		},
		{
			name: "gate unavailable status is rejected from a proposal",
			mutate: func(payload map[string]any) {
				payload["status"] = models.StatusUncheckedGateUnavail
			},
			expectErr:     true,
			expectField:   "status",
			expectMessage: "proposed status must be hypothesized",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			payload := validFindingPayload()
			tt.mutate(payload)

			data, marshalErr := json.Marshal(payload)
			require.NoError(t, marshalErr)

			finding, err := schema.DecodeAndNormalizeFinding(data)

			if !tt.expectErr {
				require.NoError(t, err)
				assert.Equal(t, tt.expectScore, finding.CriterionSatisfaction)

				return
			}

			assertFieldError(t, err, tt.expectField, tt.expectMessage)
		})
	}
}

func assertFieldError(t *testing.T, err error, field, message string) {
	t.Helper()

	var validationErr *schema.SchemaValidationError

	require.Error(t, err)
	require.ErrorAs(t, err, &validationErr)

	for _, fieldErr := range validationErr.Errors {
		if fieldErr.Field == field && fieldErr.Message == message {
			return
		}
	}

	assert.Failf(t, "expected schema field error", "field=%q message=%q", field, message)
}

func validFindingPayload() map[string]any {
	return map[string]any{
		"artifact_location": map[string]string{
			"file_path":         "design.md",
			"section_reference": "Overview",
		},
		"criterion_name":         "Security Boundaries",
		"criterion_satisfaction": 7,
		"finding_domains":        []string{"security"},
		"finding_severity":       "high",
		"quoted_excerpt":         "the system enforces structurally independent review",
		"reasoning":              "criterion is satisfied with cited evidence",
		"status":                 "hypothesized",
	}
}

func validFinding() *models.ReviewFinding {
	return &models.ReviewFinding{
		CriterionName:         "Security Boundaries",
		CriterionSatisfaction: 7,
		QuotedExcerpt:         "the system enforces structurally independent review",
		ArtifactLocation: models.ArtifactLocation{
			FilePath:         "design.md",
			SectionReference: "Overview",
		},
		Status:          models.StatusHypothesized,
		Reasoning:       "criterion is satisfied with cited evidence",
		FindingSeverity: models.SeverityHigh,
		FindingDomains:  []models.FindingDomain{models.DomainSecurity},
	}
}
