package main

import (
	"fmt"
	"os"

	"github.com/drewjocham/mongo-migration-tool/internal/cli"
)

func main() {
	if err := cli.NewMCPCmd().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
