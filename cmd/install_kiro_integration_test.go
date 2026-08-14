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
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"testing"

	"github.com/go-openapi/testify/assert"
	"github.com/go-openapi/testify/require"
	"pgregory.net/rapid"

	"go.nwlabs.dev/magneto/internal/kirofiles"
)

const (
	minimumInstallationManifestChecks = 100
	unlistedKiroSourcePath            = "steering/not-in-manifest.md"
)

type installedKiroFile struct {
	Content []byte
	Mode    fs.FileMode
}

// TestInstallKiro_IntegratesManifestAssets verifies that installation writes
// only the fixed manifest, preserves unrelated MCP data, is idempotent,
// overwrites current assets, and retains unlisted legacy files.
func TestInstallKiro_IntegratesManifestAssets(t *testing.T) {
	targetDir := t.TempDir()
	mcpPath := filepath.Join(targetDir, ".kiro", filepath.FromSlash(mcpTemplatePath))
	legacyPath := filepath.Join(targetDir, ".kiro", "steering", "retired-guidance.md")
	originalMCP := []byte(
		`{"metadata":{"owner":"team"},"mcpServers":` +
			`{"external":{"command":"other"},"oldMagneto":{"command":"obsolete"}}}`,
	)

	mkdirErr := os.MkdirAll(filepath.Dir(mcpPath), kiroDirectoryPermissions)
	require.NoError(t, mkdirErr)

	writeErr := os.WriteFile(mcpPath, originalMCP, kiroFilePermissions)
	require.NoError(t, writeErr)

	expectedMCP := mergeMCPConfigurationForTest(t, originalMCP)
	output, installErr := captureInstallKiroStdout(t, func() error {
		return runInstallKiro(&InstallKiroInput{TargetDir: targetDir, ServerName: defaultMCPServerName})
	})

	require.NoError(t, installErr)
	assert.Equal(t, manifestOutputForTest(), output)

	assertInstalledManifestForTest(t, targetDir, expectedMCP)
	assertUnrelatedMCPDataForTest(t, targetDir)
	assert.Equal(t, manifestPathsForTest(), installedManifestPathsForTest(t, targetDir))

	writeErr = os.WriteFile(legacyPath, []byte("retained legacy guidance"), kiroFilePermissions)
	require.NoError(t, writeErr)
	assert.Equal(t, []byte("retained legacy guidance"), readKiroFileForTest(t, legacyPath))

	firstInstallation := snapshotInstalledManifestForTest(t, targetDir)

	_, installErr = captureInstallKiroStdout(t, func() error {
		return runInstallKiro(&InstallKiroInput{TargetDir: targetDir, ServerName: defaultMCPServerName})
	})

	require.NoError(t, installErr)
	assert.Equal(t, firstInstallation, snapshotInstalledManifestForTest(t, targetDir))

	overwriteCurrentManifestForTest(t, targetDir)

	overwrittenMCP := []byte(`{"mcpServers":{"external":{"command":"other"},"magneto":{"command":"obsolete"}}}`)

	expectedMCP = mergeMCPConfigurationForTest(t, overwrittenMCP)
	writeErr = os.WriteFile(mcpPath, overwrittenMCP, kiroFilePermissions)
	require.NoError(t, writeErr)

	_, installErr = captureInstallKiroStdout(t, func() error {
		return runInstallKiro(&InstallKiroInput{TargetDir: targetDir, ServerName: defaultMCPServerName})
	})
	require.NoError(t, installErr)

	assertInstalledManifestForTest(t, targetDir, expectedMCP)
	assert.Equal(t, []byte("retained legacy guidance"), readKiroFileForTest(t, legacyPath))
}

// TestProperty_InstallationHasFixedManifestScopeAndContent verifies Property 6:
// fixed manifest scope and file content integrity.
//
// For arbitrary pre-existing asset content and an independently missing-parent
// target, installation writes only the exact manifest and restores embedded
// asset bytes. The unlisted source candidate is neither embedded nor written.
//
// **Validates: Requirements 3.1, 3.2, 4.1, 4.2, 4.3, 4.4, 5.5, 5.6, 7.1, 7.2,
// 7.4**.
func TestProperty_InstallationHasFixedManifestScopeAndContent(t *testing.T) {
	checks := 0
	propertyRoot := t.TempDir()

	rapid.Check(t, func(rt *rapid.T) {
		checks++

		populatedTarget := filepath.Join(propertyRoot, "populated", strconv.Itoa(checks))
		missingParentTarget := filepath.Join(propertyRoot, "missing", strconv.Itoa(checks))
		preExistingMCP := drawPreExistingMCPConfiguration(rt)

		writePreExistingManifestAssets(rt, populatedTarget, preExistingMCP)

		populatedExpectedMCP := mergeMCPConfigurationForRapidTest(rt, preExistingMCP)

		installErr := runInstallKiro(&InstallKiroInput{
			TargetDir:  populatedTarget,
			ServerName: defaultMCPServerName,
		})
		if installErr != nil {
			rt.Fatal(installErr)
		}

		assertInstalledManifestForRapidTest(rt, populatedTarget, populatedExpectedMCP)

		missingExpectedMCP := mergeMCPConfigurationForRapidTest(rt, nil)

		installErr = runInstallKiro(&InstallKiroInput{
			TargetDir:  missingParentTarget,
			ServerName: defaultMCPServerName,
		})
		if installErr != nil {
			rt.Fatal(installErr)
		}

		assertInstalledManifestForRapidTest(rt, missingParentTarget, missingExpectedMCP)
	})

	if checks < minimumInstallationManifestChecks {
		t.Fatalf("expected at least %d Rapid checks, ran %d", minimumInstallationManifestChecks, checks)
	}
}

func manifestPathsForTest() []string {
	return []string{
		"hooks/adversarial-review-trigger.json",
		"settings/mcp.json",
		"steering/adversarial-review-anti-patterns.md",
		"steering/adversarial-review-architecture-constraints.md",
		"steering/adversarial-review-rubric.md",
	}
}

func manifestOutputForTest() string {
	var output bytes.Buffer

	for _, path := range manifestPathsForTest() {
		output.WriteString(filepath.Join(".kiro", filepath.FromSlash(path)))
		output.WriteByte('\n')
	}

	return output.String()
}

func mergeMCPConfigurationForTest(t *testing.T, existing []byte) []byte {
	t.Helper()

	result, mergeErr := kirofiles.MergeMCPConfig(&kirofiles.MergeInput{
		Existing:   existing,
		ServerName: defaultMCPServerName,
		Definition: kirofiles.MCPTemplate(),
	})
	require.NoError(t, mergeErr)

	return result.Output
}

func mergeMCPConfigurationForRapidTest(rt *rapid.T, existing []byte) []byte {
	result, mergeErr := kirofiles.MergeMCPConfig(&kirofiles.MergeInput{
		Existing:   existing,
		ServerName: defaultMCPServerName,
		Definition: kirofiles.MCPTemplate(),
	})
	if mergeErr != nil {
		rt.Fatal(mergeErr)
	}

	return result.Output
}

func assertInstalledManifestForTest(t *testing.T, targetDir string, expectedMCP []byte) {
	t.Helper()

	assert.Equal(t, manifestPathsForTest(), kirofiles.Files())

	for _, path := range manifestPathsForTest() {
		destination := filepath.Join(targetDir, ".kiro", filepath.FromSlash(path))
		content := readKiroFileForTest(t, destination)
		expected := expectedEmbeddedContentForTest(t, path, expectedMCP)

		assert.Equal(t, expected, content, "unexpected content at %q", path)

		info, statErr := os.Stat(destination)
		require.NoError(t, statErr)
		assert.Equal(t, fs.FileMode(kiroFilePermissions), info.Mode().Perm(), "unexpected mode at %q", path)
	}

	assertNoUnlistedSourceFileForTest(t, targetDir)
}

func assertUnrelatedMCPDataForTest(t *testing.T, targetDir string) {
	t.Helper()

	content := readKiroFileForTest(t, filepath.Join(targetDir, ".kiro", filepath.FromSlash(mcpTemplatePath)))
	configuration := make(map[string]json.RawMessage)
	unmarshalErr := json.Unmarshal(content, &configuration)
	require.NoError(t, unmarshalErr)

	assert.JSONEq(t, `{"owner":"team"}`, string(configuration["metadata"]))

	servers := make(map[string]json.RawMessage)

	unmarshalErr = json.Unmarshal(configuration["mcpServers"], &servers)
	require.NoError(t, unmarshalErr)
	assert.JSONEq(t, `{"command":"other"}`, string(servers["external"]))
	assert.NotContains(t, servers, "oldMagneto")
}

func expectedEmbeddedContentForTest(t *testing.T, path string, expectedMCP []byte) []byte {
	t.Helper()

	if path == mcpTemplatePath {
		return expectedMCP
	}

	content, contentErr := kirofiles.Content(path)
	require.NoError(t, contentErr)

	return content
}

func installedManifestPathsForTest(t *testing.T, targetDir string) []string {
	t.Helper()

	return installedManifestPaths(t, filepath.Join(targetDir, ".kiro"))
}

func installedManifestPaths(tb testing.TB, kiroDir string) []string {
	tb.Helper()

	paths := make([]string, 0, len(manifestPathsForTest()))
	walkErr := filepath.WalkDir(kiroDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if entry.IsDir() {
			return nil
		}

		relativePath, relativeErr := filepath.Rel(kiroDir, path)
		if relativeErr != nil {
			return fmt.Errorf("determine installed relative path: %w", relativeErr)
		}

		paths = append(paths, filepath.ToSlash(relativePath))

		return nil
	})
	require.NoError(tb, walkErr)

	return paths
}

func assertNoUnlistedSourceFileForTest(t *testing.T, targetDir string) {
	t.Helper()

	_, contentErr := kirofiles.Content(unlistedKiroSourcePath)
	require.Error(t, contentErr)
	assert.True(t, errors.Is(contentErr, fs.ErrNotExist))

	_, statErr := os.Stat(filepath.Join(targetDir, ".kiro", filepath.FromSlash(unlistedKiroSourcePath)))
	assert.True(t, errors.Is(statErr, os.ErrNotExist))
}

func readKiroFileForTest(t *testing.T, path string) []byte {
	t.Helper()

	content, readErr := os.ReadFile(path) // lint:allow_dynamic_filename
	require.NoError(t, readErr)

	return content
}

func snapshotInstalledManifestForTest(t *testing.T, targetDir string) map[string]installedKiroFile {
	t.Helper()

	snapshot := make(map[string]installedKiroFile, len(manifestPathsForTest()))

	for _, path := range manifestPathsForTest() {
		destination := filepath.Join(targetDir, ".kiro", filepath.FromSlash(path))
		content := readKiroFileForTest(t, destination)
		info, statErr := os.Stat(destination)
		require.NoError(t, statErr)

		snapshot[path] = installedKiroFile{Content: content, Mode: info.Mode().Perm()}
	}

	return snapshot
}

func overwriteCurrentManifestForTest(t *testing.T, targetDir string) {
	t.Helper()

	for _, path := range manifestPathsForTest() {
		if path == mcpTemplatePath {
			continue
		}

		destination := filepath.Join(targetDir, ".kiro", filepath.FromSlash(path))
		writeErr := os.WriteFile(destination, []byte("modified current asset"), kiroFilePermissions)
		require.NoError(t, writeErr)
	}
}

func drawPreExistingMCPConfiguration(rt *rapid.T) []byte {
	externalName := "external" + rapid.StringMatching(`[A-Za-z0-9]{1,16}`).Draw(rt, "external_name")
	owner := rapid.StringMatching(`[A-Za-z0-9]{1,16}`).Draw(rt, "owner")
	command := rapid.StringMatching(`[A-Za-z0-9]{1,16}`).Draw(rt, "command")
	configuration := map[string]any{
		"metadata": map[string]string{"owner": owner},
		"mcpServers": map[string]any{
			externalName:    map[string]string{"command": command},
			"legacyMagneto": map[string]string{"command": "obsolete"},
		},
	}

	content, marshalErr := json.Marshal(configuration)
	if marshalErr != nil {
		rt.Fatal(marshalErr)
	}

	return content
}

func writePreExistingManifestAssets(rt *rapid.T, targetDir string, mcpContent []byte) {
	for _, path := range manifestPathsForTest() {
		destination := filepath.Join(targetDir, ".kiro", filepath.FromSlash(path))
		parentDir := filepath.Dir(destination)

		mkdirErr := os.MkdirAll(parentDir, kiroDirectoryPermissions)
		if mkdirErr != nil {
			rt.Fatal(mkdirErr)
		}

		content := []byte(rapid.StringMatching(`[A-Za-z0-9]{0,64}`).Draw(rt, "existing_"+path))
		if path == mcpTemplatePath {
			content = mcpContent
		}

		writeErr := os.WriteFile(destination, content, kiroFilePermissions)
		if writeErr != nil {
			rt.Fatal(writeErr)
		}
	}
}

func assertInstalledManifestForRapidTest(rt *rapid.T, targetDir string, expectedMCP []byte) {
	manifest := kirofiles.Files()
	if !slices.Equal(manifestPathsForTest(), manifest) {
		rt.Fatalf("unexpected manifest: %q", manifest)
	}

	paths, pathsErr := installedManifestPathsForRapidTest(targetDir)
	if pathsErr != nil {
		rt.Fatal(pathsErr)
	}

	if !slices.Equal(manifestPathsForTest(), paths) {
		rt.Fatalf("installation wrote paths outside the manifest: %q", paths)
	}

	for _, path := range manifest {
		destination := filepath.Join(targetDir, ".kiro", filepath.FromSlash(path))

		content, readErr := os.ReadFile(destination) // lint:allow_dynamic_filename
		if readErr != nil {
			rt.Fatal(readErr)
		}

		expected := expectedMCP
		if path != mcpTemplatePath {
			expected, readErr = kirofiles.Content(path)
			if readErr != nil {
				rt.Fatal(readErr)
			}
		}

		if !bytes.Equal(expected, content) {
			rt.Fatalf("unexpected content at %q", path)
		}

		info, statErr := os.Stat(destination)
		if statErr != nil {
			rt.Fatal(statErr)
		}

		if info.Mode().Perm() != fs.FileMode(kiroFilePermissions) {
			rt.Fatalf("unexpected mode %o at %q", info.Mode().Perm(), path)
		}
	}

	_, contentErr := kirofiles.Content(unlistedKiroSourcePath)
	if !errors.Is(contentErr, fs.ErrNotExist) {
		rt.Fatalf("unlisted source file %q was embedded", unlistedKiroSourcePath)
	}

	_, statErr := os.Stat(filepath.Join(targetDir, ".kiro", filepath.FromSlash(unlistedKiroSourcePath)))
	if !errors.Is(statErr, os.ErrNotExist) {
		rt.Fatalf("unlisted source file %q was installed", unlistedKiroSourcePath)
	}
}

func installedManifestPathsForRapidTest(targetDir string) ([]string, error) {
	kiroDir := filepath.Join(targetDir, ".kiro")
	paths := make([]string, 0, len(manifestPathsForTest()))

	walkErr := filepath.WalkDir(kiroDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		if entry.IsDir() {
			return nil
		}

		relativePath, relativeErr := filepath.Rel(kiroDir, path)
		if relativeErr != nil {
			return fmt.Errorf("determine installed relative path: %w", relativeErr)
		}

		paths = append(paths, filepath.ToSlash(relativePath))

		return nil
	})
	if walkErr != nil {
		return nil, fmt.Errorf("walk installed Kiro directory: %w", walkErr)
	}

	return paths, nil
}
