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
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/drewjocham/mongo-migration-tool/internal/cli"
	"github.com/drewjocham/mongo-migration-tool/migration"
	_ "github.com/drewjocham/mongo-migration-tool/migrations"
)

type TestEnv struct {
	ConfigPath  string
	DBName      string
	ColName     string
	MongoClient *mongo.Client
}

func TestMigrationLifecycle(t *testing.T) {
	ctx := context.Background()
	env := setupIntegrationEnv(t, ctx)

	versions := sortedMigrationVersions()
	require.NotEmpty(t, versions)
	latest := versions[len(versions)-1]

	steps := []struct {
		name         string
		args         []string
		expectOutput string
		expectState  string
	}{
		{
			name:        "Initial status is pending",
			args:        []string{"status"},
			expectState: "[ ]",
		},
		{
			name:         "Migrate up to latest",
			args:         []string{"up"},
			expectOutput: "Database is up to date",
		},
		{
			name:        "Status shows completed",
			args:        []string{"status"},
			expectState: "[✓]",
		},
	}

	for _, tt := range steps {
		t.Run(tt.name, func(t *testing.T) {
			out := env.RunCLI(t, tt.args...)
			if tt.expectOutput != "" {
				assert.Contains(t, out, tt.expectOutput)
			}
			if tt.expectState != "" {
				assertVersionState(t, out, latest, tt.expectState)
			}
		})
	}

	t.Run("Verify DB persistence", func(t *testing.T) {
		col := env.MongoClient.Database(env.DBName).Collection(env.ColName)
		count, err := col.CountDocuments(ctx, bson.M{"version": latest})
		require.NoError(t, err)
		assert.Equal(t, int64(1), count)
	})
}

func setupIntegrationEnv(t *testing.T, ctx context.Context) *TestEnv {
	container, err := mongodb.RunContainer(ctx, testcontainers.WithImage("mongo:8.0"))
	require.NoError(t, err)
	t.Cleanup(func() { container.Terminate(context.Background()) })

	connStr, err := container.ConnectionString(ctx)
	require.NoError(t, err)

	dbName := fmt.Sprintf("it_%d", time.Now().UnixNano())
	colName := "schema_migrations"

	configContent := fmt.Sprintf(
		"MONGO_URL=%s\nMONGO_DATABASE=%s\nMIGRATIONS_COLLECTION=%s\n",
		connStr,
		dbName,
		colName,
	)

	configPath := filepath.Join(t.TempDir(), "mmt.yaml")
	err = os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(connStr))
	require.NoError(t, err)
	t.Cleanup(func() { client.Disconnect(context.Background()) })

	t.Setenv("MONGO_URL", connStr)
	t.Setenv("MONGO_DATABASE", dbName)
	t.Setenv("MIGRATIONS_COLLECTION", colName)

	return &TestEnv{
		ConfigPath:  configPath,
		DBName:      dbName,
		ColName:     colName,
		MongoClient: client,
	}
}

func (e *TestEnv) RunCLI(t *testing.T, args ...string) string {
	t.Helper()
	oldArgs := os.Args
	os.Args = append([]string{"mmt", "--config", e.ConfigPath}, args...)
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
		io.Copy(&b, rOut)
		resOut <- b.String()
	}()

	go func() {
		var b bytes.Buffer
		io.Copy(&b, rErr)
		resErr <- b.String()
	}()

	fErr := <-errChan
	wOut.Close()
	wErr.Close()

	stdout, stderr := <-resOut, <-resErr
	os.Stdout, os.Stderr = oldOut, oldErr
	return stdout, stderr, fErr
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

func sortedMigrationVersions() []string {
	m := migration.RegisteredMigrations()
	v := make([]string, 0, len(m))
	for k := range m {
		v = append(v, k)
	}
	sort.Strings(v)
	return v
}
