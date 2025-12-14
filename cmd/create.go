// Package cmd provides command-line interface commands for the mongo-essential tool.
package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// createCmd represents the create command
var createCmd = &cobra.Command{
	Use:   "create [migration_name]",
	Short: "Create a new migration file",
	Long: `Create a new migration file with the specified name.

The migration file will be created in the migrations directory with a timestamp
prefix and will contain template code implementing the Migration interface.

Examples:
  mongo-essential create add_user_index
  mongo-essential create "Create product collection"`,
	Args: cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		migrationName := args[0]

		timestamp := time.Now().Format("20060102_150405")

		cleanName := strings.ReplaceAll(strings.ToLower(migrationName), " ", "_")
		cleanName = strings.ReplaceAll(cleanName, "-", "_")

		version := fmt.Sprintf("%s_%s", timestamp, cleanName)
		filename := fmt.Sprintf("%s.go", version)

		// Ensure migrations directory exists
		const dirPermissions = 0750
		if err := os.MkdirAll(cfg.MigrationsPath, dirPermissions); err != nil {
			return fmt.Errorf("failed to create migrations directory: %w", err)
		}
		filepath := filepath.Join(cfg.MigrationsPath, filename)

		if _, err := os.Stat(filepath); err == nil {
			return fmt.Errorf("migration file already exists: %s", filepath)
		}

		// Create migration file content
		content := fmt.Sprintf(`package migrations

import (
	"context"
	"fmt"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Migration_%s %s
type Migration_%s struct{}

// Version returns the unique version identifier for this migration
func (m *Migration_%s) Version() string {
	return "%s"
}

// Description returns a human-readable description of what this migration does
func (m *Migration_%s) Description() string {
	return "%s"
}

// Up executes the migration
func (m *Migration_%s) Up(ctx context.Context, db *mongo.Database) error {
	// TODO: Implement your migration logic here
	// Example:
	// collection := db.Collection("your_collection")
	// 
	// // Create indexes, insert data, etc.
	// index := mongo.IndexModel{
	//     Keys:    bson.D{{"field_name", 1}},
	//     Options: options.Index().SetName("field_name_idx"),
	// }
	// _, err := collection.Indexes().CreateOne(ctx, index)
	// return err

	fmt.Printf("Migration %%s: %%s - UP\\n", m.Version(), m.Description())
	return nil
}

// Down rolls back the migration
func (m *Migration_%s) Down(ctx context.Context, db *mongo.Database) error {
	// TODO: Implement rollback logic here
	// Example:
	// collection := db.Collection("your_collection")
	// _, err := collection.Indexes().DropOne(ctx, "field_name_idx")
	// return err

	fmt.Printf("Migration %%s: %%s - DOWN\\n", m.Version(), m.Description())
	return nil
}
`,
			version, migrationName,
			version,
			version, version,
			version, migrationName,
			version,
			version)

		// Write the file
		const filePermissions = 0600
		if err := os.WriteFile(filepath, []byte(content), filePermissions); err != nil {
			return fmt.Errorf("failed to write migration file: %w", err)
		}

		fmt.Printf("✓ Created migration file: %s\n", filepath)
		fmt.Printf("  Version: %s\n", version)
		fmt.Printf("  Description: %s\n", migrationName)
		fmt.Println()
		fmt.Println("Next steps:")
		fmt.Println("1. Edit the migration file to implement your Up() and Down() methods")
		fmt.Println("2. Register the migration in your application")
		fmt.Println("3. Run 'mongo-essential status' to see the new migration")
		fmt.Println("4. Run 'mongo-essential up' to apply the migration")

		return nil
	},
}
