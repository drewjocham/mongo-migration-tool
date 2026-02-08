package cli

type ErrorCli string

func (e ErrorCli) Error() string {
	return string(e)
}

const (
	ErrInvalidVersion      = ErrorCli("invalid version format")
	ErrMigrationNotFound   = ErrorCli("migration not found")
	ErrFailedToCreate      = ErrorCli("failed to create migration")
	ErrFailedToReadConfig  = ErrorCli("failed to read config")
	ErrFailedToParseConfig = ErrorCli("failed to parse config")
	ErrFailedToGetStatus   = ErrorCli("failed to get status")
	ErrFailedToRun         = ErrorCli("failed to run migration")
	ErrFailedToDown        = ErrorCli("failed to run down migration")
	ErrFailedToForce       = ErrorCli("failed to force migration")
	ErrInvalidForceVersion = ErrorCli("invalid force version")
)
