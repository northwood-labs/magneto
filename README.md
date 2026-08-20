# Magneto

Magneto is an **adversarial citation gate**, named after Erik Lehnsherr (aka _Magneto_) from the X-Men universe, and an old friend/adversary of Professor X (aka _Charles Xavier_).

<img width="1920" height="818" alt="old-friend" src="https://github.com/user-attachments/assets/aeaaf796-153b-481d-b325-f45074eff138" />

> [!IMPORTANT]
> **This is an experiment.** We expect that there will be lots of changes and improvements as we use it and dial-in a useful workflow. YMMV. At present, there should be no expectation of support.

Magneto is a non-LLM adversarial citation gate whose job is to prevent hallucinations by fact-checking the agent in real-time, designed for Kiro.

It is a deterministic MCP server that validates whether quoted evidence genuinely exists in the cited artifact. No LLM involved, no hallucination possible. When an AI agent claims "the design says X in section Y," Magneto checks. Either the exact text is there, or it isn't. This gives you a ground-truth anchor for adversarial review pipelines where trust in citations is non-negotiable.

## What it solves

AI code review and design review agents produce findings with citations. But citations can be fabricated — the agent might paraphrase, hallucinate, or reference the wrong section. Magneto closes that gap:

* **Deterministic verification.** Exact substring matching after whitespace normalization. No probability scores, no "close enough."

* **Provenance-correlated validation.** Each finding flows through a causal chain — schema validation, citation validation, session finalization — all tracked by correlation IDs within a single process. Fabricated or replayed results are rejected.

* **Zero trust by default.** Findings with invalid citations are automatically downgraded to "unconfirmed." Findings that assert their own validation status are rejected outright.

* **Workspace-contained.** File reads are symlink-resolved and path-checked. The server cannot be tricked into reading outside your project.

* **Size-bounded.** Files are read in 1 MiB chunks with a 64 MiB hard cap. No OOM from oversized artifacts.

* **Idempotent finalization.** Terminal review records are written exactly once per session, even if the finalization tool is retried.

## Prerequisites

* An MCP-compatible AI agent or IDE (Kiro, Claude Desktop, or any client speaking the Model Context Protocol over stdio)
* The `WORKSPACE_ROOT` environment variable set to your project's root directory
* At present, this solution only implements Kiro-compatible hooks and steering files. Other tooling will come after the project stabilizes.

## Installation

```bash
go install go.nwlabs.dev/magneto@latest
magneto install kiro --workspace
```

The `install kiro` command writes Kiro integration files (steering, hooks, MCP configuration) from the compiled binary into your project's `.kiro/` directory. Use `--user` instead of `--workspace` to install to `$HOME/.kiro/` for user-level configuration.

Verify it works:

```bash
magneto version
```

### Configuration options

```bash
# Install to the current workspace (most common)
magneto install kiro --workspace

# Install to user-level Kiro config
magneto install kiro --user

# Custom MCP server name (default: "magneto")
magneto install kiro --workspace --mcp-server-name my-citation-gate
```

The installer merges the MCP server definition into your existing `mcp.json` without disturbing other server entries. Running it again updates the Magneto entry to the latest version.

## Usage

### MCP tools

Once the IDE launches the MCP server, the agent has access to four tools:

#### `validate_citation`

Checks whether a quoted excerpt exists at a specific location in an artifact file.

```json
{
  "quoted_excerpt": "Components communicate over stdio using MCP protocol",
  "file_path": "docs/design.md",
  "section_reference": "Architecture"
}
```

Returns `valid: true` with the matching line range, or `valid: false` with a failure reason.

The `section_reference` can be either a Markdown heading name (e.g., `"Architecture"`) or a line range (e.g., `"lines 45-60"`).

For provenance-correlated sessions, include `session_id`, `finding_index`, and `schema_provenance_correlation_id` to link the citation result to a prior schema validation.

#### `validate_findings_batch`

Validates citations for multiple findings in a single call. Useful when an agent produces several findings at once and you want to gate them all before proceeding.

```json
{
  "findings": [
    {
      "quoted_excerpt": "short-lived tokens with automatic rotation",
      "file_path": "docs/design.md",
      "section_reference": "Security"
    },
    {
      "quoted_excerpt": "nonexistent claim about the system",
      "file_path": "docs/design.md",
      "section_reference": "Overview"
    }
  ]
}
```

Returns per-finding results with index, validity, and failure reason.

#### `validate_finding_schema`

Validates that a finding object has all required fields with valid values before it enters the review pipeline.

```json
{
  "finding": {
    "criterion_name": "context-isolation",
    "criterion_satisfaction": 8,
    "finding_severity": "high",
    "finding_domains": ["security", "correctness"],
    "quoted_excerpt": "reviewer subagent receives only the artifact",
    "artifact_location": {
      "file_path": "docs/design.md",
      "section_reference": "Architecture"
    },
    "status": "hypothesized",
    "reasoning": "The design correctly isolates reviewer context."
  }
}
```

Required fields: `criterion_name`, `criterion_satisfaction` (1-10), `finding_severity` (critical/high/medium/low), `finding_domains` (one or more of: security, correctness, architecture, reliability, operations, developer-experience), `quoted_excerpt`, `artifact_location.file_path`, `artifact_location.section_reference`, `status` (must be `hypothesized` for proposed findings), `reasoning`.

When `session_id` and `finding_index` are provided, the tool returns a `provenance_correlation_id` that links to subsequent citation validation.

#### `finalize_review_session`

Validates a terminal review session assembled from deterministic validation results and persists it as a Markdown record.

```json
{
  "session": {
    "metadata": {
      "spec_name": "auth-redesign",
      "artifact_path": ".kiro/specs/auth-redesign/design.md",
      "timestamp": "2026-08-20T14:30:00Z",
      "terminal_status": "not_approved",
      "task_execution_id": "task-abc-123",
      "session_id": "review-xyz-456",
      "rounds_executed": 3
    },
    "findings": []
  }
}
```

Returns the terminal status, idempotency key, and the path where the review record was written (e.g., `.kiro/reviews/auth-redesign-2026-08-20-1.md`).

### Blast-radius domains

These domains trigger mandatory adversarial review when detected in a design artifact:

* `auth`
* `secrets`
* `payments`
* `data-integrity`
* `irreversible-actions`

## How tests work

The test suite covers citation validation, schema enforcement, session state management, trigger classification, output rendering, and Kiro file installation:

```bash
go test ./...
```

Key test categories:

* **Citation validation** (`internal/citation/`) — verifies exact matching, section boundary detection, whitespace normalization, path traversal rejection, symlink escape prevention, and file size limits.

* **Schema validation** (`internal/schema/`) — verifies field-level validation, legacy score migration, domain normalization, and assertion field rejection.

* **Session management** (`internal/session/`) — verifies round progression, novelty detection, degradation tracking, citation downgrade logic, Confirmer routing, and terminal finalization.

* **Kiro file installation** (`internal/kirofiles/`) — verifies MCP config merge, server name validation, and embedded file access.

* **Output rendering** (`internal/output/`) — verifies Markdown rendering, filename generation, and idempotent terminal record persistence.

* **Property-based tests** — fuzz-style tests using `pgregory.net/rapid` that exercise validation with randomized inputs.

Run a specific package:

```bash
go test ./internal/citation/...
```

Skip long-running tests (large file generation):

```bash
go test -short ./...
```

## Troubleshooting

### `WORKSPACE_ROOT environment variable not set`

The `serve` command requires `WORKSPACE_ROOT` to resolve relative file paths. Set it to the absolute path of your project root before starting the server.

### Citation returns "section not found"

The `section_reference` must match either:

* A Markdown heading exactly (case-insensitive). Headings with inline code or emphasis markers may not resolve.
* A line range in the format `lines N-M` (1-indexed, inclusive).

If your heading contains formatting like `` ## `Config` Options ``, try using a line range instead.

### Citation returns "quoted excerpt not found within cited section"

Whitespace differences (extra spaces, tabs, newlines) are normalized before matching. However, the _words_ must match exactly. Check for:

* Typos or paraphrasing in the quoted excerpt
* The excerpt spanning a section boundary (it must exist entirely within the cited section)
* Unicode characters that look similar but differ (e.g., en-dash vs hyphen)

### "file path resolves outside workspace root"

The file path must resolve to a location within `WORKSPACE_ROOT` after symlink resolution. This error means the path (or a symlink it traverses) points outside the workspace. Use paths relative to the workspace root without `../` traversal.

### "cited file exceeds maximum allowed size"

Magneto rejects files larger than 64 MiB. If your artifact exceeds this limit, split it into smaller documents and cite the relevant section.

### Schema validation rejects proposed findings

Proposed findings must use `status: "hypothesized"`. Magneto rejects findings that assert their own validation status (`confirmed`, `citation_valid`, `citation_gate_result`). These fields are reserved for deterministic processing by Magneto itself.

### "deterministic validation provenance does not match"

In canonical mode (with `session_id` + `finding_index`), citation validation requires a matching `schema_provenance_correlation_id` from a prior `validate_finding_schema` call. Ensure schema validation completes first and you pass the returned correlation ID.

## License

Apache License, Version 2.0. See the license headers in source files for details.
