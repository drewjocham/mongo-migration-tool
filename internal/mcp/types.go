package mcp

type emptyArgs struct{}

type versionArgs struct {
	Version string `json:"version,omitempty" jsonschema:"title=Version,description=Target version limit."`
}

type createMigrationArgs struct {
	Name string `json:"name" jsonschema:"description=Short migration name (e.g., 'add user indexes')."`
	Description string `json:"description" jsonschema:"description=A brief summary of the changes in this migration."`
}
