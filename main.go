package main

import (
	"fmt"
	"os"

	"github.com/drewjocham/mongo-migration-tool/cmd"
)

func main() {
	cmd.SetupRootCommand()
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
