# mongo-migration-tool

A lightweight MongoDB migration engine, CLI, and MCP-ready toolset that lets you keep migrations, change streams, and scripted operations in sync with your clusters. The binary is exposed as `mongo-tool` (aliases `mt` and `mmt`) so you can run your status, up/down, and oplog commands with a single executable.

## Highlights
- **Stateful Migration Engine:** Distributed locking, checksum validation, and smooth `Up`/`Down` flows keep the schema registry in sync across teams.
- **Mongo-tool CLI:** `mongo-tool` wraps the migration engine plus schema, oplog, and MCP helpers. The `oplog --follow --resume-file` flag saves resume tokens to disk so you never lose events during network hiccups.
- **MCP Bridge:** The CLI can run as an MCP server (`mongo-tool mcp --with-examples`) so AI agents can query migration status or apply work for you.

## Installation
1. **Homebrew (macOS/Linux)** – `brew tap drewjocham/mongo-migration-tool` then `brew install mongo-tool`.
2. **Docker** – `docker pull ghcr.io/drewjocham/mongo-migration-tool:latest` and run `docker run --rm -v "$(pwd)":/workspace ghcr.io/drewjocham/mongo-migration-tool:latest --help`.
3. **Go tooling (development)** – `go install github.com/drewjocham/mongo-migration-tool/cmd@latest`.

Need more setup help? See [install.md](install.md).

## Quick Start
1. Copy `.env.example` to `.env` and configure `MONGO_URL`, credentials, and overrides for `MONGO_DATABASE`/`MIGRATIONS_COLLECTION`.
2. Run `mongo-tool status`, `mongo-tool up`, or `mongo-tool down --target <version>` to inspect and evolve your schema.
3. Tail migrations with `mongo-tool oplog --follow --resume-file /tmp/oplog.token`. The CLI keeps the last seen resume token on disk so reconnects never skip an oplog gap.
4. Start the MCP endpoint with `mongo-tool mcp` (add `--with-examples` to seed the sample migrations).

## CLI Overview
| Command | Purpose |
| --- | --- |
| `mongo-tool status` | Show migration state and timestamps. |
| `mongo-tool up` | Apply pending migrations (use `--dry-run` to preview). |
| `mongo-tool down` | Roll back migrations (`--target` limits how far). |
| `mongo-tool create <name>` | Scaffold a new migration stub. |
| `mongo-tool oplog` | Query and tail change stream events (use `--resume-file` to persist tokens). |
| `mongo-tool schema indexes` | Print the schema indexes registered in Go. |
| `mongo-tool mcp` | Start the Model Context Protocol server. |

## Architectural Toolbox
- **The Engine** manages distributed locks, applies migrations via registered `migration.Migration` implementations, and tracks versions in Mongo's migrations collection.
- **The Processor** in `cmd/examples` and `internal/mcp` shows how to batch scripted work such as `ReassignAssets`.
- **The CLI** exposes those capabilities, resumes oplog tails with disk-backed tokens, and serves an MCP endpoint for AI tooling.

## Documents
- **Top-level doc** – [`doc.go`](doc.go) contains the narrative that feeds the Documents section on [pkg.go.dev](https://pkg.go.dev/github.com/drewjocham/mongo-migration-tool). Keep it updated whenever you add concepts so the generated doc section stays accurate.
- **Library Guide** – [`library.md`](library.md) shows how to embed the migration engine in another Go project.
- **MCP Guide** – [`mcp.md`](mcp.md) explains how to wire the MCP server and register AI tools.
- **Contributing & tests** – [`contributing.md`](contributing.md) documents workflows, release steps, and how to run `make lint`, `make test`, and `make integration-test`.
- **Installation** – [`install.md`](install.md) details platform-specific installs, Docker compose setups, and every supported entry point.

## Support & Community
- Issues/feature requests: [github.com/drewjocham/mongo-migration-tool/issues](https://github.com/drewjocham/mongo-migration-tool/issues)
- RFCs & discussions: [github.com/drewjocham/mongo-migration-tool/discussions](https://github.com/drewjocham/mongo-migration-tool/discussions)
- Example migrations: `internal/migrations/`

## License
MIT. See [LICENSE](LICENSE).
