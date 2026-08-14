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

	"github.com/mark3labs/mcp-go/mcp"

	"go.nwlabs.dev/magneto/internal/citation"
	"go.nwlabs.dev/magneto/internal/models"
	"go.nwlabs.dev/magneto/internal/schema"
)

var (
	// validateCitationTool validates a single finding's citation against the
	// artifact on disk.
	validateCitationTool = mcp.NewTool(
		"validate_citation",
		mcp.WithDescription("Validates that a quoted excerpt exists at the cited location in the artifact"),
		mcp.WithString(
			"quoted_excerpt",
			mcp.Required(),
			mcp.Description("The literal text claimed to be in the artifact"),
		),
		mcp.WithString(
			"file_path",
			mcp.Required(),
			mcp.Description("Path to the artifact file, relative to workspace root"),
		),
		mcp.WithString(
			"section_reference",
			mcp.Required(),
			mcp.Description("Section heading name or line range"),
		),
	)

	// validateFindingsBatchTool validates citations for an array of findings in
	// one call.
	validateFindingsBatchTool = mcp.NewTool(
		"validate_findings_batch",
		mcp.WithDescription("Validates citations for an array of findings in one call"),
		mcp.WithArray(
			"findings",
			mcp.Required(),
			mcp.Description("Array of finding objects"),
		),
	)

	// validateFindingSchemaTool validates that a finding conforms to the
	// required ReviewFinding structure.
	validateFindingSchemaTool = mcp.NewTool(
		"validate_finding_schema",
		mcp.WithDescription("Validates that a finding conforms to the required ReviewFinding structure"),
		mcp.WithObject(
			"finding",
			mcp.Required(),
			mcp.Description("The finding object to validate"),
		),
	)
)

// handleValidateCitation processes a validate_citation tool call by extracting
// the input arguments, running citation validation, and returning a structured
// JSON result.
func handleValidateCitation(
	ctx context.Context,
	request mcp.CallToolRequest, // lint:allow_large_memory
) (*mcp.CallToolResult, error) {
	quotedExcerpt, excerptErr := request.RequireString("quoted_excerpt")
	if excerptErr != nil {
		return nil, fmt.Errorf("%w: %w", ErrToolInputInvalid, excerptErr)
	}

	filePath, pathErr := request.RequireString("file_path")
	if pathErr != nil {
		return nil, fmt.Errorf("%w: %w", ErrToolInputInvalid, pathErr)
	}

	sectionRef, sectionErr := request.RequireString("section_reference")
	if sectionErr != nil {
		return nil, fmt.Errorf("%w: %w", ErrToolInputInvalid, sectionErr)
	}

	input := &citation.ValidateInput{
		QuotedExcerpt:    quotedExcerpt,
		FilePath:         filePath,
		SectionReference: sectionRef,
		WorkspaceRoot:    workspaceRoot,
	}

	result, validateErr := citation.Validate(ctx, input)
	if validateErr != nil {
		return nil, fmt.Errorf("%w: %w", ErrToolExecution, validateErr)
	}

	resultJSON, marshalErr := json.Marshal(result)
	if marshalErr != nil {
		return nil, fmt.Errorf("%w: %w", ErrToolExecution, marshalErr)
	}

	return mcp.NewToolResultText(string(resultJSON)), nil
}

// handleValidateFindingsBatch processes a validate_findings_batch tool call by
// extracting the findings array, running batch citation validation, and
// returning structured JSON results.
func handleValidateFindingsBatch(
	ctx context.Context,
	request mcp.CallToolRequest, // lint:allow_large_memory
) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	findingsRaw, ok := args["findings"]
	if !ok {
		return nil, fmt.Errorf("%w: findings argument is required", ErrToolInputInvalid)
	}

	findingsJSON, marshalErr := json.Marshal(findingsRaw)
	if marshalErr != nil {
		return nil, fmt.Errorf("%w: failed to marshal findings argument", ErrToolInputInvalid)
	}

	var findings []citation.ValidateInput

	unmarshalErr := json.Unmarshal(findingsJSON, &findings)
	if unmarshalErr != nil {
		return nil, fmt.Errorf("%w: failed to parse findings array", ErrToolInputInvalid)
	}

	batchInput := &citation.BatchInput{
		WorkspaceRoot: workspaceRoot,
		Findings:      findings,
	}

	results, validateErr := citation.ValidateBatch(ctx, batchInput)
	if validateErr != nil {
		return nil, fmt.Errorf("%w: %w", ErrToolExecution, validateErr)
	}

	resultJSON, resultMarshalErr := json.Marshal(results)
	if resultMarshalErr != nil {
		return nil, fmt.Errorf("%w: %w", ErrToolExecution, resultMarshalErr)
	}

	return mcp.NewToolResultText(string(resultJSON)), nil
}

// handleValidateFindingSchema processes a validate_finding_schema tool call by
// extracting the finding object, validating it against the ReviewFinding
// schema, and returning a pass/fail result.
func handleValidateFindingSchema(
	_ context.Context,
	request mcp.CallToolRequest, // lint:allow_large_memory
) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	findingRaw, ok := args["finding"]
	if !ok {
		return nil, fmt.Errorf("%w: finding argument is required", ErrToolInputInvalid)
	}

	findingJSON, marshalErr := json.Marshal(findingRaw)
	if marshalErr != nil {
		return nil, fmt.Errorf("%w: failed to marshal finding argument", ErrToolInputInvalid)
	}

	var finding models.ReviewFinding

	unmarshalErr := json.Unmarshal(findingJSON, &finding)
	if unmarshalErr != nil {
		return nil, fmt.Errorf("%w: failed to parse finding object", ErrToolInputInvalid)
	}

	schemaErr := schema.ValidateFindingSchema(&finding)

	resultMap := map[string]any{
		"valid": schemaErr == nil,
	}

	if schemaErr != nil {
		resultMap["error"] = schemaErr.Error()
	}

	resultJSON, resultMarshalErr := json.Marshal(resultMap)
	if resultMarshalErr != nil {
		return nil, fmt.Errorf("%w: %w", ErrToolExecution, resultMarshalErr)
	}

	return mcp.NewToolResultText(string(resultJSON)), nil
}
