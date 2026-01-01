// Package mongo-migration provides MongoDB migration and AI-powered database analysis.
//
// mongo-migration is a comprehensive toolkit for MongoDB that combines database migration
// capabilities (similar to Liquibase/Flyway for relational databases) with AI-powered
// database analysis and optimization recommendations.
//
// # Migration System
//
// The migration system allows you to version control your MongoDB schema changes:
//
//	import (
//		"context"
//		"log"
//
//		"go.mongodb.org/mongo-driver/mongo"
//		"go.mongodb.org/mongo-driver/mongo/options"
//		"github.com/jocham/mongo-migration-tool/migration"
//		"github.com/jocham/mongo-migration-tool/config"
//	)
//
//	// Load configuration
//	cfg, err := config.Load(".env")
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	// Connect to MongoDB
//	client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.GetConnectionString()))
//	if err != nil {
//		log.Fatal(err)
//	}
//	db := client.Database(cfg.Database)
//
//	// Create migration engine
//	engine := migration.NewEngine(db, cfg.MigrationsCollection)
//
//	// Register migrations
//	engine.RegisterMany(
//		&CreateUsersCollection{},
//		&AddUserIndexes{},
//		&CreateOrdersCollection{},
//	)
//
//	// Run pending migrations
//	if err := engine.Up(ctx, ""); err != nil {
//		log.Fatal(err)
//	}
//
// # Creating Migrations
//
// Each migration must implement the Migration interface:
//
//	type CreateUsersCollection struct{}
//
//	func (m *CreateUsersCollection) Version() string {
//		return "20240101_001"  // Use YYYYMMDD_NNN format
//	}
//
//	func (m *CreateUsersCollection) Description() string {
//		return "Create users collection with indexes"
//	}
//
//	func (m *CreateUsersCollection) Up(ctx context.Context, db *mongo.Database) error {
//		collection := db.Collection("users")
//
//		// Create indexes
//		emailIndex := mongo.IndexModel{
//			Keys:    bson.D{{"email", 1}},
//			Options: options.Index().SetUnique(true),
//		}
//
//		_, err := collection.Indexes().CreateOne(ctx, emailIndex)
//		return err
//	}
//
//	func (m *CreateUsersCollection) Down(ctx context.Context, db *mongo.Database) error {
//		// Rollback logic
//		collection := db.Collection("users")
//		_, err := collection.Indexes().DropOne(ctx, "email_1")
//		return err
//	}
//
// GitHub Repository: https://github.com/jocham/mongo-migration-tool
//
// Documentation: https://pkg.go.dev/github.com/jocham/mongo-migration-tool
package main
