//go:build include_examples

package cmd

import "github.com/jocham/mongo-migration/examples/examplemigrations"
import "github.com/jocham/mongo-migration/migration"

func registerExampleMigrations() error {
	migration.Register(
		&examplemigrations.AddUserIndexesMigration{},
		&examplemigrations.TransformUserDataMigration{},
		&examplemigrations.CreateAuditCollectionMigration{},
	)
	return nil
}
