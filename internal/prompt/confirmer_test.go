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

package prompt_test

import (
	"strings"
	"testing"

	"github.com/go-openapi/testify/assert"
	"github.com/go-openapi/testify/require"

	"go.nwlabs.dev/magneto/internal/prompt"
)

func TestBuildConfirmerContextIncludesRoleLocalFields(t *testing.T) {
	input := &prompt.ConfirmerInput{
		CriterionName:    "Path containment",
		ClaimText:        "Symlinks can escape workspace.",
		ArtifactPath:     ".kiro/specs/example/design.md",
		SectionReference: "Security",
		QuotedExcerpt:    "The server resolves the path under WORKSPACE_ROOT.",
		FindingSeverity:  "high",
		FindingDomains:   []string{"security", "correctness"},
		AttemptNumber:    2,
		AllowedPaths:     []string{"internal/citation/", "cmd/"},
	}

	result := prompt.BuildConfirmerContext(input)

	assert.Contains(t, result, "Path containment")
	assert.Contains(t, result, "Symlinks can escape workspace.")
	assert.Contains(t, result, ".kiro/specs/example/design.md")
	assert.Contains(t, result, "Security")
	assert.Contains(t, result, "high")
	assert.Contains(t, result, "security, correctness")
	assert.Contains(t, result, "Attempt: 2 of 3")
	assert.Contains(t, result, "internal/citation/")
	assert.Contains(t, result, "cmd/")
}

func TestBuildConfirmerContextQuotedExcerptPresent(t *testing.T) {
	input := &prompt.ConfirmerInput{
		CriterionName:    "Test",
		ClaimText:        "Claim",
		ArtifactPath:     "design.md",
		SectionReference: "Overview",
		QuotedExcerpt:    "Some quoted text here.",
		FindingSeverity:  "critical",
		FindingDomains:   []string{"security"},
		AttemptNumber:    1,
	}

	result := prompt.BuildConfirmerContext(input)

	assert.Contains(t, result, "## Quoted Evidence")
	assert.Contains(t, result, "Some quoted text here.")
}

func TestBuildConfirmerContextQuotedExcerptOmittedWhenEmpty(t *testing.T) {
	input := &prompt.ConfirmerInput{
		CriterionName:    "Test",
		ClaimText:        "Claim",
		ArtifactPath:     "design.md",
		SectionReference: "Overview",
		QuotedExcerpt:    "",
		FindingSeverity:  "critical",
		FindingDomains:   []string{"security"},
		AttemptNumber:    1,
	}

	result := prompt.BuildConfirmerContext(input)

	assert.NotContains(t, result, "Quoted Evidence")
}

func TestBuildConfirmerContextExcludesProhibitedContent(t *testing.T) {
	input := &prompt.ConfirmerInput{
		CriterionName:    "Criterion",
		ClaimText:        "The claim to verify.",
		ArtifactPath:     "design.md",
		SectionReference: "Architecture",
		QuotedExcerpt:    "Evidence excerpt.",
		FindingSeverity:  "high",
		FindingDomains:   []string{"correctness"},
		AttemptNumber:    1,
		AllowedPaths:     []string{"internal/"},
	}

	result := prompt.BuildConfirmerContext(input)
	lower := strings.ToLower(result)

	prohibited := []string{
		"reasoning",
		"author",
		"shell",
		"write",
		"network",
		"deploy",
		"git push",
		"reviewer",
	}

	for _, word := range prohibited {
		require.NotContains(
			t, lower, word,
			"confirmer context must not contain prohibited term: %s", word,
		)
	}
}
