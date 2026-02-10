package cli

import (
	"bufio"
	"fmt"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"strings"
)

func promptConfirmation(cmd *cobra.Command, message string) bool {
	fmt.Fprint(cmd.OutOrStdout(), message)

	reader := bufio.NewReader(cmd.InOrStdin())
	input, err := reader.ReadString('\n')
	if err != nil {
		zap.S().Errorw("Failed to read confirmation", "error", err)
		return false
	}

	response := strings.ToLower(strings.TrimSpace(input))
	return response == "y" || response == "yes"
}

