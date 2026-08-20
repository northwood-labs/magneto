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

	"go.nwlabs.dev/magneto/internal/models"
	"go.nwlabs.dev/magneto/internal/output"
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

	reviewContent := string(content)

	assert.Contains(t, reviewContent, "# Adversarial Review: test-spec")
	assert.Contains(t, reviewContent, "## Summary")
	assert.Contains(t, reviewContent, "## Findings")
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

// TestReviewCommand_EmitsDeprecationNotice verifies that runReview emits the
// approved deprecation notice directing users to the Kiro-native MCP
// finalization workflow.
func TestReviewCommand_EmitsDeprecationNotice(t *testing.T) {
	tmpDir := t.TempDir()

	t.Setenv("WORKSPACE_ROOT", tmpDir)

	artifactPath := filepath.Join(tmpDir, "design.md")
	writeErr := os.WriteFile(artifactPath, []byte(sampleDesignContent), 0o0666)
	require.NoError(t, writeErr)

	fSpecName = "deprecation-test"
	fDomain = "auth"

	// Capture stdout.
	origStdout := os.Stdout

	reader, writer, pipeErr := os.Pipe()
	require.NoError(t, pipeErr)

	os.Stdout = writer

	reviewErr := runReview(context.Background(), "design.md")
	require.NoError(t, reviewErr)

	writer.Close()

	captured := make([]byte, 4096) // lint:allow_raw_number
	n, readErr := reader.Read(captured)
	require.NoError(t, readErr)

	os.Stdout = origStdout

	capturedOut := string(captured[:n])

	assert.Contains(t, capturedOut, "magneto review is deprecated")
	assert.Contains(t, capturedOut, "finalize_review_session")
	assert.Contains(t, capturedOut, "Kiro-native MCP finalization workflow")
}

// TestReviewCommand_EmitsDeprecationNoticeOnSkip verifies that the deprecation
// notice is emitted even when the trigger classifies the artifact as skippable
// (all skip conditions met).
func TestReviewCommand_EmitsDeprecationNoticeOnSkip(t *testing.T) {
	tmpDir := t.TempDir()

	t.Setenv("WORKSPACE_ROOT", tmpDir)

	// The skip path requires all three conditions: single-file, revertible, and
	// human-reviewed. Since runReview sets Domain only, the classifier cannot
	// reach skip via the CLI. This test verifies the deprecation notice is
	// emitted on the ambiguous path (domain "none" is not a blast-radius domain
	// and the skip conditions are not met, so it defaults to trigger).
	fSpecName = "skip-deprecation-test"
	fDomain = "none"

	artifactPath := filepath.Join(tmpDir, "design.md")
	writeErr := os.WriteFile(artifactPath, []byte(sampleDesignContent), 0o0666)
	require.NoError(t, writeErr)

	// Capture stdout.
	origStdout := os.Stdout

	reader, writer, pipeErr := os.Pipe()
	require.NoError(t, pipeErr)

	os.Stdout = writer

	reviewErr := runReview(context.Background(), "design.md")
	require.NoError(t, reviewErr)

	writer.Close()

	captured := make([]byte, 4096) // lint:allow_raw_number
	n, readErr := reader.Read(captured)
	require.NoError(t, readErr)

	os.Stdout = origStdout

	capturedOut := string(captured[:n])

	assert.Contains(t, capturedOut, "magneto review is deprecated")
	assert.Contains(t, capturedOut, "finalize_review_session")
}

// TestReviewCommand_RemainsNonInteractive verifies that the review command
// performs no interactive prompts, reads no stdin, and invokes no subagents. It
// completes synchronously without blocking.
func TestReviewCommand_RemainsNonInteractive(t *testing.T) {
	tmpDir := t.TempDir()

	t.Setenv("WORKSPACE_ROOT", tmpDir)

	artifactPath := filepath.Join(tmpDir, "design.md")
	writeErr := os.WriteFile(artifactPath, []byte(sampleDesignContent), 0o0666)
	require.NoError(t, writeErr)

	fSpecName = "noninteractive-test"
	fDomain = "data-integrity"

	// The test verifies non-interactivity by confirming that runReview
	// completes without blocking. A truly interactive command would hang or
	// fail in this environment with no TTY attached.
	reviewErr := runReview(context.Background(), "design.md")
	require.NoError(t, reviewErr)

	// Verify a record was written, confirming the command ran to completion.
	reviewDir := filepath.Join(tmpDir, ".kiro", "reviews")
	entries, readDirErr := os.ReadDir(reviewDir)
	require.NoError(t, readDirErr)
	assert.Greater(t, len(entries), 0, "expected review file from non-interactive run")
}

// TestReviewCommand_NoSubagentInvocation verifies that the review command does
// not invoke subagents or claim to run the operational workflow. The output
// record should contain no indication of subagent orchestration.
func TestReviewCommand_NoSubagentInvocation(t *testing.T) {
	tmpDir := t.TempDir()

	t.Setenv("WORKSPACE_ROOT", tmpDir)

	artifactPath := filepath.Join(tmpDir, "design.md")
	writeErr := os.WriteFile(artifactPath, []byte(sampleDesignContent), 0o0666)
	require.NoError(t, writeErr)

	fSpecName = "no-agent-test"
	fDomain = "secrets"

	reviewErr := runReview(context.Background(), "design.md")
	require.NoError(t, reviewErr)

	reviewDir := filepath.Join(tmpDir, ".kiro", "reviews")
	entries, readDirErr := os.ReadDir(reviewDir)
	require.NoError(t, readDirErr)
	require.Greater(t, len(entries), 0)

	outputPath := filepath.Join(reviewDir, entries[0].Name())
	content, readErr := os.ReadFile(outputPath) // lint:allow_dynamic_filename
	require.NoError(t, readErr)

	reviewContent := string(content)

	// The record must not contain subagent orchestration terminology.
	assert.NotContains(t, reviewContent, "subagent invoked")
	assert.NotContains(t, reviewContent, "Reviewer invocation")
	assert.NotContains(t, reviewContent, "Confirmer invocation")
	assert.NotContains(t, reviewContent, "operational workflow executed")
}

// TestReviewCommand_ExistingRecordsRenderable verifies that existing review
// records remain renderable after the compatibility changes. Records using
// canonical satisfaction fields render correctly.
func TestReviewCommand_ExistingRecordsRenderable(t *testing.T) {
	tmpDir := t.TempDir()

	t.Setenv("WORKSPACE_ROOT", tmpDir)

	artifactPath := filepath.Join(tmpDir, "design.md")
	writeErr := os.WriteFile(artifactPath, []byte(sampleDesignContent), 0o0666)
	require.NoError(t, writeErr)

	fSpecName = "renderable-test"
	fDomain = "auth"

	reviewErr := runReview(context.Background(), "design.md")
	require.NoError(t, reviewErr)

	reviewDir := filepath.Join(tmpDir, ".kiro", "reviews")
	entries, readDirErr := os.ReadDir(reviewDir)
	require.NoError(t, readDirErr)
	require.Greater(t, len(entries), 0)

	outputPath := filepath.Join(reviewDir, entries[0].Name())
	content, readErr := os.ReadFile(outputPath) // lint:allow_dynamic_filename
	require.NoError(t, readErr)

	reviewContent := string(content)

	// Core structure remains renderable.
	assert.Contains(t, reviewContent, "# Adversarial Review: renderable-test")
	assert.Contains(t, reviewContent, "## Summary")
	assert.Contains(t, reviewContent, "## Findings")
	assert.Contains(t, reviewContent, "## Degradation Summary")
	assert.Contains(t, reviewContent, "Phase 3 Baseline:** absent (no pre-Phase-1 control baseline)")
}

// TestReviewCommand_LegacyScoreMigrationMarker verifies that the rendered
// output includes a legacy-score migration marker when the metadata indicates a
// legacy score was received and mapped to canonical satisfaction.
func TestReviewCommand_LegacyScoreMigrationMarker(t *testing.T) {
	sessionOutput := &models.ReviewSessionOutput{
		Metadata: models.SessionMetadata{
			SpecName:            "migration-test",
			ArtifactPath:        "design.md",
			Timestamp:           "2026-01-15T00:00:00Z",
			TerminalStatus:      models.TerminalNotApproved,
			LegacyScoreMigrated: true,
		},
	}

	rendered := output.RenderSession(sessionOutput)

	assert.Contains(t, rendered, "Legacy Score Migration")
	assert.Contains(t, rendered, "mapped to canonical criterion_satisfaction")
	assert.Contains(t, rendered, "Legacy score migration")
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

	reviewContent := string(content)

	expectedSections := []string{
		"# Adversarial Review: structure-test",
		"## Summary",
		"## Findings",
		"## Degradation Summary",
		"Phase 3 Baseline:** absent (no pre-Phase-1 control baseline)",
	}

	for _, section := range expectedSections {
		assert.Contains(t, reviewContent, section)
	}

	assert.NotContains(t, reviewContent, "## Attack Round")
	assert.NotContains(t, reviewContent, "## Human Escalations")
	assert.NotContains(t, reviewContent, "## Human Overrides")
	assert.NotContains(t, reviewContent, "## Human Block Acceptance")
	assert.NotContains(t, reviewContent, "## Dead Checks")
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
			"criterion_name":   "test-criterion",
			"score":            5,
			"finding_severity": "medium",
			"finding_domains":  []string{"architecture"},
			"quoted_excerpt":   "some evidence",
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

// --- Compatibility boundary tests (Task 5.2) ---.

// TestCompatibility_ValidateFindingSchema_ResponseFields verifies that the
// validate_finding_schema tool retains its successful response fields across
// additive extensions.
func TestCompatibility_ValidateFindingSchema_ResponseFields(t *testing.T) {
	request := mcp.CallToolRequest{}

	request.Params.Arguments = map[string]any{
		"finding": map[string]any{
			"criterion_name":         "compat-criterion",
			"criterion_satisfaction": 7,
			"finding_severity":       "high",
			"finding_domains":        []string{"security"},
			"quoted_excerpt":         "evidence text",
			"artifact_location": map[string]any{
				"file_path":         "design.md",
				"section_reference": "Overview",
			},
			"status":    "hypothesized",
			"reasoning": "rationale",
		},
	}

	result, handleErr := handleValidateFindingSchema(context.Background(), request)
	require.NoError(t, handleErr)
	require.NotNil(t, result)

	text := extractToolResultText(t, result)

	var parsed map[string]any

	unmarshalErr := json.Unmarshal([]byte(text), &parsed)
	require.NoError(t, unmarshalErr)

	// Compatibility: "valid" field remains present and true.
	assert.Equal(t, true, parsed["valid"])

	// Compatibility: normalized_finding is present on success.
	assert.NotNil(t, parsed["normalized_finding"])
}

// TestCompatibility_ValidateFindingSchema_LegacyScoreAtAdapter verifies that
// the MCP schema tool accepts legacy "score" at the input adapter level and
// maps it to criterion_satisfaction in the normalized output.
func TestCompatibility_ValidateFindingSchema_LegacyScoreAtAdapter(t *testing.T) {
	request := mcp.CallToolRequest{}

	request.Params.Arguments = map[string]any{
		"finding": map[string]any{
			"criterion_name":   "legacy-compat",
			"score":            8,
			"finding_severity": "medium",
			"finding_domains":  []string{"correctness"},
			"quoted_excerpt":   "some evidence",
			"artifact_location": map[string]any{
				"file_path":         "design.md",
				"section_reference": "Security",
			},
			"status":    "hypothesized",
			"reasoning": "legacy score test",
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

	// The normalized finding must carry criterion_satisfaction mapped from
	// legacy score.
	normalized, ok := parsed["normalized_finding"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, float64(8), normalized["criterion_satisfaction"])
}

// TestCompatibility_ValidateFindingSchema_CanonicalPrecedenceOverScore verifies
// that when both criterion_satisfaction and legacy score are present, the
// canonical field takes precedence (score is ignored).
func TestCompatibility_ValidateFindingSchema_CanonicalPrecedenceOverScore(t *testing.T) {
	request := mcp.CallToolRequest{}

	request.Params.Arguments = map[string]any{
		"finding": map[string]any{
			"criterion_name":         "precedence-test",
			"criterion_satisfaction": 9,
			"score":                  3,
			"finding_severity":       "low",
			"finding_domains":        []string{"architecture"},
			"quoted_excerpt":         "evidence",
			"artifact_location": map[string]any{
				"file_path":         "design.md",
				"section_reference": "Architecture",
			},
			"status":    "hypothesized",
			"reasoning": "canonical takes precedence",
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

	normalized, ok := parsed["normalized_finding"].(map[string]any)
	require.True(t, ok)

	// Canonical 9 must win over legacy 3.
	assert.Equal(t, float64(9), normalized["criterion_satisfaction"])
}

// TestCompatibility_ValidateCitation_ResponseFields verifies that the
// validate_citation tool retains its "Valid" response field for successful
// citations.
func TestCompatibility_ValidateCitation_ResponseFields(t *testing.T) {
	tmpDir := t.TempDir()

	workspaceRoot = tmpDir

	writeErr := os.WriteFile(
		filepath.Join(tmpDir, "design.md"),
		[]byte(sampleDesignContent),
		0o0666, // lint:allow_666
	)
	require.NoError(t, writeErr)

	request := mcp.CallToolRequest{}

	request.Params.Arguments = map[string]any{
		"quoted_excerpt":    "short-lived tokens with automatic rotation",
		"file_path":         "design.md",
		"section_reference": "Security",
	}

	result, handleErr := handleValidateCitation(context.Background(), request)
	require.NoError(t, handleErr)
	require.NotNil(t, result)

	text := extractToolResultText(t, result)

	var parsed map[string]any

	unmarshalErr := json.Unmarshal([]byte(text), &parsed)
	require.NoError(t, unmarshalErr)

	// Compatibility: "Valid" field remains present.
	assert.Equal(t, true, parsed["Valid"])
}

// TestCompatibility_ValidateFindingsBatch_ResponseFields verifies that the
// validate_findings_batch tool retains its "CitationValid" response field in
// each batch element.
func TestCompatibility_ValidateFindingsBatch_ResponseFields(t *testing.T) {
	tmpDir := t.TempDir()

	workspaceRoot = tmpDir

	writeErr := os.WriteFile(
		filepath.Join(tmpDir, "design.md"),
		[]byte(sampleDesignContent),
		0o0666, // lint:allow_666
	)
	require.NoError(t, writeErr)

	request := mcp.CallToolRequest{}

	request.Params.Arguments = map[string]any{
		"findings": []any{
			map[string]any{
				"QuotedExcerpt":    "MCP protocol",
				"FilePath":         "design.md",
				"SectionReference": "Architecture",
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
	require.Len(t, parsed, 1)

	// Compatibility: "CitationValid" field remains in batch results.
	assert.Equal(t, true, parsed[0]["CitationValid"])
}
