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

func TestBuildAttackContextIncludesRoleLocalFields(t *testing.T) {
	input := &prompt.AttackInput{
		ArtifactPath:  ".kiro/specs/example/design.md",
		RubricContent: "## Security\n\nCheck symlink handling.",
		AttackFocus:   "Path traversal via crafted symlinks",
		AllowedPaths:  []string{"internal/citation/", "cmd/"},
		PriorFailureFingerprints: []prompt.FailureFingerprint{{
			CriterionName:         "Containment",
			CriterionSatisfaction: 6,
			EvidenceHash:          "abc123",
		}},
	}

	result := prompt.BuildAttackContext(input)

	assert.Contains(t, result, ".kiro/specs/example/design.md")
	assert.Contains(t, result, "## Security")
	assert.Contains(t, result, "Check symlink handling.")
	assert.Contains(t, result, "Path traversal via crafted symlinks")
	assert.Contains(t, result, "criterion=Containment satisfaction=6 hash=abc123")
	assert.Contains(t, result, "internal/citation/")
	assert.Contains(t, result, "cmd/")
}

func TestBuildAttackContextEmptyFingerprintsNoSection(t *testing.T) {
	input := &prompt.AttackInput{
		ArtifactPath:  "design.md",
		RubricContent: "rubric",
	}

	result := prompt.BuildAttackContext(input)

	assert.NotContains(t, result, "Prior Failure Fingerprints")
}

func TestBuildAttackContextEmptyFocusNoSection(t *testing.T) {
	input := &prompt.AttackInput{
		ArtifactPath:  "design.md",
		RubricContent: "rubric",
	}

	result := prompt.BuildAttackContext(input)

	assert.NotContains(t, result, "Attack Focus")
}

func TestBuildAttackContextEmptyPathsNoSection(t *testing.T) {
	input := &prompt.AttackInput{
		ArtifactPath:  "design.md",
		RubricContent: "rubric",
	}

	result := prompt.BuildAttackContext(input)

	assert.NotContains(t, result, "Allowed Repository Paths")
}

func TestBuildAttackContextExcludesProhibitedContent(t *testing.T) {
	input := &prompt.AttackInput{
		ArtifactPath:  "design.md",
		RubricContent: "rubric content here",
		AttackFocus:   "test the boundary conditions",
		AllowedPaths:  []string{"src/"},
		PriorFailureFingerprints: []prompt.FailureFingerprint{{
			CriterionName:         "Test",
			CriterionSatisfaction: 5,
			EvidenceHash:          "hash",
		}},
	}

	result := prompt.BuildAttackContext(input)
	lower := strings.ToLower(result)

	prohibited := []string{
		"reasoning",
		"author",
		"shell",
		"write",
		"network",
		"deploy",
		"git push",
		"git commit",
	}

	for _, word := range prohibited {
		require.NotContains(
			t, lower, word,
			"attack context must not contain prohibited term: %s", word,
		)
	}
}
