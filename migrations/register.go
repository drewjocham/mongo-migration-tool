package migrations

import "github.com/drewjocham/mongo-migration-tool/internal/migration"

func init() { //nolint:gochecknoinits // auto-registration keeps CLI zero-config
	migration.MustRegister(
		&AddUserIndexesMigration{},
		&CreateUsersCollectionMigration{},
		&Migration20251207_190640CreateProductCollection{},
		&Migration20251207_192545TestDemoAgl{},
		&Migration_20260208_030133_example1{},
		&Migration_20260208_032830_drew{},
	)
}
