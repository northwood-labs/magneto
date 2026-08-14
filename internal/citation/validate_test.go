// Copyright 2025-2026, Northwood Labs, LLC <license@northwood-labs.com>
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

package citation_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-openapi/testify/assert"
	"github.com/go-openapi/testify/require"

	"go.nwlabs.dev/magneto/internal/citation"
)

// testMarkdownContent is a multi-section Markdown document used by the
// validation tests.
const testMarkdownContent = `# Design Document

## Overview

The system enforces structurally independent review by running
a context-isolated reviewer subagent between design and tasks.

## Architecture

Components communicate over stdio using MCP protocol.
The citation gate validates quoted excerpts deterministically.

## Security

Authentication uses short-lived tokens with automatic rotation.
Secrets are never stored in plaintext on disk.
`

// TestValidate exercises citation validation with table-driven tests covering
// exact match, substring match, section boundaries, line ranges, whitespace
// normalization, and file-not-found errors.
func TestValidate(t *testing.T) {
	tmpDir := t.TempDir()

	writeErr := os.WriteFile(filepath.Join(tmpDir, "design.md"), []byte(testMarkdownContent), 0o0666)
	require.NoError(t, writeErr)

	tests := []struct {
		input         *citation.ValidateInput
		name          string
		failureReason string
		expectValid   bool
		expectErr     bool
	}{
		{
			name: "exact match in heading section",
			input: &citation.ValidateInput{
				QuotedExcerpt: "Components communicate over stdio using MCP protocol. " +
					"The citation gate validates quoted excerpts deterministically.",
				FilePath:         "design.md",
				SectionReference: "Architecture",
				WorkspaceRoot:    tmpDir,
			},
			expectValid: true,
			expectErr:   false,
		},
		{
			name: "substring match within section",
			input: &citation.ValidateInput{
				QuotedExcerpt:    "context-isolated reviewer subagent",
				FilePath:         "design.md",
				SectionReference: "Overview",
				WorkspaceRoot:    tmpDir,
			},
			expectValid: true,
			expectErr:   false,
		},
		{
			name: "section boundary detection rejects cross-section excerpt",
			input: &citation.ValidateInput{
				QuotedExcerpt:    "context-isolated reviewer subagent",
				FilePath:         "design.md",
				SectionReference: "Security",
				WorkspaceRoot:    tmpDir,
			},
			expectValid:   false,
			expectErr:     false,
			failureReason: "quoted excerpt not found within cited section",
		},
		{
			name: "line range extraction finds content",
			input: &citation.ValidateInput{
				QuotedExcerpt:    "structurally independent review",
				FilePath:         "design.md",
				SectionReference: "lines 4-6",
				WorkspaceRoot:    tmpDir,
			},
			expectValid: true,
			expectErr:   false,
		},
		{
			name: "whitespace normalization handles tabs",
			input: &citation.ValidateInput{
				QuotedExcerpt:    "structurally\tindependent\treview",
				FilePath:         "design.md",
				SectionReference: "Overview",
				WorkspaceRoot:    tmpDir,
			},
			expectValid: true,
			expectErr:   false,
		},
		{
			name: "whitespace normalization handles multiple spaces",
			input: &citation.ValidateInput{
				QuotedExcerpt:    "structurally   independent   review",
				FilePath:         "design.md",
				SectionReference: "Overview",
				WorkspaceRoot:    tmpDir,
			},
			expectValid: true,
			expectErr:   false,
		},
		{
			name: "whitespace normalization handles trailing newlines",
			input: &citation.ValidateInput{
				QuotedExcerpt:    "short-lived tokens with automatic rotation\n\n",
				FilePath:         "design.md",
				SectionReference: "Security",
				WorkspaceRoot:    tmpDir,
			},
			expectValid: true,
			expectErr:   false,
		},
		{
			name: "file not found returns ErrFileRead",
			input: &citation.ValidateInput{
				QuotedExcerpt:    "anything",
				FilePath:         "nonexistent.md",
				SectionReference: "Overview",
				WorkspaceRoot:    tmpDir,
			},
			expectValid: false,
			expectErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, validateErr := citation.Validate(
				context.Background(),
				tt.input,
			)

			if tt.expectErr {
				require.Error(t, validateErr)
				assert.True(
					t,
					errors.Is(validateErr, citation.ErrFileRead),
					"expected error wrapping ErrFileRead",
				)

				return
			}

			require.NoError(t, validateErr)
			assert.Equal(t, tt.expectValid, result.Valid)

			if !tt.expectValid && tt.failureReason != "" {
				assert.Equal(t, tt.failureReason, result.FailureReason)
			}
		})
	}
}

// TestValidateBatch exercises batch citation validation to confirm that
// multiple findings are validated in a single call and that per-finding errors
// are captured without aborting the batch.
func TestValidateBatch(t *testing.T) {
	tmpDir := t.TempDir()

	writeErr := os.WriteFile(filepath.Join(tmpDir, "design.md"), []byte(testMarkdownContent), 0o0666)
	require.NoError(t, writeErr)

	input := &citation.BatchInput{
		WorkspaceRoot: tmpDir,
		Findings: []citation.ValidateInput{
			{
				QuotedExcerpt:    "structurally independent review",
				FilePath:         "design.md",
				SectionReference: "Overview",
			},
			{
				QuotedExcerpt:    "this text does not exist anywhere",
				FilePath:         "design.md",
				SectionReference: "Overview",
			},
			{
				QuotedExcerpt:    "anything",
				FilePath:         "missing.md",
				SectionReference: "Overview",
			},
		},
	}

	results, batchErr := citation.ValidateBatch(context.Background(), input)
	require.NoError(t, batchErr)
	require.Len(t, results, 3)

	assert.Equal(t, 0, results[0].FindingIndex)
	assert.True(t, results[0].CitationValid)

	assert.Equal(t, 1, results[1].FindingIndex)
	assert.False(t, results[1].CitationValid)
	assert.Equal(t, "quoted excerpt not found within cited section", results[1].FailureReason)

	assert.Equal(t, 2, results[2].FindingIndex)
	assert.False(t, results[2].CitationValid)
	assert.Contains(t, results[2].FailureReason, "failed to read cited file")
}
