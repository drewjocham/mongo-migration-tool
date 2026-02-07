//go:build integration

package integration_tests_test

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/drewjocham/mongo-migration-tool/migration"
	_ "github.com/drewjocham/mongo-migration-tool/migrations"
)

type testEnv struct {
	projectRoot     string
	dockerConfigDir string
	projectName     string
	mongoPort       int
	env             []string
}

func TestCLIDockerCommands(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping CLI integration test in short mode")
	}

	requireDocker(t)
	env := setupTestEnv(t)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	t.Log("Building and starting Docker environment...")
	env.compose(ctx, t, "build", "cli")
	env.compose(ctx, t, "up", "-d", "mongo")
	t.Cleanup(func() { env.compose(context.Background(), t, "down", "-v") })

	env.waitForMongo(ctx, t)

	versions := sortedMigrationVersions()
	if len(versions) == 0 {
		t.Fatal("no registered migrations found")
	}
	earliest, latest := versions[0], versions[len(versions)-1]

	t.Run("migration lifecycle", func(t *testing.T) {
		status := env.runCLI(ctx, t, "status")
		requireVersionState(t, status, latest, "[ ]")

		// Migrate Up
		upOut := env.runCLI(ctx, t, "up")
		if !strings.Contains(upOut, "Database is up to date") {
			t.Errorf("unexpected 'up' output: %s", upOut)
		}

		status = env.runCLI(ctx, t, "status")
		requireVersionState(t, status, latest, "[✓]")

		// Rollback
		env.runCLI(ctx, t, "down", "--target", earliest)
		status = env.runCLI(ctx, t, "status")
		requireVersionState(t, status, latest, "[ ]")

		// Check Version Command
		versionOut := env.runCLI(ctx, t, "version")
		if !strings.Contains(strings.ToLower(versionOut), "version") {
			t.Errorf("invalid version output: %s", versionOut)
		}
	})
}

func setupTestEnv(t *testing.T) *testEnv {
	_, file, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(file), ".."))

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")
	_ = os.WriteFile(configPath, []byte(`{"auths":{}}`), 0600)

	port := getFreePort(t)
	projName := fmt.Sprintf("mm-it-%d", time.Now().UnixNano())

	return &testEnv{
		projectRoot:     root,
		dockerConfigDir: tmpDir,
		projectName:     projName,
		mongoPort:       port,
		env: append(os.Environ(),
			fmt.Sprintf("COMPOSE_PROJECT_NAME=%s", projName),
			fmt.Sprintf("INTEGRATION_MONGO_PORT=%d", port),
			fmt.Sprintf("DOCKER_CONFIG=%s", tmpDir),
		),
	}
}

func (e *testEnv) compose(ctx context.Context, t *testing.T, args ...string) string {
	t.Helper()
	composeFile := filepath.Join(e.projectRoot, "docker/integration-compose.yml")
	fullArgs := append([]string{"-f", composeFile}, args...)
	return e.execCmd(ctx, t, "docker-compose", fullArgs...)
}

func (e *testEnv) runCLI(ctx context.Context, t *testing.T, cliArgs ...string) string {
	t.Helper()
	args := append([]string{"run", "--rm", "cli"}, cliArgs...)
	return e.compose(ctx, t, args...)
}

func (e *testEnv) execCmd(ctx context.Context, t *testing.T, name string, args ...string) string {
	t.Helper()
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Env = e.env

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	combined := stdout.String() + stderr.String()

	if err != nil {
		t.Logf("CLI STDERR: %s", stderr.String())
		t.Fatalf("command [%s %s] failed: %v\nOutput: %s", name, strings.Join(args, " "), err, combined)
	}
	return stdout.String()
}

func (e *testEnv) waitForMongo(ctx context.Context, t *testing.T) {
	t.Helper()
	uri := fmt.Sprintf("mongodb://admin:password@localhost:%d/?authSource=admin", e.mongoPort)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			t.Fatalf("timed out waiting for mongo at %s: %v", uri, ctx.Err())
		case <-ticker.C:
			client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
			if err != nil {
				continue
			}
			err = client.Ping(ctx, nil)
			_ = client.Disconnect(ctx)
			if err == nil {
				return
			}
		}
	}
}

func requireVersionState(t *testing.T, output, version, marker string) {
	t.Helper()
	for _, line := range strings.Split(output, "\n") {
		if strings.Contains(line, version) {
			if !strings.Contains(line, marker) {
				t.Fatalf("version %s state mismatch: expected %s, got line: %q", version, marker, line)
			}
			return
		}
	}
	t.Fatalf("version %s not found in output", version)
}

func getFreePort(t *testing.T) int {
	l, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func requireDocker(t *testing.T) {
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("Docker not found")
	}
}

func sortedMigrationVersions() []string {
	regs := migration.RegisteredMigrations()
	versions := make([]string, 0, len(regs))
	for v := range regs {
		versions = append(versions, v)
	}
	sort.Strings(versions)
	return versions
}
