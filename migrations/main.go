package migrations

import (
	"github.com/jocham/mongo-migration/migration"
)

func init() { //nolint:gochecknoinits // init functions are used for migration registration
	migration.Register(
		&CreateUsersCollectionMigration{},
		&AddUserIndexesMigration{},
		&Migration20251207_190640CreateProductCollection{},
		&Migration20251207_192545TestDemoAgl{},
	)
}
