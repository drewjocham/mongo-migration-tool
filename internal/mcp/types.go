package mcp

type emptyArgs struct{}

type versionArgs struct {
	Version string `json:"version,omitempty"`
}

type messageOutput struct {
	Message string `json:"message"`
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


type createMigrationArgs struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}
