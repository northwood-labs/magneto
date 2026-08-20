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
	"errors"
	"testing"

	"github.com/go-openapi/testify/assert"
	"github.com/go-openapi/testify/require"

	"go.nwlabs.dev/magneto/internal/models"
	"go.nwlabs.dev/magneto/internal/session"
)

func TestFinalizeReviewSession(t *testing.T) {
	tests := []struct {
		run  func(t *testing.T)
		name string
	}{
		{
			name: "human override takes terminal precedence",
			run: func(t *testing.T) {
				t.Helper()

				review := terminalReviewSession()

				review.Metadata.DegradedComponents = []models.DegradationEntry{{
					Criticality: models.CriticalityRequired,
				}}
				review.HumanOverrides = []models.HumanOverride{{
					Decision:       models.HumanDecisionOverride,
					HumanRationale: "The human accepts this risk.",
				}}

				result, finalizationErr := session.FinalizeReviewSession(&session.FinalizeReviewSessionInput{
					Session: review,
				})
				require.NoError(t, finalizationErr)
				assert.Equal(t, models.TerminalHumanOverride, result.Session.Metadata.TerminalStatus)
			},
		},
		{
			name: "required degradation produces partial review with unavailable values",
			run: func(t *testing.T) {
				t.Helper()

				review := terminalReviewSession()

				review.Metadata.DegradedComponents = []models.DegradationEntry{
					{Criticality: models.CriticalityRequired},
				}
				review.UnavailableValues = []models.UnavailableValue{{
					Field:  "loaded_rubric_criteria",
					Reason: "Rubric loader was unavailable.",
				}}

				result, finalizationErr := session.FinalizeReviewSession(&session.FinalizeReviewSessionInput{
					Session: review,
				})
				require.NoError(t, finalizationErr)
				assert.Equal(t, models.TerminalPartialReview, result.Session.Metadata.TerminalStatus)
				assert.Equal(t, review.UnavailableValues, result.Session.UnavailableValues)
			},
		},
		{
			name: "successful attack round approves without required degradation",
			run: func(t *testing.T) {
				t.Helper()

				review := terminalReviewSession()

				result, finalizationErr := session.FinalizeReviewSession(&session.FinalizeReviewSessionInput{
					Session: review,
				})
				require.NoError(t, finalizationErr)
				assert.Equal(t, models.TerminalApproved, result.Session.Metadata.TerminalStatus)
			},
		},
		{
			name: "no successful attack round is not approved",
			run: func(t *testing.T) {
				t.Helper()

				review := terminalReviewSession()

				review.AttackRoundResult = nil

				result, finalizationErr := session.FinalizeReviewSession(&session.FinalizeReviewSessionInput{
					Session: review,
				})
				require.NoError(t, finalizationErr)
				assert.Equal(t, models.TerminalNotApproved, result.Session.Metadata.TerminalStatus)
			},
		},
		{
			name: "invalid terminal status is rejected",
			run: func(t *testing.T) {
				t.Helper()

				review := terminalReviewSession()
				review.Metadata.TerminalStatus = "pending"

				_, finalizationErr := session.FinalizeReviewSession(&session.FinalizeReviewSessionInput{
					Session: review,
				})

				assert.True(t, errors.Is(finalizationErr, session.ErrTerminalStatusInvalid))
			},
		},
		{
			name: "human decisions require non-empty rationales",
			run: func(t *testing.T) {
				t.Helper()

				tests := []struct {
					expected  error
					configure func(review *models.ReviewSessionOutput)
					name      string
				}{
					{
						name: "override",
						configure: func(review *models.ReviewSessionOutput) {
							review.HumanOverrides = []models.HumanOverride{
								{
									Decision:       models.HumanDecisionOverride,
									HumanRationale: " \t ",
								},
							}
						},
						expected: session.ErrHumanOverrideRationale,
					},
					{
						name: "block acceptance",
						configure: func(review *models.ReviewSessionOutput) {
							review.HumanBlockAcceptance = &models.HumanBlockAcceptance{
								Decision:       models.HumanDecisionBlockAcceptance,
								HumanRationale: " \n ",
							}
						},
						expected: session.ErrHumanBlockRationale,
					},
				}

				for _, tt := range tests {
					t.Run(tt.name, func(t *testing.T) {
						review := terminalReviewSession()
						tt.configure(review)

						_, finalizationErr := session.FinalizeReviewSession(&session.FinalizeReviewSessionInput{
							Session: review,
						})
						assert.True(t, errors.Is(finalizationErr, tt.expected))
					})
				}
			},
		},
		{
			name: "unavailable values are rejected outside partial review",
			run: func(t *testing.T) {
				t.Helper()

				review := terminalReviewSession()

				review.UnavailableValues = []models.UnavailableValue{
					{
						Field:  "attack_round_result",
						Reason: "No attack round ran.",
					},
				}

				_, finalizationErr := session.FinalizeReviewSession(&session.FinalizeReviewSessionInput{
					Session: review,
				})

				assert.True(t, errors.Is(finalizationErr, session.ErrUnavailableValueStatus))
			},
		},
		{
			name: "idempotency key is stable and bound to both identifiers",
			run: func(t *testing.T) {
				t.Helper()

				first := terminalReviewSession()
				second := terminalReviewSession()
				third := terminalReviewSession()

				third.Metadata.SessionID = "session-2"

				firstResult, firstErr := session.FinalizeReviewSession(
					&session.FinalizeReviewSessionInput{Session: first},
				)
				require.NoError(t, firstErr)

				secondResult, secondErr := session.FinalizeReviewSession(
					&session.FinalizeReviewSessionInput{Session: second},
				)
				require.NoError(t, secondErr)

				thirdResult, thirdErr := session.FinalizeReviewSession(
					&session.FinalizeReviewSessionInput{Session: third},
				)
				require.NoError(t, thirdErr)

				assert.Equal(t, firstResult.IdempotencyKey, secondResult.IdempotencyKey)
				assert.Equal(t, firstResult.IdempotencyKey, firstResult.Session.TerminalIdempotencyKey)
				assert.NotEqual(t, firstResult.IdempotencyKey, thirdResult.IdempotencyKey)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.run(t)
		})
	}
}

func terminalReviewSession() *models.ReviewSessionOutput {
	return &models.ReviewSessionOutput{
		Metadata: models.SessionMetadata{
			TaskExecutionID: "task-1",
			SessionID:       "session-1",
			TerminalStatus:  models.TerminalApproved,
		},
		AttackRoundResult: &models.AttackRoundResult{},
	}
}
