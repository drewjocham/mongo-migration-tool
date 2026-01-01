//go:build include_examples

package cmd

import "github.com/drewjocham/mongo-migration-tool/examples/examplemigrations"
import "github.com/drewjocham/mongo-migration-tool/migration"

func registerExampleMigrations() error {
	migration.Register(
		&examplemigrations.AddUserIndexesMigration{},
		&examplemigrations.TransformUserDataMigration{},
		&examplemigrations.CreateAuditCollectionMigration{},
	)
	return nil
}
