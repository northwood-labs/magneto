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
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-openapi/testify/assert"
	"github.com/go-openapi/testify/require"
)

type (
	// hookConfig represents the structure of a Kiro hook JSON configuration
	// file for parsing and verification purposes.
	hookConfig struct {
		Version string    `json:"version"`
		Hooks   []hookDef `json:"hooks"`
	}

	// hookDef represents a single hook definition within a hook configuration.
	hookDef struct {
		Name    string     `json:"name"`
		Trigger string     `json:"trigger"`
		Action  hookAction `json:"action"`
	}

	// hookAction represents the action block of a hook definition.
	hookAction struct {
		Type   string `json:"type"`
		Prompt string `json:"prompt"`
	}
)

// loadHookConfig reads and parses the adversarial-review-trigger.json file from
// the project root relative to the cmd test working directory.
func loadHookConfig(t *testing.T) *hookConfig {
	t.Helper()

	hookPath := filepath.Join(
		"..", ".kiro", "hooks", "adversarial-review-trigger.json",
	)

	data, readErr := os.ReadFile(hookPath) // lint:allow_dynamic_filename
	require.NoError(t, readErr, "failed to read hook configuration file")

	var cfg hookConfig

	unmarshalErr := json.Unmarshal(data, &cfg)
	require.NoError(t, unmarshalErr, "hook configuration is not valid JSON")

	return &cfg
}

// TestHookConfig_WellFormedJSON verifies the hook configuration file is valid
// JSON and contains the expected top-level structure.
//
// Requirements: 1.1–1.7, 7.2–7.8.
func TestHookConfig_WellFormedJSON(t *testing.T) {
	cfg := loadHookConfig(t)

	assert.NotEmpty(t, cfg.Version)
	require.Len(t, cfg.Hooks, 1, "expected exactly one hook definition")
}

// TestHookConfig_HookName verifies the hook is named according to the approved
// operational workflow design.
//
// Requirements: 7.2.
func TestHookConfig_HookName(t *testing.T) {
	cfg := loadHookConfig(t)

	assert.Equal(t, "Magneto: Adversarial Review Trigger", cfg.Hooks[0].Name)
}

// TestHookConfig_TriggerIsPostTaskExec verifies the hook fires on the
// PostTaskExec lifecycle event.
//
// Requirements: 1.1, 7.2.
func TestHookConfig_TriggerIsPostTaskExec(t *testing.T) {
	cfg := loadHookConfig(t)

	assert.Equal(t, "PostTaskExec", cfg.Hooks[0].Trigger)
}

// TestHookConfig_SnapshotFirstOrdering verifies the prompt requires a pre-task
// snapshot comparison (SHA-256 digest) before classification.
//
// Requirements: 1.1, 7.3.
func TestHookConfig_SnapshotFirstOrdering(t *testing.T) {
	cfg := loadHookConfig(t)
	prompt := cfg.Hooks[0].Action.Prompt

	hasDigest := strings.Contains(prompt, "SHA-256 digest") || strings.Contains(prompt, "pre-task snapshot")
	assert.True(t, hasDigest, "prompt must mention SHA-256 digest or pre-task snapshot for snapshot-first ordering")
}

// TestHookConfig_AmbiguousFallback verifies the prompt handles ambiguous
// classification by starting a review session.
//
// Requirements: 1.3, 7.4.
func TestHookConfig_AmbiguousFallback(t *testing.T) {
	cfg := loadHookConfig(t)
	prompt := cfg.Hooks[0].Action.Prompt

	assert.Contains(t, prompt, "ambiguous",
		"prompt must mention ambiguous classification fallback")
}

// TestHookConfig_OneSessionSelection verifies the prompt enforces exactly one
// review session per PostTaskExec event.
//
// Requirements: 1.2, 1.3, 1.7, 7.4.
func TestHookConfig_OneSessionSelection(t *testing.T) {
	cfg := loadHookConfig(t)
	prompt := cfg.Hooks[0].Action.Prompt

	assert.Contains(t, prompt, "exactly one review session",
		"prompt must enforce exactly one review session selection")
}

// TestHookConfig_AdvisorySkipAnnotation verifies the prompt produces a
// review-skipped annotation for unchanged or ineligible artifacts.
//
// Requirements: 1.4, 1.5, 9.8.
func TestHookConfig_AdvisorySkipAnnotation(t *testing.T) {
	cfg := loadHookConfig(t)
	prompt := cfg.Hooks[0].Action.Prompt

	assert.Contains(t, prompt, "review-skipped",
		"prompt must produce review-skipped annotation for skipped artifacts")
}

// TestHookConfig_AdvisoryFailureAnnotation verifies the prompt produces a
// review-trigger-failed annotation on session start failure.
//
// Requirements: 1.6, 9.8.
func TestHookConfig_AdvisoryFailureAnnotation(t *testing.T) {
	cfg := loadHookConfig(t)
	prompt := cfg.Hooks[0].Action.Prompt

	assert.Contains(t, prompt, "review-trigger-failed",
		"prompt must produce review-trigger-failed annotation on failure")
}

// TestHookConfig_ReferencesOperationalProtocol verifies the prompt references
// the steering file that defines the operational protocol.
//
// Requirements: 7.2, 7.3.
func TestHookConfig_ReferencesOperationalProtocol(t *testing.T) {
	cfg := loadHookConfig(t)
	prompt := cfg.Hooks[0].Action.Prompt

	assert.Contains(t, prompt, "adversarial-review-operational-protocol",
		"prompt must reference the operational protocol steering file")
}
