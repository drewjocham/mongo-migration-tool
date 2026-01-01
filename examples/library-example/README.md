# Library Usage Example

This directory contains a standalone example of how to use mongo-migration as a library in your own Go projects.

## What's Included

- `main.go` - Complete example showing how to:
  - Load configuration
  - Connect to MongoDB
  - Create and register migrations
  - Run migrations up/down
  - Check migration status

## How to Use

### 1. Build the Example
```bash
# From the mongo-migration root directory
go build -o examples/library-example/library-example examples/library-example/main.go

# Or using make
make test-examples
```

### 2. Run the Example
```bash
# Set required environment variables first
export MONGO_URL="mongodb://localhost:27017"
export MONGO_DATABASE="example_db"

# Run the example
./examples/library-example/library-example
```

### 3. Use in Your Own Project
```bash
# Initialize your Go module
go mod init your-project-name

# Add mongo-migration dependency
go get github.com/drewjocham/mongo-migration-tool@latest

# Copy and adapt the example code from main.go
```

## Example Migration

The example includes a sample migration that:
- Creates a `sample_collection` with a document
- Adds an index on `created_at`
- Demonstrates rollback by dropping the collection

## Configuration

The example uses environment variables or falls back to default values:
- `MONGO_URL` - MongoDB connection string (default: mongodb://localhost:27017)
- `MONGO_DATABASE` - Database name (default: standalone_example)  
- `MIGRATIONS_COLLECTION` - Collection for migration tracking (default: schema_migrations)

## Output

When run successfully, you'll see:
```
🚀 mongo-migration Standalone Example
=====================================
ℹ️  Using default configuration (no .env file found)
🔗 Connecting to MongoDB: mongodb://localhost:27017/standalone_example
✅ Connected to MongoDB successfully

📊 Migration Status:
   20240109_001     ❌ No    Example migration - creates sample_collection with index

⬆️  Running migrations up...
✅ Created sample_collection with index
✅ All migrations applied successfully

📊 Updated Migration Status:
   20240109_001     ✅ Yes   Example migration - creates sample_collection with index

⬇️  Rolling back last migration...
✅ Dropped sample_collection
✅ Rolled back migration: 20240109_001

🎉 Standalone example completed successfully!
```

This demonstrates the complete lifecycle of using mongo-migration as a library.
