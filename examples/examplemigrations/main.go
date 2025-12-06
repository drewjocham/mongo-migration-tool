package examplemigrations

import (
	"github.com/jocham/mongo-essential/migration"
)

func init() {
	migration.Register(
		&AddUserIndexesMigration{},
		&TransformUserDataMigration{},
		&CreateAuditCollectionMigration{},
	)
}
