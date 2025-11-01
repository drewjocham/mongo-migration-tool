package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"

	"github.com/jocham/mongo-essential/migration"
)

// TestMigration is a simple test migration
type TestMigration struct {
	version      string
	description  string
	upExecuted   bool
	downExecuted bool
}

func (m *TestMigration) Version() string {
	return m.version
}

func (m *TestMigration) Description() string {
	return m.description
}

func (m *TestMigration) Up(_ context.Context, _ *mongo.Database) error {
	m.upExecuted = true
	return nil
}

func (m *TestMigration) Down(_ context.Context, _ *mongo.Database) error {
	m.downExecuted = true
	return nil
}

func TestMCPRequest(t *testing.T) {
	tests := []struct {
		name     string
		jsonData string
		wantErr  bool
	}{
		{
			name:     "valid initialize request",
			jsonData: `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`,
			wantErr:  false,
		},
		{
			name:     "valid tools/list request",
			jsonData: `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`,
			wantErr:  false,
		},
		{
			name:     "invalid json",
			jsonData: `{invalid json}`,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req MCPRequest
			err := json.Unmarshal([]byte(tt.jsonData), &req)
			if (err != nil) != tt.wantErr {
				t.Errorf("Unmarshal() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMCPResponse(t *testing.T) {
	tests := []struct {
		name     string
		response MCPResponse
	}{
		{
			name: "success response",
			response: MCPResponse{
				JSONRPC: "2.0",
				ID:      1,
				Result:  map[string]interface{}{"status": "ok"},
			},
		},
		{
			name: "error response",
			response: MCPResponse{
				JSONRPC: "2.0",
				ID:      1,
				Error: &MCPError{
					Code:    -32600,
					Message: "Invalid Request",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.response)
			if err != nil {
				t.Fatalf("Failed to marshal response: %v", err)
			}

			var decoded MCPResponse
			if err := json.Unmarshal(data, &decoded); err != nil {
				t.Fatalf("Failed to unmarshal response: %v", err)
			}

			if decoded.JSONRPC != "2.0" {
				t.Errorf("Expected JSONRPC 2.0, got %s", decoded.JSONRPC)
			}
		})
	}
}

func TestHandleInitialize(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("returns correct protocol version", func(mt *mtest.T) {
		server := &MCPServer{
			db:     mt.DB,
			engine: migration.NewEngine(mt.DB, "test_migrations"),
		}

		request := &MCPRequest{
			JSONRPC: "2.0",
			ID:      1,
			Method:  "initialize",
		}

		response := server.handleInitialize(request)

		if response.Error != nil {
			t.Fatalf("Expected no error, got %v", response.Error)
		}

		result, ok := response.Result.(map[string]interface{})
		if !ok {
			t.Fatal("Result is not a map")
		}

		if result["protocolVersion"] != "2024-11-05" {
			t.Errorf("Expected protocol version 2024-11-05, got %v", result["protocolVersion"])
		}

		serverInfo, ok := result["serverInfo"].(map[string]interface{})
		if !ok {
			t.Fatal("serverInfo is not a map")
		}

		if serverInfo["name"] != "mongo-essential" {
			t.Errorf("Expected server name mongo-essential, got %v", serverInfo["name"])
		}
	})
}

func TestHandleToolsList(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("returns all available tools", func(mt *mtest.T) {
		server := &MCPServer{
			db:     mt.DB,
			engine: migration.NewEngine(mt.DB, "test_migrations"),
		}

		request := &MCPRequest{
			JSONRPC: "2.0",
			ID:      2,
			Method:  "tools/list",
		}

		response := server.handleToolsList(request)

		if response.Error != nil {
			t.Fatalf("Expected no error, got %v", response.Error)
		}

		result, ok := response.Result.(map[string]interface{})
		if !ok {
			t.Fatal("Result is not a map")
		}

		tools, ok := result["tools"].([]Tool)
		if !ok {
			t.Fatal("tools is not a Tool slice")
		}

		expectedTools := []string{
			"migration_status",
			"migration_up",
			"migration_down",
			"migration_create",
			"migration_list",
		}

		if len(tools) != len(expectedTools) {
			t.Errorf("Expected %d tools, got %d", len(expectedTools), len(tools))
		}

		toolNames := make(map[string]bool)
		for _, tool := range tools {
			toolNames[tool.Name] = true
		}

		for _, expected := range expectedTools {
			if !toolNames[expected] {
				t.Errorf("Expected tool %s not found", expected)
			}
		}
	})
}

func TestGetAvailableTools(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("returns correct tool definitions", func(mt *mtest.T) {
		server := &MCPServer{
			db:     mt.DB,
			engine: migration.NewEngine(mt.DB, "test_migrations"),
		}

		tools := server.getAvailableTools()

		if len(tools) != 5 {
			t.Errorf("Expected 5 tools, got %d", len(tools))
		}

		// Check each tool has required fields
		for _, tool := range tools {
			if tool.Name == "" {
				t.Error("Tool name should not be empty")
			}
			if tool.Description == "" {
				t.Error("Tool description should not be empty")
			}
			if tool.InputSchema == nil {
				t.Error("Tool input schema should not be nil")
			}
		}
	})
}

func TestHandleRequest_UnknownMethod(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("returns error for unknown method", func(mt *mtest.T) {
		server := &MCPServer{
			db:     mt.DB,
			engine: migration.NewEngine(mt.DB, "test_migrations"),
		}

		request := &MCPRequest{
			JSONRPC: "2.0",
			ID:      3,
			Method:  "unknown_method",
		}

		response := server.handleRequest(request)

		if response.Error == nil {
			t.Fatal("Expected error for unknown method")
		}

		if response.Error.Code != -32601 {
			t.Errorf("Expected error code -32601, got %d", response.Error.Code)
		}

		if response.Error.Message != "Method not found" {
			t.Errorf("Expected 'Method not found', got %s", response.Error.Message)
		}
	})
}

func TestCreateErrorResponse(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("creates correct error response", func(mt *mtest.T) {
		server := &MCPServer{
			db:     mt.DB,
			engine: migration.NewEngine(mt.DB, "test_migrations"),
		}

		response := server.createErrorResponse(1, -32600, "Invalid Request", "Additional data")

		if response.JSONRPC != "2.0" {
			t.Errorf("Expected JSONRPC 2.0, got %s", response.JSONRPC)
		}

		if response.ID != 1 {
			t.Errorf("Expected ID 1, got %v", response.ID)
		}

		if response.Error == nil {
			t.Fatal("Expected error to be set")
		}

		if response.Error.Code != -32600 {
			t.Errorf("Expected error code -32600, got %d", response.Error.Code)
		}

		if response.Error.Message != "Invalid Request" {
			t.Errorf("Expected 'Invalid Request', got %s", response.Error.Message)
		}

		if response.Error.Data != "Additional data" {
			t.Errorf("Expected 'Additional data', got %s", response.Error.Data)
		}
	})
}

func TestCreateSuccessResponse(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("creates correct success response", func(mt *mtest.T) {
		server := &MCPServer{
			db:     mt.DB,
			engine: migration.NewEngine(mt.DB, "test_migrations"),
		}

		response := server.createSuccessResponse(1, "test result")

		if response.JSONRPC != "2.0" {
			t.Errorf("Expected JSONRPC 2.0, got %s", response.JSONRPC)
		}

		if response.ID != 1 {
			t.Errorf("Expected ID 1, got %v", response.ID)
		}

		if response.Error != nil {
			t.Errorf("Expected no error, got %v", response.Error)
		}

		result, ok := response.Result.(map[string]interface{})
		if !ok {
			t.Fatal("Result is not a map")
		}

		content, ok := result["content"].([]map[string]interface{})
		if !ok {
			t.Fatal("content is not a slice of maps")
		}

		if len(content) != 1 {
			t.Errorf("Expected 1 content item, got %d", len(content))
		}

		if content[0]["type"] != "text" {
			t.Errorf("Expected type 'text', got %v", content[0]["type"])
		}

		if content[0]["text"] != "test result" {
			t.Errorf("Expected text 'test result', got %v", content[0]["text"])
		}
	})
}

func TestToCamelCase(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"snake_case_name", "SnakeCaseName"},
		{"single", "Single"},
		{"multiple_words_here", "MultipleWordsHere"},
		{"", ""},
		{"_leading_underscore", "LeadingUnderscore"},
		{"trailing_underscore_", "TrailingUnderscore"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := toCamelCase(tt.input)
			if result != tt.expected {
				t.Errorf("toCamelCase(%s) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestValidateMigrationParams(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("validates migration parameters", func(mt *mtest.T) {
		server := &MCPServer{
			db:     mt.DB,
			engine: migration.NewEngine(mt.DB, "test_migrations"),
		}

		tests := []struct {
			name        string
			description string
			wantErr     bool
		}{
			{"valid_name", "valid description", false},
			{"", "valid description", true},
			{"valid_name", "", true},
			{"", "", true},
		}

		for _, tt := range tests {
			err := server.validateMigrationParams(tt.name, tt.description)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateMigrationParams(%s, %s) error = %v, wantErr %v",
					tt.name, tt.description, err, tt.wantErr)
			}
		}
	})
}

func TestGenerateMigrationInfo(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("generates correct migration info", func(mt *mtest.T) {
		server := &MCPServer{
			db:     mt.DB,
			engine: migration.NewEngine(mt.DB, "test_migrations"),
		}

		version, cleanName, filepath := server.generateMigrationInfo("Add User Index")

		// Check version format (YYYYMMDD_HHMMSS)
		if len(version) != 15 {
			t.Errorf("Expected version length 15, got %d", len(version))
		}

		// Check clean name
		if cleanName != "add_user_index" {
			t.Errorf("Expected cleanName 'add_user_index', got %s", cleanName)
		}

		// Check filepath
		expectedPrefix := "migrations/" + version + "_add_user_index.go"
		if filepath != expectedPrefix {
			t.Errorf("Expected filepath %s, got %s", expectedPrefix, filepath)
		}
	})
}

func TestGenerateMigrationTemplate(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("generates valid migration template", func(mt *mtest.T) {
		server := &MCPServer{
			db:     mt.DB,
			engine: migration.NewEngine(mt.DB, "test_migrations"),
		}

		template := server.generateMigrationTemplate("add_user_index", "Add index to users", "20240101_120000")

		// Check that template contains expected content
		expectedStrings := []string{
			"package migrations",
			"AddUserIndexMigration",
			"20240101_120000",
			"Add index to users",
			"func (m *AddUserIndexMigration) Up",
			"func (m *AddUserIndexMigration) Down",
		}

		for _, expected := range expectedStrings {
			if !strings.Contains(template, expected) {
				t.Errorf("Template should contain '%s'", expected)
			}
		}
	})
}

func TestHandleToolsCall_InvalidParams(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("returns error for invalid params", func(mt *mtest.T) {
		server := &MCPServer{
			db:     mt.DB,
			engine: migration.NewEngine(mt.DB, "test_migrations"),
		}

		request := &MCPRequest{
			JSONRPC: "2.0",
			ID:      4,
			Method:  "tools/call",
			Params:  json.RawMessage(`{invalid json}`),
		}

		response := server.handleToolsCall(request)

		if response.Error == nil {
			t.Fatal("Expected error for invalid params")
		}

		if response.Error.Code != -32602 {
			t.Errorf("Expected error code -32602, got %d", response.Error.Code)
		}
	})
}

func TestExecuteTool_UnknownTool(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("returns error for unknown tool", func(mt *mtest.T) {
		server := &MCPServer{
			db:     mt.DB,
			engine: migration.NewEngine(mt.DB, "test_migrations"),
		}

		params := ToolCallParams{
			Name:      "unknown_tool",
			Arguments: map[string]interface{}{},
		}

		ctx := context.Background()
		_, err := server.executeTool(ctx, params)

		if err == nil {
			t.Fatal("Expected error for unknown tool")
		}

		if !strings.Contains(err.Error(), "unknown tool") {
			t.Errorf("Expected 'unknown tool' error, got %v", err)
		}
	})
}

func TestGetMigrationStatus(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("formats status correctly", func(mt *mtest.T) {
		server := &MCPServer{
			db:     mt.DB,
			engine: migration.NewEngine(mt.DB, "test_migrations"),
		}

		// Register test migrations
		server.RegisterMigrations(
			&TestMigration{version: "20240101_001", description: "Test migration 1"},
			&TestMigration{version: "20240101_002", description: "Test migration 2"},
		)

		ctx := context.Background()

		// Mock the collection operations
		mt.AddMockResponses(mtest.CreateCursorResponse(1, "test.test_migrations", mtest.FirstBatch))

		result, err := server.getMigrationStatus(ctx)

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if !strings.Contains(result, "Migration Status") {
			t.Error("Result should contain 'Migration Status'")
		}

		if !strings.Contains(result, "20240101_001") {
			t.Error("Result should contain migration version")
		}
	})
}

func TestListMigrations(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("lists migrations correctly", func(mt *mtest.T) {
		server := &MCPServer{
			db:     mt.DB,
			engine: migration.NewEngine(mt.DB, "test_migrations"),
		}

		// Register test migrations
		server.RegisterMigrations(
			&TestMigration{version: "20240101_001", description: "Test migration 1"},
			&TestMigration{version: "20240101_002", description: "Test migration 2"},
		)

		ctx := context.Background()

		// Mock the collection operations
		mt.AddMockResponses(mtest.CreateCursorResponse(1, "test.test_migrations", mtest.FirstBatch))

		result, err := server.listMigrations(ctx)

		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if !strings.Contains(result, "Registered Migrations") {
			t.Error("Result should contain 'Registered Migrations'")
		}

		if !strings.Contains(result, "Total migrations: 2") {
			t.Error("Result should show 2 migrations")
		}
	})
}

func TestMCPServerLifecycle(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("register and close server", func(mt *mtest.T) {
		server := &MCPServer{
			db:     mt.DB,
			client: mt.Client,
			engine: migration.NewEngine(mt.DB, "test_migrations"),
		}

		// Register a migration
		migration := &TestMigration{
			version:     "20240101_001",
			description: "Test migration",
		}
		server.RegisterMigration(migration)

		// Close should not error
		if err := server.Close(); err != nil {
			t.Errorf("Close() error = %v", err)
		}
	})
}

func BenchmarkHandleInitialize(b *testing.B) {
	mt := mtest.New(&testing.T{}, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("benchmark initialize", func(mt *mtest.T) {
		server := &MCPServer{
			db:     mt.DB,
			engine: migration.NewEngine(mt.DB, "test_migrations"),
		}

		request := &MCPRequest{
			JSONRPC: "2.0",
			ID:      1,
			Method:  "initialize",
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = server.handleInitialize(request)
		}
	})
}

func BenchmarkHandleToolsList(b *testing.B) {
	mt := mtest.New(&testing.T{}, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("benchmark tools list", func(mt *mtest.T) {
		server := &MCPServer{
			db:     mt.DB,
			engine: migration.NewEngine(mt.DB, "test_migrations"),
		}

		request := &MCPRequest{
			JSONRPC: "2.0",
			ID:      2,
			Method:  "tools/list",
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = server.handleToolsList(request)
		}
	})
}

func BenchmarkToCamelCase(b *testing.B) {
	input := "add_user_email_index"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = toCamelCase(input)
	}
}

// TestMCPProtocolCompliance tests MCP protocol compliance
func TestMCPProtocolCompliance(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("all responses include JSONRPC 2.0", func(mt *mtest.T) {
		server := &MCPServer{
			db:     mt.DB,
			engine: migration.NewEngine(mt.DB, "test_migrations"),
		}

		testCases := []struct {
			name    string
			request *MCPRequest
		}{
			{"initialize", &MCPRequest{JSONRPC: "2.0", ID: 1, Method: "initialize"}},
			{"tools/list", &MCPRequest{JSONRPC: "2.0", ID: 2, Method: "tools/list"}},
			{"unknown", &MCPRequest{JSONRPC: "2.0", ID: 3, Method: "unknown"}},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				response := server.handleRequest(tc.request)
				if response.JSONRPC != "2.0" {
					t.Errorf("Expected JSONRPC 2.0, got %s", response.JSONRPC)
				}
				if response.ID != tc.request.ID {
					t.Errorf("Expected ID %v, got %v", tc.request.ID, response.ID)
				}
			})
		}
	})
}

// Helper function to create a test server with in-memory processing
func createTestServer(t *testing.T, mt *mtest.T) *MCPServer {
	return &MCPServer{
		db:     mt.DB,
		client: mt.Client,
		engine: migration.NewEngine(mt.DB, "test_migrations"),
	}
}

// TestJSONRPCFlow tests complete JSON-RPC request/response flow
func TestJSONRPCFlow(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))

	mt.Run("complete initialize flow", func(mt *mtest.T) {
		server := createTestServer(t, mt)

		// Create request
		reqData := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}`
		var request MCPRequest
		if err := json.Unmarshal([]byte(reqData), &request); err != nil {
			t.Fatalf("Failed to unmarshal request: %v", err)
		}

		// Handle request
		response := server.handleRequest(&request)

		// Marshal response
		respData, err := json.Marshal(response)
		if err != nil {
			t.Fatalf("Failed to marshal response: %v", err)
		}

		// Verify response can be decoded
		var decodedResp MCPResponse
		if err := json.Unmarshal(respData, &decodedResp); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if decodedResp.Error != nil {
			t.Errorf("Expected no error, got %v", decodedResp.Error)
		}
	})
}
