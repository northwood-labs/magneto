# Deep Architecture Audit

## Entry points

Magneto exposes two runtime entry points:

1. **CLI binary** — `main.go` delegates to `cmd.Execute()`, which invokes Cobra via `fang.Execute`. The binary ships three subcommands: `serve`, `review`, and `version`.

2. **MCP tool interface** — when running as `magneto serve`, external agents invoke the three registered tools (`validate_citation`, `validate_findings_batch`, `validate_finding_schema`) over stdin/stdout.

There is no HTTP listener, no daemon mode, and no background process. All communication uses the MCP stdio transport.

## CLI startup and initialization flow

```text
main.go
  │
  ├─ cmd.Execute()
  │    └─ fang.Execute(context.Background(), rootCmd)
  │         ├─ PersistentFlags: --verbose / -v (CountVarP)
  │         └─ dispatches to matched subcommand
  │
  └─ os.Exit(1) on any error
```

### Initialization sequence

1. `cmd/root.go` — `init()` registers the `--verbose` persistent flag on `rootCmd`.
2. `cmd/serve.go` — `init()` adds `serveCmd` to `rootCmd`.
3. `cmd/review.go` — `init()` adds `reviewCmd` to `rootCmd` with `--spec-name` and `--domain` flags.
4. `cmd/version.go` — `init()` adds `versionCmd` (delegated to `cli-helpers`).

The `fang` library wraps Cobra execution to provide terminal color detection and glamour-based long help rendering.

## Command-specific flows

### `magneto serve`

```text
serveCmd.RunE
  ├─ Read WORKSPACE_ROOT from environment
  │    └─ Fatal error if unset (ErrWorkspaceRootNotSet)
  ├─ newMCPServer()
  │    ├─ server.NewMCPServer("magneto", "1.0.0", WithToolCapabilities)
  │    ├─ AddTool(validateCitationTool, handleValidateCitation)
  │    ├─ AddTool(validateFindingsBatchTool, handleValidateFindingsBatch)
  │    └─ AddTool(validateFindingSchemaTool, handleValidateFindingSchema)
  └─ runStdioServer(ctx)
        └─ server.NewStdioServer(s).Listen(ctx, os.Stdin, os.Stdout)
```

The server blocks on stdio until the context is cancelled or the transport closes.

### `magneto review`

```text
reviewCmd.RunE(ctx, args[0] = artifactPath)
  │
  ├─ trigger.Classify(artifact domain, flags)
  │    ├─ DecisionSkip → print reason, return nil
  │    └─ DecisionTrigger → continue
  │
  ├─ session.NewRoundManager()
  ├─ session.NewDegradationTracker()
  │
  ├─ Build ReviewSessionOutput
  │    ├─ Metadata (spec name, artifact path, timestamp)
  │    ├─ TerminalStatus via dt.AllowedTerminalStatus()
  │    ├─ RoundsExecuted from RoundManager
  │    ├─ DegradedComponents from DegradationTracker
  │    └─ Findings from RoundManager.AllFindings()
  │
  ├─ output.RenderSession(sessionOutput) → Markdown string
  │
  ├─ output.GenerateFilename(spec, workspace, timestamp)
  │    └─ .kiro/reviews/{spec-name}-{YYYY-MM-DD}-{seq}.md
  │
  └─ os.WriteFile(filename, rendered)
```

### `magneto version`

Delegated entirely to `cli-helpers.VersionScreen()`. Prints build metadata injected at compile time via ldflags.

### MCP tool: `validate_citation`

```text
handleValidateCitation(ctx, request)
  ├─ Extract: quoted_excerpt, file_path, section_reference
  ├─ citation.Validate(ctx, input)
  │    ├─ resolveContainedPath(workspaceRoot, filePath)
  │    │    ├─ filepath.Abs(workspaceRoot)
  │    │    ├─ filepath.Join(absRoot, filePath)
  │    │    ├─ filepath.EvalSymlinks(joined) → resolved
  │    │    ├─ filepath.EvalSymlinks(absRoot) → resolvedRoot
  │    │    └─ Verify: strings.HasPrefix(resolved, resolvedRoot + "/")
  │    │         └─ Reject with ErrPathTraversal if outside workspace
  │    ├─ readFileChunked(absPath)
  │    │    ├─ os.Open(path)
  │    │    ├─ Read in 1 MiB chunks into bytes.Buffer
  │    │    └─ Reject with ErrFileTooLarge if > 64 MiB
  │    ├─ matchExcerptInContent(content, input)
  │    │    ├─ ExtractSection(content, sectionReference)
  │    │    │    ├─ Try extractByLineRange (regex: "lines? N-M")
  │    │    │    └─ Try extractByHeading (goldmark AST walk)
  │    │    ├─ NormalizeWhitespace(excerpt)
  │    │    ├─ NormalizeWhitespace(section.Content)
  │    │    ├─ strings.Index(normalizedSection, normalizedExcerpt)
  │    │    └─ computeLineLocation → MatchLocation{LineStart, LineEnd}
  │    └─ Return ValidateResult{Valid, MatchLocation, FailureReason}
  └─ Marshal to JSON, return mcp.NewToolResultText
```

### MCP tool: `validate_findings_batch`

```text
handleValidateFindingsBatch(ctx, request)
  ├─ Extract "findings" array from arguments
  ├─ JSON marshal/unmarshal into []citation.ValidateInput
  ├─ citation.ValidateBatch(ctx, batchInput)
  │    └─ Loop: Validate each finding, collect BatchResult per index
  └─ Marshal []BatchResult to JSON
```

### MCP tool: `validate_finding_schema`

```text
handleValidateFindingSchema(ctx, request)
  ├─ Extract "finding" object from arguments
  ├─ JSON unmarshal into models.ReviewFinding
  ├─ schema.ValidateFindingSchema(&finding)
  │    ├─ Check criterion_name non-empty
  │    ├─ Check score in [1, 10]
  │    ├─ Check quoted_excerpt non-empty
  │    ├─ Check artifact_location.file_path non-empty
  │    ├─ Check artifact_location.section_reference non-empty
  │    └─ Check status is valid FindingStatus enum value
  └─ Return {"valid": bool, "error": string|omitted}
```

## File and module responsibilities

### `cmd/` package

| File             | Role                                                                            |
|------------------|---------------------------------------------------------------------------------|
| `root.go`        | Defines `rootCmd`, `Execute()`, global `--verbose` flag                         |
| `serve.go`       | `serveCmd` — MCP server lifecycle, `newMCPServer()`, `runStdioServer()`         |
| `review.go`      | `reviewCmd` — full review pipeline orchestration                                |
| `tools.go`       | MCP tool definitions (schemas) and handler functions                            |
| `version.go`     | Delegates to `cli-helpers` version screen                                       |
| `errors.go`      | All sentinel errors grouped by domain (file I/O, citation, schema, config, MCP) |
| `review_test.go` | Integration tests for review pipeline and MCP tool handlers                     |

### `internal/citation/`

| File           | Role                                                                                                                                                |
|----------------|-----------------------------------------------------------------------------------------------------------------------------------------------------|
| `validate.go`  | Core validation: `Validate()`, `ValidateBatch()`, `resolveContainedPath()`, `readFileChunked()`, `matchExcerptInContent()`, `computeLineLocation()` |
| `section.go`   | `ExtractSection()` — resolves heading names or line ranges to content slices using goldmark                                                         |
| `normalize.go` | `NormalizeWhitespace()` — collapses whitespace runs to single spaces                                                                                |
| `errors.go`    | Sentinel errors: `ErrFileRead`, `ErrPathTraversal`, `ErrFileTooLarge`, `ErrSectionNotFound`, `ErrInvalidLineRange`                                  |

### `internal/models/`

| File         | Role                                                                                                                                         |
|--------------|----------------------------------------------------------------------------------------------------------------------------------------------|
| `finding.go` | `ReviewFinding` struct, `FindingStatus` enum, `ArtifactLocation` struct                                                                      |
| `session.go` | `ReviewSessionOutput`, `SessionMetadata`, `TerminalStatus` enum, `AttackRoundResult`, `HumanEscalation`, `HumanOverride`, `DegradationEntry` |

### `internal/novelty/`

| File       | Role                                                                                                                              |
|------------|-----------------------------------------------------------------------------------------------------------------------------------|
| `check.go` | `Check()` — compares current vs. prior round findings. Novel if new criterion, new evidence, or different score with new evidence |

### `internal/output/`

| File          | Role                                                                                                                                                                  |
|---------------|-----------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `markdown.go` | `RenderSession()` — produces structured Markdown with all review sections (header, summary, findings, attack round, escalations, overrides, dead checks, degradation) |
| `filename.go` | `GenerateFilename()` — creates unique output path with date and sequence number                                                                                       |
| `errors.go`   | `ErrOutputDirCreate` sentinel                                                                                                                                         |

### `internal/prompt/`

| File           | Role                                                                                 |
|----------------|--------------------------------------------------------------------------------------|
| `reviewer.go`  | `BuildReviewerContext()` — artifact path, rubric, round metadata, prior findings     |
| `confirmer.go` | `BuildConfirmerContext()` — single claim, artifact location, quoted evidence         |
| `attack.go`    | `BuildAttackContext()` — challenges prior passing conclusions from a different angle |

### `internal/schema/`

| File          | Role                                                                                                                                |
|---------------|-------------------------------------------------------------------------------------------------------------------------------------|
| `validate.go` | `ValidateFindingSchema()` — field-level validation with structured `SchemaValidationError` collecting multiple `FieldError` entries |

### `internal/session/`

| File                    | Role                                                                                                                                                                                             |
|-------------------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `round.go`              | `RoundManager` — state machine tracking round progression (active → attack_round → approved/stopped/cap_reached), novelty checks, max 5 rounds, max 5 findings per round, passing threshold of 7 |
| `degradation.go`        | `DegradationTracker` — records component failures, enforces the invariant that degraded sessions never produce `approved`                                                                        |
| `citation_downgrade.go` | `DowngradeUncitedFindings()` — marks findings as `unconfirmed` when citation validation fails or citation fields are missing                                                                     |

### `internal/trigger/`

| File          | Role                                                                                                                                                                 |
|---------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `classify.go` | `Classify()` — blast-radius domain matching, foundational trust detection, skip conditions (single file + revertible + human-reviewed), ambiguous default to trigger |

## Decision points and side effects

### Trigger classification decision tree

```text
Classify(artifact)
  ├─ Domain in blast-radius list? → TRIGGER (reason: blast-radius)
  ├─ IsFoundational? → TRIGGER (reason: foundational)
  ├─ IsSingleFile AND IsRevertible AND IsHumanReviewedBefore? → SKIP
  └─ Otherwise → TRIGGER (reason: ambiguous, flag set)
```

Default blast-radius domains: `auth`, `secrets`, `payments`, `data-integrity`, `irreversible-actions`.

### Round progression state machine

```text
NewRoundManager() → StateActive, round=1

SubmitFindings(findings)
  └─ Truncate to max 5, append to rounds[]

AdvanceRound()
  ├─ currentRound >= 5? → StateCapReached
  ├─ Novelty check fails? → StateStopped
  ├─ All criteria passing (score >= 7)? → StateAttackRound
  └─ Otherwise → increment round, StateActive

SubmitAttackRound(findings)
  ├─ No findings? → StateApproved
  ├─ Would exceed cap? → StateCapReached
  └─ Otherwise → feed back, increment round, StateActive
```

### Terminal status resolution

The degradation tracker enforces a single invariant: if `IsDegraded()` is true and the proposed status is `TerminalApproved`, it downgrades to `TerminalPartialReview`. All other statuses pass through unchanged.

### Citation downgrade side effect

`DowngradeUncitedFindings` mutates finding status to `unconfirmed` when:

* `QuotedExcerpt` is empty
* `ArtifactLocation.FilePath` is empty
* `ArtifactLocation.SectionReference` is empty
* Validation result indicates citation is invalid

This is a copy-on-write operation — it returns a new slice rather than mutating the input.

### File system side effects

* `output.GenerateFilename` creates `.kiro/reviews/` directory via `os.MkdirAll` if it does not exist.
* `runReview` writes the rendered Markdown to the generated filename.
* `citation.Validate` reads artifact files from disk after verifying the resolved path stays within `WORKSPACE_ROOT` (symlinks are evaluated, path traversal is rejected). Files are read in 1 MiB chunks up to a 64 MiB hard cap.

## Risks, gaps, and follow-up inspections

### Architectural gaps

* **Single-pass review command.** The `review` command constructs a `RoundManager` and `DegradationTracker` but never calls `SubmitFindings` or `AdvanceRound`. The multi-round loop is expected to be driven externally by an agent calling MCP tools. The in-binary path produces a review with zero findings and zero rounds executed.

* **Rubric loading absent.** `ErrRubricNotFound` and `ErrRubricMalformed` are defined but no code path loads or validates a rubric file. The `ReviewerInput.RubricContent` field is populated by the external agent, not by this binary.

* **No confirmer integration.** `BuildConfirmerContext` constructs the prompt but nothing in the binary invokes a confirmer subagent. Confirmation status relies on the external orchestrator setting `FindingStatus` values before passing findings to the MCP tools.

### Correctness risks

* **Line-location heuristic.** `computeLineLocation` estimates match positions by counting words per line. Dense single-line content or lines with many short words may produce off-by-one or off-by-many line numbers.

* **Goldmark heading matching.** `extractByHeading` uses case-insensitive full-heading comparison. Partial matches, headings with inline code, or headings with emphasis markers will not resolve.

* **Batch validation error accumulation.** `ValidateBatch` continues on file-read errors and reports them per-finding rather than failing the batch. A missing artifact file produces N individual failures rather than one clear signal.

### Security controls

* **Path containment.** `resolveContainedPath` resolves symlinks via `filepath.EvalSymlinks` on both the workspace root and the joined path, then verifies the resolved path has the resolved root as a prefix (with trailing separator to prevent `/workspace-extra` matching `/workspace`). Paths escaping the boundary return `ErrPathTraversal`.

* **File size cap.** `readFileChunked` reads in 1 MiB chunks and rejects files exceeding 64 MiB with `ErrFileTooLarge`. This prevents OOM from unbounded artifact sizes while keeping peak memory allocation predictable.

### Testing gaps

* **No unit tests for `internal/session/round.go`.** The state machine logic (round progression, novelty interaction, attack round feedback) has no dedicated test file. Coverage comes indirectly from `cmd/review_test.go` integration tests.

* **No test for `computeLineLocation`.** The word-counting heuristic is untested in isolation.

## Design rationale

### Why deterministic validation instead of LLM-based checking

The citation gate must be a fixed function, not a probabilistic one. If the gate itself uses an LLM, the agent could game it by adjusting phrasing. Exact substring matching (after whitespace normalization) provides a ground-truth anchor that no prompt manipulation can circumvent.

### Why context isolation in prompt builders

The adversarial review model assumes the Reviewer subagent must not see Author session content. If the Reviewer has access to the Author's reasoning, it may defer to the Author's framing rather than independently evaluating the artifact. Each prompt builder constructs only the minimum context needed for its role.

### Why novelty-based stopping instead of fixed round counts

A fixed round limit (e.g., "always run 5 rounds") wastes resources when the reviewer loops on the same findings. Novelty checking detects convergence: if a round produces nothing new, further rounds are unlikely to surface new issues. The hard cap of 5 rounds provides a safety bound.

### Why MCP over HTTP or gRPC

MCP stdio transport matches the deployment model: an AI agent spawns the server as a subprocess and communicates over pipes. No port allocation, no TLS configuration, no service discovery. The agent manages the process lifecycle directly.

### Why structured Markdown output over JSON

The review output is consumed by both humans (reading the Markdown in `.kiro/reviews/`) and machines (parsing the structured sections). Markdown with predictable headings and tables serves both audiences without requiring a separate rendering step.

### Why the degradation invariant

If a component (citation gate, confirmer, schema validator) fails during a session, the system cannot guarantee completeness. Allowing an `approved` status from an incomplete review would undermine trust. The invariant forces any degraded session to `partial_review`, signaling to downstream consumers that human follow-up is required.

### Why path containment with symlink resolution

A naive `filepath.Join` + prefix check is insufficient because symlinks inside the workspace could point outside it. Resolving symlinks on both the root and the target before comparing ensures the containment check operates on real filesystem paths. The trailing-separator trick prevents false positives when workspace names share a common prefix (e.g., `/workspace` vs `/workspace-staging`).

### Why chunked reading with a hard cap

The MCP server accepts user-supplied file paths. Without a size limit, an attacker (or a misconfigured agent) could point at a multi-gigabyte file and crash the process. Reading in 1 MiB chunks with a 64 MiB hard cap bounds memory usage while still accommodating large design documents. The chunk size matches common OS page-cache alignment for efficient I/O.
