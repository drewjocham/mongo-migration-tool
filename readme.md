# mongo-migration-tool

[![Go Reference](https://pkg.go.dev/badge/github.com/drewjocham/mongo-migration-tool.svg)](https://pkg.go.dev/github.com/drewjocham/mongo-migration-tool)
[![Go Report Card](https://goreportcard.com/badge/github.com/drewjocham/mongo-migration-tool)](https://goreportcard.com/report/github.com/drewjocham/mongo-migration-tool)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A lightweight migration engine, CLI toolbox, and MCP bridge for MongoDB with resume-capable oplog tails and models for scripted data movement.

## Highlights
- **Migration Engine** with locking, checksum validation, and smooth `Up`/`Down` workflows.
- **CLI tooling** that also exposes the oplog, schema introspection, MCP, and resume-aware change-stream tails.
- **MCP integration** for AI assistants, plus example migrations and connectors for CI/CD.
- **MongoDB Go v2** migration and CLI code so the toolkit works with the current official driver API.

## Installation
1. **Homebrew (macOS/Linux)** – `brew tap drewjocham/mongo-migration-tool` then `brew install mongo-migration-tool`.
2. **Docker** – `docker pull ghcr.io/drewjocham/mongo-migration-tool:latest` and run `docker run --rm -v "$(pwd)":/workspace ghcr.io/drewjocham/mongo-migration-tool:latest --help`.
3. **Go tooling (development)** – `go install github.com/drewjocham/mongo-migration-tool@latest`.

For full deployment details, see [install.md](install.md).

## Quick Start
1. Copy `.env.example` to `.env` and set `MONGO_URL`, credentials, and other overrides.
2. Run `mmt status` to inspect migrations, `mmt up` to apply pending work, and `mmt down --target <version>` when you need to roll back.
3. Tail ongoing work with the oplog: `mmt oplog --follow --resume-file /tmp/oplog.token`.
4. Launch the MCP endpoint with `mmt mcp` (add `--with-examples` to register sample migrations).

## Key CLI Commands
| Command | Purpose |
| --- | --- |
| `mmt up` | Apply pending migrations (use `--dry-run` to preview). |
| `mmt down` | Roll back migrations to a given version. |
| `mmt status` | Show migration state and timestamps. |
| `mmt create <name>` | Scaffold a new migration stub. |
| `mmt oplog` | Query and tail oplog/change stream events (resume-file supported). |
| `mmt schema indexes` | Print the schema indexes registered in Go code. |
| `mmt mcp` | Start the Model Context Protocol server; use `--with-examples` to seed sample work. |

## Documentation
| Guide | Description |
| --- | --- |
| [install.md](install.md) | Installation paths, Docker workflow, and troubleshooting tips. |
| [library.md](library.md) | Using the migration engine as a Go library. |
| [mcp.md](mcp.md) | MCP setup for AI assistants. |
| [contributing.md](contributing.md) | Development workflow, testing, and CI expectations. |

## Documents
The GoDoc page at [pkg.go.dev/github.com/drewjocham/mongo-migration-tool](https://pkg.go.dev/github.com/drewjocham/mongo-migration-tool) now reflects the v2 driver rewrite, but the *Documents* section still needs cleanup and additional narrative. Contributions to flesh out that section (and keep the generated examples aligned with reality) are welcome.

## Support & Community
- Issues/feature requests: [github.com/drewjocham/mongo-migration-tool/issues](https://github.com/drewjocham/mongo-migration-tool/issues)
- Discussions and RFCs: [github.com/drewjocham/mongo-migration-tool/discussions](https://github.com/drewjocham/mongo-migration-tool/discussions)
- Example migrations live under `internal/migrations/` and `examples/`.

## License
MIT. See [LICENSE](LICENSE).
