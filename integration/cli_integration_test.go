//go:build integration

package integration_test

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

	"github.com/jocham/mongo-migration/migration"
	_ "github.com/jocham/mongo-migration/migrations"
)

var (
	projectRoot         = determineProjectRoot()
	dockerConfigDirPath = setupDockerConfigDir()
)

func TestCLIDockerCommands(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping CLI integration test in short mode")
	}

	requireDocker(t)

	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Minute)
	defer cancel()

	hostPort := getFreePort(t)
	projectName := fmt.Sprintf("mm-cli-it-%d", time.Now().UnixNano())
	composeEnv := append(os.Environ(),
		fmt.Sprintf("COMPOSE_PROJECT_NAME=%s", projectName),
		fmt.Sprintf("INTEGRATION_MONGO_PORT=%d", hostPort),
	)

	composeBuild(ctx, t, composeEnv, "cli")
	composeUp(ctx, t, composeEnv, "mongo")
	t.Cleanup(func() {
		composeDown(t, composeEnv)
	})

	waitForMongo(ctx, t, hostPort)

	versions := sortedMigrationVersions()
	if len(versions) == 0 {
		t.Fatal("no registered migrations found")
	}
	earliestVersion := versions[0]
	latestVersion := versions[len(versions)-1]

	statusBefore := runCLICommand(ctx, t, composeEnv, "status")
	requireVersionState(t, statusBefore, latestVersion, "[ ]")

	upOutput := runCLICommand(ctx, t, composeEnv, "up")
	if !strings.Contains(upOutput, "Database is up to date") {
		t.Fatalf("expected successful up output, got:\n%s", upOutput)
	}

	statusAfterUp := runCLICommand(ctx, t, composeEnv, "status")
	requireVersionState(t, statusAfterUp, latestVersion, "[✓]")

	runCLICommand(ctx, t, composeEnv, "down", "--target", earliestVersion)

	statusAfterDown := runCLICommand(ctx, t, composeEnv, "status")
	requireVersionState(t, statusAfterDown, latestVersion, "[ ]")

	versionOutput := runCLICommand(ctx, t, composeEnv, "version")
	if !strings.Contains(strings.ToLower(versionOutput), "version") {
		t.Fatalf("expected version output, got:\n%s", versionOutput)
	}
}

func requireDocker(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skipf("docker not found: %v", err)
	}
	cmd := exec.Command("docker", "version", "--format", "{{.Server.Version}}")
	if err := cmd.Run(); err != nil {
		t.Skipf("docker daemon not available: %v", err)
	}
}

func runDockerCmd(ctx context.Context, t *testing.T, args ...string) string {
	t.Helper()
	commandCtx, cancel := context.WithTimeout(ctx, 4*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(commandCtx, "docker", args...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("DOCKER_CONFIG=%s", dockerConfigDirPath))
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		t.Fatalf("docker %s failed: %v\nOutput:\n%s", strings.Join(args, " "), err, buf.String())
	}
	return buf.String()
}

func dockerComposeCmd(ctx context.Context, t *testing.T, env []string, args ...string) string {
	t.Helper()
	commandCtx, cancel := context.WithTimeout(ctx, 6*time.Minute)
	defer cancel()
	composeArgs := append([]string{"-f", filepath.Join(projectRoot, "integration-compose.yml")}, args...)
	cmd := exec.CommandContext(commandCtx, "docker-compose", composeArgs...)
	cmd.Env = append(env, fmt.Sprintf("DOCKER_CONFIG=%s", dockerConfigDirPath))

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		t.Fatalf("docker %s failed: %v\nOutput:\n%s", strings.Join(composeArgs, " "), err, buf.String())
	}
	return buf.String()
}

func composeBuild(ctx context.Context, t *testing.T, env []string, services ...string) {
	args := append([]string{"build"}, services...)
	dockerComposeCmd(ctx, t, env, args...)
}

func composeUp(ctx context.Context, t *testing.T, env []string, services ...string) {
	args := append([]string{"up", "-d"}, services...)
	dockerComposeCmd(ctx, t, env, args...)
}

func composeDown(t *testing.T, env []string) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	dockerComposeCmd(ctx, t, env, "down", "-v")
}

func waitForMongo(ctx context.Context, t *testing.T, port int) {
	t.Helper()
	uri := fmt.Sprintf("mongodb://admin:password@localhost:%d/?authSource=admin", port)
	deadline := time.Now().Add(2 * time.Minute)
	for time.Now().Before(deadline) {
		client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
		if err == nil {
			pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			err = client.Ping(pingCtx, nil)
			cancel()
			_ = client.Disconnect(context.Background())
			if err == nil {
				return
			}
		}
		time.Sleep(3 * time.Second)
	}
	t.Fatalf("mongo not ready at %s", uri)
}

func runCLICommand(ctx context.Context, t *testing.T, env []string, cliArgs ...string) string {
	t.Helper()
	dockerArgs := append([]string{"run", "--rm", "cli"}, cliArgs...)
	return dockerComposeCmd(ctx, t, env, dockerArgs...)
}

func getFreePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to get free port: %v", err)
	}
	defer func() { _ = l.Close() }()
	return l.Addr().(*net.TCPAddr).Port
}

func sortedMigrationVersions() []string {
	regs := migration.RegisteredMigrations()
	versions := make([]string, 0, len(regs))
	for version := range regs {
		versions = append(versions, version)
	}
	sort.Strings(versions)
	return versions
}

func requireVersionState(t *testing.T, statusOutput, version, marker string) {
	t.Helper()
	for _, line := range strings.Split(statusOutput, "\n") {
		if strings.Contains(line, version) {
			if strings.Contains(line, marker) {
				return
			}
			t.Fatalf("expected %s to contain %s; line=%q", version, marker, line)
		}
	}
	t.Fatalf("version %s not found in status output:\n%s", version, statusOutput)
}

func determineProjectRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("unable to determine caller info for integration tests")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), ".."))
}

func setupDockerConfigDir() string {
	dir, err := os.MkdirTemp("", "mongo-migration-docker-config-")
	if err != nil {
		panic(fmt.Sprintf("unable to create temp docker config dir: %v", err))
	}
	configPath := filepath.Join(dir, "config.json")
	if err := os.WriteFile(configPath, []byte(`{"auths":{}}`), 0o600); err != nil {
		panic(fmt.Sprintf("unable to write docker config: %v", err))
	}
	return dir
}
