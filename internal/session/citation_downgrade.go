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

type (
	// DowngradeInput contains the parameters for downgrading uncited findings.
	DowngradeInput struct {
		Findings          []models.ReviewFinding
		ValidationResults []CitationValidationResult
	}

	// CitationValidationResult represents the validation outcome for a single
	// finding's citation.
	CitationValidationResult struct {
		FailureReason string
		FindingIndex  int
		CitationValid bool
	}
)

// DowngradeUncitedFindings marks findings as "unconfirmed" when their citation
// validation fails. A finding is downgraded if:
//
//  1. It has no quoted excerpt (missing citation)
//  2. It has no artifact location (missing citation)
//  3. Its citation validation result indicates failure (verbatim match failed)
//
// Findings that already have valid citations keep their original status
// unchanged.
func DowngradeUncitedFindings(input *DowngradeInput) []models.ReviewFinding {
	results := make([]models.ReviewFinding, len(input.Findings))
	copy(results, input.Findings)

	// Build a lookup of validation results by finding index.
	validationByIndex := make(map[int]CitationValidationResult, len(input.ValidationResults))
	for _, vr := range input.ValidationResults {
		validationByIndex[vr.FindingIndex] = vr
	}

	for i := range results {
		if hasMissingCitation(&results[i]) {
			results[i].Status = models.StatusUnconfirmed

			continue
		}

		vr, exists := validationByIndex[i]
		if exists && !vr.CitationValid {
			results[i].Status = models.StatusUnconfirmed
		}
	}

	return results
}

// hasMissingCitation checks whether a finding lacks the required citation
// fields (empty quoted excerpt or empty artifact location).
func hasMissingCitation(f *models.ReviewFinding) bool {
	if f.QuotedExcerpt == "" {
		return true
	}

	if f.ArtifactLocation.FilePath == "" {
		return true
	}

	if f.ArtifactLocation.SectionReference == "" {
		return true
	}

	return false
}
