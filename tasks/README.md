# Resile Task Specifications

This directory contains granular task specifications for the **resile** project. Each task is designed to be completed by an independent Gemini CLI instance.

## Workflow
1.  **Consult `SPEC.md`**: The root specification remains the source of truth for architecture and rationale.
2.  **Pick a Task**: Follow the numbered order for dependencies (e.g., Task 05 depends on Task 03).
3.  **Implement & Validate**: Every task requires unit tests for verification.

## Current Tasks
- [x] TASK-01: Project Initialization
- [ ] TASK-03: AWS Full Jitter Backoff
- [ ] TASK-04: Error Policy & Fatal Errors
- [ ] TASK-05: Context-Aware Execution Loop
- [ ] TASK-06: Dynamic Delay (Retry-After)
- [ ] TASK-07: State & Telemetry Interface
- [ ] TASK-08: Testing Ergonomics
- [ ] TASK-09: Slog Instrumentation
- [ ] TASK-10: OpenTelemetry Instrumentation
- [ ] TASK-11: Circuit Breaker Integration

*Note: Task 02 (Core API) is intentionally deferred or handled separately.*
