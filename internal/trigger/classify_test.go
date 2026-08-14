// Copyright 2025-2026, Northwood Labs, LLC <license@northwood-labs.com>
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

package trigger_test

import (
	"testing"

	"github.com/go-openapi/testify/assert"

	"go.nwlabs.dev/magneto/internal/trigger"
)

// TestClassify exercises the trigger classification logic with table-driven
// tests covering blast-radius domain matching, foundational trust detection,
// skip conditions, ambiguous defaults, custom domain lists, and empty domain
// handling.
func TestClassify(t *testing.T) {
	tests := []struct {
		name             string
		input            *trigger.ClassifyInput
		expectedDecision trigger.Decision
		expectedReason   string
		expectedAmbig    bool
	}{
		{
			name: "blast-radius domain match triggers review",
			input: &trigger.ClassifyInput{
				Artifact: &trigger.ArtifactInfo{
					Domain: "auth",
				},
			},
			expectedDecision: trigger.DecisionTrigger,
			expectedReason:   trigger.ReasonBlastRadius,
			expectedAmbig:    false,
		},
		{
			name: "blast-radius domain match is case-insensitive",
			input: &trigger.ClassifyInput{
				Artifact: &trigger.ArtifactInfo{
					Domain: "AUTH",
				},
			},
			expectedDecision: trigger.DecisionTrigger,
			expectedReason:   trigger.ReasonBlastRadius,
			expectedAmbig:    false,
		},
		{
			name: "foundational trust triggers review",
			input: &trigger.ClassifyInput{
				Artifact: &trigger.ArtifactInfo{
					IsFoundational: true,
				},
			},
			expectedDecision: trigger.DecisionTrigger,
			expectedReason:   trigger.ReasonFoundational,
			expectedAmbig:    false,
		},
		{
			name: "single-file revertible human-reviewed skips review",
			input: &trigger.ClassifyInput{
				Artifact: &trigger.ArtifactInfo{
					IsSingleFile:          true,
					IsRevertible:          true,
					IsHumanReviewedBefore: true,
				},
			},
			expectedDecision: trigger.DecisionSkip,
			expectedReason:   trigger.ReasonSkipConditions,
			expectedAmbig:    false,
		},
		{
			name: "partial skip conditions only single file defaults to trigger",
			input: &trigger.ClassifyInput{
				Artifact: &trigger.ArtifactInfo{
					IsSingleFile: true,
				},
			},
			expectedDecision: trigger.DecisionTrigger,
			expectedReason:   trigger.ReasonAmbiguous,
			expectedAmbig:    true,
		},
		{
			name: "ambiguous classification defaults to trigger",
			input: &trigger.ClassifyInput{
				Artifact: &trigger.ArtifactInfo{
					Domain:         "unknown-domain",
					IsSingleFile:   true,
					IsRevertible:   false,
					IsFoundational: false,
				},
			},
			expectedDecision: trigger.DecisionTrigger,
			expectedReason:   trigger.ReasonAmbiguous,
			expectedAmbig:    true,
		},
		{
			name: "custom blast-radius domains override defaults",
			input: &trigger.ClassifyInput{
				Artifact: &trigger.ArtifactInfo{
					Domain: "deployment",
				},
				BlastRadiusDomains: []string{
					"deployment",
					"networking",
				},
			},
			expectedDecision: trigger.DecisionTrigger,
			expectedReason:   trigger.ReasonBlastRadius,
			expectedAmbig:    false,
		},
		{
			name: "empty domain does not match blast-radius",
			input: &trigger.ClassifyInput{
				Artifact: &trigger.ArtifactInfo{
					Domain: "",
				},
			},
			expectedDecision: trigger.DecisionTrigger,
			expectedReason:   trigger.ReasonAmbiguous,
			expectedAmbig:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := trigger.Classify(tt.input)

			assert.Equal(t, tt.expectedDecision, result.Decision)
			assert.Equal(t, tt.expectedReason, result.Reason)
			assert.Equal(t, tt.expectedAmbig, result.Ambiguous)
		})
	}
}
