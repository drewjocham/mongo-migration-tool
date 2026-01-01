package examplemigrations

import (
	"github.com/drewjocham/mongo-migration-tool/migration"
)

func init() { //nolint:gochecknoinits // init functions are used for migration registration
	migration.Register(&AddUserIndexesMigration{})
	migration.Register(&TransformUserDataMigration{})
	migration.Register(&CreateAuditCollectionMigration{})
}
