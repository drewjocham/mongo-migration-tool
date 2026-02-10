package cli

import (
	"bufio"
	"context"
	"fmt"
	"strings"

	"github.com/drewjocham/mongo-migration-tool/internal/config"
	"github.com/drewjocham/mongo-migration-tool/migration"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
)

type cliCtxKey string

const (
	ctxEngineKey   cliCtxKey = "engine"
	ctxConfigKey   cliCtxKey = "config"
	ctxServicesKey cliCtxKey = "services"
)

func getEngine(ctx context.Context) (*migration.Engine, error) {
	e, ok := ctx.Value(ctxEngineKey).(*migration.Engine)
	if !ok {
		return nil, fmt.Errorf("internal error: migration engine not found in context")
	}
	return e, nil
}

func getConfig(ctx context.Context) (*config.Config, error) {
	cfg, ok := ctx.Value(ctxConfigKey).(*config.Config)
	if !ok {
		return nil, fmt.Errorf("internal error: config not found in context")
	}
	return cfg, nil
}

func getServices(ctx context.Context) (*Services, error) {
	s, ok := ctx.Value(ctxServicesKey).(*Services)
	if !ok || s == nil {
		return nil, fmt.Errorf("internal error: services not found in context")
	}
	return s, nil
}

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
