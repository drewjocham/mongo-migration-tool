package mcp

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/drewjocham/mongo-migration-tool/internal/jsonutil"
	"github.com/drewjocham/mongo-migration-tool/internal/parser"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.mongodb.org/mongo-driver/v2/bson"
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

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "parse_payload",
		Description: "Parse JSON or BSON payload into normalized JSON.",
	}, s.handleParsePayload)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "validate_payload",
		Description: "Parse and validate payload using registered types.",
	}, s.handleValidatePayload)
}

func newMessageResult(text string) (*mcp.CallToolResult, messageOutput) {
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: text},
		},
	}, messageOutput{Message: text}
}

func (s *MCPServer) handleStatus(
	ctx context.Context, _ *mcp.CallToolRequest, _ emptyArgs,
) (*mcp.CallToolResult, messageOutput, error) {
	if err := s.ensureConnection(ctx); err != nil {
		return nil, messageOutput{}, err
	}
	status, err := s.engine.GetStatus(ctx)
	if err != nil {
		return nil, messageOutput{}, err
	}
	res, out := newMessageResult(formatStatusTable(status))
	return res, out, nil
}

func (s *MCPServer) handleUp(
	ctx context.Context, _ *mcp.CallToolRequest, args versionArgs,
) (*mcp.CallToolResult, messageOutput, error) {
	if err := s.ensureConnection(ctx); err != nil {
		return nil, messageOutput{}, err
	}
	if err := s.engine.Up(ctx, args.Version); err != nil {
		return nil, messageOutput{}, fmt.Errorf("migration up failed: %w", err)
	}
	res, out := newMessageResult("✅ Migrations applied successfully.")
	return res, out, nil
}

func (s *MCPServer) handleDown(
	ctx context.Context, _ *mcp.CallToolRequest, args versionArgs,
) (*mcp.CallToolResult, messageOutput, error) {
	if err := s.ensureConnection(ctx); err != nil {
		return nil, messageOutput{}, err
	}
	if err := s.engine.Down(ctx, args.Version); err != nil {
		return nil, messageOutput{}, fmt.Errorf("migration down failed: %w", err)
	}
	res, out := newMessageResult("✅ Rollback completed successfully.")
	return res, out, nil
}

func (s *MCPServer) handleSchema(
	ctx context.Context, _ *mcp.CallToolRequest, _ emptyArgs,
) (*mcp.CallToolResult, messageOutput, error) {
	if err := s.ensureConnection(ctx); err != nil {
		return nil, messageOutput{}, err
	}
	collections, err := s.db.ListCollectionNames(ctx, bson.D{})
	if err != nil {
		return nil, messageOutput{}, err
	}

	var b strings.Builder
	fmt.Fprintf(&b, "### Database Schema: `%s`\n\n", s.db.Name())
	for _, name := range collections {
		s.appendCollectionSchema(&b, ctx, name)
	}
	res, out := newMessageResult(b.String())
	return res, out, nil
}


func (s *MCPServer) handleCreate(
	ctx context.Context, _ *mcp.CallToolRequest, args createMigrationArgs,
) (*mcp.CallToolResult, messageOutput, error) {
	version := time.Now().Format("20060102_150405")
	slug := strings.ToLower(strings.ReplaceAll(args.Name, " ", "_"))
	path := filepath.Join("migrations", fmt.Sprintf("%s_%s.go", version, slug))

	if err := os.MkdirAll("migrations", 0750); err != nil {
		return nil, messageOutput{}, err
	}

	var buf bytes.Buffer
	data := migrationData{
		StructName:  toCamelCase(slug),
		Version:     version,
		Description: args.Description,
	}

	if err := migrationTemplate.Execute(&buf, data); err != nil {
		return nil, messageOutput{}, err
	}

	if err := os.WriteFile(path, buf.Bytes(), 0600); err != nil {
		return nil, messageOutput{}, err
	}

	res, out := newMessageResult(fmt.Sprintf("🚀 Created migration: `%s`", path))
	return res, out, nil
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

func (s *MCPServer) handleParsePayload(
	ctx context.Context, _ *mcp.CallToolRequest, args parsePayloadArgs,
) (*mcp.CallToolResult, messageOutput, error) {
	format := strings.ToLower(args.Format)
	if format == "" {
		format = "json"
	}

	raw, err := parser.DecodePayload(args.Payload, parser.Format(format))
	if err != nil {
		return nil, messageOutput{}, err
	}

	parsed, err := parser.ParseMap(raw, parser.WithFormat(parser.Format(format)))
	if err != nil {
		return nil, messageOutput{}, err
	}

	outBytes, err := jsonutil.MarshalIndent(parsed, "", "  ")
	if err != nil {
		return nil, messageOutput{}, err
	}

	res, out := newMessageResult(string(outBytes))
	return res, out, nil
}

func (s *MCPServer) handleValidatePayload(
	ctx context.Context, _ *mcp.CallToolRequest, args parsePayloadArgs,
) (*mcp.CallToolResult, messageOutput, error) {
	format := strings.ToLower(args.Format)
	if format == "" {
		format = "json"
	}

	raw, err := parser.DecodePayload(args.Payload, parser.Format(format))
	if err != nil {
		return nil, messageOutput{}, err
	}

	if args.TypeName == "" && args.TypeField == "" {
		return nil, messageOutput{}, fmt.Errorf("type or typeField is required")
	}

	if args.TypeName != "" {
		ctor := parser.DefaultRegistry[strings.ToLower(args.TypeName)]
		if ctor == nil {
			return nil, messageOutput{}, fmt.Errorf("no registered type: %s", args.TypeName)
		}
		instance := ctor()
		if err := parser.ParseInto(raw, instance,
			parser.WithFormat(parser.Format(format)),
			parser.WithValidation(true),
		); err != nil {
			return nil, messageOutput{}, err
		}
		res, out := newMessageResult("valid")
		return res, out, nil
	}

	if _, err := parser.ParseByType(raw, args.TypeField, parser.DefaultRegistry,
		parser.WithFormat(parser.Format(format)),
		parser.WithValidation(true),
	); err != nil {
		return nil, messageOutput{}, err
	}
	res, out := newMessageResult("valid")
	return res, out, nil
}
