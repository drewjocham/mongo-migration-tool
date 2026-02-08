package mcp

type emptyArgs struct{}

type versionArgs struct {
	Version string `json:"version,omitempty"`
}

type messageOutput struct {
	Message string `json:"message"`
}


type createMigrationArgs struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}
