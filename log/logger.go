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
	//slog.New(slog.NewTextHandler(io.Discard, nil))
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

//nolint:gocritic //need to implement interface
func (h *ContextHandler) handle(ctx context.Context, r slog.Record) error {
	for _, keyName := range h.keyProvider() {
		value := ctx.Value(keyName)
		if value == nil {
			continue
		}
		r.AddAttrs(slog.Attr{Key: string(keyName), Value: slog.AnyValue(value)})
	}

	return h.Handler.Handle(ctx, r)
}

func (h *ContextHandler) Handle(ctx context.Context, r slog.Record) error {
	return h.Handle(ctx, r)
}
