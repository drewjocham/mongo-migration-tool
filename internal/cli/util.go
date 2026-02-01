package cli

import (
	"context"
	"fmt"

	"github.com/drewjocham/mongo-migration-tool/internal/config"
	"github.com/drewjocham/mongo-migration-tool/migration"
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
