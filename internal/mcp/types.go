package mcp

import (
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type emptyArgs struct{}

type versionArgs struct {
	Version string `json:"version,omitempty" jsonschema:"title=Version identifier"`
}

type createMigrationArgs struct {
	Name        string `json:"name" jsonschema:"description=Migration name"`
	Description string `json:"description" jsonschema:"description=Brief summary"`
}

func toolErrorResult(msg string, err error) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: fmt.Sprintf("❌ %s: %v", msg, err)}},
	}
}

func toolTextResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: text}},
	}
}
