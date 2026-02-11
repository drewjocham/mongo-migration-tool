# Contributing to mongo-migration

Thank you for considering contributing to mongo-migration! This document provides guidelines and information for contributors.

## Table of Contents

1. [Code of Conduct](#code-of-conduct)
2. [How to Contribute](#how-to-contribute)
3. [Development Setup](#development-setup)
4. [Project Structure](#project-structure)
5. [Development Workflow](#development-workflow)
6. [Testing](#testing)
7. [Code Style](#code-style)
8. [Submitting Changes](#submitting-changes)
9. [Release Process](#release-process)

## Code of Conduct

This project adheres to the [Contributor Covenant Code of Conduct](https://www.contributor-covenant.org/version/2/0/code_of_conduct/). By participating, you are expected to uphold this code.

## How to Contribute

### Reporting Bugs

Before reporting a bug:
1. Check the [existing issues](https://github.com/drewjocham/mongo-migration-tool/issues) to avoid duplicates
2. Use the latest version of mongo-migration
3. Include detailed information about your environment

When reporting bugs, include:
- **Version**: mongo-migration version and Go version
- **Environment**: OS, MongoDB version, deployment details
- **Steps to Reproduce**: Clear, minimal reproduction steps
- **Expected vs Actual**: What should happen vs what actually happens
- **Logs**: Relevant error messages or logs
- **Configuration**: Sanitized configuration (remove sensitive data)

### Suggesting Features

Feature requests are welcome! Please:
1. Check if the feature already exists or is planned
2. Open an issue with the "enhancement" label
3. Describe the use case and business value
4. Provide implementation suggestions if you have them

### Contributing Code

We welcome pull requests for:
- Bug fixes
- New features
- Documentation improvements
- Performance optimizations
- Test coverage improvements

## Development Setup

### Prerequisites

- **Go 1.24 or later**
- **Git**
- **MongoDB** (for testing)
- **Docker** (optional, for containerized testing)
- **Make** (optional, for build automation)

### Clone and Setup

```bash
# Fork the repository on GitHub, then clone your fork
git clone https://github.com/drewjocham/mongo-migration-tool.git
cd mongo-migration-tool

# Add the upstream remote
git remote add upstream https://github.com/drewjocham/mongo-migration-tool.git
```

### Build the Project

```bash
make build

# Run the binary
./mongo-migration version
```

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run integration tests
make test-integration
```

### MongoDB for Development

#### Option 1: Local MongoDB

```bash
# Install MongoDB locally
brew tap mongo-migration
brew install mongo-migration

# Start MongoDB
brew services start mongodb-community

# Or run manually
mongod --config /usr/local/etc/mongod.conf
```

#### Option 2: Docker

```bash
# Start MongoDB in Docker
docker run --name mongo-dev -p 27017:27017 -d mongo:8

# Stop when done
docker stop mongo-dev
docker rm mongo-dev
```

#### Option 3: Docker Compose

```bash
# Use the development compose file
docker-compose -f docker/docker-compose.dev.yml up -d

# Stop when done
docker-compose -f docker/docker-compose.dev.yml down
```

### Environment Configuration

Create a `.env.dev` file for development:

```bash
# Copy the example configuration
cp .env.example .env.dev

# Edit the configuration
MONGO_URL=mongodb://localhost:27017
MONGO_DATABASE=mongo_migration_tool_dev
MIGRATIONS_PATH=./test-migrations
MIGRATIONS_COLLECTION=schema_migrations

# AI Agents (optional)
AI_ENABLED=false
# AI_PROVIDER=openai
# OPENAI_API_KEY=your_key_here

# Google Docs (optional)
GOOGLE_DOCS_ENABLED=false
# GOOGLE_CREDENTIALS_PATH=./credentials.json
```

## Project Structure

```
mongo-migration/
├── .github/                    # GitHub Actions workflows
│   └── workflows/             
├── cmd/                        # CLI commands and main entry
│   └── mongo-migration/       
├── internal/                   # Internal packages
│   ├── config/                # Configuration management
│   ├── migration/             # Migration engine
│   ├── ai/                    # AI analysis functionality
│   ├── cert/                  # Certificate utilities
│   └── mcp/                   # Model Context Protocol server
├── pkg/                        # Public packages (exported)
├── examples/                   # Usage examples
├── migrations/                 # Example migrations
├── docs/                       # Additional documentation
├── scripts/                    # Build and deployment scripts
├── .goreleaser.yaml           # GoReleaser configuration
├── Makefile                   # Build automation
├── Dockerfile                 # Container image definition
└── docker-compose.*.yml       # Docker Compose configurations
```

### Package Guidelines

- **`cmd/`**: CLI-specific code, keep minimal
- **`internal/`**: Private packages, core business logic
- **`pkg/`**: Public packages that external users can import
- **`examples/`**: Working examples and tutorials

## Development Workflow

### 1. Start Working on an Issue

```bash
# Make sure you're on the main branch
git checkout main

# Pull latest changes
git pull upstream main

# Create a feature branch
git checkout -b feature/your-feature-name
# or for bug fixes
git checkout -b fix/issue-description
```

### 2. Make Changes

Follow these guidelines:
- Write clear, concise commit messages
- Keep changes focused and atomic
- Add tests for new functionality
- Update documentation as needed
- Follow the existing code style

### 3. Test Your Changes

```bash
# Run tests
make test

# Run linting
make lint

# Build and test manually
make build
./mongo-migration --help

# Test with real MongoDB (if applicable)
./mongo-migration status
```

### 4. Commit Changes

```bash
# Add your changes
git add .

# Commit with a descriptive message
git commit -m "feat: add new migration rollback functionality

- Add rollback command to CLI
- Implement down migration logic
- Add comprehensive tests
- Update documentation

Fixes #123"
```

#### Commit Message Convention

We follow [Conventional Commits](https://conventionalcommits.org/):

- `feat:` new features
- `fix:` bug fixes
- `docs:` documentation changes
- `style:` formatting, missing semicolons, etc.
- `refactor:` code changes that neither fix bugs nor add features
- `test:` adding missing tests
- `chore:` changes to build process, dependencies, etc.

## Testing

### Unit Tests

```bash
# Run unit tests
go test ./internal/...

# Run with coverage
go test -cover ./internal/...

# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Integration Tests

```bash
# Start MongoDB for testing
docker run --name mongo-test -p 27017:27017 -d mongo:7

# Run integration tests
go test -tags integration ./...

# Clean up
docker stop mongo-test && docker rm mongo-test
```

### Test Structure

### Writing Good Tests

- Use table-driven tests for multiple scenarios
- Test both success and error cases
- Use meaningful test names
- Keep tests focused and independent
- Use test helpers for common setup

## Code Style

### Guidelines

- Use `gofmt` and `goimports`
- Follow Go naming conventions
- Add comments for exported functions
- Use meaningful variable names
- Keep functions focused and small
- Handle errors properly
- Use context for cancellation

### Linting

```bash
   # Run the linter
   make lint
   
   # Or directly with golangci-lint
   golangci-lint run
   
   # Auto-fix some issues
   golangci-lint run --fix
```

## Submitting Changes

### 1. Push to Your Fork

```bash
git push origin feature/your-feature-name
```

### 2. Create a Pull Request

1. Go to GitHub and create a pull request from your fork
2. Use a clear title and description
3. Reference any related issues
4. Add screenshots for UI changes
5. Check that CI passes

### Pull Request Template

```markdown
## Description
Brief description of the changes.

## Type of Change
- [ ] Bug fix (non-breaking change that fixes an issue)
- [ ] New feature (non-breaking change that adds functionality)
- [ ] Breaking change (fix or feature that would cause existing functionality to not work as expected)
- [ ] Documentation update

## Testing
- [ ] Unit tests pass
- [ ] Integration tests pass (if applicable)
- [ ] Manual testing performed

## Checklist
- [ ] Code follows the project's style guidelines
- [ ] Self-review of code completed
- [ ] Code is commented, particularly in hard-to-understand areas
- [ ] Documentation updated (if needed)
- [ ] No new warnings introduced
```

### 3. Code Review Process

- Maintainers will review your pull request
- Address feedback promptly
- Keep discussions focused and professional
- Be open to suggestions and changes

## Release Process

### Versioning

We use [Semantic Versioning](https://semver.org/):
- **MAJOR**: Breaking changes
- **MINOR**: New features (backward compatible)
- **PATCH**: Bug fixes (backward compatible)

### Release Workflow

1. **Create Release Branch**:

```bash
  git checkout -b release/v0.1.0
```

2. **Update Version Files**:
   - Update version in `cmd/mongo-migration/version.go`
   - Update CHANGELOG.md
   - Update documentation if needed

3. **Test Release**:
   ```bash
   make test
   make lint
   make build-all
   ```

4. **Create Pull Request**: Target the main branch

5. **Tag Release** (maintainers only):

```bash
   git tag -a v1.2.0 -m "Release v1.2.0"
   git push origin v1.2.0
```

6. **GitHub Actions** will automatically:
   - Build binaries for all platforms
   - Create GitHub release
   - Build and push Docker images
   - Update Homebrew formula

### Manual Homebrew Update (if needed)

If the Homebrew formula does not update automatically, you can update it by hand:

1. Open the tap repo: `https://github.com/drewjocham/homebrew-mongo-migration-tool`
2. Update the formula file (typically `Formula/mongo-migration-tool.rb`) with:
   - The new version
   - The release asset URLs
   - The SHA256 for each asset (download the assets and run `shasum -a 256 <file>`)
3. Commit and push the tap repo changes.

Users can then run:

```bash
brew update
brew upgrade mongo-migration-tool
```

### Changelog Format

```markdown
## [1.2.0] - 2024-01-15

### Added
- New AI analysis features
- MCP server integration
- Docker image support

### Changed
- Improved error handling
- Updated dependencies

### Fixed
- Migration rollback issues
- Certificate validation bugs

### Deprecated
- Old configuration format (will be removed in v2.0.0)
```

## Getting Help

- **Documentation**: Start with README.md and other docs
- **Issues**: Search existing issues or create a new one
- **Discussions**: Use GitHub Discussions for questions
- **Code**: Read the source code and tests for examples

## Recognition

Contributors are recognized in:
- CHANGELOG.md for their contributions
- GitHub contributors page
- Special thanks in release notes for significant contributions

Thank you for contributing to mongo-migration!
