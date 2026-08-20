# AGENTS.md

This file provides guidance for AI agents working in this repository.

## Project overview

Magneto is a deterministic, non-LLM MCP server that validates citation evidence in adversarial review findings. It verifies quoted excerpts exist verbatim within cited artifact locations. The binary exposes four MCP tools over stdin/stdout, a deprecated `review` command, and an `install kiro` command that distributes embedded integration files.

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
cmd/
  root.go                → Root command, Execute(), --verbose flag
  serve.go               → MCP stdio server lifecycle
  tools.go               → MCP tool definitions, handlers, provenance registry
  review.go              → Deprecated compatibility wrapper
  install.go             → Parent "install" command
  install_kiro.go        → "install kiro" subcommand, flag handling, orchestration
  errors.go              → All sentinel errors grouped by domain
  version.go             → Version display (cli-helpers)
internal/
  citation/              → Path-safe file reading, section extraction, citation matching
  kirofiles/             → Embedded Kiro assets (embed.FS), MCP config merge, validation
  models/                → Shared data types (ReviewFinding, SessionOutput, etc.)
  novelty/               → Inter-round novelty detection
  output/                → Markdown rendering, filename generation, terminal record persistence
  prompt/                → Context-isolated prompt builders for subagents
  schema/                → ReviewFinding schema validation and legacy score migration
  session/               → Round state machine, degradation tracking, citation downgrade, finalization
  trigger/               → Blast-radius trigger classification
docs/                    → Architecture documentation
tools/                   → Brewfiles, Taskfile includes
.kiro/
  steering/              → Agent steering files (always-on + conditional + manual)
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

* **Provenance correlation:** Terminal review records require findings validated by deterministic gate results issued by the same MCP process. Fabricated or replayed provenance IDs are rejected.

* **Outcome assertion rejection:** MCP tool requests that assert validation outcomes (e.g., `citation_valid`, `confirmed`) are rejected with `ErrToolOutcomeAssertion`.

## Testing

* Use `github.com/go-openapi/testify` for assertions (`assert`, `require`).

* Use `pgregory.net/rapid` for property-based tests.

* Test files use `_test` package suffix (black-box testing).

* `t.TempDir()` for filesystem isolation.

* `t.Setenv()` for environment variables in tests.

* Run `go test ./...` after any change to verify no regressions.

## Steering files

Steering files in `.kiro/steering/` provide additional context to agents:

| File                                             | Inclusion        | Purpose                                            |
|--------------------------------------------------|------------------|----------------------------------------------------|
| `core-premises.md`                               | always           | Four core principles for all agent work            |
| `go-code-conventions.md`                         | fileMatch `*.go` | Full Go coding standards and lint rules            |
| `go-cli-application.md`                          | auto             | Patterns for Go CLI/TUI applications               |
| `markdown-style.md`                              | fileMatch `*.md` | Markdown formatting rules (enforced by rumdl)      |
| `adversarial-review-rubric.md`                   | manual           | Scoring guidance for review criteria               |
| `adversarial-review-anti-patterns.md`            | manual           | Known failure patterns to watch for                |
| `adversarial-review-architecture-constraints.md` | manual           | Blast-radius domains and foundational trust        |
| `adversarial-review-operational-protocol.md`     | manual           | Full Kiro coordinator protocol for review sessions |
| `comprehensive-and-quickstart.md`                | manual           | Instructions for generating architecture docs      |
| `generate-agents-md.md`                          | manual           | Instructions for generating this file              |
| `root-level-readme.md`                           | manual           | Instructions for README generation                 |
| `add-code-comments.md`                           | manual           | Instructions for adding code comments              |

## Hooks

| Hook                         | Trigger      | Behavior                                                                                                                 |
|------------------------------|--------------|--------------------------------------------------------------------------------------------------------------------------|
| `adversarial-review-trigger` | PostTaskExec | Prompts the agent to check if a design artifact changed and, if it touches a blast-radius domain, run adversarial review |

## Specs

| Spec                                      | Purpose                                                                    |
|-------------------------------------------|----------------------------------------------------------------------------|
| `adversarial-review-agent`                | Phase 1 adversarial reviewer — context isolation, citation gate, confirmer |
| `adversarial-review-operational-workflow` | Operational round-state machine — Kiro-native end-to-end workflow          |
| `install-kiro-command`                    | CLI install command for distributing Kiro integration files                |

## Verification workflow (after any code change)

1. Run `golangci-lint run --fix ./...` — must report 0 issues.
2. Run `go vet ./...` — must be clean.
3. Run `go test ./...` for the affected package — must pass.
4. If Markdown files changed, run `rumdl check <files>` — must report no issues.

## Key architectural decisions

1. **Deterministic validation.** Citation matching uses exact string comparison after whitespace normalization. No LLM inference.
2. **Context isolation.** Subagent prompts intentionally exclude author session content.
3. **Provenance correlation.** Schema → citation → finalization form a causal chain verified within a single MCP process lifetime.
4. **Novelty-based stopping.** Review rounds terminate when findings are non-novel (prevents infinite loops).
5. **MCP stdio transport.** No HTTP, no ports — agents spawn the binary as a subprocess.
6. **Workspace containment.** All file reads are path-checked and symlink-resolved against `WORKSPACE_ROOT`.
7. **Idempotent finalization.** Terminal records are written exactly once per session regardless of retry attempts.
8. **Embedded file distribution.** `install kiro` uses Go's `embed.FS` for zero-runtime-dependency file installation.

## Common pitfalls

* Forgetting to wrap errors from internal packages (`wrapcheck` will catch this).
* Placing `const` after `type` in a file (`decorder` order: const, var, type, func).
* Using `os.ReadFile` for user-supplied paths (use `readFileChunked` for size safety).
* Creating `var` blocks inside Cobra `RunE` closures (convert to short declarations).
* Breaking comment lines early — use full 80-char width.
* Using `*text*` for italic or `__text__` for bold — use `_text_` and `**text**`.
* Submitting assertion fields in MCP tool requests (the server rejects them).
* Forgetting `session_id` + `finding_index` in canonical validation requests (provenance is required).
