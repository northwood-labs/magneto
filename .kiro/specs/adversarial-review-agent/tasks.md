# Implementation Plan: Adversarial Review Agent (Phase 1)

## Overview

This plan implements the Phase 1 Adversarial Review Agent as a Go CLI application providing an MCP server for citation validation, along with the supporting Kiro configuration files (hooks, steering files) and data models needed for the adversarial review pipeline. The implementation follows the project's Go CLI patterns: minimal `main.go`, Cobra commands in `cmd/`, business logic in `internal/` packages, Apache 2.0 headers, and `pgregory.net/rapid` for property-based testing.

## Tasks

* [x] 1. Initialize project structure and Go module
  * [x] 1.1 Create Go module and directory scaffold
    * Initialize `go.mod` with module path `go.nwlabs.dev/magneto`
    * Create directory structure: `cmd/`, `internal/citation/`, `internal/schema/`, `internal/models/`, `internal/novelty/`, `internal/trigger/`
    * Create `main.go` with minimal entrypoint calling `cmd.Execute()`
    * Create `doc.go` for every package with Apache 2.0 header and package comment
    * _Requirements: 11.2_
  * [x] 1.2 Create `cmd/root.go` with Cobra root command
    * Define `rootCmd` using `cobra.Command` with app name and description
    * Use `charm.land/fang/v2` for `Execute()` wrapper
    * Use `go.nwlabs.dev/x/logutils` for logging with verbosity flag
    * Register `--verbose` / `-v` flag prefixed with `f` (e.g., `fVerbosity`)
    * _Requirements: 11.2_
  * [x] 1.3 Create `cmd/version.go`
    * Use `clihelpers.VersionScreen()` pattern exactly as specified in steering file
    * Register on `rootCmd` in `init()`
    * _Requirements: 11.2_

* [x] 2. Implement shared data models
  * [x] 2.1 Create `internal/models/finding.go`
    * Define `FindingStatus` type and its constants (`StatusConfirmed`, `StatusHypothesized`, `StatusUnconfirmed`, `StatusUnconfirmedInconclusive`, `StatusUncheckedGateUnavail`)
    * Define `ArtifactLocation` struct with JSON tags
    * Define `ReviewFinding` struct with all fields from design (criterion name, score, quoted excerpt, artifact location, status, reasoning, confirmer evidence)
    * _Requirements: 3.1, 4.1_
  * [x] 2.2 Create `internal/models/session.go`
    * Define `TerminalStatus` type and constants (`TerminalNotApproved`, `TerminalApproved`, `TerminalHumanOverride`, `TerminalPartialReview`)
    * Define `ReviewSessionOutput` struct with metadata, findings, escalations, overrides, dead checks, attack round result
    * Define `SessionMetadata`, `HumanEscalation`, `HumanOverride`, `DegradationEntry`, `AttackRoundResult` structs
    * _Requirements: 3.1, 3.4, 7.5, 10.5_
  * [x] 2.3 Write property test for finding schema completeness (Property 4)
    * **Property 4: Finding schema validation rejects incomplete findings**
    * **Validates: Requirements 3.1**
    * Generate arbitrary `ReviewFinding` objects with randomly removed required fields; assert that schema validation rejects them and identifies the missing field(s)

* [x] 3. Implement citation validation logic
  * [x] 3.1 Create `internal/citation/normalize.go`
    * Implement `NormalizeWhitespace` function that collapses runs of whitespace while preserving word boundaries
    * _Requirements: 4.4_
  * [x] 3.2 Create `internal/citation/section.go`
    * Implement `ExtractSection` function that locates Markdown section boundaries by heading name or line range
    * Use `github.com/yuin/goldmark` for Markdown heading detection
    * Define `Section` struct with `Content`, `StartLine`, `EndLine`
    * Handle both heading-based references (e.g., "Architecture") and line-range references (e.g., "lines 45-60")
    * _Requirements: 4.4_
  * [x] 3.3 Create `internal/citation/validate.go`
    * Implement `Validate` function matching the design signature: accepts `ValidateInput`, returns `ValidateResult`
    * Implement `ValidateBatch` function for batch validation
    * Define `ValidateInput`, `ValidateResult`, `MatchLocation`, `BatchInput`, `BatchResult` types
    * Define sentinel errors (e.g., `ErrFileRead`, `ErrSectionNotFound`) in `internal/citation/errors.go`
    * Logic: resolve path, read file, extract section, normalize whitespace, perform substring match, compute line location
    * _Requirements: 4.1, 4.4, 4.5_
  * [x] 3.4 Write property test for citation round-trip (Property 1)
    * **Property 1: Citation validation round-trip**
    * **Validates: Requirements 4.4**
    * Generate arbitrary file content with headings and bodies; extract a substring from within a section; validate it returns `Valid: true`
  * [x] 3.5 Write property test for non-existent citations (Property 2)
    * **Property 2: Non-existent citations always fail**
    * **Validates: Requirements 4.5**
    * Generate file content and an excerpt guaranteed not to be a substring; validate it returns `Valid: false`
  * [x] 3.6 Write property test for whitespace normalization (Property 3)
    * **Property 3: Whitespace normalization preserves match semantics**
    * **Validates: Requirements 4.4**
    * Generate text that exists in a file; add/remove whitespace within runs; validate that the match outcome is preserved
  * [x] 3.7 Write unit tests for citation validation edge cases
    * Table-driven tests for: exact match, substring match, section boundary detection, line range extraction, whitespace normalization (tabs, multiple spaces, trailing newlines), file not found error
    * _Requirements: 4.4, 4.5_

* [x] 4. Checkpoint
  * Ensure all tests pass, ask the user if questions arise.

* [x] 5. Implement schema validation
  * [x] 5.1 Create `internal/schema/validate.go`
    * Implement `ValidateFindingSchema` function that checks a `ReviewFinding` has all required fields (non-empty criterion name, score 1-10, non-empty quoted excerpt, valid artifact location, valid status)
    * Return structured validation errors identifying missing/invalid fields
    * _Requirements: 3.1, 11.2_
  * [x] 5.2 Write unit tests for schema validation
    * Table-driven tests for: valid finding passes, missing criterion name, score out of range (0, 11, -1), empty quoted excerpt, missing file path, missing section reference, invalid status value
    * _Requirements: 3.1_

* [x] 6. Implement novelty check logic
  * [x] 6.1 Create `internal/novelty/check.go`
    * Implement `Check` function that compares current round findings against prior round findings
    * A round is non-novel if every finding references a criterion and evidence already present in prior rounds
    * New criterion, new evidence, or different score with new evidence counts as novel
    * _Requirements: 6.2_
  * [x] 6.2 Write property test for novelty subset detection (Property 5)
    * **Property 5: Novelty check detects subset repetition**
    * **Validates: Requirements 6.2**
    * Generate a set of findings for round N; create a subset for round N+1; assert novelty check returns `novel: false`
  * [x] 6.3 Write unit tests for novelty check
    * Table-driven tests for: identical findings = non-novel, subset = non-novel, new criterion = novel, same criterion with new evidence = novel, completely different findings = novel
    * _Requirements: 6.2_

* [x] 7. Implement trigger classification logic
  * [x] 7.1 Create `internal/trigger/classify.go`
    * Implement `Classify` function that determines whether an artifact should trigger review
    * Check blast-radius domain list from steering file configuration
    * Check foundational trust (artifact consumed by downstream automation without independent verification)
    * Check skip conditions (single file, revertible, human-reviewed before consumption)
    * Default ambiguous cases to triggering review
    * _Requirements: 2.1, 2.2, 2.3, 2.5_
  * [x] 7.2 Write unit tests for trigger classification
    * Table-driven tests for: blast-radius domain match triggers, foundational trust triggers, single-file revertible skips, ambiguous defaults to trigger
    * _Requirements: 2.1, 2.2, 2.3, 2.5_

* [x] 8. Checkpoint
  * Ensure all tests pass, ask the user if questions arise.

* [x] 9. Implement MCP server commands
  * [x] 9.1 Create `cmd/serve.go` with MCP stdio server command
    * Define `serveCmd` as a Cobra command that starts the MCP server over stdio
    * Instantiate MCP server using `github.com/mark3labs/mcp-go/server` with tool capabilities
    * Register `validate_citation`, `validate_findings_batch`, and `validate_finding_schema` tools
    * Read `WORKSPACE_ROOT` from environment variable
    * _Requirements: 4.3, 11.2_
  * [x] 9.2 Create `cmd/tools.go` with MCP tool definitions and handlers
    * Define `validateCitationTool`, `validateFindingsBatchTool`, `validateFindingSchemaTool` using `mcp.NewTool`
    * Implement `handleValidateCitation` that parses input, calls `citation.Validate`, returns structured result
    * Implement `handleValidateFindingsBatch` that parses array input, calls `citation.ValidateBatch`
    * Implement `handleValidateFindingSchema` that parses finding object, calls `schema.ValidateFindingSchema`
    * _Requirements: 4.1, 4.3, 4.4, 4.5, 11.2_
  * [x] 9.3 Create `cmd/errors.go` with sentinel errors
    * Define all sentinel errors in a single `var()` block with individual doc comments
    * Group by domain: file I/O, citation validation, schema validation, configuration
    * Follow `Err` prefix + PascalCase naming convention
    * _Requirements: 4.5, 10.1_

* [x] 10. Implement review session round management
  * [x] 10.1 Create `internal/session/round.go`
    * Define `RoundManager` struct tracking round count, findings per round, and state transitions
    * Implement round cap enforcement (max 5 rounds)
    * Implement stopping conditions: novelty check failure, all criteria passing (triggers attack round)
    * Implement max 5 findings per round limit
    * _Requirements: 6.1, 6.2, 6.3, 6.6_
  * [x] 10.2 Write property test for round cap enforcement (Property 6)
    * **Property 6: Round cap is never exceeded**
    * **Validates: Requirements 6.1**
    * Generate arbitrary sequences of round transitions; assert total rounds never exceed 5
  * [x] 10.3 Write unit tests for session round management
    * Table-driven tests for: round cap at 5, attack round triggered when all criteria ≥ 7, new issues from attack round return to cycle, novelty check stops loop, findings capped at 5 per round
    * _Requirements: 6.1, 6.2, 6.3, 6.4, 6.6_

* [x] 11. Implement degradation handling
  * [x] 11.1 Create `internal/session/degradation.go`
    * Implement `DegradationTracker` that records component failures during a session
    * Enforce invariant: degraded sessions never produce "approved" or "reviewed" terminal status
    * Generate `DegradationEntry` records with timestamp, component, failure mode, affected criteria
    * _Requirements: 10.1, 10.2, 10.3, 10.4, 10.5_
  * [x] 11.2 Write property test for degraded session status (Property 7)
    * **Property 7: Degraded sessions never produce "approved" status**
    * **Validates: Requirements 10.4**
    * Generate sessions with at least one degradation event; assert terminal status is never "approved" or "reviewed"
  * [x] 11.3 Write unit tests for degradation handling
    * Table-driven tests for: each component failure triggers correct status, partial sessions produce valid output, correct degradation summary in output
    * _Requirements: 10.1, 10.2, 10.3, 10.4, 10.5_

* [x] 12. Implement citation downgrade logic
  * [x] 12.1 Create `internal/session/citation_downgrade.go`
    * Implement `DowngradeUncitedFindings` function that marks findings as "unconfirmed" when citation validation fails
    * Handle both missing citations (no excerpt/location) and failed verbatim match
    * _Requirements: 4.2, 4.5_
  * [x] 12.2 Write property test for uncited findings downgrade (Property 8)
    * **Property 8: Uncited findings are always downgraded**
    * **Validates: Requirements 4.2, 4.5**
    * Generate findings that fail citation validation; assert status is always set to "unconfirmed" regardless of original status
  * [x] 12.3 Write unit tests for citation downgrade
    * Table-driven tests for: missing excerpt → unconfirmed, missing location → unconfirmed, failed verbatim match → unconfirmed, valid citation → status preserved
    * _Requirements: 4.2, 4.5_

* [x] 13. Checkpoint
  * Ensure all tests pass, ask the user if questions arise.

* [x] 14. Implement review output generation
  * [x] 14.1 Create `internal/output/markdown.go`
    * Implement `RenderSession` function that produces the structured Markdown output matching the design's output format
    * Include all sections: summary, findings, attack round, human escalations, human overrides, dead checks, degradation summary
    * _Requirements: 3.4, 7.5, 10.5_
  * [x] 14.2 Create `internal/output/filename.go`
    * Implement `GenerateFilename` function producing `{spec-name}-{ISO-8601-date}-{sequence-number}.md` names
    * Handle sequence number disambiguation for multiple reviews on same date
    * Ensure output directory `.kiro/reviews/` is created if it does not exist
    * _Requirements: 3.4_
  * [x] 14.3 Write unit tests for review output generation
    * Test Markdown rendering includes all sections, correct heading structure, proper formatting
    * Test filename generation with date formatting and sequence numbering
    * _Requirements: 3.4_

* [x] 15. Create Kiro integration configuration files
  * [x] 15.1 Create agent hook configuration
    * Write `.kiro/hooks/adversarial-review-trigger.json` with PostTaskExec event, design phase filter, and prompt action
    * _Requirements: 2.4, 11.3_
  * [x] 15.2 Create steering file templates
    * Write `.kiro/steering/adversarial-review-rubric.md` with rubric schema (named criteria, scoring guidance, pass/fail examples)
    * Write `.kiro/steering/adversarial-review-anti-patterns.md` with initial structure for accumulated failure patterns
    * Write `.kiro/steering/adversarial-review-architecture-constraints.md` with blast-radius domain list
    * _Requirements: 8.1, 8.2, 8.5, 11.4_
  * [x] 15.3 Create MCP server configuration
    * Write MCP server registration config pointing to compiled binary with `WORKSPACE_ROOT` env var
    * _Requirements: 11.2_

* [x] 16. Wire components together
  * [x] 16.1 Create orchestrator entry point in `cmd/review.go`
    * Define `reviewCmd` Cobra command that orchestrates the full review pipeline
    * Load steering file rubric criteria using `github.com/knadh/koanf`
    * Classify artifact against trigger heuristics
    * Coordinate round progression using `internal/session`
    * Invoke Citation Gate validation on each finding
    * Manage human escalation state
    * Write final review output using `internal/output`
    * _Requirements: 1.1, 1.5, 2.1, 2.2, 6.1, 6.2, 6.3, 7.2, 7.3, 9.2, 9.4, 11.5_
  * [x] 16.2 Create subagent prompt construction in `internal/prompt/`
    * Create `internal/prompt/doc.go`
    * Create `internal/prompt/reviewer.go` — builds the Reviewer subagent's `environmental_context` from artifact paths, rubric content, and round metadata (no Author session content)
    * Create `internal/prompt/confirmer.go` — builds the Confirmer subagent's context with claim details and artifact location only
    * Create `internal/prompt/attack.go` — builds the Attack Round variation prompt
    * _Requirements: 1.1, 1.2, 1.4, 1.5, 5.1, 12.1, 12.3_
  * [x] 16.3 Write integration tests for the review pipeline
    * Test end-to-end review session with sample design artifact and rubric
    * Test MCP server tool calls via stdio
    * Test steering file loading with valid and malformed entries
    * Test output file generation with correct naming and structure
    * _Requirements: 3.4, 4.3, 8.1, 8.6, 11.2_

* [x] 17. Final checkpoint
  * Ensure all tests pass, ask the user if questions arise.

## Notes

* Tasks marked with `*` are optional and can be skipped for faster MVP
* Each task references specific requirements for traceability
* Checkpoints ensure incremental validation
* Property tests validate universal correctness properties from the design document
* Unit tests validate specific examples and edge cases
* The MCP server binary is built as a standalone Go CLI using the same patterns as the main project
* All Go files require Apache 2.0 license headers
* Sentinel errors live in `cmd/errors.go` grouped by domain
* Use `pgregory.net/rapid` for property-based tests and `github.com/go-openapi/testify` for assertions

## Task dependency graph

```json
{
  "waves": [
    { "id": 0, "tasks": ["1.1"] },
    { "id": 1, "tasks": ["1.2", "1.3", "2.1", "2.2"] },
    { "id": 2, "tasks": ["2.3", "3.1", "3.2", "5.1", "9.3"] },
    { "id": 3, "tasks": ["3.3", "5.2", "6.1", "7.1"] },
    { "id": 4, "tasks": ["3.4", "3.5", "3.6", "3.7", "6.2", "6.3", "7.2"] },
    { "id": 5, "tasks": ["9.1", "9.2", "10.1", "11.1", "12.1"] },
    { "id": 6, "tasks": ["10.2", "10.3", "11.2", "11.3", "12.2", "12.3"] },
    { "id": 7, "tasks": ["14.1", "14.2", "15.1", "15.2", "15.3"] },
    { "id": 8, "tasks": ["14.3", "16.1", "16.2"] },
    { "id": 9, "tasks": ["16.3"] }
  ]
}
```
