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

package citation

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	// readChunkSize is the size of each read chunk (1 MiB).
	readChunkSize = 1048576

	// maxFileSize is the maximum file size allowed for citation
	// validation (64 MiB).
	maxFileSize = 67108864
)

type (
	// ValidateInput contains the parameters for a single citation validation.
	ValidateInput struct {
		QuotedExcerpt    string
		FilePath         string
		SectionReference string
		WorkspaceRoot    string
	}

	// ValidateResult contains the outcome of a citation validation check.
	ValidateResult struct {
		MatchLocation *MatchLocation
		FailureReason string
		Valid         bool
	}

	// MatchLocation identifies the exact line range where a citation matched.
	MatchLocation struct {
		LineStart int
		LineEnd   int
	}

	// BatchInput contains the parameters for a batch citation validation.
	BatchInput struct {
		WorkspaceRoot string
		Findings      []ValidateInput
	}

	// BatchResult contains the outcome of a single finding within a batch.
	BatchResult struct {
		FailureReason string
		FindingIndex  int
		CitationValid bool
	}
)

// Validate checks whether a quoted excerpt exists verbatim within the cited
// section of the cited file. It resolves the file path relative to the
// workspace root, verifies the resolved path remains within the workspace
// boundary, reads the file in chunks, locates the section, and performs a
// normalized substring match.
func Validate(_ context.Context, input *ValidateInput) (ValidateResult, error) {
	absPath, containErr := resolveContainedPath(
		input.WorkspaceRoot,
		input.FilePath,
	)
	if containErr != nil {
		return ValidateResult{}, fmt.Errorf(
			"path resolution failed: %w", containErr,
		)
	}

	content, readErr := readFileChunked(absPath)
	if readErr != nil {
		return ValidateResult{}, fmt.Errorf(
			"file read failed: %w", readErr,
		)
	}

	return matchExcerptInContent(string(content), input), nil
}

// resolveContainedPath resolves a file path relative to the workspace root and
// verifies the result remains within the workspace boundary. This prevents
// path traversal attacks using sequences like "../../" or symlinks that escape
// the workspace.
func resolveContainedPath(workspaceRoot, filePath string) (string, error) {
	absRoot, rootErr := filepath.Abs(workspaceRoot)
	if rootErr != nil {
		return "", fmt.Errorf(
			"%w: failed to resolve workspace root",
			ErrPathTraversal,
		)
	}

	joined := filepath.Join(absRoot, filePath)

	resolved, evalErr := filepath.EvalSymlinks(joined)
	if evalErr != nil {
		return "", fmt.Errorf("%w: %s", ErrFileRead, joined)
	}

	resolvedRoot, rootEvalErr := filepath.EvalSymlinks(absRoot)
	if rootEvalErr != nil {
		return "", fmt.Errorf(
			"%w: failed to resolve workspace root",
			ErrPathTraversal,
		)
	}

	// Ensure the resolved path is contained within the resolved workspace
	// root. The trailing separator prevents prefix collisions (e.g.,
	// "/workspace-extra" matching "/workspace").
	if !strings.HasPrefix(resolved, resolvedRoot+string(filepath.Separator)) &&
		resolved != resolvedRoot {
		return "", fmt.Errorf(
			"%w: %s escapes %s",
			ErrPathTraversal,
			filePath,
			workspaceRoot,
		)
	}

	return resolved, nil
}

// readFileChunked reads a file in 1 MiB chunks, accumulating content up to the
// maximum allowed file size. This limits peak memory allocation compared to
// [os.ReadFile] which allocates the full file size immediately.
func readFileChunked(path string) ([]byte, error) {
	f, openErr := os.Open(path) // lint:allow_dynamic_filename
	if openErr != nil {
		return nil, fmt.Errorf("%w: %s", ErrFileRead, path)
	}

	defer f.Close() // lint:allow_defer_close

	var buf bytes.Buffer

	chunk := make([]byte, readChunkSize)
	totalRead := 0

	for {
		n, readErr := f.Read(chunk)
		if n > 0 {
			totalRead += n
			if totalRead > maxFileSize {
				return nil, fmt.Errorf(
					"%w: %s exceeds %d bytes",
					ErrFileTooLarge,
					path,
					maxFileSize,
				)
			}

			buf.Write(chunk[:n])
		}

		if readErr == io.EOF {
			break
		}

		if readErr != nil {
			return nil, fmt.Errorf("%w: %s", ErrFileRead, path)
		}
	}

	return buf.Bytes(), nil
}

// matchExcerptInContent locates the cited section within the file content and
// performs a normalized substring match of the quoted excerpt.
func matchExcerptInContent(content string, input *ValidateInput) ValidateResult {
	section, sectionErr := ExtractSection(content, input.SectionReference)
	if sectionErr != nil {
		return ValidateResult{
			Valid:         false,
			FailureReason: "section not found: " + input.SectionReference,
		}
	}

	normalizedExcerpt := NormalizeWhitespace(input.QuotedExcerpt)
	normalizedSection := NormalizeWhitespace(section.Content)

	idx := strings.Index(normalizedSection, normalizedExcerpt)
	if idx < 0 {
		return ValidateResult{
			Valid:         false,
			FailureReason: "quoted excerpt not found within cited section",
		}
	}

	loc := computeLineLocation(content, section.StartLine, normalizedSection, idx)

	return ValidateResult{
		Valid:         true,
		MatchLocation: &loc,
	}
}

// ValidateBatch validates citations for multiple findings in one call. Each
// finding's WorkspaceRoot is overridden with the batch- level WorkspaceRoot.
func ValidateBatch(ctx context.Context, input *BatchInput) ([]BatchResult, error) { // lint:allow_param
	results := make([]BatchResult, 0, len(input.Findings))

	for i := range input.Findings {
		input.Findings[i].WorkspaceRoot = input.WorkspaceRoot

		result, validateErr := Validate(ctx, &input.Findings[i])
		if validateErr != nil {
			results = append(results, BatchResult{
				FindingIndex:  i,
				CitationValid: false,
				FailureReason: validateErr.Error(),
			})

			continue
		}

		results = append(results, BatchResult{
			FindingIndex:  i,
			CitationValid: result.Valid,
			FailureReason: result.FailureReason,
		})
	}

	return results, nil
}

// computeLineLocation determines the line range of a match within the original
// file content. It maps from the byte offset in the normalized section text
// back to line numbers in the source.
func computeLineLocation(
	fullContent string,
	sectionStartLine int,
	normalizedSection string,
	matchIndex int,
) MatchLocation {
	// Count words before the match point in normalized section to approximate
	// the position in the original content.
	prefix := normalizedSection[:matchIndex]
	matchText := normalizedSection[matchIndex:]

	// Count spaces in prefix to estimate word position.
	wordsBeforeMatch := strings.Count(prefix, " ") + 1

	lines := strings.Split(fullContent, "\n")

	// Walk lines from section start, counting words until we reach the match
	// position.
	wordCount := 0
	matchLineStart := sectionStartLine

	for i := sectionStartLine - 1; i < len(lines); i++ {
		lineWords := len(strings.Fields(lines[i]))
		if wordCount+lineWords >= wordsBeforeMatch {
			matchLineStart = i + 1

			break
		}

		wordCount += lineWords
	}

	// Estimate end line by counting words in the match text.
	matchWords := len(strings.Fields(matchText))
	matchLineEnd := matchLineStart
	remainingWords := matchWords

	for i := matchLineStart - 1; i < len(lines); i++ {
		lineWords := len(strings.Fields(lines[i]))

		remainingWords -= lineWords

		if remainingWords <= 0 {
			matchLineEnd = i + 1

			break
		}
	}

	return MatchLocation{
		LineStart: matchLineStart,
		LineEnd:   matchLineEnd,
	}
}
