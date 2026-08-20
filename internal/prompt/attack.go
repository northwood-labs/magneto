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

// AttackInput contains the parameters for building the attack round prompt
// variation. It uses opaque prior-failure fingerprints instead of full prior
// finding text to maintain context isolation.
type AttackInput struct {
	ArtifactPath             string
	RubricContent            string
	AttackFocus              string
	AllowedPaths             []string
	PriorFailureFingerprints []FailureFingerprint
}

// BuildAttackContext constructs the prompt for the mandatory attack round. The
// attack round explicitly challenges the artifact from a different angle than
// prior rounds, focusing on failure modes that previous rounds may have missed.
// It renders only opaque fingerprints and excludes all prior finding reasoning
// and author context.
func BuildAttackContext(input *AttackInput) string {
	var b strings.Builder

	fmt.Fprint(&b, "## Attack Round Context\n\n")
	fmt.Fprint(&b, "Prior rounds concluded all criteria are satisfied. Challenge that conclusion.\n\n")

	fmt.Fprint(&b, "## Artifact Under Review\n\n")
	fmt.Fprintf(&b, "Path: %s\n\n", input.ArtifactPath)

	fmt.Fprint(&b, "## Review Rubric\n\n")
	fmt.Fprintf(&b, "%s\n\n", input.RubricContent)

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

	if input.AttackFocus != "" {
		fmt.Fprint(&b, "## Attack Focus\n\n")
		fmt.Fprintf(&b, "%s\n\n", input.AttackFocus)
	}

	if len(input.AllowedPaths) > 0 {
		fmt.Fprint(&b, "## Allowed Repository Paths\n\n")

		for _, p := range input.AllowedPaths {
			fmt.Fprintf(&b, "- %s\n", p)
		}

		fmt.Fprint(&b, "\n")
	}

	return b.String()
}
