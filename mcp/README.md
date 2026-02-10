# MCP Client Examples

This directory contains examples and configurations for connecting your mongo-tool MCP server to various AI agents and tools.

## Prerequisites

1. **Build the tool**: Make sure you have built mongo-tool (mongo-tool) with MCP support:
 ```bash
   make build
```

2. **Configure MongoDB**: Set up your MongoDB connection:
```bash
   export MONGO_URI="mongodb://localhost:27017"
   export MONGO_DATABASE="your_database"
   export MIGRATIONS_COLLECTION="schema_migrations"
```

3. **Start MongoDB**: Ensure MongoDB is running (for testing):
```bash
   make db-up
```

## Testing the MCP Server

### Basic Test
Test the MCP server directly:
```bash
  make mcp-test
```

### Interactive Test
Test the MCP server interactively:
```bash
  make mcp-client-test
```

Then type JSON-RPC commands like:
```json
{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}
{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}
{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"migration_status","arguments":{}}}
```

## Integration Examples

### 1. Ollama Integration

Create a configuration file for Ollama:

**`~/.config/ollama/mcp-config.json`**:
```json
{
   "mongo-mcp": {
      "command": "$GOBIN/mongo-tool",
      "args": [
         "mcp"
      ],
      "env": {
         "MDB_MCP_CONNECTION_STRING": "",
         "MONGO_DATABASE": "",
         "MONGO_URL": ""
      }
   }
}

```

Then start Ollama with MCP support:
```bash
  ollama serve --mcp-config ~/.config/ollama/mcp-config.json
```

Now you can ask Ollama to help with migrations:
- "Check the status of my MongoDB migrations"
- "Create a new migration to add user email index"
- "Apply all pending migrations"

### 2. Goose Integration

Goose supports MCP through configuration files.

**`goose-mcp.json`**:
```json
{
  "tools": {
    "mongo-tool": {
      "type": "mcp",
      "server": {
        "command": "/path/to/mongo-tool",
        "args": ["mcp"],
        "cwd": "/path/to/your/project",
        "env": {
          "MONGO_URI": "mongodb://localhost:27017",
          "MONGO_DATABASE": "your_database"
        }
      }
    }
  }
}
```

Start Goose with the MCP configuration:
```bash
goose --config goose-mcp.json
```

### 3. Custom MCP Client

Here's a simple Python client example:

**`mcp_client.py`**:
```python
import json
import subprocess
import sys

def run_mcp_command(method, params=None):
    """Run an MCP command and return the response."""
    request = {
        "jsonrpc": "2.0",
        "id": 1,
        "method": method,
        "params": params or {}
    }
    
    # Start the MCP server
    process = subprocess.Popen(
        ["/path/to/mongo-tool", "mcp", "--with-examples"],
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True
    )
    
    # Send request and get response
    stdout, stderr = process.communicate(json.dumps(request))
    
    if stderr:
        print(f"Error: {stderr}", file=sys.stderr)
        return None
    
    try:
        return json.loads(stdout)
    except json.JSONDecodeError as e:
        print(f"Failed to parse response: {e}", file=sys.stderr)
        return None

def main():
    # Initialize the server
    init_response = run_mcp_command("initialize")
    print("Initialize:", json.dumps(init_response, indent=2))
    
    # List available tools
    tools_response = run_mcp_command("tools/list")
    print("Available tools:", json.dumps(tools_response, indent=2))
    
    # Check migration status
    status_response = run_mcp_command("tools/call", {
        "name": "migration_status",
        "arguments": {}
    })
    print("Migration status:", json.dumps(status_response, indent=2))

if __name__ == "__main__":
    main()
```

### 4. Claude Desktop Integration

For Claude Desktop, add to your configuration:

**`~/Library/Application Support/Claude/claude_desktop_config.json`** (macOS):
```json
{
  "mcpServers": {
    "mongo-tool": {
      "command": "/path/to/mongo-tool",
      "args": ["mcp", "--with-examples"],
      "env": {
        "MONGO_URI": "mongodb://localhost:27017",
        "MONGO_DATABASE": "your_database"
      }
    }
  }
}
```

### 5. VS Code Integration

Create a VS Code task for MCP server:

**`.vscode/tasks.json`**:
```json
{
  "version": "2.0.0",
  "tasks": [
    {
      "label": "Start MongoDB MCP Server",
      "type": "shell",
      "command": "./build/mongo-tool",
      "args": ["mcp", "--with-examples"],
      "group": "build",
      "presentation": {
        "echo": true,
        "reveal": "always",
        "focus": false,
        "panel": "new"
      },
      "env": {
        "MONGO_URI": "mongodb://localhost:27017",
        "MONGO_DATABASE": "your_database"
      },
      "problemMatcher": []
    }
  ]
}
```

## Available MCP Tools

The mongo-tool MCP server exposes these tools:

### 1. `migration_status`
**Description**: Get the status of all migrations  
**Parameters**: None  
**Example**:
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "tools/call",
  "params": {
    "name": "migration_status",
    "arguments": {}
  }
}
```

### 2. `migration_up` 
**Description**: Apply migrations up to a specific version or all pending  
**Parameters**:
- `version` (optional): Migration version to migrate up to  

**Examples**:
```json
// Apply all pending migrations
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/call", 
  "params": {
    "name": "migration_up",
    "arguments": {}
  }
}

// Apply up to specific version
{
  "jsonrpc": "2.0",
  "id": 3,
  "method": "tools/call",
  "params": {
    "name": "migration_up", 
    "arguments": {
      "version": "20240101_001"
    }
  }
}
```

### 3. `migration_down`
**Description**: Roll back migrations  
**Parameters**:
- `version` (optional): Migration version to roll back to  

**Examples**:
```json
// Roll back last migration
{
  "jsonrpc": "2.0",
  "id": 4,
  "method": "tools/call",
  "params": {
    "name": "migration_down",
    "arguments": {}
  }
}
```

### 4. `migration_create`
**Description**: Create a new migration file  
**Parameters**:
- `name` (required): Name for the migration  
- `description` (required): Description of what the migration does  

**Example**:
```json
{
  "jsonrpc": "2.0",
  "id": 5,
  "method": "tools/call",
  "params": {
    "name": "migration_create",
    "arguments": {
      "name": "add_user_preferences",
      "description": "Add user preferences table and indexes"
    }
  }
}
```

### 5. `migration_list`
**Description**: List all registered migrations  
**Parameters**: None  

**Example**:
```json
{
  "jsonrpc": "2.0",
  "id": 6,
  "method": "tools/call",
  "params": {
    "name": "migration_list",
    "arguments": {}
  }
}
```

## AI Assistant Prompts

Here are some example prompts you can use with AI assistants:

### Migration Management
- "Check the status of my MongoDB migrations and tell me what's pending"
- "Apply all pending migrations to bring my database up to date"
- "Create a new migration to add an index on the user email field"
- "Roll back the last migration I applied"

### Analysis and Planning
- "List all my migrations and show me which ones are applied"
- "I need to add a new field called 'preferences' to my user collection. Create a migration for this"
- "What migrations do I have pending and what do they do?"

### Troubleshooting
- "Something went wrong with my last migration. Show me the status and help me roll it back"
- "I want to see all my migrations and their current status to debug an issue"

## Troubleshooting

### Common Issues

1. **Connection refused**: Make sure MongoDB is running
   ```bash
   make db-up
   ```

2. **Permission denied**: Ensure the binary is executable
   ```bash
   chmod +x ./build/mongo-tool
   ```

3. **Environment variables**: Make sure MongoDB connection variables are set
   ```bash
   export MONGO_URI="mongodb://localhost:27017"
   export MONGO_DATABASE="your_database"
   ```

4. **JSON parsing errors**: Ensure your JSON-RPC requests are properly formatted

### Debug Mode

Enable debug logging by setting:
```bash
export LOG_LEVEL=debug
```

### Logs

Check MCP server logs for troubleshooting - the server logs to stderr while JSON-RPC communication happens on stdout.

## Next Steps

1. **Customize for your project**: Replace example migrations with your actual migrations
2. **Set up CI/CD**: Integrate MCP server into your deployment pipeline
3. **Create custom tools**: Extend the MCP server with additional MongoDB operations
4. **Monitor usage**: Add logging and metrics to track AI assistant usage
