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
	"maps"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-openapi/testify/assert"
	"github.com/go-openapi/testify/require"
	"github.com/mark3labs/mcp-go/mcp"

	"go.nwlabs.dev/magneto/internal/models"
)

type (
	schemaToolResponse struct {
		ProvenanceCorrelationID string               `json:"provenance_correlation_id"` // lint:allow_format
		NormalizedFinding       models.ReviewFinding `json:"normalized_finding"`        // lint:allow_format
		Valid                   bool                 `json:"valid"`
	}

	canonicalBatchToolResponse struct {
		ProvenanceCorrelationID string `json:"provenance_correlation_id"` // lint:allow_format
		FindingIndex            int    `json:"FindingIndex"`              // lint:allow_format
		CitationValid           bool   `json:"CitationValid"`             // lint:allow_format
		SchemaValid             bool   `json:"schema_valid"`              // lint:allow_format
	}

	terminalSessionPayloadInput struct {
		Finding          map[string]any
		SessionID        string
		SpecName         string
		TaskExecutionID  string
		CitationResponse canonicalBatchToolResponse
	}
)

// TestMCPHandlerSchemaNormalization accepts both canonical and legacy inputs
// while returning only canonical normalized findings.
func TestMCPHandlerSchemaNormalization(t *testing.T) {
	tests := []struct {
		configure   func(map[string]any)
		name        string
		expectation int
	}{
		{
			name:        "canonical satisfaction input",
			expectation: 8,
			configure: func(finding map[string]any) {
				finding["criterion_satisfaction"] = 8
			},
		},
		{
			name:        "legacy score input is normalized and clamped",
			expectation: 10,
			configure: func(finding map[string]any) {
				delete(finding, "criterion_satisfaction")

				finding["score"] = 11
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			finding := canonicalFindingPayload("context-isolated reviewer subagent")
			tt.configure(finding)

			response := validateCanonicalSchema(t, "schema-normalization", 0, finding)

			assert.True(t, response.Valid)
			assert.Equal(t, tt.expectation, response.NormalizedFinding.CriterionSatisfaction)
			assert.Empty(t, response.NormalizedFinding.CitationGateResult)
		})
	}
}

// TestMCPHandlerCanonicalBatchPreservesInputIndices verifies that batch
// responses remain index-aligned even when a later citation cannot be found.
func TestMCPHandlerCanonicalBatchPreservesInputIndices(t *testing.T) {
	workspace := t.TempDir()
	setTestWorkspaceRoot(t, workspace)

	writeErr := os.WriteFile(
		filepath.Join(workspace, "design.md"),
		[]byte(sampleDesignContent),
		0o0666,
	) // lint:allow_666
	require.NoError(t, writeErr)

	validFinding := canonicalFindingPayload("context-isolated reviewer subagent")
	invalidFinding := canonicalFindingPayload("excerpt absent from the design artifact")
	sessionID := "batch-alignment"

	validSchema := validateCanonicalSchema(t, sessionID, 0, validFinding)
	invalidSchema := validateCanonicalSchema(t, sessionID, 1, invalidFinding)

	request := mcp.CallToolRequest{}

	request.Params.Arguments = map[string]any{
		"findings": []any{
			canonicalBatchFinding(validFinding, validSchema.ProvenanceCorrelationID, 0),
			canonicalBatchFinding(invalidFinding, invalidSchema.ProvenanceCorrelationID, 1),
		},
		"finding_index": 0,
		"session_id":    sessionID,
	}

	result, handleErr := handleValidateFindingsBatch(context.Background(), request)
	require.NoError(t, handleErr)

	var responses []canonicalBatchToolResponse

	unmarshalErr := json.Unmarshal([]byte(extractToolResultText(t, result)), &responses)
	require.NoError(t, unmarshalErr)
	require.Len(t, responses, 2)
	assert.Equal(t, 0, responses[0].FindingIndex)
	assert.True(t, responses[0].SchemaValid)
	assert.True(t, responses[0].CitationValid)
	assert.NotEmpty(t, responses[0].ProvenanceCorrelationID)
	assert.Equal(t, 1, responses[1].FindingIndex)
	assert.True(t, responses[1].SchemaValid)
	assert.False(t, responses[1].CitationValid)
	assert.NotEmpty(t, responses[1].ProvenanceCorrelationID)
}

// TestMCPHandlerRejectsMismatchedSchemaProvenance prevents a schema result from
// being replayed with a changed finding.
func TestMCPHandlerRejectsMismatchedSchemaProvenance(t *testing.T) {
	workspace := t.TempDir()
	setTestWorkspaceRoot(t, workspace)

	writeErr := os.WriteFile(
		filepath.Join(workspace, "design.md"),
		[]byte(sampleDesignContent),
		0o0666,
	) // lint:allow_666
	require.NoError(t, writeErr)

	finding := canonicalFindingPayload("context-isolated reviewer subagent")
	schemaResponse := validateCanonicalSchema(t, "provenance-rejection", 0, finding)
	changedFinding := canonicalFindingPayload("Components communicate over stdio using MCP protocol.")

	request := mcp.CallToolRequest{}

	request.Params.Arguments = map[string]any{
		"findings": []any{
			canonicalBatchFinding(changedFinding, schemaResponse.ProvenanceCorrelationID, 0),
		},
		"finding_index": 0,
		"session_id":    "provenance-rejection",
	}

	_, handleErr := handleValidateFindingsBatch(context.Background(), request)

	assert.ErrorIs(t, handleErr, ErrValidationProvenanceMismatch)
}

// TestMCPHandlerRejectsNonterminalSessionStatus prevents persistence of session
// data before the coordinator has reached a supported terminal status.
func TestMCPHandlerRejectsNonterminalSessionStatus(t *testing.T) {
	request := mcp.CallToolRequest{}

	request.Params.Arguments = map[string]any{
		"session": map[string]any{
			"findings": nil,
			"metadata": map[string]any{
				"session_id":        "pending-session",
				"task_execution_id": "pending-task",
				"terminal_status":   "pending",
			},
		},
	}

	_, handleErr := handleFinalizeReviewSession(context.Background(), request)

	assert.ErrorIs(t, handleErr, ErrToolInputInvalid)
	assert.Error(t, handleErr)
	assert.Contains(t, handleErr.Error(), "terminal status is not supported")
}

func canonicalFindingPayload(quotedExcerpt string) map[string]any {
	return map[string]any{
		"artifact_location": map[string]any{
			"file_path":         "design.md",
			"section_reference": "Overview",
		},
		"criterion_name":         "Read-only review boundary",
		"criterion_satisfaction": 8,
		"finding_domains":        []string{"security", "correctness"},
		"finding_severity":       "high",
		"quoted_excerpt":         quotedExcerpt,
		"reasoning":              "The artifact explicitly specifies a deterministic boundary.",
		"status":                 "hypothesized",
	}
}

func canonicalBatchFinding(finding map[string]any, schemaID string, index int) map[string]any {
	batchFinding := make(map[string]any, len(finding)+2)
	maps.Copy(batchFinding, finding)

	batchFinding["finding_index"] = index
	batchFinding["schema_provenance_correlation_id"] = schemaID

	return batchFinding
}

func canonicalCitationProvenance(t *testing.T, sessionID string, finding map[string]any) canonicalBatchToolResponse {
	t.Helper()

	schemaResponse := validateCanonicalSchema(t, sessionID, 0, finding)
	request := mcp.CallToolRequest{}

	request.Params.Arguments = map[string]any{
		"findings": []any{
			canonicalBatchFinding(finding, schemaResponse.ProvenanceCorrelationID, 0),
		},
		"finding_index": 0,
		"session_id":    sessionID,
	}

	result, handleErr := handleValidateFindingsBatch(context.Background(), request)
	require.NoError(t, handleErr)

	var responses []canonicalBatchToolResponse

	unmarshalErr := json.Unmarshal([]byte(extractToolResultText(t, result)), &responses)
	require.NoError(t, unmarshalErr)
	require.Len(t, responses, 1)
	require.True(t, responses[0].CitationValid)

	return responses[0]
}

func setTestWorkspaceRoot(t *testing.T, root string) {
	t.Helper()

	previousRoot := workspaceRoot

	workspaceRoot = root

	t.Cleanup(func() {
		workspaceRoot = previousRoot
	})
}

func validateCanonicalSchema(
	t *testing.T,
	sessionID string,
	findingIndex int,
	finding map[string]any,
) schemaToolResponse {
	t.Helper()

	request := mcp.CallToolRequest{}

	request.Params.Arguments = map[string]any{
		"finding":       finding,
		"finding_index": findingIndex,
		"session_id":    sessionID,
	}

	result, handleErr := handleValidateFindingSchema(context.Background(), request)
	require.NoError(t, handleErr)

	var response schemaToolResponse

	unmarshalErr := json.Unmarshal([]byte(extractToolResultText(t, result)), &response)
	require.NoError(t, unmarshalErr)
	require.True(t, response.Valid)
	require.NotEmpty(t, response.ProvenanceCorrelationID)

	return response
}

func TestMCPHandlerRejectsForgedCitationProvenance(t *testing.T) {
	workspace := t.TempDir()
	setTestWorkspaceRoot(t, workspace)

	writeErr := os.WriteFile(
		filepath.Join(workspace, "design.md"),
		[]byte(sampleDesignContent),
		0o0666,
	) // lint:allow_666
	require.NoError(t, writeErr)

	finding := canonicalFindingPayload("context-isolated reviewer subagent")
	request := mcp.CallToolRequest{}

	request.Params.Arguments = map[string]any{
		"file_path":                        "design.md",
		"finding_index":                    0,
		"quoted_excerpt":                   finding["quoted_excerpt"],
		"schema_provenance_correlation_id": "not-issued-by-magneto",
		"section_reference":                "Overview",
		"session_id":                       "forged-provenance",
	}

	_, handleErr := handleValidateCitation(context.Background(), request)

	assert.ErrorIs(t, handleErr, ErrValidationProvenanceMismatch)
}
