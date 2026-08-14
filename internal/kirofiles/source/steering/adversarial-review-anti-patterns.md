---
inclusion: manual
---

# Adversarial Review Anti-Patterns

## Purpose

This file accumulates failure patterns observed across review sessions. The adversarial reviewer references these when evaluating new artifacts to avoid known blind spots.

## Known anti-patterns

### Placeholder pattern

Artifact contains TODO markers or placeholder text that is treated as complete implementation.

### Wishful specification

Artifact describes desired behavior without specifying error cases, edge conditions, or failure modes.

### Single-path thinking

Artifact describes only the happy path without considering concurrent access, partial failures, or degraded operation.

### Implicit dependency

Artifact assumes availability of external systems without documenting fallback behavior.
