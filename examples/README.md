# Migration Examples

This directory contains example migrations and a sample CLI application demonstrating how to use the mongo-migration migration library.

## Example Migrations

### 1. Add User Indexes (`example_20240101_001_add_user_indexes.go`)

Demonstrates how to:
- Create unique indexes
- Create compound indexes
- Set index options like background creation
- Handle index cleanup in rollback

**Operations:**
- Creates unique index on `email` field
- Creates descending index on `created_at` field
- Creates compound index on `status` and `created_at` fields

### 2. Transform User Data (`example_20240101_002_transform_user_data.go`)

Shows data transformation patterns:
- Iterating over collection documents
- Normalizing data (email to lowercase)
- Creating computed fields (`full_name` from `first_name` + `last_name`)
- Adding missing timestamps
- Conditional updates

### 3. Create Audit Collection (`example_20240101_003_create_audit_collection.go`)

Advanced collection operations:
- Creating collections with JSON Schema validation
- Setting up multiple indexes for different query patterns
- Using TTL indexes for automatic data expiration
- Schema validation with required fields and enums

## Sample CLI Application (`main.go`)

A complete example showing how to:
1. Load configuration using the config package
2. Connect to MongoDB
3. Create and configure a migration engine
4. Register multiple migrations
5. Implement up/down/status commands

### Quick Setup

#### Option 1: Using Docker Compose (Recommended)

```bash
# From the examples directory
cd examples

# Start MongoDB with sample data
docker-compose up -d

# Copy environment configuration
cp .env.example .env

# Run the examples
go run main.go status
go run main.go up
go run main.go status
go run main.go down

# Clean up when done
docker-compose down
```

#### Option 2: Local MongoDB

If you have MongoDB running locally:

```bash
# Set environment variables
export MONGO_URL="mongodb://localhost:27017"
export MONGO_DATABASE="migration_examples"
export MIGRATIONS_COLLECTION="schema_migrations"

# Or copy and edit the config file
cp .env.example .env
# Edit .env with your settings

# Run the examples
go run main.go status
go run main.go up
go run main.go down
```

### Expected Output

**Status command:**
```
Migration Status:
--------------------------------------------------------------------------------
Version              Applied    Applied At           Description
--------------------------------------------------------------------------------
example_20240101_001 ❌ No      Never                Add indexes to users collection for email and created_at fields
example_20240101_002 ❌ No      Never                Transform user data: normalize email case, add full_name field, and update timestamps
example_20240101_003 ❌ No      Never                Create audit collection with schema validation and indexes
```

**Up command:**
```
Running migrations up...
Running migration: example_20240101_001 - Add indexes to users collection for email and created_at fields
✅ Completed migration: example_20240101_001
Running migration: example_20240101_002 - Transform user data: normalize email case, add full_name field, and update timestamps
✅ Completed migration: example_20240101_002
Running migration: example_20240101_003 - Create audit collection with schema validation and indexes
✅ Completed migration: example_20240101_003
All migrations completed!
```

**Down command:**
```
Rolling back last migration...
Rolling back migration: example_20240101_003 - Create audit collection with schema validation and indexes
✅ Rolled back migration: example_20240101_003
```

## Library Usage Example

See `library-usage-example.go` for a complete example of using mongo-migration as a library in your own project. This example:

- Shows how to use mongo-migration without depending on the main project structure
- Demonstrates programmatic configuration as well as .env file usage
- Includes a complete migration lifecycle (up, status, down)
- Can be copied and adapted for your own projects

To run the library usage example:

```bash
# Make sure MongoDB is running (via Docker or locally)
go run library-usage-example.go
```

## Integration with Your Project

To use these examples in your own project:

1. **Install the library**: `go get github.com/drewjocham/mongo-migration-tool@latest`
2. **Copy migration patterns**: Use the migration structs in `examplemigrations/` as templates
3. **Use the library example**: Copy and adapt `library-usage-example.go` for your needs
4. **Configure properly**: Set up your .env file or programmatic configuration
5. **Add error handling**: Enhance error handling as needed for production use

## Testing the Examples

You can test the examples with a local MongoDB instance:

```bash
# Start MongoDB 
docker run -d -p 27017:27017 --name mongo-test mongo:latest

# Run the examples
cd examples
go run main.go status
go run main.go up
go run main.go status
go run main.go down
```

## Best Practices Demonstrated

1[library-usage-example.go](library-usage-example.go). **Idempotent operations**: Migrations handle cases where operations might be run multiple times
2. **Proper error handling**: Each migration returns errors appropriately
3. **Background index creation**: Uses background option to avoid blocking
4. **Schema validation**: Shows how to enforce data quality with JSON Schema
5. **Rollback strategies**: Each migration includes a proper down method
6. **Performance considerations**: Uses efficient query patterns and indexing
