package migrations

import (
	"github.com/drewjocham/mongo-migration-tool/internal/schema"
	"go.mongodb.org/mongo-driver/v2/bson"
)

func init() { //nolint:gochecknoinits // registration for schema metadata
	schema.MustRegister(
		schema.IndexSpec{
			Collection: "users",
			Name:       "idx_users_email_unique",
			Keys: bson.D{
				{Key: "email", Value: 1},
			},
			Unique: true,
		},
		schema.IndexSpec{
			Collection: "users",
			Name:       "idx_users_created_at",
			Keys: bson.D{
				{Key: "created_at", Value: -1},
			},
		},
		schema.IndexSpec{
			Collection: "users",
			Name:       "idx_users_status_created_at",
			Keys: bson.D{
				{Key: "status", Value: 1},
				{Key: "created_at", Value: -1},
			},
		},
		schema.IndexSpec{
			Collection: "drew",
			Name:       "idx_address_created_at",
			Keys: bson.D{
				{Key: "address", Value: 1},
				{Key: "created_at", Value: -1},
			},
		},
	)
}
