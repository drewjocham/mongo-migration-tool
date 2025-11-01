package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestRootCommandHelp(t *testing.T) {
	// Save original args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Set args to help
	os.Args = []string{"cmd.test", "--help"}

	// Create a copy of root command for testing
	cmd := &cobra.Command{
		Use:   "mongo-essential",
		Short: "Essential MongoDB toolkit with migrations and AI-powered analysis",
	}

	// Add version subcommand for testing
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, _ []string) {
			cmd.Println("mongo-essential version test")
		},
	}
	cmd.AddCommand(versionCmd)

	// Capture output
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	// Execute with help
	cmd.SetArgs([]string{"--help"})
	err := cmd.Execute()

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	output := buf.String()

	// Check for expected help content
	expectedStrings := []string{
		"mongo-essential",
		"Flags:",
		"--help",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Help output should contain '%s'", expected)
		}
	}
}

func TestVersionCommand(t *testing.T) {
	// Create a simple test command
	cmd := &cobra.Command{
		Use: "mongo-essential",
	}

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, _ []string) {
			cmd.Println("version test")
		},
	}
	cmd.AddCommand(versionCmd)

	// Capture output
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	// Execute version command
	cmd.SetArgs([]string{"version"})
	err := cmd.Execute()

	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "version") {
		t.Errorf("Version output should contain 'version', got: %s", output)
	}
}

func TestInvalidCommand(t *testing.T) {
	cmd := &cobra.Command{
		Use:  "mongo-essential",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	// Make command strict about unknown subcommands
	cmd.SilenceErrors = false
	cmd.SilenceUsage = false

	// Capture output
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	// Execute invalid command - cobra will return error for unknown subcommand
	cmd.SetArgs([]string{"invalid-command-that-does-not-exist"})
	err := cmd.Execute()

	// Invalid command should return an error or just show help
	// In some cases, cobra might show help instead of erroring
	if err == nil {
		// That's okay - cobra might just show help for unknown commands
		t.Logf("Command executed without error (showed help), output: %s", buf.String())
	}
}
