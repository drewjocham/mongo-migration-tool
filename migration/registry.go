package migration

import (
	"fmt"
	"sync"
)

var (
	registryMu sync.RWMutex
	registered = make(map[string]Migration)
)

func Register(m Migration) error {
	if m == nil {
		return fmt.Errorf("migration must not be nil")
	}

	version := m.Version()

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
