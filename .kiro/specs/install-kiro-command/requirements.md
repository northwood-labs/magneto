# Requirements Document

## Introduction

The `magneto install kiro` command installs Kiro IDE integration files into a target directory so that end-users can use Magneto's adversarial review capabilities in their own projects. The command embeds the necessary steering files, hooks, and MCP server configuration at compile time using Go's `embed` package, and writes them to either the workspace `.kiro/` directory or the user-level `$HOME/.kiro/` directory.

## Glossary

- **Installer**: The `magneto install kiro` subcommand responsible for writing Kiro integration files to the filesystem.
- **Target_Directory**: The root directory where `.kiro/` files are written; either the current working directory (workspace mode) or `$HOME` (user mode).
- **Embedded_Files**: Static file contents compiled into the Magneto binary via Go's `embed` package.
- **MCP_Config**: The `.kiro/settings/mcp.json` file that defines MCP server connections for the Kiro IDE.
- **Steering_Files**: Markdown files in `.kiro/steering/` that provide behavioral guidance to the Kiro agent.
- **Hook_Files**: JSON files in `.kiro/hooks/` that define automated triggers for Kiro agent actions.
- **Magneto_Key**: Any JSON key within the `mcpServers` object of `mcp.json` whose name contains the substring "magneto".
- **MCP_Server_Name**: The configurable key name used when writing the MCP server definition into `mcp.json`. Defaults to `magneto`. Must conform to VS Code naming conventions (camelCase, alphanumeric only).

## Requirements

### Requirement 1: Command Structure

**User Story:** As a developer, I want a `magneto install kiro` subcommand, so that I can install Kiro integration files from the CLI.

#### Acceptance Criteria

1. THE Installer SHALL be registered as a subcommand of a new `install` parent command under the `magneto` root command.
2. WHEN the user invokes `magneto install kiro`, THE Installer SHALL execute the Kiro file installation process and return a nil error on success.
3. THE Installer SHALL follow existing CLI patterns using Cobra, fang, `clihelpers.LongHelpText`, and `init()` registration.
4. WHEN the user invokes `magneto install` without a subcommand, THE CLI SHALL display the `install` command's usage text listing available subcommands.

### Requirement 2: Target Directory Selection

**User Story:** As a developer, I want to choose between workspace-level and user-level installation, so that I can control where integration files are placed.

#### Acceptance Criteria

1. THE Installer SHALL accept a `--workspace` flag that sets the Target_Directory to the current working directory.
2. THE Installer SHALL accept a `--user` flag that sets the Target_Directory to the value of the `$HOME` environment variable.
3. IF the `--user` flag is provided and the `$HOME` environment variable is unset or empty, THEN THE Installer SHALL return a non-zero exit code and an error message indicating that `$HOME` is not defined.
4. IF neither `--workspace` nor `--user` is provided, THEN THE Installer SHALL return a non-zero exit code and an error message indicating that one flag is required.
5. IF both `--workspace` and `--user` are provided, THEN THE Installer SHALL return a non-zero exit code and an error message indicating that the flags are mutually exclusive.
6. IF the resolved Target_Directory does not exist or is not a directory, THEN THE Installer SHALL return a non-zero exit code and an error message indicating the path is not a valid directory.

### Requirement 3: Embedded File Storage

**User Story:** As a maintainer, I want integration files stored in the binary at compile time, so that the command has no runtime dependency on external file locations.

#### Acceptance Criteria

1. THE Installer SHALL embed the following files using Go's `embed` package from a source directory within the repository: `hooks/adversarial-review-trigger.json`, `settings/mcp.json`, `steering/adversarial-review-anti-patterns.md`, `steering/adversarial-review-architecture-constraints.md`, `steering/adversarial-review-rubric.md`.
2. THE list of embedded files SHALL be exhaustive — the binary SHALL fail to compile if any listed source file is missing from the embed directory or if the Go embed directive fails to include a file.
3. THE embedded content SHALL reflect whatever file content was present at the time the binary was built.

### Requirement 4: Steering and Hook File Installation

**User Story:** As a developer, I want steering and hook files replaced on each install, so that running the command always brings files up to date.

#### Acceptance Criteria

1. WHEN the Installer writes Steering_Files, THE Installer SHALL overwrite any existing file at the destination path within the `steering` subdirectory of `.kiro/` under Target_Directory.
2. WHEN the Installer writes Hook_Files, THE Installer SHALL overwrite any existing file at the destination path within the `hooks` subdirectory of `.kiro/` under Target_Directory.
3. WHEN any intermediate directory in a destination file path does not exist, THE Installer SHALL recursively create all missing directories with permissions `0o0755`.
4. THE Installer SHALL write files with permissions `0o0666`.
5. THE Installer SHALL construct all file paths using `filepath.Join`.
6. IF a file write operation fails after one or more files have already been written, THEN THE Installer SHALL return a wrapped error identifying the failed file without rolling back previously written files.

### Requirement 5: MCP Configuration Merge

**User Story:** As a developer, I want the MCP server configuration merged into my existing `mcp.json`, so that other MCP server definitions are preserved while the Magneto entry is updated.

#### Acceptance Criteria

1. WHEN `mcp.json` does not exist at the target path, THE Installer SHALL create a new file containing only the Magneto MCP server definition within an `mcpServers` object.
2. WHEN `mcp.json` exists at the target path, THE Installer SHALL read and parse the existing JSON content.
3. WHEN merging, THE Installer SHALL remove all existing keys that match the provided MCP_Server_Name exactly AND any key containing the case-insensitive substring "magneto" (for backward compatibility cleanup) from the `mcpServers` object.
4. WHEN merging, THE Installer SHALL add the current Magneto MCP server definition under the key specified by `--mcp-server-name` in the `mcpServers` object.
5. WHEN merging, THE Installer SHALL preserve all non-Magneto entries in the `mcpServers` object unchanged.
6. WHEN merging, THE Installer SHALL preserve any top-level keys outside of `mcpServers` (e.g., comments or metadata fields) in the output file.
7. THE Installer SHALL write the merged `mcp.json` with 2-space JSON indentation and a trailing newline, using permissions `0o0666`.
8. IF the existing `mcp.json` contains invalid JSON, THEN THE Installer SHALL return an error indicating the file is malformed.

### Requirement 6: MCP Server Definition

**User Story:** As a developer, I want the installed MCP definition to launch Magneto correctly, so that the Kiro IDE can communicate with the MCP server.

#### Acceptance Criteria

1. THE Installer SHALL define the MCP server entry as a JSON object containing: a `command` field with string value `magneto`, an `args` field with array value `["serve"]`, an `env` field with object value `{"WORKSPACE_ROOT": "${workspaceFolder}"}`, and a `disabled` field with boolean value `false`.
2. THE Installer SHALL write the MCP server definition under the key specified by `--mcp-server-name` in the `mcpServers` object.

### Requirement 7: Idempotent Execution

**User Story:** As a developer, I want to run the install command repeatedly without side effects, so that upgrading is safe and predictable.

#### Acceptance Criteria

1. WHEN the Installer is executed 2 or more times with the same binary version against the same target directory, THE Installer SHALL produce byte-identical file content and identical file permissions at each installed path as a single execution.
2. WHEN a newer binary version contains updated Embedded_Files, THE Installer SHALL overwrite previously installed files with the new content regardless of whether the existing file content was modified after initial installation.
3. IF the Installer cannot overwrite a target file due to insufficient filesystem permissions, THEN THE Installer SHALL report an error message indicating which path failed and the permission condition.
4. WHEN Embedded_Files in the current binary version no longer include a file that was installed by a previous version, THE Installer SHALL leave the previously installed file in place without deletion.

### Requirement 8: User Feedback

**User Story:** As a developer, I want clear output about what the command did, so that I can verify the installation succeeded.

#### Acceptance Criteria

1. WHEN installation completes successfully, THE Installer SHALL print to stdout one line per file written, each containing the file's path relative to the Target_Directory.
2. THE Installer SHALL use `strings.Builder` with `fmt.Fprint` for all stdout output.
3. IF any file write operation fails, THEN THE Installer SHALL return a wrapped error that includes the destination path of the file that failed.
4. IF any file write operation fails, THEN THE Installer SHALL NOT write any summary output to stdout.
5. IF a file write operation fails after other files have already been written, THEN THE Installer SHALL leave the previously written files in place.

### Requirement 9: Sentinel Errors

**User Story:** As a maintainer, I want well-defined sentinel errors for the install command, so that error handling follows project conventions.

#### Acceptance Criteria

1. THE Installer SHALL define sentinel errors in `cmd/errors.go` for: missing flag selection, mutually exclusive flags, MCP config parse failure, file write failure, and invalid MCP server name format, where each sentinel follows the `Err` prefix with PascalCase naming convention and is placed in the existing single `var()` block with an individual doc comment starting with the variable name.
2. THE Installer SHALL wrap all errors returned from the install command's `RunE` function and any helper functions it calls using `fmt.Errorf` with the `%w` verb, where: missing flag selection wraps the missing-flag sentinel, mutually exclusive flags wraps the mutual-exclusion sentinel, invalid JSON in existing `mcp.json` wraps the MCP config parse sentinel, any failed file or directory write operation wraps the file write sentinel, and an invalid `--mcp-server-name` value wraps the invalid MCP server name sentinel.
3. WHEN a wrapped error is returned, THE Installer SHALL include a detail string after the sentinel that identifies the specific context of the failure (e.g., which file failed to write or which flags conflicted).
4. WHEN multiple error conditions could theoretically occur, THE Installer SHALL return the first error encountered (properly wrapped with its sentinel) without attempting to collect or report additional errors.

### Requirement 10: Configurable MCP server name

**User Story:** As a developer, I want to customize the MCP server name used in `mcp.json`, so that I can avoid naming conflicts or follow my team's naming conventions.

#### Acceptance Criteria

1. THE Installer SHALL accept a `--mcp-server-name` flag (short form `-S`) with a default value of `magneto`.
2. THE Installer SHALL use the value of `--mcp-server-name` as the key under which the MCP server definition is written in the `mcpServers` object.
3. THE Installer SHALL validate that the `--mcp-server-name` value conforms to VS Code MCP server naming conventions: camelCase format, no whitespace, no special characters (only ASCII letters and digits are allowed).
4. IF the `--mcp-server-name` value is an empty string, THEN THE Installer SHALL return a validation error indicating the name must not be empty.
