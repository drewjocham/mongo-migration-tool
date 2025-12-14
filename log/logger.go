package log

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
)

type LoggerContextKey string

type ContextKeyProvider func() []LoggerContextKey

func defaultProvider() []LoggerContextKey {
	return []LoggerContextKey{}
}

// TODO: replace string with the type log.Level
func CustomLogger(levelStr string, keyProvider ContextKeyProvider) *slog.Logger {
	options := &slog.HandlerOptions{}
	switch strings.ToUpper(levelStr) {
	case "DEBUG_SOURCE":
		options.Level = slog.LevelDebug
	case "DEBUG":
		options.Level = slog.LevelDebug
	case "WARN":
		options.Level = slog.LevelWarn
	case "ERROR":
		options.Level = slog.LevelError
	case "INFO":
		options.Level = slog.LevelInfo
	default:
		return slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	options.AddSource = true
	if keyProvider != nil {
		return slog.New(customHandler(slog.NewJSONHandler(os.Stdout, options), keyProvider))
	}

	return slog.New(customHandler(slog.NewJSONHandler(os.Stdout, options), defaultProvider))
}

type ContextHandler struct {
	slog.Handler
	keyProvider ContextKeyProvider
}

func customHandler(baseHandler slog.Handler, keyProvider ContextKeyProvider) slog.Handler {
	return ContextHandler{baseHandler, keyProvider}
}

//nolint:gocritic //need to implement interface
func (h ContextHandler) Handle(ctx context.Context, r slog.Record) error {
	for _, keyName := range h.keyProvider() {
		value := ctx.Value(keyName)
		if value == nil {
			continue
		}
		r.AddAttrs(slog.Attr{Key: string(keyName), Value: slog.AnyValue(value)})
	}

	return h.Handler.Handle(ctx, r)
}
