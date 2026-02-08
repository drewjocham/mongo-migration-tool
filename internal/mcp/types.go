package mcp

type emptyArgs struct{}

type versionArgs struct {
	Version string `json:"version,omitempty" jsonschema:"title=Version identifier,description=The version to migrate to. If omitted, all pending migrations will be applied."`
}

type createMigrationArgs struct {
	Name string `json:"name" jsonschema:"description=A short, descriptive name for the migration (e.g., 'add user indexes')."`
	// A brief summary of the changes in this migration.
	Description string `json:"description" jsonschema:"description=A brief summary of the changes in this migration."`
}
