package migrations

import (
	"github.com/jocham/mongo-migration/migration"
)

func init() {
	migration.Register(
		&CreateUsersCollectionMigration{},
		&AddUserIndexesMigration{},
		&Migration_20251207_190640_create_product_collection{},
		&Migration_20251207_192545_test_demo_agl{},
	)
}
