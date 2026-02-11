package migrations

import "go.mongodb.org/mongo-driver/v2/mongo/options"

func indexName(opts *options.IndexOptionsBuilder) (string, bool) {
	if opts == nil {
		return "", false
	}

	values := &options.IndexOptions{}
	for _, setter := range opts.List() {
		if setter == nil {
			continue
		}
		if err := setter(values); err != nil {
			return "", false
		}
	}

	if values.Name == nil || *values.Name == "" {
		return "", false
	}
	return *values.Name, true
}
