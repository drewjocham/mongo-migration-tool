package cli

import (
	"fmt"
	"io"

	"github.com/drewjocham/mongo-migration-tool/internal/config"
	"github.com/drewjocham/mongo-migration-tool/internal/jsonutil"
)

type safeConfig struct {
	MongoURL             string `json:"mongo_url"`
	Database             string `json:"database"`
	MigrationsPath       string `json:"migrations_path"`
	MigrationsCollection string `json:"migrations_collection"`
	Username             string `json:"username"`
	Password             string `json:"password"`
	AuthSource           string `json:"auth_source"`
	SSLEnabled           bool   `json:"ssl_enabled"`
	SSLInsecure          bool   `json:"ssl_insecure"`
	MaxPoolSize          int    `json:"max_pool_size"`
	MinPoolSize          int    `json:"min_pool_size"`
	TimeoutSeconds       int    `json:"timeout_seconds"`
	GoogleDocsEnabled    bool   `json:"google_docs_enabled"`
	GoogleCredentials    string `json:"google_credentials"`
}

func renderConfig(out io.Writer, cfg *config.Config) error {
	enc := jsonutil.NewEncoder(out)
	enc.SetIndent("", "  ")
	safe := safeConfig{
		MongoURL:             cfg.MongoURL,
		Database:             cfg.Database,
		MigrationsPath:       cfg.MigrationsPath,
		MigrationsCollection: cfg.MigrationsCollection,
		Username:             cfg.Username,
		Password:             maskSecret(cfg.Password),
		AuthSource:           cfg.MongoAuthSource,
		SSLEnabled:           cfg.SSLEnabled,
		SSLInsecure:          cfg.SSLInsecure,
		MaxPoolSize:          cfg.MaxPoolSize,
		MinPoolSize:          cfg.MinPoolSize,
		TimeoutSeconds:       cfg.Timeout,
		GoogleDocsEnabled:    cfg.GoogleDocsEnabled,
		GoogleCredentials:    maskSecret(firstNonEmpty(cfg.GoogleCredentialsPath, cfg.GoogleCredentialsJSON)),
	}
	if err := enc.Encode(safe); err != nil {
		return fmt.Errorf("render config: %w", err)
	}
	return nil
}

func maskSecret(value string) string {
	if value == "" {
		return ""
	}
	if len(value) <= 4 {
		return "***"
	}
	return fmt.Sprintf("%s***%s", value[:2], value[len(value)-2:])
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
