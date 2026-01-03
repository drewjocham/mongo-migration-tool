package mcp

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/drewjocham/mongo-migration-tool/config"
	"github.com/drewjocham/mongo-migration-tool/migration"
)

type MCPServer struct {
	mu        sync.Mutex
	mcpServer *mcp.Server
	engine    *migration.Engine
	db        *mongo.Database
	client    *mongo.Client
	config    *config.Config
}

// NewMCPServer initializes the SDK server and our migration logic.
func NewMCPServer() (*MCPServer, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}
	serverImpl := &mcp.Implementation{
		Name:    "mongo-migration",
		Version: "1.0.0",
	}
	s := mcp.NewServer(serverImpl, nil)

	mcpSrv := &MCPServer{
		mcpServer: s,
		config:    cfg,
	}

	mcpSrv.registerTools()

	return mcpSrv, nil
}

// registerTools defines the tool schemas and connects them to handlers.
func (s *MCPServer) registerTools() {
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "migration_status",
		Description: "Get a list of all migrations and whether they have been applied to the database.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, s.handleStatus)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "migration_up",
		Description: "Apply pending migrations. If a version is provided, it migrates up to that version.",
	}, s.handleUp)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "migration_down",
		Description: "Roll back applied migrations. If a version is provided, it rolls back to that version.",
	}, s.handleDown)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "migration_create",
		Description: "Generate a new Go migration file template.",
	}, s.handleCreate)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "database_schema",
		Description: "Get the schema of the database, including collections and their indexes.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, s.handleSchema)
}

// --- Handlers ---
func (s *MCPServer) handleStatus(
	ctx context.Context,
	req *mcp.CallToolRequest,
	_ emptyArgs,
) (*mcp.CallToolResult, any, error) {
	if err := s.ensureConnection(ctx); err != nil {
		return toolErrorResult(fmt.Sprintf("Database error: %v", err)), nil, nil
	}

	status, err := s.engine.GetStatus(ctx)
	if err != nil {
		return toolErrorResult(err.Error()), nil, nil
	}

	var b strings.Builder
	b.WriteString("### Migration Status\n\n")
	b.WriteString("| Version | Status | Applied At | Description |\n")
	b.WriteString("| :--- | :--- | :--- | :--- |\n")
	for _, st := range status {
		applied, at := "⏳ Pending", "N/A"
		if st.Applied {
			applied = "✅ Applied"
			if st.AppliedAt != nil {
				at = st.AppliedAt.Format("2006-01-02 15:04")
			}
		}
		b.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n", st.Version, applied, at, st.Description))
	}

	return toolTextResult(b.String()), nil, nil
}

func (s *MCPServer) handleUp(
	ctx context.Context,
	req *mcp.CallToolRequest,
	args versionArgs,
) (*mcp.CallToolResult, any, error) {
	if err := s.ensureConnection(ctx); err != nil {
		return toolErrorResult(err.Error()), nil, nil
	}

	if err := s.engine.Up(ctx, args.Version); err != nil {
		return toolErrorResult(err.Error()), nil, nil
	}

	return toolTextResult("✅ Migration 'Up' operation completed successfully."), nil, nil
}

func (s *MCPServer) handleDown(
	ctx context.Context,
	req *mcp.CallToolRequest,
	args versionArgs,
) (*mcp.CallToolResult, any, error) {
	if err := s.ensureConnection(ctx); err != nil {
		return toolErrorResult(err.Error()), nil, nil
	}

	if err := s.engine.Down(ctx, args.Version); err != nil {
		return toolErrorResult(err.Error()), nil, nil
	}

	return toolTextResult("✅ Migration 'Down' operation completed successfully."), nil, nil
}

func (s *MCPServer) handleCreate(
	ctx context.Context,
	req *mcp.CallToolRequest,
	args createMigrationArgs,
) (*mcp.CallToolResult, any, error) {

	version := time.Now().Format("20060102_150405")
	cleanName := strings.ToLower(strings.ReplaceAll(args.Name, " ", "_"))
	fname := fmt.Sprintf("migrations/%s_%s.go", version, cleanName)

	if err := os.MkdirAll("migrations", 0750); err != nil {
		return toolErrorResult(fmt.Sprintf("failed to create directory: %v", err)), nil, nil
	}

	tmpl, err := template.New("migration").Parse(migrationTemplate)
	if err != nil {
		return toolErrorResult(err.Error()), nil, nil
	}

	var buf bytes.Buffer
	data := struct {
		StructName, Version, Description string
	}{
		StructName:  toCamelCase(cleanName),
		Version:     version,
		Description: args.Description,
	}

	if err := tmpl.Execute(&buf, data); err != nil {
		return toolErrorResult(err.Error()), nil, nil
	}

	if err := os.WriteFile(fname, buf.Bytes(), 0600); err != nil {
		return toolErrorResult(err.Error()), nil, nil
	}

	return toolTextResult(fmt.Sprintf("🚀 Created new migration file: %s", fname)), nil, nil
}

func (s *MCPServer) handleSchema(
	ctx context.Context,
	req *mcp.CallToolRequest,
	_ emptyArgs,
) (*mcp.CallToolResult, any, error) {
	if err := s.ensureConnection(ctx); err != nil {
		return toolErrorResult(fmt.Sprintf("Database error: %v", err)), nil, nil
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("### Database Schema for '%s'\n\n", s.db.Name()))

	collections, err := s.db.ListCollectionNames(ctx, bson.D{})
	if err != nil {
		return toolErrorResult(fmt.Sprintf("Failed to list collections: %v", err)), nil, nil
	}

	for _, collName := range collections {
		b.WriteString(fmt.Sprintf("#### Collection: `%s`\n\n", collName))
		b.WriteString("| Index Name | Keys | Unique |\n")
		b.WriteString("| :--- | :--- | :--- |\n")

		coll := s.db.Collection(collName)
		cursor, err := coll.Indexes().List(ctx)
		if err != nil {
			b.WriteString(fmt.Sprintf("| *Error fetching indexes: %v* | | |\n", err))
			continue
		}

		var indexes []bson.M
		if err = cursor.All(ctx, &indexes); err != nil {
			b.WriteString(fmt.Sprintf("| *Error decoding indexes: %v* | | |\n", err))
			continue
		}

		for _, index := range indexes {
			name, ok := index["name"].(string)
			if !ok {
				name = "[unknown]"
			}
			keys := formatIndexKeys(index["key"])
			unique := "No"
			if u, ok := index["unique"].(bool); ok && u {
				unique = "Yes"
			}
			b.WriteString(fmt.Sprintf("| `%s` | `%s` | %s |\n", name, keys, unique))
		}
		b.WriteString("\n")
	}

	return toolTextResult(b.String()), nil, nil
}

func formatIndexKeys(keys interface{}) string {
	var keyParts []string
	if doc, ok := keys.(bson.D); ok {
		for _, elem := range doc {
			keyParts = append(keyParts, fmt.Sprintf("%s: %v", elem.Key, elem.Value))
		}
	}
	return strings.Join(keyParts, ", ")
}

// --- Lifecycle & Helpers ---

func (s *MCPServer) Start() error {
	return s.mcpServer.Run(context.Background(), &mcp.StdioTransport{})
}

// Serve runs the MCP server against arbitrary io streams.
func (s *MCPServer) Serve(ctx context.Context, reader io.Reader, writer io.Writer) error {
	var rc io.ReadCloser
	if r, ok := reader.(io.ReadCloser); ok {
		rc = r
	} else {
		rc = io.NopCloser(reader)
	}

	var wc io.WriteCloser
	if w, ok := writer.(io.WriteCloser); ok {
		wc = w
	} else {
		wc = nopWriteCloser{Writer: writer}
	}

	return s.mcpServer.Run(ctx, &mcp.IOTransport{
		Reader: rc,
		Writer: wc,
	})
}

func (s *MCPServer) ensureConnection(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.client != nil {
		if err := s.client.Ping(ctx, nil); err == nil {
			return nil
		}
	}

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(s.config.GetConnectionString()))
	if err != nil {
		return err
	}

	s.client = client
	s.db = client.Database(s.config.Database)
	s.engine = migration.NewEngine(s.db, s.config.MigrationsCollection, migration.RegisteredMigrations())
	return nil
}

func (s *MCPServer) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.client != nil {
		return s.client.Disconnect(context.Background())
	}
	return nil
}

func toolErrorResult(message string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: message}},
	}
}

func toolTextResult(message string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: message}},
	}
}

type nopWriteCloser struct {
	io.Writer
}

func (nopWriteCloser) Close() error { return nil }

type emptyArgs struct{}

type versionArgs struct {
	Version string `json:"version,omitempty" jsonschema:"Version identifier such as 20240101_001"`
}

type createMigrationArgs struct {
	Name        string `json:"name" jsonschema:"Migration name (e.g., add_users_collection)"`
	Description string `json:"description" jsonschema:"Brief summary of what the migration does"`
}

func toCamelCase(s string) string {
	parts := strings.Split(s, "_")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, "")
}

const migrationTemplate = `package migrations

import (
	"context"
	"go.mongodb.org/mongo-driver/mongo"
)

// {{.StructName}} Migration: {{.Description}}
type {{.StructName}} struct{}

func (m *{{.StructName}}) Version() string     { return "{{.Version}}" }
func (m *{{.StructName}}) Description() string { return "{{.Description}}" }

func (m *{{.StructName}}) Up(ctx context.Context, db *mongo.Database) error {
	// Implement migration logic here
	return nil
}

func (m *{{.StructName}}) Down(ctx context.Context, db *mongo.Database) error {
	// Implement rollback logic here
	return nil
}
`
