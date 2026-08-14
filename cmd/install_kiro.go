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
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	clihelpers "go.nwlabs.dev/cli-helpers/v2"
	"go.nwlabs.dev/magneto/internal/kirofiles"
)

const (
	defaultMCPServerName = "magneto"
	mcpTemplatePath      = "settings/mcp.json"

	kiroDirectoryPermissions = 0o755
	kiroFilePermissions      = 0o666
)

var (
	fInstallKiroWorkspace bool
	fInstallKiroUser      bool
	fMCPServerName        string

	installKiroCmd = &cobra.Command{
		Use:   "kiro",
		Short: "Install Kiro integration files",
		Long: clihelpers.LongHelpText(`
		Installs Magneto integration files for the Kiro IDE.
		`),
		RunE: func(_ *cobra.Command, _ []string) error {
			input, inputErr := resolveInstallKiroInput()
			if inputErr != nil {
				return fmt.Errorf("resolve Kiro installation input: %w", inputErr)
			}

			installErr := runInstallKiro(input)
			if installErr != nil {
				return fmt.Errorf("run Kiro installation: %w", installErr)
			}

			return nil
		},
	}
)

type (
	// InstallKiroInput contains the validated settings for a Kiro installation.
	InstallKiroInput struct {
		TargetDir  string
		ServerName string
	}
)

func init() { // lint:allow_init
	installCmd.AddCommand(installKiroCmd)

	installKiroCmd.Flags().BoolVarP(
		&fInstallKiroWorkspace,
		"workspace",
		"w",
		false,
		"Install to the current working directory",
	)
	installKiroCmd.Flags().BoolVarP(&fInstallKiroUser, "user", "u", false, "Install to the user's home directory")
	installKiroCmd.Flags().StringVarP(
		&fMCPServerName,
		"mcp-server-name",
		"S",
		defaultMCPServerName,
		"MCP server name in mcpServers",
	)
}

// resolveInstallKiroInput validates the location and server-name flags, then
// resolves their values into an installation input.
func resolveInstallKiroInput() (*InstallKiroInput, error) {
	targetDir, targetErr := resolveInstallKiroTargetDir()
	if targetErr != nil {
		return nil, fmt.Errorf("resolve Kiro target directory: %w", targetErr)
	}

	validationErr := kirofiles.ValidateServerName(fMCPServerName)
	if validationErr != nil {
		return nil, fmt.Errorf("%w: %w", ErrMCPServerNameInvalid, validationErr)
	}

	return &InstallKiroInput{
		TargetDir:  targetDir,
		ServerName: fMCPServerName,
	}, nil
}

// resolveInstallKiroTargetDir requires exactly one target flag, resolves the
// corresponding path, and verifies that the path is an existing directory.
func resolveInstallKiroTargetDir() (string, error) {
	if fInstallKiroWorkspace && fInstallKiroUser {
		return "", fmt.Errorf("%w: --workspace and --user", ErrFlagsMutuallyExclusive)
	}

	if !fInstallKiroWorkspace && !fInstallKiroUser {
		return "", fmt.Errorf("%w: --workspace or --user", ErrFlagRequired)
	}

	targetDir, targetErr := selectedInstallKiroTargetDir()
	if targetErr != nil {
		return "", fmt.Errorf("resolve selected Kiro target directory: %w", targetErr)
	}

	targetInfo, statErr := os.Stat(targetDir)
	if statErr != nil {
		return "", fmt.Errorf("%w: target directory %q: %w", ErrFileWrite, targetDir, statErr)
	}

	if !targetInfo.IsDir() {
		return "", fmt.Errorf("%w: target path %q is not a directory", ErrFileWrite, targetDir)
	}

	return targetDir, nil
}

// selectedInstallKiroTargetDir resolves the selected installation location
// before it is verified by resolveInstallKiroTargetDir.
func selectedInstallKiroTargetDir() (string, error) {
	if fInstallKiroUser {
		homeDir := os.Getenv("HOME")
		if homeDir == "" {
			return "", fmt.Errorf("%w: HOME environment variable", ErrFlagRequired)
		}

		return homeDir, nil
	}

	workingDir, workingDirErr := os.Getwd()
	if workingDirErr != nil {
		return "", fmt.Errorf("%w: resolve current working directory: %w", ErrFileWrite, workingDirErr)
	}

	return workingDir, nil
}

// runInstallKiro installs the fixed embedded Kiro asset manifest sequentially.
func runInstallKiro(input *InstallKiroInput) error {
	installedPaths := make([]string, 0, len(kirofiles.Files()))

	for _, manifestPath := range kirofiles.Files() {
		destination := filepath.Join(input.TargetDir, ".kiro", manifestPath)

		parentErr := createInstallKiroParent(destination)
		if parentErr != nil {
			return fmt.Errorf("create parent directory for %q: %w", destination, parentErr)
		}

		content, contentErr := installKiroContent(manifestPath, destination, input.ServerName)
		if contentErr != nil {
			return fmt.Errorf("prepare Kiro file %q: %w", destination, contentErr)
		}

		writeErr := os.WriteFile(destination, content, kiroFilePermissions) // lint:allow_dynamic_filename
		if writeErr != nil {
			return fmt.Errorf("%w: %s: %w", ErrFileWrite, destination, writeErr)
		}

		permissionErr := os.Chmod(destination, kiroFilePermissions) // lint:allow_dynamic_filename
		if permissionErr != nil {
			return fmt.Errorf("%w: %s: %w", ErrFileWrite, destination, permissionErr)
		}

		relativePath, relativeErr := filepath.Rel(input.TargetDir, destination)
		if relativeErr != nil {
			return fmt.Errorf("%w: determine installed file path %q: %w", ErrFileWrite, destination, relativeErr)
		}

		installedPaths = append(installedPaths, relativePath)
	}

	var output strings.Builder

	for _, installedPath := range installedPaths {
		fmt.Fprint(&output, installedPath, "\n")
	}

	fmt.Fprint(os.Stdout, output.String())

	return nil
}

// createInstallKiroParent creates the directory that will contain a Kiro asset.
func createInstallKiroParent(destination string) error {
	parentDir := filepath.Dir(destination)

	mkdirErr := os.MkdirAll(parentDir, kiroDirectoryPermissions) // lint:allow_dynamic_filename
	if mkdirErr != nil {
		return fmt.Errorf("%w: %s: %w", ErrFileWrite, parentDir, mkdirErr)
	}

	kiroDir := filepath.Dir(parentDir)

	permissionErr := os.Chmod(kiroDir, kiroDirectoryPermissions) // lint:allow_dynamic_filename
	if permissionErr != nil {
		return fmt.Errorf("%w: %s: %w", ErrFileWrite, kiroDir, permissionErr)
	}

	permissionErr = os.Chmod(parentDir, kiroDirectoryPermissions) // lint:allow_dynamic_filename
	if permissionErr != nil {
		return fmt.Errorf("%w: %s: %w", ErrFileWrite, parentDir, permissionErr)
	}

	return nil
}

// installKiroContent obtains an embedded asset or merges the MCP configuration.
func installKiroContent(manifestPath, destination, serverName string) ([]byte, error) {
	content, contentErr := kirofiles.Content(manifestPath)
	if contentErr != nil {
		return nil, fmt.Errorf("%w: embedded file %q: %w", ErrFileWrite, manifestPath, contentErr)
	}

	if manifestPath != mcpTemplatePath {
		return content, nil
	}

	existing, readErr := readInstallKiroMCPConfiguration(destination)
	if readErr != nil {
		return nil, fmt.Errorf("read MCP configuration %q: %w", destination, readErr)
	}

	merged, mergeErr := kirofiles.MergeMCPConfig(&kirofiles.MergeInput{
		Existing:   existing,
		ServerName: serverName,
		Definition: content,
	})
	if mergeErr != nil {
		return nil, fmt.Errorf("%w: %s: %w", ErrMCPConfigParse, destination, mergeErr)
	}

	return merged.Output, nil
}

// readInstallKiroMCPConfiguration reads an existing configuration if present.
func readInstallKiroMCPConfiguration(path string) ([]byte, error) {
	existing, readErr := os.ReadFile(path) // lint:allow_dynamic_filename
	if readErr == nil {
		return existing, nil
	}

	if errors.Is(readErr, os.ErrNotExist) {
		return nil, nil
	}

	return nil, fmt.Errorf("%w: %s: %w", ErrFileWrite, path, readErr)
}
