# MCP Integration Guide

This document explains how to use mongo-essential as a Model Context Protocol (MCP) server for AI assistants like Ollama, Goose, Claude Desktop, and others.

## What is MCP?

The Model Context Protocol (MCP) is an open standard that enables AI assistants to securely connect to external data sources and tools. By implementing MCP, mongo-essential can be controlled by AI assistants, allowing you to manage MongoDB migrations using natural language.

## Quick Start

### 1. Build the Tool
```bash
make build
```

### 2. Test MCP Server
```bash
# Test basic functionality
make mcp-test

# Test interactively
make mcp-client-test
```

### 3. Configure Environment
```bash
export MONGO_URI="mongodb://localhost:27017"
export MONGO_DATABASE="your_database"
export MIGRATIONS_COLLECTION="schema_migrations"
```

### 4. Start MCP Server
```bash
# Start with your own migrations
./build/mongo-essential mcp

# Start with example migrations for testing
./build/mongo-essential mcp --with-examples
```

## AI Assistant Integration

### Ollama Integration

1. **Create MCP configuration**:
   ```bash
   mkdir -p ~/.config/ollama
   cat > ~/.config/ollama/mcp-config.json << EOF
   {
     "mcpServers": {
       "mongo-essential": {
         "command": "$(pwd)/build/mongo-essential",
         "args": ["mcp"],
         "env": {
           "MONGO_URI": "mongodb://localhost:27017",
           "MONGO_DATABASE": "your_database"
         }
       }
     }
   }
   EOF
   ```

2. **Start Ollama with MCP support**:
   ```bash
   ollama serve --mcp-config ~/.config/ollama/mcp-config.json
   ```

3. **Use with natural language**:
   - "Check the status of my MongoDB migrations"
   - "Apply all pending migrations to my database"
   - "Create a migration to add an email index to users"
   - "Roll back the last migration"

### Claude Desktop Integration

1. **Add to Claude Desktop config** (`~/Library/Application Support/Claude/claude_desktop_config.json` on macOS):
   ```json
   {
     "mcpServers": {
       "mongo-essential": {
         "command": "/absolute/path/to/mongo-essential",
         "args": ["mcp"],
         "env": {
           "MONGO_URI": "mongodb://localhost:27017",
           "MONGO_DATABASE": "your_database"
         }
       }
     }
   }
   ```

2. **Restart Claude Desktop** and start asking about migrations!

### Goose Integration

1. **Create Goose MCP config** (`goose-mcp.json`):
   ```json
   {
     "tools": {
       "mongo-essential": {
         "type": "mcp",
         "server": {
           "command": "/path/to/mongo-essential",
           "args": ["mcp"],
           "env": {
             "MONGO_URI": "mongodb://localhost:27017",
             "MONGO_DATABASE": "your_database"
           }
         }
       }
     }
   }
   ```

2. **Start Goose**:
   ```bash
   goose --config goose-mcp.json
   ```

### Custom Integration

Use the Python client example:
```bash
# Interactive mode
./mcp/examples/mcp_client.py interactive

# Direct commands
./mcp/examples/mcp_client.py status
./mcp/examples/mcp_client.py up
```

## Available MCP Tools

The MCP server exposes these tools that AI assistants can use:

### 🔍 `migration_status`
**Purpose**: Get the current status of all migrations  
**Parameters**: None  
**Returns**: Formatted table showing which migrations are applied

**Example AI prompt**: *"What's the status of my database migrations?"*

### ⬆️ `migration_up`
**Purpose**: Apply migrations forward  
**Parameters**:
- `version` (optional): Target version to migrate to

**Examples**:
- *"Apply all pending migrations"*
- *"Migrate up to version 20240101_001"*

### ⬇️ `migration_down`
**Purpose**: Roll back migrations  
**Parameters**:
- `version` (optional): Target version to roll back to

**Examples**:
- *"Roll back the last migration"*
- *"Roll back to version 20240101_001"*

### 📝 `migration_create`
**Purpose**: Create a new migration file  
**Parameters**:
- `name` (required): Migration name
- `description` (required): What the migration does

**Example**: *"Create a migration called 'add_user_email_index' that adds an index on user emails"*

### 📋 `migration_list`
**Purpose**: List all registered migrations  
**Parameters**: None  
**Returns**: List of all migrations with their status

**Example**: *"Show me all my migrations"*

## Example AI Conversations

### Migration Status Check
```
You: "Check my MongoDB migration status"

AI: I'll check the status of your MongoDB migrations.

[Uses migration_status tool]

Your database has 3 migrations:
- ✅ example_20240101_001 (Applied on 2024-01-15 10:30:00): Add user indexes
- ✅ example_20240101_002 (Applied on 2024-01-15 10:30:05): Transform user data  
- ❌ example_20240101_003 (Pending): Create audit collection

You have 1 pending migration that needs to be applied.
```

### Creating and Applying Migrations
```
You: "I need to add an index on the user email field. Create and apply a migration for this."

AI: I'll help you create and apply a migration for adding an email index to users.

[Uses migration_create tool]
[Uses migration_up tool]

✅ Created migration: 20240109_143022_add_email_index
✅ Applied migration successfully

Your users collection now has a unique index on the email field for better query performance and data integrity.
```

### Troubleshooting
```
You: "Something went wrong with my last migration. Help me roll it back."

AI: I'll check your migration status and help roll back the problematic migration.

[Uses migration_status tool]
[Uses migration_down tool]

I can see that the last applied migration was "example_20240101_003 - Create audit collection".
✅ Successfully rolled back the migration.

Your database is now back to the previous state. You can review and fix the migration before applying it again.
```

## Configuration Options

### Environment Variables
- `MONGO_URI`: MongoDB connection string (default: mongodb://localhost:27017)
- `MONGO_DATABASE`: Database name (required)
- `MIGRATIONS_COLLECTION`: Collection for migration tracking (default: schema_migrations)
- `LOG_LEVEL`: Logging level (debug, info, warn, error)

### Command Line Options
- `--with-examples`: Register example migrations (useful for testing)

### Security Considerations
- The MCP server connects to your MongoDB database
- Only use with trusted AI assistants
- Consider using read-only database users for status checks
- Review migration files before allowing AI to create them
- Use environment-specific database configurations

## Development and Testing

### Testing MCP Integration

1. **Start a test MongoDB**:
   ```bash
   make db-up
   ```

2. **Test MCP server directly**:
   ```bash
   # Basic test
   echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}' | ./build/mongo-essential mcp --with-examples

   # Interactive test
   make mcp-client-test
   ```

3. **Use Python client**:
   ```bash
   ./mcp/examples/mcp_client.py interactive
   ```

### Custom MCP Tools

To add custom tools to the MCP server:

1. **Extend the MCP server** (`mcp/server.go`):
   ```go
   // Add new tool to handleToolsList
   {
       Name:        "my_custom_tool",
       Description: "Description of what it does",
       InputSchema: map[string]interface{}{
           "type": "object",
           "properties": map[string]interface{}{
               "param": map[string]interface{}{
                   "type": "string",
                   "description": "Parameter description",
               },
           },
       },
   }

   // Add handler in handleToolsCall
   case "my_custom_tool":
       result, err = s.myCustomFunction(ctx, params.Arguments)
   ```

2. **Implement the function**:
   ```go
   func (s *MCPServer) myCustomFunction(ctx context.Context, args map[string]interface{}) (string, error) {
       // Your custom logic here
       return "Result", nil
   }
   ```

## Troubleshooting

### Common Issues

**MCP Server Won't Start**
- Check MongoDB connection: `mongo "mongodb://localhost:27017"`
- Verify binary exists: `ls -la ./build/mongo-essential`
- Check environment variables: `env | grep MONGO`

**AI Assistant Can't Connect**
- Ensure absolute paths in configuration files
- Check file permissions: `chmod +x ./build/mongo-essential`
- Verify JSON syntax in configuration files

**Migration Commands Fail**
- Check database permissions
- Verify migration files are properly registered
- Review server logs for detailed error messages

**JSON-RPC Errors**
- Ensure proper JSON formatting in requests
- Check that method names match exactly
- Verify parameter structures match tool schemas

### Debug Mode

Enable debug logging:
```bash
export LOG_LEVEL=debug
./build/mongo-essential mcp --with-examples
```

### Logs

- MCP communication happens on stdout/stdin
- Server logs and errors go to stderr
- Use `2>/dev/null` to suppress logs if needed

## Next Steps

1. **Production Setup**:
   - Use environment-specific configurations
   - Set up proper database access controls
   - Configure logging and monitoring

2. **Custom Migrations**:
   - Replace example migrations with your own
   - Create migration templates for common operations
   - Set up automated migration generation

3. **CI/CD Integration**:
   - Add MCP server to deployment pipelines
   - Create automated migration testing
   - Set up migration rollback procedures

4. **Advanced Features**:
   - Add custom MCP tools for your specific needs
   - Integrate with monitoring and alerting systems
   - Create migration analytics and reporting

## Contributing

To contribute to MCP integration:

1. **Test new AI assistants** and create integration examples
2. **Add new MCP tools** for additional MongoDB operations
3. **Improve error handling** and user experience
4. **Create tutorials** for specific AI assistant setups
5. **Submit bug reports** and feature requests

For more details, see the [MCP examples directory](mcp/examples/) and the [main README](README.md).