# Testing Guide

This directory contains the test suite for mongo-essential.

## Test Structure

```
test/
├── integration/          # Integration tests
│   └── mcp_integration_test.go
├── helpers/             # Test utilities and helpers
│   └── test_helpers.go
└── README.md           # This file
```

## Test Types

### Unit Tests

Unit tests are located alongside the code they test:

- `mcp/server_test.go` - MCP server unit tests
- `migration/engine_test.go` - Migration engine tests
- `config/config_test.go` - Configuration tests
- `cmd/root_test.go` - CLI command tests

**Run unit tests:**
```bash
make test
```

### Integration Tests

Integration tests require a MongoDB instance and test the complete system:

- `test/integration/mcp_integration_test.go` - End-to-end MCP server tests

**Run integration tests:**
```bash
# Start MongoDB first
make db-up

# Run integration tests
make test-integration

# Or run in Docker (includes MongoDB)
make test-integration-docker
```

## Running Tests

### Quick Test Commands

```bash
# Run all unit tests
make test

# Run specific component tests
make test-mcp         # MCP server tests
make test-migration   # Migration engine tests
make test-config      # Config tests
make test-cmd         # CLI tests

# Run with coverage
make test-coverage

# Run with race detector
make test-race

# Run benchmarks
make test-bench

# Run all tests (unit + integration)
make test-all
```

### Advanced Testing

```bash
# Watch mode (requires entr)
make test-watch

# Docker-based integration tests
make test-integration-docker

# Full coverage including integration
make test-coverage-full

# Clean test cache
make test-clean
```

## Test Helpers

The `test/helpers` package provides common testing utilities:

### TestHelper

A helper struct that manages test database lifecycle:

```go
func TestMyFeature(t *testing.T) {
    helper := helpers.NewTestHelper(t)
    ctx := context.Background()

    if err := helper.Setup(ctx); err != nil {
        t.Fatal(err)
    }
    defer helper.Cleanup(ctx)

    engine := helper.CreateEngine()
    // Your test code here
}
```

### Test Migrations

Use `TestMigration` for creating test migrations:

```go
migration := helpers.NewTestMigration("20240101_001", "Test migration")
migration.UpFunc = func(ctx context.Context, db *mongo.Database) error {
    // Custom up logic
    return nil
}
```

### Assertions

```go
helper.AssertNoError(err)
helper.AssertError(err)
helper.AssertEqual(expected, actual)
helper.AssertContains(haystack, needle)
```

### Environment Setup

```go
cleanup := helpers.SetTestEnv(t, map[string]string{
    "MONGO_URL": "mongodb://localhost:27017",
    "MONGO_DATABASE": "test_db",
})
defer cleanup()
```

## Writing Tests

### Unit Test Example

```go
func TestMigrationEngine(t *testing.T) {
    helper := helpers.NewTestHelper(t)
    ctx := context.Background()

    if err := helper.Setup(ctx); err != nil {
        t.Fatal(err)
    }
    defer helper.Cleanup(ctx)

    engine := helper.CreateEngine()

    // Register test migration
    migration := helpers.NewTestMigration("20240101_001", "Add user index")
    engine.Register(migration)

    // Test migration up
    err := engine.Up(ctx, "")
    helper.AssertNoError(err)

    // Verify migration was applied
    status, err := engine.GetStatus(ctx)
    helper.AssertNoError(err)
    helper.AssertEqual(1, len(status))
    helper.AssertEqual(true, status[0].Applied)
}
```

### Integration Test Example

```go
// +build integration

func TestMCPServerIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping integration test")
    }

    ctx := context.Background()

    server, err := mcp.NewMCPServer()
    if err != nil {
        t.Fatal(err)
    }
    defer server.Close()

    // Test MCP protocol
    request := mcp.MCPRequest{
        JSONRPC: "2.0",
        ID:      1,
        Method:  "initialize",
    }

    // Your test logic here
}
```

### Benchmark Example

```go
func BenchmarkMigrationEngine(b *testing.B) {
    helper := helpers.NewTestHelper(&testing.T{})
    ctx := context.Background()

    if err := helper.Setup(ctx); err != nil {
        b.Fatal(err)
    }
    defer helper.Cleanup(ctx)

    engine := helper.CreateEngine()

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        migration := helpers.NewTestMigration(
            fmt.Sprintf("2024010%d_001", i),
            "Test migration",
        )
        engine.Register(migration)
    }
}
```

## CI/CD Testing

Tests run automatically in GitHub Actions on:
- Push to `main`, `develop`, or `claude/*` branches
- Pull requests to `main` or `develop`

### CI Workflows

- **Unit Tests**: Run on multiple Go versions (1.21-1.24)
- **Integration Tests**: Run with MongoDB service
- **Coverage**: Upload to Codecov
- **Lint**: Run golangci-lint
- **Benchmarks**: Performance testing
- **Docker Tests**: Full integration in Docker
- **Build Verification**: Test builds on Linux, macOS, Windows

### Local CI Simulation

```bash
# Run what CI runs
make ci-test

# Run full CI suite including linting
make ci-test-full
```

## Test Coverage

### Viewing Coverage

```bash
# Generate coverage report
make test-coverage

# Open HTML report
open coverage.html  # macOS
xdg-open coverage.html  # Linux
```

### Coverage Goals

- **Overall**: > 80%
- **Core packages** (`migration`, `mcp`): > 85%
- **Config**: > 75%
- **CLI**: > 60%

## MongoDB for Testing

### Local MongoDB

```bash
# Start MongoDB
make db-up

# Stop MongoDB
make db-down
```

### Docker Compose

Integration tests can run in Docker with MongoDB:

```bash
make test-integration-docker
```

This uses `docker-compose.test.yml` which includes:
- MongoDB 7.0 service
- Test runner container
- Isolated network

## Troubleshooting

### MongoDB Connection Issues

```bash
# Check MongoDB is running
docker ps | grep mongo

# Check connection
mongosh mongodb://localhost:27017
```

### Test Failures

```bash
# Clean test cache
make test-clean

# Run with verbose output
go test -v ./...

# Run specific test
go test -v -run TestMigrationEngine ./migration
```

### Integration Test Issues

```bash
# Check MongoDB health
docker logs mongo-essential-test-db

# Run integration tests with more details
go test -v -tags=integration ./test/integration/...
```

## Best Practices

1. **Isolation**: Each test should be independent
2. **Cleanup**: Always clean up resources (use `defer`)
3. **Naming**: Use descriptive test names (`TestMigrationEngine_AppliesMigrations`)
4. **Table Tests**: Use table-driven tests for multiple scenarios
5. **Context**: Always use context with timeout
6. **Mocking**: Use mtest for MongoDB mocking in unit tests
7. **Tags**: Use build tags for integration tests

## Resources

- [Go Testing Package](https://pkg.go.dev/testing)
- [MongoDB Go Driver Testing](https://pkg.go.dev/go.mongodb.org/mongo-driver/mongo/integration/mtest)
- [Table Driven Tests](https://github.com/golang/go/wiki/TableDrivenTests)
- [Test Coverage](https://go.dev/blog/cover)

## Contributing

When adding new features:

1. Write unit tests first (TDD)
2. Add integration tests for end-to-end flows
3. Ensure coverage doesn't decrease
4. Run `make ci-test` before committing
5. Update this documentation if adding new test utilities
