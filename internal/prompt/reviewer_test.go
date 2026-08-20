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

func TestBuildReviewerContextIncludesRoleLocalFields(t *testing.T) {
	input := &prompt.ReviewerInput{
		ArtifactPath:  ".kiro/specs/example/design.md",
		RubricContent: "## Criterion A\n\nCheck for correctness.",
		RoundNumber:   3,
		AllowedPaths:  []string{"internal/", "cmd/"},
		PriorFailureFingerprints: []prompt.FailureFingerprint{{
			CriterionName:         "Path containment",
			CriterionSatisfaction: 4,
			EvidenceHash:          "abc123",
		}},
	}

	result := prompt.BuildReviewerContext(input)

	assert.Contains(t, result, ".kiro/specs/example/design.md")
	assert.Contains(t, result, "## Criterion A")
	assert.Contains(t, result, "Check for correctness.")
	assert.Contains(t, result, "Round: 3 of 5")
	assert.Contains(t, result, "internal/")
	assert.Contains(t, result, "cmd/")
}

func TestBuildReviewerContextRendersFingerprintsAsOpaque(t *testing.T) {
	input := &prompt.ReviewerInput{
		ArtifactPath:  "design.md",
		RubricContent: "rubric",
		RoundNumber:   1,
		PriorFailureFingerprints: []prompt.FailureFingerprint{
			{
				CriterionName:         "Security",
				CriterionSatisfaction: 3,
				EvidenceHash:          "deadbeef",
			},
			{
				CriterionName:         "Correctness",
				CriterionSatisfaction: 5,
				EvidenceHash:          "cafebabe",
			},
		},
	}

	result := prompt.BuildReviewerContext(input)

	assert.Contains(t, result, "criterion=Security satisfaction=3 hash=deadbeef")
	assert.Contains(t, result, "criterion=Correctness satisfaction=5 hash=cafebabe")
}

func TestBuildReviewerContextEmptyFingerprintsNoSection(t *testing.T) {
	input := &prompt.ReviewerInput{
		ArtifactPath:  "design.md",
		RubricContent: "rubric",
		RoundNumber:   1,
	}

	result := prompt.BuildReviewerContext(input)

	assert.NotContains(t, result, "Prior Failure Fingerprints")
}

func TestBuildReviewerContextEmptyPathsNoSection(t *testing.T) {
	input := &prompt.ReviewerInput{
		ArtifactPath:  "design.md",
		RubricContent: "rubric",
		RoundNumber:   1,
	}

	result := prompt.BuildReviewerContext(input)

	assert.NotContains(t, result, "Allowed Repository Paths")
}

func TestBuildReviewerContextExcludesProhibitedContent(t *testing.T) {
	input := &prompt.ReviewerInput{
		ArtifactPath:  "design.md",
		RubricContent: "rubric content here",
		RoundNumber:   2,
		AllowedPaths:  []string{"src/"},
		PriorFailureFingerprints: []prompt.FailureFingerprint{{
			CriterionName:         "Test",
			CriterionSatisfaction: 5,
			EvidenceHash:          "hash",
		}},
	}

	result := prompt.BuildReviewerContext(input)
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
			"reviewer context must not contain prohibited term: %s", word,
		)
	}
}
