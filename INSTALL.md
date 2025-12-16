# Installation Guide

This guide covers all the ways you can install and use mongo-migration: as a CLI tool via Homebrew, as a standalone binary, as a Docker container, or as a Go library.

## Table of Contents

1. [Homebrew Installation (macOS/Linux)](#homebrew-installation-macoslinux)
2. [Binary Installation](#binary-installation)
3. [Docker Installation](#docker-installation)
4. [Go Library Installation](#go-library-installation)
5. [Building from Source](#building-from-source)
6. [Configuration](#configuration)
7. [Verification](#verification)

## Homebrew Installation (macOS/Linux)

The easiest way to install mongo-migration on macOS and Linux is via Homebrew.

### Prerequisites

- [Homebrew](https://brew.sh/) installed on your system

### Install from Custom Tap

```bash
# Add our custom Homebrew tap
brew tap jocham/mongo-migration-tool

# Install mongo-migration
brew install mongo-migration-tool

# Verify installation
mongo-migration version
```

### Upgrade

```bash
# Upgrade to the latest version
brew upgrade migration-tool
```

### Uninstall

```bash
# Uninstall mongo-migration
brew uninstall mongo-migration-tool

# Remove the tap (optional)
brew untap jocham/mongo-migration-tool
```

## Binary Installation

Download pre-built binaries for your platform from our [GitHub Releases](https://github.com/jocham/mongo-migration-tool/releases).

### Available Platforms

- **Linux**: x86_64, ARM64
- **macOS**: x86_64 (Intel), ARM64 (Apple Silicon)
- **Windows**: x86_64
- **FreeBSD**: x86_64, ARM64

### Linux / macOS

```bash
# Download the latest release (adjust URL for your platform)
curl -LO https://github.com/jocham/mongo-migration-tool/releases/latest/download/mongo-migration_linux_amd64.tar.gz

# Extract the binary
tar -xzf mongo-migration_linux_amd64.tar.gz

# Make executable and move to PATH
chmod +x mongo-migration
sudo mv mongo-migration /usr/local/bin/

# Verify installation
mongo-migration version
```

### Windows

1. Download the Windows binary from the [releases page](https://github.com/jocham/mongo-migration-tool/releases)
2. Extract the `.zip` file
3. Add the binary location to your system PATH
4. Open a new command prompt and verify: `mongo-migration version`

### Installing Specific Versions

```bash
# Install specific version (replace v1.2.3 with desired version)
curl -LO https://github.com/jocham/mongo-migration-tool/releases/download/v1.2.3/mongo-migration_linux_am
```

## Docker Installation

mongo-migration is available as a Docker image for containerized environments.

### Available Images

- **Multi-arch support**: AMD64, ARM64
- **Tags**: `latest`, version tags (e.g., `v1.2.3`)

### Basic Usage

```bash
# Pull the latest image
docker pull ghcr.io/jocham/mongo-migration-tool:latest

# Run migrations (mount your migrations directory)
docker run --rm \
  -v $(pwd)/migrations:/migrations \
  -e MONGO_URL="mongodb://your-mongo-host:27017" \
  -e MONGO_DATABASE="your-database" \
  ghcr.io/jocham/mongo-migration-tool:latest \
  up

# Run with custom configuration file
docker run --rm \
  -v $(pwd)/.env:/app/.env \
  -v $(pwd)/migrations:/migrations \
  ghcr.io/jocham/mongo-migration-tool:latest \
  status
```

### Docker Compose Example

Create a `docker-compose.yml` file:

```yaml
version: '3.8'

services:
  mongo-migration:
    image: ghcr.io/jocham/mongo-migration-tool:latest
    environment:
      - MONGO_URL=mongodb://mongodb:27017
      - MONGO_DATABASE=myapp
      - MIGRATIONS_PATH=/migrations
    volumes:
      - ./migrations:/migrations
      - ./.env:/app/.env
    depends_on:
      - mongodb
    command: ["status"]

  mongodb:
    image: mongo:7
    ports:
      - "27017:27017"
    volumes:
      - mongo-data:/data/db

volumes:
  mongo-data:
```

Run with:

```bash
docker-compose up mongo-migration
```

## Go Library Installation

Use mongo-migration as a library in your Go projects.

### Prerequisites

- Go 1.24 or later

### Installation

```bash
# Add to your Go project
go get github.com/jocham/mongo-migration-tool@latest

# Or install specific version
go get github.com/jocham/mongo-migration-tool@v1.2.3
```

### Basic Usage

```go
package main

import (
    "context"
    "log"
    
    "github.com/jocham/mongo-migration-tool/config"
    "github.com/jocham/mongo-migration-tool/migration"
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
    // Load configuration
    cfg, err := config.Load()
    if err != nil {
        log.Fatal(err)
    }
    
    // Connect to MongoDB
    client, err := mongo.Connect(context.Background(), 
        options.Client().ApplyURI(cfg.GetConnectionString()))
    if err != nil {
        log.Fatal(err)
    }
    defer client.Disconnect(context.Background())
    
    // Create migration engine
    engine := migration.NewEngine(
        client.Database(cfg.Database), 
        cfg.MigrationsCollection)
    
    // Run migrations
    if err := engine.Up(context.Background(), ""); err != nil {
        log.Fatal(err)
    }
    
    log.Println("Migrations completed!")
}
```

For detailed library usage, see [LIBRARY.md](LIBRARY.md).

## Building from Source

Build mongo-migration from source code.

### Prerequisites

- Go 1.25 or later
- Git

### Build Steps

```bash
# Clone the repository
git clone https://github.com/jocham/mongo-migration-tool.git
cd mongo-migration-tool

# Build for your current platform
go build -o mongo-migration ./cmd/mongo-migration

# Or use make
make build

# Install to GOPATH/bin
go install ./cmd/mongo-migration

# Build for all platforms (requires goreleaser)
make build-all
```

### Run GoReleaser

1.  **Install GoReleaser**:
    *   **Using Homebrew**:
        ```bash
        brew install goreleaser
        ```
    *   **Using `go install`**:
        ```bash
        go install github.com/goreleaser/goreleaser@latest
        ```

2.  **Set GitHub Token**:
    Make sure you have a `GITHUB_TOKEN` environment variable set with `repo` scope.
    ```bash
    export GITHUB_TOKEN="your_github_token"
    ```

3.  **Run GoReleaser**:
    ```bash
    export GITHUB_TOKEN=<PLACEHOLDER>
    goreleaser release --clean
    ```

### Development Build

```bash
# Build with debug information
go build -ldflags "-X main.version=dev" -o mongo-migration ./cmd/mongo-migration

# Run tests
make test

# Run linting
make lint
```

## Configuration

mongo-migration can be configured through environment variables or configuration files.

### Environment Variables

Create a `.env` file or set environment variables:

```bash
# Required MongoDB settings
export MONGO_URL="mongodb://localhost:27017"
export MONGO_DATABASE="myapp"

# Optional settings
export MIGRATIONS_PATH="./migrations"
export MIGRATIONS_COLLECTION="schema_migrations"

# Authentication (if required)
export MONGO_USERNAME="username"
export MONGO_PASSWORD="password"

# SSL/TLS (for cloud providers)
export MONGO_SSL_ENABLED="true"
export MONGO_SSL_CERT_PATH="./certs/client.pem"
export MONGO_SSL_KEY_PATH="./certs/client-key.pem"
export MONGO_SSL_CA_CERT_PATH="./certs/ca.pem"
```

### Configuration File

Create a `.env` file in your project directory:

```env
MONGO_URL=mongodb://localhost:27017
MONGO_DATABASE=myapp
MIGRATIONS_PATH=./migrations
MIGRATIONS_COLLECTION=schema_migrations

# Connection pool settings
MONGO_MAX_POOL_SIZE=10
MONGO_MIN_POOL_SIZE=1
MONGO_TIMEOUT=60

# AI Analysis (optional)
AI_ENABLED=false
AI_PROVIDER=openai
OPENAI_API_KEY=your_openai_key

# Google Docs Integration (optional)
GOOGLE_DOCS_ENABLED=false
GOOGLE_CREDENTIALS_PATH=./credentials.json
```

### Configuration Priority

Configuration is loaded in the following order (later sources override earlier ones):

1. Default values
2. Configuration file (`.env`)
3. Environment variables
4. Command-line flags

## Verification

### Verify CLI Installation

```bash
# Check version
mongo-migration version

# Check available commands
mongo-migration help

# Test connection (requires configuration)
mongo-migration status
```

### Verify Docker Installation

```bash
# Check Docker image
docker run --rm ghcr.io/jocham/mongo-migration-tool:latest version

# Test with sample configuration
docker run --rm \
  -e MONGO_URL="mongodb://host.docker.internal:27017" \
  -e MONGO_DATABASE="test" \
  ghcr.io/jocham/mongo-migration-tool:latest \
  status
```

### Verify Go Library Installation

Create a test file `test.go`:

```go
package main

import (
    "fmt"
    "github.com/jocham/mongo-migration-tool/config"
)

func main() {
    cfg := &config.Config{
        MongoURL: "mongodb://localhost:27017",
        Database: "test",
    }
    fmt.Println("Connection string:", cfg.GetConnectionString())
}
```

Run it:

```bash
go mod init test
go get github.com/jocham/mongo-migration-tool@latest
go run test.go
```

## Troubleshooting

### Common Issues

#### Homebrew Installation Issues

```bash
# If tap already exists
brew untap jocham/mongo-migration-tool
brew tap jocham/mongo-migration-tool

# Clear Homebrew cache
brew cleanup
rm -rf $(brew --cache)
```

#### Binary Permission Issues (Linux/macOS)

```bash
# Make binary executable
chmod +x mongo-migration

# If "command not found"
echo $PATH
# Make sure /usr/local/bin is in your PATH
```

#### Docker Issues

```bash
# If image pull fails
docker logout ghcr.io
docker login ghcr.io

# Check if image exists
docker images | grep mongo-migration
```

#### Go Module Issues

```bash
# Clean module cache
go clean -modcache

# Update dependencies
go mod tidy
go mod download
```

### Getting Help

- **Documentation**: [GitHub Repository](https://github.com/jocham/mongo-migration-tool)
- **Issues**: [GitHub Issues](https://github.com/jocham/mongo-migration-tool/issues)
- **Library Docs**: [pkg.go.dev](https://pkg.go.dev/github.com/jocham/mongo-migration-tool)

## Next Steps

After installation, you might want to:

1. **Create your first migration**: `mongo-migration create add_user_index`
2. **Set up your project**: Create migrations directory and configure environment
3. **Explore examples**: Check the [examples directory](examples/) in the repository
4. **Read the library documentation**: See [LIBRARY.md](LIBRARY.md) for Go library usage
5. **Set up CI/CD**: Integrate migrations into your deployment pipeline

## Supported Versions

- **Go**: 1.24+
- **MongoDB**: 4.4+ (tested with 4.4, 5.0, 6.0, 7.0)
- **Operating Systems**: Linux, macOS, Windows, FreeBSD
- **Architectures**: AMD64, ARM64

## Security Considerations

- Store credentials securely (use environment variables, not hardcoded values)
- Use SSL/TLS for production connections
- Limit MongoDB user permissions to what's needed for migrations
- Consider using MongoDB Atlas or other managed services for production
- Regularly update to the latest version for security patches
