package middleware

import (
	"context"
	"log/slog"

	"github.com/Minh-Tam-Solution/MTClaw/internal/store"
)

// ContextLogAttrs returns slog attributes extracted from the context.
// Use this to enrich structured log entries with tenant_id, agent_key, and user_id.
//
// Example:
//
//	slog.InfoContext(ctx, "soul invoked",
//	    append(middleware.ContextLogAttrs(ctx),
//	        slog.String("action", "/spec"),
//	    )...,
//	)
func ContextLogAttrs(ctx context.Context) []any {
	var attrs []any

	if tid := store.TenantIDFromContext(ctx); tid != "" {
		attrs = append(attrs, slog.String("tenant_id", tid))
	}
	if ak := store.AgentKeyFromContext(ctx); ak != "" {
		attrs = append(attrs, slog.String("agent_key", ak))
	}
	if uid := store.UserIDFromContext(ctx); uid != "" {
		attrs = append(attrs, slog.String("user_id", uid))
	}

	return attrs
}

// LogInfoCtx logs an info message with context fields automatically included.
func LogInfoCtx(ctx context.Context, msg string, args ...any) {
	args = append(ContextLogAttrs(ctx), args...)
	slog.InfoContext(ctx, msg, args...)
}

// LogWarnCtx logs a warning message with context fields automatically included.
func LogWarnCtx(ctx context.Context, msg string, args ...any) {
	args = append(ContextLogAttrs(ctx), args...)
	slog.WarnContext(ctx, msg, args...)
}

// LogErrorCtx logs an error message with context fields automatically included.
func LogErrorCtx(ctx context.Context, msg string, args ...any) {
	args = append(ContextLogAttrs(ctx), args...)
	slog.ErrorContext(ctx, msg, args...)
}
