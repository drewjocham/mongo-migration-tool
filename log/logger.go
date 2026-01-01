package log

import (
	"context"
	"log/slog"
	"os"
)

type LoggerContextKey string

type ContextKeyProvider func() []LoggerContextKey

func defaultProvider() []LoggerContextKey {
	var mcp LoggerContextKey = "migration-ctx"
	return []LoggerContextKey{
		mcp,
	}
}

func CustomLogger(level slog.Level, keyProvider ContextKeyProvider) *slog.Logger {
	options := &slog.HandlerOptions{}
	options.Level = level
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
	return &ContextHandler{baseHandler, keyProvider}
}

func (h *ContextHandler) Handle(ctx context.Context, r slog.Record) error {
	provider := h.keyProvider
	if provider == nil {
		provider = defaultProvider
	}

	for _, keyName := range provider() {
		value := ctx.Value(keyName)
		if value == nil {
			continue
		}

		r.AddAttrs(slog.Attr{Key: string(keyName), Value: slog.AnyValue(value)})
	}

	return h.Handler.Handle(ctx, r)
}
