//go:build integration

package integration_tests_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/drewjocham/mongo-migration-tool/internal/config"
	"github.com/drewjocham/mongo-migration-tool/internal/mcp"
	_ "github.com/drewjocham/mongo-migration-tool/migrations"
)

type rpcRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
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
	ID      interface{}     `json:"id"`
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

// call handles the JSON-RPC request/response cycle with basic ID verification.
func (c *mcpRPCClient) call(t *testing.T, method string, id int, params interface{}) rpcResponse {
	t.Helper()

	req := rpcRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	if err := c.enc.Encode(req); err != nil {
		t.Fatalf("failed to encode request %v: %v", id, err)
	}

	var resp rpcResponse
	if err := c.dec.Decode(&resp); err != nil {
		t.Fatalf("failed to decode response for %s: %v", method, err)
	}

	if fmt.Sprintf("%v", resp.ID) != fmt.Sprintf("%v", id) {
		t.Fatalf("id mismatch: expected %v, got %v", id, resp.ID)
	}

	if resp.Error != nil {
		t.Fatalf("rpc error [%s]: %s (code: %d)", method, resp.Error.Message, resp.Error.Code)
	}

	return resp
}

func parseToolText(t *testing.T, resp rpcResponse) string {
	t.Helper()
	var res toolCallResult
	if err := json.Unmarshal(resp.Result, &res); err != nil {
		t.Fatalf("failed to unmarshal tool result: %v", err)
	}
	if len(res.Content) == 0 {
		t.Fatalf("tool returned empty content")
	}
	return res.Content[0].Text
}

func connectMongoOrSkip(t *testing.T) (*mongo.Client, *mongo.Database, func()) {
	t.Helper()
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.GetConnectionString()))
	if err != nil || client.Ping(ctx, nil) != nil {
		t.Skip("MongoDB unavailable; skipping integration test")
	}

	db := client.Database(cfg.Database)
	return client, db, func() {
		_ = db.Drop(context.Background())
		_ = client.Disconnect(context.Background())
	}
}

func TestMCPIntegration_FullLifecycle(t *testing.T) {
	dbName := fmt.Sprintf("mcp_it_%d", time.Now().UnixNano())
	t.Setenv("MONGO_DATABASE", dbName)
	t.Setenv("MIGRATIONS_COLLECTION", "schema_migrations_it")

	_, db, cleanupMongo := connectMongoOrSkip(t)
	t.Cleanup(cleanupMongo)

	clientToSrvR, clientToSrvW := io.Pipe()
	srvToClientR, srvToClientW := io.Pipe()

	server, err := mcp.NewMCPServer()
	if err != nil {
		t.Fatalf("server creation failed: %v", err)
	}

	serverCtx, cancelServer := context.WithCancel(context.Background())
	serverDone := make(chan error, 1)

	go func() {
		serverDone <- server.Serve(serverCtx, clientToSrvR, srvToClientW)
	}()

	t.Cleanup(func() {
		cancelServer()
		_ = clientToSrvW.Close()
		_ = srvToClientR.Close()
		server.Close()
	})

	client := &mcpRPCClient{
		enc: json.NewEncoder(clientToSrvW),
		dec: json.NewDecoder(srvToClientR),
	}

	t.Run("Protocol: Initialize", func(t *testing.T) {
		client.call(t, "initialize", 1, map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]interface{}{},
			"clientInfo":      map[string]string{"name": "test-client", "version": "1.0"},
		})
	})

	t.Run("Discovery: ListTools", func(t *testing.T) {
		resp := client.call(t, "tools/list", 2, nil)
		var list toolsListResult
		if err := json.Unmarshal(resp.Result, &list); err != nil {
			t.Fatal(err)
		}

		required := map[string]bool{"migration_up": false, "migration_status": false}
		for _, tool := range list.Tools {
			if _, ok := required[tool.Name]; ok {
				required[tool.Name] = true
			}
		}
		for name, found := range required {
			if !found {
				t.Errorf("missing tool: %s", name)
			}
		}
	})

	t.Run("Execution: MigrationUp", func(t *testing.T) {
		resp := client.call(t, "tools/call", 3, toolCallParams{
			Name: "migration_up",
		})
		text := parseToolText(t, resp)
		if !strings.Contains(text, "Successfully applied") && !strings.Contains(text, "✅") {
			t.Errorf("unexpected output: %s", text)
		}
	})

	t.Run("Verification: Database State", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		cursor, err := db.Collection("users").Indexes().List(ctx)
		if err != nil {
			t.Fatalf("failed to list indexes: %v", err)
		}

		var indexes []bson.M
		if err := cursor.All(ctx, &indexes); err != nil {
			t.Fatal(err)
		}

		expected := map[string]bool{
			"idx_users_email_unique": false,
			"idx_users_created_at":   false,
		}

		for _, idx := range indexes {
			name := idx["name"].(string)
			if _, ok := expected[name]; ok {
				expected[name] = true

				if name == "idx_users_email_unique" {
					if unique, ok := idx["unique"].(bool); !ok || !unique {
						t.Errorf("expected %s to be unique, but it wasn't", name)
					}
				}
			}
		}

		for name, found := range expected {
			if !found {
				t.Errorf("index not found: %s", name)
			}
		}
	})

	t.Run("Consistency: MigrationStatus", func(t *testing.T) {
		resp := client.call(t, "tools/call", 4, toolCallParams{
			Name: "migration_status",
		})
		text := parseToolText(t, resp)
		if !strings.Contains(text, "✅") {
			t.Errorf("status should show applied migration: %s", text)
		}
	})
}
