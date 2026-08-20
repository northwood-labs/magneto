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
	"fmt"
	"testing"

	"pgregory.net/rapid"

	"go.nwlabs.dev/magneto/internal/models"
	"go.nwlabs.dev/magneto/internal/session"
)

// Feature: adversarial-review-operational-workflow, Property 9: Required
// degradation prevents approval while optional degradation does not
//
// For any proposed terminal status and mixture of required and optional
// component failures, a required failure without a human override cannot
// finalize as `approved` and finalizes as `partial_review`; optional failures
// alone preserve approval evaluation, while a non-empty human override
// rationale finalizes as `human_override` and retains all degradation details.
//
// **Validates: Requirements 9.3, 9.4, 9.5, 9.6**.

type (
	// DegradationPropertyEntry holds the generated inputs for a single
	// degradation entry in the property test.
	DegradationPropertyEntry struct {
		Component        string
		FailureMode      string
		Criticality      models.ComponentCriticality
		AffectedCriteria []string
	}

	// DegradationPropertyAssertInput holds the generated state for verifying
	// degradation invariants.
	DegradationPropertyAssertInput struct {
		Tracker          *session.DegradationTracker
		ProposedStatus   models.TerminalStatus
		Entries          []DegradationPropertyEntry
		OverrideProvided bool
	}
)

// TestProperty_RequiredDegradationPreventsApproval verifies Property 9:
// Required degradation prevents approval while optional degradation does not.
//
// Feature: adversarial-review-operational-workflow, Property 9: Required
// degradation prevents approval while optional degradation does not.
//
// **Validates: Requirements 9.3, 9.4, 9.5, 9.6**.
func TestProperty_RequiredDegradationPreventsApproval(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate 0-5 degradation entries with random criticality.
		entryCount := rapid.IntRange(0, 5).Draw(t, "entry_count")
		entries := drawDegradationEntries(t, entryCount)

		// Create and populate the tracker.
		tracker := session.NewDegradationTracker()

		for i := range entries {
			tracker.RecordFailure(&session.RecordFailureInput{
				Component:        entries[i].Component,
				FailureMode:      entries[i].FailureMode,
				Criticality:      entries[i].Criticality,
				AffectedCriteria: entries[i].AffectedCriteria,
			})
		}

		// Generate a proposed terminal status.
		proposedStatus := drawTerminalStatus(t)

		// Generate whether a human override is provided.
		overrideProvided := rapid.Bool().Draw(t, "override_provided")

		// Determine the effective proposed status when an override is provided.
		effectiveProposed := proposedStatus
		if overrideProvided {
			effectiveProposed = models.TerminalHumanOverride
		}

		// Get the allowed terminal status.
		allowed := tracker.AllowedTerminalStatus(effectiveProposed)

		// Assert all degradation invariants.
		assertDegradationInvariants(t, &DegradationPropertyAssertInput{
			Tracker:          tracker,
			Entries:          entries,
			ProposedStatus:   effectiveProposed,
			OverrideProvided: overrideProvided,
		}, allowed)
	})
}

// drawDegradationEntries generates a slice of degradation entries with random
// required or optional criticality.
func drawDegradationEntries(t *rapid.T, count int) []DegradationPropertyEntry {
	entries := make([]DegradationPropertyEntry, 0, count)

	for i := range count {
		criticality := drawCriticality(t)
		criteriaCount := rapid.IntRange(1, 3).Draw(t, "criteria_count")

		criteria := make([]string, 0, criteriaCount)
		for j := range criteriaCount {
			criteria = append(criteria, fmt.Sprintf("criterion-%d-%d", i, j))
		}

		entries = append(entries, DegradationPropertyEntry{
			Component:        fmt.Sprintf("component-%d", i),
			FailureMode:      fmt.Sprintf("failure-mode-%d", i),
			Criticality:      criticality,
			AffectedCriteria: criteria,
		})
	}

	return entries
}

// drawCriticality generates either required or optional criticality.
func drawCriticality(t *rapid.T) models.ComponentCriticality {
	if rapid.Bool().Draw(t, "is_required") {
		return models.CriticalityRequired
	}

	return models.CriticalityOptional
}

// drawTerminalStatus generates one of the four valid terminal statuses.
func drawTerminalStatus(t *rapid.T) models.TerminalStatus {
	statuses := []models.TerminalStatus{
		models.TerminalApproved,
		models.TerminalNotApproved,
		models.TerminalPartialReview,
		models.TerminalHumanOverride,
	}

	idx := rapid.IntRange(0, len(statuses)-1).Draw(t, "terminal_status_index")

	return statuses[idx]
}

// assertDegradationInvariants verifies all degradation property invariants for
// the given tracker state and allowed status.
func assertDegradationInvariants(t *rapid.T, input *DegradationPropertyAssertInput, allowed models.TerminalStatus) {
	assertRequiredBlocksApproval(t, input, allowed)
	assertOptionalPreservesApproval(t, input, allowed)
	assertOverridePreservesDegradation(t, input, allowed)
	assertEntriesPreserved(t, input)
}

// assertRequiredBlocksApproval verifies that a required failure without a human
// override cannot produce approved status (Requirement 9.3, 9.4).
func assertRequiredBlocksApproval(t *rapid.T, input *DegradationPropertyAssertInput, allowed models.TerminalStatus) {
	hasRequired := input.Tracker.HasRequiredFailure()

	if !hasRequired {
		return
	}

	if input.ProposedStatus == models.TerminalHumanOverride {
		return
	}

	if input.ProposedStatus == models.TerminalApproved && allowed == models.TerminalApproved {
		t.Fatal("required degradation must prevent approved status without human override (Requirement 9.3, 9.4)")
	}

	if input.ProposedStatus == models.TerminalApproved && allowed != models.TerminalPartialReview {
		t.Fatalf(
			"required degradation with proposed approved must yield partial_review, got %q (Requirement 9.4)",
			allowed,
		)
	}
}

// assertOptionalPreservesApproval verifies that optional-only failures preserve
// approval evaluation (Requirement 9.3).
func assertOptionalPreservesApproval(t *rapid.T, input *DegradationPropertyAssertInput, allowed models.TerminalStatus) {
	if len(input.Entries) == 0 {
		return
	}

	hasRequired := input.Tracker.HasRequiredFailure()

	if hasRequired {
		return
	}

	// All entries are optional: approval evaluation must be preserved.
	if input.ProposedStatus == models.TerminalApproved && allowed != models.TerminalApproved {
		t.Fatalf(
			"optional-only degradation must preserve approval evaluation, got %q instead of approved (Requirement 9.3)",
			allowed,
		)
	}
}

// assertOverridePreservesDegradation verifies that a human override produces
// human_override status and retains all degradation details (Requirement 9.5,
// 9.6).
func assertOverridePreservesDegradation(
	t *rapid.T,
	input *DegradationPropertyAssertInput,
	allowed models.TerminalStatus,
) {
	if input.ProposedStatus != models.TerminalHumanOverride {
		return
	}

	if allowed != models.TerminalHumanOverride {
		t.Fatalf("human override must produce human_override status, got %q (Requirement 9.5)", allowed)
	}

	// Verify degradation details are retained.
	trackerEntries := input.Tracker.Entries()

	if len(trackerEntries) != len(input.Entries) {
		t.Fatalf(
			"human override must retain all degradation details: expected %d entries, got %d (Requirement 9.6)",
			len(input.Entries),
			len(trackerEntries),
		)
	}
}

// assertEntriesPreserved verifies that the tracker always retains all
// degradation entries regardless of the terminal status.
func assertEntriesPreserved(t *rapid.T, input *DegradationPropertyAssertInput) {
	trackerEntries := input.Tracker.Entries()

	if len(trackerEntries) != len(input.Entries) {
		t.Fatalf(
			"tracker must preserve all degradation entries: expected %d, got %d",
			len(input.Entries),
			len(trackerEntries),
		)
	}

	// Verify each entry's component and criticality match.
	for i, entry := range input.Entries {
		if trackerEntries[i].Component != entry.Component {
			t.Fatalf(
				"entry %d component mismatch: expected %q, got %q",
				i,
				entry.Component,
				trackerEntries[i].Component,
			)
		}

		if trackerEntries[i].Criticality != entry.Criticality {
			t.Fatalf(
				"entry %d criticality mismatch: expected %q, got %q",
				i,
				entry.Criticality,
				trackerEntries[i].Criticality,
			)
		}
	}
}
