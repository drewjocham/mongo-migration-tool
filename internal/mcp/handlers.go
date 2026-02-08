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

func (s *MCPServer) registerTools() {
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "migration_status",
		Description: "Check applied and pending migrations.",
	}, s.handleStatus)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "migration_up",
		Description: "Apply pending migrations.",
	}, s.handleUp)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "migration_down",
		Description: "Roll back migrations.",
	}, s.handleDown)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "migration_create",
		Description: "Generate a new migration file.",
	}, s.handleCreate)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "database_schema",
		Description: "View collections and indexes.",
	}, s.handleSchema)
}

func (s *MCPServer) handleStatus(
	ctx context.Context, _ *mcp.CallToolRequest, _ emptyArgs,
) (*mcp.CallToolResult, string, error) {
	if err := s.ensureConnection(ctx); err != nil {
		return nil, "", err
	}
	status, err := s.engine.GetStatus(ctx)
	if err != nil {
		return nil, "", err
	}
	return nil, formatStatusTable(status), nil
}

func (s *MCPServer) handleUp(
	ctx context.Context, _ *mcp.CallToolRequest, args versionArgs,
) (*mcp.CallToolResult, string, error) {
	if err := s.ensureConnection(ctx); err != nil {
		return nil, "", err
	}
	if err := s.engine.Up(ctx, args.Version); err != nil {
		return nil, "", fmt.Errorf("migration up failed: %w", err)
	}
	return nil, "✅ Migrations applied successfully.", nil
}

func (s *MCPServer) handleDown(
	ctx context.Context, _ *mcp.CallToolRequest, args versionArgs,
) (*mcp.CallToolResult, string, error) {
	if err := s.ensureConnection(ctx); err != nil {
		return nil, "", err
	}
	if err := s.engine.Down(ctx, args.Version); err != nil {
		return nil, "", fmt.Errorf("migration down failed: %w", err)
	}
	return nil, "✅ Rollback completed successfully.", nil
}

func (s *MCPServer) handleSchema(
	ctx context.Context, _ *mcp.CallToolRequest, _ emptyArgs,
) (*mcp.CallToolResult, string, error) {
	if err := s.ensureConnection(ctx); err != nil {
		return nil, "", err
	}
	collections, err := s.db.ListCollectionNames(ctx, bson.D{})
	if err != nil {
		return nil, "", err
	}

	var b strings.Builder
	fmt.Fprintf(&b, "### Database Schema: `%s`\n\n", s.db.Name())
	for _, name := range collections {
		s.appendCollectionSchema(&b, ctx, name)
	}
	return nil, b.String(), nil
}

func (s *MCPServer) handleCreate(
	ctx context.Context, _ *mcp.CallToolRequest, args createMigrationArgs,
) (*mcp.CallToolResult, string, error) {
	version := time.Now().Format("20060102_150405")
	slug := strings.ToLower(strings.ReplaceAll(args.Name, " ", "_"))
	path := filepath.Join("migrations", fmt.Sprintf("%s_%s.go", version, slug))

	if err := os.MkdirAll("migrations", 0750); err != nil {
		return nil, "", err
	}

	var buf bytes.Buffer
	data := migrationData{
		StructName:  toCamelCase(slug),
		Version:     version,
		Description: args.Description,
	}

	if err := migrationTemplate.Execute(&buf, data); err != nil {
		return nil, "", err
	}

	if err := os.WriteFile(path, buf.Bytes(), 0600); err != nil {
		return nil, "", err
	}

	return nil, fmt.Sprintf("🚀 Created migration: `%s`", path), nil
}

func (s *MCPServer) appendCollectionSchema(b *strings.Builder, ctx context.Context, name string) {
	fmt.Fprintf(b, "#### Collection: `%s`\n\n| Index Name | Keys | Unique |\n| :--- | :--- | :--- |\n", name)

	cursor, err := s.db.Collection(name).Indexes().List(ctx)
	if err != nil {
		fmt.Fprintf(b, "| *Error: %v* | | |\n\n", err)
		return
	}
	defer cursor.Close(ctx)

	var idxs []bson.M
	if err := cursor.All(ctx, &idxs); err != nil {
		return
	}

	for _, idx := range idxs {
		unique := "No"
		if u, ok := idx["unique"].(bool); ok && u {
			unique = "Yes"
		}
		fmt.Fprintf(b, "| `%v` | `%s` | %s |\n", idx["name"], formatIndexKeys(idx["key"]), unique)
	}
	b.WriteString("\n")
}
