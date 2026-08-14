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

// ConfirmerInput contains the parameters for building the Confirmer subagent's
// context.
type ConfirmerInput struct {
	CriterionName    string
	ClaimText        string
	ArtifactPath     string
	SectionReference string
	QuotedExcerpt    string
}

// BuildConfirmerContext constructs the context for the Confirmer subagent. This
// provides ONLY the specific claim to verify, the artifact location, and the
// quoted evidence. No broader session context is included.
func BuildConfirmerContext(input *ConfirmerInput) string {
	var b strings.Builder

	fmt.Fprint(&b, "## Claim to Verify\n\n")
	fmt.Fprintf(&b, "Criterion: %s\n\n", input.CriterionName)
	fmt.Fprintf(&b, "%s\n\n", input.ClaimText)

	fmt.Fprint(&b, "## Artifact Location\n\n")
	fmt.Fprintf(&b, "Path: %s\n", input.ArtifactPath)
	fmt.Fprintf(&b, "Section: %s\n\n", input.SectionReference)

	if input.QuotedExcerpt != "" {
		fmt.Fprint(&b, "## Quoted Evidence\n\n")
		fmt.Fprintf(&b, "> %s\n", input.QuotedExcerpt)
	}

	return b.String()
}
