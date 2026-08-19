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

import "errors"

var (
	// -------------------------------------------------------------------------
	// File I/O errors.

	// ErrFileRead indicates a failure to read the artifact file from disk.
	ErrFileRead = errors.New("failed to read artifact file")

	// ErrFileNotFound indicates the artifact file does not exist at the
	// specified path.
	ErrFileNotFound = errors.New("artifact file not found")

	// -------------------------------------------------------------------------
	// Citation validation error.

	// ErrSectionNotFound indicates the referenced section could not be located
	// in the artifact.
	ErrSectionNotFound = errors.New("referenced section not found in artifact")

	// ErrCitationInvalid indicates the quoted excerpt was not found in the
	// cited section of the artifact.
	ErrCitationInvalid = errors.New("quoted excerpt not found in cited section")

	// ErrCitationMissing indicates a finding lacks required citation fields.
	ErrCitationMissing = errors.New("finding lacks required citation fields")

	// -------------------------------------------------------------------------
	// Schema validation errors.

	// ErrSchemaInvalid indicates a finding does not conform to the required
	// ReviewFinding schema.
	ErrSchemaInvalid = errors.New("finding does not conform to required schema")

	// ErrScoreOutOfRange indicates a finding score is outside the valid 1-10
	// range.
	ErrScoreOutOfRange = errors.New("finding score outside valid range")

	// -------------------------------------------------------------------------
	// Configuration errors.

	// ErrWorkspaceRootNotSet indicates the WORKSPACE_ROOT environment variable
	// is not configured.
	ErrWorkspaceRootNotSet = errors.New("WORKSPACE_ROOT environment variable not set")

	// ErrRubricNotFound indicates no rubric steering file was found.
	ErrRubricNotFound = errors.New("rubric steering file not found")

	// ErrRubricMalformed indicates the rubric steering file has an invalid
	// format.
	ErrRubricMalformed = errors.New("rubric steering file has invalid format")

	// -------------------------------------------------------------------------
	// MCP server errors.

	// ErrToolInputInvalid indicates the MCP tool received invalid input
	// parameters.
	ErrToolInputInvalid = errors.New("MCP tool received invalid input")

	// ErrToolExecution indicates the MCP tool execution encountered an error.
	ErrToolExecution = errors.New("MCP tool execution failed")

	// ErrToolOutcomeAssertion indicates an MCP request attempted to provide a
	// validation or confirmation result reserved for deterministic processing.
	ErrToolOutcomeAssertion = errors.New("MCP tool request asserted a validation or confirmation outcome")

	// ErrValidationProvenanceMissing indicates a canonical validation request
	// did not include the required session and finding correlation data.
	ErrValidationProvenanceMissing = errors.New("deterministic validation provenance is required")

	// ErrValidationProvenanceMismatch indicates submitted correlation data does
	// not identify a matching deterministic validation result.
	ErrValidationProvenanceMismatch = errors.New("deterministic validation provenance does not match this finding")

	// -------------------------------------------------------------------------
	// Kiro installer errors.

	// ErrFlagRequired indicates a required flag was not provided.
	ErrFlagRequired = errors.New("required flag not provided")

	// ErrFlagsMutuallyExclusive indicates mutually exclusive flags were both
	// set.
	ErrFlagsMutuallyExclusive = errors.New("mutually exclusive flags provided")

	// ErrMCPConfigParse indicates the existing mcp.json contains invalid JSON.
	ErrMCPConfigParse = errors.New("failed to parse MCP configuration file")

	// ErrFileWrite indicates a file or directory write operation failed.
	ErrFileWrite = errors.New("failed to write file")

	// ErrMCPServerNameInvalid indicates the MCP server name does not conform
	// to naming conventions.
	ErrMCPServerNameInvalid = errors.New("invalid MCP server name format")
)
