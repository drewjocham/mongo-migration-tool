package logs

import (
	"context"
	"log/slog"
)

type LoggerContextKey string

type ContextKeyProvider func() []LoggerContextKey

func defaultProvider() []LoggerContextKey {
	var mcp LoggerContextKey = "migration-ctx"
	return []LoggerContextKey{
		mcp,
	}
}

type ContextHandler struct {
	slog.Handler
	keyProvider ContextKeyProvider
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
