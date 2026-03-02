# Contributing to Resile

Thank you for your interest in contributing to **Resile**! We welcome all kinds of contributions, from bug fixes to new features and documentation improvements.

## Code of Conduct

Please be respectful and professional in all your interactions within this project.

## How to Contribute

1.  **Fork the repository**: Create your own copy of the project on GitHub.
2.  **Clone the fork**: Download the repository to your local machine.
3.  **Create a branch**: Use a descriptive name for your feature or fix (e.g., `fix/context-leak` or `feat/prometheus-labels`).
4.  **Implement your changes**:
    - Follow standard Go idioms and naming conventions.
    - Ensure your code is formatted with `go fmt`.
    - Use the `WithTestingBypass` context for faster unit tests.
5.  **Write tests**: Every new feature or bug fix MUST include corresponding unit tests.
6.  **Run tests**: Verify that all tests pass:
    ```bash
    go test -v ./...
    ```
7.  **Submit a Pull Request**: Provide a clear description of your changes and why they are needed.

## Architectural Mandates

- **Zero Dependencies**: The core package must not depend on any external libraries outside the Go standard library.
- **Type Safety**: Use generics for all execution wrappers.
- **Concurrency**: Adhere to the strict context-aware patterns for timer management as defined in the [Architectural Specification](docs/SPEC.md).

## Questions?

If you have questions or need help, please open an issue on GitHub.
