package migration

import (
	"fmt"
	"sync"
)

var (
	mu       sync.RWMutex
	registry = make(map[string]Migration)
)

// Register adds one or more migrations to the global registry.
func Register(migrations ...Migration) {
	mu.Lock()
	defer mu.Unlock()

	for _, m := range migrations {
		if m == nil {
			continue
		}

		version := m.Version()
		if _, exists := registry[version]; exists {
			panic(fmt.Sprintf("migration: duplicate version registered: %s", version))
		}
		registry[version] = m
	}
}

func All() map[string]Migration {
	mu.RLock()
	defer mu.RUnlock()

	copied := make(map[string]Migration, len(registry))
	for k, v := range registry {
		copied[k] = v
	}
	return copied
}

// RegisteredMigrations returns a copy of every registered migration.
func RegisteredMigrations() map[string]Migration {
	return All()
}
