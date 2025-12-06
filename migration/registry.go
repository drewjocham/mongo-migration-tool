package migration

import "fmt"

var (
	// registry is a global collection of all registered migrations.
	registry = make(map[string]Migration)
)

// Register adds one or more migrations to the global registry.
//
// This function is typically called from the init() function of a migration file,
// allowing migrations to be auto-registered when the application starts.
//
// Example:
//
//	func init() {
//		migration.Register(&MyMigration{})
//	}
func Register(migrations ...Migration) {
	for _, m := range migrations {
		version := m.Version()
		if _, exists := registry[version]; exists {
			panic(fmt.Sprintf("migration: duplicate version registered: %s", version))
		}
		registry[version] = m
	}
}

// RegisteredMigrations returns a map of all registered migrations.
//
// The returned map is a copy to prevent modification of the internal registry.
func RegisteredMigrations() map[string]Migration {
	// Return a copy to prevent external modification
	copied := make(map[string]Migration, len(registry))
	for k, v := range registry {
		copied[k] = v
	}
	return copied
}
