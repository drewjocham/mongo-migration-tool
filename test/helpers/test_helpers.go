// Package helpers provides common test utilities and helpers for testing the mongo-essential application.
package helpers

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/jocham/mongo-essential/migration"
)

const (
	// retryDelayMillis is the delay between MongoDB connection retries
	retryDelayMillis = 100
)

// TestHelper provides common test utilities
type TestHelper struct {
	T              *testing.T
	MongoClient    *mongo.Client
	Database       *mongo.Database
	DatabaseName   string
	CollectionName string
}

// NewTestHelper creates a new test helper
func NewTestHelper(t *testing.T) *TestHelper {
	t.Helper()

	dbName := fmt.Sprintf("test_%s_%d", t.Name(), time.Now().Unix())
	collName := "test_migrations"

	return &TestHelper{
		T:              t,
		DatabaseName:   dbName,
		CollectionName: collName,
	}
}

// Setup connects to MongoDB and creates a test database
func (h *TestHelper) Setup(ctx context.Context) error {
	h.T.Helper()

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(getMongoURL()))
	if err != nil {
		return fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	if err := client.Ping(ctx, nil); err != nil {
		return fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	h.MongoClient = client
	h.Database = client.Database(h.DatabaseName)

	return nil
}

// Cleanup drops the test database and disconnects
func (h *TestHelper) Cleanup(ctx context.Context) {
	h.T.Helper()

	if h.Database != nil {
		if err := h.Database.Drop(ctx); err != nil {
			h.T.Logf("Warning: failed to drop database: %v", err)
		}
	}

	if h.MongoClient != nil {
		if err := h.MongoClient.Disconnect(ctx); err != nil {
			h.T.Logf("Warning: failed to disconnect: %v", err)
		}
	}
}

// CreateEngine creates a migration engine for testing
func (h *TestHelper) CreateEngine() *migration.Engine {
	h.T.Helper()

	if h.Database == nil {
		h.T.Fatal("Database not initialized. Call Setup() first")
	}

	return migration.NewEngine(h.Database, h.CollectionName)
}

// AssertNoError fails the test if err is not nil
func (h *TestHelper) AssertNoError(err error) {
	h.T.Helper()

	if err != nil {
		h.T.Fatalf("Unexpected error: %v", err)
	}
}

// AssertError fails the test if err is nil
func (h *TestHelper) AssertError(err error) {
	h.T.Helper()

	if err == nil {
		h.T.Fatal("Expected error but got nil")
	}
}

// AssertEqual fails the test if expected != actual
func (h *TestHelper) AssertEqual(expected, actual interface{}) {
	h.T.Helper()

	if expected != actual {
		h.T.Fatalf("Expected %v, got %v", expected, actual)
	}
}

// AssertContains fails the test if haystack doesn't contain needle
func (h *TestHelper) AssertContains(haystack, needle string) {
	h.T.Helper()

	if !contains(haystack, needle) {
		h.T.Fatalf("Expected '%s' to contain '%s'", haystack, needle)
	}
}

func getMongoURL() string {
	if url := os.Getenv("MONGO_URL"); url != "" {
		return url
	}
	return "mongodb://localhost:27017"
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) &&
		(haystack == needle || len(needle) == 0 ||
			indexOf(haystack, needle) >= 0)
}

func indexOf(haystack, needle string) int {
	for i := 0; i <= len(haystack)-len(needle); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}

// TestMigration is a migration implementation for testing
type TestMigration struct {
	VersionStr string
	DescStr    string
	UpFunc     func(ctx context.Context, db *mongo.Database) error
	DownFunc   func(ctx context.Context, db *mongo.Database) error
	UpCalled   bool
	DownCalled bool
}

// Version returns the migration version
func (m *TestMigration) Version() string {
	return m.VersionStr
}

// Description returns the migration description
func (m *TestMigration) Description() string {
	return m.DescStr
}

// Up executes the up migration
func (m *TestMigration) Up(ctx context.Context, db *mongo.Database) error {
	m.UpCalled = true
	if m.UpFunc != nil {
		return m.UpFunc(ctx, db)
	}
	return nil
}

// Down executes the down migration
func (m *TestMigration) Down(ctx context.Context, db *mongo.Database) error {
	m.DownCalled = true
	if m.DownFunc != nil {
		return m.DownFunc(ctx, db)
	}
	return nil
}

// NewTestMigration creates a new test migration
func NewTestMigration(version, description string) *TestMigration {
	return &TestMigration{
		VersionStr: version,
		DescStr:    description,
	}
}

// WaitForMongo waits for MongoDB to be available
func WaitForMongo(ctx context.Context, maxWait time.Duration) error {
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(getMongoURL()))
	if err != nil {
		return err
	}
	defer func() {
		if disconnectErr := client.Disconnect(ctx); disconnectErr != nil {
			// Log but don't fail on disconnect error during cleanup
			fmt.Printf("Warning: failed to disconnect MongoDB client: %v\n", disconnectErr)
		}
	}()

	deadline := time.Now().Add(maxWait)
	for time.Now().Before(deadline) {
		if err := client.Ping(ctx, nil); err == nil {
			return nil
		}
		time.Sleep(retryDelayMillis * time.Millisecond)
	}

	return fmt.Errorf("MongoDB not available after %v", maxWait)
}

// SetTestEnv sets environment variables for testing
func SetTestEnv(t *testing.T, vars map[string]string) func() {
	t.Helper()

	oldVars := make(map[string]string)
	for key, value := range vars {
		oldVars[key] = os.Getenv(key)
		if err := os.Setenv(key, value); err != nil {
			t.Logf("Warning: failed to set env var %s: %v", key, err)
		}
	}

	return func() {
		for key, oldValue := range oldVars {
			if oldValue == "" {
				if err := os.Unsetenv(key); err != nil {
					t.Logf("Warning: failed to unset env var %s: %v", key, err)
				}
			} else {
				if err := os.Setenv(key, oldValue); err != nil {
					t.Logf("Warning: failed to restore env var %s: %v", key, err)
				}
			}
		}
	}
}
