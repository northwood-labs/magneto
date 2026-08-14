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
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-openapi/testify/assert"
	"github.com/go-openapi/testify/require"
)

const overriddenMCPServerName = "reviewServer"

type installKiroMCPConfiguration struct {
	MCPServers map[string]json.RawMessage `json:"mcpServers"`
}

// TestInstallKiroCommand_RejectsMissingAndConflictingLocationFlags verifies
// that location selection errors wrap their documented sentinels.
func TestInstallKiroCommand_RejectsMissingAndConflictingLocationFlags(t *testing.T) {
	t.Run("missing location flag", func(t *testing.T) {
		executeErr := executeInstallKiroCommand(t, nil)

		require.Error(t, executeErr)
		assert.ErrorIs(t, executeErr, ErrFlagRequired)
		assert.Contains(t, executeErr.Error(), "--workspace or --user")
	})

	t.Run("conflicting location flags", func(t *testing.T) {
		executeErr := executeInstallKiroCommand(t, []string{"--workspace", "--user"})

		require.Error(t, executeErr)
		assert.ErrorIs(t, executeErr, ErrFlagsMutuallyExclusive)
		assert.Contains(t, executeErr.Error(), "--workspace and --user")
	})
}

// TestInstallKiroCommand_RejectsEmptyHome verifies that user installation
// requires a defined HOME directory.
func TestInstallKiroCommand_RejectsEmptyHome(t *testing.T) {
	t.Setenv("HOME", "")

	executeErr := executeInstallKiroCommand(t, []string{"--user"})

	require.Error(t, executeErr)
	assert.ErrorIs(t, executeErr, ErrFlagRequired)
	assert.Contains(t, executeErr.Error(), "HOME environment variable")
}

// TestInstallKiroCommand_RejectsInvalidTarget verifies that user installation
// rejects targets that are missing or regular files.
func TestInstallKiroCommand_RejectsInvalidTarget(t *testing.T) {
	t.Run("missing directory", func(t *testing.T) {
		missingHome := filepath.Join(t.TempDir(), "missing")

		t.Setenv("HOME", missingHome)

		executeErr := executeInstallKiroCommand(t, []string{"--user"})

		require.Error(t, executeErr)
		assert.ErrorIs(t, executeErr, ErrFileWrite)
		assert.Contains(t, executeErr.Error(), missingHome)
	})

	t.Run("regular file", func(t *testing.T) {
		targetFile := filepath.Join(t.TempDir(), "not-a-directory")
		writeErr := os.WriteFile(targetFile, []byte("not a directory"), 0o666)
		require.NoError(t, writeErr)

		t.Setenv("HOME", targetFile)

		executeErr := executeInstallKiroCommand(t, []string{"--user"})

		require.Error(t, executeErr)
		assert.ErrorIs(t, executeErr, ErrFileWrite)
		assert.Contains(t, executeErr.Error(), "is not a directory")
	})
}

// TestInstallKiroCommand_UsesDefaultAndOverrideServerName verifies that the
// default MCP server name is installed and the -S flag overrides it.
func TestInstallKiroCommand_UsesDefaultAndOverrideServerName(t *testing.T) {
	homeDir := t.TempDir()

	t.Setenv("HOME", homeDir)

	defaultFlag := installKiroCmd.Flags().Lookup("mcp-server-name")
	require.NotNil(t, defaultFlag)
	assert.Equal(t, defaultMCPServerName, defaultFlag.DefValue)
	assert.Equal(t, "S", defaultFlag.Shorthand)

	_, executeErr := captureInstallKiroStdout(t, func() error {
		return executeInstallKiroCommand(t, []string{"--user"})
	})
	require.NoError(t, executeErr)

	configuration := readInstallKiroMCPConfigurationForTest(t, homeDir)
	assert.Contains(t, configuration.MCPServers, defaultMCPServerName)

	_, executeErr = captureInstallKiroStdout(t, func() error {
		return executeInstallKiroCommand(t, []string{"--user", "-S", overriddenMCPServerName})
	})
	require.NoError(t, executeErr)

	configuration = readInstallKiroMCPConfigurationForTest(t, homeDir)
	assert.Contains(t, configuration.MCPServers, overriddenMCPServerName)
	assert.NotContains(t, configuration.MCPServers, defaultMCPServerName)
}

// TestInstallKiroCommand_WriteFailureDoesNotPrintSummary verifies that the
// command returns the file-write sentinel and emits no success summary.
func TestInstallKiroCommand_WriteFailureDoesNotPrintSummary(t *testing.T) {
	homeDir := t.TempDir()
	hooksPath := filepath.Join(homeDir, ".kiro", "hooks")
	mkdirErr := os.MkdirAll(filepath.Dir(hooksPath), kiroDirectoryPermissions)
	require.NoError(t, mkdirErr)

	writeErr := os.WriteFile(hooksPath, []byte("prevents hook directory creation"), kiroFilePermissions)
	require.NoError(t, writeErr)

	t.Setenv("HOME", homeDir)

	output, executeErr := captureInstallKiroStdout(t, func() error {
		return executeInstallKiroCommand(t, []string{"--user"})
	})

	require.Error(t, executeErr)
	assert.ErrorIs(t, executeErr, ErrFileWrite)
	assert.Contains(t, executeErr.Error(), hooksPath)
	assert.Empty(t, output)
}

// executeInstallKiroCommand configures and invokes the installer RunE handler
// without dispatching to the root command or an interactive terminal.
func executeInstallKiroCommand(t *testing.T, arguments []string) error {
	t.Helper()

	resetInstallKiroCommandFlags(t)
	installKiroCmd.SetContext(context.Background())

	t.Cleanup(func() {
		resetInstallKiroCommandFlags(t)
		installKiroCmd.SetContext(context.Background())
	})

	parseErr := installKiroCmd.ParseFlags(arguments)
	if parseErr != nil {
		return fmt.Errorf("parse installer flags: %w", parseErr)
	}

	runErr := installKiroCmd.RunE(installKiroCmd, installKiroCmd.Flags().Args())
	if runErr != nil {
		return fmt.Errorf("run Kiro installer: %w", runErr)
	}

	return nil
}

// resetInstallKiroCommandFlags restores installer flags and their bindings so
// each command execution starts with its documented defaults.
func resetInstallKiroCommandFlags(t *testing.T) {
	t.Helper()

	for _, name := range []string{"workspace", "user", "mcp-server-name"} {
		flag := installKiroCmd.Flags().Lookup(name)
		require.NotNil(t, flag)

		setErr := flag.Value.Set(flag.DefValue)
		require.NoError(t, setErr)

		flag.Changed = false
	}
}

// captureInstallKiroStdout captures standard output while executing an
// installer action.
func captureInstallKiroStdout(t *testing.T, action func() error) (string, error) {
	t.Helper()

	reader, writer, pipeErr := os.Pipe()
	require.NoError(t, pipeErr)

	originalStdout := os.Stdout

	os.Stdout = writer

	actionErr := action()

	closeErr := writer.Close()
	require.NoError(t, closeErr)

	os.Stdout = originalStdout

	output, readErr := io.ReadAll(reader)
	require.NoError(t, readErr)

	readerCloseErr := reader.Close()
	require.NoError(t, readerCloseErr)

	return string(output), actionErr
}

// readInstallKiroMCPConfigurationForTest reads and parses the installed MCP
// configuration for assertions about server-name behavior.
func readInstallKiroMCPConfigurationForTest(t *testing.T, homeDir string) *installKiroMCPConfiguration {
	t.Helper()

	path := filepath.Join(homeDir, ".kiro", mcpTemplatePath)
	content, readErr := os.ReadFile(path) // lint:allow_dynamic_filename
	require.NoError(t, readErr)

	configuration := &installKiroMCPConfiguration{}
	unmarshalErr := json.Unmarshal(content, configuration)
	require.NoError(t, unmarshalErr)

	return configuration
}
