package migration

import (
	"fmt"
	"regexp"
	"sync"
)

var (
	registryMu sync.RWMutex
	registered = make(map[string]Migration)

	versionPattern = regexp.MustCompile(`^\d{8}(?:_\d{3,6})?(?:_[a-z0-9_]+)?$`)
)

func Register(m Migration) error {
	if m == nil {
		return fmt.Errorf("migration must not be nil")
	}

	version := m.Version()
	if !isValidVersionFormat(version) {
		return fmt.Errorf("invalid version format: %s (expected YYYYMMDD[_HHMMSS][_slug])", version)
	}

	registryMu.Lock()
	defer registryMu.Unlock()

	if _, exists := registered[version]; exists {
		return fmt.Errorf("migration %s already registered", version)
	}

	registered[version] = m
	return nil
}

func MustRegister(ms ...Migration) {
	for _, m := range ms {
		if err := Register(m); err != nil {
			panic(err)
		}
	}
}

func RegisteredMigrations() map[string]Migration {
	registryMu.RLock()
	defer registryMu.RUnlock()

	copy := make(map[string]Migration, len(registered))
	for k, v := range registered {
		copy[k] = v
	}
	return copy
}

type MigrationFilter func(version string, m Migration) bool

func GetMigrations(filters ...MigrationFilter) map[string]Migration {
	registryMu.RLock()
	defer registryMu.RUnlock()

	results := make(map[string]Migration)
	for v, m := range registered {
		keep := true
		for _, filter := range filters {
			if !filter(v, m) {
				keep = false
				break
			}
		}
		if keep {
			results[v] = m
		}
	}
	return results
}

func isValidVersionFormat(version string) bool {
	return versionPattern.MatchString(version)
}
