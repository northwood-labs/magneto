# AGENTS.md

This file provides guidance for AI agents working in this repository.

## Project overview

Magneto is a deterministic, non-LLM MCP server that validates citation evidence in adversarial review findings. It verifies quoted excerpts exist verbatim within cited artifact locations. The binary exposes three MCP tools over stdin/stdout and a `review` command that orchestrates the adversarial review pipeline.

Module path: `go.nwlabs.dev/magneto`

## Technology stack

| Tool             | Purpose                                                        |
|------------------|----------------------------------------------------------------|
| Go 1.26          | Primary language                                               |
| Cobra + fang     | CLI framework with terminal color support                      |
| mcp-go           | Model Context Protocol server implementation (stdio transport) |
| goldmark         | Markdown parsing for section extraction                        |
| golangci-lint v2 | Linting (60+ linters, zero-diagnostics policy)                 |
| Task (Taskfile)  | Build automation (`task` or `make` shim)                       |
| hk               | Git hooks and linter orchestration                             |
| rumdl            | Markdown linting                                               |
| Kiro             | IDE with steering, hooks, and specs                            |

## Repository layout

```text
main.go                  → Entrypoint, delegates to cmd.Execute()
cmd/                     → CLI commands, MCP tool handlers, sentinel errors
internal/
  citation/              → Path-safe file reading, section extraction, citation matching
  models/                → Shared data types (ReviewFinding, SessionOutput, etc.)
  novelty/               → Inter-round novelty detection
  output/                → Markdown rendering and filename generation
  prompt/                → Context-isolated prompt builders for subagents
  schema/                → ReviewFinding schema validation
  session/               → Round state machine, degradation tracking, citation downgrade
  trigger/               → Blast-radius trigger classification
docs/                    → Architecture documentation
tools/                   → Brewfiles, Taskfile includes
.kiro/
  steering/              → Agent steering files (always-on + manual)
  hooks/                 → Agent hooks (PostTaskExec trigger)
  specs/                 → Kiro spec definitions
```

## Commands to know

```bash
# Run all tests
go test ./...

# Lint (zero-issue policy)
golangci-lint run --fix ./...

# Build check
go vet ./...

# Task runner (preferred over raw commands)
task lint          # hk fix on changed files
task lint:deep     # hk fix on all files
task install:tools # install all dev dependencies
task clean         # run all cleaning tasks
```

## Code conventions

### Strict rules

* **Zero diagnostics.** All Go files must produce zero issues from `golangci-lint run --fix ./...`. Do not present work as finished while diagnostics remain.

* **Declaration order:** `const`, `var`, `type`, `func` (enforced by `decorder`). One `var` block per file.

* **Error handling:** Sentinel errors in `cmd/errors.go`. Wrap with `fmt.Errorf("%w: detail", ErrSentinel)`. Never create dynamic errors with `fmt.Errorf("...")` alone.

* **Function parameters:** Max 4 params (excluding `context.Context`). Extract into named `*Input` struct passed by pointer.

* **Line length:** 120 characters for code. 80 characters for comment prose (URLs exempt).

* **Error check separation:** Do not combine function call and nil-check in the same `if` statement.

* **No `os.Exit`** except in `main` or `init`.

* **Suppression:** Use `// lint:allow_*` comments. Never `// nolint:`. See `.kiro/steering/go-code-conventions.md` for the full suppression table.

### Patterns

* `strings.Builder` + `fmt.Fprint*` for stdout/stderr output (performance).

* `filepath.Join` for all path construction (Windows compatibility).

* `slices.SortFunc` with `strings.Compare` (not `sort.Slice`).

* Named interfaces at package level (no anonymous interfaces).

* Pass structs by pointer for uniformity and to avoid `hugeParam` warnings.

## Security invariants

* **Path containment:** `resolveContainedPath` in `internal/citation/validate.go` resolves symlinks and verifies paths stay within `WORKSPACE_ROOT`. Never bypass this.

* **File size cap:** `readFileChunked` reads in 1 MiB chunks with a 64 MiB hard limit. Never use `os.ReadFile` for user-supplied paths.

* **Degradation invariant:** A degraded session can never reach `approved` status — only `partial_review`.

## Testing

* Use `github.com/go-openapi/testify` for assertions (`assert`, `require`).

* Use `pgregory.net/rapid` for property-based tests.

* Test files use `_test` package suffix (black-box testing).

* `t.TempDir()` for filesystem isolation.

* `t.Setenv()` for environment variables in tests.

* Run `go test ./...` after any change to verify no regressions.

## Steering files

Steering files in `.kiro/steering/` provide additional context to agents:

| File                                             | Inclusion        | Purpose                                       |
|--------------------------------------------------|------------------|-----------------------------------------------|
| `core-premises.md`                               | always           | Four core principles for all agent work       |
| `go-code-conventions.md`                         | fileMatch `*.go` | Full Go coding standards and lint rules       |
| `markdown-style.md`                              | fileMatch `*.md` | Markdown formatting rules (enforced by rumdl) |
| `adversarial-review-rubric.md`                   | manual           | Scoring guidance for review criteria          |
| `adversarial-review-anti-patterns.md`            | manual           | Known failure patterns to watch for           |
| `adversarial-review-architecture-constraints.md` | manual           | Blast-radius domains and foundational trust   |
| `comprehensive-and-quickstart.md`                | manual           | Instructions for generating architecture docs |
| `generate-agents-md.md`                          | manual           | Instructions for generating this file         |
| `go-cli-application.md`                          | manual           | Patterns for Go CLI/TUI applications          |
| `root-level-readme.md`                           | manual           | Instructions for README generation            |
| `add-code-comments.md`                           | manual           | Instructions for adding code comments         |

## Hooks

| Hook                         | Trigger      | Behavior                                                                                                                 |
|------------------------------|--------------|--------------------------------------------------------------------------------------------------------------------------|
| `adversarial-review-trigger` | PostTaskExec | Prompts the agent to check if a design artifact changed and, if it touches a blast-radius domain, run adversarial review |

## Verification workflow (after any code change)

1. Run `golangci-lint run --fix ./...` — must report 0 issues.
2. Run `go vet ./...` — must be clean.
3. Run `go test ./...` for the affected package — must pass.
4. If Markdown files changed, run `rumdl check <files>` — must report no issues.

## Key architectural decisions

1. **Deterministic validation.** Citation matching uses exact string comparison after whitespace normalization. No LLM inference.
2. **Context isolation.** Subagent prompts intentionally exclude author session content.
3. **Novelty-based stopping.** Review rounds terminate when findings are non-novel (prevents infinite loops).
4. **MCP stdio transport.** No HTTP, no ports — agents spawn the binary as a subprocess.
5. **Workspace containment.** All file reads are path-checked and symlink-resolved against `WORKSPACE_ROOT`.

## Common pitfalls

* Forgetting to wrap errors from internal packages (`wrapcheck` will catch this).
* Placing `const` after `type` in a file (`decorder` order: const, var, type, func).
* Using `os.ReadFile` for user-supplied paths (use `readFileChunked` for size safety).
* Creating `var` blocks inside Cobra `RunE` closures (convert to short declarations).
* Breaking comment lines early — use full 80-char width.
* Using `*text*` for italic or `__text__` for bold — use `_text_` and `**text**`.
