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
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/server"
	"github.com/spf13/cobra"

	clihelpers "go.nwlabs.dev/cli-helpers/v2"
)

var (
	// workspaceRoot is the base directory from which artifact file paths are
	// resolved. It is read from the WORKSPACE_ROOT environment variable at
	// server startup.
	workspaceRoot string

	serveCmd = &cobra.Command{
		Use:   "serve",
		Short: "Start the Magneto MCP server over stdio",
		Long: clihelpers.LongHelpText(`
		Starts the Magneto MCP server communicating over stdin/stdout using the
		MCP stdio transport.
		`),
		RunE: func(cmd *cobra.Command, _ []string) error {
			workspaceRoot = os.Getenv("WORKSPACE_ROOT")
			if workspaceRoot == "" {
				return fmt.Errorf("%w", ErrWorkspaceRootNotSet)
			}

			return runStdioServer(cmd.Context())
		},
	}
)

func init() { // lint:allow_init
	rootCmd.AddCommand(serveCmd)
}

// newMCPServer creates and configures the MCP server with all registered tools.
func newMCPServer() *server.MCPServer {
	s := server.NewMCPServer(
		"magneto",
		"1.0.0",
		server.WithToolCapabilities(true),
	)

	s.AddTool(validateCitationTool, handleValidateCitation)
	s.AddTool(validateFindingsBatchTool, handleValidateFindingsBatch)
	s.AddTool(validateFindingSchemaTool, handleValidateFindingSchema)
	s.AddTool(finalizeReviewSessionTool, handleFinalizeReviewSession)

	return s
}

// runStdioServer starts the MCP server listening on stdin/stdout.
func runStdioServer(ctx context.Context) error {
	s := newMCPServer()
	stdio := server.NewStdioServer(s)

	listenErr := stdio.Listen(ctx, os.Stdin, os.Stdout)
	if listenErr != nil {
		return fmt.Errorf("stdio server listen failed: %w", listenErr)
	}

	return nil
}
