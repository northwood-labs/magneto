# Copilot Instructions

This file tells GitHub Copilot how to find and use the project's agent guidance, coding standards, and design documentation. Read the referenced files before generating code or answering questions about this project.

## Primary guidance file

Start with `AGENTS.md` at the repository root. It contains the authoritative summary of:

* Technology stack and module path
* Repository layout
* Build, lint, and test commands
* Code conventions (strict rules and patterns)
* Security invariants
* Testing conventions
* Verification workflow (required after every code change)
* Common pitfalls

Treat `AGENTS.md` as the single source of truth for project-wide rules. When any steering file or spec conflicts with `AGENTS.md`, the more specific document wins for its scope.

## Kiro steering files

Steering files live in `.kiro/steering/` and provide domain-specific coding standards. Each file contains a front-matter block indicating when it applies:

* `inclusion: always` — read on every task.
* `inclusion: fileMatch` with a `fileMatchPattern` — read when working on files matching that glob.
* `inclusion: manual` — read when the topic is explicitly relevant.

Before generating or modifying code, identify which steering files match the file type or domain you are working in and read them. They define the _how_ — formatting rules, lint policies, naming conventions, structural patterns, and suppression mechanisms specific to this project.

## Kiro specs

Specs live in `.kiro/specs/<feature-name>/` and document requirements, design decisions, and implementation tasks for features. Each spec directory contains:

* `requirements.md` — user stories and acceptance criteria
* `design.md` — architecture, component design, and locked decisions
* `tasks.md` — ordered implementation tasks with completion status

Read the relevant spec before making changes to a feature area. Specs define the _why_ and _what_; steering files define the _how_.

## Project documentation

Architecture and design docs live in `docs/`. Read `docs/quickstart.md` (if present) for a fast orientation. Read `docs/comprehensive.md` (if present) for deep understanding of module interactions. Any file prefixed with a feature name contains decision logs, evidence ledgers, or implementation plans for that feature.

## How to apply this guidance

1. For any code generation task, read `AGENTS.md` and the relevant steering files first.
2. For feature work, also read the associated spec's `requirements.md` and `design.md`.
3. Follow the verification workflow described in `AGENTS.md` after every change.
4. When uncertain about a convention, check the steering file that matches your file type — it has the canonical rules.
5. Respect security invariants listed in `AGENTS.md` — they are non-negotiable.
