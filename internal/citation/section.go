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
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

// lineRangePattern matches references like "lines 45-60" or "line 10-20" with
// optional en-dash.
var lineRangePattern = regexp.MustCompile(
	`(?i)^lines?\s+(\d+)\s*[-–]\s*(\d+)$`,
)

// Section represents a located section within a document.
type Section struct {
	Content   string
	StartLine int
	EndLine   int
}

// ExtractSection locates a section within content by heading name or line range
// reference. A heading reference (e.g., "Architecture") finds the named heading
// and returns content from after it to the next same-or-higher-level heading. A
// line range reference (e.g., "lines 45-60") returns content between those
// 1-indexed line numbers.
func ExtractSection(content, reference string) (*Section, error) {
	section, rangeErr := extractByLineRange(content, reference)
	if rangeErr == nil {
		return section, nil
	}

	headingSection, headingErr := extractByHeading(content, reference)
	if headingErr != nil {
		return nil, fmt.Errorf("%w: %s", ErrSectionNotFound, reference)
	}

	return headingSection, nil
}

// extractByLineRange parses a line range reference and extracts the
// corresponding lines from content.
func extractByLineRange(content, reference string) (*Section, error) {
	matches := lineRangePattern.FindStringSubmatch(reference)
	if matches == nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidLineRange, reference)
	}

	startLine, parseErr := strconv.Atoi(matches[1])
	if parseErr != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidLineRange, reference)
	}

	endLine, parseErr := strconv.Atoi(matches[2])
	if parseErr != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidLineRange, reference)
	}

	lines := strings.Split(content, "\n")

	if startLine < 1 || endLine < startLine || startLine > len(lines) {
		return nil, fmt.Errorf(
			"%w: range %d-%d exceeds document length %d",
			ErrInvalidLineRange,
			startLine,
			endLine,
			len(lines),
		)
	}

	if endLine > len(lines) {
		endLine = len(lines)
	}

	extracted := strings.Join(lines[startLine-1:endLine], "\n")

	return &Section{
		Content:   extracted,
		StartLine: startLine,
		EndLine:   endLine,
	}, nil
}

// extractByHeading uses goldmark to parse Markdown and locate a section by its
// heading text. The section spans from the line after the heading to the line
// before the next same-or-higher-level heading (or end of document).
func extractByHeading(content, reference string) (*Section, error) {
	source := []byte(content)
	lines := strings.Split(content, "\n")

	md := goldmark.New()
	reader := text.NewReader(source)
	doc := md.Parser().Parse(reader)

	lowerRef := strings.ToLower(strings.TrimSpace(reference))

	type headingInfo struct {
		level int
		line  int
	}

	var matchedHeading *headingInfo

	walkErr := ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		heading, ok := n.(*ast.Heading)
		if !ok {
			return ast.WalkContinue, nil
		}

		headingText := extractHeadingText(heading, source)
		if strings.EqualFold(headingText, lowerRef) {
			line := lineNumberFromNode(n, source)

			matchedHeading = &headingInfo{
				level: heading.Level,
				line:  line,
			}

			return ast.WalkStop, nil
		}

		return ast.WalkContinue, nil
	})
	if walkErr != nil {
		return nil, fmt.Errorf("%w: %s", ErrSectionNotFound, reference)
	}

	if matchedHeading == nil {
		return nil, fmt.Errorf("%w: %s", ErrSectionNotFound, reference)
	}

	// Find the end of the section: the next heading at the same or
	// higher level.
	sectionStart := matchedHeading.line + 1
	sectionEnd := len(lines)
	foundEnd := false

	endWalkErr := ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering || foundEnd {
			return ast.WalkContinue, nil
		}

		heading, ok := n.(*ast.Heading)
		if !ok {
			return ast.WalkContinue, nil
		}

		line := lineNumberFromNode(n, source)

		if line > matchedHeading.line && heading.Level <= matchedHeading.level {
			sectionEnd = line
			foundEnd = true

			return ast.WalkStop, nil
		}

		return ast.WalkContinue, nil
	})
	if endWalkErr != nil {
		return nil, fmt.Errorf("%w: %s", ErrSectionNotFound, reference)
	}

	if sectionStart > len(lines) {
		sectionStart = len(lines)
	}

	extracted := strings.Join(lines[sectionStart:sectionEnd], "\n")

	return &Section{
		Content:   extracted,
		StartLine: sectionStart + 1,
		EndLine:   sectionEnd,
	}, nil
}

// extractHeadingText returns the plain text content of a heading node.
func extractHeadingText(heading *ast.Heading, source []byte) string {
	var buf strings.Builder

	child := heading.FirstChild()
	for child != nil {
		if child.Kind() == ast.KindText {
			textNode, ok := child.(*ast.Text)
			if ok {
				buf.Write(textNode.Segment.Value(source))
			}
		}

		child = child.NextSibling()
	}

	return buf.String()
}

// lineNumberFromNode determines the 0-indexed line number of an AST node based
// on its byte offset in the source.
func lineNumberFromNode(n ast.Node, source []byte) int {
	if n.Lines().Len() > 0 {
		seg := n.Lines().At(0)

		return countNewlines(source[:seg.Start])
	}

	// Fallback: use first child's segment.
	child := n.FirstChild()
	if child != nil && child.Kind() == ast.KindText {
		textNode, ok := child.(*ast.Text)
		if ok {
			return countNewlines(source[:textNode.Segment.Start])
		}
	}

	return 0
}

// countNewlines counts the number of newline characters in a byte slice.
func countNewlines(b []byte) int {
	count := 0

	for _, c := range b {
		if c == '\n' {
			count++
		}
	}

	return count
}
