package cmd

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

var (
	forceYes bool

	forceCmd = func() *cobra.Command {
		cmd := &cobra.Command{
			Use:   "force [version]",
			Short: "Force mark a migration as applied without running it",
			Long: `Force mark a specific migration as applied in the database without 
					actually executing its Up() logic. This updates the migration tracking 
					collection to prevent the engine from trying to run it again.`,
			Args: cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				version := args[0]
				ctx := cmd.Context()

				if !forceYes {
					slog.Warn("DANGER: Force marking migration", "version", version)
					fmt.Printf("WARNING: You are about to force mark migration %s as applied.\n", version)
					fmt.Println("This will NOT execute the migration logic.")
					fmt.Print("Are you sure you want to continue? (y/N): ")

					reader := bufio.NewReader(os.Stdin)
					response, _ := reader.ReadString('\n')
					response = strings.ToLower(strings.TrimSpace(response))

					if response != "y" && response != "yes" {
						slog.Info("Operation cancelled by user")
						return nil
					}
				}

				if err := engine.Force(ctx, version); err != nil {
					slog.Error("Failed to force mark migration",
						"version", version, "error", err)
					return err
				}

				slog.Info("Migration force marked successfully",
					"version", version)
				return nil
			},
		}

		cmd.Flags().BoolVarP(&forceYes, "yes", "y", false,
			"Confirm the action without prompting")

		return cmd
	}()
)
