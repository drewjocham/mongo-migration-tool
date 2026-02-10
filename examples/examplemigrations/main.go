package examplemigrations

import (
	"github.com/drewjocham/mongo-migration-tool/internal/migration"
)

func init() { //nolint:gochecknoinits // init functions are used for migration registration
	migration.MustRegister(
		&AddUserIndexesMigration{},
		&TransformUserDataMigration{},
		&CreateAuditCollectionMigration{},
	)
}
