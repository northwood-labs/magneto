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
	// distinguishes required failures from auditable optional failures.
	DegradationTracker struct {
		entries []models.DegradationEntry
	}

	// RecordFailureInput contains the parameters for recording a component
	// failure. An omitted criticality remains required for compatibility and to
	// preserve the conservative legacy safety behavior.
	RecordFailureInput struct {
		Component           string
		FailureMode         string
		UnavailableValueKey string
		Criticality         models.ComponentCriticality
		AffectedCriteria    []string
	}
)

// NewDegradationTracker creates a new tracker with no degradation events.
func NewDegradationTracker() *DegradationTracker {
	return &DegradationTracker{}
}

// RecordFailure records an auditable component failure during the session.
func (dt *DegradationTracker) RecordFailure(input *RecordFailureInput) {
	criticality := input.Criticality
	if criticality == "" {
		criticality = models.CriticalityRequired
	}

	entry := models.DegradationEntry{
		Component:           input.Component,
		FailureMode:         input.FailureMode,
		Timestamp:           time.Now().UTC().Format(time.RFC3339),
		Criticality:         criticality,
		UnavailableValueKey: input.UnavailableValueKey,
		AffectedCriteria:    append([]string(nil), input.AffectedCriteria...),
	}

	dt.entries = append(dt.entries, entry)
}

// IsDegraded returns true if any required or optional component failures have
// been recorded.
func (dt *DegradationTracker) IsDegraded() bool {
	return len(dt.entries) > 0
}

// HasRequiredFailure returns true if a required workflow component failed.
func (dt *DegradationTracker) HasRequiredFailure() bool {
	return hasRequiredDegradation(dt.entries)
}

// Entries returns a copy of all recorded degradation entries.
func (dt *DegradationTracker) Entries() []models.DegradationEntry {
	entries := make([]models.DegradationEntry, len(dt.entries))
	copy(entries, dt.entries)

	for index := range entries {
		entries[index].AffectedCriteria = append([]string(nil), entries[index].AffectedCriteria...)
	}

	return entries
}

// AllowedTerminalStatus enforces terminal-status precedence for degradation
// state. Required failures downgrade proposed approval to partial review;
// optional failures remain auditable while preserving approval evaluation. A
// caller may preserve human_override only after validating its rationale.
func (dt *DegradationTracker) AllowedTerminalStatus(proposed models.TerminalStatus) models.TerminalStatus {
	if proposed == models.TerminalHumanOverride {
		return proposed
	}

	if dt.HasRequiredFailure() && proposed == models.TerminalApproved {
		return models.TerminalPartialReview
	}

	return proposed
}
