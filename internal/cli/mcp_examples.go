//go:build include_examples

package cli

import (
	"github.com/drewjocham/mongo-migration-tool/examples/examplemigrations"
	"github.com/drewjocham/mongo-migration-tool/internal/migration"
)

func registerExampleMigrations() error {
	migration.MustRegister(
		&examplemigrations.AddUserIndexesMigration{},
		&examplemigrations.TransformUserDataMigration{},
		&examplemigrations.CreateAuditCollectionMigration{},
	)
	return nil
}
