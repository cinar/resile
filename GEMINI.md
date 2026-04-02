# Gemini CLI Guidance for Resile

This repository is governed by the architectural specification in `SPEC.md`.

## Core Mandates
1.  **Zero Dependencies**: The core package must have no external dependencies outside the Go standard library.
2.  **Type Safety**: Use Go 1.18+ Generics for all execution wrappers.
3.  **Concurrency**: Adhere to the strict context-aware timer management patterns defined in `SPEC.md`.
4.  **Testing**: Every feature MUST be accompanied by unit tests. Use `t.Parallel()` where appropriate.
5.  **Examples**: Add an example for every new feature.
6.  **Documentation**: Update the main README.md file.
7.  **Coverage**: Keep the code coverage above 90% at all times.
8.  **Formatting**: Run Go source code formatting at the end.
9.  **Articles**: With every new feature, write an article about it in `docs/articles/`.

## Workflow Mandates
1.  **Issue Research**: When given an issue number, use `gh issue view <number>` to gather requirements and context.
2.  **Branch Management**: Always create a new feature branch (e.g., `feat/issue-<number>`) before making any changes.
3.  **Planning Phase**: You MUST use `enter_plan_mode` to research, design, and present a comprehensive strategy for user approval before starting any implementation.

## Strategic Delegation Mandate
To maintain session efficiency and adhere to specialized workflows (like branch management and cross-reference integrity), you MUST delegate all implementation, review, and documentation tasks to the respective sub-agents:

- **Implementation**: Always delegate the implementation of features, bug fixes, and unit tests to the `code-writer` sub-agent.
- **Review**: Always delegate code quality and architectural reviews to the `code-reviewer` sub-agent.
- **Documentation**: Always delegate the creation of technical articles, README updates, and documentation cross-referencing to the `technical-writer` sub-agent.
- **Orchestration**: Act as the high-level orchestrator. Your responsibility is to define the strategy, coordinate the sub-agents, and perform the final verification of their work against the original request.
- **Direct Action**: Only perform implementation or documentation tasks directly in the main session if a sub-agent is technically unable to complete a specific, narrow task or if the user explicitly overrides this mandate with a direct instruction for a specific turn.
