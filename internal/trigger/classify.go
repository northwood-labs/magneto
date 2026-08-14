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

package trigger

import (
	"slices"
	"strings"
)

const (
	// DecisionTrigger indicates the artifact should trigger adversarial review.
	DecisionTrigger Decision = "trigger"

	// DecisionSkip indicates the artifact should skip adversarial review.
	DecisionSkip Decision = "skip"

	// ReasonBlastRadius indicates the artifact domain matched the blast-radius
	// domain list.
	ReasonBlastRadius = "artifact domain matches blast-radius domain list"

	// ReasonFoundational indicates the artifact is consumed by downstream
	// automation without independent verification.
	ReasonFoundational = "artifact is foundational (consumed by downstream automation)"

	// ReasonSkipConditions indicates all skip conditions were met: single file,
	// revertible, and human-reviewed before consumption.
	ReasonSkipConditions = "single file, revertible, and human-reviewed before consumption"

	// ReasonAmbiguous indicates the classification was ambiguous and defaulted
	// to triggering review.
	ReasonAmbiguous = "ambiguous classification — defaulting to trigger"
)

// DefaultBlastRadiusDomains defines the default set of domains that trigger
// mandatory adversarial review when detected in an artifact.
var DefaultBlastRadiusDomains = []string{
	"auth",
	"secrets",
	"payments",
	"data-integrity",
	"irreversible-actions",
}

type (
	// Decision represents the review trigger decision.
	Decision string

	// ArtifactInfo describes the artifact being evaluated for review
	// triggering.
	ArtifactInfo struct {
		Domain                string
		IsFoundational        bool
		IsSingleFile          bool
		IsRevertible          bool
		IsHumanReviewedBefore bool
	}

	// ClassifyInput contains the parameters for trigger classification.
	ClassifyInput struct {
		Artifact           *ArtifactInfo
		BlastRadiusDomains []string
	}

	// ClassifyResult contains the trigger classification outcome.
	ClassifyResult struct {
		Decision  Decision
		Reason    string
		Ambiguous bool
	}
)

// Classify determines whether an artifact should trigger adversarial review
// based on blast-radius domains, foundational trust, and skip conditions.
// Ambiguous cases default to triggering review.
func Classify(input *ClassifyInput) *ClassifyResult {
	domains := input.BlastRadiusDomains
	if len(domains) == 0 {
		domains = DefaultBlastRadiusDomains
	}

	artifact := input.Artifact

	if domainMatchesList(artifact.Domain, domains) {
		return &ClassifyResult{
			Decision: DecisionTrigger,
			Reason:   ReasonBlastRadius,
		}
	}

	if artifact.IsFoundational {
		return &ClassifyResult{
			Decision: DecisionTrigger,
			Reason:   ReasonFoundational,
		}
	}

	if artifact.IsSingleFile && artifact.IsRevertible && artifact.IsHumanReviewedBefore {
		return &ClassifyResult{
			Decision: DecisionSkip,
			Reason:   ReasonSkipConditions,
		}
	}

	return &ClassifyResult{
		Decision:  DecisionTrigger,
		Reason:    ReasonAmbiguous,
		Ambiguous: true,
	}
}

// domainMatchesList performs case-insensitive matching of a domain against the
// blast-radius domain list.
func domainMatchesList(domain string, domains []string) bool {
	if domain == "" {
		return false
	}

	lower := strings.ToLower(domain)

	return slices.ContainsFunc(domains, func(d string) bool {
		return strings.EqualFold(d, lower)
	})
}
