package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/jocham/mongo-essential/config"
	"github.com/jocham/mongo-essential/migration"
)

// MCPServer implements the Model Context Protocol for MongoDB migrations
type MCPServer struct { //nolint:revive // MCPServer is clearer than Server in this context
	engine *migration.Engine
	db     *mongo.Database
	client *mongo.Client
	config *config.Config
}

// MCPRequest represents an incoming MCP request
type MCPRequest struct { //nolint:revive // MCPRequest is clearer than Request in this context
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// MCPResponse represents an MCP response
type MCPResponse struct { //nolint:revive // MCPResponse is clearer than Response in this context
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *MCPError   `json:"error,omitempty"`
}

// MCPError represents an MCP error
type MCPError struct { //nolint:revive // MCPError is clearer than Error in this context
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data,omitempty"`
}

// ToolListParams represents parameters for tools/list
type ToolListParams struct{}

// ToolCallParams represents parameters for tools/call
type ToolCallParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// Tool represents an MCP tool definition
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// NewMCPServer creates a new MCP server instance
func NewMCPServer() (*MCPServer, error) {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Connect to MongoDB
	const connectionTimeout = 10
	ctx, cancel := context.WithTimeout(context.Background(), connectionTimeout*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.GetConnectionString()))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Test the connection
	if err := client.Ping(ctx, nil); err != nil {
		if disconnectErr := client.Disconnect(ctx); disconnectErr != nil {
			log.Printf("Warning: failed to disconnect client: %v", disconnectErr)
		}
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	db := client.Database(cfg.Database)
	engine := migration.NewEngine(db, cfg.MigrationsCollection)

	return &MCPServer{
		engine: engine,
		db:     db,
		client: client,
		config: cfg,
	}, nil
}

// Close closes the MCP server and database connections
func (s *MCPServer) Close() error {
	if s.client != nil {
		const closeTimeout = 5
		ctx, cancel := context.WithTimeout(context.Background(), closeTimeout*time.Second)
		defer cancel()
		return s.client.Disconnect(ctx)
	}
	return nil
}

// RegisterMigration registers a migration with the engine
func (s *MCPServer) RegisterMigration(m migration.Migration) {
	s.engine.Register(m)
}

// RegisterMigrations registers multiple migrations with the engine
func (s *MCPServer) RegisterMigrations(migrations ...migration.Migration) {
	s.engine.RegisterMany(migrations...)
}

// Start starts the MCP server
func (s *MCPServer) Start() error {
	decoder := json.NewDecoder(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)

	for {
		var request MCPRequest
		if err := decoder.Decode(&request); err != nil {
			if err == io.EOF {
				break
			}
			log.Printf("Error decoding request: %v", err)
			continue
		}

		response := s.handleRequest(&request)
		if err := encoder.Encode(response); err != nil {
			log.Printf("Error encoding response: %v", err)
		}
	}

	return nil
}

// HandleRequest handles an MCP request (exported for testing)
func (s *MCPServer) HandleRequest(request *MCPRequest) *MCPResponse {
	return s.handleRequest(request)
}

// handleRequest handles an MCP request
func (s *MCPServer) handleRequest(request *MCPRequest) *MCPResponse {
	switch request.Method {
	case "initialize":
		return s.handleInitialize(request)
	case "tools/list":
		return s.handleToolsList(request)
	case "tools/call":
		return s.handleToolsCall(request)
	default:
		return &MCPResponse{
			JSONRPC: "2.0",
			ID:      request.ID,
			Error: &MCPError{
				Code:    -32601,
				Message: "Method not found",
				Data:    fmt.Sprintf("Unknown method: %s", request.Method),
			},
		}
	}
}

// handleInitialize handles the initialize request
func (s *MCPServer) handleInitialize(request *MCPRequest) *MCPResponse {
	result := map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]interface{}{
			"tools": map[string]interface{}{},
		},
		"serverInfo": map[string]interface{}{
			"name":    "mongo-essential",
			"version": "1.0.0",
		},
	}

	return &MCPResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
		Result:  result,
	}
}

// handleToolsList handles the tools/list request
func (s *MCPServer) handleToolsList(request *MCPRequest) *MCPResponse {
	tools := s.getAvailableTools()

	return &MCPResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
		Result: map[string]interface{}{
			"tools": tools,
		},
	}
}

// getAvailableTools returns all available MCP tools
func (s *MCPServer) getAvailableTools() []Tool {
	return []Tool{
		s.createMigrationStatusTool(),
		s.createMigrationUpTool(),
		s.createMigrationDownTool(),
		s.createMigrationCreateTool(),
		s.createMigrationListTool(),
	}
}

// createMigrationStatusTool creates the migration_status tool definition
func (s *MCPServer) createMigrationStatusTool() Tool {
	return Tool{
		Name:        "migration_status",
		Description: "Get the status of all migrations - shows which migrations are applied and when",
		InputSchema: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
	}
}

// createMigrationUpTool creates the migration_up tool definition
func (s *MCPServer) createMigrationUpTool() Tool {
	return Tool{
		Name:        "migration_up",
		Description: "Apply migrations up to a specific version or all pending migrations",
		InputSchema: s.createVersionInputSchema(
			"Migration version to migrate up to (optional - if not provided, applies all pending migrations)"),
	}
}

// createMigrationDownTool creates the migration_down tool definition
func (s *MCPServer) createMigrationDownTool() Tool {
	return Tool{
		Name:        "migration_down",
		Description: "Roll back migrations down to a specific version or roll back the last migration",
		InputSchema: s.createVersionInputSchema(
			"Migration version to roll back to (optional - " +
				"if not provided, rolls back the last applied migration)"),
	}
}

// createMigrationCreateTool creates the migration_create tool definition
func (s *MCPServer) createMigrationCreateTool() Tool {
	return Tool{
		Name:        "migration_create",
		Description: "Create a new migration file with a given name and description",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"name": map[string]interface{}{
					"type":        "string",
					"description": "Name for the migration (will be used to generate filename)",
					"required":    true,
				},
				"description": map[string]interface{}{
					"type":        "string",
					"description": "Description of what the migration does",
					"required":    true,
				},
			},
			"required": []string{"name", "description"},
		},
	}
}

// createMigrationListTool creates the migration_list tool definition
func (s *MCPServer) createMigrationListTool() Tool {
	return Tool{
		Name:        "migration_list",
		Description: "List all registered migrations with their versions and descriptions",
		InputSchema: map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		},
	}
}

// createVersionInputSchema creates a common input schema for version parameter
func (s *MCPServer) createVersionInputSchema(description string) map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"version": map[string]interface{}{
				"type":        "string",
				"description": description,
			},
		},
	}
}

// handleToolsCall handles the tools/call request
func (s *MCPServer) handleToolsCall(request *MCPRequest) *MCPResponse {
	var params ToolCallParams
	if err := json.Unmarshal(request.Params, &params); err != nil {
		return s.createErrorResponse(request.ID, -32602, "Invalid params", err.Error())
	}

	const timeoutDuration = 30
	ctx, cancel := context.WithTimeout(context.Background(), timeoutDuration*time.Second)
	defer cancel()

	result, err := s.executeTool(ctx, params)
	if err != nil {
		return s.createToolExecutionErrorResponse(request.ID, err)
	}

	return s.createSuccessResponse(request.ID, result)
}

// executeTool executes the requested tool
func (s *MCPServer) executeTool(ctx context.Context, params ToolCallParams) (interface{}, error) {
	switch params.Name {
	case "migration_status":
		return s.getMigrationStatus(ctx)
	case "migration_up":
		version, _ := params.Arguments["version"].(string)
		return s.runMigrationUp(ctx, version)
	case "migration_down":
		version, _ := params.Arguments["version"].(string)
		return s.runMigrationDown(ctx, version)
	case "migration_create":
		name, _ := params.Arguments["name"].(string)
		description, _ := params.Arguments["description"].(string)
		return s.createMigration(name, description)
	case "migration_list":
		return s.listMigrations(ctx)
	default:
		return nil, fmt.Errorf("unknown tool: %s", params.Name)
	}
}

// createErrorResponse creates a standard error response
func (s *MCPServer) createErrorResponse(id interface{}, code int, message, data string) *MCPResponse {
	return &MCPResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &MCPError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
}

// createToolExecutionErrorResponse creates an error response for tool execution failures
func (s *MCPServer) createToolExecutionErrorResponse(id interface{}, err error) *MCPResponse {
	return s.createErrorResponse(id, -32603, "Tool execution error", err.Error())
}

// createSuccessResponse creates a successful response with the result
func (s *MCPServer) createSuccessResponse(id interface{}, result interface{}) *MCPResponse {
	return &MCPResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: map[string]interface{}{
			"content": []map[string]interface{}{
				{
					"type": "text",
					"text": result,
				},
			},
		},
	}
}

// getMigrationStatus gets the migration status
func (s *MCPServer) getMigrationStatus(ctx context.Context) (string, error) {
	status, err := s.engine.GetStatus(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get migration status: %w", err)
	}

	var result strings.Builder
	result.WriteString("Migration Status:\n")
	result.WriteString("================================================================================\n")
	result.WriteString(fmt.Sprintf("%-20s %-10s %-20s %s\n", "Version", "Applied", "Applied At", "Description"))
	result.WriteString("================================================================================\n")

	for _, s := range status {
		appliedStr := "❌ No"
		appliedAt := "Never"

		if s.Applied {
			appliedStr = "✅ Yes"
			if s.AppliedAt != nil {
				appliedAt = s.AppliedAt.Format("2006-01-02 15:04:05")
			}
		}

		result.WriteString(fmt.Sprintf("%-20s %-10s %-20s %s\n", s.Version, appliedStr, appliedAt, s.Description))
	}

	return result.String(), nil
}

// runMigrationUp runs migrations up
func (s *MCPServer) runMigrationUp(ctx context.Context, version string) (string, error) {
	var result strings.Builder

	if version != "" {
		result.WriteString(fmt.Sprintf("Running migration up to version: %s\n", version))
		if err := s.engine.Up(ctx, version); err != nil {
			return "", fmt.Errorf("failed to run migration %s: %w", version, err)
		}
		result.WriteString(fmt.Sprintf("✅ Successfully applied migration: %s\n", version))
	} else {
		result.WriteString("Running all pending migrations...\n")
		status, err := s.engine.GetStatus(ctx)
		if err != nil {
			return "", fmt.Errorf("failed to get migration status: %w", err)
		}

		appliedCount := 0
		for _, migStatus := range status {
			if !migStatus.Applied {
				result.WriteString(fmt.Sprintf("Applying migration: %s - %s\n", migStatus.Version, migStatus.Description))
				if err := s.engine.Up(ctx, migStatus.Version); err != nil {
					return "", fmt.Errorf("failed to run migration %s: %w", migStatus.Version, err)
				}
				result.WriteString(fmt.Sprintf("✅ Applied migration: %s\n", migStatus.Version))
				appliedCount++
			}
		}

		if appliedCount == 0 {
			result.WriteString("No pending migrations found.\n")
		} else {
			result.WriteString(fmt.Sprintf("Successfully applied %d migrations!\n", appliedCount))
		}
	}

	return result.String(), nil
}

// runMigrationDown runs migrations down
func (s *MCPServer) runMigrationDown(ctx context.Context, version string) (string, error) {
	var result strings.Builder

	if version != "" {
		result.WriteString(fmt.Sprintf("Rolling back migration: %s\n", version))
		if err := s.engine.Down(ctx, version); err != nil {
			return "", fmt.Errorf("failed to roll back migration %s: %w", version, err)
		}
		result.WriteString(fmt.Sprintf("✅ Successfully rolled back migration: %s\n", version))
	} else {
		result.WriteString("Rolling back last applied migration...\n")
		status, err := s.engine.GetStatus(ctx)
		if err != nil {
			return "", fmt.Errorf("failed to get migration status: %w", err)
		}

		// Find the last applied migration
		var lastApplied *migration.MigrationStatus
		for i := len(status) - 1; i >= 0; i-- {
			if status[i].Applied {
				lastApplied = &status[i]
				break
			}
		}

		if lastApplied == nil {
			result.WriteString("No migrations to roll back.\n")
		} else {
			result.WriteString(fmt.Sprintf("Rolling back migration: %s - %s\n", lastApplied.Version, lastApplied.Description))
			if err := s.engine.Down(ctx, lastApplied.Version); err != nil {
				return "", fmt.Errorf("failed to roll back migration %s: %w", lastApplied.Version, err)
			}
			result.WriteString(fmt.Sprintf("✅ Successfully rolled back migration: %s\n", lastApplied.Version))
		}
	}

	return result.String(), nil
}

// createMigration creates a new migration file
func (s *MCPServer) createMigration(name, description string) (string, error) {
	if err := s.validateMigrationParams(name, description); err != nil {
		return "", err
	}

	version, cleanName, filepath := s.generateMigrationInfo(name)

	if err := s.createMigrationsDirectory(); err != nil {
		return "", err
	}

	template := s.generateMigrationTemplate(cleanName, description, version)

	if err := s.writeMigrationFile(filepath, template); err != nil {
		return "", err
	}

	return s.formatMigrationCreationResult(filepath, version, name, description, cleanName), nil
}

// validateMigrationParams validates migration creation parameters
func (s *MCPServer) validateMigrationParams(name, description string) error {
	if name == "" {
		return fmt.Errorf("migration name is required")
	}
	if description == "" {
		return fmt.Errorf("migration description is required")
	}
	return nil
}

// generateMigrationInfo generates version, clean name, and file path for migration
func (s *MCPServer) generateMigrationInfo(name string) (version, cleanName, filepath string) {
	version = time.Now().Format("20060102_150405")
	cleanName = strings.ToLower(strings.ReplaceAll(name, " ", "_"))
	filename := fmt.Sprintf("%s_%s.go", version, cleanName)
	filepath = fmt.Sprintf("migrations/%s", filename)
	return
}

// createMigrationsDirectory creates the migrations directory if it doesn't exist
func (s *MCPServer) createMigrationsDirectory() error {
	const migrationsDir = "migrations"
	const dirPermissions = 0750
	if err := os.MkdirAll(migrationsDir, dirPermissions); err != nil {
		return fmt.Errorf("failed to create migrations directory: %w", err)
	}
	return nil
}

// generateMigrationTemplate generates the migration file template
func (s *MCPServer) generateMigrationTemplate(cleanName, description, version string) string {
	camelCaseName := toCamelCase(cleanName)
	return fmt.Sprintf(`package migrations

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// %sMigration %s
type %sMigration struct{}

func (m *%sMigration) Version() string {
	return "%s"
}

func (m *%sMigration) Description() string {
	return "%s"
}

func (m *%sMigration) Up(ctx context.Context, db *mongo.Database) error {
	// TODO: Implement migration up logic
	// Example:
	// collection := db.Collection("your_collection")
	// _, err := collection.UpdateMany(ctx, bson.D{}, bson.D{{"$set", bson.D{{"new_field", "default_value"}}}})
	// return err
	return nil
}

func (m *%sMigration) Down(ctx context.Context, db *mongo.Database) error {
	// TODO: Implement migration down logic (rollback)
	// Example:
	// collection := db.Collection("your_collection")
	// _, err := collection.UpdateMany(ctx, bson.D{}, bson.D{{"$unset", bson.D{{"new_field", ""}}}})
	// return err
	return nil
}
`,
		camelCaseName, description, camelCaseName, camelCaseName, version,
		camelCaseName, description, camelCaseName, camelCaseName)
}

// writeMigrationFile writes the migration template to file
func (s *MCPServer) writeMigrationFile(filepath, template string) error {
	const filePermissions = 0600
	if err := os.WriteFile(filepath, []byte(template), filePermissions); err != nil {
		return fmt.Errorf("failed to write migration file: %w", err)
	}
	return nil
}

// formatMigrationCreationResult formats the result message for migration creation
func (s *MCPServer) formatMigrationCreationResult(filepath, version, name, description, cleanName string) string {
	return fmt.Sprintf(`Created new migration file: %s

Migration Details:
- Version: %s
- Name: %s
- Description: %s
- File: %s

Next steps:
1. Edit the file to implement your migration logic in the Up() method
2. Implement the rollback logic in the Down() method
3. Register the migration in your main application
4. Run the migration with: mongo-essential migrate up

Example registration:
engine.Register(&examplemigrations.%sMigration{})
`, filepath, version, name, description, filepath, toCamelCase(cleanName))
}

// listMigrations lists all registered migrations
func (s *MCPServer) listMigrations(ctx context.Context) (string, error) {
	status, err := s.engine.GetStatus(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get migration status: %w", err)
	}

	var result strings.Builder
	result.WriteString("Registered Migrations:\n")
	result.WriteString("================================================================================\n")

	if len(status) == 0 {
		result.WriteString("No migrations registered.\n")
		result.WriteString("\nTo register migrations, use engine.Register() or engine.RegisterMany() in your application.\n")
	} else {
		result.WriteString(fmt.Sprintf("Total migrations: %d\n\n", len(status)))
		for i, s := range status {
			result.WriteString(fmt.Sprintf("%d. %s\n", i+1, s.Version))
			result.WriteString(fmt.Sprintf("   Description: %s\n", s.Description))
			if s.Applied {
				result.WriteString("   Status: ✅ Applied")
				if s.AppliedAt != nil {
					result.WriteString(fmt.Sprintf(" on %s", s.AppliedAt.Format("2006-01-02 15:04:05")))
				}
				result.WriteString("\n")
			} else {
				result.WriteString("   Status: ⏳ Pending\n")
			}
			result.WriteString("\n")
		}
	}

	return result.String(), nil
}

// toCamelCase converts a snake_case string to CamelCase
func toCamelCase(s string) string {
	parts := strings.Split(s, "_")
	for i, part := range parts {
		if len(part) > 0 {
			parts[i] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, "")
}
