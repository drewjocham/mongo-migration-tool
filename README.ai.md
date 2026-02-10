# AI Developer Guide: mongo-migration-tool

## 1. Project Purpose

This project, `mongo-migration-tool`, is a command-line utility designed to manage and execute database migrations for MongoDB. It provides a structured way to apply versioned changes to a MongoDB database, supporting both upward and downward migrations.

## 2. Architecture and Design

The application follows a standard Go CLI structure.

*   **Entry Point**: The application starts in `cmd/main.go`, which is a minimal wrapper around the core CLI logic.
*   **CLI Logic**: The primary logic for command handling resides in the `internal/cli` package. This package is responsible for parsing commands, arguments, and flags, likely using the `Cobra` library.
*   **Core Functionality**: The core migration logic (connecting to MongoDB, applying migrations, tracking migration state) is located in the `internal/migration` package.
*   **Configuration**: Configuration is handled in `internal/config` and loaded from files or environment variables. The `config_view.go` file indicates a command exists to inspect the current configuration.

### Design Patterns

*   **Command Pattern**: The CLI is structured around commands (e.g., `migrate up`, `migrate down`, `create migration`). Each command encapsulates a specific action. This is a key pattern to follow.
*   **Dependency Injection**: We inject dependencies like the database connection and configuration into the components that need them. This makes testing easier. When adding new functionality, prefer passing dependencies as arguments rather than using global state.
*   **Functional Programming**: We prefer a functional style where it enhances clarity. This means:
    *   **Immutability**: Avoid modifying data structures in place. Return new copies with the changes.
    *   **Pure Functions**: Functions should, whenever possible, have no side effects and return the same output for the same input.
    *   **Readability is the highest priority**. If a functional approach becomes overly complex, a clear, imperative style is acceptable.
    * **mcp** `github.com/modelcontextprotocol/go-sdk/mcp` is used for MCP and should be designed to support the libraries features. 

## 3. Project Structure

*   `/cmd`: The main application entry point.
*   `/internal`: Contains all the core application logic.
    *   `/internal/cli`: Defines the CLI commands and their structure.
    *   `/internal/migration`: Core logic for running migrations.
    *   `/internal/config`: Configuration loading and management.
*   `/migrations`: Default directory where migration files are stored.
*   `/scripts`: Helper scripts for development, building, etc.
*   `/examples`: Example usage of the tool.
*   `/configs`: Example configuration files.

## 4. Testing Strategy

Our testing philosophy is centered around **table-driven tests**. This is a mandatory practice for all new test files.

*   **Why**: They are DRY, clear, and easy to maintain. Adding a new test case is as simple as adding a new struct to the test table.
*   **How**: Use a slice of structs, where each struct represents a complete test case with inputs and expected outputs. The test function iterates over this slice and executes the same test logic for each case.

### Example of a Table-Driven Test

```go
package mypackage

import "testing"

func TestMyFunction(t *testing.T) {
    testCases := []struct {
        name     string
        input    string
        expected int
        hasError bool
    }{
        {
            name:     "test with valid input",
            input:    "abc",
            expected: 3,
            hasError: false,
        },
        {
            name:     "test with empty input",
            input:    "",
            expected: 0,
            hasError: true,
        },
    }

    for _, tc := range testCases {
        t.Run(tc.name, func(t *testing.T) {
            result, err := MyFunction(tc.input)

            if tc.hasError {
                if err == nil {
                    t.Errorf("expected an error but got none")
                }
            } else {
                if err != nil {
                    t.Errorf("did not expect an error but got: %v", err)
                }
                if result != tc.expected {
                    t.Errorf("expected %d, but got %d", tc.expected, result)
                }
            }
        })
    }
}
```

*   **Location**: Tests for a file `foo.go` should be in `foo_test.go` in the same package.
*   **Mocks/Fakes**: For external dependencies like the database, use interfaces and fakes/mocks to isolate the code under test.
