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
	"fmt"
	"strings"

	"go.nwlabs.dev/magneto/internal/models"
)

const (
	minScore = 1
	maxScore = 10
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

// ValidateFindingSchema checks that a ReviewFinding has all required fields
// populated with valid values. Returns nil if valid, or a
// *SchemaValidationError listing each problem found.
func ValidateFindingSchema(finding *models.ReviewFinding) error {
	errs := &SchemaValidationError{}

	if finding.CriterionName == "" {
		errs.Errors = append(errs.Errors, FieldError{
			Field:   "criterion_name",
			Message: "criterion name is required",
		})
	}

	if finding.Score < minScore || finding.Score > maxScore {
		errs.Errors = append(errs.Errors, FieldError{
			Field:   "score",
			Message: fmt.Sprintf("score must be between %d and %d", minScore, maxScore),
		})
	}

	if finding.QuotedExcerpt == "" {
		errs.Errors = append(errs.Errors, FieldError{
			Field:   "quoted_excerpt",
			Message: "quoted excerpt is required",
		})
	}

	if finding.ArtifactLocation.FilePath == "" {
		errs.Errors = append(errs.Errors, FieldError{
			Field:   "artifact_location.file_path",
			Message: "artifact file path is required",
		})
	}

	if finding.ArtifactLocation.SectionReference == "" {
		errs.Errors = append(errs.Errors, FieldError{
			Field:   "artifact_location.section_reference",
			Message: "artifact section reference is required",
		})
	}

	if !isValidStatus(finding.Status) {
		errs.Errors = append(errs.Errors, FieldError{
			Field:   "status",
			Message: "status must be a valid FindingStatus value",
		})
	}

	if len(errs.Errors) > 0 {
		return errs
	}

	return nil
}

func isValidStatus(s models.FindingStatus) bool {
	switch s {
	case models.StatusConfirmed,
		models.StatusHypothesized,
		models.StatusUnconfirmed,
		models.StatusUnconfirmedInconclusive,
		models.StatusUncheckedGateUnavail:
		return true
	default:
		return false
	}
}
