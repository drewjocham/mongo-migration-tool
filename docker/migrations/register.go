package migrations

import "github.com/drewjocham/mongo-migration-tool/internal/migration"

func init() { //nolint:gochecknoinits // auto-registration keeps CLI zero-config
	migration.MustRegister(
		&AddUserIndexesMigration{},
		&CreateUsersCollectionMigration{},
		&Migration20251207_190640CreateProductCollection{},
		&Migration20251207_192545TestDemoAgl{},
	)
}
