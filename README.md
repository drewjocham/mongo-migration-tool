# mongo-migration

[![Go Report Card](https://goreportcard.com/badge/github.com/drewjocham/mongo-migration-tool)](https://goreportcard.com/report/github.com/drewjocham/mongo-migration-tool)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Reference](https://pkg.go.dev/badge/github.com/drewjocham/mongo-migration-tool.svg)](https://pkg.go.dev/github.com/drewjocham/mongo-migration-tool)

A comprehensive MongoDB migration tool with AI-powered insights via MCP. Think Liquibase/Flyway for MongoDB, with a protocol for intelligent database optimization recommendations from your favorite AI assistant.

## Features

### **Database Migration Management**
- **Version Control**: Track and manage database schema changes
- **Up/Down Migrations**: Rollback capability
- **Migration Status**: Track applied and pending migrations
- **Force Migration**: Mark migrations as applied without execution
- **Integration Ready**: Works with existing Go projects and CI/CD pipelines

### **Developer Tools**
- **Certificate Management**: Debug and fix SSL/TLS certificate issues
- **CLI Interface**:  command-line interface built with Cobra
- **MCP Integration**: Model Context Protocol server for AI agents

## Installation

Choose your preferred installation method:

### Homebrew (macOS/Linux) - Recommended

```bash
brew tap drewjocham/mongo-migration-tool
brew install mongo-migration-tool

mongo-migration version
```

### Docker

```bash
    docker pull ghcr.io/drewjocham/mongo-migration-tool:latest
    docker run --rm -v $(pwd):/workspace ghcr.io/drewjocham/mongo-migration-tool:latest --help
```

* Run this command in your project's root directory to create a simple, static key file for development:
```bash
    echo "ThisIsADevKeyForInternalRSCommunication" > mongo_keyfile
    # permissions for the MongoDB Key File
    chmod 600 mongo_keyfile
    # 2. Verify
    ls -l mongo_keyfile
```

* Stop and starting the compose project
* 
```bash
    docker-compose down &&  \
    docker compose up -d mongo-migrate
```

```bash
    # Delete the volume (ALL DATA WILL BE LOST)
    docker volume rm mongo-migration-_mongo_data
```
* Quick path to get it working
* 
```bash
     docker compose up -d mongo-cli
     docker compose run --rm mongo-migrate status
```
If you want to run mongo-migration on your local machine:
*  Ensure Mongo is running via Docker (docker compose up -d mongo-cli).
*  Set .env like above with localhost and credentials.

```shell
  go run . --config .env status
```
* Check the staus and whether the library is installed
```shell
# or if installed:
    mongo-migration status
```
* Access the shell (using authentication):
```shell
    docker exec -it mongo-migration--mongo-cli-1 mongosh -u admin -p password --authenticationDatabase admin
```
* Another way to access the shell:
```shell
  docker run --rm -it --network mongo-migration-_cli-network alpine sh
```
```bash
    # Add to your Go project
    go get github.com/drewjocham/mongo-migration-tool@latest
```

### Binary Download

Download pre-built binaries from [GitHub Releases](https://github.com/drewjocham/mongo-migration-tool/releases) for Linux, macOS, Windows, and FreeBSD.

### Go Install (Development)

```bash
    go install github.com/drewjocham/mongo-migration-tool@latest
```

** For detailed installation instructions, platform-specific guides, and troubleshooting, see [INSTALL.md](INSTALL.md)**

## Quick Start

### 1. Database Migrations

```bash
    # Initialize configuration
    cp .env.example .env
    # Edit .env with your MongoDB connection details
```
```bash
    # Check migration status
    ./mongo-essential status
```
```bash
    # Create a new migration
    mongo-essential create add_user_indexes
```
```bash
    # Run pending migrations
    mongo-essential up
```
```bash
    # Rollback last migration
    mongo-essential down --target 20231201_001
```

### 2. AI Assistant Integration (MCP)

```bash
# Start MCP server for AI assistants like Ollama, Claude, Goose
mongo-migration mcp

# Start with example migrations for testing
mongo-migration mcp --with-examples

# Test MCP integration
make mcp-test
```

Then configure your AI assistant to use the MCP server:
- **Ollama**: Add to `~/.config/ollama/mcp-config.json`
- **Claude Desktop**: Add to Claude Desktop configuration
- **Goose**: Use with `--mcp-config` flag

See [MCP Integration Guide](MCP.md) for detailed setup instructions.

## Configuration

### Environment Variables

```bash
# MongoDB Configuration
MONGO_URL=mongodb://localhost:27017
MONGO_DATABASE=your_database
MONGO_USERNAME=username
MONGO_PASSWORD=password

# SSL/TLS Settings
MONGO_SSL_ENABLED=true
MONGO_SSL_INSECURE=false
```

See [.env.example](./.env.example) for complete configuration options.

## Library Usage

Use mongo-migration as a Go library in your applications:

```go
package main

import (
    "context"
    "log"
    
    "github.com/drewjocham/mongo-migration-tool/config"
    "github.com/drewjocham/mongo-migration-tool/migration"
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
    
    // Register your migrations
    engine.RegisterMany(
        &AddUserIndexesMigration{},
        &CreateProductCollection{},
    )
    
    // Run migrations
    if err := engine.Up(context.Background(), ""); err != nil {
        log.Fatal(err)
    }
    
    log.Println("Migrations completed successfully!")
}
```

**For complete library documentation and examples, see [LIBRARY.md](LIBRARY.md)**

## Documentation

### Comprehensive Guides

| Guide | Description |
|-------|-------------|
| **[INSTALL.md](INSTALL.md)** | Complete installation guide for all platforms |
| **[LIBRARY.md](LIBRARY.md)** | Go library usage, API reference, and examples |
| **[MCP.md](MCP.md)** | Model Context Protocol integration guide |
| **[CONTRIBUTING.md](CONTRIBUTING.md)** | Development and contribution guidelines |

### Commands

| Command | Description |
|---------|-------------|
| `mongo-migration up` | Run pending migrations |
| `mongo-migration down` | Rollback migrations |
| `mongo-migration status` | Show migration status |
| `mongo-migration create <name>` | Create new migration |
| `mongo-migration force <version>` | Force mark migration as applied |
| `mongo-migration mcp` | Start MCP server for AI assistants |
| `mongo-migration mcp --with-examples` | Start MCP server with example migrations |

## Use Cases

### Development Teams
- **Schema Evolution**: Version-controlled database migrations
- **CI/CD Integration**: Automated migration deployment
- **Development Setup**: Quick database setup and seeding
- **Certificate Issues**: Debug connectivity problems in corporate environments

## Architecture

```
mongo-migration/
├── cmd/                    # CLI commands
│   ├── root.go            # Root command and global flags
│   └── migration.go       # Migration commands
├── internal/
│   ├── config/            # Configuration management
│   └── migration/         # Migration engine
├── migrations/            # Sample migrations
└── docs/                  # Additional documentation
```

## 🤝 Contributing

We welcome contributions! Please see [CONTRIBUTING.md](./CONTRIBUTING.md) for details.

### Development Setup

```bash
# Clone the repository
git clone https://github.com/drewjocham/mongo-migration-tool.git
cd mongo-migration-tool

# Install dependencies
go mod tidy

# Build the binary
go build -o mongo-migration .

# Run tests (disable go.work.bak so vendored deps resolve)
go test ./...

# Run Docker-backed CLI integration tests (requires Docker)
go test -tags=integration ./integration
# or use the Makefile shortcut
make integration-test

# Run linter
golangci-lint run
```

### Adding New Features

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Add tests and documentation
5. Submit a pull request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [Cobra](https://github.com/spf13/cobra) for CLI framework
- [MongoDB Go Driver](https://github.com/mongodb/mongo-go-driver) for database connectivity

## 🔗 Links & Resources

### Project Resources
- **[Go Package Documentation](https://pkg.go.dev/github.com/drewjocham/mongo-migration-tool)** - Complete API reference
- **[GitHub Repository](https://github.com/drewjocham/mongo-migration-tool)** - Source code and releases
- **[Issue Tracker](https://github.com/drewjocham/mongo-migration-tool/issues)** - Bug reports and feature requests
- **[Homebrew Formula](https://github.com/drewjocham/homebrew-mongo-migration-tool)** - Homebrew tap repository
- **[Docker Images](https://ghcr.io/drewjocham/mongo-migration-tool)** - Container registry

### Documentation
- **[Installation Guide](INSTALL.md)** - All installation methods and troubleshooting
- **[Library Documentation](LIBRARY.md)** - Go library usage and examples
- **[MCP Integration](MCP.md)** - AI assistant integration guide
- **[Contributing Guide](CONTRIBUTING.md)** - Development setup and guidelines

## Support & Community

- **Issues**: [GitHub Issues](https://github.com/drewjocham/mongo-migration-tool/issues)
- **Discussions**: [GitHub Discussions](https://github.com/drewjocham/mongo-migration-tool/discussions)
- **Contact**: [Project Maintainer](https://github.com/drewjocham)
- **Examples**: See the `examples/` directory in the repository

---

**Made with ❤️ for the MongoDB community**
