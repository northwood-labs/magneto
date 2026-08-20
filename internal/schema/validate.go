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

package schema

import (
	"encoding/json"
	"fmt"
	"strings"

	"go.nwlabs.dev/magneto/internal/models"
)

const (
	minCriterionSatisfaction = 1
	maxCriterionSatisfaction = 10
)

type (
	// FieldError represents a single field validation failure.
	FieldError struct {
		Field   string
		Message string
	}

	// SchemaValidationError collects multiple validation failures.
	SchemaValidationError struct {
		Errors []FieldError
	}

	// findingInput represents an untrusted candidate finding. Pointer score
	// fields preserve the distinction between an omitted canonical value and a
	// submitted zero value while decoding JSON.
	findingInput struct {
		CriterionSatisfaction *int                    `json:"criterion_satisfaction"` // lint:allow_format
		Score                 *int                    `json:"score,omitempty"`
		ArtifactLocation      models.ArtifactLocation `json:"artifact_location"` // lint:allow_format
		CriterionName         string                  `json:"criterion_name"`    // lint:allow_format
		QuotedExcerpt         string                  `json:"quoted_excerpt"`    // lint:allow_format
		Status                models.FindingStatus    `json:"status"`
		Reasoning             string                  `json:"reasoning"`
		FindingSeverity       models.FindingSeverity  `json:"finding_severity"` // lint:allow_format
		FindingDomains        []models.FindingDomain  `json:"finding_domains"`  // lint:allow_format
	}
)

// Error implements the error interface for SchemaValidationError.
func (ve *SchemaValidationError) Error() string {
	var b strings.Builder

	fmt.Fprintf(&b, "validation failed with %d error(s):", len(ve.Errors))

	for _, e := range ve.Errors {
		fmt.Fprintf(&b, "\n  - %s: %s", e.Field, e.Message)
	}

	return b.String()
}

// DecodeAndNormalizeFinding decodes an untrusted finding, applies the legacy
// score migration when needed, and returns its canonical representation.
func DecodeAndNormalizeFinding(data []byte) (*models.ReviewFinding, error) {
	rawFields := make(map[string]json.RawMessage)

	decodeRawErr := json.Unmarshal(data, &rawFields)
	if decodeRawErr != nil {
		return nil, &SchemaValidationError{Errors: []FieldError{{
			Field:   "finding",
			Message: "finding must be a JSON object",
		}}}
	}

	input := &findingInput{}

	decodeInputErr := json.Unmarshal(data, input)
	if decodeInputErr != nil {
		return nil, &SchemaValidationError{Errors: []FieldError{{
			Field:   "finding",
			Message: "finding must contain fields with valid JSON types",
		}}}
	}

	normalized, normalizeErr := normalizeFinding(input, proposedAssertionFields(rawFields))
	if normalizeErr != nil {
		return normalized, normalizeErr
	}

	return normalized, nil
}

// ValidateFindingSchema normalizes an in-memory ReviewFinding and checks that
// it has the required proposed-finding fields. It mutates finding only when
// normalization succeeds.
func ValidateFindingSchema(finding *models.ReviewFinding) error {
	if finding == nil {
		return &SchemaValidationError{Errors: []FieldError{{
			Field:   "finding",
			Message: "finding is required",
		}}}
	}

	criterionSatisfaction := finding.CriterionSatisfaction
	input := &findingInput{
		ArtifactLocation:      finding.ArtifactLocation,
		CriterionName:         finding.CriterionName,
		QuotedExcerpt:         finding.QuotedExcerpt,
		Status:                finding.Status,
		Reasoning:             finding.Reasoning,
		FindingSeverity:       finding.FindingSeverity,
		FindingDomains:        finding.FindingDomains,
		CriterionSatisfaction: &criterionSatisfaction,
	}

	normalized, normalizeErr := normalizeFinding(input, findingAssertionFields(finding))
	if normalizeErr != nil {
		return normalizeErr
	}

	*finding = *normalized

	return nil
}

func normalizeFinding(input *findingInput, assertions []FieldError) (*models.ReviewFinding, *SchemaValidationError) {
	errs := &SchemaValidationError{Errors: assertions}

	criterionSatisfaction, satisfactionErr := normalizedSatisfaction(input)
	if satisfactionErr != nil {
		errs.Errors = append(errs.Errors, *satisfactionErr)
	}

	if strings.TrimSpace(input.CriterionName) == "" {
		errs.Errors = append(errs.Errors, FieldError{
			Field:   "criterion_name",
			Message: "criterion name is required",
		})
	}

	if !isValidSeverity(input.FindingSeverity) {
		errs.Errors = append(errs.Errors, FieldError{
			Field:   "finding_severity",
			Message: "finding severity must be critical, high, medium, or low",
		})
	}

	domains, domainErrors := normalizedDomains(input.FindingDomains)

	errs.Errors = append(errs.Errors, domainErrors...)

	if strings.TrimSpace(input.QuotedExcerpt) == "" {
		errs.Errors = append(errs.Errors, FieldError{
			Field:   "quoted_excerpt",
			Message: "quoted excerpt is required",
		})
	}

	if strings.TrimSpace(input.ArtifactLocation.FilePath) == "" {
		errs.Errors = append(errs.Errors, FieldError{
			Field:   "artifact_location.file_path",
			Message: "artifact file path is required",
		})
	}

	if strings.TrimSpace(input.ArtifactLocation.SectionReference) == "" {
		errs.Errors = append(errs.Errors, FieldError{
			Field:   "artifact_location.section_reference",
			Message: "artifact section reference is required",
		})
	}

	if input.Status == "" {
		errs.Errors = append(errs.Errors, FieldError{
			Field:   "status",
			Message: "status is required",
		})
	} else if input.Status != models.StatusHypothesized {
		errs.Errors = append(errs.Errors, FieldError{
			Field:   "status",
			Message: "proposed status must be hypothesized",
		})
	}

	if strings.TrimSpace(input.Reasoning) == "" {
		errs.Errors = append(errs.Errors, FieldError{
			Field:   "reasoning",
			Message: "reasoning is required",
		})
	}

	finding := &models.ReviewFinding{
		ArtifactLocation:      input.ArtifactLocation,
		CriterionName:         input.CriterionName,
		QuotedExcerpt:         input.QuotedExcerpt,
		Status:                input.Status,
		Reasoning:             input.Reasoning,
		FindingSeverity:       input.FindingSeverity,
		FindingDomains:        domains,
		CriterionSatisfaction: criterionSatisfaction,
	}

	if len(errs.Errors) > 0 {
		return finding, errs
	}

	return finding, nil
}

func normalizedSatisfaction(input *findingInput) (int, *FieldError) {
	if input.CriterionSatisfaction == nil && input.Score == nil {
		return 0, &FieldError{
			Field:   "criterion_satisfaction",
			Message: "criterion satisfaction is required",
		}
	}

	value := input.Score
	if input.CriterionSatisfaction != nil {
		value = input.CriterionSatisfaction
	}

	return clampSatisfaction(*value), nil
}

func clampSatisfaction(value int) int {
	if value < minCriterionSatisfaction {
		return minCriterionSatisfaction
	}

	if value > maxCriterionSatisfaction {
		return maxCriterionSatisfaction
	}

	return value
}

func normalizedDomains(domains []models.FindingDomain) ([]models.FindingDomain, []FieldError) {
	if len(domains) == 0 {
		return nil, []FieldError{{
			Field:   "finding_domains",
			Message: "at least one finding domain is required",
		}}
	}

	var errs []FieldError

	seen := make(map[models.FindingDomain]struct{}, len(domains))

	for _, domain := range domains {
		if !isValidDomain(domain) {
			errs = append(errs, FieldError{
				Field:   "finding_domains",
				Message: fmt.Sprintf("finding domain %q is not valid", domain),
			})

			continue
		}

		if _, exists := seen[domain]; exists {
			errs = append(errs, FieldError{
				Field:   "finding_domains",
				Message: fmt.Sprintf("finding domain %q is duplicated", domain),
			})

			continue
		}

		seen[domain] = struct{}{}
	}

	if len(errs) > 0 {
		return nil, errs
	}

	return models.CanonicalFindingDomains(domains), nil
}

func proposedAssertionFields(rawFields map[string]json.RawMessage) []FieldError {
	var assertions []FieldError

	for _, field := range []string{
		"citation_gate_result",
		"citation_valid",
		"schema_valid",
		"provenance_correlation_id",
		"confirmer_evidence",
		"confirmer_attempts",
	} {
		if _, exists := rawFields[field]; !exists {
			continue
		}

		assertions = append(assertions, FieldError{
			Field:   field,
			Message: assertionMessage(field),
		})
	}

	return assertions
}

func findingAssertionFields(finding *models.ReviewFinding) []FieldError {
	var assertions []FieldError

	if finding.CitationGateResult != nil {
		assertions = append(assertions, FieldError{
			Field:   "citation_gate_result",
			Message: assertionMessage("citation_gate_result"),
		})
	}

	if finding.ConfirmerEvidence != "" {
		assertions = append(assertions, FieldError{
			Field:   "confirmer_evidence",
			Message: assertionMessage("confirmer_evidence"),
		})
	}

	if len(finding.ConfirmerAttempts) > 0 {
		assertions = append(assertions, FieldError{
			Field:   "confirmer_attempts",
			Message: assertionMessage("confirmer_attempts"),
		})
	}

	return assertions
}

func assertionMessage(field string) string {
	switch field {
	case "citation_gate_result", "citation_valid", "schema_valid":
		return "gate assertions are not accepted from requests"
	case "provenance_correlation_id":
		return "provenance assertions are not accepted from requests"
	default:
		return "confirmation assertions are not accepted from requests"
	}
}

func isValidSeverity(severity models.FindingSeverity) bool {
	switch severity {
	case models.SeverityCritical, models.SeverityHigh, models.SeverityMedium, models.SeverityLow:
		return true
	default:
		return false
	}
}

func isValidDomain(domain models.FindingDomain) bool {
	switch domain {
	case models.DomainSecurity,
		models.DomainCorrectness,
		models.DomainArchitecture,
		models.DomainReliability,
		models.DomainOperations,
		models.DomainDeveloperExperience:
		return true
	default:
		return false
	}
}
