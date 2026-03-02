# Gemini CLI Guidance for Resile

This repository is governed by the architectural specification in `SPEC.md`.

## Task Workflow
Implementation should follow the granular specifications in the `tasks/` directory.

## Core Mandates
1.  **Zero Dependencies**: The core package must have no external dependencies outside the Go standard library.
2.  **Type Safety**: Use Go 1.18+ Generics for all execution wrappers.
3.  **Concurrency**: Adhere to the strict context-aware timer management patterns defined in `SPEC.md`.
4.  **Testing**: Every feature MUST be accompanied by unit tests. Use `t.Parallel()` where appropriate.
