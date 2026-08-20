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

// deprecationNotice is the message emitted when `magneto review` is invoked
// directly. The command remains a compatibility wrapper that neither invokes
// subagents nor claims to run the operational workflow.
const deprecationNotice = "NOTICE: magneto review is deprecated for " +
	"interactive use. Use the Kiro-native MCP finalization workflow " +
	"(finalize_review_session) for operational review orchestration."

var (
	fSpecName string
	fDomain   string

	reviewCmd = &cobra.Command{
		Use:   "review [artifact-path]",
		Short: "Run adversarial review on a design artifact (deprecated)",
		Long: clihelpers.LongHelpText(`
		Non-interactive compatibility wrapper that classifies a design artifact,
		writes a terminal review record, and emits a deprecation notice. It does
		not invoke subagents or run the operational workflow. Use the Kiro-native
		MCP finalization workflow (finalize_review_session) instead.
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

// runReview is a non-interactive compatibility wrapper. It classifies the
// artifact against trigger heuristics, writes a terminal record using canonical
// satisfaction fields, emits a deprecation notice, and returns. It never
// invokes subagents or claims to run the full Kiro-native operational workflow.
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
		fmt.Fprintf(&b, "%s\n", deprecationNotice)
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

	// This compatibility wrapper does not invoke subagents or drive a review
	// loop. It records terminal state from the existing session primitives and
	// writes one record.
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

	fmt.Fprintf(&out, "%s\n", deprecationNotice)
	fmt.Fprintf(&out, "Review written to %s\n", filename)
	fmt.Fprint(os.Stdout, out.String())

	return nil
}
