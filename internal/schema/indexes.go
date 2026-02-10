package schema

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type IndexSpec struct {
	Collection          string
	Name                string
	Keys                bson.D
	Unique              bool
	Sparse              bool
	PartialFilter       bson.D
	ExpireAfterSeconds  *int32
	AdditionalStatement string
}

var (
	registryMu sync.RWMutex
	registry   = make(map[string]IndexSpec)
)

func Register(spec IndexSpec) error {
	if spec.Collection == "" {
		return fmt.Errorf("index collection is required")
	}
	if spec.Name == "" {
		return fmt.Errorf("index name is required")
	}
	if len(spec.Keys) == 0 {
		return fmt.Errorf("index %s.%s must define at least one key", spec.Collection, spec.Name)
	}

	key := makeRegistryKey(spec.Collection, spec.Name)

	registryMu.Lock()
	defer registryMu.Unlock()

	if _, exists := registry[key]; exists {
		return fmt.Errorf("index %s already registered", key)
	}

	registry[key] = spec
	return nil
}

// MustRegister adds specs and panics if any registration fails.
func MustRegister(specs ...IndexSpec) {
	for _, spec := range specs {
		if err := Register(spec); err != nil {
			panic(err)
		}
	}
}

func Indexes() []IndexSpec {
	registryMu.RLock()
	defer registryMu.RUnlock()

	result := make([]IndexSpec, 0, len(registry))
	for _, spec := range registry {
		result = append(result, spec)
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Collection == result[j].Collection {
			return result[i].Name < result[j].Name
		}
		return result[i].Collection < result[j].Collection
	})

	return result
}

// IndexesByCollection groups specs by collection name.
func IndexesByCollection() map[string][]IndexSpec {
	grouped := make(map[string][]IndexSpec)
	for _, spec := range Indexes() {
		grouped[spec.Collection] = append(grouped[spec.Collection], spec)
	}
	return grouped
}

// KeyString renders the index keys in deterministic order.
func (s IndexSpec) KeyString() string {
	return formatBsonD(s.Keys)
}

// PartialFilterString renders the partial filter expression, if any.
func (s IndexSpec) PartialFilterString() string {
	if len(s.PartialFilter) == 0 {
		return ""
	}
	return formatBsonD(s.PartialFilter)
}

func makeRegistryKey(collection, name string) string {
	return fmt.Sprintf("%s.%s", collection, name)
}

func formatBsonD(doc bson.D) string {
	if len(doc) == 0 {
		return ""
	}
	parts := make([]string, len(doc))
	for i, elem := range doc {
		parts[i] = fmt.Sprintf("%s:%v", elem.Key, elem.Value)
	}
	return strings.Join(parts, ", ")
}
