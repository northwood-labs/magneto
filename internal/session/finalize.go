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
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"go.nwlabs.dev/magneto/internal/models"
)

var (
	// ErrHumanBlockDecision indicates a block acceptance has an invalid
	// decision.
	ErrHumanBlockDecision = errors.New("human block acceptance has an invalid decision")

	// ErrHumanBlockRationale indicates a block acceptance lacks a rationale.
	ErrHumanBlockRationale = errors.New("human block acceptance requires a rationale")

	// ErrHumanOverrideDecision indicates an override has an invalid decision.
	ErrHumanOverrideDecision = errors.New("human override has an invalid decision")

	// ErrHumanOverrideRationale indicates an override lacks a rationale.
	ErrHumanOverrideRationale = errors.New("human override requires a rationale")

	// ErrReviewSessionRequired indicates finalization received no session data.
	ErrReviewSessionRequired = errors.New("review session is required")

	// ErrSessionIDRequired indicates finalization lacks the terminal session
	// ID.
	ErrSessionIDRequired = errors.New("review session ID is required for finalization")

	// ErrTaskExecutionIDRequired indicates finalization lacks the task
	// execution ID.
	ErrTaskExecutionIDRequired = errors.New("task execution ID is required for finalization")

	// ErrTerminalStatusInvalid indicates a session requested an unsupported
	// terminal status.
	ErrTerminalStatusInvalid = errors.New("terminal status is not supported")

	// ErrUnavailableValueInvalid indicates a partial review lacks an
	// unavailable value field or reason.
	ErrUnavailableValueInvalid = errors.New("unavailable value requires a field and reason")

	// ErrUnavailableValueStatus indicates unavailable values were supplied for
	// a terminal status other than partial review.
	ErrUnavailableValueStatus = errors.New("unavailable values are allowed only for partial review")
)

type (
	// FinalizeReviewSessionInput contains the completed review session to
	// validate and normalize before terminal record persistence.
	FinalizeReviewSessionInput struct {
		Session *models.ReviewSessionOutput
	}

	// FinalizeReviewSessionResult contains the normalized terminal session and
	// its deterministic idempotency key.
	FinalizeReviewSessionResult struct {
		Session        *models.ReviewSessionOutput
		IdempotencyKey string
	}
)

// FinalizeReviewSession validates terminal-only session data and derives the
// final status. Human overrides take precedence over required degradation,
// followed by a successful attack round; all other sessions are not approved.
func FinalizeReviewSession(input *FinalizeReviewSessionInput) (*FinalizeReviewSessionResult, error) {
	if input == nil || input.Session == nil {
		return nil, ErrReviewSessionRequired
	}

	if !isTerminalStatus(input.Session.Metadata.TerminalStatus) {
		return nil, ErrTerminalStatusInvalid
	}

	taskExecutionID := strings.TrimSpace(input.Session.Metadata.TaskExecutionID)
	if taskExecutionID == "" {
		return nil, ErrTaskExecutionIDRequired
	}

	sessionID := strings.TrimSpace(input.Session.Metadata.SessionID)
	if sessionID == "" {
		return nil, ErrSessionIDRequired
	}

	validationErr := validateHumanDecisions(input.Session)
	if validationErr != nil {
		return nil, fmt.Errorf("validate human decisions: %w", validationErr)
	}

	finalized := *input.Session
	finalized.Metadata.TerminalStatus = terminalStatus(input.Session)

	unavailableErr := validateUnavailableValues(&finalized)
	if unavailableErr != nil {
		return nil, fmt.Errorf("validate unavailable values: %w", unavailableErr)
	}

	idempotencyKey := terminalIdempotencyKey(taskExecutionID, sessionID)
	finalized.TerminalIdempotencyKey = idempotencyKey

	return &FinalizeReviewSessionResult{
		Session:        &finalized,
		IdempotencyKey: idempotencyKey,
	}, nil
}

// isTerminalStatus reports whether status is accepted as a terminal request.
func isTerminalStatus(status models.TerminalStatus) bool {
	switch status {
	case models.TerminalApproved,
		models.TerminalNotApproved,
		models.TerminalPartialReview,
		models.TerminalHumanOverride:
		return true
	default:
		return false
	}
}

// validateHumanDecisions ensures decisions that affect terminal persistence
// are explicit and have an auditable rationale.
func validateHumanDecisions(session *models.ReviewSessionOutput) error {
	for _, override := range session.HumanOverrides {
		if override.Decision != models.HumanDecisionOverride {
			return ErrHumanOverrideDecision
		}

		if strings.TrimSpace(override.HumanRationale) == "" {
			return ErrHumanOverrideRationale
		}
	}

	blockAcceptance := session.HumanBlockAcceptance
	if blockAcceptance == nil {
		return nil
	}

	if blockAcceptance.Decision != models.HumanDecisionBlockAcceptance {
		return ErrHumanBlockDecision
	}

	if strings.TrimSpace(blockAcceptance.HumanRationale) == "" {
		return ErrHumanBlockRationale
	}

	return nil
}

// terminalStatus derives status according to the required terminal precedence.
func terminalStatus(session *models.ReviewSessionOutput) models.TerminalStatus {
	if len(session.HumanOverrides) > 0 {
		return models.TerminalHumanOverride
	}

	if hasRequiredDegradation(session.Metadata.DegradedComponents) {
		return models.TerminalPartialReview
	}

	if attackRoundSucceeded(session.AttackRoundResult) {
		return models.TerminalApproved
	}

	return models.TerminalNotApproved
}

// attackRoundSucceeded reports whether the mandatory attack round completed
// without any novel issues.
func attackRoundSucceeded(result *models.AttackRoundResult) bool {
	return result != nil && !result.NewIssuesFound
}

// validateUnavailableValues retains unavailable values only for partial review
// records, where each value must name the unavailable field and its reason.
func validateUnavailableValues(session *models.ReviewSessionOutput) error {
	if session.Metadata.TerminalStatus != models.TerminalPartialReview {
		if len(session.UnavailableValues) > 0 {
			return ErrUnavailableValueStatus
		}

		return nil
	}

	for _, unavailable := range session.UnavailableValues {
		if strings.TrimSpace(unavailable.Field) == "" || strings.TrimSpace(unavailable.Reason) == "" {
			return ErrUnavailableValueInvalid
		}
	}

	return nil
}

// terminalIdempotencyKey returns a stable key derived from the task execution
// and terminal session identifiers without exposing delimiter ambiguity.
func terminalIdempotencyKey(taskExecutionID, sessionID string) string {
	payload := taskExecutionID + "\x00" + sessionID
	digest := sha256.Sum256([]byte(payload))

	return hex.EncodeToString(digest[:])
}
