package examplemigrations

import (
	"github.com/jocham/mongo-migration/migration"
)

func init() {
	migration.Register(
		&AddUserIndexesMigration{},
		&TransformUserDataMigration{},
		&CreateAuditCollectionMigration{},
	)
}
