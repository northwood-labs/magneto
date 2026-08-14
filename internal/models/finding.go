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
)

type (
	// FindingStatus represents the verification state of a review finding.
	FindingStatus string

	// ArtifactLocation identifies where in a reviewed artifact a finding's
	// evidence was found.
	ArtifactLocation struct {
		FilePath         string `json:"file_path"`         // lint:allow_format
		SectionReference string `json:"section_reference"` // lint:allow_format
	}

	// ReviewFinding represents a single criterion-level finding from the
	// adversarial reviewer.
	ReviewFinding struct {
		ArtifactLocation  ArtifactLocation `json:"artifact_location"`            // lint:allow_format
		CriterionName     string           `json:"criterion_name"`               // lint:allow_format
		QuotedExcerpt     string           `json:"quoted_excerpt"`               // lint:allow_format
		Status            FindingStatus    `json:"status"`                       // lint:allow_format
		Reasoning         string           `json:"reasoning"`                    // lint:allow_format
		ConfirmerEvidence string           `json:"confirmer_evidence,omitempty"` // lint:allow_format
		Score             int              `json:"score"`                        // lint:allow_format
	}
)
