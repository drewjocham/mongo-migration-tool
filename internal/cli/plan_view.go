package cli

import (
	"fmt"
	"io"
)

func renderPlan(out io.Writer, direction string, plan []string) {
	if len(plan) == 0 {
		fmt.Fprintf(out, "No migrations to %s.\n", direction)
		return
	}

	fmt.Fprintf(out, "Planned migrations to %s:\n", direction)
	for i, version := range plan {
		fmt.Fprintf(out, "  %02d. %s\n", i+1, version)
	}
}
