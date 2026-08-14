# Magneto

Magneto is an **adversarial citation gate** agent, named after Erik Lehnsherr (aka _Magneto_) from the X-Men universe, and an old friend/adversary of Professor X (aka _Charles Xavier_).

[IMAGE]

> [!IMPORTANT]
> **This is an experiment.** We expect that there will be lots of changes and improvements as we use it and dial-in a useful workflow. YMMV. At present, there should be no expectation of support.

A citation gate for AI agents. Magneto is a system containing a deterministic MCP server which validates whether quoted evidence actually exists in the cited artifact — no LLM involved, no hallucination possible.

When an AI agent claims "the design says X in section Y," Magneto checks. Either the exact text is there, or it isn't. This gives you a ground-truth anchor for adversarial review pipelines where trust in citations is non-negotiable.

## What it solves

AI code review and design review agents produce findings with citations. But citations can be fabricated — the agent might paraphrase, hallucinate, or reference the wrong section. Magneto closes that gap:

* **Deterministic verification.** Exact substring matching after whitespace normalization. No probability scores, no "close enough."

* **Zero trust by default.** Findings with invalid citations are automatically downgraded to "unconfirmed."

* **Workspace-contained.** File reads are symlink-resolved and path-checked. The server cannot be tricked into reading outside your project.

* **Size-bounded.** Files are read in 1 MiB chunks with a 64 MiB hard cap. No OOM from oversized artifacts.

## Prerequisites

* Go 1.26 or later
* An MCP-compatible AI agent or IDE (Kiro, Claude Desktop, or any client speaking the Model Context Protocol over stdio)
* The `WORKSPACE_ROOT` environment variable set to your project's root directory

## Installation

```bash
go install go.nwlabs.dev/magneto@latest
magneto install kiro --workspace
```

Verify it works:

```bash
magneto version
```

## Usage

### As an MCP server

Magneto runs as a subprocess of your AI agent, communicating over stdin/stdout. Configure it in your MCP client settings:

```json
{
  "mcpServers": {
    "magneto": {
      "command": "magneto",
      "args": ["serve"],
      "env": {
        "WORKSPACE_ROOT": "${workspaceFolder}"
      }
    }
  }
}
```

Once running, the agent has access to three tools:

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

#### `validate_findings_batch`

Validates citations for multiple findings in a single call. Useful when an agent produces several findings at once and you want to gate them all before proceeding.

```json
{
  "findings": [
    {
      "QuotedExcerpt": "short-lived tokens with automatic rotation",
      "FilePath": "docs/design.md",
      "SectionReference": "Security"
    },
    {
      "QuotedExcerpt": "nonexistent claim about the system",
      "FilePath": "docs/design.md",
      "SectionReference": "Overview"
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
    "score": 8,
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

Required fields: `criterion_name`, `score` (1-10), `quoted_excerpt`, `artifact_location.file_path`, `artifact_location.section_reference`, `status`.

### As a CLI review command

Run a standalone adversarial review pass against a design artifact:

```bash
export WORKSPACE_ROOT=/path/to/your/project

magneto review docs/design.md \
  --spec-name "auth-redesign" \
  --domain "auth"
```

This classifies the artifact against blast-radius domains, runs the review framework, and writes a structured Markdown report to `.kiro/reviews/auth-redesign-2026-08-14-1.md`.

Blast-radius domains that trigger mandatory review: `auth`, `secrets`, `payments`, `data-integrity`, `irreversible-actions`.

## How tests work

The test suite covers citation validation, schema enforcement, session state management, trigger classification, and output rendering:

```bash
go test ./...
```

Key test categories:

* **Citation validation** (`internal/citation/`) — verifies exact matching, section boundary detection, whitespace normalization, path traversal rejection, symlink escape prevention, and file size limits.

* **Session management** (`internal/session/`) — verifies round progression, novelty detection, degradation tracking, and citation downgrade logic.

* **Integration tests** (`cmd/`) — verifies the full pipeline from MCP tool invocation through validation to output file generation.

* **Property-based tests** (`internal/citation/validate_property_test.go`) — fuzz-style tests that exercise validation with randomized inputs.

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

## License

Apache License, Version 2.0. See the license headers in source files for details.
