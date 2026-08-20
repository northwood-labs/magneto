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

package kirofiles_test

import (
	"encoding/json"
	"errors"
	"io/fs"
	"testing"

	"github.com/go-openapi/testify/assert"
	"github.com/go-openapi/testify/require"

	"go.nwlabs.dev/magneto/internal/kirofiles"
)

func TestFilesAndContent(t *testing.T) {
	expectedFiles := []string{
		"hooks/adversarial-review-trigger.json",
		"settings/mcp.json",
		"steering/adversarial-review-anti-patterns.md",
		"steering/adversarial-review-architecture-constraints.md",
		"steering/adversarial-review-operational-protocol.md",
		"steering/adversarial-review-rubric.md",
	}

	assert.Equal(t, expectedFiles, kirofiles.Files())

	for _, path := range expectedFiles {
		content, contentErr := kirofiles.Content(path)
		require.NoError(t, contentErr)
		assert.NotEmpty(t, content, "expected embedded content for %q", path)
	}

	template, templateErr := kirofiles.Content("settings/mcp.json")
	require.NoError(t, templateErr)
	assert.Equal(t, template, kirofiles.MCPTemplate())

	_, contentErr := kirofiles.Content("steering/unlisted.md")
	require.Error(t, contentErr)
	assert.True(t, errors.Is(contentErr, fs.ErrNotExist))
}

func TestMergeMCPConfigCreatesMinimalConfiguration(t *testing.T) {
	definition := kirofiles.MCPTemplate()
	result, mergeErr := kirofiles.MergeMCPConfig(&kirofiles.MergeInput{
		ServerName: "magneto",
		Definition: definition,
	})
	require.NoError(t, mergeErr)
	require.NotNil(t, result)

	assert.JSONEq(t, `{
  "mcpServers": {
    "magneto": {
      "command": "magneto",
      "args": ["serve"],
      "env": {
        "WORKSPACE_ROOT": "${workspaceFolder}"
      },
      "disabled": false
    }
  }
}`, string(result.Output))
	assert.Equal(t, byte('\n'), result.Output[len(result.Output)-1])
}

func TestMergeMCPConfigAddsMCPServersToExistingConfiguration(t *testing.T) {
	definition := kirofiles.MCPTemplate()
	existing := []byte(`{
  "metadata": {"owner": "team"},
  "enabled": true
}`)

	result, mergeErr := kirofiles.MergeMCPConfig(&kirofiles.MergeInput{
		Existing:   existing,
		ServerName: "customMagneto",
		Definition: definition,
	})
	require.NoError(t, mergeErr)
	require.NotNil(t, result)

	configuration := make(map[string]json.RawMessage)
	unmarshalErr := json.Unmarshal(result.Output, &configuration)
	require.NoError(t, unmarshalErr)
	assert.JSONEq(t, `{"owner":"team"}`, string(configuration["metadata"]))
	assert.JSONEq(t, `true`, string(configuration["enabled"]))
	assert.JSONEq(t, `{
  "customMagneto": {
    "command": "magneto",
    "args": ["serve"],
    "env": {"WORKSPACE_ROOT": "${workspaceFolder}"},
    "disabled": false
  }
}`, string(configuration["mcpServers"]))
}

func TestMergeMCPConfigRejectsMalformedConfiguration(t *testing.T) {
	result, mergeErr := kirofiles.MergeMCPConfig(&kirofiles.MergeInput{
		Existing:   []byte(`{"mcpServers":`),
		ServerName: "magneto",
		Definition: kirofiles.MCPTemplate(),
	})

	require.Error(t, mergeErr)
	assert.Nil(t, result)
	assert.ErrorContains(t, mergeErr, "failed to parse MCP configuration")
}
