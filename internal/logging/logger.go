package logging

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// New initializes a new zap logger.
// If debug is true, it returns a development-friendly logger.
// If logFile is not empty, it writes to the specified file; otherwise, it writes to stderr.
func New(debug bool, logFile string) (*zap.Logger, error) {
	var cfg zap.Config
	if debug {
		cfg = zap.NewDevelopmentConfig()
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		cfg = zap.NewProductionConfig()
	}

	// If a log file is specified, direct output there.
	// Otherwise, use the default (stderr).
	if logFile != "" {
		cfg.OutputPaths = []string{logFile}
		cfg.ErrorOutputPaths = []string{logFile}
	}

	logger, err := cfg.Build()
	if err != nil {
		return nil, err
	}

	// Replace the global logger with this new logger.
	// This allows access via zap.L() and zap.S() throughout the app.
	zap.ReplaceGlobals(logger)

	return logger, nil
}
