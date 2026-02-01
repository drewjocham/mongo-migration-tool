package mcp

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.mongodb.org/mongo-driver/bson"
)

func (s *MCPServer) withConnection(
	ctx context.Context,
	fn func() (*mcp.CallToolResult, any, error),
) (*mcp.CallToolResult, any, error) {
	if err := s.ensureConnection(ctx); err != nil {
		return toolErrorResult("Database Connection Error", err), nil, nil
	}
	return fn()
}

func (s *MCPServer) handleStatus(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	_ emptyArgs,
) (*mcp.CallToolResult, any, error) {
	return s.withConnection(ctx, func() (*mcp.CallToolResult, any, error) {
		status, err := s.engine.GetStatus(ctx)
		if err != nil {
			return toolErrorResult("Failed to retrieve migration status", err), nil, nil
		}
		return toolTextResult(formatStatusTable(status)), nil, nil
	})
}

func (s *MCPServer) handleUp(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	args versionArgs,
) (*mcp.CallToolResult, any, error) {
	return s.withConnection(ctx, func() (*mcp.CallToolResult, any, error) {
		if err := s.engine.Up(ctx, args.Version); err != nil {
			return toolErrorResult("Migration 'Up' failed", err), nil, nil
		}
		return toolTextResult("✅ Migrations applied successfully."), nil, nil
	})
}

func (s *MCPServer) handleDown(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	args versionArgs,
) (*mcp.CallToolResult, any, error) {
	return s.withConnection(ctx, func() (*mcp.CallToolResult, any, error) {
		if err := s.engine.Down(ctx, args.Version); err != nil {
			return toolErrorResult("Migration 'Down' failed", err), nil, nil
		}
		return toolTextResult("✅ Rollback completed successfully."), nil, nil
	})
}

func (s *MCPServer) handleSchema(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	_ emptyArgs,
) (*mcp.CallToolResult, any, error) {
	return s.withConnection(ctx, func() (*mcp.CallToolResult, any, error) {
		collections, err := s.db.ListCollectionNames(ctx, bson.D{})
		if err != nil {
			return toolErrorResult("Failed to list collections", err), nil, nil
		}

		var b strings.Builder
		fmt.Fprintf(&b, "### Database Schema: `%s`\n\n", s.db.Name())

		for _, name := range collections {
			s.appendCollectionSchema(&b, ctx, name)
		}
		return toolTextResult(b.String()), nil, nil
	})
}

func (s *MCPServer) appendCollectionSchema(
	b *strings.Builder,
	ctx context.Context,
	name string,
) {
	fmt.Fprintf(b, "#### Collection: `%s`\n\n", name)
	b.WriteString("| Index Name | Keys | Unique |\n| :--- | :--- | :--- |\n")

	cursor, err := s.db.Collection(name).Indexes().List(ctx)
	if err != nil {
		fmt.Fprintf(b, "| *Error listing indexes: %v* | | |\n\n", err)
		return
	}
	defer cursor.Close(ctx)

	var idxs []bson.M
	if err := cursor.All(ctx, &idxs); err != nil {
		fmt.Fprintf(b, "| *Error decoding indexes: %v* | | |\n\n", err)
		return
	}

	for _, idx := range idxs {
		unique := "No"
		if u, ok := idx["unique"].(bool); ok && u {
			unique = "Yes"
		}
		fmt.Fprintf(
			b,
			"| `%v` | `%s` | %s |\n",
			idx["name"],
			formatIndexKeys(idx["key"]),
			unique,
		)
	}
	b.WriteString("\n")
}

func (s *MCPServer) handleCreate(
	ctx context.Context,
	_ *mcp.CallToolRequest,
	args createMigrationArgs,
) (*mcp.CallToolResult, any, error) {
	version := time.Now().Format("20060102_150405")
	cleanName := strings.ToLower(strings.ReplaceAll(args.Name, " ", "_"))

	// Use filepath.Join for OS-agnostic path handling
	dir := "migrations"
	fname := filepath.Join(dir, fmt.Sprintf("%s_%s.go", version, cleanName))

	if err := os.MkdirAll(dir, 0750); err != nil {
		return toolErrorResult("Could not create migrations directory", err), nil, nil
	}

	var buf bytes.Buffer
	data := migrationData{
		StructName:  toCamelCase(cleanName),
		Version:     version,
		Description: args.Description,
	}

	if err := migrationTemplate.Execute(&buf, data); err != nil {
		return toolErrorResult("Failed to generate migration template", err), nil, nil
	}

	if err := os.WriteFile(fname, buf.Bytes(), 0600); err != nil {
		return toolErrorResult("Failed to write migration file", err), nil, nil
	}

	return toolTextResult(fmt.Sprintf("🚀 Created migration: `%s`", fname)), nil, nil
}
