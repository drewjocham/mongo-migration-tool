//go:build integration

package integration_tests_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"github.com/drewjocham/mongo-migration-tool/internal/cli"
	"github.com/drewjocham/mongo-migration-tool/internal/migration"
	_ "github.com/drewjocham/mongo-migration-tool/migrations"
)

type TestEnv struct {
	ConfigPath     string
	DBName         string
	ColName        string
	MigrationsPath string
	MongoClient    *mongo.Client
}

func TestCLICommands(t *testing.T) {
	ctx := context.Background()
	env := setupIntegrationEnv(t, ctx)

	versions := sortedMigrationVersions()
	require.NotEmpty(t, versions)
	latest := versions[len(versions)-1]

	type testCase struct {
		name   string
		setup  func(t *testing.T, env *TestEnv)
		args   []string
		assert func(t *testing.T, env *TestEnv, output string)
	}

	cases := []testCase{
		{
			name: "Version command",
			args: []string{"version"},
			assert: func(t *testing.T, _ *TestEnv, output string) {
				assert.Contains(t, output, "commit:")
			},
		},
		{
			name: "Schema indexes table output",
			args: []string{"schema", "indexes"},
			assert: func(t *testing.T, _ *TestEnv, output string) {
				assert.Contains(t, output, "COLLECTION")
				assert.Contains(t, output, "users")
			},
		},
		{
			name: "Schema indexes JSON output",
			args: []string{"schema", "indexes", "--output", "json"},
			assert: func(t *testing.T, _ *TestEnv, output string) {
				assert.Contains(t, output, "\"Collection\"")
				assert.Contains(t, output, "idx_users_email_unique")
			},
		},
		{
			name: "MCP config command",
			args: []string{"mcp", "config"},
			assert: func(t *testing.T, _ *TestEnv, output string) {
				assert.Contains(t, output, "\"mcpServers\"")
				assert.Contains(t, output, "\"mt\"")
			},
		},
		{
			name: "Initial status is pending",
			args: []string{"status"},
			assert: func(t *testing.T, _ *TestEnv, output string) {
				assertVersionState(t, output, latest, "[ ]")
			},
		},
		{
			name: "Migrate up to latest",
			args: []string{"up"},
			assert: func(t *testing.T, _ *TestEnv, output string) {
				assert.Contains(t, output, "Database is")
				assertMigrationRecordExists(t, env, latest)
			},
		},
		{
			name: "Users indexes exist after migrations",
			args: []string{"status"},
			assert: func(t *testing.T, env *TestEnv, _ string) {
				requireUsersIndexes(t, env)
			},
		},
		{
			name: "Opslog table output",
			args: []string{"opslog"},
			assert: func(t *testing.T, _ *TestEnv, output string) {
				assert.Contains(t, output, "APPLIED AT")
				assert.Contains(t, output, latest)
			},
		},
		{
			name: "Opslog JSON output with search",
			args: []string{"opslog", "--output", "json", "--search", latest},
			assert: func(t *testing.T, _ *TestEnv, output string) {
				assert.Contains(t, output, latest)
			},
		},
		{
			name: "Status shows completed",
			args: []string{"status"},
			assert: func(t *testing.T, _ *TestEnv, output string) {
				assertVersionState(t, output, latest, "[✓]")
			},
		},
		{
			name: "Down command dry run",
			args: []string{"down", "--dry-run"},
			assert: func(t *testing.T, _ *TestEnv, output string) {
				assert.Contains(t, output, "Planned migrations to down")
			},
		},
		{
			name: "Force marks migration as applied",
			args: []string{"force", "--yes", latest},
			assert: func(t *testing.T, _ *TestEnv, _ string) {
			},
		},
		{
			name: "Status shows forced migration applied",
			args: []string{"status"},
			assert: func(t *testing.T, _ *TestEnv, output string) {
				assertVersionState(t, output, latest, "[✓]")
			},
		},
		{
			name: "Unlock releases lock",
			setup: func(t *testing.T, env *TestEnv) {
				lockColl := env.MongoClient.Database(env.DBName).Collection("migrations_lock")
				_, err := lockColl.InsertOne(ctx, bson.M{
					"lock_id":     "migration_engine_lock",
					"acquired_at": time.Now().UTC(),
				})
				require.NoError(t, err)
			},
			args: []string{"unlock", "--yes"},
			assert: func(t *testing.T, _ *TestEnv, output string) {
				assert.Contains(t, output, "Migration lock released")
				assertLockReleased(t, env)
			},
		},
		{
			name: "Create migration file",
			args: []string{"create", "add_users_index"},
			assert: func(t *testing.T, env *TestEnv, output string) {
				assert.Contains(t, output, "Migration created")

				entries, err := os.ReadDir(env.MigrationsPath)
				require.NoError(t, err)
				require.NotEmpty(t, entries)
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setup != nil {
				tc.setup(t, env)
			}
			out := env.RunCLI(t, tc.args...)
			if tc.assert != nil {
				tc.assert(t, env, out)
			}
		})
	}
}

func setupIntegrationEnv(t *testing.T, ctx context.Context) *TestEnv {
	container, err := mongodb.RunContainer(ctx, testcontainers.WithImage("mongo:8.0"))
	require.NoError(t, err)
	t.Cleanup(func() { container.Terminate(context.Background()) })

	connStr, err := container.ConnectionString(ctx)
	require.NoError(t, err)

	dbName := fmt.Sprintf("it_%d", time.Now().UnixNano())
	colName := "schema_migrations"
	migrationsPath := filepath.Join(t.TempDir(), "migrations")

	configContent := fmt.Sprintf(
		"MONGO_URL=%s\nMONGO_DATABASE=%s\nMIGRATIONS_COLLECTION=%s\nMIGRATIONS_PATH=%s\n",
		connStr,
		dbName,
		colName,
		migrationsPath,
	)

	configPath := filepath.Join(t.TempDir(), "mongo-tool.yaml")
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	client, err := mongo.Connect(options.Client().ApplyURI(connStr))
	require.NoError(t, err)
	t.Cleanup(func() { client.Disconnect(context.Background()) })

	t.Setenv("MONGO_URL", connStr)
	t.Setenv("MONGO_DATABASE", dbName)
	t.Setenv("MIGRATIONS_COLLECTION", colName)
	t.Setenv("MIGRATIONS_PATH", migrationsPath)

	return &TestEnv{
		ConfigPath:     configPath,
		DBName:         dbName,
		ColName:        colName,
		MigrationsPath: migrationsPath,
		MongoClient:    client,
	}
}

func (e *TestEnv) RunCLI(t *testing.T, args ...string) string {
	t.Helper()
	oldArgs := os.Args
	os.Args = append([]string{"mongo-tool", "--config", e.ConfigPath}, args...)
	defer func() { os.Args = oldArgs }()

	stdout, stderr, err := captureOutput(cli.Execute)
	require.NoError(t, err, stderr)
	return stdout
}

func captureOutput(f func() error) (string, string, error) {
	oldOut, oldErr := os.Stdout, os.Stderr
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout, os.Stderr = wOut, wErr

	errChan := make(chan error, 1)
	go func() { errChan <- f() }()

	resOut := make(chan string)
	resErr := make(chan string)
	go func() {
		var b bytes.Buffer
		_, _ = io.Copy(&b, rOut)
		resOut <- b.String()
	}()

	go func() {
		var b bytes.Buffer
		_, _ = io.Copy(&b, rErr)
		resErr <- b.String()
	}()

	fErr := <-errChan
	wOut.Close()
	wErr.Close()

	stdout, stderr := <-resOut, <-resErr
	os.Stdout, os.Stderr = oldOut, oldErr
	return stdout, stderr, fErr
}

func assertMigrationRecordExists(t *testing.T, env *TestEnv, version string) {
	t.Helper()
	coll := env.MongoClient.Database(env.DBName).Collection(env.ColName)
	ctx := context.Background()
	err := coll.FindOne(ctx, bson.M{"version": version}).Err()
	require.NoError(t, err)
}

func assertLockReleased(t *testing.T, env *TestEnv) {
	t.Helper()
	coll := env.MongoClient.Database(env.DBName).Collection("migrations_lock")
	ctx := context.Background()
	count, err := coll.CountDocuments(ctx, bson.M{"lock_id": "migration_engine_lock"})
	require.NoError(t, err)
	assert.Zero(t, count)
}

func assertVersionState(t *testing.T, output, version, state string) {
	t.Helper()
	cleanOut := regexp.MustCompile(`\x1b\[[0-9;]*m`).ReplaceAllString(output, "")
	found := false
	for _, line := range strings.Split(cleanOut, "\n") {
		if strings.Contains(line, version) {
			assert.Contains(t, line, state)
			found = true
			break
		}
	}
	assert.True(t, found)
}

func requireUsersIndexes(t *testing.T, env *TestEnv) {
	t.Helper()
	ctx := context.Background()
	coll := env.MongoClient.Database(env.DBName).Collection("users")
	cursor, err := coll.Indexes().List(ctx)
	require.NoError(t, err)

	var indexes []bson.M
	require.NoError(t, cursor.All(ctx, &indexes))

	names := make(map[string]struct{}, len(indexes))
	for _, idx := range indexes {
		if name, ok := idx["name"].(string); ok && name != "" {
			names[name] = struct{}{}
		}
	}

	require.Contains(t, names, "idx_users_email_unique")
	require.Contains(t, names, "idx_users_created_at")
	require.Contains(t, names, "idx_users_status_created_at")
}

func sortedMigrationVersions() []string {
	m := migration.RegisteredMigrations()
	v := make([]string, 0, len(m))
	for k := range m {
		v = append(v, k)
	}
	sort.Strings(v)
	return v
}
