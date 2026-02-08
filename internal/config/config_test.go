package config

import (
	"testing"
)

func TestLoad(t *testing.T) {
	t.Setenv("MONGO_URL", "mongodb://testhost:27017")
	t.Setenv("MONGO_DATABASE", "testdb")
	t.Setenv("MIGRATIONS_COLLECTION", "test_migrations")
	t.Setenv("GOOGLE_DRIVE_FOLDER_ID", "folder-123")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	assert(t, cfg.MongoURL, "mongodb://testhost:27017", "MongoURL")
	assert(t, cfg.Database, "testdb", "Database")
	assert(t, cfg.MigrationsCollection, "test_migrations", "MigrationsCollection")
}

func TestLoadDefaults(t *testing.T) {
	t.Setenv("MONGO_DATABASE", "default_test")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	assert(t, cfg.MongoURL, "mongodb://localhost:27017", "Default MongoURL")
	assert(t, cfg.MigrationsCollection, "schema_migrations", "Default MigrationsCollection")
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "Valid Configuration",
			config: &Config{
				Database: "ok",
			},
			wantErr: false,
		},
		{
			name: "Missing Database",
			config: &Config{
				Database: "",
			},
			wantErr: true,
		},
		{
			name: "Google Docs Enabled but missing credentials",
			config: &Config{
				Database:          "ok",
				GoogleDocsEnabled: true,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func assert(t *testing.T, got, want, field string) {
	t.Helper()
	if got != want {
		t.Errorf("%s: got %q, want %q", field, got, want)
	}
}
