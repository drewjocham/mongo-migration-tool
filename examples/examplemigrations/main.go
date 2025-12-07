package examplemigrations

import (
	"github.com/jocham/mongo-migration/migration"
)

func init() { //nolint:gochecknoinits // init functions are used for migration registration
	migration.Register(
		&AddUserIndexesMigration{},
		&TransformUserDataMigration{},
		&CreateAuditCollectionMigration{},
	)
}
