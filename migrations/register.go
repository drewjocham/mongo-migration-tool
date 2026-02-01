package migrations

import "github.com/drewjocham/mongo-migration-tool/migration"

func init() { //nolint:gochecknoinits // auto-registration keeps CLI zero-config
	migration.Register(&AddUserIndexesMigration{})
	migration.Register(&CreateUsersCollectionMigration{})
	migration.Register(&Migration20251207_190640CreateProductCollection{})
	migration.Register(&Migration20251207_192545TestDemoAgl{})
}
