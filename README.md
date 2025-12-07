# mongo-migration

[![Go Report Card](https://goreportcard.com/badge/github.com/jocham/mongo-migration)](https://goreportcard.com/report/github.com/drewjocham/mongo-migration)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Reference](https://pkg.go.dev/badge/github.com/jocham/mongo-migration.svg)](https://pkg.go.dev/github.com/drewjocham/mongo-migration)

A comprehensive MongoDB migration and database analysis tool with AI-powered insights. Think Liquibase/Flyway for MongoDB, plus intelligent database optimization recommendations.

## 🚀 Features

### 📊 **AI-Powered Database Analysis**
- **Multi-Provider AI Support**: OpenAI GPT-4, Google Gemini, Anthropic Claude
- **Comprehensive Analysis**: Schema, performance, indexing, and optimization recommendations
- **Oplog & Replication**: Deep analysis of MongoDB replication health and oplog patterns
- **Change Stream Optimization**: Real-time data processing pattern analysis
- **Google Docs Integration**: Export professional reports directly to Google Docs

### 🔄 **Database Migration Management**
- **Version Control**: Track and manage database schema changes
- **Up/Down Migrations**: Full rollback capability
- **Migration Status**: Track applied and pending migrations
- **Force Migration**: Mark migrations as applied without execution
- **Integration Ready**: Works with existing Go projects and CI/CD pipelines

### 🔧 **Developer Tools**
- **Certificate Management**: Debug and fix SSL/TLS certificate issues
- **Cloud Provider Support**: Optimized for STACKIT, AWS, Azure, GCP
- **CLI Interface**: Intuitive command-line interface built with Cobra
- **Configuration Flexible**: Environment variables, config files, or CLI flags
- **🤖 MCP Integration**: Model Context Protocol server for AI assistants (Ollama, Claude, Goose)

## 📦 Installation

Choose your preferred installation method:

### Homebrew (macOS/Linux) - Recommended

```bash
# Add the tap and install
brew tap jocham/mongo-migration
brew install mongo-migration

# Verify installation
mongo-migration version
```

### Docker

```bash
  # Pull and run
    docker pull ghcr.io/drewjocham/mongo-migration:latest
    docker run --rm -v $(pwd):/workspace ghcr.io/jocham/mongo-migration:latest --help
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
    go get github.com/drewjocham/mongo-migration@latest
```

### Binary Download

Download pre-built binaries from [GitHub Releases](https://github.com/drewjocham/mongo-migration/releases) for Linux, macOS, Windows, and FreeBSD.

### Go Install (Development)

```bash
    go install github.com/drewjocham/mongo-migration@latest
```

**📚 For detailed installation instructions, platform-specific guides, and troubleshooting, see [INSTALL.md](INSTALL.md)**

## 🎯 Quick Start

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

### 2. AI-Powered Analysis

```bash
# Basic database analysis
mongo-migration ai analyze --provider openai

# Detailed schema analysis
mongo-migration ai schema --provider gemini --detail

# Oplog and replication analysis
mongo-migration ai oplog --provider openai --google-docs

# Change stream optimization
mongo-migration ai changestream --collection events --provider gemini

# Performance analysis with Google Docs export
mongo-migration ai performance --provider openai --google-docs \
  --docs-title "Production Performance Report" \
  --docs-share "team@company.com"
```

### 3. Certificate Troubleshooting

```bash
# Diagnose certificate issues
mongo-migration cert diagnose

# Check specific host certificate
mongo-migration cert check login.microsoftonline.com --verbose

# Fix common certificate problems
mongo-migration cert fix --apply
```

### 4. AI Assistant Integration (MCP)

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

## ⚙️ Configuration

### Environment Variables

```bash
# MongoDB Configuration
MONGO_URL=mongodb://localhost:27017
MONGO_DATABASE=your_database
MONGO_USERNAME=username
MONGO_PASSWORD=password

# AI Analysis
AI_ENABLED=true
AI_PROVIDER=openai  # openai, gemini, claude
OPENAI_API_KEY=your_openai_key
GEMINI_API_KEY=your_gemini_key

# Google Docs Integration
GOOGLE_DOCS_ENABLED=true
GOOGLE_CREDENTIALS_PATH=./credentials.json
GOOGLE_DRIVE_FOLDER_ID=folder_id
GOOGLE_DOCS_SHARE_WITH_EMAIL=team@company.com

# SSL/TLS Settings
MONGO_SSL_ENABLED=true
MONGO_SSL_INSECURE=false
```

See [.env.example](./.env.example) for complete configuration options.

## 🚀 Library Usage

Use mongo-migration as a Go library in your applications:

```go
package main

import (
    "context"
    "log"
    
    "github.com/drewjocham/mongo-migration/config"
    "github.com/drewjocham/mongo-migration/migration"
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
    
    log.Println("Migrations completed!")
}
```

**📚 For complete library documentation and examples, see [LIBRARY.md](LIBRARY.md)**

## 📖 Documentation

### Comprehensive Guides

| Guide | Description |
|-------|-------------|
| **[INSTALL.md](INSTALL.md)** | Complete installation guide for all platforms |
| **[LIBRARY.md](LIBRARY.md)** | Go library usage, API reference, and examples |
| **[MCP.md](MCP.md)** | Model Context Protocol integration guide |
| **[AI_ANALYSIS.md](AI_ANALYSIS.md)** | AI-powered database analysis documentation |
| **[CONTRIBUTING.md](CONTRIBUTING.md)** | Development and contribution guidelines |

### Commands

| Command | Description |
|---------|-------------|
| `mongo-migration up` | Run pending migrations |
| `mongo-migration down` | Rollback migrations |
| `mongo-migration status` | Show migration status |
| `mongo-migration create <name>` | Create new migration |
| `mongo-migration force <version>` | Force mark migration as applied |
| `mongo-migration ai analyze` | AI database analysis |
| `mongo-migration ai schema` | AI schema analysis |
| `mongo-migration ai performance` | AI performance analysis |
| `mongo-migration ai oplog` | AI oplog/replication analysis |
| `mongo-migration ai changestream` | AI change stream analysis |
| `mongo-migration cert diagnose` | Certificate diagnostics |
| `mongo-migration cert check <host>` | Check host certificate |
| `mongo-migration cert fix` | Fix certificate issues |
| `mongo-migration mcp` | Start MCP server for AI assistants |
| `mongo-migration mcp --with-examples` | Start MCP server with example migrations |

### AI Providers

| Provider | Models | Setup |
|----------|---------|-------|
| **OpenAI** | GPT-4o, GPT-4o-mini, GPT-3.5-turbo | Get API key from [OpenAI](https://platform.openai.com/api-keys) |
| **Google Gemini** | Gemini-1.5-flash, Gemini-1.5-pro | Get API key from [Google AI Studio](https://ai.google.dev/) |
| **Anthropic Claude** | Claude-3.5-Sonnet | Get API key from [Anthropic Console](https://console.anthropic.com/) |

### Google Docs Setup

1. Create a [Google Cloud Project](https://console.cloud.google.com/)
2. Enable Google Docs and Drive APIs
3. Create a service account and download JSON credentials
4. Set `GOOGLE_CREDENTIALS_PATH` to the JSON file path

## 💡 Use Cases

### Database Operations Teams
- **Health Monitoring**: Regular AI-powered database health checks
- **Performance Optimization**: Identify and fix performance bottlenecks
- **Replication Management**: Monitor oplog and replica set health
- **Change Tracking**: Optimize change stream configurations

### Development Teams
- **Schema Evolution**: Version-controlled database migrations
- **CI/CD Integration**: Automated migration deployment
- **Development Setup**: Quick database setup and seeding
- **Certificate Issues**: Debug connectivity problems in corporate environments

### Enterprise Teams
- **Compliance Reporting**: Professional reports in Google Docs
- **Knowledge Sharing**: Automated documentation and recommendations
- **Multi-Environment**: Support for dev, staging, production databases
- **Security**: SSL/TLS certificate management and troubleshooting

## 🏗️ Architecture

```
mongo-migration/
├── cmd/                    # CLI commands
│   ├── root.go            # Root command and global flags
│   ├── ai.go              # AI analysis commands
│   ├── cert.go            # Certificate utilities
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
git clone https://github.com/drewjocham/mongo-migration.git
cd mongo-migration

# Install dependencies
go mod tidy

# Build the binary
go build -o mongo-migration .

# Run tests
go test ./...

# Run linter
golangci-lint run
```

### Adding New Features

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Add tests and documentation
5. Submit a pull request

## 📝 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- [Cobra](https://github.com/spf13/cobra) for CLI framework
- [MongoDB Go Driver](https://github.com/mongodb/mongo-go-driver) for database connectivity
- [OpenAI](https://openai.com/), [Google](https://ai.google.dev/), [Anthropic](https://www.anthropic.com/) for AI capabilities
- [Google APIs](https://developers.google.com/docs/api) for Docs integration

## 🔗 Links & Resources

### Project Resources
- **[Go Package Documentation](https://pkg.go.dev/github.com/drewjocham/mongo-migration)** - Complete API reference
- **[GitHub Repository](https://github.com/drewjocham/mongo-migration)** - Source code and releases
- **[Issue Tracker](https://github.com/drewjocham/mongo-migration/issues)** - Bug reports and feature requests
- **[Homebrew Formula](https://github.com/drewjocham/homebrew-mongo-migration)** - Homebrew tap repository
- **[Docker Images](https://ghcr.io/drewjocham/mongo-migration)** - Container registry

### Documentation
- **[Installation Guide](INSTALL.md)** - All installation methods and troubleshooting
- **[Library Documentation](LIBRARY.md)** - Go library usage and examples
- **[MCP Integration](MCP.md)** - AI assistant integration guide
- **[AI Analysis Guide](AI_ANALYSIS.md)** - Database analysis documentation
- **[Contributing Guide](CONTRIBUTING.md)** - Development setup and guidelines

## 🐛 Support & Community

- 🐛 **Issues**: [GitHub Issues](https://github.com/drewjocham/mongo-migration/issues)
- 💬 **Discussions**: [GitHub Discussions](https://github.com/drewjocham/mongo-migration/discussions)
- 📧 **Contact**: [Project Maintainer](https://github.com/drewjocham)
- 📖 **Examples**: See the `examples/` directory in the repository

---

**Made with ❤️ for the MongoDB community**
