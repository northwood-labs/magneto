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

package session_test

import (
	"testing"

	"github.com/go-openapi/testify/assert"

	"go.nwlabs.dev/magneto/internal/models"
	"go.nwlabs.dev/magneto/internal/session"
)

func TestDegradationTracker(t *testing.T) {
	tests := []struct {
		fn   func(t *testing.T)
		name string
	}{
		{
			name: "new tracker is not degraded",
			fn: func(t *testing.T) {
				t.Helper()

				dt := session.NewDegradationTracker()
				assert.False(t, dt.IsDegraded())
			},
		},
		{
			name: "single failure makes degraded",
			fn: func(t *testing.T) {
				t.Helper()

				dt := session.NewDegradationTracker()
				dt.RecordFailure(&session.RecordFailureInput{
					Component:        "citation-gate",
					FailureMode:      "unreachable",
					AffectedCriteria: []string{"error-handling"},
				})

				assert.True(t, dt.IsDegraded())
			},
		},
		{
			name: "multiple failures tracked",
			fn: func(t *testing.T) {
				t.Helper()

				dt := session.NewDegradationTracker()
				dt.RecordFailure(&session.RecordFailureInput{
					Component:        "citation-gate",
					FailureMode:      "unreachable",
					AffectedCriteria: []string{"error-handling"},
				})
				dt.RecordFailure(&session.RecordFailureInput{
					Component:        "confirmer",
					FailureMode:      "model unavailable",
					AffectedCriteria: []string{"security"},
				})
				dt.RecordFailure(&session.RecordFailureInput{
					Component:        "steering-file",
					FailureMode:      "malformed entry",
					AffectedCriteria: []string{"data-integrity"},
				})

				assert.Len(t, dt.Entries(), 3)
			},
		},
		{
			name: "entries contain correct data",
			fn: func(t *testing.T) {
				t.Helper()

				dt := session.NewDegradationTracker()
				dt.RecordFailure(&session.RecordFailureInput{
					Component:   "confirmer",
					FailureMode: "timeout after 3 attempts",
					AffectedCriteria: []string{
						"security-boundaries",
						"correctness",
					},
				})

				entries := dt.Entries()
				assert.Len(t, entries, 1)
				assert.Equal(t, "confirmer", entries[0].Component)
				assert.Equal(t, "timeout after 3 attempts", entries[0].FailureMode)
				assert.Equal(t, []string{"security-boundaries", "correctness"}, entries[0].AffectedCriteria)
			},
		},
		{
			name: "approved status downgraded when degraded",
			fn: func(t *testing.T) {
				t.Helper()

				dt := session.NewDegradationTracker()
				dt.RecordFailure(&session.RecordFailureInput{
					Component:        "citation-gate",
					FailureMode:      "unreachable",
					AffectedCriteria: []string{"error-handling"},
				})

				result := dt.AllowedTerminalStatus(models.TerminalApproved)

				assert.Equal(t, models.TerminalPartialReview, result)
			},
		},
		{
			name: "non-approved status unchanged when degraded",
			fn: func(t *testing.T) {
				t.Helper()

				dt := session.NewDegradationTracker()
				dt.RecordFailure(&session.RecordFailureInput{
					Component:        "reviewer",
					FailureMode:      "model error",
					AffectedCriteria: []string{"all"},
				})

				result := dt.AllowedTerminalStatus(models.TerminalNotApproved)

				assert.Equal(t, models.TerminalNotApproved, result)
			},
		},
		{
			name: "approved status unchanged when not degraded",
			fn: func(t *testing.T) {
				t.Helper()

				dt := session.NewDegradationTracker()

				result := dt.AllowedTerminalStatus(models.TerminalApproved)

				assert.Equal(t, models.TerminalApproved, result)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.fn(t)
		})
	}
}
