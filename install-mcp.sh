#!/bin/bash
set -e

echo "🚀 Installing mongo-essential MCP server..."

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "❌ Error: Go is not installed. Please install Go 1.21 or later."
    echo "Visit: https://golang.org/dl/"
    exit 1
fi

# Check Go version
GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
REQUIRED_VERSION="1.21"

if [ "$(printf '%s\n' "$REQUIRED_VERSION" "$GO_VERSION" | sort -V | head -n1)" != "$REQUIRED_VERSION" ]; then
    echo "❌ Error: Go version $GO_VERSION is too old. Please install Go $REQUIRED_VERSION or later."
    exit 1
fi

echo "✅ Go $GO_VERSION detected"

# Build the binary
echo "🔨 Building mongo-essential..."
make build

if [ ! -f "./build/mongo-essential" ]; then
    echo "❌ Error: Build failed. Binary not found."
    exit 1
fi

echo "✅ Build successful"

# Make binary executable
chmod +x ./build/mongo-essential

# Test the binary
echo "🧪 Testing binary..."
if ./build/mongo-essential --help > /dev/null 2>&1; then
    echo "✅ Binary is working"
else
    echo "❌ Error: Binary test failed"
    exit 1
fi

# Display installation location
INSTALL_PATH=$(pwd)/build/mongo-essential
echo ""
echo "🎉 Installation complete!"
echo ""
echo "Binary location: $INSTALL_PATH"
echo ""
echo "📋 Next steps:"
echo ""
echo "1. Configure your MongoDB connection:"
echo "   export MONGO_URL='mongodb://localhost:27017'"
echo "   export MONGO_DATABASE='your_database'"
echo ""
echo "2. Test the MCP server:"
echo "   make mcp-test"
echo ""
echo "3. Configure your AI assistant (Claude Desktop example):"
echo "   Add to ~/Library/Application Support/Claude/claude_desktop_config.json:"
echo ""
echo '   {
     "mcpServers": {
       "mongo-essential": {
         "command": "'$INSTALL_PATH'",
         "args": ["mcp"],
         "env": {
           "MONGO_URL": "mongodb://localhost:27017",
           "MONGO_DATABASE": "your_database"
         }
       }
     }
   }'
echo ""
echo "4. For more integration examples, see: MCP.md"
echo ""
echo "✨ Start using: mongo-essential mcp"
