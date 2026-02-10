package cli

import (
	"context"
	"fmt"

	"github.com/drewjocham/mongo-migration-tool/internal/config"
	"github.com/drewjocham/mongo-migration-tool/internal/migration"
)

type ctxKey string

const (
	ctxServicesKey ctxKey = "services"
	ctxConfigKey   ctxKey = "config"
	ctxEngineKey   ctxKey = "engine"
)

func getServices(ctx context.Context) (*Services, error) {
	s, ok := ctx.Value(ctxServicesKey).(*Services)
	if !ok {
		return nil, fmt.Errorf("unable to retrieve services from context (check if command is running offline)")
	}
	return s, nil
}

func getEngine(ctx context.Context) (*migration.Engine, error) {
	e, ok := ctx.Value(ctxEngineKey).(*migration.Engine)
	if !ok {
		return nil, fmt.Errorf("migration engine not found in context")
	}
	return e, nil
}

func getConfig(ctx context.Context) (*config.Config, error) {
	cfg, ok := ctx.Value(ctxConfigKey).(*config.Config)
	if !ok {
		return nil, fmt.Errorf("config not found in context")
	}
	return cfg, nil
}
