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
	"strings"
	"time"

	"github.com/spf13/cobra"

	clihelpers "go.nwlabs.dev/cli-helpers/v2"
	"go.nwlabs.dev/magneto/internal/models"
	"go.nwlabs.dev/magneto/internal/output"
	"go.nwlabs.dev/magneto/internal/session"
	"go.nwlabs.dev/magneto/internal/trigger"
)

var (
	fSpecName string
	fDomain   string

	reviewCmd = &cobra.Command{
		Use:   "review [artifact-path]",
		Short: "Run adversarial review on a design artifact",
		Long: clihelpers.LongHelpText(`
		Orchestrates the adversarial review pipeline: trigger classification,
		review rounds, citation validation, and output generation.
		`),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runReview(cmd.Context(), args[0])
		},
	}
)

func init() { // lint:allow_init
	rootCmd.AddCommand(reviewCmd)

	reviewCmd.Flags().StringVar(&fSpecName, "spec-name", "", "Name of the spec being reviewed")
	reviewCmd.Flags().StringVar(&fDomain, "domain", "", "Blast-radius domain of the artifact")
}

// runReview orchestrates the full adversarial review pipeline for the given
// artifact. It classifies the artifact against trigger heuristics, initializes
// session state, coordinates the review framework, and writes the final output.
func runReview(ctx context.Context, artifactPath string) error {
	_ = ctx

	classifyResult := trigger.Classify(&trigger.ClassifyInput{
		Artifact: &trigger.ArtifactInfo{
			Domain: fDomain,
		},
	})

	if classifyResult.Decision == trigger.DecisionSkip {
		var b strings.Builder

		fmt.Fprintf(&b, "Skipping review: %s\n", classifyResult.Reason)
		fmt.Fprint(os.Stdout, b.String())

		return nil
	}

	rm := session.NewRoundManager()
	dt := session.NewDegradationTracker()

	sessionOutput := &models.ReviewSessionOutput{
		Metadata: models.SessionMetadata{
			SpecName:     fSpecName,
			ArtifactPath: artifactPath,
			Timestamp:    time.Now().UTC().Format(time.RFC3339),
		},
	}

	// The actual review loop is driven by the agent system calling MCP tools
	// (validate_citation, etc.). This command sets up the framework and
	// finalizes output after rounds complete.
	sessionOutput.Metadata.TerminalStatus = dt.AllowedTerminalStatus(models.TerminalNotApproved)
	sessionOutput.Metadata.RoundsExecuted = rm.RoundsExecuted()
	sessionOutput.Metadata.DegradedComponents = dt.Entries()
	sessionOutput.Findings = rm.AllFindings()

	rendered := output.RenderSession(sessionOutput)

	wsRoot := os.Getenv("WORKSPACE_ROOT")
	if wsRoot == "" {
		wsRoot = "."
	}

	filename, filenameErr := output.GenerateFilename(&output.FilenameInput{
		SpecName:      fSpecName,
		WorkspaceRoot: wsRoot,
		Timestamp:     time.Now().UTC(),
	})
	if filenameErr != nil {
		return fmt.Errorf("%w: %w", ErrToolExecution, filenameErr)
	}

	writeErr := os.WriteFile(filename, []byte(rendered), 0o0666)
	if writeErr != nil {
		return fmt.Errorf("%w: %s", ErrFileRead, filename)
	}

	var out strings.Builder

	fmt.Fprintf(&out, "Review written to %s\n", filename)
	fmt.Fprint(os.Stdout, out.String())

	return nil
}
