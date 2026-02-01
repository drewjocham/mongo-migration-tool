package migration

import (
	"sync"
)

var (
	registryMu sync.RWMutex
	registered = make(map[string]Migration)
)

func Register(m Migration) {
	registryMu.Lock()
	defer registryMu.Unlock()
	registered[m.Version()] = m
}

func RegisteredMigrations() map[string]Migration {
	registryMu.RLock()
	defer registryMu.RUnlock()

	copy := make(map[string]Migration)
	for k, v := range registered {
		copy[k] = v
	}
	return copy
}
