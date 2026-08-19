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

package models // lint:allow_bad_package_name

const (
	// StatusConfirmed indicates the finding was verified by the confirmer
	// agent.
	StatusConfirmed FindingStatus = "confirmed"

	// StatusHypothesized indicates the finding was proposed by the reviewer but
	// not yet verified.
	StatusHypothesized FindingStatus = "hypothesized"

	// StatusUnconfirmed indicates the confirmer checked the finding and could
	// not verify it.
	StatusUnconfirmed FindingStatus = "unconfirmed"

	// StatusUnconfirmedInconclusive indicates the confirmer checked the finding
	// but reached no definitive conclusion.
	StatusUnconfirmedInconclusive FindingStatus = "unconfirmed (inconclusive)"

	// StatusUncheckedGateUnavail indicates the finding was not checked because
	// the confirmation gate was unavailable.
	StatusUncheckedGateUnavail FindingStatus = "unchecked (gate unavailable)"

	// SeverityCritical identifies a finding with the highest impact.
	SeverityCritical FindingSeverity = "critical"

	// SeverityHigh identifies a finding with high impact.
	SeverityHigh FindingSeverity = "high"

	// SeverityMedium identifies a finding with medium impact.
	SeverityMedium FindingSeverity = "medium"

	// SeverityLow identifies a finding with low impact.
	SeverityLow FindingSeverity = "low"

	// DomainSecurity identifies a finding in the security domain.
	DomainSecurity FindingDomain = "security"

	// DomainCorrectness identifies a finding in the correctness domain.
	DomainCorrectness FindingDomain = "correctness"

	// DomainArchitecture identifies a finding in the architecture domain.
	DomainArchitecture FindingDomain = "architecture"

	// DomainReliability identifies a finding in the reliability domain.
	DomainReliability FindingDomain = "reliability"

	// DomainOperations identifies a finding in the operations domain.
	DomainOperations FindingDomain = "operations"

	// DomainDeveloperExperience identifies a finding in the
	// developer-experience domain.
	DomainDeveloperExperience FindingDomain = "developer-experience"
)

var findingDomainOrder = []FindingDomain{
	DomainSecurity,
	DomainCorrectness,
	DomainArchitecture,
	DomainReliability,
	DomainOperations,
	DomainDeveloperExperience,
}

type (
	// FindingStatus represents the verification state of a review finding.
	FindingStatus string

	// FindingSeverity represents the impact level of a review finding.
	FindingSeverity string

	// FindingDomain represents a category affected by a review finding.
	FindingDomain string

	// ArtifactLocation identifies where in a reviewed artifact a finding's
	// evidence was found.
	ArtifactLocation struct {
		FilePath         string `json:"file_path"`         // lint:allow_format
		SectionReference string `json:"section_reference"` // lint:allow_format
	}

	// CitationMatchedLines identifies the line range where a citation matched.
	CitationMatchedLines struct {
		Start int `json:"start"`
		End   int `json:"end"`
	}

	// CitationGateResult records deterministic schema and citation outcomes for
	// a finding and binds them to a Magneto validation invocation.
	CitationGateResult struct {
		MatchedLines            *CitationMatchedLines `json:"matched_lines,omitempty"`             // lint:allow_format
		FailureReason           string                `json:"failure_reason,omitempty"`            // lint:allow_format
		ProvenanceCorrelationID string                `json:"provenance_correlation_id,omitempty"` // lint:allow_format
		SchemaValid             bool                  `json:"schema_valid"`                        // lint:allow_format
		CitationValid           bool                  `json:"citation_valid"`                      // lint:allow_format
	}

	// ConfirmerAttempt records one independent verification attempt for a
	// high-impact finding.
	ConfirmerAttempt struct {
		Strategy              string `json:"strategy"`
		Observation           string `json:"observation"`
		DemonstrationEvidence string `json:"demonstration_evidence,omitempty"` // lint:allow_format
		AttemptNumber         int    `json:"attempt_number"`                   // lint:allow_format
		Demonstrated          bool   `json:"demonstrated"`
	}

	// ReviewFinding represents a single criterion-level finding from the
	// adversarial reviewer.
	ReviewFinding struct {
		CitationGateResult    *CitationGateResult `json:"citation_gate_result,omitempty"` // lint:allow_format
		ArtifactLocation      ArtifactLocation    `json:"artifact_location"`              // lint:allow_format
		CriterionName         string              `json:"criterion_name"`                 // lint:allow_format
		QuotedExcerpt         string              `json:"quoted_excerpt"`                 // lint:allow_format
		Status                FindingStatus       `json:"status"`
		Reasoning             string              `json:"reasoning"`
		ConfirmerEvidence     string              `json:"confirmer_evidence,omitempty"` // lint:allow_format
		FindingSeverity       FindingSeverity     `json:"finding_severity"`             // lint:allow_format
		ConfirmerAttempts     []ConfirmerAttempt  `json:"confirmer_attempts,omitempty"` // lint:allow_format
		FindingDomains        []FindingDomain     `json:"finding_domains"`              // lint:allow_format
		CriterionSatisfaction int                 `json:"criterion_satisfaction"`       // lint:allow_format
	}
)

// CanonicalFindingDomains returns the recognized domains once each in their
// stable enum order. It does not mutate the input slice.
func CanonicalFindingDomains(domains []FindingDomain) []FindingDomain {
	present := make(map[FindingDomain]struct{}, len(domains))
	for _, domain := range domains {
		present[domain] = struct{}{}
	}

	canonical := make([]FindingDomain, 0, len(present))
	for _, domain := range findingDomainOrder {
		if _, exists := present[domain]; exists {
			canonical = append(canonical, domain)
		}
	}

	return canonical
}
