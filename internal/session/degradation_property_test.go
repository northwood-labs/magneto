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

	"pgregory.net/rapid"

	"go.nwlabs.dev/magneto/internal/models"
	"go.nwlabs.dev/magneto/internal/session"
)

// TestProperty_DegradedSessionNeverApproved verifies Property 7: Degraded
// sessions never produce "approved" status.
//
// For any review session where at least one component experienced a failure or
// degradation event, the terminal status SHALL NOT be "reviewed" or "approved".
//
// **Validates: Requirements 10.4**.
func TestProperty_DegradedSessionNeverApproved(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Create a fresh degradation tracker.
		tracker := session.NewDegradationTracker()

		// Record 1-5 random failures with generated component names and failure
		// modes.
		failureCount := rapid.IntRange(1, 5).Draw(t, "failure_count")

		for i := range failureCount {
			_ = i

			component := rapid.StringMatching(`[A-Z][a-z]{3,15}`).Draw(t, "component")
			failureMode := rapid.StringMatching(`[a-z]{3,10} [a-z]{3,10}`).Draw(t, "failure_mode")

			tracker.RecordFailure(&session.RecordFailureInput{
				Component:   component,
				FailureMode: failureMode,
				AffectedCriteria: []string{
					rapid.StringMatching(`[a-z]{3,12}`).Draw(t, "criterion"),
				},
			})
		}

		// Assert that IsDegraded returns true after recording
		// failures.
		if !tracker.IsDegraded() {
			t.Fatal("expected IsDegraded()=true after recording failures")
		}

		// Call AllowedTerminalStatus with TerminalApproved.
		result := tracker.AllowedTerminalStatus(models.TerminalApproved)

		// Property: the result must NEVER be TerminalApproved for a
		// degraded session.
		if result == models.TerminalApproved {
			t.Fatalf(
				"expected terminal status != %q for degraded session, got %q with %d failures recorded",
				models.TerminalApproved,
				result,
				failureCount,
			)
		}

		// Assert the result is specifically TerminalPartialReview.
		if result != models.TerminalPartialReview {
			t.Fatalf(
				"expected terminal status %q for degraded session proposing approved, got %q",
				models.TerminalPartialReview,
				result,
			)
		}
	})
}
