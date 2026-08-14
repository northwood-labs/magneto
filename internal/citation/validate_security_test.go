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

// TestValidate_PathTraversal verifies that file paths resolving outside the
// workspace root are rejected with ErrPathTraversal.
func TestValidate_PathTraversal(t *testing.T) {
	tmpDir := t.TempDir()

	writeErr := os.WriteFile(
		filepath.Join(tmpDir, "legit.md"),
		[]byte("# Test\n\n## Section\n\nContent here.\n"),
		0o0666,
	)
	require.NoError(t, writeErr)

	tests := []struct {
		name     string
		filePath string
	}{
		{
			name:     "dot-dot traversal escapes workspace",
			filePath: "../../etc/passwd",
		},
		{
			name:     "absolute path disguised as relative",
			filePath: "../../../tmp/evil.md",
		},
		{
			name:     "nested traversal with valid prefix",
			filePath: "subdir/../../..",
		},
		{
			name:     "dot-dot at start",
			filePath: "../outside.md",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := &citation.ValidateInput{
				QuotedExcerpt:    "anything",
				FilePath:         tt.filePath,
				SectionReference: "Section",
				WorkspaceRoot:    tmpDir,
			}

			_, validateErr := citation.Validate(
				context.Background(),
				input,
			)

			require.Error(t, validateErr)
			assert.True(
				t,
				errors.Is(validateErr, citation.ErrPathTraversal) ||
					errors.Is(validateErr, citation.ErrFileRead),
				"expected ErrPathTraversal or ErrFileRead, got: %v",
				validateErr,
			)
		})
	}
}

// TestValidate_SymlinkEscape verifies that symlinks pointing outside the
// workspace root are rejected.
func TestValidate_SymlinkEscape(t *testing.T) {
	tmpDir := t.TempDir()
	outsideDir := t.TempDir()

	outsideFile := filepath.Join(outsideDir, "secret.md")
	writeErr := os.WriteFile(
		outsideFile,
		[]byte("# Secret\n\n## Data\n\nSensitive content.\n"),
		0o0666,
	)
	require.NoError(t, writeErr)

	symlinkPath := filepath.Join(tmpDir, "escape.md")
	linkErr := os.Symlink(outsideFile, symlinkPath)
	require.NoError(t, linkErr)

	input := &citation.ValidateInput{
		QuotedExcerpt:    "Sensitive content",
		FilePath:         "escape.md",
		SectionReference: "Data",
		WorkspaceRoot:    tmpDir,
	}

	_, validateErr := citation.Validate(context.Background(), input)

	require.Error(t, validateErr)
	assert.True(
		t,
		errors.Is(validateErr, citation.ErrPathTraversal),
		"expected ErrPathTraversal for symlink escape, got: %v",
		validateErr,
	)
}

// TestValidate_ValidPathsStillWork verifies that legitimate paths within the
// workspace continue to work after the containment check is added.
func TestValidate_ValidPathsStillWork(t *testing.T) {
	tmpDir := t.TempDir()

	subDir := filepath.Join(tmpDir, "specs", "v1")
	mkdirErr := os.MkdirAll(subDir, 0o0755)
	require.NoError(t, mkdirErr)

	content := "# Spec\n\n## Details\n\nThe design is solid.\n"
	writeErr := os.WriteFile(
		filepath.Join(subDir, "design.md"),
		[]byte(content),
		0o0666,
	)
	require.NoError(t, writeErr)

	input := &citation.ValidateInput{
		QuotedExcerpt:    "The design is solid.",
		FilePath:         "specs/v1/design.md",
		SectionReference: "Details",
		WorkspaceRoot:    tmpDir,
	}

	result, validateErr := citation.Validate(context.Background(), input)
	require.NoError(t, validateErr)
	assert.True(t, result.Valid)
}

// TestValidate_FileTooLarge verifies that files exceeding the maximum size
// limit are rejected with ErrFileTooLarge.
func TestValidate_FileTooLarge(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping large file test in short mode")
	}

	tmpDir := t.TempDir()

	// Create a file just over the 64 MiB limit (67108864 + 1 byte).
	largePath := filepath.Join(tmpDir, "huge.md")

	f, createErr := os.Create(largePath) // lint:allow_dynamic_filename
	require.NoError(t, createErr)

	// Write a header so the file is valid Markdown.
	_, headerErr := f.WriteString("# Huge\n\n## Content\n\n")
	require.NoError(t, headerErr)

	// Fill with padding to exceed the limit. Write in 1 MiB chunks to
	// avoid allocating 64 MiB in a single buffer.
	chunkSize := 1048576
	chunk := make([]byte, chunkSize)

	for i := range chunk {
		chunk[i] = 'x'
	}

	written := 0
	target := 67108864 + 1

	for written < target {
		toWrite := chunkSize
		if written+toWrite > target {
			toWrite = target - written
		}

		_, writeErr := f.Write(chunk[:toWrite])
		require.NoError(t, writeErr)

		written += toWrite
	}

	closeErr := f.Close()
	require.NoError(t, closeErr)

	input := &citation.ValidateInput{
		QuotedExcerpt:    "anything",
		FilePath:         "huge.md",
		SectionReference: "Content",
		WorkspaceRoot:    tmpDir,
	}

	_, validateErr := citation.Validate(context.Background(), input)

	require.Error(t, validateErr)
	assert.True(
		t,
		errors.Is(validateErr, citation.ErrFileTooLarge),
		"expected ErrFileTooLarge, got: %v",
		validateErr,
	)
}

// TestValidate_FileWithinSizeLimit verifies that a file under the size limit
// is read successfully via chunked reading.
func TestValidate_FileWithinSizeLimit(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a file of ~2 MiB to exercise multi-chunk reading.
	content := "# Multi Chunk\n\n## Evidence\n\nThe target phrase is here.\n"
	padding := make([]byte, 2*1048576)

	for i := range padding {
		padding[i] = ' '
	}

	fullContent := content + string(padding)

	writeErr := os.WriteFile(
		filepath.Join(tmpDir, "medium.md"),
		[]byte(fullContent),
		0o0666,
	)
	require.NoError(t, writeErr)

	input := &citation.ValidateInput{
		QuotedExcerpt:    "The target phrase is here.",
		FilePath:         "medium.md",
		SectionReference: "Evidence",
		WorkspaceRoot:    tmpDir,
	}

	result, validateErr := citation.Validate(context.Background(), input)
	require.NoError(t, validateErr)
	assert.True(t, result.Valid)
}
