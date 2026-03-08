// Package middleware provides HTTP middleware for MTClaw.
package middleware

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/Minh-Tam-Solution/MTClaw/internal/store"
)

// SetTenantInTx executes SET LOCAL app.tenant_id within a transaction.
// Call this immediately after BeginTx in any store method that needs
// tenant-scoped queries. RLS policies reference current_setting('app.tenant_id', true).
//
// The SET LOCAL is scoped to the current transaction and auto-resets on COMMIT/ROLLBACK,
// making it safe for connection pooling (PgBouncer transaction mode).
func SetTenantInTx(ctx context.Context, tx *sql.Tx) error {
	tenantID := store.TenantIDFromContext(ctx)
	if tenantID == "" {
		return nil // No tenant in context — admin/migration mode (RLS returns 0 rows)
	}
	_, err := tx.ExecContext(ctx, "SET LOCAL app.tenant_id = $1", tenantID)
	if err != nil {
		return fmt.Errorf("set tenant_id: %w", err)
	}
	return nil
}

// SetTenantOnConn executes SET app.tenant_id on a dedicated connection.
// Use this for non-transactional queries that still need tenant isolation.
// The caller must close the connection when done.
func SetTenantOnConn(ctx context.Context, conn *sql.Conn) error {
	tenantID := store.TenantIDFromContext(ctx)
	if tenantID == "" {
		return nil
	}
	_, err := conn.ExecContext(ctx, "SET app.tenant_id = $1", tenantID)
	if err != nil {
		return fmt.Errorf("set tenant_id on conn: %w", err)
	}
	return nil
}

// TenantHTTPMiddleware injects the tenant ID into the request context.
// For Phase 1 (MTS-only), defaultTenantID is used when no X-Tenant-ID header is present.
// For Phase 2 (multi-tenant), the header becomes required.
func TenantHTTPMiddleware(defaultTenantID string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tenantID := r.Header.Get("X-Tenant-ID")
			if tenantID == "" {
				tenantID = defaultTenantID
			}
			if tenantID == "" {
				slog.Warn("tenant.missing", "path", r.URL.Path)
				http.Error(w, `{"error":"missing tenant ID"}`, http.StatusBadRequest)
				return
			}

			ctx := store.WithTenantID(r.Context(), tenantID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// TenantHandlerMiddleware is the same as TenantHTTPMiddleware but wraps
// an http.HandlerFunc directly (matching GoClaw's per-handler middleware pattern).
func TenantHandlerMiddleware(defaultTenantID string, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		tenantID := r.Header.Get("X-Tenant-ID")
		if tenantID == "" {
			tenantID = defaultTenantID
		}
		if tenantID == "" {
			slog.Warn("tenant.missing", "path", r.URL.Path)
			http.Error(w, `{"error":"missing tenant ID"}`, http.StatusBadRequest)
			return
		}

		ctx := store.WithTenantID(r.Context(), tenantID)
		next(w, r.WithContext(ctx))
	}
}

// BeginTenantTx starts a transaction with tenant isolation.
// Combines BeginTx + SET LOCAL app.tenant_id in one call for convenience.
func BeginTenantTx(ctx context.Context, db *sql.DB, opts *sql.TxOptions) (*sql.Tx, error) {
	tx, err := db.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	if err := SetTenantInTx(ctx, tx); err != nil {
		tx.Rollback()
		return nil, err
	}
	return tx, nil
}
