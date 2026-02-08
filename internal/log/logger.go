package logging

import (
	"fmt"
	"log/slog"

	"go.uber.org/zap"
	"go.uber.org/zap/exp/zapslog"
	"go.uber.org/zap/zapcore"
)

func New(debug bool, logFile string) (*slog.Logger, error) {
	var config zap.Config

	if debug {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		config = zap.NewProductionConfig()
		config.Encoding = "json"
	}

	if logFile != "" {
		config.OutputPaths = []string{logFile}
		config.ErrorOutputPaths = []string{logFile}
	}

	zLogger, err := config.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build zap logger: %w", err)
	}
	logger := slog.New(zapslog.NewHandler(zLogger.Core()))
	zap.ReplaceGlobals(zLogger)

	return logger, nil
}
