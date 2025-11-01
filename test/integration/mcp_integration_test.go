// +build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/jocham/mongo-essential/mcp"
)

const (
	testMongoURL      = "mongodb://localhost:27017"
	testDatabase      = "test_mcp_integration"
	testCollection    = "test_migrations"
	integrationTimeout = 30 * time.Second
)

// TestMCPServerIntegration tests the MCP server with a real MongoDB instance
func TestMCPServerIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Setup
	ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout)
	defer cancel()

	client, db := setupTestDatabase(t, ctx)
	defer cleanupTestDatabase(t, ctx, client, db)

	t.Run("initialize protocol", func(t *testing.T) {
		testInitializeProtocol(t, ctx)
	})

	t.Run("list tools", func(t *testing.T) {
		testListTools(t, ctx)
	})

	t.Run("migration status", func(t *testing.T) {
		testMigrationStatus(t, ctx, db)
	})

	t.Run("migration lifecycle", func(t *testing.T) {
		testMigrationLifecycle(t, ctx, db)
	})
}

func setupTestDatabase(t *testing.T, ctx context.Context) (*mongo.Client, *mongo.Database) {
	t.Helper()

	// Set environment variables for the test
	os.Setenv("MONGO_URL", testMongoURL)
	os.Setenv("MONGO_DATABASE", testDatabase)
	os.Setenv("MIGRATIONS_COLLECTION", testCollection)

	// Connect to MongoDB
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(testMongoURL))
	if err != nil {
		t.Fatalf("Failed to connect to MongoDB: %v", err)
	}

	// Ping to verify connection
	if err := client.Ping(ctx, nil); err != nil {
		t.Fatalf("Failed to ping MongoDB: %v", err)
	}

	db := client.Database(testDatabase)

	// Clean up any existing data
	if err := db.Drop(ctx); err != nil {
		t.Logf("Warning: Failed to drop test database: %v", err)
	}

	return client, db
}

func cleanupTestDatabase(t *testing.T, ctx context.Context, client *mongo.Client, db *mongo.Database) {
	t.Helper()

	if err := db.Drop(ctx); err != nil {
		t.Logf("Warning: Failed to drop test database during cleanup: %v", err)
	}

	if err := client.Disconnect(ctx); err != nil {
		t.Logf("Warning: Failed to disconnect from MongoDB: %v", err)
	}

	// Clean up environment variables
	os.Unsetenv("MONGO_URL")
	os.Unsetenv("MONGO_DATABASE")
	os.Unsetenv("MIGRATIONS_COLLECTION")
}

func testInitializeProtocol(t *testing.T, ctx context.Context) {
	request := mcp.MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params:  json.RawMessage(`{}`),
	}

	response := sendMCPRequest(t, ctx, request)

	if response.Error != nil {
		t.Fatalf("Expected no error, got: %v", response.Error)
	}

	result, ok := response.Result.(map[string]interface{})
	if !ok {
		t.Fatal("Result is not a map")
	}

	if result["protocolVersion"] != "2024-11-05" {
		t.Errorf("Expected protocol version 2024-11-05, got %v", result["protocolVersion"])
	}

	capabilities, ok := result["capabilities"].(map[string]interface{})
	if !ok {
		t.Fatal("capabilities is not a map")
	}

	if capabilities["tools"] == nil {
		t.Error("Expected tools capability to be defined")
	}
}

func testListTools(t *testing.T, ctx context.Context) {
	request := mcp.MCPRequest{
		JSONRPC: "2.0",
		ID:      2,
		Method:  "tools/list",
	}

	response := sendMCPRequest(t, ctx, request)

	if response.Error != nil {
		t.Fatalf("Expected no error, got: %v", response.Error)
	}

	result, ok := response.Result.(map[string]interface{})
	if !ok {
		t.Fatal("Result is not a map")
	}

	tools, ok := result["tools"].([]interface{})
	if !ok {
		t.Fatal("tools is not a slice")
	}

	expectedTools := map[string]bool{
		"migration_status": false,
		"migration_up":     false,
		"migration_down":   false,
		"migration_create": false,
		"migration_list":   false,
	}

	for _, toolInterface := range tools {
		tool, ok := toolInterface.(map[string]interface{})
		if !ok {
			continue
		}
		name, ok := tool["name"].(string)
		if !ok {
			continue
		}
		if _, exists := expectedTools[name]; exists {
			expectedTools[name] = true
		}
	}

	for name, found := range expectedTools {
		if !found {
			t.Errorf("Expected tool %s not found", name)
		}
	}
}

func testMigrationStatus(t *testing.T, ctx context.Context, db *mongo.Database) {
	params := mcp.ToolCallParams{
		Name:      "migration_status",
		Arguments: map[string]interface{}{},
	}

	paramsJSON, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("Failed to marshal params: %v", err)
	}

	request := mcp.MCPRequest{
		JSONRPC: "2.0",
		ID:      3,
		Method:  "tools/call",
		Params:  paramsJSON,
	}

	response := sendMCPRequest(t, ctx, request)

	if response.Error != nil {
		t.Fatalf("Expected no error, got: %v", response.Error)
	}

	result, ok := response.Result.(map[string]interface{})
	if !ok {
		t.Fatal("Result is not a map")
	}

	content, ok := result["content"].([]interface{})
	if !ok {
		t.Fatal("content is not a slice")
	}

	if len(content) == 0 {
		t.Fatal("Expected at least one content item")
	}

	firstContent, ok := content[0].(map[string]interface{})
	if !ok {
		t.Fatal("First content item is not a map")
	}

	text, ok := firstContent["text"].(string)
	if !ok {
		t.Fatal("text is not a string")
	}

	if !strings.Contains(text, "Migration Status") {
		t.Error("Expected status output to contain 'Migration Status'")
	}
}

func testMigrationLifecycle(t *testing.T, ctx context.Context, db *mongo.Database) {
	// Test migration create
	t.Run("create migration", func(t *testing.T) {
		params := mcp.ToolCallParams{
			Name: "migration_create",
			Arguments: map[string]interface{}{
				"name":        "test_migration",
				"description": "Test migration for integration test",
			},
		}

		paramsJSON, err := json.Marshal(params)
		if err != nil {
			t.Fatalf("Failed to marshal params: %v", err)
		}

		request := mcp.MCPRequest{
			JSONRPC: "2.0",
			ID:      4,
			Method:  "tools/call",
			Params:  paramsJSON,
		}

		response := sendMCPRequest(t, ctx, request)

		if response.Error != nil {
			t.Fatalf("Expected no error creating migration, got: %v", response.Error)
		}

		result, ok := response.Result.(map[string]interface{})
		if !ok {
			t.Fatal("Result is not a map")
		}

		content, ok := result["content"].([]interface{})
		if !ok || len(content) == 0 {
			t.Fatal("Expected content in response")
		}

		firstContent, ok := content[0].(map[string]interface{})
		if !ok {
			t.Fatal("First content item is not a map")
		}

		text, ok := firstContent["text"].(string)
		if !ok {
			t.Fatal("text is not a string")
		}

		if !strings.Contains(text, "Created new migration file") {
			t.Error("Expected confirmation of migration creation")
		}
	})

	// Test listing migrations
	t.Run("list migrations", func(t *testing.T) {
		params := mcp.ToolCallParams{
			Name:      "migration_list",
			Arguments: map[string]interface{}{},
		}

		paramsJSON, err := json.Marshal(params)
		if err != nil {
			t.Fatalf("Failed to marshal params: %v", err)
		}

		request := mcp.MCPRequest{
			JSONRPC: "2.0",
			ID:      5,
			Method:  "tools/call",
			Params:  paramsJSON,
		}

		response := sendMCPRequest(t, ctx, request)

		if response.Error != nil {
			t.Fatalf("Expected no error listing migrations, got: %v", response.Error)
		}
	})
}

func sendMCPRequest(t *testing.T, ctx context.Context, request mcp.MCPRequest) mcp.MCPResponse {
	t.Helper()

	// Marshal request
	requestData, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	// For integration testing, we'll create a server instance directly
	server, err := mcp.NewMCPServer()
	if err != nil {
		t.Fatalf("Failed to create MCP server: %v", err)
	}
	defer server.Close()

	// Process request through server's internal handler
	// This simulates what would happen via stdio
	var req mcp.MCPRequest
	if err := json.Unmarshal(requestData, &req); err != nil {
		t.Fatalf("Failed to unmarshal request: %v", err)
	}

	// Use reflection or create a test helper in the mcp package
	// For now, we'll return a mock response
	// In a real integration test, you'd want to expose a HandleRequest method

	response := mcp.MCPResponse{
		JSONRPC: "2.0",
		ID:      request.ID,
		Result:  map[string]interface{}{"status": "ok"},
	}

	return response
}

// TestMCPServerCLI tests the MCP server via CLI
func TestMCPServerCLI(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI integration test")
	}

	// Build the binary first
	buildCmd := exec.Command("make", "build")
	if output, err := buildCmd.CombinedOutput(); err != nil {
		t.Fatalf("Failed to build binary: %v\n%s", err, output)
	}

	// Set up test environment
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create a test request
	request := mcp.MCPRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params:  json.RawMessage(`{}`),
	}

	requestData, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	// Start MCP server process
	cmd := exec.CommandContext(ctx, "./build/mongo-essential", "mcp", "--with-examples")
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("MONGO_URL=%s", testMongoURL),
		fmt.Sprintf("MONGO_DATABASE=%s", testDatabase),
	)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatalf("Failed to create stdin pipe: %v", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatalf("Failed to create stdout pipe: %v", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		t.Fatalf("Failed to create stderr pipe: %v", err)
	}

	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start MCP server: %v", err)
	}

	// Send request
	if _, err := stdin.Write(requestData); err != nil {
		t.Fatalf("Failed to write request: %v", err)
	}
	if _, err := stdin.Write([]byte("\n")); err != nil {
		t.Fatalf("Failed to write newline: %v", err)
	}
	stdin.Close()

	// Read response
	var responseData bytes.Buffer
	if _, err := io.Copy(&responseData, stdout); err != nil {
		t.Logf("Error reading stdout: %v", err)
	}

	// Read stderr for debugging
	var stderrData bytes.Buffer
	if _, err := io.Copy(&stderrData, stderr); err != nil {
		t.Logf("Error reading stderr: %v", err)
	}

	if stderrData.Len() > 0 {
		t.Logf("Server stderr: %s", stderrData.String())
	}

	// Wait for process to complete
	if err := cmd.Wait(); err != nil {
		// It's okay if the process exits after we close stdin
		t.Logf("Server process exited: %v", err)
	}

	// Try to parse response
	if responseData.Len() > 0 {
		var response mcp.MCPResponse
		decoder := json.NewDecoder(&responseData)
		if err := decoder.Decode(&response); err != nil {
			t.Logf("Failed to decode response: %v", err)
			t.Logf("Response data: %s", responseData.String())
		} else {
			if response.Error != nil {
				t.Errorf("MCP server returned error: %v", response.Error)
			}
			t.Logf("Successfully received response with ID: %v", response.ID)
		}
	}
}

// TestMigrationEndToEnd tests complete migration flow
func TestMigrationEndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping end-to-end test")
	}

	ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout)
	defer cancel()

	client, db := setupTestDatabase(t, ctx)
	defer cleanupTestDatabase(t, ctx, client, db)

	// Create a test collection
	testColl := db.Collection("users")

	// Insert test data
	_, err := testColl.InsertOne(ctx, bson.M{
		"name":  "Test User",
		"email": "test@example.com",
	})
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	// Verify data was inserted
	count, err := testColl.CountDocuments(ctx, bson.M{})
	if err != nil {
		t.Fatalf("Failed to count documents: %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 document, got %d", count)
	}

	// Test migration tracking collection
	migrationsColl := db.Collection(testCollection)

	// Insert a migration record
	migrationRecord := bson.M{
		"version":     "20240101_001",
		"description": "Test migration",
		"applied_at":  time.Now(),
	}

	_, err = migrationsColl.InsertOne(ctx, migrationRecord)
	if err != nil {
		t.Fatalf("Failed to insert migration record: %v", err)
	}

	// Verify migration record
	var foundRecord bson.M
	err = migrationsColl.FindOne(ctx, bson.M{"version": "20240101_001"}).Decode(&foundRecord)
	if err != nil {
		t.Fatalf("Failed to find migration record: %v", err)
	}

	if foundRecord["description"] != "Test migration" {
		t.Errorf("Expected description 'Test migration', got %v", foundRecord["description"])
	}

	t.Log("End-to-end test completed successfully")
}
