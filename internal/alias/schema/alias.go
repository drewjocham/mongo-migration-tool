package schema

import is "github.com/drewjocham/mongo-migration-tool/internal/schema"

type IndexSpec = is.IndexSpec

var (
	Register            = is.Register
	MustRegister        = is.MustRegister
	Indexes             = is.Indexes
	IndexesByCollection = is.IndexesByCollection
)
