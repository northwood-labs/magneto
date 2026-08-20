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

package prompt

import (
	"fmt"
	"strings"
)

type (
	// FailureFingerprint is an opaque reference to a prior-round failure. It
	// contains only the criterion name, canonical satisfaction score, and a
	// normalized evidence hash. It intentionally excludes all reasoning,
	// intermediate outputs, and author context to maintain context isolation.
	FailureFingerprint struct {
		CriterionName         string
		EvidenceHash          string
		CriterionSatisfaction int
	}

	// ReviewerInput contains the parameters for building the Reviewer
	// subagent's environmental context.
	ReviewerInput struct {
		ArtifactPath             string
		RubricContent            string
		AllowedPaths             []string
		PriorFailureFingerprints []FailureFingerprint
		RoundNumber              int
	}
)

// BuildReviewerContext constructs the environmental_context for the Reviewer
// subagent. This contains ONLY the artifact location, rubric criteria, round
// metadata, allowed repository paths, and opaque prior-failure fingerprints. It
// explicitly excludes any Author session content, prior finding reasoning,
// intermediate outputs, and mutable capabilities to maintain context isolation.
func BuildReviewerContext(input *ReviewerInput) string {
	var b strings.Builder

	fmt.Fprint(&b, "## Artifact Under Review\n\n")
	fmt.Fprintf(&b, "Path: %s\n\n", input.ArtifactPath)

	fmt.Fprint(&b, "## Review Rubric\n\n")
	fmt.Fprintf(&b, "%s\n\n", input.RubricContent)

	fmt.Fprint(&b, "## Round Metadata\n\n")
	fmt.Fprintf(&b, "Round: %d of 5\n\n", input.RoundNumber)

	if len(input.AllowedPaths) > 0 {
		fmt.Fprint(&b, "## Allowed Repository Paths\n\n")

		for _, p := range input.AllowedPaths {
			fmt.Fprintf(&b, "- %s\n", p)
		}

		fmt.Fprint(&b, "\n")
	}

	if len(input.PriorFailureFingerprints) > 0 {
		fmt.Fprint(&b, "## Prior Failure Fingerprints\n\n")

		for _, fp := range input.PriorFailureFingerprints {
			fmt.Fprintf(
				&b,
				"- criterion=%s satisfaction=%d hash=%s\n",
				fp.CriterionName,
				fp.CriterionSatisfaction,
				fp.EvidenceHash,
			)
		}

		fmt.Fprint(&b, "\n")
	}

	return b.String()
}
