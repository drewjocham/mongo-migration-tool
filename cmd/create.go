package cmd

import (
	"bytes"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/spf13/cobra"
)

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:   "create [migration_name]",
	Short: "Create a new migration file",
	Long: `Create a new migration file with a timestamp prefix.
The file will include boilerplate code and an init() function for auto-registration.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		migrationName := args[0]
		timestamp := time.Now().Format("20060102_150405")

		cleanName := strings.NewReplacer(" ", "_", "-", "_").Replace(strings.ToLower(migrationName))
		version := fmt.Sprintf("%s_%s", timestamp, cleanName)
		filename := version + ".go"

		if err := os.MkdirAll(cfg.MigrationsPath, 0750); err != nil {
			slog.Error("Failed to create migrations directory", "path", cfg.MigrationsPath, "error", err)
			return err
		}

		targetPath := filepath.Join(cfg.MigrationsPath, filename)
		if _, err := os.Stat(targetPath); err == nil {
			return fmt.Errorf("migration file already exists: %s", targetPath)
		}

		data := struct {
			Version     string
			Description string
			StructName  string
		}{
			Version:     version,
			Description: migrationName,
			StructName:  "Migration_" + version,
		}

		tmpl, err := template.New("migration").Parse(migrationTemplate)
		if err != nil {
			return fmt.Errorf("failed to parse template: %w", err)
		}

		var buf bytes.Buffer
		if err := tmpl.Execute(&buf, data); err != nil {
			return fmt.Errorf("failed to execute template: %w", err)
		}

		if err := os.WriteFile(targetPath, buf.Bytes(), 0600); err != nil {
			slog.Error("Failed to write migration file", "file", targetPath, "error", err)
			return err
		}

		slog.Info("Created migration file", "path", targetPath)
		printNextSteps(targetPath, version)

		return nil
	},
}

// printNextSteps provides user-friendly feedback
func printNextSteps(path, version string) {
	fmt.Printf("\n✓ Created migration: %s\n", path)
	fmt.Println("\nNext steps:")
	fmt.Printf("  1. Open %s\n", path)
	fmt.Println("  2. Implement Up() and Down() logic")
	fmt.Printf("  3. Ensure 'import _ \"yourproject/path/%s\"' is in your main file\n", filepath.Dir(path))
	fmt.Printf("  4. Run 'mongo-essential up --target %s'\n\n", version)
}

// migrationTemplate uses Go template
const migrationTemplate = `package migrations

import (
	"context"
	"log/slog"

	"github.com/drewjocham/mongo-migration-tool-/migration"
	"go.mongodb.org/mongo-driver/mongo"
)

func init() {
	migration.Register(&{{.StructName}}{})
}

type {{.StructName}} struct{}

func (m *{{.StructName}}) Version() string {
	return "{{.Version}}"
}

func (m *{{.StructName}}) Description() string {
	return "{{.Description}}"
}

func (m *{{.StructName}}) Up(ctx context.Context, db *mongo.Database) error {
	slog.Info("Running migration UP", "version", m.Version())
	// TODO: collection := db.Collection("example")
	return nil
}

func (m *{{.StructName}}) Down(ctx context.Context, db *mongo.Database) error {
	slog.Info("Running migration DOWN", "version", m.Version())
	// TODO: Rollback logic
	return nil
}
`
