# mongo-migration Go Library

[![Go Reference](https://pkg.go.dev/badge/github.com/drewjocham/mongo-migration-tool.svg)](https://pkg.go.dev/github.com/drewjocham/mongo-migration-tool)
[![Go Report Card](https://goreportcard.com/badge/github.com/drewjocham/mongo-migration-tool)](https://goreportcard.com/report/github.com/drewjocham/mongo-migration-tool)

Use mongo-migration as a Go library to integrate MongoDB migration capabilities into your applications.

## Installation

```bash
go get github.com/drewjocham/mongo-migration-tool@latest
```

## Quick Start

### Basic Migration Setup

```go
package main

import (
    "context"
    "log"
    "time"

    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"

    "github.com/drewjocham/mongo-migration-tool/config"
    "github.com/drewjocham/mongo-migration-tool/internal/migration"
)

func main() {
    cfg, err := config.Load()
    if err != nil {
        log.Fatal(err)
    }

    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    client, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.GetConnectionString()))
    if err != nil {
        log.Fatal(err)
    }
    defer client.Disconnect(ctx)
	
    engine := migration.NewEngine(client.Database(cfg.Database), cfg.MigrationsCollection)

    // Register your migrations
    engine.RegisterMany(
        &AddUserIndexesMigration{},
        &CreateProductCollection{},
        // ... migrations
    )
	
    if err := engine.Up(ctx, ""); err != nil {
        log.Fatal("Migration failed:", err)
    }

    log.Println("Migrations completed successfully!")
}
```

## Core Packages

### 1. Migration Engine (`migration`)

The heart of the library - provides migration management functionality.

#### Creating Migrations

```go
package migrations

import (
    "context"
    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"
)

type AddUserIndexesMigration struct{}

func (m *AddUserIndexesMigration) Version() string {
    return "20240109_001"
}

func (m *AddUserIndexesMigration) Description() string {
    return "Add indexes to users collection"
}

func (m *AddUserIndexesMigration) Up(ctx context.Context, db *mongo.Database) error {
    collection := db.Collection("users")
    
    indexModel := mongo.IndexModel{
        Keys: bson.D{
            {Key: "email", Value: 1},
        },
        Options: options.Index().
            SetUnique(true).
            SetName("email_unique_idx"),
    }
    
    _, err := collection.Indexes().CreateOne(ctx, indexModel)
    return err
}

func (m *AddUserIndexesMigration) Down(ctx context.Context, db *mongo.Database) error {
    collection := db.Collection("users")
    _, err := collection.Indexes().DropOne(ctx, "email_unique_idx")
    return err
}
```

#### Engine Operations

```go
// Create engine
engine := migration.NewEngine(database, "migrations")

// Register migrations
engine.Register(&MyMigration{})
engine.RegisterMany(migration1, migration2, migration3)

// Run migrations up
err := engine.Up(ctx, "") // All pending migrations
err := engine.Up(ctx, "20240109_002") // Up to specific version

// Run migrations down
err := engine.Down(ctx, "20240109_001") // Down to specific version

// Force mark migration as applied
err := engine.Force(ctx, "20240109_001")

// Get migration status
status, err := engine.GetStatus(ctx)
for _, s := range status {
    fmt.Printf("Migration %s: %s (Applied: %v)\n", 
        s.Version, s.Description, s.Applied)
}
```

### 2. Configuration (`config`)

Handles environment-based configuration with validation.

```go
package main

import (
    "github.com/drewjocham/mongo-migration-tool/config"
)

func main() {
    // Load from environment variables
    cfg, err := config.Load()
    if err != nil {
        log.Fatal(err)
    }

    // Load from specific files
    cfg, err = config.Load(".env", ".env.local")
    if err != nil {
        log.Fatal(err)
    }

    // Load only from environment (no files)
    cfg, err = config.LoadFromEnv()
    if err != nil {
        log.Fatal(err)
    }

    // Validate configuration
    if err := cfg.Validate(); err != nil {
        log.Fatal("Config validation failed:", err)
    }

    // Use configuration
    connectionString := cfg.GetConnectionString()
    database := cfg.Database
    migrationsCollection := cfg.MigrationsCollection
}
```

#### Available Configuration

```go
type Config struct {
    // MongoDB settings
    MongoURL             string
    Database             string
    MigrationsPath       string
    MigrationsCollection string
    Username             string
    Password             string
    MongoAuthSource      string
    
    // SSL/TLS settings
    SSLEnabled           bool
    SSLInsecure          bool
    SSLCertificatePath   string
    SSLPrivateKeyPath    string
    SSLCACertificatePath string
    
    // Connection pool
    MaxPoolSize int
    MinPoolSize int
    MaxIdleTime int
    Timeout     int
    
    // AI Analysis (if enabled)
    AIProvider   string
    AIEnabled    bool
    OpenAIAPIKey string
    OpenAIModel  string
    GeminiAPIKey string
    GeminiModel  string
    ClaudeAPIKey string
    ClaudeModel  string
    
    // Google Docs Integration
    GoogleDocsEnabled        bool
    GoogleCredentialsPath    string
    GoogleCredentialsJSON    string
    GoogleDriveFolderID      string
    GoogleDocsTemplate       string
    GoogleDocsShareWithEmail string
}
```

### 3. MCP Integration (`mcp`)

Model Context Protocol server integration.

```go
package main

import (
    "log"
    "github.com/drewjocham/mongo-migration-tool/mcp"
)

func main() {
    // Create MCP server
    server, err := mcp.NewMCPServer()
    if err != nil {
        log.Fatal(err)
    }
    defer server.Close()

    // Register your migrations
    server.RegisterMigrations(
        &MyMigration1{},
        &MyMigration2{},
    )

    // Start MCP server
    if err := server.Start(); err != nil {
        log.Fatal(err)
    }
}
```

## Advanced Usage

### Custom Migration Engine

```go
// Create engine with custom settings
engine := migration.NewEngine(database, "my_custom_migrations")

type ComplexMigration struct{}

func (m *ComplexMigration) Up(ctx context.Context, db *mongo.Database) error {
    session, err := db.Client().StartSession()
    if err != nil {
        return err
    }
    defer session.EndSession(ctx)

    callback := func(sessCtx mongo.SessionContext) (interface{}, error) {
        validation := bson.M{
            "$jsonSchema": bson.M{
                "bsonType": "object",
                "required": []string{"name", "email"},
                "properties": bson.M{
                    "name": bson.M{"bsonType": "string"},
                    "email": bson.M{
                        "bsonType": "string",
                        "pattern": "^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$",
                    },
                },
            },
        }

        opts := options.CreateCollection().SetValidator(validation)
        if err := db.CreateCollection(sessCtx, "validated_users", opts); err != nil {
            return nil, err
        }

        // Create indexes
        collection := db.Collection("validated_users")
        indexes := []mongo.IndexModel{
            {
                Keys:    bson.D{{Key: "email", Value: 1}},
                Options: options.Index().SetUnique(true),
            },
            {
                Keys:    bson.D{{Key: "name", Value: "text"}},
                Options: options.Index().SetName("name_text"),
            },
        }

        _, err := collection.Indexes().CreateMany(sessCtx, indexes)
        return nil, err
    }

    _, err = session.WithTransaction(ctx, callback)
    return err
}
```

### Error Handling

```go
func runMigrations(engine *migration.Engine) error {
    ctx := context.Background()
    
    // Get migration status
    status, err := engine.GetStatus(ctx)
    if err != nil {
        return fmt.Errorf("failed to get migration status: %w", err)
    }

    // Check for pending migrations
    var pending []migration.MigrationStatus
    for _, s := range status {
        if !s.Applied {
            pending = append(pending, s)
        }
    }

    if len(pending) == 0 {
        log.Println("No pending migrations")
        return nil
    }

    log.Printf("Found %d pending migrations", len(pending))

    // Run migrations with proper error handling
    for _, p := range pending {
        log.Printf("Running migration: %s - %s", p.Version, p.Description)
        
        if err := engine.Up(ctx, p.Version); err != nil {
            if mongo.IsDuplicateKeyError(err) {
                log.Printf("Warning: Duplicate key in migration %s (may be safe to ignore)", p.Version)
                continue
            }
            
            return fmt.Errorf("migration %s failed: %w", p.Version, err)
        }
        
        log.Printf("✅ Completed migration: %s", p.Version)
    }

    return nil
}
```

### Testing Migrations

```go
package migrations_test

import (
    "context"
    "testing"
    
    "go.mongodb.org/mongo-driver/mongo/integration/mtest"
    
    "github.com/drewjocham/mongo-migration-tool/internal/migration"
)

func TestAddUserIndexesMigration(t *testing.T) {
    mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
    defer mt.Close()

    mt.Run("should create email index", func(mt *mtest.T) {
        // Create migration
        m := &AddUserIndexesMigration{}
		
        err := m.Up(context.Background(), mt.DB)
        if err != nil {
            t.Fatalf("Up migration failed: %v", err)
        }
		
        err = m.Down(context.Background(), mt.DB)
        if err != nil {
            t.Fatalf("Down migration failed: %v", err)
        }
    })
}

func TestMigrationEngine(t *testing.T) {
    mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
    defer mt.Close()

    mt.Run("should track migration status", func(mt *mtest.T) {
        engine := migration.NewEngine(mt.DB, "test_migrations")
        
        // Register test migration
        testMig := &AddUserIndexesMigration{}
        engine.Register(testMig)
        
        // Check initial status
        status, err := engine.GetStatus(context.Background())
        if err != nil {
            t.Fatalf("GetStatus failed: %v", err)
        }
        
        if len(status) != 1 {
            t.Fatalf("Expected 1 migration, got %d", len(status))
        }
        
        if status[0].Applied {
            t.Fatal("Migration should not be applied initially")
        }
    })
}
```

## Environment Configuration

Create a `.env` file in your project:

```bash
# MongoDB Configuration
MONGO_URL=mongodb://localhost:27017
MONGO_DATABASE=myapp
MIGRATIONS_COLLECTION=schema_migrations
MIGRATIONS_PATH=./migrations

# MongoDB Authentication 
MONGO_USERNAME=username
MONGO_PASSWORD=password
MONGO_AUTH_SOURCE=admin

# SSL/TLS 
MONGO_SSL_ENABLED=true
MONGO_SSL_INSECURE=false
MONGO_SSL_CERT_PATH=./certs/client.pem
MONGO_SSL_KEY_PATH=./certs/client-key.pem
MONGO_SSL_CA_CERT_PATH=./certs/ca.pem

# Connection Pool Settings
MONGO_MAX_POOL_SIZE=10
MONGO_MIN_POOL_SIZE=1
MONGO_MAX_IDLE_TIME=300
MONGO_TIMEOUT=60

# AI Models (optional)
AI_ENABLED=false
AI_PROVIDER=openai
OPENAI_API_KEY=your_openai_key
OPENAI_MODEL=gpt-4o-mini

# Google Docs Integration (optional)
GOOGLE_DOCS_ENABLED=false
GOOGLE_CREDENTIALS_PATH=./credentials.json
GOOGLE_DRIVE_FOLDER_ID=your_folder_id
```

## Best Practices

### 1. Migration Naming
```go
// Good: Use timestamp + descriptive name
"20240109_143022_add_user_email_index"
"20240109_143045_create_product_collection"

// Bad: Non-descriptive or non-sequential
"migration1"
"fix_stuff"
```

### 2. Idempotent Migrations
```go
func (m *CreateIndexMigration) Up(ctx context.Context, db *mongo.Database) error {
    collection := db.Collection("users")
    
    // Check if index already exists
    indexes := collection.Indexes()
    cursor, err := indexes.List(ctx)
    if err != nil {
        return err
    }
    
    var existingIndexes []bson.M
    if err := cursor.All(ctx, &existingIndexes); err != nil {
        cursor.Close(ctx)
        return err
    }
    cursor.Close(ctx)
    
    // Check if our index already exists
    for _, idx := range existingIndexes {
        if name, ok := idx["name"].(string); ok && name == "email_unique_idx" {
            return nil // Index already exists
        }
    }
    
    // Create index only if it doesn't exist
    indexModel := mongo.IndexModel{
        Keys: bson.D{{Key: "email", Value: 1}},
        Options: options.Index().SetUnique(true).SetName("email_unique_idx"),
    }
    
    _, err = indexes.CreateOne(ctx, indexModel)
    return err
}
```

### 3. Rollback Safety
```go
func (m *AddFieldMigration) Down(ctx context.Context, db *mongo.Database) error {
    collection := db.Collection("users")
    
    // Use $unset to remove field, not $set with null
    update := bson.M{"$unset": bson.M{"new_field": ""}}
    
    _, err := collection.UpdateMany(ctx, bson.M{}, update)
    return err
}
```

### 4. Performance Considerations
```go
func (m *LargeDataMigration) Up(ctx context.Context, db *mongo.Database) error {
    collection := db.Collection("large_collection")
    
    // Process in batches for large collections
    batchSize := 1000
    skip := 0
    
    for {
        cursor, err := collection.Find(ctx, bson.M{}, 
            options.Find().SetLimit(int64(batchSize)).SetSkip(int64(skip)))
        if err != nil {
            return err
        }
        
        var docs []bson.M
        if err := cursor.All(ctx, &docs); err != nil {
            cursor.Close(ctx)
            return err
        }
        cursor.Close(ctx)
        
        if len(docs) == 0 {
            break // No more documents
        }
        
        // Process batch
        for _, doc := range docs {
            // Update logic here
            filter := bson.M{"_id": doc["_id"]}
            update := bson.M{"$set": bson.M{"processed": true}}
            
            if _, err := collection.UpdateOne(ctx, filter, update); err != nil {
                return err
            }
        }
        
        skip += len(docs)
        
        // Check if we got a partial batch (indicates end)
        if len(docs) < batchSize {
            break
        }
    }
    
    return nil
}
```

## API Reference

For complete API documentation, visit [pkg.go.dev/github.com/drewjocham/mongo-migration-tool](https://pkg.go.dev/github.com/drewjocham/mongo-migration-tool).

## Examples

See the [examples directory](examples/) for complete working examples of:
- Basic migration setup
- Complex migration patterns
- MCP integration
- Testing strategies

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
