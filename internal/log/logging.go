package logs

import (
	"io"
	"log/slog"
	"os"
)

func New(debug bool, logFile string) (*slog.Logger, error) {
	var writers []io.Writer
	writers = append(writers, os.Stderr)

	if logFile != "" {
		f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
		if err != nil {
			return nil, err
		}
		writers = append(writers, f)
	}

	handler := slog.NewTextHandler(io.MultiWriter(writers...), &slog.HandlerOptions{
		Level: chooseLevel(debug),
	})
	ctxHandler := &ContextHandler{Handler: handler}
	logger := slog.New(ctxHandler)
	slog.SetDefault(logger)
	return logger, nil
}

func chooseLevel(debug bool) slog.Level {
	if debug {
		return slog.LevelDebug
	}
	return slog.LevelInfo
}
