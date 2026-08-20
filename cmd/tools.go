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
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/mark3labs/mcp-go/mcp"

	"go.nwlabs.dev/magneto/internal/citation"
	"go.nwlabs.dev/magneto/internal/models"
	"go.nwlabs.dev/magneto/internal/output"
	"go.nwlabs.dev/magneto/internal/schema"
	"go.nwlabs.dev/magneto/internal/session"
)

var (
	// validationProvenance retains only the deterministic validation responses
	// issued by this MCP process for the lifetime of the stdio server.
	validationProvenance = &validationProvenanceRegistry{
		schemas:   make(map[string]schemaProvenance),
		citations: make(map[string]citationProvenance),
	}

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
		mcp.WithString(
			"session_id",
			mcp.Description("Canonical review session identifier for provenance correlation"),
		),
		mcp.WithNumber(
			"finding_index",
			mcp.Description("Zero-based index of the finding within the review session"),
		),
		mcp.WithString(
			"schema_provenance_correlation_id",
			mcp.Description("Correlation identifier returned by validate_finding_schema"),
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
			mcp.Description("Array of citation inputs or canonical finding objects"),
		),
		mcp.WithString(
			"session_id",
			mcp.Description("Canonical review session identifier for provenance correlation"),
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
		mcp.WithString(
			"session_id",
			mcp.Description("Canonical review session identifier for provenance correlation"),
		),
		mcp.WithNumber(
			"finding_index",
			mcp.Description("Zero-based index of the finding within the review session"),
		),
	)

	// finalizeReviewSessionTool verifies a terminal session is composed from
	// deterministic gate results. Record rendering and persistence are wired in
	// the following task.
	finalizeReviewSessionTool = mcp.NewTool(
		"finalize_review_session",
		mcp.WithDescription("Validates a terminal review session assembled from deterministic validation results"),
		mcp.WithObject(
			"session",
			mcp.Required(),
			mcp.Description("Terminal review session with citation gate provenance identifiers"),
		),
	)
)

type (
	// validationIdentity identifies a specific finding within a review session
	// for provenance correlation.
	validationIdentity struct {
		FindingIndex *int   `json:"finding_index"` // lint:allow_format
		SessionID    string `json:"session_id"`    // lint:allow_format
	}

	// validationProvenanceRegistry retains deterministic schema and citation
	// provenance for the lifetime of the stdio server.
	validationProvenanceRegistry struct {
		schemas   map[string]schemaProvenance
		citations map[string]citationProvenance
		mu        sync.Mutex
	}

	// schemaProvenance records that a finding passed schema validation with a
	// specific identity.
	schemaProvenance struct {
		Identity  validationIdentity
		SessionID string
		Finding   models.ReviewFinding
		Valid     bool
	}

	// citationProvenance records that a citation was validated against a
	// specific schema provenance and identity.
	citationProvenance struct {
		Identity     validationIdentity
		SchemaID     string
		SessionID    string
		GateResult   models.CitationGateResult
		FindingIndex int
	}

	// canonicalCitationResult extends ValidateResult with provenance
	// correlation fields for canonical review sessions.
	canonicalCitationResult struct {
		SessionID               string `json:"session_id"`                       // lint:allow_format
		SchemaProvenanceID      string `json:"schema_provenance_correlation_id"` // lint:allow_format
		ProvenanceCorrelationID string `json:"provenance_correlation_id"`        // lint:allow_format
		citation.ValidateResult
		FindingIndex  int  `json:"finding_index"`  // lint:allow_format
		SchemaValid   bool `json:"schema_valid"`   // lint:allow_format
		CitationValid bool `json:"citation_valid"` // lint:allow_format
	}

	// canonicalBatchResult extends BatchResult with provenance correlation
	// fields for canonical batch validation.
	canonicalBatchResult struct {
		SessionID               string `json:"session_id"`                       // lint:allow_format
		SchemaProvenanceID      string `json:"schema_provenance_correlation_id"` // lint:allow_format
		ProvenanceCorrelationID string `json:"provenance_correlation_id"`        // lint:allow_format
		citation.BatchResult
		SchemaValid bool `json:"schema_valid"` // lint:allow_format
	}
)

// handleValidateCitation processes a validate_citation tool call by extracting
// the input arguments, running citation validation, and returning a structured
// JSON result.
func handleValidateCitation(
	ctx context.Context,
	request mcp.CallToolRequest, // lint:allow_large_memory
) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	assertionErr := rejectOutcomeAssertions(args)
	if assertionErr != nil {
		return nil, fmt.Errorf("%w", assertionErr)
	}

	input, inputErr := citationInputFromArguments(&request)
	if inputErr != nil {
		return nil, fmt.Errorf("%w", inputErr)
	}

	identity, canonical, identityErr := validationIdentityFromArguments(args)
	if identityErr != nil {
		return nil, fmt.Errorf("%w", identityErr)
	}

	if !canonical {
		legacyResult, legacyErr := validateCitationLegacy(ctx, input)
		if legacyErr != nil {
			return nil, fmt.Errorf("%w", legacyErr)
		}

		return legacyResult, nil
	}

	schemaID, schemaIDErr := requiredStringArgument(args, "schema_provenance_correlation_id")
	if schemaIDErr != nil {
		return nil, fmt.Errorf("%w", schemaIDErr)
	}

	provenanceErr := validationProvenance.requireSchemaForCitation(identity, schemaID, input)
	if provenanceErr != nil {
		return nil, fmt.Errorf("%w", provenanceErr)
	}

	result, validateErr := citation.Validate(ctx, input)
	if validateErr != nil {
		return nil, fmt.Errorf("%w: %w", ErrToolExecution, validateErr)
	}

	response := validationProvenance.recordCitation(identity, schemaID, result)

	toolRes, toolErr := toolResult(response)
	if toolErr != nil {
		return nil, fmt.Errorf("%w", toolErr)
	}

	return toolRes, nil
}

// handleValidateFindingsBatch processes a validate_findings_batch tool call by
// extracting the findings array, running batch citation validation, and
// returning structured JSON results.
func handleValidateFindingsBatch(
	ctx context.Context,
	request mcp.CallToolRequest, // lint:allow_large_memory
) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	assertionErr := rejectOutcomeAssertions(args)
	if assertionErr != nil {
		return nil, fmt.Errorf("%w", assertionErr)
	}

	identity, canonical, identityErr := validationIdentityFromArguments(args)
	if identityErr != nil {
		return nil, fmt.Errorf("%w", identityErr)
	}

	if canonical {
		canonicalRes, canonicalErr := validateCanonicalBatch(ctx, args, identity)
		if canonicalErr != nil {
			return nil, fmt.Errorf("%w", canonicalErr)
		}

		return canonicalRes, nil
	}

	legacyRes, legacyErr := validateLegacyBatch(ctx, args)
	if legacyErr != nil {
		return nil, fmt.Errorf("%w", legacyErr)
	}

	return legacyRes, nil
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

	normalized, schemaErr := schema.DecodeAndNormalizeFinding(findingJSON)

	identity, canonical, identityErr := validationIdentityFromArguments(args)
	if identityErr != nil {
		return nil, fmt.Errorf("%w", identityErr)
	}

	resultMap := map[string]any{
		"valid": schemaErr == nil,
	}

	if normalized != nil {
		resultMap["normalized_finding"] = normalized
	}

	if schemaErr != nil {
		resultMap["error"] = schemaErr.Error()
	}

	if canonical && schemaErr == nil {
		provenanceID, provenanceErr := validationProvenance.recordSchema(identity, normalized)
		if provenanceErr != nil {
			return nil, fmt.Errorf("%w", provenanceErr)
		}

		resultMap["session_id"] = identity.SessionID
		resultMap["finding_index"] = *identity.FindingIndex
		resultMap["provenance_correlation_id"] = provenanceID
	}

	toolRes, toolErr := toolResult(resultMap)
	if toolErr != nil {
		return nil, fmt.Errorf("%w", toolErr)
	}

	return toolRes, nil
}

// handleFinalizeReviewSession verifies terminal data against the provenance
// this MCP process issued, then persists its one terminal review record.
func handleFinalizeReviewSession(
	_ context.Context,
	request mcp.CallToolRequest, // lint:allow_large_memory
) (*mcp.CallToolResult, error) {
	args := request.GetArguments()

	sessionRaw, ok := args["session"]
	if !ok {
		return nil, fmt.Errorf("%w: session argument is required", ErrToolInputInvalid)
	}

	sessionJSON, marshalErr := json.Marshal(sessionRaw)
	if marshalErr != nil {
		return nil, fmt.Errorf("%w: failed to marshal session argument", ErrToolInputInvalid)
	}

	normalized, normalizeErr := normalizeFinalizationSession(sessionJSON)
	if normalizeErr != nil {
		return nil, fmt.Errorf("%w", normalizeErr)
	}

	finalized, finalizeErr := session.FinalizeReviewSession(&session.FinalizeReviewSessionInput{
		Session: normalized,
	})
	if finalizeErr != nil {
		return nil, fmt.Errorf("%w: %w", ErrToolInputInvalid, finalizeErr)
	}

	persisted, persistErr := output.PersistSession(&output.PersistSessionInput{
		Session:       finalized.Session,
		WorkspaceRoot: workspaceRoot,
	})
	if persistErr != nil {
		return nil, fmt.Errorf("%w: %w", ErrToolExecution, persistErr)
	}

	result := map[string]any{
		"terminal_status": finalized.Session.Metadata.TerminalStatus,
		"idempotency_key": finalized.IdempotencyKey,
		"record_path":     persisted.RecordPath,
		"session":         finalized.Session,
		"warnings":        nil,
	}

	toolRes, toolErr := toolResult(result)
	if toolErr != nil {
		return nil, fmt.Errorf("%w", toolErr)
	}

	return toolRes, nil
}

// validateCitationLegacy handles citation validation for non-canonical requests
// that do not include session provenance.
func validateCitationLegacy(ctx context.Context, input *citation.ValidateInput) (*mcp.CallToolResult, error) {
	result, validateErr := citation.Validate(ctx, input)
	if validateErr != nil {
		return nil, fmt.Errorf("%w: %w", ErrToolExecution, validateErr)
	}

	toolRes, toolErr := toolResult(result)
	if toolErr != nil {
		return nil, fmt.Errorf("%w", toolErr)
	}

	return toolRes, nil
}

// validateLegacyBatch handles batch citation validation for non-canonical
// requests that do not include session provenance.
func validateLegacyBatch(
	ctx context.Context,
	args map[string]any,
) (*mcp.CallToolResult, error) {
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

	toolRes, toolErr := toolResult(results)
	if toolErr != nil {
		return nil, fmt.Errorf("%w", toolErr)
	}

	return toolRes, nil
}

// validateCanonicalBatch handles batch citation validation for canonical
// requests with session provenance correlation.
func validateCanonicalBatch(
	ctx context.Context,
	args map[string]any,
	identity validationIdentity,
) (*mcp.CallToolResult, error) {
	findingsRaw, ok := args["findings"]
	if !ok {
		return nil, fmt.Errorf("%w: findings argument is required", ErrToolInputInvalid)
	}

	findingsJSON, marshalErr := json.Marshal(findingsRaw)
	if marshalErr != nil {
		return nil, fmt.Errorf("%w: failed to marshal findings argument", ErrToolInputInvalid)
	}

	var rawFindings []json.RawMessage

	unmarshalErr := json.Unmarshal(findingsJSON, &rawFindings)
	if unmarshalErr != nil {
		return nil, fmt.Errorf("%w: failed to parse findings array", ErrToolInputInvalid)
	}

	results := make([]canonicalBatchResult, 0, len(rawFindings))

	for findingIndex, rawFinding := range rawFindings {
		result, resultErr := validateCanonicalBatchFinding(ctx, rawFinding, identity.SessionID, findingIndex)
		if resultErr != nil {
			return nil, fmt.Errorf("%w", resultErr)
		}

		results = append(results, *result)
	}

	toolRes, toolErr := toolResult(results)
	if toolErr != nil {
		return nil, fmt.Errorf("%w", toolErr)
	}

	return toolRes, nil
}

// validateCanonicalBatchFinding validates a single finding within a canonical
// batch by verifying schema provenance and running citation validation.
func validateCanonicalBatchFinding(
	ctx context.Context,
	rawFinding json.RawMessage,
	sessionID string,
	findingIndex int,
) (*canonicalBatchResult, error) {
	fields := make(map[string]json.RawMessage)

	unmarshalErr := json.Unmarshal(rawFinding, &fields)
	if unmarshalErr != nil {
		return nil, fmt.Errorf("%w: each finding must be a JSON object", ErrToolInputInvalid)
	}

	schemaID, schemaIDErr := requiredRawString(fields, "schema_provenance_correlation_id")
	if schemaIDErr != nil {
		return nil, fmt.Errorf("%w", schemaIDErr)
	}

	indexedIdentity := validationIdentity{
		SessionID:    sessionID,
		FindingIndex: &findingIndex,
	}

	requestedIndex, indexErr := requiredRawInt(fields, "finding_index")
	if indexErr != nil {
		return nil, fmt.Errorf("%w", indexErr)
	}

	if requestedIndex != findingIndex {
		return nil, fmt.Errorf("%w: finding_index must match the batch array position", ErrValidationProvenanceMismatch)
	}

	delete(fields, "schema_provenance_correlation_id")
	delete(fields, "finding_index")

	findingJSON, marshalErr := json.Marshal(fields)
	if marshalErr != nil {
		return nil, fmt.Errorf("%w: failed to marshal canonical finding", ErrToolInputInvalid)
	}

	normalized, schemaErr := schema.DecodeAndNormalizeFinding(findingJSON)
	if schemaErr != nil {
		return nil, fmt.Errorf("%w: %w", ErrToolInputInvalid, schemaErr)
	}

	provenanceErr := validationProvenance.requireSchemaForFinding(indexedIdentity, schemaID, normalized)
	if provenanceErr != nil {
		return nil, fmt.Errorf("%w", provenanceErr)
	}

	input := &citation.ValidateInput{
		QuotedExcerpt:    normalized.QuotedExcerpt,
		FilePath:         normalized.ArtifactLocation.FilePath,
		SectionReference: normalized.ArtifactLocation.SectionReference,
		WorkspaceRoot:    workspaceRoot,
	}

	citationResult, validateErr := citation.Validate(ctx, input)
	if validateErr != nil {
		citationResult = citation.ValidateResult{
			FailureReason: validateErr.Error(),
		}
	}

	response := validationProvenance.recordCitation(indexedIdentity, schemaID, citationResult)

	return &canonicalBatchResult{
		BatchResult: citation.BatchResult{
			FailureReason: response.FailureReason,
			FindingIndex:  findingIndex,
			CitationValid: response.CitationValid,
		},
		SessionID:               response.SessionID,
		SchemaProvenanceID:      response.SchemaProvenanceID,
		ProvenanceCorrelationID: response.ProvenanceCorrelationID,
		SchemaValid:             response.SchemaValid,
	}, nil
}

// citationInputFromArguments extracts and validates the citation input fields
// from a tool call request.
func citationInputFromArguments(request *mcp.CallToolRequest) (*citation.ValidateInput, error) {
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

	return &citation.ValidateInput{
		QuotedExcerpt:    quotedExcerpt,
		FilePath:         filePath,
		SectionReference: sectionRef,
		WorkspaceRoot:    workspaceRoot,
	}, nil
}

// validationIdentityFromArguments extracts session_id and finding_index from
// tool arguments for provenance correlation.
func validationIdentityFromArguments(args map[string]any) (validationIdentity, bool, error) {
	_, hasSessionID := args["session_id"]
	_, hasFindingIndex := args["finding_index"]

	if !hasSessionID && !hasFindingIndex {
		return validationIdentity{}, false, nil
	}

	if !hasSessionID || !hasFindingIndex {
		return validationIdentity{}, false, fmt.Errorf(
			"%w: session_id and finding_index must be supplied together",
			ErrValidationProvenanceMissing,
		)
	}

	identityJSON, marshalErr := json.Marshal(args)
	if marshalErr != nil {
		return validationIdentity{}, false, fmt.Errorf("%w: invalid validation identity", ErrToolInputInvalid)
	}

	identity := validationIdentity{}

	unmarshalErr := json.Unmarshal(identityJSON, &identity)
	if unmarshalErr != nil || strings.TrimSpace(identity.SessionID) == "" || identity.FindingIndex == nil ||
		*identity.FindingIndex < 0 {
		return validationIdentity{}, false, fmt.Errorf(
			"%w: invalid session_id or finding_index",
			ErrValidationProvenanceMissing,
		)
	}

	return identity, true, nil
}

// requiredStringArgument extracts a non-empty string from arguments.
func requiredStringArgument(args map[string]any, key string) (string, error) {
	value, ok := args[key]
	if !ok {
		return "", fmt.Errorf("%w: %s is required", ErrValidationProvenanceMissing, key)
	}

	stringValue, ok := value.(string)
	if !ok || strings.TrimSpace(stringValue) == "" {
		return "", fmt.Errorf("%w: %s must be a non-empty string", ErrValidationProvenanceMissing, key)
	}

	return stringValue, nil
}

// requiredRawString extracts a non-empty string from raw JSON fields.
func requiredRawString(fields map[string]json.RawMessage, key string) (string, error) {
	raw, ok := fields[key]
	if !ok {
		return "", fmt.Errorf("%w: %s is required", ErrValidationProvenanceMissing, key)
	}

	value := ""

	unmarshalErr := json.Unmarshal(raw, &value)
	if unmarshalErr != nil || strings.TrimSpace(value) == "" {
		return "", fmt.Errorf("%w: %s must be a non-empty string", ErrValidationProvenanceMissing, key)
	}

	return value, nil
}

// requiredRawInt extracts a non-negative integer from raw JSON fields.
func requiredRawInt(fields map[string]json.RawMessage, key string) (int, error) {
	raw, ok := fields[key]
	if !ok {
		return 0, fmt.Errorf("%w: %s is required", ErrValidationProvenanceMissing, key)
	}

	value := 0

	unmarshalErr := json.Unmarshal(raw, &value)
	if unmarshalErr != nil || value < 0 {
		return 0, fmt.Errorf("%w: %s must be a non-negative integer", ErrValidationProvenanceMissing, key)
	}

	return value, nil
}

// rejectOutcomeAssertions prevents tool callers from asserting validation or
// confirmation outcomes that are derived by Magneto.
func rejectOutcomeAssertions(args map[string]any) error {
	for _, field := range []string{
		"citation_gate_result",
		"citation_valid",
		"schema_valid",
		"provenance_correlation_id",
		"confirmer_evidence",
		"confirmer_attempts",
		"confirmation_status",
	} {
		if _, exists := args[field]; exists {
			return fmt.Errorf("%w: %s is derived by Magneto", ErrToolOutcomeAssertion, field)
		}
	}

	return nil
}

// toolResult marshals a value into an MCP text result.
func toolResult(value any) (*mcp.CallToolResult, error) {
	resultJSON, marshalErr := json.Marshal(value)
	if marshalErr != nil {
		return nil, fmt.Errorf("%w: %w", ErrToolExecution, marshalErr)
	}

	return mcp.NewToolResultText(string(resultJSON)), nil
}

// recordSchema stores a validated finding schema result and returns a
// provenance correlation identifier.
func (reg *validationProvenanceRegistry) recordSchema(
	identity validationIdentity,
	finding *models.ReviewFinding,
) (string, error) {
	fingerprint, fingerprintErr := findingFingerprint(finding)
	if fingerprintErr != nil {
		return "", fmt.Errorf("%w", fingerprintErr)
	}

	correlationID := correlationID("schema", identity.SessionID, strconv.Itoa(*identity.FindingIndex), fingerprint)
	copyFinding := *finding

	reg.mu.Lock()
	defer reg.mu.Unlock()

	reg.schemas[correlationID] = schemaProvenance{
		Identity:  identity,
		Finding:   copyFinding,
		Valid:     true,
		SessionID: identity.SessionID,
	}

	return correlationID, nil
}

// requireSchemaForCitation verifies that a prior schema validation matches the
// citation being validated.
func (reg *validationProvenanceRegistry) requireSchemaForCitation(
	identity validationIdentity,
	schemaID string,
	input *citation.ValidateInput,
) error {
	reg.mu.Lock()
	defer reg.mu.Unlock()

	record, exists := reg.schemas[schemaID]
	if !exists || !record.Valid || !sameIdentity(record.Identity, identity) ||
		!citationInputMatchesFinding(input, &record.Finding) {
		return fmt.Errorf("%w: schema validation does not match this citation", ErrValidationProvenanceMismatch)
	}

	return nil
}

// requireSchemaForFinding verifies that a prior schema validation matches the
// finding being processed in a batch.
func (reg *validationProvenanceRegistry) requireSchemaForFinding(
	identity validationIdentity,
	schemaID string,
	finding *models.ReviewFinding,
) error {
	fingerprint, fingerprintErr := findingFingerprint(finding)
	if fingerprintErr != nil {
		return fmt.Errorf("%w", fingerprintErr)
	}

	reg.mu.Lock()
	defer reg.mu.Unlock()

	record, exists := reg.schemas[schemaID]
	if !exists || !record.Valid || !sameIdentity(record.Identity, identity) {
		return fmt.Errorf("%w: schema validation does not match this finding", ErrValidationProvenanceMismatch)
	}

	recordFingerprint, recordErr := findingFingerprint(&record.Finding)
	if recordErr != nil || recordFingerprint != fingerprint {
		return fmt.Errorf("%w: schema validation does not match this finding", ErrValidationProvenanceMismatch)
	}

	return nil
}

// recordCitation stores a citation validation result with provenance and
// returns the canonical response with correlation identifiers.
func (reg *validationProvenanceRegistry) recordCitation(
	identity validationIdentity,
	schemaID string,
	result citation.ValidateResult,
) *canonicalCitationResult {
	matchedLines := citationMatchedLines(result.MatchLocation)
	gateResult := models.CitationGateResult{
		MatchedLines:  matchedLines,
		FailureReason: result.FailureReason,
		SchemaValid:   true,
		CitationValid: result.Valid,
	}
	correlationID := correlationID(
		"citation",
		identity.SessionID,
		strconv.Itoa(*identity.FindingIndex),
		schemaID,
		result.FailureReason,
		matchedLinesFingerprint(matchedLines),
		strconv.FormatBool(result.Valid),
	)

	gateResult.ProvenanceCorrelationID = correlationID

	reg.mu.Lock()
	reg.citations[correlationID] = citationProvenance{
		Identity:     identity,
		GateResult:   gateResult,
		SchemaID:     schemaID,
		SessionID:    identity.SessionID,
		FindingIndex: *identity.FindingIndex,
	}
	reg.mu.Unlock()

	return &canonicalCitationResult{
		ValidateResult:          result,
		SessionID:               identity.SessionID,
		SchemaProvenanceID:      schemaID,
		ProvenanceCorrelationID: correlationID,
		FindingIndex:            *identity.FindingIndex,
		SchemaValid:             true,
		CitationValid:           result.Valid,
	}
}

// citationForFinalization retrieves a stored citation gate result for use
// during session finalization.
func (reg *validationProvenanceRegistry) citationForFinalization(
	identity validationIdentity,
	correlationID string,
) (*models.CitationGateResult, error) {
	reg.mu.Lock()
	defer reg.mu.Unlock()

	record, exists := reg.citations[correlationID]
	if !exists || !sameIdentity(record.Identity, identity) {
		return nil, fmt.Errorf("%w: citation validation does not match this finding", ErrValidationProvenanceMismatch)
	}

	gateResult := record.GateResult

	gateResult.MatchedLines = copyCitationMatchedLines(gateResult.MatchedLines)

	return &gateResult, nil
}

// normalizeFinalizationSession decodes and validates terminal session data,
// normalizing each finding through schema validation and provenance
// verification.
func normalizeFinalizationSession(data []byte) (*models.ReviewSessionOutput, error) {
	rawSession := make(map[string]json.RawMessage)

	unmarshalRawErr := json.Unmarshal(data, &rawSession)
	if unmarshalRawErr != nil {
		return nil, fmt.Errorf("%w: session must be a JSON object", ErrToolInputInvalid)
	}

	findingsRaw, ok := rawSession["findings"]
	if !ok {
		return nil, fmt.Errorf("%w: session findings are required", ErrToolInputInvalid)
	}

	var rawFindings []json.RawMessage

	unmarshalFindingsErr := json.Unmarshal(findingsRaw, &rawFindings)
	if unmarshalFindingsErr != nil {
		return nil, fmt.Errorf("%w: session findings must be an array", ErrToolInputInvalid)
	}

	normalizedSession := &models.ReviewSessionOutput{}

	unmarshalSessionErr := json.Unmarshal(data, normalizedSession)
	if unmarshalSessionErr != nil {
		return nil, fmt.Errorf("%w: session contains invalid field types", ErrToolInputInvalid)
	}

	for findingIndex, rawFinding := range rawFindings {
		normalized, normalizeErr := normalizeFinalizationFinding(
			rawFinding,
			normalizedSession.Metadata.SessionID,
			findingIndex,
		)
		if normalizeErr != nil {
			return nil, fmt.Errorf("%w", normalizeErr)
		}

		normalizedSession.Findings[findingIndex] = *normalized
	}

	return normalizedSession, nil
}

// normalizeFinalizationFinding validates a single finding within the terminal
// session by verifying its citation gate provenance.
func normalizeFinalizationFinding(
	rawFinding json.RawMessage,
	sessionID string,
	findingIndex int,
) (*models.ReviewFinding, error) {
	fields := make(map[string]json.RawMessage)

	unmarshalErr := json.Unmarshal(rawFinding, &fields)
	if unmarshalErr != nil {
		return nil, fmt.Errorf("%w: each session finding must be a JSON object", ErrToolInputInvalid)
	}

	gateRaw, ok := fields["citation_gate_result"]
	if !ok {
		return nil, fmt.Errorf("%w: citation gate provenance is required", ErrValidationProvenanceMissing)
	}

	correlationID, correlationErr := gateProvenanceCorrelationID(gateRaw)
	if correlationErr != nil {
		return nil, fmt.Errorf("%w", correlationErr)
	}

	delete(fields, "citation_gate_result")

	findingJSON, marshalErr := json.Marshal(fields)
	if marshalErr != nil {
		return nil, fmt.Errorf("%w: failed to marshal session finding", ErrToolInputInvalid)
	}

	normalized, schemaErr := schema.DecodeAndNormalizeFinding(findingJSON)
	if schemaErr != nil {
		return nil, fmt.Errorf("%w: %w", ErrToolInputInvalid, schemaErr)
	}

	identity := validationIdentity{
		SessionID:    sessionID,
		FindingIndex: &findingIndex,
	}

	gateResult, provenanceErr := validationProvenance.citationForFinalization(identity, correlationID)
	if provenanceErr != nil {
		return nil, fmt.Errorf("%w", provenanceErr)
	}

	normalized.CitationGateResult = gateResult

	transitioned := session.TransitionFindingStatus(&session.FindingStatusTransitionInput{
		GateAvailability: session.GateAvailable,
		Finding:          *normalized,
	})

	return &transitioned, nil
}

// gateProvenanceCorrelationID extracts the provenance correlation identifier
// from a citation gate result object.
func gateProvenanceCorrelationID(raw json.RawMessage) (string, error) {
	gateFields := make(map[string]json.RawMessage)

	unmarshalErr := json.Unmarshal(raw, &gateFields)
	if unmarshalErr != nil {
		return "", fmt.Errorf("%w: citation gate result must be an object", ErrToolInputInvalid)
	}

	if len(gateFields) != 1 {
		return "", fmt.Errorf("%w: citation gate outcomes are derived by Magneto", ErrToolOutcomeAssertion)
	}

	correlationResult, correlationErr := requiredRawString(gateFields, "provenance_correlation_id")
	if correlationErr != nil {
		return "", fmt.Errorf("%w", correlationErr)
	}

	return correlationResult, nil
}

// findingFingerprint computes a SHA-256 digest of the finding's JSON
// representation for deterministic correlation.
func findingFingerprint(finding *models.ReviewFinding) (string, error) {
	findingJSON, marshalErr := json.Marshal(finding)
	if marshalErr != nil {
		return "", fmt.Errorf("%w: failed to encode canonical finding", ErrToolExecution)
	}

	digest := sha256.Sum256(findingJSON)

	return hex.EncodeToString(digest[:]), nil
}

// correlationID produces a deterministic identifier from a prefix and ordered
// values using SHA-256.
func correlationID(prefix string, values ...string) string {
	payload := prefix + "\x00" + strings.Join(values, "\x00")
	digest := sha256.Sum256([]byte(payload))

	return hex.EncodeToString(digest[:])
}

// sameIdentity returns true when both identities reference the same finding
// within the same session.
func sameIdentity(first, second validationIdentity) bool {
	return first.SessionID == second.SessionID && first.FindingIndex != nil && second.FindingIndex != nil &&
		*first.FindingIndex == *second.FindingIndex
}

// citationInputMatchesFinding checks whether a citation input corresponds to
// the same artifact location as a finding.
func citationInputMatchesFinding(input *citation.ValidateInput, finding *models.ReviewFinding) bool {
	return input.QuotedExcerpt == finding.QuotedExcerpt && input.FilePath == finding.ArtifactLocation.FilePath &&
		input.SectionReference == finding.ArtifactLocation.SectionReference
}

// citationMatchedLines converts a citation match location to the model
// representation.
func citationMatchedLines(location *citation.MatchLocation) *models.CitationMatchedLines {
	if location == nil {
		return nil
	}

	return &models.CitationMatchedLines{
		Start: location.LineStart,
		End:   location.LineEnd,
	}
}

// copyCitationMatchedLines creates a defensive copy of matched lines.
func copyCitationMatchedLines(lines *models.CitationMatchedLines) *models.CitationMatchedLines {
	if lines == nil {
		return nil
	}

	copied := *lines

	return &copied
}

// matchedLinesFingerprint produces a stable string representation of matched
// lines for use in correlation identifiers.
func matchedLinesFingerprint(lines *models.CitationMatchedLines) string {
	if lines == nil {
		return ""
	}

	return strconv.Itoa(lines.Start) + ":" + strconv.Itoa(lines.End)
}
