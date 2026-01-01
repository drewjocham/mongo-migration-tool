package migrations

import "github.com/jocham/mongo-migration-tool/migration"

func init() { //nolint:gochecknoinits // auto-registration keeps CLI zero-config
	migration.Register(
		&AddUserIndexesMigration{},
		&CreateUsersCollectionMigration{},
		&Migration20251207_190640CreateProductCollection{},
		&Migration20251207_192545TestDemoAgl{},
	)
}
