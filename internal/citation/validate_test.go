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
	"runtime"
	"strings"
	"testing"

	"github.com/go-openapi/testify/assert"
	"github.com/go-openapi/testify/require"

	"go.nwlabs.dev/magneto/internal/citation"
)

const (
	// testMarkdownContent is a multi-section Markdown document used by the
	// validation tests.
	testMarkdownContent = `# Design Document

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

	// headingScopeContent provides a document where the same phrase appears in
	// two different sections.
	headingScopeContent = `# Design

## Authentication

The service uses JWT tokens for authentication.
Tokens expire after 15 minutes.

## Authorization

The service uses JWT tokens for authorization checks.
Role-based access controls are enforced at every endpoint.

## Logging

All requests are logged with correlation IDs.
`
)

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

// TestValidate_PathTraversalRejection verifies that a file path containing
// "../" sequences that attempt to escape the workspace root is rejected by
// the containment check.
//
// Requirements: 4.5, 7.5.
func TestValidate_PathTraversalRejection(t *testing.T) {
	workspace := t.TempDir()

	// Create a file outside the workspace to ensure traversal would
	// succeed without containment.
	outsideDir := t.TempDir()

	writeErr := os.WriteFile(
		filepath.Join(outsideDir, "secret.md"),
		[]byte("# Secret\n\nTop secret content here.\n"),
		0o0666,
	)
	require.NoError(t, writeErr)

	// Construct a relative path that escapes workspace using "../".
	relToOutside, relErr := filepath.Rel(workspace, outsideDir)
	require.NoError(t, relErr)

	traversalPath := filepath.Join(relToOutside, "secret.md")

	// The path should contain ".." to confirm the test is meaningful.
	assert.Contains(t, traversalPath, "..")

	_, validateErr := citation.Validate(
		context.Background(),
		&citation.ValidateInput{
			QuotedExcerpt:    "Top secret content here.",
			FilePath:         traversalPath,
			SectionReference: "Secret",
			WorkspaceRoot:    workspace,
		},
	)

	require.Error(t, validateErr)
	assert.True(
		t,
		errors.Is(validateErr, citation.ErrPathTraversal),
		"expected ErrPathTraversal, got: %v",
		validateErr,
	)
}

// TestValidate_SymlinkEscapeRejection verifies that a symlink within the
// workspace that resolves to a file outside the workspace root is rejected by
// the containment check.
//
// Requirements: 4.5, 7.5.
func TestValidate_SymlinkEscapeRejection(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink tests not reliable on Windows")
	}

	workspace := t.TempDir()

	// Create a target file outside the workspace.
	outsideDir := t.TempDir()
	outsideFile := filepath.Join(outsideDir, "external.md")

	writeErr := os.WriteFile(
		outsideFile,
		[]byte("# External\n\nThis is outside the workspace.\n"),
		0o0666,
	)
	require.NoError(t, writeErr)

	// Create a symlink inside the workspace pointing outside.
	symlinkPath := filepath.Join(workspace, "escape.md")

	symlinkErr := os.Symlink(outsideFile, symlinkPath)
	require.NoError(t, symlinkErr)

	_, validateErr := citation.Validate(
		context.Background(),
		&citation.ValidateInput{
			QuotedExcerpt:    "This is outside the workspace.",
			FilePath:         "escape.md",
			SectionReference: "External",
			WorkspaceRoot:    workspace,
		},
	)

	require.Error(t, validateErr)
	assert.True(
		t,
		errors.Is(validateErr, citation.ErrPathTraversal),
		"expected ErrPathTraversal for symlink escape, got: %v",
		validateErr,
	)
}

// TestValidate_OversizedFileRejection verifies that a file exceeding the 64 MiB
// hard limit is rejected by the size safety check during chunked reading.
// Rather than allocating a full 64 MiB, this test creates a file slightly over
// the limit using sparse writes.
//
// Requirements: 4.5, 4.6.
func TestValidate_OversizedFileRejection(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping large file test in short mode")
	}

	workspace := t.TempDir()
	largeFile := filepath.Join(workspace, "large.md")

	// Create a file that exceeds 64 MiB (67108864 bytes). Write a header plus
	// enough data to cross the boundary.
	f, createErr := os.Create(largeFile) // lint:allow_dynamic_filename
	require.NoError(t, createErr)

	header := "# Large\n\nContent start.\n"

	_, writeErr := f.WriteString(header)
	require.NoError(t, writeErr)

	// Write 1 MiB chunks to exceed the 64 MiB limit. We need 65 chunks of 1 MiB
	// to guarantee exceeding 64 MiB given the header.
	chunk := strings.Repeat("x", 1048576) // lint:allow_raw_number

	for i := range 65 { // lint:allow_raw_number
		_ = i

		_, chunkErr := f.WriteString(chunk)
		require.NoError(t, chunkErr)
	}

	closeErr := f.Close()
	require.NoError(t, closeErr)

	_, validateErr := citation.Validate(
		context.Background(),
		&citation.ValidateInput{
			QuotedExcerpt:    "Content start.",
			FilePath:         "large.md",
			SectionReference: "Large",
			WorkspaceRoot:    workspace,
		},
	)

	require.Error(t, validateErr)
	assert.True(
		t,
		errors.Is(validateErr, citation.ErrFileTooLarge),
		"expected ErrFileTooLarge, got: %v",
		validateErr,
	)
}

// TestValidate_HeadingScopeRestriction verifies that a citation specifying a
// section heading only matches within that heading's scope and not in other
// sections containing the same text.
//
// Requirements: 4.2.
func TestValidate_HeadingScopeRestriction(t *testing.T) {
	workspace := t.TempDir()

	writeErr := os.WriteFile(
		filepath.Join(workspace, "design.md"),
		[]byte(headingScopeContent),
		0o0666,
	)
	require.NoError(t, writeErr)

	// "Role-based access controls" exists only in Authorization. Citing it
	// under Authentication should fail.
	result, validateErr := citation.Validate(
		context.Background(),
		&citation.ValidateInput{
			QuotedExcerpt:    "Role-based access controls are enforced at every endpoint.",
			FilePath:         "design.md",
			SectionReference: "Authentication",
			WorkspaceRoot:    workspace,
		},
	)

	require.NoError(t, validateErr)
	assert.False(t, result.Valid)
	assert.Equal(
		t,
		"quoted excerpt not found within cited section",
		result.FailureReason,
	)

	// Same excerpt cited under Authorization should succeed.
	result, validateErr = citation.Validate(
		context.Background(),
		&citation.ValidateInput{
			QuotedExcerpt:    "Role-based access controls are enforced at every endpoint.",
			FilePath:         "design.md",
			SectionReference: "Authorization",
			WorkspaceRoot:    workspace,
		},
	)

	require.NoError(t, validateErr)
	assert.True(t, result.Valid)
}

// TestValidate_LineRangeScopeRestriction verifies that when a citation
// specifies a line range, the match is only sought within those lines.
//
// Requirements: 4.2.
func TestValidate_LineRangeScopeRestriction(t *testing.T) {
	workspace := t.TempDir()

	writeErr := os.WriteFile(
		filepath.Join(workspace, "design.md"),
		[]byte(headingScopeContent),
		0o0666,
	)
	require.NoError(t, writeErr)

	// "Tokens expire after 15 minutes" is on line 5 of the content. Citing it
	// in a line range that excludes line 5 should fail.
	result, validateErr := citation.Validate(
		context.Background(),
		&citation.ValidateInput{
			QuotedExcerpt:    "Tokens expire after 15 minutes.",
			FilePath:         "design.md",
			SectionReference: "lines 8-12",
			WorkspaceRoot:    workspace,
		},
	)

	require.NoError(t, validateErr)
	assert.False(t, result.Valid)
	assert.Equal(
		t,
		"quoted excerpt not found within cited section",
		result.FailureReason,
	)

	// Citing it in the correct line range should succeed.
	result, validateErr = citation.Validate(
		context.Background(),
		&citation.ValidateInput{
			QuotedExcerpt:    "Tokens expire after 15 minutes.",
			FilePath:         "design.md",
			SectionReference: "lines 4-6",
			WorkspaceRoot:    workspace,
		},
	)

	require.NoError(t, validateErr)
	assert.True(t, result.Valid)
}

// TestValidate_WhitespaceNormalizationVariants verifies that whitespace
// differences between the quoted excerpt and actual text still produce a valid
// match. This exercises tabs, extra spaces, newlines, and mixed whitespace.
//
// Requirements: 4.2.
func TestValidate_WhitespaceNormalizationVariants(t *testing.T) {
	workspace := t.TempDir()

	content := "# Architecture\n\nComponents communicate over " +
		"stdio using MCP protocol.\n"

	writeErr := os.WriteFile(
		filepath.Join(workspace, "arch.md"),
		[]byte(content),
		0o0666,
	)
	require.NoError(t, writeErr)

	tests := []struct {
		name    string
		excerpt string
	}{
		{
			name:    "tabs between words",
			excerpt: "Components\tcommunicate\tover\tstdio",
		},
		{
			name:    "multiple spaces between words",
			excerpt: "Components    communicate    over    stdio",
		},
		{
			name:    "mixed tabs and spaces",
			excerpt: "Components \t communicate \t over \t stdio",
		},
		{
			name:    "newlines between words",
			excerpt: "Components\ncommunicate\nover\nstdio",
		},
		{
			name:    "leading and trailing whitespace",
			excerpt: "  \t Components communicate over stdio \n ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, validateErr := citation.Validate(
				context.Background(),
				&citation.ValidateInput{
					QuotedExcerpt:    tt.excerpt,
					FilePath:         "arch.md",
					SectionReference: "Architecture",
					WorkspaceRoot:    workspace,
				},
			)

			require.NoError(t, validateErr)
			assert.True(t, result.Valid,
				"whitespace variant %q should still match", tt.excerpt,
			)
		})
	}
}
