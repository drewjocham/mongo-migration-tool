package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

type Config struct {
	MongoURL             string `env:"MONGO_URL" envDefault:"mongodb://localhost:27017"`
	Database             string `env:"MONGO_DATABASE,required"`
	MigrationsPath       string `env:"MIGRATIONS_PATH" envDefault:"./migrations"`
	MigrationsCollection string `env:"MIGRATIONS_COLLECTION" envDefault:"schema_migrations"`
	Username             string `env:"MONGO_USERNAME"`
	Password             string `env:"MONGO_PASSWORD"`
	MongoAuthSource      string `env:"MONGO_AUTH_SOURCE" envDefault:"admin"`
	SSLEnabled           bool   `env:"MONGO_SSL_ENABLED" envDefault:"false"`
	SSLInsecure          bool   `env:"MONGO_SSL_INSECURE" envDefault:"false"`
	MaxPoolSize          int    `env:"MONGO_MAX_POOL_SIZE" envDefault:"10"`
	MinPoolSize          int    `env:"MONGO_MIN_POOL_SIZE" envDefault:"1"`
	Timeout              int    `env:"MONGO_TIMEOUT" envDefault:"60"`

	GoogleDocsEnabled     bool   `env:"GOOGLE_DOCS_ENABLED" envDefault:"false"`
	GoogleCredentialsPath string `env:"GOOGLE_CREDENTIALS_PATH"`
	GoogleCredentialsJSON string `env:"GOOGLE_CREDENTIALS_JSON"`
}

func Load(envFiles ...string) (*Config, error) {
	for _, file := range envFiles {
		if _, err := os.Stat(file); err == nil {
			_ = godotenv.Load(file)
		}
	}

	cfg := &Config{}
	if err := env.Parse(cfg); err != nil {
		return nil, fmt.Errorf("env parse error: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) GetConnectionString() string {
	u, err := url.Parse(c.MongoURL)
	if err != nil {
		return c.MongoURL
	}

	if c.Username != "" && u.User == nil {
		u.User = url.UserPassword(c.Username, c.Password)
	}

	q := u.Query()

	if strings.Contains(u.Host, "localhost") && q.Get("connect") == "" {
		q.Set("connect", "direct")
	}

	if c.MongoAuthSource != "" && q.Get("authSource") == "" {
		q.Set("authSource", c.MongoAuthSource)
	}

	if c.SSLEnabled && q.Get("ssl") == "" {
		q.Set("ssl", "true")
	}

	u.RawQuery = q.Encode()
	return u.String()
}

func (c *Config) Validate() error {
	if c.Database == "" {
		return fmt.Errorf("MONGO_DATABASE is required")
	}
	if c.GoogleDocsEnabled {
		if c.GoogleCredentialsPath == "" && c.GoogleCredentialsJSON == "" {
			return fmt.Errorf("google Docs enabled but credentials missing")
		}
	}
	return nil
}
