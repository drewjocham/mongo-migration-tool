package main

import (
	"log"
	"os"

	"github.com/jocham/mongo-essential/cmd"
	_ "github.com/jocham/mongo-essential/examples/examplemigrations"
)

// Version information set by GoReleaser
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	cmd.SetVersion(version, commit, date)
	cmd.SetupRootCommand()

	if err := cmd.Execute(); err != nil {
		log.Printf("Error: %v", err)
		os.Exit(1)
	}
}
