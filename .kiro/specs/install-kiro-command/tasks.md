# Implementation Plan: Install Kiro Command

## Overview

Implement the Go-based `magneto install kiro` command incrementally: establish the fixed embedded asset package, add deterministic MCP configuration merging and validation, wire the Cobra command hierarchy and installation flow, then verify it with focused unit, rapid property, integration, and repository validation tests.

## Tasks

* [x] 1. Build the embedded Kiro asset and configuration foundation
  * [x] 1.1 Add the five canonical Kiro source files and the `internal/kirofiles` embedded manifest API
    * Create only `hooks/adversarial-review-trigger.json`, `settings/mcp.json`, `steering/adversarial-review-anti-patterns.md`, `steering/adversarial-review-architecture-constraints.md`, and `steering/adversarial-review-rubric.md` beneath `internal/kirofiles/source/`.
    * Implement `kirofiles.go` with exactly five explicit `//go:embed` file patterns, a deterministic five-path `Files()` manifest, `Content`, and `MCPTemplate`; do not embed a directory or expose files outside the allowlist.
    * Ensure the embedded `settings/mcp.json` represents only the Magneto server definition required to launch `magneto serve` with `WORKSPACE_ROOT` and `disabled: false`.
    * _Requirements: 3.1, 3.2, 3.3, 6.1, 6.2_
  * [x] 1.2 Implement the pure MCP configuration merge function in `internal/kirofiles/merge.go`
    * Accept existing bytes, the selected server name, and the embedded definition; create a minimal `mcpServers` document when no file exists.
    * Preserve unrelated top-level JSON values and non-Magneto `mcpServers` values unchanged; remove the exact requested server-name key and every legacy key containing case-insensitive `magneto`, then insert the current definition.
    * Return 2-space-indented JSON with exactly one trailing newline and contextual wrapped errors for malformed configuration.
    * _Requirements: 5.1, 5.2, 5.3, 5.4, 5.5, 5.6, 5.7, 5.8, 7.1_
  * [x] 1.3 Implement MCP server-name validation in `internal/kirofiles/validate.go`
    * Accept only non-empty names beginning with a lowercase ASCII letter and containing ASCII letters or digits only.
    * Return the invalid-name sentinel wrapped with a specific detail for empty, malformed, and otherwise invalid names.
    * _Requirements: 9.1, 9.2, 9.3, 10.3, 10.4_
  * [x] 1.4 Write unit tests for embedded manifest access and MCP merge edge cases
    * Assert the exact ordered five-file manifest and content lookup behavior, and cover absent configuration, an absent `mcpServers` object, and malformed JSON.
    * Use `go-openapi/testify` assertions and `require` in the repository's black-box `_test` packages.
    * _Requirements: 3.1, 3.2, 3.3, 5.1, 5.2, 5.8_
  * [x] 1.5 Write property test for MCP merge preservation
    * **Property 1: MCP merge preserves non-Magneto entries.**
    * Generate valid configurations with arbitrary non-Magneto server entries and top-level metadata, run at least 100 rapid checks, and verify their raw JSON values are unchanged.
    * **Validates: Requirements 5.5, 5.6**
  * [x] 1.6 Write property test for legacy Magneto-key cleanup
    * **Property 2: MCP merge removes all Magneto-related keys and adds the new entry.**
    * Generate case variants and embedded-substring legacy keys, then verify only the explicitly selected key remains and contains the current definition after at least 100 rapid checks.
    * **Validates: Requirements 5.3, 5.4**
  * [x] 1.7 Write property test for MCP merge formatting
    * **Property 3: MCP merge output formatting.**
    * Generate valid merge inputs, verify valid JSON, two-space indentation, and exactly one trailing newline in at least 100 rapid checks.
    * **Validates: Requirements 5.7**
  * [x] 1.8 Write property test for MCP merge idempotence
    * **Property 4: MCP merge idempotence.**
    * Generate valid merge inputs and prove a second merge with the same name and definition is byte-identical to the first output in at least 100 rapid checks.
    * **Validates: Requirements 7.1**
  * [x] 1.9 Write validation unit and property tests
    * Add table-driven invalid and valid server-name examples using `go-openapi/testify`.
    * **Property 5: Server name validation accepts only valid names.** Generate arbitrary strings, run at least 100 rapid checks, and verify the specification predicate and wrapped invalid-name sentinel agree.
    * **Validates: Requirements 9.1, 9.2, 10.3, 10.4**

* [x] 2. Add the command hierarchy and filesystem installation flow
  * [x] 2.1 Add the `install` Cobra parent command and register it with `rootCmd`
    * Follow existing Cobra, fang, `clihelpers.LongHelpText`, and `init()` registration conventions; provide no `RunE` so invoking `magneto install` displays usage and available subcommands.
    * _Requirements: 1.1, 1.3, 1.4_
  * [x] 2.2 Add the five documented install sentinel errors to the existing `cmd/errors.go` `var` block
    * Add individually documented PascalCase `Err` variables for required flag selection, mutually exclusive flags, MCP configuration parsing, file or directory writes, and invalid MCP server names.
    * Preserve the project's declaration order and use each sentinel through `fmt.Errorf` with `%w` plus failure context.
    * _Requirements: 9.1, 9.2, 9.3, 9.4_
  * [x] 2.3 Implement `magneto install kiro` command, flag validation, and target resolution in `cmd/install_kiro.go`
    * Register `--workspace` and `--user` with no short flags; require exactly one, resolve workspace from the current directory or user mode from `$HOME`, and reject empty `$HOME`, missing paths, and non-directory targets with wrapped sentinel errors.
    * Register `--mcp-server-name` with uppercase short flag `-S`, default `magneto`, and validation before any filesystem mutation.
    * Use an `InstallKiroInput` struct and fail fast, returning the first wrapped error from `RunE` or its helpers.
    * _Requirements: 1.2, 2.1, 2.2, 2.3, 2.4, 2.5, 2.6, 9.2, 9.3, 9.4, 10.1, 10.3, 10.4_
  * [x] 2.4 Implement sequential Kiro installation, MCP merge persistence, and success output
    * Iterate only `kirofiles.Files()`, compute every target path with `filepath.Join`, recursively create parents at `0o0755`, and overwrite manifest files at `0o0666` without rollback.
    * Read and merge the target `.kiro/settings/mcp.json`, persist the merged output at `0o0666`, and wrap all directory or file failures with `ErrFileWrite` and the affected path.
    * Build stdout only after every write has succeeded using `strings.Builder` and `fmt.Fprint`; emit exactly one target-relative path per written manifest file and no summary on failure.
    * Leave prior-version files outside the current five-file manifest untouched.
    * _Requirements: 4.1, 4.2, 4.3, 4.4, 4.5, 4.6, 5.1, 5.2, 5.3, 5.4, 5.5, 5.6, 5.7, 5.8, 6.1, 6.2, 7.1, 7.2, 7.3, 7.4, 8.1, 8.2, 8.3, 8.4, 8.5, 9.2, 9.3, 9.4_
  * [x] 2.5 Write command and installation unit tests
    * Cover missing and conflicting location flags, empty `HOME`, invalid targets, `-S` default and override behavior, sentinel wrapping, and no stdout summary on a write failure.
    * Use `t.TempDir()` and `t.Setenv()` to isolate filesystem and environment cases.
    * _Requirements: 1.2, 2.3, 2.4, 2.5, 2.6, 7.3, 8.2, 8.3, 8.4, 9.1, 9.2, 9.3, 10.1_
  * [x] 2.6 Write installation integration and manifest integrity property tests
    * Verify a complete installation writes only the five manifest paths with embedded byte content and required permissions, preserves unrelated MCP data, remains byte- and permission-idempotent, overwrites modified current files, and leaves an old unlisted file in place.
    * **Property 6: Fixed manifest scope and file content integrity.** Generate pre-existing content and missing-parent states, run at least 100 rapid checks, and verify no unlisted source-directory file is installed.
    * **Validates: Requirements 3.1, 3.2, 4.1, 4.2, 4.3, 4.4, 5.5, 5.6, 7.1, 7.2, 7.4**

* [x] 3. Checkpoint - Ensure all tests pass
  * Ensure all tests pass, ask the user if questions arise.

* [x] 4. Run final automated repository validation
  * [x] 4.1 Run formatting and quality gates for the completed Go implementation
    * Run the repository's applicable formatter, `golangci-lint run --fix ./...`, `go vet ./...`, and `go test ./...`; resolve any diagnostics before completing the implementation.
    * _Requirements: 1.1, 1.2, 2.1, 2.2, 2.3, 2.4, 2.5, 2.6, 3.1, 3.2, 3.3, 4.1, 4.2, 4.3, 4.4, 4.5, 4.6, 5.1, 5.2, 5.3, 5.4, 5.5, 5.6, 5.7, 5.8, 6.1, 6.2, 7.1, 7.2, 7.3, 7.4, 8.1, 8.2, 8.3, 8.4, 8.5, 9.1, 9.2, 9.3, 9.4, 10.1, 10.2, 10.3, 10.4_

* [x] 5. Final checkpoint - Ensure all tests pass
  * Ensure all tests pass, ask the user if questions arise.

## Notes

* Tasks are ordered so that the asset contract and pure logic exist before CLI orchestration and filesystem integration.
* Tasks marked with `*` are optional automated-test tasks; each property maps directly to one numbered correctness property in the design.
* All file-writing tasks retain the fixed five-file manifest, explicit embed patterns, sentinel error wrapping, and non-destructive MCP merge requirements.
* Implementation completes only after the final lint, vet, and test quality gates are clean.

## Task dependency graph

```json
{
  "waves": [
    { "id": 0, "tasks": ["1.1", "1.2", "1.3", "2.1", "2.2"] },
    { "id": 1, "tasks": ["1.4", "1.9", "2.3"] },
    { "id": 2, "tasks": ["1.5", "2.4"] },
    { "id": 3, "tasks": ["1.6", "2.5"] },
    { "id": 4, "tasks": ["1.7", "2.6"] },
    { "id": 5, "tasks": ["1.8"] },
    { "id": 6, "tasks": ["4.1"] }
  ]
}
```
