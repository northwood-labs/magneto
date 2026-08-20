# Quick Flow Summary

## Entrypoint

`main.go` calls `cmd.Execute()` and exits non-zero on error. All logic lives in the `cmd` and `internal` packages.

## Primary flow

```text
main.go
  └─ cmd.Execute()
        └─ fang.Execute(rootCmd)  [cobra + terminal color support]
            ├─ serve         → starts MCP stdio server with 4 tools
            ├─ review        → deprecated compatibility wrapper
            ├─ install kiro  → writes embedded Kiro integration files
            └─ version       → prints build info
```

### `magneto serve` (MCP server mode)

1. Reads `WORKSPACE_ROOT` from env (fatal if unset).
2. Creates an MCP server with four registered tools:
   * `validate_citation` — checks a single quoted excerpt against a file section.
   * `validate_findings_batch` — batch-validates citations for multiple findings.
   * `validate_finding_schema` — validates a finding struct against the required schema.
   * `finalize_review_session` — validates terminal session data and persists the review record.
3. Listens on stdin/stdout via MCP stdio transport.

### `magneto install kiro` (installer mode)

1. Validates mutually exclusive `--workspace` / `--user` target flags.
2. Validates the `--mcp-server-name` format (default: `magneto`).
3. Resolves the target directory (cwd or `$HOME`).
4. Writes embedded steering files, hooks, and the operational protocol from `internal/kirofiles`.
5. Merges the MCP server definition into `.kiro/settings/mcp.json`, removing stale Magneto entries.
6. Prints the list of installed paths.

### `magneto review` (deprecated)

Non-interactive compatibility wrapper. Classifies the artifact via `trigger.Classify`, writes a terminal review record with no subagent invocation, and emits a deprecation notice directing users to the Kiro-native `finalize_review_session` MCP workflow.

## Module roles

| Package              | Responsibility                                                                            |
|----------------------|-------------------------------------------------------------------------------------------|
| `cmd`                | CLI commands, MCP tool handlers, sentinel errors, provenance registry                     |
| `internal/citation`  | Path-safe file reading, section extraction (goldmark), normalized substring matching      |
| `internal/kirofiles` | Embedded Kiro assets (embed.FS), MCP config merge, server name validation                 |
| `internal/models`    | Shared data types: `ReviewFinding`, `ReviewSessionOutput`, terminal statuses, domains     |
| `internal/novelty`   | Compares current round findings to prior rounds to detect non-novel loops                 |
| `internal/output`    | Renders structured Markdown output, generates unique filenames, persists terminal records |
| `internal/prompt`    | Constructs context-isolated prompts for Reviewer, Confirmer, and Attack subagents         |
| `internal/schema`    | Decodes, normalizes, and validates `ReviewFinding` structs with legacy score migration    |
| `internal/session`   | Round state machine, degradation tracking, citation downgrade, finalization logic         |
| `internal/trigger`   | Trigger classification heuristics (blast-radius domains, foundational, skip conditions)   |

## Design decisions

1. **Deterministic, non-LLM.** Magneto validates citations using exact string matching after whitespace normalization. No inference, no probabilistic output. MCP tools are called by an external agent that drives the review.

2. **Context isolation.** Prompt builders (`internal/prompt`) construct subagent contexts that intentionally exclude Author session content. Each subagent sees only its own scope — criterion, artifact, and opaque prior-failure fingerprints.

3. **Provenance correlation.** The MCP server tracks schema and citation validation provenance per finding. `finalize_review_session` verifies that terminal findings were assembled from deterministic gate results issued by the same process.

4. **Degradation invariant.** If any required component fails during a session, the degradation tracker prevents the session from ever reaching `approved` status. It can only reach `partial_review`.

5. **Novelty-based stopping.** Rounds terminate when the current round produces no novel findings compared to prior rounds (same criterion + same evidence). This prevents infinite review loops.

6. **MCP stdio transport.** The server communicates exclusively over stdin/stdout using the Model Context Protocol. No HTTP, no ports — agents spawn the binary as a subprocess.

7. **Normalized matching.** Citations are validated after collapsing all whitespace to single spaces. This tolerates formatting differences between the quoted excerpt and the source text.

8. **Workspace containment.** All file reads are confined to the workspace root via `resolveContainedPath`, which resolves symlinks and verifies the final path stays within bounds.

9. **Chunked file reading.** Artifact files are read in 1 MiB chunks with a 64 MiB hard cap, preventing OOM from oversized files.

10. **Idempotent finalization.** `PersistSession` uses an in-memory registry keyed by task execution ID and session ID to ensure exactly one terminal record is written per review session, even if the MCP tool is retried.

## Risks and unknowns

* **Agent orchestration not in this binary.** The multi-round review loop depends on an external Kiro agent calling MCP tools. Magneto provides the validation primitives and finalization, not the orchestration.

* **Line-location approximation.** `computeLineLocation` uses word-counting heuristics to map normalized text offsets back to source line numbers. Edge cases with dense single-line content may report inaccurate ranges.

* **No rubric loading in review command.** The deprecated `review` command does not load or parse a rubric file. The rubric errors are defined but unused in the CLI orchestration path.

* **Goldmark heading extraction.** Section extraction relies on goldmark AST walking. Non-standard Markdown (e.g., ATX headings without space after `#`) may not resolve correctly.

* **Single-process provenance.** Provenance correlation is stored in memory. If the MCP server process restarts mid-session, previously issued provenance IDs are lost and finalization will reject the session.
