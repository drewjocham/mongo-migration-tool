package cli

import (
	"fmt"
	"path/filepath"

	"github.com/drewjocham/mongo-migration-tool/internal/migration"
	"github.com/spf13/cobra"
)

func newCreateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:         "create [migration_name]",
		Short:       "Create a new migration file",
		Args:        cobra.ExactArgs(1),
		Annotations: map[string]string{annotationOffline: "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := getConfig(cmd.Context())
			if err != nil {
				return err
			}

			gen := &migration.Generator{
				OutputPath: cfg.MigrationsPath,
			}

			path, version, err := gen.Create(args[0])
			if err != nil {
				return err
			}

			renderSuccess(path, version)
			return nil
		},
	}

	return cmd
}

func renderSuccess(path, version string) {
	displayPath := path
	if rel, err := filepath.Rel(".", path); err == nil {
		displayPath = rel
	}

	fmt.Printf("\n✨ Migration created: %s\n", displayPath)
	fmt.Printf("\nNext steps:\n")
	fmt.Printf("  1. Edit logic: code %s\n", displayPath)
	fmt.Printf("  2. Test run:   mt up --target %s\n\n", version)
}
