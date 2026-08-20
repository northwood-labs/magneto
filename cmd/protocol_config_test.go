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
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-openapi/testify/assert"
	"github.com/go-openapi/testify/require"
)

// loadProtocolContent reads the adversarial-review-operational-protocol.md
// steering file and returns its content as a string.
func loadProtocolContent(t *testing.T) string {
	t.Helper()

	protocolPath := filepath.Join("..", ".kiro", "steering", "adversarial-review-operational-protocol.md")

	data, readErr := os.ReadFile(protocolPath) // lint:allow_dynamic_filename
	require.NoError(t, readErr, "failed to read operational protocol steering file")

	content := string(data)
	require.NotEmpty(t, content, "operational protocol file must not be empty")

	return content
}

// TestProtocol_NonEmpty verifies the steering protocol file exists and has
// content.
//
// Requirements: 7.2.
func TestProtocol_NonEmpty(t *testing.T) {
	content := loadProtocolContent(t)

	assert.Greater(t, len(content), 100, // lint:allow_raw_number
		"protocol file should contain substantial content")
}

// TestProtocol_ReadOnlyCapabilityManifest verifies the protocol defines a
// read-only capability manifest section.
//
// Requirements: 2.1–2.5, 9.2.
func TestProtocol_ReadOnlyCapabilityManifest(t *testing.T) {
	content := loadProtocolContent(t)

	hasManifest := strings.Contains(content, "Read-only capability manifest") ||
		strings.Contains(content, "read-only capability manifest")
	assert.True(t, hasManifest, "protocol must contain a read-only capability manifest section")
}

// TestProtocol_ForbiddenCapabilities verifies the protocol lists all required
// forbidden capabilities for review roles.
//
// Requirements: 2.2, 2.5, 9.2.
func TestProtocol_ForbiddenCapabilities(t *testing.T) {
	content := loadProtocolContent(t)
	contentLower := strings.ToLower(content)

	forbiddenCaps := []string{
		"write",
		"shell",
		"network",
		"deploy",
	}

	for _, cap := range forbiddenCaps {
		assert.True(t, strings.Contains(contentLower, cap), "protocol must list forbidden capability: "+cap)
	}

	// Source control or git mutation must be forbidden.
	hasSourceControl := strings.Contains(contentLower, "source control") ||
		strings.Contains(contentLower, "source-control") ||
		strings.Contains(content, "git") ||
		strings.Contains(content, "commit")
	assert.True(t, hasSourceControl, "protocol must forbid source-control/git mutation")
}

// TestProtocol_AuthorDistinctConfirmer verifies the protocol requires Confirmer
// identity to differ from the author identity.
//
// Requirements: 2.4.
func TestProtocol_AuthorDistinctConfirmer(t *testing.T) {
	content := loadProtocolContent(t)

	hasDistinct := strings.Contains(content, "differs from the author") ||
		strings.Contains(content, "author-distinct") ||
		strings.Contains(content, "different from the author")
	assert.True(t, hasDistinct, "protocol must require author-distinct Confirmer selection")
}

// TestProtocol_OneSessionRule verifies the protocol enforces the one-session
// selection rule per PostTaskExec event.
//
// Requirements: 1.2, 1.3, 1.7, 7.4.
func TestProtocol_OneSessionRule(t *testing.T) {
	content := loadProtocolContent(t)

	hasOneSession := strings.Contains(content, "exactly one") ||
		strings.Contains(content, "exactly zero or exactly one")
	assert.True(t, hasOneSession, "protocol must enforce exactly one (or zero) session rule")
}

// TestProtocol_AdvisoryContinuationAnnotations verifies the protocol
// defines all required advisory continuation annotations.
//
// Requirements: 1.4–1.6, 7.7, 9.8.
func TestProtocol_AdvisoryContinuationAnnotations(t *testing.T) {
	content := loadProtocolContent(t)

	annotations := []string{
		"review-skipped",
		"review-trigger-failed",
		"review-unresolved",
	}

	for _, annotation := range annotations {
		assert.Contains(t, content, annotation, "protocol must define advisory annotation: "+annotation)
	}
}

// TestProtocol_HumanEscalation verifies the protocol describes human escalation
// behavior.
//
// Requirements: 7.2–7.4.
func TestProtocol_HumanEscalation(t *testing.T) {
	content := loadProtocolContent(t)

	hasEscalation := strings.Contains(content, "Human escalation") ||
		strings.Contains(content, "human escalation") ||
		strings.Contains(content, "escalat")
	assert.True(t, hasEscalation, "protocol must describe human escalation")
}

// TestProtocol_TerminalStatusPrecedence verifies the protocol defines terminal
// status precedence rules.
//
// Requirements: 7.6–7.8.
func TestProtocol_TerminalStatusPrecedence(t *testing.T) {
	content := loadProtocolContent(t)

	hasTerminalPrecedence := strings.Contains(content, "terminal status") ||
		strings.Contains(content, "Terminal status") ||
		strings.Contains(content, "status precedence")
	assert.True(t, hasTerminalPrecedence, "protocol must describe terminal status precedence")

	// Verify the specific terminal statuses are mentioned.
	statuses := []string{
		"human_override",
		"partial_review",
		"approved",
		"not_approved",
	}

	for _, status := range statuses {
		assert.Contains(t, content, status, "protocol must mention terminal status: "+status)
	}
}

// TestProtocol_SnapshotFirstOrdering verifies the protocol describes pre-task
// snapshot comparison before classification.
//
// Requirements: 1.1, 7.3.
func TestProtocol_SnapshotFirstOrdering(t *testing.T) {
	content := loadProtocolContent(t)

	hasSnapshot := strings.Contains(content, "pre-task snapshot") ||
		strings.Contains(content, "Pre-task snapshot") ||
		strings.Contains(content, "SHA-256 digest")
	assert.True(t, hasSnapshot, "protocol must describe pre-task snapshot ordering")
}
