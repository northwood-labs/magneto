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

package session

import (
	"time"

	"go.nwlabs.dev/magneto/internal/models"
)

type (
	// DegradationTracker records component failures during a review session and
	// enforces the invariant that degraded sessions never produce "approved"
	// terminal status.
	DegradationTracker struct {
		entries []models.DegradationEntry
	}

	// RecordFailureInput contains the parameters for recording a component
	// failure.
	RecordFailureInput struct {
		Component        string
		FailureMode      string
		AffectedCriteria []string
	}
)

// NewDegradationTracker creates a new tracker with no degradation events.
func NewDegradationTracker() *DegradationTracker {
	return &DegradationTracker{}
}

// RecordFailure records a component failure during the session.
func (dt *DegradationTracker) RecordFailure(input *RecordFailureInput) {
	entry := models.DegradationEntry{
		Component:        input.Component,
		FailureMode:      input.FailureMode,
		AffectedCriteria: input.AffectedCriteria,
		Timestamp:        time.Now().UTC().Format(time.RFC3339),
	}

	dt.entries = append(dt.entries, entry)
}

// IsDegraded returns true if any component failures have been recorded.
func (dt *DegradationTracker) IsDegraded() bool {
	return len(dt.entries) > 0
}

// Entries returns all recorded degradation entries.
func (dt *DegradationTracker) Entries() []models.DegradationEntry {
	return dt.entries
}

// AllowedTerminalStatus returns the appropriate terminal status given the
// degradation state. If degraded and proposed is TerminalApproved, only
// TerminalPartialReview is allowed. If not degraded, the proposed status is
// returned unchanged.
func (dt *DegradationTracker) AllowedTerminalStatus(proposed models.TerminalStatus) models.TerminalStatus {
	if dt.IsDegraded() && proposed == models.TerminalApproved {
		return models.TerminalPartialReview
	}

	return proposed
}
