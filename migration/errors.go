package migration

type ErrorMigration string

func (e ErrorMigration) Error() string {
	return string(e)
}

const (
	ErrInvalidMigrationVersion = ErrorMigration("invalid migration version")
	ErrMigrationNotFound       = ErrorMigration("migration not found")
	ErrFailedToGenerate        = ErrorMigration("failed to generate migration")
	ErrFailedToReadTemplate    = ErrorMigration("failed to read template")
	ErrFailedToParseTemplate   = ErrorMigration("failed to parse template")
	ErrFailedToExecuteTemplate = ErrorMigration("failed to execute template")
	ErrFailedToCreateFile      = ErrorMigration("failed to create migration file")
	ErrFailedToConnect         = ErrorMigration("failed to connect to database")
	ErrFailedToPing            = ErrorMigration("failed to ping database")
	ErrFailedToLock            = ErrorMigration("failed to acquire lock")
	ErrFailedToUnlock          = ErrorMigration("failed to release lock")
	ErrFailedToReadMigrations  = ErrorMigration("failed to read migrations")
	ErrFailedToRunMigration    = ErrorMigration("failed to run migration")
	ErrFailedToSetVersion      = ErrorMigration("failed to set version")
)
