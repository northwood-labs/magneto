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

package cmd

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-openapi/testify/assert"
	"github.com/go-openapi/testify/require"
	"github.com/mark3labs/mcp-go/mcp"

	"go.nwlabs.dev/magneto/internal/models"
	"go.nwlabs.dev/magneto/internal/output"
	"go.nwlabs.dev/magneto/internal/session"
	"go.nwlabs.dev/magneto/internal/trigger"
)

// integrationDesignContent is a design artifact fixture used by the operational
// workflow integration tests.
const integrationDesignContent = `# Design

## Overview

The system enforces structurally independent review of design
artifacts by running a context-isolated reviewer subagent.

## Architecture

Components communicate over stdio using MCP protocol.
The citation gate validates quoted excerpts deterministically.

## Security

Authentication uses short-lived tokens with automatic rotation.
Path containment prevents symlink escape from WORKSPACE_ROOT.
`

// TestIntegration_ChangedEligibleDesign_FullWorkflow verifies the complete
// operational path: trigger classification, session start, schema gate,
// citation gate, round management, attack round, and terminal finalization
// without modifying the design artifact.
func TestIntegration_ChangedEligibleDesign_FullWorkflow(t *testing.T) {
	tmpDir := t.TempDir()

	t.Setenv("WORKSPACE_ROOT", tmpDir)

	workspaceRoot = tmpDir

	artifactPath := filepath.Join(tmpDir, "design.md")
	writeErr := os.WriteFile(artifactPath, []byte(integrationDesignContent), 0o0666)
	require.NoError(t, writeErr)

	// Step 1: Trigger classification with a blast-radius domain.
	classifyResult := trigger.Classify(&trigger.ClassifyInput{
		Artifact: &trigger.ArtifactInfo{
			Domain: "auth",
		},
	})
	require.Equal(t, trigger.DecisionTrigger, classifyResult.Decision)
	assert.Equal(t, trigger.ReasonBlastRadius, classifyResult.Reason)

	// Step 2: Start session through round manager.
	rm := session.NewRoundManager()
	dt := session.NewDegradationTracker()

	assert.Equal(t, session.StateActive, rm.State())
	assert.Equal(t, 1, rm.CurrentRound())

	// Step 3: Validate finding schema through the real MCP handler.
	findingIndex := 0
	sessionID := "integration-test-session-001"

	schemaRequest := mcp.CallToolRequest{}

	schemaRequest.Params.Arguments = map[string]any{
		"finding": map[string]any{
			"criterion_name":         "Path containment",
			"criterion_satisfaction": 8,
			"finding_severity":       "high",
			"finding_domains":        []string{"security", "correctness"},
			"quoted_excerpt":         "context-isolated reviewer subagent",
			"artifact_location": map[string]any{
				"file_path":         "design.md",
				"section_reference": "Overview",
			},
			"status":    "hypothesized",
			"reasoning": "The design enforces structural isolation.",
		},
		"session_id":    sessionID,
		"finding_index": findingIndex,
	}

	schemaResult, schemaErr := handleValidateFindingSchema(context.Background(), schemaRequest)
	require.NoError(t, schemaErr)
	require.NotNil(t, schemaResult)

	schemaText := extractToolResultText(t, schemaResult)

	var schemaParsed map[string]any

	unmarshalSchemaErr := json.Unmarshal([]byte(schemaText), &schemaParsed)
	require.NoError(t, unmarshalSchemaErr)
	assert.Equal(t, true, schemaParsed["valid"])

	provenanceID, hasProvenance := schemaParsed["provenance_correlation_id"].(string)
	require.True(t, hasProvenance)
	assert.NotEmpty(t, provenanceID)

	// Step 4: Validate citation through the real MCP handler.
	citationRequest := mcp.CallToolRequest{}

	citationRequest.Params.Arguments = map[string]any{
		"quoted_excerpt":                   "context-isolated reviewer subagent",
		"file_path":                        "design.md",
		"section_reference":                "Overview",
		"session_id":                       sessionID,
		"finding_index":                    findingIndex,
		"schema_provenance_correlation_id": provenanceID,
	}

	citationResult, citationErr := handleValidateCitation(context.Background(), citationRequest)
	require.NoError(t, citationErr)
	require.NotNil(t, citationResult)

	citationText := extractToolResultText(t, citationResult)

	var citationParsed map[string]any

	unmarshalCitationErr := json.Unmarshal([]byte(citationText), &citationParsed)
	require.NoError(t, unmarshalCitationErr)
	assert.Equal(t, true, citationParsed["citation_valid"])
	assert.Equal(t, true, citationParsed["schema_valid"])

	citationProvenanceID, hasCitationProvenance := citationParsed["provenance_correlation_id"].(string)
	require.True(t, hasCitationProvenance)
	assert.NotEmpty(t, citationProvenanceID)

	// Step 5: Submit findings to round manager and advance through attack.
	finding := models.ReviewFinding{
		CriterionName:         "Path containment",
		CriterionSatisfaction: 8,
		FindingSeverity:       models.SeverityHigh,
		FindingDomains:        []models.FindingDomain{models.DomainSecurity, models.DomainCorrectness},
		QuotedExcerpt:         "context-isolated reviewer subagent",
		ArtifactLocation: models.ArtifactLocation{
			FilePath:         "design.md",
			SectionReference: "Overview",
		},
		Status:    models.StatusHypothesized,
		Reasoning: "The design enforces structural isolation.",
		CitationGateResult: &models.CitationGateResult{
			SchemaValid:             true,
			CitationValid:           true,
			ProvenanceCorrelationID: citationProvenanceID,
		},
	}

	rm.SubmitFindings([]models.ReviewFinding{finding})

	state := rm.AdvanceRound()
	assert.Equal(t, session.StateAttackRound, state)

	// Step 6: Submit an empty attack round (no novel findings).
	attackState := rm.SubmitAttackRound(nil)
	assert.Equal(t, session.StateApproved, attackState)

	// Step 7: Finalize through output.PersistSession.
	sessionOutput := &models.ReviewSessionOutput{
		Metadata: models.SessionMetadata{
			SpecName:                    "integration-full-workflow",
			ArtifactPath:                "design.md",
			Timestamp:                   time.Now().UTC().Format(time.RFC3339),
			TerminalStatus:              models.TerminalApproved,
			TaskExecutionID:             "task-exec-001",
			SessionID:                   sessionID,
			SelectionDecision:           models.SelectionSelected,
			SelectionReason:             trigger.ReasonBlastRadius,
			TriggeredBlastRadiusDomains: []string{"auth"},
			RoundsExecuted:              rm.RoundsExecuted(),
			Phase3Baseline:              models.Phase3BaselineAbsent,
			DegradedComponents:          dt.Entries(),
		},
		AttackRoundResult: &models.AttackRoundResult{
			NewIssuesFound: false,
		},
		Findings: rm.AllFindings(),
	}

	finalizeResult, finalizeErr := session.FinalizeReviewSession(
		&session.FinalizeReviewSessionInput{Session: sessionOutput},
	)
	require.NoError(t, finalizeErr)
	require.NotNil(t, finalizeResult)
	assert.Equal(t, models.TerminalApproved, finalizeResult.Session.Metadata.TerminalStatus)
	assert.NotEmpty(t, finalizeResult.IdempotencyKey)

	finalizeResult.Session.TerminalIdempotencyKey = finalizeResult.IdempotencyKey

	persistResult, persistErr := output.PersistSession(&output.PersistSessionInput{
		Session:       finalizeResult.Session,
		WorkspaceRoot: tmpDir,
	})
	require.NoError(t, persistErr)
	require.NotNil(t, persistResult)
	assert.True(t, persistResult.Created)
	assert.NotEmpty(t, persistResult.RecordPath)

	// Step 8: Verify one record created under .kiro/reviews/.
	reviewDir := filepath.Join(tmpDir, ".kiro", "reviews")
	entries, readDirErr := os.ReadDir(reviewDir)
	require.NoError(t, readDirErr)
	assert.Len(t, entries, 1)

	// Step 9: Verify artifact was NOT modified.
	afterContent, readAfterErr := os.ReadFile(artifactPath) // lint:allow_dynamic_filename
	require.NoError(t, readAfterErr)
	assert.Equal(t, integrationDesignContent, string(afterContent))
}

// TestIntegration_UnchangedArtifact_SkipsReview verifies that the unchanged
// path produces no review session and no review record.
func TestIntegration_UnchangedArtifact_SkipsReview(t *testing.T) {
	tmpDir := t.TempDir()

	t.Setenv("WORKSPACE_ROOT", tmpDir)

	// Simulate: artifact meets all skip conditions.
	classifyResult := trigger.Classify(&trigger.ClassifyInput{
		Artifact: &trigger.ArtifactInfo{
			Domain:                "",
			IsFoundational:        false,
			IsSingleFile:          true,
			IsRevertible:          true,
			IsHumanReviewedBefore: true,
		},
	})

	assert.Equal(t, trigger.DecisionSkip, classifyResult.Decision)
	assert.Equal(t, trigger.ReasonSkipConditions, classifyResult.Reason)
	assert.False(t, classifyResult.Ambiguous)

	// Verify: no review directory or record is created.
	reviewDir := filepath.Join(tmpDir, ".kiro", "reviews")
	_, statErr := os.Stat(reviewDir)
	assert.True(t, os.IsNotExist(statErr))
}

// TestIntegration_AmbiguousClassification_CreatesSession verifies that an
// artifact domain outside the blast-radius list that does not meet skip
// conditions triggers an ambiguous classification and starts a session.
func TestIntegration_AmbiguousClassification_CreatesSession(t *testing.T) {
	tmpDir := t.TempDir()

	t.Setenv("WORKSPACE_ROOT", tmpDir)

	workspaceRoot = tmpDir

	artifactPath := filepath.Join(tmpDir, "design.md")
	writeErr := os.WriteFile(artifactPath, []byte(integrationDesignContent), 0o0666)
	require.NoError(t, writeErr)

	// Domain "logging" is not in the blast-radius list and skip conditions
	// are not met.
	classifyResult := trigger.Classify(&trigger.ClassifyInput{
		Artifact: &trigger.ArtifactInfo{
			Domain:                "logging",
			IsFoundational:        false,
			IsSingleFile:          false,
			IsRevertible:          false,
			IsHumanReviewedBefore: false,
		},
	})

	assert.Equal(t, trigger.DecisionTrigger, classifyResult.Decision)
	assert.Equal(t, trigger.ReasonAmbiguous, classifyResult.Reason)
	assert.True(t, classifyResult.Ambiguous)

	// Session starts with ambiguous selection metadata.
	sessionID := "ambiguous-session-001"
	sessionOutput := &models.ReviewSessionOutput{
		Metadata: models.SessionMetadata{
			SpecName:           "ambiguous-test",
			ArtifactPath:       "design.md",
			Timestamp:          time.Now().UTC().Format(time.RFC3339),
			TerminalStatus:     models.TerminalNotApproved,
			TaskExecutionID:    "task-exec-ambiguous",
			SessionID:          sessionID,
			SelectionDecision:  models.SelectionAmbiguous,
			SelectionReason:    trigger.ReasonAmbiguous,
			SelectionAmbiguous: true,
			RoundsExecuted:     0,
			Phase3Baseline:     models.Phase3BaselineAbsent,
		},
		Findings: nil,
	}

	finalizeResult, finalizeErr := session.FinalizeReviewSession(
		&session.FinalizeReviewSessionInput{Session: sessionOutput},
	)
	require.NoError(t, finalizeErr)
	require.NotNil(t, finalizeResult)
	assert.Equal(t, models.TerminalNotApproved, finalizeResult.Session.Metadata.TerminalStatus)

	finalizeResult.Session.TerminalIdempotencyKey = finalizeResult.IdempotencyKey

	persistResult, persistErr := output.PersistSession(&output.PersistSessionInput{
		Session:       finalizeResult.Session,
		WorkspaceRoot: tmpDir,
	})
	require.NoError(t, persistErr)
	assert.True(t, persistResult.Created)

	// Verify record contents include ambiguous classification.
	recordContent, readErr := os.ReadFile(persistResult.RecordPath) // lint:allow_dynamic_filename
	require.NoError(t, readErr)
	assert.Contains(t, string(recordContent), "ambiguous")
}

// TestIntegration_RequiredComponentFailure_PartialReview verifies that a
// required component failure results in a partial_review terminal status with
// degradation details preserved.
func TestIntegration_RequiredComponentFailure_PartialReview(t *testing.T) {
	tmpDir := t.TempDir()

	t.Setenv("WORKSPACE_ROOT", tmpDir)

	workspaceRoot = tmpDir

	artifactPath := filepath.Join(tmpDir, "design.md")
	writeErr := os.WriteFile(artifactPath, []byte(integrationDesignContent), 0o0666)
	require.NoError(t, writeErr)

	// Session with a required component failure (citation gate unavailable).
	dt := session.NewDegradationTracker()
	dt.RecordFailure(&session.RecordFailureInput{
		Component:           "citation_gate",
		FailureMode:         "transport timeout",
		Criticality:         models.CriticalityRequired,
		UnavailableValueKey: "citation_result",
		AffectedCriteria:    []string{"Path containment", "Input validation"},
	})

	assert.True(t, dt.IsDegraded())
	assert.True(t, dt.HasRequiredFailure())

	// AllowedTerminalStatus downgrades proposed approval to partial_review.
	allowedStatus := dt.AllowedTerminalStatus(models.TerminalApproved)
	assert.Equal(t, models.TerminalPartialReview, allowedStatus)

	sessionID := "partial-review-session-001"
	sessionOutput := &models.ReviewSessionOutput{
		Metadata: models.SessionMetadata{
			SpecName:           "partial-review-test",
			ArtifactPath:       "design.md",
			Timestamp:          time.Now().UTC().Format(time.RFC3339),
			TerminalStatus:     models.TerminalPartialReview,
			TaskExecutionID:    "task-exec-partial",
			SessionID:          sessionID,
			SelectionDecision:  models.SelectionSelected,
			SelectionReason:    trigger.ReasonBlastRadius,
			RoundsExecuted:     1,
			Phase3Baseline:     models.Phase3BaselineAbsent,
			DegradedComponents: dt.Entries(),
		},
		Findings: nil,
		UnavailableValues: []models.UnavailableValue{
			{Field: "citation_result", Reason: "citation gate transport timeout"},
		},
	}

	finalizeResult, finalizeErr := session.FinalizeReviewSession(
		&session.FinalizeReviewSessionInput{Session: sessionOutput},
	)
	require.NoError(t, finalizeErr)
	require.NotNil(t, finalizeResult)
	assert.Equal(t, models.TerminalPartialReview, finalizeResult.Session.Metadata.TerminalStatus)

	finalizeResult.Session.TerminalIdempotencyKey = finalizeResult.IdempotencyKey

	persistResult, persistErr := output.PersistSession(&output.PersistSessionInput{
		Session:       finalizeResult.Session,
		WorkspaceRoot: tmpDir,
	})
	require.NoError(t, persistErr)
	assert.True(t, persistResult.Created)

	// Verify record includes degradation details and unavailable values.
	recordContent, readErr := os.ReadFile(persistResult.RecordPath) // lint:allow_dynamic_filename
	require.NoError(t, readErr)

	content := string(recordContent)
	assert.Contains(t, content, "citation_gate")
	assert.Contains(t, content, "transport timeout")
	assert.Contains(t, content, "required")
	assert.Contains(t, content, "partial_review")
	assert.Contains(t, content, "Unavailable Values")
	assert.Contains(t, content, "citation_result")
}

// TestIntegration_HumanOverride_AdvisoryAndAuditable verifies that a human
// override produces human_override terminal status with all degradation and
// override details preserved.
func TestIntegration_HumanOverride_AdvisoryAndAuditable(t *testing.T) {
	tmpDir := t.TempDir()

	t.Setenv("WORKSPACE_ROOT", tmpDir)

	workspaceRoot = tmpDir

	artifactPath := filepath.Join(tmpDir, "design.md")
	writeErr := os.WriteFile(artifactPath, []byte(integrationDesignContent), 0o0666)
	require.NoError(t, writeErr)

	// Session with degradation and a human override.
	dt := session.NewDegradationTracker()
	dt.RecordFailure(&session.RecordFailureInput{
		Component:        "reviewer",
		FailureMode:      "invocation timeout",
		Criticality:      models.CriticalityRequired,
		AffectedCriteria: []string{"Security boundary"},
	})

	sessionID := "override-session-001"
	sessionOutput := &models.ReviewSessionOutput{
		Metadata: models.SessionMetadata{
			SpecName:           "override-test",
			ArtifactPath:       "design.md",
			Timestamp:          time.Now().UTC().Format(time.RFC3339),
			TerminalStatus:     models.TerminalHumanOverride,
			TaskExecutionID:    "task-exec-override",
			SessionID:          sessionID,
			SelectionDecision:  models.SelectionSelected,
			SelectionReason:    trigger.ReasonBlastRadius,
			RoundsExecuted:     0,
			Phase3Baseline:     models.Phase3BaselineAbsent,
			DegradedComponents: dt.Entries(),
		},
		Findings: nil,
		HumanOverrides: []models.HumanOverride{
			{
				CriterionName:                 "Security boundary",
				Decision:                      models.HumanDecisionOverride,
				HumanRationale:                "Reviewed manually; risk is acceptable for this iteration.",
				FindingIndex:                  0,
				OriginalCriterionSatisfaction: 4,
			},
		},
	}

	finalizeResult, finalizeErr := session.FinalizeReviewSession(
		&session.FinalizeReviewSessionInput{Session: sessionOutput},
	)
	require.NoError(t, finalizeErr)
	require.NotNil(t, finalizeResult)
	assert.Equal(t, models.TerminalHumanOverride, finalizeResult.Session.Metadata.TerminalStatus)

	finalizeResult.Session.TerminalIdempotencyKey = finalizeResult.IdempotencyKey

	persistResult, persistErr := output.PersistSession(&output.PersistSessionInput{
		Session:       finalizeResult.Session,
		WorkspaceRoot: tmpDir,
	})
	require.NoError(t, persistErr)
	assert.True(t, persistResult.Created)

	// Verify record includes override details and degradation is preserved.
	recordContent, readErr := os.ReadFile(persistResult.RecordPath) // lint:allow_dynamic_filename
	require.NoError(t, readErr)

	content := string(recordContent)
	assert.Contains(t, content, "human_override")
	assert.Contains(t, content, "Human Overrides")
	assert.Contains(t, content, "Reviewed manually; risk is acceptable for this iteration.")
	assert.Contains(t, content, "Security boundary")
	assert.Contains(t, content, "reviewer")
	assert.Contains(t, content, "invocation timeout")
}

// TestIntegration_ArtifactNotModified verifies that a complete workflow
// execution never modifies the design artifact content.
func TestIntegration_ArtifactNotModified(t *testing.T) {
	tmpDir := t.TempDir()

	t.Setenv("WORKSPACE_ROOT", tmpDir)

	workspaceRoot = tmpDir

	artifactPath := filepath.Join(tmpDir, "design.md")
	writeErr := os.WriteFile(artifactPath, []byte(integrationDesignContent), 0o0666)
	require.NoError(t, writeErr)

	// Read the artifact content before the workflow.
	beforeContent, readBeforeErr := os.ReadFile(artifactPath) // lint:allow_dynamic_filename
	require.NoError(t, readBeforeErr)

	// Exercise: schema validation, citation validation, and finalization.
	schemaRequest := mcp.CallToolRequest{}

	schemaRequest.Params.Arguments = map[string]any{
		"finding": map[string]any{
			"criterion_name":         "Artifact immutability",
			"criterion_satisfaction": 9,
			"finding_severity":       "medium",
			"finding_domains":        []string{"architecture"},
			"quoted_excerpt":         "short-lived tokens with automatic rotation",
			"artifact_location": map[string]any{
				"file_path":         "design.md",
				"section_reference": "Security",
			},
			"status":    "hypothesized",
			"reasoning": "Artifact must not be modified by the workflow.",
		},
		"session_id":    "immutability-session",
		"finding_index": 0,
	}

	schemaResult, schemaErr := handleValidateFindingSchema(context.Background(), schemaRequest)
	require.NoError(t, schemaErr)

	schemaText := extractToolResultText(t, schemaResult)

	var schemaParsed map[string]any

	unmarshalErr := json.Unmarshal([]byte(schemaText), &schemaParsed)
	require.NoError(t, unmarshalErr)

	provenanceID, hasID := schemaParsed["provenance_correlation_id"].(string)
	require.True(t, hasID)

	citationRequest := mcp.CallToolRequest{}

	citationRequest.Params.Arguments = map[string]any{
		"quoted_excerpt":                   "short-lived tokens with automatic rotation",
		"file_path":                        "design.md",
		"section_reference":                "Security",
		"session_id":                       "immutability-session",
		"finding_index":                    0,
		"schema_provenance_correlation_id": provenanceID,
	}

	_, citationErr := handleValidateCitation(context.Background(), citationRequest)
	require.NoError(t, citationErr)

	// Read the artifact content after the workflow.
	afterContent, readAfterErr := os.ReadFile(artifactPath) // lint:allow_dynamic_filename
	require.NoError(t, readAfterErr)

	// Byte-for-byte comparison: artifact must be unchanged.
	assert.Equal(t, beforeContent, afterContent)
}
