package main

import (
	"errors"
	"fmt"
	"os"

	"github.com/drewjocham/mongo-migration-tool/internal/cli"
)

func main() {
	if err := cli.Execute(); err != nil {
		if errors.Is(err, cli.ErrShowConfigDisplayed) {
			return
		}
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
