package cli

import (
	"fmt"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the current migration status",
	RunE:  runStatus,
}

func runStatus(cmd *cobra.Command, _ []string) error {
	engine, err := getEngine(cmd.Context())
	if err != nil {
		return err
	}

	stats, err := engine.GetStatus(cmd.Context())
	if err != nil {
		return fmt.Errorf("failed to get status: %w", err)
	}

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 3, ' ', 0)
	fmt.Fprintln(w, "VERSION\tSTATUS\tDESCRIPTION")

	for _, s := range stats {
		statusText := "pending"
		if s.Applied {
			statusText = "applied"
		}
		fmt.Fprintf(w, "%s\t[%s]\t%s\n", s.Version, statusText, s.Description)
	}

	return w.Flush()
}
