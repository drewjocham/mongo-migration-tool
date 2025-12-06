// Package config provides configuration management for mongo-essential.
package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

// Config holds all configuration options for mongo-essential.
type Config struct {
	MongoURL string `env:"MONGO_URL" envDefault:"mongodb://localhost:27017"`
	Database string `env:"MONGO_DATABASE,required"`

	MigrationsPath       string `env:"MIGRATIONS_PATH" envDefault:"./migrations"`
	MigrationsCollection string `env:"MIGRATIONS_COLLECTION" envDefault:"schema_migrations"`

	Username string `env:"MONGO_USERNAME"`
	Password string `env:"MONGO_PASSWORD"`

	// SSL/TLS settings - important for cloud providers like STACKIT
	SSLEnabled           bool   `env:"MONGO_SSL_ENABLED" envDefault:"false"`
	SSLInsecure          bool   `env:"MONGO_SSL_INSECURE" envDefault:"false"`
	SSLCertificatePath   string `env:"MONGO_SSL_CERT_PATH"`
	SSLPrivateKeyPath    string `env:"MONGO_SSL_KEY_PATH"`
	SSLCACertificatePath string `env:"MONGO_SSL_CA_CERT_PATH"`

	// Connection pool settings
	MaxPoolSize int `env:"MONGO_MAX_POOL_SIZE" envDefault:"10"`
	MinPoolSize int `env:"MONGO_MIN_POOL_SIZE" envDefault:"1"`
	MaxIdleTime int `env:"MONGO_MAX_IDLE_TIME" envDefault:"300"` // in seconds

	// Connection timeout
	Timeout int `env:"MONGO_TIMEOUT" envDefault:"60"`

	// AI Analysis settings
	AIProvider string `env:"AI_PROVIDER" envDefault:"openai"`
	AIEnabled  bool   `env:"AI_ENABLED" envDefault:"false"`

	// OpenAI settings
	OpenAIAPIKey string `env:"OPENAI_API_KEY"`
	OpenAIModel  string `env:"OPENAI_MODEL" envDefault:"gpt-4o-mini"`

	// Google Gemini settings
	GeminiAPIKey string `env:"GEMINI_API_KEY"`
	GeminiModel  string `env:"GEMINI_MODEL" envDefault:"gemini-1.5-flash"`

	// Claude settings
	ClaudeAPIKey string `env:"CLAUDE_API_KEY"`
	ClaudeModel  string `env:"CLAUDE_MODEL" envDefault:"claude-3-5-sonnet-20241022"`

	// Google Docs Integration
	GoogleDocsEnabled        bool   `env:"GOOGLE_DOCS_ENABLED" envDefault:"false"`
	GoogleCredentialsPath    string `env:"GOOGLE_CREDENTIALS_PATH"`
	GoogleCredentialsJSON    string `env:"GOOGLE_CREDENTIALS_JSON"`
	GoogleDriveFolderID      string `env:"GOOGLE_DRIVE_FOLDER_ID"`
	GoogleDocsTemplate       string `env:"GOOGLE_DOCS_TEMPLATE" envDefault:"analysis"`
	GoogleDocsShareWithEmail string `env:"GOOGLE_DOCS_SHARE_WITH_EMAIL"`
}

func Load(envFiles ...string) (*Config, error) {
	for _, file := range envFiles {
		if _, err := os.Stat(file); err == nil {
			if err := godotenv.Load(file); err != nil {
				return nil, fmt.Errorf("failed to load env file %s: %w", file, err)
			}
		}
	}

	config := &Config{}
	if err := env.Parse(config); err != nil {
		return nil, fmt.Errorf("failed to parse environment variables: %w", err)
	}

	return config, nil
}

// LoadFromEnv loads configuration only from environment variables, ignoring .env files.
func LoadFromEnv() (*Config, error) {
	config := &Config{}
	if err := env.Parse(config); err != nil {
		return nil, fmt.Errorf("failed to parse environment variables: %w", err)
	}

	return config, nil
}

// GetConnectionString builds the MongoDB connection string with authentication.
func (c *Config) GetConnectionString() string {
	connStr := c.MongoURL

	if c.Username != "" && c.Password != "" {
		if c.MongoURL == "mongodb://localhost:27017" ||
			(c.MongoURL != "" && !strings.Contains(c.MongoURL, "@")) {
			if strings.HasPrefix(c.MongoURL, "mongodb://") {
				connStr = strings.Replace(c.MongoURL, "mongodb://",
					fmt.Sprintf("mongodb://%s:%s@", c.Username, c.Password), 1)
			} else if strings.HasPrefix(c.MongoURL, "mongodb+srv://") {
				connStr = strings.Replace(c.MongoURL, "mongodb+srv://",
					fmt.Sprintf("mongodb+srv://%s:%s@", c.Username, c.Password), 1)
			}
		}
	}

	// For local development, it's often easier to connect directly
	// without replica set discovery.
	if strings.Contains(connStr, "localhost") && !strings.Contains(connStr, "replicaSet") {
		if !strings.Contains(connStr, "?") {
			connStr += "?connect=direct"
		} else {
			connStr += "&connect=direct"
		}
	}

	return connStr
}

// Validate performs basic validation on the configuration.
func (c *Config) Validate() error {
	if err := c.validateDatabase(); err != nil {
		return err
	}

	if err := c.validateAISettings(); err != nil {
		return err
	}

	if err := c.validateGoogleDocsSettings(); err != nil {
		return err
	}

	return nil
}

// validateDatabase validates database configuration
func (c *Config) validateDatabase() error {
	if c.Database == "" {
		return fmt.Errorf("MONGO_DATABASE is required")
	}
	return nil
}

// validateAISettings validates AI provider configuration
func (c *Config) validateAISettings() error {
	if !c.AIEnabled {
		return nil
	}

	switch c.AIProvider {
	case "openai":
		return c.validateOpenAIKey()
	case "gemini":
		return c.validateGeminiKey()
	case "claude":
		return c.validateClaudeKey()
	default:
		return fmt.Errorf("invalid AI_PROVIDER: %s (must be openai, gemini, or claude)", c.AIProvider)
	}
}

// validateOpenAIKey validates OpenAI API key
func (c *Config) validateOpenAIKey() error {
	if c.OpenAIAPIKey == "" {
		return fmt.Errorf("OPENAI_API_KEY is required when AI_PROVIDER is openai")
	}
	return nil
}

// validateGeminiKey validates Gemini API key
func (c *Config) validateGeminiKey() error {
	if c.GeminiAPIKey == "" {
		return fmt.Errorf("GEMINI_API_KEY is required when AI_PROVIDER is gemini")
	}
	return nil
}

// validateClaudeKey validates Claude API key
func (c *Config) validateClaudeKey() error {
	if c.ClaudeAPIKey == "" {
		return fmt.Errorf("CLAUDE_API_KEY is required when AI_PROVIDER is claude")
	}
	return nil
}

// validateGoogleDocsSettings validates Google Docs configuration
func (c *Config) validateGoogleDocsSettings() error {
	if !c.GoogleDocsEnabled {
		return nil
	}

	if c.GoogleCredentialsPath == "" && c.GoogleCredentialsJSON == "" {
		return fmt.Errorf(
			"either GOOGLE_CREDENTIALS_PATH or GOOGLE_CREDENTIALS_JSON is required " +
				"when GOOGLE_DOCS_ENABLED is true")
	}
	return nil
}
