//go:build integration

package mcp_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/jocham/mongo-migration/config"
	"github.com/jocham/mongo-migration/mcp"
	_ "github.com/jocham/mongo-migration/migrations" // ensure built-in migrations register via init()
)

type rpcRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      int         `json:"id"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    string `json:"data,omitempty"`
}

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type toolCallParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

type toolsListResult struct {
	Tools []struct {
		Name string `json:"name"`
	} `json:"tools"`
}

type toolCallResult struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
}

type mcpRPCClient struct {
	enc *json.Encoder
	dec *json.Decoder
}

func (c *mcpRPCClient) call(t *testing.T, req rpcRequest) rpcResponse {
	t.Helper()

	if err := c.enc.Encode(req); err != nil {
		t.Fatalf("failed to encode request %d/%s: %v", req.ID, req.Method, err)
	}

	var resp rpcResponse
	if err := c.dec.Decode(&resp); err != nil {
		t.Fatalf("failed to decode response for request %d/%s: %v", req.ID, req.Method, err)
	}

	if resp.ID != req.ID {
		t.Fatalf("response id mismatch: expected %d, got %d", req.ID, resp.ID)
	}

	if resp.Error != nil {
		t.Fatalf("rpc error for %s: code=%d message=%s data=%s", req.Method, resp.Error.Code, resp.Error.Message, resp.Error.Data)
	}

	return resp
}

func toolText(t *testing.T, resp rpcResponse) string {
	t.Helper()
	var res toolCallResult
	if err := json.Unmarshal(resp.Result, &res); err != nil {
		t.Fatalf("failed to unmarshal tool call result: %v", err)
	}
	if len(res.Content) == 0 {
		t.Fatalf("expected at least one content item")
	}
	return res.Content[0].Text
}

func connectMongoOrSkip(t *testing.T) (*mongo.Client, *mongo.Database, func()) {
	t.Helper()

	cfg, err := config.LoadFromEnv()
	if err != nil {
		t.Fatalf("failed to load config from env: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.GetConnectionString()))
	if err != nil {
		t.Skipf("MongoDB not reachable (%v). Set MONGO_URL / start mongod and rerun with: go test -tags=integration ./...", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(ctx)
		t.Skipf("MongoDB ping failed (%v). Set MONGO_URL / start mongod and rerun with: go test -tags=integration ./...", err)
	}

	db := client.Database(cfg.Database)
	cleanup := func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		_ = db.Drop(cleanupCtx)
		_ = client.Disconnect(cleanupCtx)
	}

	return client, db, cleanup
}

func TestMCPIntegration_IndexingAndMigrations(t *testing.T) {
	// Unique DB per test run to avoid clobbering a dev DB.
	dbName := fmt.Sprintf("mcp_it_%d", time.Now().UnixNano())
	if os.Getenv("MONGO_DATABASE") == "" {
		t.Setenv("MONGO_DATABASE", dbName)
	} else {
		// Always isolate by overriding when running under integration tag.
		t.Setenv("MONGO_DATABASE", dbName)
	}
	t.Setenv("MIGRATIONS_COLLECTION", "schema_migrations_it")

	_, db, cleanupMongo := connectMongoOrSkip(t)
	t.Cleanup(cleanupMongo)

	server, err := mcp.NewMCPServer()
	if err != nil {
		t.Fatalf("failed to create MCP server: %v", err)
	}
	t.Cleanup(func() { _ = server.Close() })

	clientToServerR, clientToServerW := io.Pipe()
	serverToClientR, serverToClientW := io.Pipe()

	serverDone := make(chan error, 1)
	go func() {
		defer func() { _ = serverToClientW.Close() }()
		serverDone <- server.Serve(context.Background(), clientToServerR, serverToClientW)
	}()
	defer func() {
		_ = clientToServerW.Close()
		_ = clientToServerR.Close()
		_ = serverToClientR.Close()
		_ = serverToClientW.Close()
		select {
		case <-serverDone:
		case <-time.After(2 * time.Second):
			t.Fatal("server did not stop after closing pipes")
		}
	}()

	client := &mcpRPCClient{enc: json.NewEncoder(clientToServerW), dec: json.NewDecoder(serverToClientR)}

	// 1) initialize
	_ = client.call(t, rpcRequest{JSONRPC: "2.0", ID: 1, Method: "initialize", Params: map[string]interface{}{}})

	// 2) tools/list (verify MCP is alive and exposes migration tools)
	toolsResp := client.call(t, rpcRequest{JSONRPC: "2.0", ID: 2, Method: "tools/list", Params: map[string]interface{}{}})
	var tools toolsListResult
	if err := json.Unmarshal(toolsResp.Result, &tools); err != nil {
		t.Fatalf("failed to unmarshal tools/list result: %v", err)
	}
	toolSet := make(map[string]struct{}, len(tools.Tools))
	for _, tool := range tools.Tools {
		toolSet[tool.Name] = struct{}{}
	}
	for _, required := range []string{"migration_status", "migration_up", "migration_down", "migration_list"} {
		if _, ok := toolSet[required]; !ok {
			t.Fatalf("tools/list missing required tool: %s", required)
		}
	}

	// 3) Run migrations via MCP (this will create indexes via example migrations).
	upResp := client.call(t, rpcRequest{
		JSONRPC: "2.0",
		ID:      3,
		Method:  "tools/call",
		Params: toolCallParams{
			Name:      "migration_up",
			Arguments: map[string]interface{}{},
		},
	})
	upText := toolText(t, upResp)
	if !strings.Contains(upText, "Successfully applied") && !strings.Contains(upText, "✅") {
		t.Fatalf("unexpected migration_up output: %q", upText)
	}

	// 4) Verify indexes exist (tests the indexing capability).
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	idxCursor, err := db.Collection("users").Indexes().List(ctx)
	if err != nil {
		t.Fatalf("failed to list indexes: %v", err)
	}
	var idxDocs []bson.M
	if err := idxCursor.All(ctx, &idxDocs); err != nil {
		t.Fatalf("failed to decode indexes: %v", err)
	}

	idxNames := make(map[string]struct{}, len(idxDocs))
	for _, d := range idxDocs {
		if name, ok := d["name"].(string); ok {
			idxNames[name] = struct{}{}
		}
	}

	for _, expected := range []string{"idx_users_email_unique", "idx_users_created_at", "idx_users_status_created_at"} {
		if _, ok := idxNames[expected]; !ok {
			t.Fatalf("expected index %q to exist; got names=%v", expected, keys(idxNames))
		}
	}

	// 5) Ensure MCP still responds after doing work.
	statusResp := client.call(t, rpcRequest{
		JSONRPC: "2.0",
		ID:      4,
		Method:  "tools/call",
		Params:  toolCallParams{Name: "migration_status", Arguments: map[string]interface{}{}},
	})
	statusText := toolText(t, statusResp)
	if !strings.Contains(statusText, "20240101_001") {
		t.Fatalf("migration_status output missing expected version: %q", statusText)
	}
	if !strings.Contains(statusText, "✅") {
		t.Fatalf("migration_status output missing applied indicator: %q", statusText)
	}
}

func keys(m map[string]struct{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
