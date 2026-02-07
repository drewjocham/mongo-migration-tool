package logging

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func New(debug bool, logFile string) (*zap.Logger, error) {
	var cfg zap.Config
	if debug {
		cfg = zap.NewDevelopmentConfig()
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	} else {
		cfg = zap.NewProductionConfig()
	}

	if logFile != "" {
		cfg.OutputPaths = []string{logFile}
		cfg.ErrorOutputPaths = []string{logFile}
	}

	logger, err := cfg.Build()
	if err != nil {
		return nil, err
	}

	zap.ReplaceGlobals(logger)

	return logger, nil
}
