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

// ReviewerInput contains the parameters for building the Reviewer subagent's
// environmental context.
type ReviewerInput struct {
	ArtifactPath  string
	RubricContent string
	PriorFindings string
	RoundNumber   int
}

// BuildReviewerContext constructs the environmental_context for the Reviewer
// subagent. This contains ONLY the artifact location, rubric criteria, and
// round metadata. It explicitly excludes any Author session content to maintain
// context isolation.
func BuildReviewerContext(input *ReviewerInput) string {
	var b strings.Builder

	fmt.Fprint(&b, "## Artifact Under Review\n\n")
	fmt.Fprintf(&b, "Path: %s\n\n", input.ArtifactPath)

	fmt.Fprint(&b, "## Review Rubric\n\n")
	fmt.Fprintf(&b, "%s\n\n", input.RubricContent)

	fmt.Fprint(&b, "## Round Metadata\n\n")
	fmt.Fprintf(&b, "Round: %d of 5\n\n", input.RoundNumber)

	if input.PriorFindings != "" {
		fmt.Fprint(&b, "## Prior Round Findings\n\n")
		fmt.Fprintf(&b, "%s\n", input.PriorFindings)
	}

	return b.String()
}
