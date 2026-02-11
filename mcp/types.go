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

type parsePayloadArgs struct {
	Payload   string `json:"payload"`
	Format    string `json:"format,omitempty"`
	TypeField string `json:"typeField,omitempty"`
	TypeName  string `json:"type,omitempty"`
}
