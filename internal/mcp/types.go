package mcp

type emptyArgs struct{}

type versionArgs struct {
	Version string `json:"version,omitempty" jsonschema:"title=Version,description=Limit applied version; empty applies all."` //nolint:lll // it is needed
}

type createMigrationArgs struct {
	Name string `json:"name" jsonschema:"description=Short migration name (e.g., 'add user indexes')."`
	// A brief summary of the changes in this migration.
	Description string `json:"description" jsonschema:"description=A brief summary of the changes in this migration."`
}
