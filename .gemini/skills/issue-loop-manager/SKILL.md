---
name: issue-loop-manager
description: Manages a continuous development loop that picks the next open GitHub issue, implements it using sub-agents (code-writer, code-reviewer, technical-writer), and submits a detailed Pull Request. Use when the user wants to automate the implementation of a project backlog while adhering to Zero Dependency mandates.
---

# Issue Loop Manager

This skill automates the lifecycle of transforming a GitHub issue into a verified Pull Request.

## Operational Mandates

- **Zero Dependencies:** All implementations MUST use only the Go standard library. No external packages (including `golang.org/x/`) are allowed in the core.
- **GPG Signing:** ALWAYS use GPG signatures for all commits (e.g., `git commit -S -m "..."`).
- **Sub-Agent Orchestration:** 
  - Implementation & Tests -> `code-writer`
  - Architectural Review -> `code-reviewer`
  - Documentation & Articles -> `technical-writer`
- **Testing Requirements:** Every feature MUST have unit tests with `t.Parallel()` and >90% coverage.

## The Loop Workflow

1.  **Sync & Cleanup:**
    - `git checkout main && git pull origin main`
    - Prune merged branches.
2.  **Pick Next Issue:**
    - Use `gh issue list --limit 1 --json number,title,body` to identify the target.
3.  **Feature Branching:**
    - `git checkout -b feat/issue-<number>`
4.  **Execution (Plan -> Act -> Validate):**
    - Research the issue and existing code.
    - Delegate implementation to `code-writer`.
    - Delegate review to `code-reviewer`.
    - Delegate documentation to `technical-writer`.
5.  **Pull Request Submission:**
    - Create PR with `gh pr create`.
    - **PR Body Requirements:**
        - **Overview:** High-level summary.
        - **Implementation Details:** Technical approach (e.g., "Used atomic counters").
        - **Verification:** Test results and coverage report.
        - **Closing:** Include `fixes #<number>`.

## Interaction Logic

- **Automatic Loop:** When the user says "Sync and continue the issue loop" or provides no specific issue, trigger the workflow from Step 1, using `gh issue list` to identify the next target.
- **Targeted Implementation:** When the user provides a specific issue number (e.g., "Implement issue #123"), skip Step 2 and proceed directly to Step 3 using the provided number. Always perform Step 1 (Sync & Cleanup) regardless of the entry point to ensure a clean workspace.
