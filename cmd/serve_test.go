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
)

type finalizationToolResponse struct {
	IdempotencyKey string `json:"idempotency_key"` // lint:allow_format
	RecordPath     string `json:"record_path"`     // lint:allow_format
	TerminalStatus string `json:"terminal_status"` // lint:allow_format
}

// TestMCPHandlerFinalizationPersistsOneContainedRecord verifies that the served
// finalizer writes only under the temporary workspace and returns the original
// record for an idempotent retry.
func TestMCPHandlerFinalizationPersistsOneContainedRecord(t *testing.T) {
	workspace := t.TempDir()
	setTestWorkspaceRoot(t, workspace)

	writeErr := os.WriteFile(
		filepath.Join(workspace, "design.md"),
		[]byte(sampleDesignContent),
		0o0666,
	) // lint:allow_666
	require.NoError(t, writeErr)

	finding := canonicalFindingPayload("context-isolated reviewer subagent")
	citationResponse := canonicalCitationProvenance(t, "terminal-session", finding)
	session := terminalSessionPayload(&terminalSessionPayloadInput{
		CitationResponse: citationResponse,
		Finding:          finding,
		SessionID:        "terminal-session",
		SpecName:         "contained-spec",
		TaskExecutionID:  "terminal-task",
	})

	first := finalizeMCPReviewSession(t, session)
	second := finalizeMCPReviewSession(t, session)

	reviewDirectory := filepath.Join(workspace, ".kiro", "reviews")
	relativeRecordPath, relativeErr := filepath.Rel(workspace, first.RecordPath)
	require.NoError(t, relativeErr)

	assert.Equal(t, "approved", first.TerminalStatus)
	assert.NotEmpty(t, first.IdempotencyKey)
	assert.Equal(t, first.RecordPath, second.RecordPath)
	assert.True(t, first.RecordPath == filepath.Join(reviewDirectory, filepath.Base(first.RecordPath)))
	assert.NotEqual(t, "..", relativeRecordPath)
	assert.NotContains(t, relativeRecordPath, ".."+string(filepath.Separator))

	records, readDirErr := os.ReadDir(reviewDirectory)
	require.NoError(t, readDirErr)
	assert.Len(t, records, 1)
	assert.FileExists(t, first.RecordPath)
}

// TestMCPHandlerFinalizationRejectsEscapingRecordPath verifies that a caller
// cannot use a spec name to write a terminal record outside .kiro/reviews.
func TestMCPHandlerFinalizationRejectsEscapingRecordPath(t *testing.T) {
	workspace := t.TempDir()
	setTestWorkspaceRoot(t, workspace)

	writeErr := os.WriteFile(
		filepath.Join(workspace, "design.md"),
		[]byte(sampleDesignContent),
		0o0666,
	) // lint:allow_666
	require.NoError(t, writeErr)

	finding := canonicalFindingPayload("context-isolated reviewer subagent")
	citationResponse := canonicalCitationProvenance(t, "escape-session", finding)
	session := terminalSessionPayload(&terminalSessionPayloadInput{
		CitationResponse: citationResponse,
		Finding:          finding,
		SessionID:        "escape-session",
		SpecName:         "../outside",
		TaskExecutionID:  "escape-task",
	})
	request := mcp.CallToolRequest{}

	request.Params.Arguments = map[string]any{"session": session}

	_, handleErr := handleFinalizeReviewSession(context.Background(), request)

	assert.ErrorIs(t, handleErr, ErrToolExecution)

	reviewDirectory := filepath.Join(workspace, ".kiro", "reviews")

	entries, readDirErr := os.ReadDir(reviewDirectory)
	if os.IsNotExist(readDirErr) {
		return
	}

	require.NoError(t, readDirErr)
	assert.Empty(t, entries)
}

func finalizeMCPReviewSession(t *testing.T, session map[string]any) finalizationToolResponse {
	t.Helper()

	request := mcp.CallToolRequest{}

	request.Params.Arguments = map[string]any{"session": session}

	result, handleErr := handleFinalizeReviewSession(context.Background(), request)
	require.NoError(t, handleErr)

	var response finalizationToolResponse

	unmarshalErr := json.Unmarshal([]byte(extractToolResultText(t, result)), &response)
	require.NoError(t, unmarshalErr)

	return response
}

func terminalSessionPayload(input *terminalSessionPayloadInput) map[string]any {
	terminalFinding := make(map[string]any, len(input.Finding)+1)
	maps.Copy(terminalFinding, input.Finding)

	terminalFinding["citation_gate_result"] = map[string]any{
		"provenance_correlation_id": input.CitationResponse.ProvenanceCorrelationID,
	}

	return map[string]any{
		"attack_round_result": map[string]any{
			"new_issues_found": false,
		},
		"findings": []any{terminalFinding},
		"metadata": map[string]any{
			"artifact_path":     "design.md",
			"session_id":        input.SessionID,
			"spec_name":         input.SpecName,
			"task_execution_id": input.TaskExecutionID,
			"terminal_status":   "approved",
			"timestamp":         "2026-08-13T10:00:00Z",
		},
	}
}
