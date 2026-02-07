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

type rpcResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

type mcpClient struct {
	t   *testing.T
	enc *json.Encoder
	dec *json.Decoder
}

func (c *mcpClient) call(method string, id int, params interface{}, target interface{}) {
	c.t.Helper()
	if err := c.enc.Encode(rpcRequest{"2.0", id, method, params}); err != nil {
		c.t.Fatalf("rpc encode failed: %v", err)
	}

	var resp rpcResponse
	if err := c.dec.Decode(&resp); err != nil {
		c.t.Fatalf("rpc decode failed: %v", err)
	}

	if resp.Error != nil {
		c.t.Fatalf("rpc error [%s]: %s (code: %d)", method, resp.Error.Message, resp.Error.Code)
	}

	if target != nil {
		if err := json.Unmarshal(resp.Result, target); err != nil {
			c.t.Fatalf("failed to unmarshal rpc result: %v", err)
		}
	}
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

func parseToolText(t *testing.T, res struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
}) string {
	if len(res.Content) == 0 {
		t.Fatal("tool returned empty content")
	}
	return res.Content[0].Text
}

func TestMCPIntegration_FullLifecycle(t *testing.T) {
	dbName := fmt.Sprintf("mcp_it_%d", time.Now().UnixNano())
	t.Setenv("MONGO_DATABASE", dbName)
	t.Setenv("MIGRATIONS_COLLECTION", "schema_migrations_it")

	_, db, cleanupMongo := connectMongoOrSkip(t)
	t.Cleanup(cleanupMongo)

	clientToSrvR, clientToSrvW := io.Pipe()
	srvToClientR, srvToClientW := io.Pipe()

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config error: %v", err)
	}

	server, err := mcp.NewMCPServer(cfg)
	if err != nil {
		t.Fatalf("failed to init mcp server: %v", err)
	}
	serverCtx, cancelServer := context.WithCancel(context.Background())

	go func() { _ = server.Serve(serverCtx, clientToSrvR, srvToClientW) }()

	t.Cleanup(func() {
		cancelServer()
		_ = clientToSrvW.Close()
		_ = srvToClientR.Close()
		server.Close()
	})

	client := &mcpClient{
		t:   t,
		enc: json.NewEncoder(clientToSrvW),
		dec: json.NewDecoder(srvToClientR),
	}

	t.Run("Initialize", func(t *testing.T) {
		client.call("initialize", 1, map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"clientInfo":      map[string]string{"name": "test-client", "version": "1.0"},
		}, nil)
	})

	t.Run("ListTools", func(t *testing.T) {
		var res struct {
			Tools []struct {
				Name string `json:"name"`
			}
		}
		client.call("tools/list", 2, nil, &res)

		found := make(map[string]bool)
		for _, tool := range res.Tools {
			found[tool.Name] = true
		}

		for _, name := range []string{"migration_up", "migration_status"} {
			if !found[name] {
				t.Errorf("missing tool: %s", name)
			}
		}
	})

	t.Run("MigrationUp", func(t *testing.T) {
		var res struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
		}
		client.call("tools/call", 3, map[string]interface{}{"name": "migration_up"}, &res)

		text := parseToolText(t, res)
		if !strings.Contains(text, "✅") && !strings.Contains(text, "applied") {
			t.Errorf("unexpected output: %s", text)
		}
	})

	t.Run("VerifyDatabaseState", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		cursor, err := db.Collection("users").Indexes().List(ctx)
		if err != nil {
			t.Fatalf("failed to list indexes: %v", err)
		}

		var indexes []bson.M
		_ = cursor.All(ctx, &indexes)

		expected := map[string]bool{
			"idx_users_email_unique": false,
			"idx_users_created_at":   false,
		}

		for _, idx := range indexes {
			name := idx["name"].(string)
			if _, ok := expected[name]; ok {
				expected[name] = true
				if name == "idx_users_email_unique" && idx["unique"] != true {
					t.Errorf("index %s should be unique", name)
				}
			}
		}

		for name, found := range expected {
			if !found {
				t.Errorf("missing index: %s", name)
			}
		}
	})

	t.Run("MigrationStatus", func(t *testing.T) {
		var res struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
		}
		client.call("tools/call", 4, map[string]interface{}{"name": "migration_status"}, &res)

		text := parseToolText(t, res)
		if !strings.Contains(text, "✅") {
			t.Errorf("status should show success: %s", text)
		}
	})
}
