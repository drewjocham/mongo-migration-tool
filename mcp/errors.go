package mcp

type ErrorMcp string

func (e ErrorMcp) Error() string {
	return string(e)
}

const (
	ErrFailedToStartServer     = ErrorMcp("failed to start server")
	ErrFailedToReadBody        = ErrorMcp("failed to read request body")
	ErrFailedToParseJSON       = ErrorMcp("failed to parse JSON")
	ErrFailedToCreateMigration = ErrorMcp("failed to create migration")
	ErrFailedToGetMigrations   = ErrorMcp("failed to get migrations")
	ErrFailedToRenderTemplate  = ErrorMcp("failed to render template")
	ErrFailedToGetMigration    = ErrorMcp("failed to get migration")
	ErrFailedToUpdateMigration = ErrorMcp("failed to update migration")
	ErrFailedToDeleteMigration = ErrorMcp("failed to delete migration")
	ErrMigrationNotFound       = ErrorMcp("migration not found")
)
