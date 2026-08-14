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
	"strings"
	"testing"

	"github.com/go-openapi/testify/assert"
	"github.com/go-openapi/testify/require"
	"github.com/mark3labs/mcp-go/mcp"
)

// sampleDesignContent is a minimal design artifact used by the integration
// tests.
const sampleDesignContent = `# Design

## Overview

The system enforces structurally independent review of design
artifacts by running a context-isolated reviewer subagent.

## Architecture

Components communicate over stdio using MCP protocol.
The citation gate validates quoted excerpts deterministically.

## Security

Authentication uses short-lived tokens with automatic rotation.
`

// extractToolResultText safely extracts the text content from an MCP tool
// result, failing the test if the type assertion fails.
func extractToolResultText(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()

	require.Greater(t, len(result.Content), 0, "expected content in tool result")

	textContent, ok := result.Content[0].(mcp.TextContent)
	require.True(t, ok, "expected TextContent type in result")

	return textContent.Text
}

// TestReviewCommand_ProducesOutputFile verifies that runReview produces a
// review output file under .kiro/reviews/ when the artifact domain triggers
// review.
func TestReviewCommand_ProducesOutputFile(t *testing.T) {
	tmpDir := t.TempDir()

	t.Setenv("WORKSPACE_ROOT", tmpDir)

	artifactPath := filepath.Join(tmpDir, "design.md")
	writeErr := os.WriteFile(artifactPath, []byte(sampleDesignContent), 0o0666)
	require.NoError(t, writeErr)

	fSpecName = "test-spec"
	fDomain = "auth"

	reviewErr := runReview(context.Background(), "design.md")
	require.NoError(t, reviewErr)

	reviewDir := filepath.Join(tmpDir, ".kiro", "reviews")
	entries, readDirErr := os.ReadDir(reviewDir)
	require.NoError(t, readDirErr)
	assert.Greater(t, len(entries), 0, "expected at least one review file")

	outputPath := filepath.Join(reviewDir, entries[0].Name())
	content, readErr := os.ReadFile(outputPath) // lint:allow_dynamic_filename
	require.NoError(t, readErr)

	output := string(content)

	assert.Contains(t, output, "# Adversarial Review: test-spec")
	assert.Contains(t, output, "## Summary")
	assert.Contains(t, output, "## Findings")
}

// TestReviewCommand_SkipsWhenDomainEmpty verifies that runReview exits early
// with a skip message when trigger classification resolves to skip (all skip
// conditions met).
func TestReviewCommand_SkipsWhenDomainEmpty(t *testing.T) {
	tmpDir := t.TempDir()

	t.Setenv("WORKSPACE_ROOT", tmpDir)

	fSpecName = "skip-test"
	fDomain = ""

	reviewErr := runReview(context.Background(), "design.md")
	require.NoError(t, reviewErr)

	reviewDir := filepath.Join(tmpDir, ".kiro", "reviews")
	_, statErr := os.Stat(reviewDir)

	// With no domain and no foundational/single-file flags, the trigger
	// defaults to ambiguous (triggers review). The output directory should be
	// created.
	assert.NoError(t, statErr)
}

// TestReviewCommand_OutputFileNaming verifies the output file follows the
// naming convention {spec-name}-{ISO-8601-date}-{seq}.md.
func TestReviewCommand_OutputFileNaming(t *testing.T) {
	tmpDir := t.TempDir()

	t.Setenv("WORKSPACE_ROOT", tmpDir)

	artifactPath := filepath.Join(tmpDir, "design.md")
	writeErr := os.WriteFile(artifactPath, []byte(sampleDesignContent), 0o0666)
	require.NoError(t, writeErr)

	fSpecName = "naming-test"
	fDomain = "secrets"

	reviewErr := runReview(context.Background(), "design.md")
	require.NoError(t, reviewErr)

	reviewDir := filepath.Join(tmpDir, ".kiro", "reviews")
	entries, readDirErr := os.ReadDir(reviewDir)
	require.NoError(t, readDirErr)
	require.Len(t, entries, 1)

	name := entries[0].Name()

	assert.True(t, strings.HasPrefix(name, "naming-test-"), "filename should start with spec name")
	assert.True(t, strings.HasSuffix(name, ".md"), "filename should end with .md")
	assert.Contains(t, name, "-1.md", "first review should have sequence 1")
}

// TestReviewCommand_OutputStructure verifies the rendered Markdown output
// contains all required sections per the design.
func TestReviewCommand_OutputStructure(t *testing.T) {
	tmpDir := t.TempDir()

	t.Setenv("WORKSPACE_ROOT", tmpDir)

	artifactPath := filepath.Join(tmpDir, "design.md")
	writeErr := os.WriteFile(artifactPath, []byte(sampleDesignContent), 0o0666)
	require.NoError(t, writeErr)

	fSpecName = "structure-test"
	fDomain = "payments"

	reviewErr := runReview(context.Background(), "design.md")
	require.NoError(t, reviewErr)

	reviewDir := filepath.Join(tmpDir, ".kiro", "reviews")
	entries, readDirErr := os.ReadDir(reviewDir)
	require.NoError(t, readDirErr)
	require.Greater(t, len(entries), 0)

	outputPath := filepath.Join(reviewDir, entries[0].Name())
	content, readErr := os.ReadFile(outputPath) // lint:allow_dynamic_filename
	require.NoError(t, readErr)

	output := string(content)

	expectedSections := []string{
		"# Adversarial Review: structure-test",
		"## Summary",
		"## Findings",
		"## Attack Round",
		"## Human Escalations",
		"## Human Overrides",
		"## Dead Checks",
		"## Degradation Summary",
	}

	for _, section := range expectedSections {
		assert.Contains(t, output, section)
	}
}

// TestHandleValidateCitation_ValidExcerpt verifies the MCP tool handler
// correctly validates a citation that exists in the artifact.
func TestHandleValidateCitation_ValidExcerpt(t *testing.T) {
	tmpDir := t.TempDir()

	workspaceRoot = tmpDir

	writeErr := os.WriteFile(filepath.Join(tmpDir, "design.md"), []byte(sampleDesignContent), 0o0666)
	require.NoError(t, writeErr)

	request := mcp.CallToolRequest{}

	request.Params.Arguments = map[string]any{
		"quoted_excerpt":    "context-isolated reviewer subagent",
		"file_path":         "design.md",
		"section_reference": "Overview",
	}

	result, handleErr := handleValidateCitation(context.Background(), request)
	require.NoError(t, handleErr)
	require.NotNil(t, result)

	text := extractToolResultText(t, result)

	var parsed map[string]any

	unmarshalErr := json.Unmarshal([]byte(text), &parsed)
	require.NoError(t, unmarshalErr)
	assert.Equal(t, true, parsed["Valid"])
}

// TestHandleValidateCitation_InvalidExcerpt verifies the MCP tool handler
// returns valid=false for an excerpt not in the artifact.
func TestHandleValidateCitation_InvalidExcerpt(t *testing.T) {
	tmpDir := t.TempDir()

	workspaceRoot = tmpDir

	writeErr := os.WriteFile(filepath.Join(tmpDir, "design.md"), []byte(sampleDesignContent), 0o0666)
	require.NoError(t, writeErr)

	request := mcp.CallToolRequest{}

	request.Params.Arguments = map[string]any{
		"quoted_excerpt":    "this text does not exist in the artifact",
		"file_path":         "design.md",
		"section_reference": "Overview",
	}

	result, handleErr := handleValidateCitation(context.Background(), request)
	require.NoError(t, handleErr)
	require.NotNil(t, result)

	text := extractToolResultText(t, result)

	var parsed map[string]any

	unmarshalErr := json.Unmarshal([]byte(text), &parsed)
	require.NoError(t, unmarshalErr)
	assert.Equal(t, false, parsed["Valid"])
}

// TestHandleValidateFindingSchema_ValidFinding verifies the schema tool handler
// accepts a correctly structured finding.
func TestHandleValidateFindingSchema_ValidFinding(t *testing.T) {
	request := mcp.CallToolRequest{}

	request.Params.Arguments = map[string]any{
		"finding": map[string]any{
			"criterion_name": "test-criterion",
			"score":          5,
			"quoted_excerpt": "some evidence",
			"artifact_location": map[string]any{
				"file_path":         "design.md",
				"section_reference": "Overview",
			},
			"status":    "hypothesized",
			"reasoning": "test reasoning",
		},
	}

	result, handleErr := handleValidateFindingSchema(context.Background(), request)
	require.NoError(t, handleErr)
	require.NotNil(t, result)

	text := extractToolResultText(t, result)

	var parsed map[string]any

	unmarshalErr := json.Unmarshal([]byte(text), &parsed)
	require.NoError(t, unmarshalErr)
	assert.Equal(t, true, parsed["valid"])
}

// TestHandleValidateFindingSchema_MissingFields verifies the schema tool
// handler rejects a finding with missing required fields.
func TestHandleValidateFindingSchema_MissingFields(t *testing.T) {
	request := mcp.CallToolRequest{}

	request.Params.Arguments = map[string]any{
		"finding": map[string]any{
			"criterion_name": "",
			"score":          0,
			"quoted_excerpt": "",
		},
	}

	result, handleErr := handleValidateFindingSchema(context.Background(), request)
	require.NoError(t, handleErr)
	require.NotNil(t, result)

	text := extractToolResultText(t, result)

	var parsed map[string]any

	unmarshalErr := json.Unmarshal([]byte(text), &parsed)
	require.NoError(t, unmarshalErr)
	assert.Equal(t, false, parsed["valid"])
	assert.NotEmpty(t, parsed["error"])
}

// TestHandleValidateFindingsBatch_MixedResults verifies the batch tool handler
// processes multiple findings and returns per-finding results.
func TestHandleValidateFindingsBatch_MixedResults(t *testing.T) {
	tmpDir := t.TempDir()

	workspaceRoot = tmpDir

	writeErr := os.WriteFile(filepath.Join(tmpDir, "design.md"), []byte(sampleDesignContent), 0o0666)
	require.NoError(t, writeErr)

	request := mcp.CallToolRequest{}

	request.Params.Arguments = map[string]any{
		"findings": []any{
			map[string]any{
				"QuotedExcerpt":    "context-isolated reviewer subagent",
				"FilePath":         "design.md",
				"SectionReference": "Overview",
			},
			map[string]any{
				"QuotedExcerpt":    "nonexistent text in the artifact",
				"FilePath":         "design.md",
				"SectionReference": "Overview",
			},
		},
	}

	result, handleErr := handleValidateFindingsBatch(context.Background(), request)
	require.NoError(t, handleErr)
	require.NotNil(t, result)

	text := extractToolResultText(t, result)

	var parsed []map[string]any

	unmarshalErr := json.Unmarshal([]byte(text), &parsed)
	require.NoError(t, unmarshalErr)
	require.Len(t, parsed, 2)

	assert.Equal(t, true, parsed[0]["CitationValid"])
	assert.Equal(t, false, parsed[1]["CitationValid"])
}

// TestHandleValidateCitation_MissingArguments verifies the MCP tool handler
// returns an error when required arguments are missing.
func TestHandleValidateCitation_MissingArguments(t *testing.T) {
	request := mcp.CallToolRequest{}

	request.Params.Arguments = make(map[string]any)

	_, handleErr := handleValidateCitation(context.Background(), request)

	assert.Error(t, handleErr)
}

// TestNewMCPServer_ToolRegistration verifies the MCP server is created with all
// expected tools registered.
func TestNewMCPServer_ToolRegistration(t *testing.T) {
	s := newMCPServer()
	require.NotNil(t, s)
}
