# Deep Architecture Audit

## Entry points

Magneto exposes two runtime entry points:

1. **CLI binary** — `main.go` delegates to `cmd.Execute()`, which invokes Cobra via `fang.Execute`. The binary ships four subcommands: `serve`, `review` (deprecated), `install kiro`, and `version`.

2. **MCP tool interface** — when running as `magneto serve`, external agents invoke the four registered tools (`validate_citation`, `validate_findings_batch`, `validate_finding_schema`, `finalize_review_session`) over stdin/stdout.

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
4. `cmd/install.go` — `init()` adds `installCmd` to `rootCmd`.
5. `cmd/install_kiro.go` — `init()` adds `installKiroCmd` to `installCmd` with `--workspace`, `--user`, and `--mcp-server-name` flags.
6. `cmd/version.go` — `init()` adds `versionCmd` (delegated to `cli-helpers`).

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
  │    ├─ AddTool(validateFindingSchemaTool, handleValidateFindingSchema)
  │    └─ AddTool(finalizeReviewSessionTool, handleFinalizeReviewSession)
  └─ runStdioServer(ctx)
        └─ server.NewStdioServer(s).Listen(ctx, os.Stdin, os.Stdout)
```

The server blocks on stdio until the context is cancelled or the transport closes.

### `magneto install kiro`

```text
installKiroCmd.RunE
  ├─ resolveInstallKiroInput()
  │    ├─ resolveInstallKiroTargetDir()
  │    │    ├─ Validate --workspace XOR --user (ErrFlagsMutuallyExclusive / ErrFlagRequired)
  │    │    ├─ selectedInstallKiroTargetDir() → cwd or $HOME
  │    │    └─ os.Stat → verify directory exists
  │    └─ kirofiles.ValidateServerName(fMCPServerName)
  │         └─ Reject invalid chars (ErrMCPServerNameInvalid)
  └─ runInstallKiro(input)
       └─ for each kirofiles.Files():
            ├─ createInstallKiroParent(destination) → os.MkdirAll 0o755
            ├─ installKiroContent(manifestPath, destination, serverName)
            │    ├─ kirofiles.Content(path) → read from embed.FS
            │    └─ if mcp.json: kirofiles.MergeMCPConfig(existing, serverName, definition)
            ├─ os.WriteFile(destination, content, 0o666)
            ├─ os.Chmod(destination, 0o666)
            └─ collect relative path for summary output
```

### `magneto review` (deprecated)

```text
reviewCmd.RunE(ctx, args[0] = artifactPath)
  │
  ├─ trigger.Classify(artifact domain, flags)
  │    ├─ DecisionSkip → print reason + deprecation notice, return nil
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
  ├─ output.GenerateFilename(spec, workspace, timestamp)
  │    └─ .kiro/reviews/{spec-name}-{YYYY-MM-DD}-{seq}.md
  ├─ os.WriteFile(filename, rendered)
  └─ Print deprecation notice + output path
```

### `magneto version`

Delegated entirely to `cli-helpers.VersionScreen()`. Prints build metadata injected at compile time via ldflags.

## MCP tool flows

### `validate_citation`

```text
handleValidateCitation(ctx, request)
  ├─ rejectOutcomeAssertions(args)
  ├─ citationInputFromArguments → (quoted_excerpt, file_path, section_reference)
  ├─ validationIdentityFromArguments → (identity, canonical?)
  ├─ if !canonical:
  │    └─ validateCitationLegacy(ctx, input)
  │         └─ citation.Validate → resolveContainedPath → readFileChunked → matchExcerptInContent
  └─ if canonical:
       ├─ requiredStringArgument("schema_provenance_correlation_id")
       ├─ validationProvenance.requireSchemaForCitation(identity, schemaID, input)
       ├─ citation.Validate(ctx, input)
       └─ validationProvenance.recordCitation(identity, schemaID, result)
```

### `validate_findings_batch`

```text
handleValidateFindingsBatch(ctx, request)
  ├─ rejectOutcomeAssertions(args)
  ├─ validationIdentityFromArguments → (identity, canonical?)
  ├─ if canonical:
  │    └─ validateCanonicalBatch(ctx, args, identity)
  │         └─ for each finding:
  │              ├─ Extract schema_provenance_correlation_id
  │              ├─ schema.DecodeAndNormalizeFinding
  │              ├─ validationProvenance.requireSchemaForFinding
  │              ├─ citation.Validate
  │              └─ validationProvenance.recordCitation
  └─ if !canonical:
       └─ validateLegacyBatch(ctx, args)
            └─ citation.ValidateBatch (loop over findings)
```

### `validate_finding_schema`

```text
handleValidateFindingSchema(ctx, request)
  ├─ Extract "finding" object from arguments
  ├─ schema.DecodeAndNormalizeFinding(findingJSON)
  │    ├─ Decode into findingInput (legacy score → criterion_satisfaction)
  │    ├─ Reject assertion fields (citation_gate_result, citation_valid, etc.)
  │    ├─ Validate all required fields
  │    ├─ Validate severity ∈ {critical, high, medium, low}
  │    ├─ Validate domains ⊂ {security, correctness, architecture, reliability, operations, developer-experience}
  │    ├─ Validate status == "hypothesized" (proposed findings only)
  │    └─ Clamp criterion_satisfaction to [1, 10]
  ├─ validationIdentityFromArguments → (identity, canonical?)
  └─ if canonical && valid:
       └─ validationProvenance.recordSchema(identity, normalized)
            → return provenance_correlation_id
```

### `finalize_review_session`

```text
handleFinalizeReviewSession(ctx, request)
  ├─ Extract "session" object from arguments
  ├─ normalizeFinalizationSession(sessionJSON)
  ├─ session.FinalizeReviewSession(input)
  │    ├─ Validate terminal status ∈ {approved, not_approved, partial_review, human_override}
  │    ├─ Require task_execution_id and session_id
  │    ├─ validateHumanDecisions (overrides need rationale, block acceptance needs rationale)
  │    ├─ terminalStatus: human_override > required_degradation > attack_round_succeeded > not_approved
  │    ├─ validateUnavailableValues (only for partial_review)
  │    └─ terminalIdempotencyKey(taskExecutionID, sessionID) → SHA-256
  └─ output.PersistSession(input)
       ├─ Validate terminal status
       ├─ Require idempotency key
       ├─ Check in-memory registry (prevent duplicate records)
       ├─ GenerateFilename → .kiro/reviews/{spec}-{date}-{seq}.md
       ├─ RenderSession → structured Markdown
       └─ writeTerminalRecord (O_EXCL to prevent overwrite)
```

## File and module responsibilities

### `cmd/` package

| File              | Role                                                                           |
|-------------------|--------------------------------------------------------------------------------|
| `root.go`         | Defines `rootCmd`, `Execute()`, global `--verbose` flag                        |
| `serve.go`        | `serveCmd` — MCP server lifecycle, `newMCPServer()`, `runStdioServer()`        |
| `review.go`       | `reviewCmd` — deprecated compatibility wrapper                                 |
| `install.go`      | `installCmd` — parent command for install targets                              |
| `install_kiro.go` | `installKiroCmd` — Kiro file installation, flag handling, orchestration        |
| `tools.go`        | MCP tool definitions, handlers, provenance registry, canonical/legacy dispatch |
| `errors.go`       | All sentinel errors grouped by domain                                          |
| `version.go`      | Version display delegated to cli-helpers                                       |

### `internal/citation/`

| File          | Role                                                                                                                             |
|---------------|----------------------------------------------------------------------------------------------------------------------------------|
| `validate.go` | `Validate`, `ValidateBatch`, `resolveContainedPath`, `readFileChunked`, `matchExcerptInContent`                                  |
| `section.go`  | `ExtractSection`, `extractByLineRange`, `extractByHeading` (goldmark AST), `NormalizeWhitespace`                                 |
| `errors.go`   | Package-level sentinel errors: `ErrPathTraversal`, `ErrFileRead`, `ErrFileTooLarge`, `ErrSectionNotFound`, `ErrInvalidLineRange` |

### `internal/kirofiles/`

| File           | Role                                                            |
|----------------|-----------------------------------------------------------------|
| `kirofiles.go` | `embed.FS` declaration, `Files()`, `Content()`, `MCPTemplate()` |
| `merge.go`     | `MergeMCPConfig` — non-destructive JSON merge for `mcp.json`    |
| `validate.go`  | `ValidateServerName` — format validation for MCP server names   |

### `internal/models/`

| File         | Role                                                                                                                                                                                |
|--------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `finding.go` | `ReviewFinding`, `FindingStatus`, `FindingSeverity`, `FindingDomain`, `ArtifactLocation`, `CitationGateResult`, `ConfirmerAttempt`, `CanonicalFindingDomains`                       |
| `session.go` | `ReviewSessionOutput`, `SessionMetadata`, `TerminalStatus`, `DegradationEntry`, `HumanEscalation`, `HumanOverride`, `HumanBlockAcceptance`, `AttackRoundResult`, `UnavailableValue` |

### `internal/novelty/`

| File       | Role                                                                                                         |
|------------|--------------------------------------------------------------------------------------------------------------|
| `check.go` | `Check` — compares current vs prior findings, classifies novelty by criterion name + evidence + satisfaction |

### `internal/output/`

| File          | Role                                                                             |
|---------------|----------------------------------------------------------------------------------|
| `markdown.go` | `RenderSession` — structured Markdown with findings, escalations, degradation    |
| `filename.go` | `GenerateFilename`, `PersistSession` — unique paths, idempotent terminal records |
| `errors.go`   | Package-level sentinel errors for output operations                              |

### `internal/prompt/`

| File           | Role                                                                  |
|----------------|-----------------------------------------------------------------------|
| `reviewer.go`  | `BuildReviewerContext` — artifact, rubric, round, opaque fingerprints |
| `confirmer.go` | `BuildConfirmerContext` — claim, severity, domains, attempt number    |
| `attack.go`    | `BuildAttackContext` — challenge prior approval, different angle      |

### `internal/schema/`

| File          | Role                                                                                                                 |
|---------------|----------------------------------------------------------------------------------------------------------------------|
| `validate.go` | `DecodeAndNormalizeFinding`, `ValidateFindingSchema` — field validation, legacy score migration, assertion rejection |

### `internal/session/`

| File                    | Role                                                                                                                                 |
|-------------------------|--------------------------------------------------------------------------------------------------------------------------------------|
| `round.go`              | `RoundManager` — state machine (active → attack → approved/stopped/capped), novelty integration, 5 round cap, 5 findings per round   |
| `routing.go`            | `IsHighImpactFinding`, `IsGateValid`, `IsConfirmerTarget`, `SelectConfirmerTargets`, `IsApparentApproval`, `TransitionFindingStatus` |
| `degradation.go`        | `DegradationTracker` — records component failures, enforces terminal status precedence                                               |
| `citation_downgrade.go` | `DowngradeUncitedFindings` — applies deterministic gate results to finding copies                                                    |
| `finalize.go`           | `FinalizeReviewSession` — terminal validation, human decision audit, idempotency key                                                 |

### `internal/trigger/`

| File          | Role                                                                                               |
|---------------|----------------------------------------------------------------------------------------------------|
| `classify.go` | `Classify` — blast-radius domain matching, foundational trust, skip conditions, ambiguous fallback |

## Decision points and side effects

### Provenance validation pipeline

The MCP server maintains an in-memory provenance registry (`validationProvenance`) that correlates schema validation → citation validation → finalization. This enforces a causal chain:

1. `validate_finding_schema` with `session_id` + `finding_index` records schema provenance and returns a `provenance_correlation_id`.
2. `validate_citation` with `schema_provenance_correlation_id` verifies the schema was validated first, then records citation provenance.
3. `finalize_review_session` assembles the terminal record from findings that carry provenance IDs matching results this MCP process issued.

This prevents an agent from fabricating validation results or reusing results from a different session.

### Outcome assertion rejection

`rejectOutcomeAssertions` checks incoming tool arguments for fields that only Magneto should produce (`citation_valid`, `confirmed`, `citation_gate_result`, `provenance_correlation_id`). If an agent submits these, the tool returns `ErrToolOutcomeAssertion`.

### Terminal status precedence

`session.FinalizeReviewSession` applies terminal status in this fixed order:

1. Human override present → `human_override`
2. Required degradation → `partial_review`
3. Attack round succeeded (no novel issues) → `approved`
4. Otherwise → `not_approved`

### Degradation invariant

`DegradationTracker.AllowedTerminalStatus` enforces that a session with required component failure can never reach `approved`. Only `human_override` bypasses this check.

### Idempotent record persistence

`output.PersistSession` maintains an in-memory path registry keyed by `workspaceRoot + "\x00" + idempotencyKey`. The first call writes the record; subsequent calls with the same key return the existing path without writing.

## Risks, gaps, and follow-up inspections

1. **Single-process provenance lifetime.** Provenance correlation is stored in memory. An MCP server restart mid-session loses all issued provenance IDs, causing finalization to reject the session. No persistence mechanism exists.

2. **No rubric loading in CLI review path.** The deprecated `review` command does not load a rubric file. `ErrRubricNotFound` and `ErrRubricMalformed` are defined but unused outside test scaffolding.

3. **Line-location heuristic.** `computeLineLocation` approximates line positions via word counting. Single-line content blocks, code fences, and dense prose produce inaccurate line ranges.

4. **Legacy score migration.** `DecodeAndNormalizeFinding` accepts both `score` and `criterion_satisfaction`. If both are present, `criterion_satisfaction` wins. There is no validation that an external agent does not submit conflicting values.

5. **Embedded file manifest is explicit.** `kirofiles.Files()` returns a hardcoded list. Adding new steering files requires updating the embed directives, the constant list, and the `Content()` switch statement — three synchronized locations.

6. **File size cap UX.** The 64 MiB hard limit rejects the entire file rather than providing partial content. For large generated artifacts, this may prevent citation validation entirely.

7. **Goldmark heading extraction.** Headings with inline code (`` ## `Config` Options ``), emphasis, or link content may not match the plain-text reference.

8. **No concurrent session support.** The provenance registry and terminal record registry are per-process singletons. Running multiple review sessions through the same MCP process would require session-scoped isolation (currently achieved by one session per process invocation).

## Design rationale

1. **Deterministic over probabilistic.** Citation validation uses exact string comparison after normalization. This eliminates false positives from LLM-based similarity scoring and provides a binary ground truth.

2. **Provenance as trust anchor.** By correlating schema → citation → finalization within a single process lifetime, the system ensures findings cannot acquire `confirmed` or `approved` status through fabricated intermediate results.

3. **Advisory-only architecture.** Magneto never edits artifacts, blocks task progression, or writes to the reviewed spec. This preserves human agency while providing machine-verified evidence.

4. **Embedded file distribution.** Using Go's `embed.FS` eliminates runtime dependencies on external file locations. The binary is self-contained and `install kiro` works without network access.

5. **MCP stdio transport.** No ports, no HTTP, no service discovery. Agents spawn the binary as a subprocess and communicate over pipes. This simplifies deployment, avoids security exposure, and integrates cleanly with IDE process management.

6. **Separation of satisfaction from severity.** Criterion satisfaction (1-10 rubric score) records how well a rubric criterion is met. Finding severity (critical/high/medium/low) records downstream impact. Confirmer routing uses only severity and domain, not satisfaction, preventing rubric gaming.

7. **Novelty-based termination.** Rather than running a fixed number of rounds, the system stops when findings repeat. This adapts review depth to artifact complexity while preventing infinite loops.

8. **Fail-open trigger classification.** Ambiguous artifacts default to triggering review (`DecisionTrigger` with `Ambiguous: true`). This conservative design ensures uncertain cases receive review rather than silent skipping.
