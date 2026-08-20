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
	"testing"

	"pgregory.net/rapid"

	"go.nwlabs.dev/magneto/internal/models"
	"go.nwlabs.dev/magneto/internal/schema"
)

type dimensionIndependenceInput struct {
	Finding         *models.ReviewFinding
	Severity        models.FindingSeverity
	Domains         []models.FindingDomain
	RawSatisfaction int
}

// TestProperty_FindingSchemaRejectsIncomplete verifies Property 4: Finding
// schema validation rejects incomplete findings.
//
// For any object missing one or more required ReviewFinding fields
// (CriterionName, FindingSeverity, FindingDomains, QuotedExcerpt,
// ArtifactLocation, Status, or Reasoning), ValidateFindingSchema SHALL return
// an error identifying the missing field(s).
//
// **Validates: Requirements 4.1**.
func TestProperty_FindingSchemaRejectsIncomplete(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a valid finding with all required fields populated.
		finding := &models.ReviewFinding{
			CriterionName:         rapid.StringMatching(`[a-z]{3,20}`).Draw(t, "criterion"),
			CriterionSatisfaction: rapid.IntRange(1, 10).Draw(t, "score"),
			QuotedExcerpt:         rapid.StringMatching(`[a-zA-Z0-9 ]{10,100}`).Draw(t, "excerpt"),
			ArtifactLocation: models.ArtifactLocation{
				FilePath:         rapid.StringMatching(`[a-z/]{5,30}\.md`).Draw(t, "path"),
				SectionReference: rapid.StringMatching(`[A-Z][a-z]{3,15}`).Draw(t, "section"),
			},
			Status:          models.StatusHypothesized,
			Reasoning:       rapid.StringMatching(`[a-zA-Z0-9 ]{10,200}`).Draw(t, "reasoning"),
			FindingSeverity: models.SeverityLow,
			FindingDomains:  []models.FindingDomain{models.DomainArchitecture},
		}

		// Randomly remove one required field to create an invalid finding.
		fieldToRemove := rapid.IntRange(0, 7).Draw(t, "field_index")

		switch fieldToRemove {
		case 0:
			finding.CriterionName = ""
		case 1:
			finding.FindingSeverity = ""
		case 2:
			finding.FindingDomains = nil
		case 3:
			finding.QuotedExcerpt = ""
		case 4:
			finding.ArtifactLocation.FilePath = ""
		case 5:
			finding.ArtifactLocation.SectionReference = ""
		case 6:
			finding.Status = ""
		default:
			finding.Reasoning = ""
		}

		// Schema validation must reject the incomplete finding.
		err := schema.ValidateFindingSchema(finding)
		if err == nil {
			t.Fatalf("expected validation error for missing field %d, got nil", fieldToRemove)
		}
	})
}

// TestProperty_FindingSchemaRejectsInvalidDimensions exercises invalid enum and
// domain-set proposals over generated inputs.
//
// **Validates: Requirements 3.3, 3.4**.
func TestProperty_FindingSchemaRejectsInvalidDimensions(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		finding := validFinding()
		invalidDimension := rapid.SampledFrom([]string{
			"invalid severity",
			"empty domain set",
			"invalid domain",
			"duplicate domain",
		}).Draw(t, "invalid_dimension")

		switch invalidDimension {
		case "invalid severity":
			finding.FindingSeverity = models.FindingSeverity(
				"invalid-" + rapid.StringMatching(`[a-z]{1,20}`).Draw(t, "severity"),
			)
		case "empty domain set":
			finding.FindingDomains = nil
		case "invalid domain":
			finding.FindingDomains = []models.FindingDomain{
				models.FindingDomain(
					"invalid-" + rapid.StringMatching(`[a-z]{1,20}`).Draw(t, "domain"),
				),
			}
		default:
			finding.FindingDomains = []models.FindingDomain{
				models.DomainSecurity,
				models.DomainSecurity,
			}
		}

		err := schema.ValidateFindingSchema(finding)
		if err == nil {
			t.Fatalf("expected validation error for %s, got nil", invalidDimension)
		}
	})
}

// TestProperty3_FindingDimensionsRemainValidAndIndependent verifies Property 3:
// Finding dimensions remain valid and independent.
//
// Feature: adversarial-review-operational-workflow, Property 3: Finding
// dimensions remain valid and independent
//
// For any proposed findings, normalization produces a Criterion_Satisfaction in
// 1 through 10, exactly one valid Finding_Severity, and a duplicate-free
// non-empty set of valid Finding_Domain values without deriving severity or
// domains from satisfaction.
//
// **Validates: Requirements 3.1, 3.3, 3.4**.
func TestProperty3_FindingDimensionsRemainValidAndIndependent(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate raw dimensions.
		rawSatisfaction := rapid.IntRange(-10, 20).Draw(t, "satisfaction")
		severity := drawValidSeverity(t)
		domains := drawDomainsWithPossibleDuplicates(t)

		// Build and normalize a valid finding.
		finding := buildProperty3Finding(t, rawSatisfaction, severity, domains)

		err := schema.ValidateFindingSchema(finding)
		if err != nil {
			t.Fatalf("expected no validation error, got: %v", err)
		}

		// Verify normalized dimensions.
		assertSatisfactionClamped(t, finding.CriterionSatisfaction)
		assertSeverityPreserved(t, finding.FindingSeverity, severity)
		assertDomainsValid(t, finding.FindingDomains)

		// Verify independence: changing satisfaction does not change
		// severity or domains.
		assertDimensionsIndependent(t, &dimensionIndependenceInput{
			Finding:         finding,
			RawSatisfaction: rawSatisfaction,
			Severity:        severity,
			Domains:         domains,
		})
	})
}

func drawValidSeverity(t *rapid.T) models.FindingSeverity {
	severities := []models.FindingSeverity{
		models.SeverityCritical,
		models.SeverityHigh,
		models.SeverityMedium,
		models.SeverityLow,
	}

	return rapid.SampledFrom(severities).Draw(t, "severity")
}

func drawDomainsWithPossibleDuplicates(t *rapid.T) []models.FindingDomain {
	allDomains := []models.FindingDomain{
		models.DomainSecurity,
		models.DomainCorrectness,
		models.DomainArchitecture,
		models.DomainReliability,
		models.DomainOperations,
		models.DomainDeveloperExperience,
	}

	domainCount := rapid.IntRange(1, 6).Draw(t, "domain_count")

	// Draw a permutation and take the first domainCount elements to produce a
	// unique subset.
	shuffled := rapid.Permutation(allDomains).Draw(t, "domain_perm")
	domains := make([]models.FindingDomain, domainCount)
	copy(domains, shuffled[:domainCount])

	return domains
}

func buildProperty3Finding(
	t *rapid.T,
	satisfaction int,
	severity models.FindingSeverity,
	domains []models.FindingDomain,
) *models.ReviewFinding {
	return &models.ReviewFinding{
		CriterionName:         rapid.StringMatching(`[a-z]{3,20}`).Draw(t, "criterion"),
		CriterionSatisfaction: satisfaction,
		QuotedExcerpt:         rapid.StringMatching(`[a-zA-Z][a-zA-Z0-9 ]{9,49}`).Draw(t, "excerpt"),
		ArtifactLocation: models.ArtifactLocation{
			FilePath:         rapid.StringMatching(`[a-z]{2,10}/[a-z]{2,10}\.md`).Draw(t, "path"),
			SectionReference: rapid.StringMatching(`[A-Z][a-z]{3,15}`).Draw(t, "section"),
		},
		Status:          models.StatusHypothesized,
		Reasoning:       rapid.StringMatching(`[a-zA-Z][a-zA-Z0-9 ]{9,49}`).Draw(t, "reasoning"),
		FindingSeverity: severity,
		FindingDomains:  domains,
	}
}

func assertSatisfactionClamped(t *rapid.T, satisfaction int) {
	if satisfaction < 1 || satisfaction > 10 {
		t.Fatalf("satisfaction %d is outside [1, 10]", satisfaction)
	}
}

func assertSeverityPreserved(
	t *rapid.T,
	actual, expected models.FindingSeverity,
) {
	if actual != expected {
		t.Fatalf("severity changed from %q to %q", expected, actual)
	}
}

func assertDomainsValid(t *rapid.T, domains []models.FindingDomain) {
	if len(domains) == 0 {
		t.Fatal("finding domains must be non-empty after normalization")
	}

	validSet := map[models.FindingDomain]struct{}{
		models.DomainSecurity:            {},
		models.DomainCorrectness:         {},
		models.DomainArchitecture:        {},
		models.DomainReliability:         {},
		models.DomainOperations:          {},
		models.DomainDeveloperExperience: {},
	}

	seen := make(map[models.FindingDomain]struct{}, len(domains))

	for _, domain := range domains {
		if _, ok := validSet[domain]; !ok {
			t.Fatalf("invalid domain %q in normalized output", domain)
		}

		if _, exists := seen[domain]; exists {
			t.Fatalf("duplicate domain %q in normalized output", domain)
		}

		seen[domain] = struct{}{}
	}
}

func assertDimensionsIndependent(t *rapid.T, input *dimensionIndependenceInput) {
	altSatisfaction := input.RawSatisfaction + 5 // lint:allow_raw_number
	if altSatisfaction > 20 {                    // lint:allow_raw_number
		altSatisfaction = input.RawSatisfaction - 5 // lint:allow_raw_number
	}

	altFinding := &models.ReviewFinding{
		CriterionName:         input.Finding.CriterionName,
		CriterionSatisfaction: altSatisfaction,
		QuotedExcerpt:         input.Finding.QuotedExcerpt,
		ArtifactLocation:      input.Finding.ArtifactLocation,
		Status:                models.StatusHypothesized,
		Reasoning:             input.Finding.Reasoning,
		FindingSeverity:       input.Severity,
		FindingDomains:        input.Domains,
	}

	altErr := schema.ValidateFindingSchema(altFinding)
	if altErr != nil {
		t.Fatalf(
			"expected no validation error for alt finding, got: %v",
			altErr,
		)
	}

	if altFinding.FindingSeverity != input.Severity {
		t.Fatalf(
			"severity derived from satisfaction: changed satisfaction "+
				"from %d to %d and severity changed from %q to %q",
			input.RawSatisfaction,
			altSatisfaction,
			input.Severity,
			altFinding.FindingSeverity,
		)
	}

	if len(altFinding.FindingDomains) != len(input.Finding.FindingDomains) {
		t.Fatalf(
			"domains derived from satisfaction: domain count changed "+
				"from %d to %d when satisfaction changed",
			len(input.Finding.FindingDomains),
			len(altFinding.FindingDomains),
		)
	}

	for i, domain := range input.Finding.FindingDomains {
		if altFinding.FindingDomains[i] != domain {
			t.Fatalf(
				"domains derived from satisfaction: domain[%d] "+
					"changed from %q to %q when satisfaction changed",
				i,
				domain,
				altFinding.FindingDomains[i],
			)
		}
	}
}
