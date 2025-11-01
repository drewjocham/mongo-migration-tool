# Linting Guide

## Current Linting Setup

This project uses `golangci-lint` for code quality checks. However, there's a known compatibility issue with Go 1.24.

## Issue: golangci-lint v1.50.1 + Go 1.24 Incompatibility

The current `golangci-lint` version (v1.50.1) has compatibility issues with Go 1.24, causing a panic:

```
panic: unsupported version: 2
```

This is due to changes in Go's export data format that older versions of golangci-lint don't support.

## Workarounds

### Option 1: Use go vet (Recommended for now)

The project includes a `make vet` command that uses Go's built-in static analysis:

```bash
make vet
```

This works perfectly with Go 1.24 and covers essential checks.

### Option 2: Upgrade golangci-lint

Upgrade to golangci-lint v1.56.0 or later, which supports Go 1.24:

```bash
# Using Go install
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Or using Homebrew on macOS
brew upgrade golangci-lint
```

After upgrading, the `.golangci.yml` file has been updated to:
- Remove the deprecated `version: 2` field
- Change `mnd` to `gomnd` (for v1.50.x)
- Remove `copyloopvar` (not available in v1.50.x)
- Disable `gocritic` temporarily due to GOROOT issues

### Option 3: Use staticcheck

An alternative linter that works well with Go 1.24:

```bash
go install honnef.co/go/tools/cmd/staticcheck@latest
staticcheck ./...
```

## Configuration Changes Made

The `.golangci.yml` file has been updated:

1. **Removed**: `version: 2` field (causes config parsing error)
2. **Renamed**: `mnd` → `gomnd` (for compatibility with v1.50.1)
3. **Removed**: `copyloopvar` linter (not available in v1.50.1)
4. **Disabled**: `gocritic` linter (GOROOT issues with v1.50.1)

## Recommended Approach

For development:
1. Use `make vet` for quick static analysis
2. Use `make test` for unit tests
3. Consider upgrading golangci-lint to v1.56.0+ for full linting

For CI/CD:
1. Pin golangci-lint to v1.56.0 or later
2. Run `make ci-test` which includes vet, tests, and coverage

## Additional Static Analysis Tools

Other tools that work well with Go 1.24:

```bash
# Go vet (built-in)
go vet ./...

# Staticcheck
staticcheck ./...

# Go fix (auto-fixes some issues)
go fix ./...

# Ineffassign (finds ineffectual assignments)
go install github.com/gordonklaus/ineffassign@latest
ineffassign ./...
```

## Future Updates

Once golangci-lint is upgraded to v1.56.0+:
- Re-enable all linters
- Update linter names to current versions
- Consider adding new linters like `govet/copylocks`, `govet/nilness`
