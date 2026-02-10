/*
Package main documents the mongo-tool CLI that sits on top of the migration engine.

Commands such as mongo-tool status, up, down, oplog --follow --resume-file, and mcp
are the recommended ways to inspect migrations, tail change streams, and expose an MCP
bridge for AI agents. The resume-token support ensures long-running observers can reconnect
without losing oplog events even when the driver loses connectivity.
*/
package main
