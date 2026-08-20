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
	// CitationFailureMissingEvidence identifies a finding without the evidence
	// required for deterministic citation validation.
	CitationFailureMissingEvidence = "missing required citation evidence"

	// CitationFailureResultUnavailable identifies a finding that did not receive
	// a deterministic citation result.
	CitationFailureResultUnavailable = "citation validation result unavailable"
)

type (
	// DowngradeInput contains the findings and deterministic citation outcomes
	// used to apply the citation gate before confirmation routing.
	DowngradeInput struct {
		Findings          []models.ReviewFinding
		ValidationResults []CitationValidationResult
	}

	// CitationValidationResult represents the deterministic schema and citation
	// outcome for one finding.
	CitationValidationResult struct {
		MatchedLines            *models.CitationMatchedLines
		FailureReason           string
		ProvenanceCorrelationID string
		GateAvailability        GateAvailability
		FindingIndex            int
		SchemaValid             bool
		CitationValid           bool
	}
)

// DowngradeUncitedFindings applies deterministic citation outcomes without
// mutating its input. Invalid or unavailable results cannot route to a
// Confirmer or satisfy approval, while a valid gate leaves findings
// hypothesized until confirmation handling runs.
func DowngradeUncitedFindings(input *DowngradeInput) []models.ReviewFinding {
	results := copyFindings(input.Findings)
	validationByIndex := validationResultsByFindingIndex(input.ValidationResults)

	for index := range results {
		validation, exists := validationByIndex[index]
		if !exists {
			setCitationFailure(&results[index], CitationFailureResultUnavailable)

			continue
		}

		applyCitationValidation(&results[index], validation)
	}

	return results
}

// validationResultsByFindingIndex returns the final reported validation result
// for each finding index.
func validationResultsByFindingIndex(results []CitationValidationResult) map[int]CitationValidationResult {
	byFindingIndex := make(map[int]CitationValidationResult, len(results))
	for _, result := range results {
		byFindingIndex[result.FindingIndex] = result
	}

	return byFindingIndex
}

// applyCitationValidation records one deterministic gate outcome and applies
// the status that outcome permits.
func applyCitationValidation(finding *models.ReviewFinding, validation CitationValidationResult) {
	if validation.GateAvailability == GateUnavailable {
		setGateUnavailable(finding, validation)

		return
	}

	finding.CitationGateResult = citationGateResult(&validation)
	if hasMissingCitation(finding) {
		setCitationFailure(finding, CitationFailureMissingEvidence)

		return
	}

	if !validation.SchemaValid || !validation.CitationValid {
		setCitationFailure(finding, validation.FailureReason)

		return
	}

	finding.Status = models.StatusHypothesized
}

// setGateUnavailable records that a required deterministic gate could not run.
func setGateUnavailable(finding *models.ReviewFinding, validation CitationValidationResult) {
	finding.CitationGateResult = citationGateResult(&validation)
	finding.CitationGateResult.SchemaValid = false
	finding.CitationGateResult.CitationValid = false
	finding.Status = models.StatusUncheckedGateUnavail
}

// setCitationFailure records a deterministic gate failure and prevents
// confirmation or approval based on the affected finding.
func setCitationFailure(finding *models.ReviewFinding, failureReason string) {
	if failureReason == "" {
		failureReason = CitationFailureMissingEvidence
	}

	if finding.CitationGateResult == nil {
		finding.CitationGateResult = &models.CitationGateResult{}
	}

	finding.CitationGateResult.SchemaValid = false
	finding.CitationGateResult.CitationValid = false
	finding.CitationGateResult.FailureReason = failureReason
	finding.Status = models.StatusUnconfirmed
}

// citationGateResult copies the evidence retained from a deterministic gate
// response so the caller's findings and response values remain immutable.
func citationGateResult(validation *CitationValidationResult) *models.CitationGateResult {
	return &models.CitationGateResult{
		MatchedLines:            copyMatchedLines(validation.MatchedLines),
		FailureReason:           validation.FailureReason,
		ProvenanceCorrelationID: validation.ProvenanceCorrelationID,
		SchemaValid:             validation.SchemaValid,
		CitationValid:           validation.CitationValid,
	}
}

// copyFindings creates an independent result slice, including citation gate
// values, while leaving the input findings intact.
func copyFindings(findings []models.ReviewFinding) []models.ReviewFinding {
	copied := make([]models.ReviewFinding, len(findings))
	copy(copied, findings)

	for index := range copied {
		if copied[index].CitationGateResult == nil {
			continue
		}

		gateResult := *copied[index].CitationGateResult

		gateResult.MatchedLines = copyMatchedLines(gateResult.MatchedLines)
		copied[index].CitationGateResult = &gateResult
	}

	return copied
}

// copyMatchedLines duplicates the optional line range retained as citation
// evidence.
func copyMatchedLines(lines *models.CitationMatchedLines) *models.CitationMatchedLines {
	if lines == nil {
		return nil
	}

	copied := *lines

	return &copied
}

// hasMissingCitation checks whether a finding lacks the required citation
// fields: a quoted excerpt, file path, or section reference.
func hasMissingCitation(finding *models.ReviewFinding) bool {
	return finding.QuotedExcerpt == "" ||
		finding.ArtifactLocation.FilePath == "" ||
		finding.ArtifactLocation.SectionReference == ""
}
