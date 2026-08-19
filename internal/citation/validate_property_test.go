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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-openapi/testify/assert"
	"github.com/go-openapi/testify/require"
	"pgregory.net/rapid"

	"go.nwlabs.dev/magneto/internal/citation"
)

// TestProperty_CitationRoundTrip verifies Property 1: Citation validation
// round-trip.
//
// For any file content and any substring of that content, if the substring is
// extracted from within a valid section boundary, then Validate called with
// that exact substring and correct section reference SHALL return Valid: true.
//
// **Validates: Requirements 4.4**.
func TestProperty_CitationRoundTrip(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		// Generate a heading name and body text.
		heading := rapid.StringMatching(`[A-Z][a-z]{3,20}`).Draw(rt, "heading")
		body := rapid.StringMatching(`[a-zA-Z0-9 .,;:!?]{10,200}`).Draw(rt, "body")

		// Construct Markdown content with a heading and body.
		content := "# " + heading + "\n\n" + body + "\n"

		// Pick a substring from the body as the excerpt.
		start := rapid.IntRange(0, len(body)-5).Draw(rt, "start")
		end := rapid.IntRange(start+3, min(start+50, len(body))).Draw(rt, "end")
		excerpt := body[start:end]

		// Write generated content to a temp file.
		dir := t.TempDir()
		filePath := filepath.Join(dir, "test.md")

		writeErr := os.WriteFile(filePath, []byte(content), 0o0666)
		if writeErr != nil {
			rt.Fatal(writeErr)
		}

		// Call Validate with the extracted substring and section reference.
		result, validateErr := citation.Validate(
			context.Background(),
			&citation.ValidateInput{
				QuotedExcerpt:    excerpt,
				FilePath:         "test.md",
				SectionReference: heading,
				WorkspaceRoot:    dir,
			},
		)
		if validateErr != nil {
			rt.Fatal(validateErr)
		}

		// Assert Valid: true is returned.
		if !result.Valid {
			rt.Fatalf(
				"expected valid=true for excerpt %q in section %q, got failure: %s",
				excerpt,
				heading,
				result.FailureReason,
			)
		}
	})
}

// TestProperty_NonExistentCitationsAlwaysFail validates Property 2:
// Non-existent citations always fail.
//
// For any quoted excerpt that does not appear as a substring within the cited
// section of the cited file, Validate SHALL return Valid: false.
//
// **Validates: Requirements 4.5**.
func TestProperty_NonExistentCitationsAlwaysFail(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		// Generate a heading using a capitalized word.
		heading := rapid.StringMatching(`[A-Z][a-z]{3,15}`).Draw(rt, "heading")

		// Generate body text using only lowercase letters and spaces so that
		// the fake excerpt (which uses an uppercase prefix) cannot appear in
		// it.
		body := rapid.StringMatching(`[a-z ]{10,100}`).Draw(rt, "body")
		content := "# " + heading + "\n\n" + body + "\n"

		// Generate a fake excerpt with an uppercase prefix that cannot exist in
		// the lowercase-only body.
		suffix := rapid.StringMatching(`[A-Z]{5,15}`).Draw(rt, "suffix")
		fakeExcerpt := "NOTFOUND" + suffix

		// Write content to a temp file.
		dir := t.TempDir()
		filePath := filepath.Join(dir, "test.md")

		writeErr := os.WriteFile(filePath, []byte(content), 0o0666)
		if writeErr != nil {
			rt.Fatal(writeErr)
		}

		result, validateErr := citation.Validate(
			context.Background(),
			&citation.ValidateInput{
				QuotedExcerpt:    fakeExcerpt,
				FilePath:         "test.md",
				SectionReference: heading,
				WorkspaceRoot:    dir,
			},
		)
		if validateErr != nil {
			rt.Fatal(validateErr)
		}

		// Property: non-existent excerpts must always return Valid: false.
		if result.Valid {
			rt.Fatalf("expected Valid=false for fake excerpt %q in body %q", fakeExcerpt, body)
		}
	})
}

// TestProperty_WhitespaceNormalizationPreservesMatch verifies Property 3:
// Whitespace normalization preserves match semantics.
//
// For any text excerpt that exists in a file, adding or removing whitespace
// within runs (but not altering word boundaries or content) SHALL NOT change
// the validation outcome — the normalized forms must still match.
//
// **Validates: Requirements 4.4**.
func TestProperty_WhitespaceNormalizationPreservesMatch(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		// Generate a heading for the Markdown section.
		heading := rapid.StringMatching(`[A-Z][a-z]{3,15}`).Draw(rt, "heading")

		// Generate a multi-word body (at least 3 words).
		wordCount := rapid.IntRange(3, 8).Draw(rt, "word_count")
		words := make([]string, wordCount)

		for i := range words {
			words[i] = rapid.StringMatching(`[a-z]{2,10}`).Draw(rt, "word")
		}

		body := strings.Join(words, " ")
		content := "# " + heading + "\n\n" + body + "\n"

		// Use all words as the canonical excerpt.
		excerpt := body

		// Insert random whitespace between words to create a modified version
		// of the excerpt.
		whitespaceChars := []string{
			" ",
			"  ",
			"\t",
			" \t ",
			"\n",
			"  \t\n  ",
		}
		modifiedWords := make([]string, len(words))
		copy(modifiedWords, words)

		var builder strings.Builder

		for i, w := range modifiedWords {
			if i > 0 {
				wsIdx := rapid.IntRange(0, len(whitespaceChars)-1).Draw(rt, "ws_idx")
				builder.WriteString(whitespaceChars[wsIdx])
			}

			builder.WriteString(w)
		}

		modifiedExcerpt := builder.String()

		// Write content to a temp file.
		dir := t.TempDir()
		testFile := filepath.Join(dir, "test.md")

		writeErr := os.WriteFile(testFile, []byte(content), 0o0666)
		if writeErr != nil {
			rt.Fatal(writeErr)
		}

		// Validate with the whitespace-modified excerpt.
		result, validateErr := citation.Validate(
			context.Background(),
			&citation.ValidateInput{
				QuotedExcerpt:    modifiedExcerpt,
				FilePath:         "test.md",
				SectionReference: heading,
				WorkspaceRoot:    dir,
			},
		)
		if validateErr != nil {
			rt.Fatal(validateErr)
		}

		// Property: whitespace normalization should still find the match
		// because both excerpt and content normalize to the same words.
		if !result.Valid {
			rt.Fatalf(
				"expected Valid=true for whitespace-modified excerpt %q (original: %q) in section %q, got failure: %s",
				modifiedExcerpt,
				excerpt,
				heading,
				result.FailureReason,
			)
		}
	})
}

// TestProperty6_NormalizedQuotationMatchingInvariantUnderWhitespace verifies
// Property 6: Normalized quotation matching is invariant under whitespace
// variation.
//
// For any quoted excerpt and cited section that differ only by runs of
// whitespace, deterministic citation validation returns the same successful
// match result as validation of the original text.
//
// Feature: adversarial-review-operational-workflow, Property 6: Normalized
// quotation matching is invariant under whitespace variation
//
// **Validates: Requirements 4.2**.
func TestProperty6_NormalizedQuotationMatchingInvariantUnderWhitespace(t *testing.T) {
	rapid.Check(t, func(rt *rapid.T) {
		// Generate a heading for the Markdown section.
		heading := rapid.StringMatching(`[A-Z][a-z]{3,15}`).Draw(rt, "heading")

		// Generate a multi-word body (3-10 words of
		// non-whitespace characters).
		wordCount := rapid.IntRange(3, 10).Draw(rt, "word_count")
		words := make([]string, wordCount)

		for i := range words {
			words[i] = rapid.StringMatching(`[a-zA-Z0-9]{2,12}`).Draw(rt, "word")
		}

		body := strings.Join(words, " ")
		content := "# " + heading + "\n\n" + body + "\n"

		// Choose a contiguous sub-slice of words as the excerpt
		// (at least 2 words).
		excerptStart := rapid.IntRange(0, wordCount-2).Draw(rt, "excerpt_start")
		excerptEnd := rapid.IntRange(excerptStart+2, wordCount).Draw(rt, "excerpt_end")

		originalExcerpt := strings.Join(words[excerptStart:excerptEnd], " ")

		// Create a whitespace-varied version of the excerpt by inserting varied
		// whitespace between the same words.
		wsOptions := []string{
			"  ",
			"   ",
			"    ",
			"\t",
			"\n",
			" \t ",
			"\t\t",
			"  \n  ",
		}

		var builder strings.Builder

		for i, w := range words[excerptStart:excerptEnd] {
			if i > 0 {
				wsIdx := rapid.IntRange(0, len(wsOptions)-1).Draw(rt, "ws_idx")
				builder.WriteString(wsOptions[wsIdx])
			}

			builder.WriteString(w)
		}

		variedExcerpt := builder.String()

		// Write content to a temp file.
		dir := t.TempDir()
		testFile := filepath.Join(dir, "test.md")

		writeErr := os.WriteFile(testFile, []byte(content), 0o0666)
		require.NoError(rt, writeErr)

		// Validate with the original excerpt.
		originalResult, originalErr := citation.Validate(
			context.Background(),
			&citation.ValidateInput{
				QuotedExcerpt:    originalExcerpt,
				FilePath:         "test.md",
				SectionReference: heading,
				WorkspaceRoot:    dir,
			},
		)
		require.NoError(rt, originalErr)

		// Validate with the whitespace-varied excerpt.
		variedResult, variedErr := citation.Validate(
			context.Background(),
			&citation.ValidateInput{
				QuotedExcerpt:    variedExcerpt,
				FilePath:         "test.md",
				SectionReference: heading,
				WorkspaceRoot:    dir,
			},
		)
		require.NoError(rt, variedErr)

		// Both must succeed (Valid: true).
		assert.True(
			rt,
			originalResult.Valid,
			"original excerpt %q should match in section %q",
			originalExcerpt,
			heading,
		)
		assert.True(
			rt,
			variedResult.Valid,
			"whitespace-varied excerpt %q should match in section %q (original: %q)",
			variedExcerpt,
			heading,
			originalExcerpt,
		)

		// Both produce the same Valid: true outcome.
		assert.Equal(
			rt,
			originalResult.Valid,
			variedResult.Valid,
			"whitespace variation must not change validation outcome",
		)
	})
}
