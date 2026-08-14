# Quick Flow Summary

## Entrypoint

`main.go` calls `cmd.Execute()` and exits non-zero on error. All logic lives in the `cmd` and `internal` packages.

## Primary flow

```text
main.go
  └─ cmd.Execute()
       └─ fang.Execute(rootCmd)  [cobra + terminal color support]
            ├─ serve   → starts MCP stdio server with 3 tools
            ├─ review  → orchestrates adversarial review pipeline
            └─ version → prints build info
```

### `magneto serve` (MCP server mode)

1. Reads `WORKSPACE_ROOT` from env (fatal if unset).
2. Creates an MCP server with three registered tools:
   * `validate_citation` — checks a single quoted excerpt against a file section.
   * `validate_findings_batch` — batch-validates citations for multiple findings.
   * `validate_finding_schema` — validates a finding struct against the required schema.
3. Listens on stdin/stdout via MCP stdio transport.

### `magneto review` (orchestration mode)

1. Classifies the artifact via `trigger.Classify` (blast-radius domain check, foundational trust, skip conditions).
2. If decision is `skip`, prints reason and exits.
3. Initializes `session.RoundManager` and `session.DegradationTracker`.
4. Builds a `ReviewSessionOutput` and resolves terminal status.
5. Renders Markdown via `output.RenderSession`.
6. Writes output to `.kiro/reviews/{spec-name}-{date}-{seq}.md`.

## Module roles

| Package             | Responsibility                                                                    |
|---------------------|-----------------------------------------------------------------------------------|
| `cmd`               | CLI commands, MCP tool handlers, sentinel errors                                  |
| `internal/citation` | File reading, section extraction (goldmark), normalized substring matching        |
| `internal/models`   | Shared data types: `ReviewFinding`, `ReviewSessionOutput`, terminal statuses      |
| `internal/novelty`  | Compares current round findings to prior rounds to detect non-novel loops         |
| `internal/output`   | Renders structured Markdown output, generates unique filenames                    |
| `internal/prompt`   | Constructs context-isolated prompts for Reviewer, Confirmer, and Attack subagents |
| `internal/schema`   | Validates `ReviewFinding` structs (required fields, score range, valid status)    |
| `internal/session`  | Round state machine, degradation tracking, citation downgrade logic               |
| `internal/trigger`  | Trigger classification heuristics (blast-radius domains, skip conditions)         |

## Design decisions

1. **Deterministic, non-LLM.** Magneto validates citations using exact string matching. No inference, no probabilistic output. The MCP tools are called by an external agent system that drives the review.

2. **Context isolation.** Prompt builders (`internal/prompt`) construct subagent contexts that intentionally exclude Author session content. Each subagent sees only its own scope.

3. **Degradation invariant.** If any component fails during a session, the degradation tracker prevents the session from ever reaching `approved` status. It can only reach `partial_review`.

4. **Novelty-based stopping.** Rounds terminate when the current round produces no novel findings compared to prior rounds (same criterion + same evidence). This prevents infinite review loops without requiring arbitrary retry limits.

5. **MCP stdio transport.** The server communicates exclusively over stdin/stdout using the Model Context Protocol, allowing any MCP-compatible agent to drive it.

6. **Normalized matching.** Citations are validated after collapsing all whitespace to single spaces. This tolerates formatting differences between the quoted excerpt and the source text.

7. **Workspace containment.** All file reads are confined to the workspace root via `resolveContainedPath`, which resolves symlinks and verifies the final path stays within bounds. This prevents path traversal attacks.

8. **Chunked file reading.** Artifact files are read in 1 MiB chunks with a 64 MiB hard cap, preventing OOM from oversized files while keeping memory usage predictable.

## Risks and unknowns

* **Agent orchestration not in this binary.** The `review` command sets up the framework but the actual multi-round loop depends on an external agent calling MCP tools. The in-binary review path is a simplified single-pass.

* **Line-location approximation.** `computeLineLocation` uses word-counting heuristics to map normalized text offsets back to source line numbers. Edge cases with dense single-line content may report inaccurate ranges.

* **No rubric loading in review command.** The `review` command's `runReview` function does not currently load or parse a rubric file. The rubric errors are defined but unused in the orchestration path.

* **Goldmark heading extraction.** Section extraction relies on goldmark AST walking. Non-standard Markdown (e.g., ATX headings without space after `#`) may not resolve correctly.
