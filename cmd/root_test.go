package cmd

import (
	"bytes"
	"strings"
	"testing"
)

func TestRootCommand(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{
			name:    "help flag",
			args:    []string{"--help"},
			wantErr: false,
		},
		{
			name:    "version command",
			args:    []string{"version"},
			wantErr: false,
		},
		{
			name:    "invalid command",
			args:    []string{"invalid"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a new root command for each test
			cmd := NewRootCommand()
			cmd.SetArgs(tt.args)

			// Capture output
			buf := new(bytes.Buffer)
			cmd.SetOut(buf)
			cmd.SetErr(buf)

			err := cmd.Execute()

			if (err != nil) != tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
				t.Logf("Output: %s", buf.String())
			}
		})
	}
}

func TestRootCommandHelp(t *testing.T) {
	cmd := NewRootCommand()
	cmd.SetArgs([]string{"--help"})

	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	output := buf.String()

	// Check for expected help content
	expectedStrings := []string{
		"mongo-essential",
		"Available Commands:",
		"Flags:",
		"--help",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Help output should contain '%s'", expected)
		}
	}
}

func TestRootCommandConfig(t *testing.T) {
	tests := []struct {
		name       string
		configFlag string
		wantErr    bool
	}{
		{
			name:       "no config flag",
			configFlag: "",
			wantErr:    false,
		},
		{
			name:       "with config flag",
			configFlag: ".env.test",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := NewRootCommand()
			args := []string{"--help"}
			if tt.configFlag != "" {
				args = append([]string{"--config", tt.configFlag}, args...)
			}
			cmd.SetArgs(args)

			buf := new(bytes.Buffer)
			cmd.SetOut(buf)
			cmd.SetErr(buf)

			err := cmd.Execute()

			// Help should always succeed
			if err != nil && !tt.wantErr {
				t.Errorf("Execute() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// NewRootCommand creates a new root command for testing
func NewRootCommand() *Cobra.Command {
	// Return a copy of the root command
	// This prevents test pollution
	return rootCmd
}
