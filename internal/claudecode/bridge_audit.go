package claudecode

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// AuditWriter provides dual-write audit logging (L3).
// Primary: JSONL file (append-only, never skipped).
// Secondary: PostgreSQL (best-effort, failure logged but not fatal).
type AuditWriter struct {
	mu      sync.Mutex
	dir     string // JSONL output directory
	file    *os.File
	db      *sql.DB // nil = standalone mode (no PG secondary)
	encoder *json.Encoder
}

// NewAuditWriter creates an audit writer.
// dir: directory for JSONL files. db: optional PostgreSQL connection (nil for standalone).
func NewAuditWriter(dir string, db *sql.DB) (*AuditWriter, error) {
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".mtclaw", "bridge-audit")
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("create audit dir %s: %w", dir, err)
	}

	logPath := filepath.Join(dir, auditFileName())
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("open audit file %s: %w", logPath, err)
	}

	return &AuditWriter{
		dir:     dir,
		file:    f,
		db:      db,
		encoder: json.NewEncoder(f),
	}, nil
}

// auditFileName returns date-based JSONL filename: bridge-audit-2026-03-07.jsonl
func auditFileName() string {
	return fmt.Sprintf("bridge-audit-%s.jsonl", time.Now().Format("2006-01-02"))
}

// Write records an audit event. JSONL write is mandatory; PG is best-effort.
func (w *AuditWriter) Write(event AuditEvent) error {
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}

	// Primary: JSONL (never skipped)
	if err := w.writeJSONL(event); err != nil {
		return fmt.Errorf("audit JSONL write failed (critical): %w", err)
	}

	// Secondary: PostgreSQL (best-effort)
	if w.db != nil {
		if err := w.writePG(event); err != nil {
			slog.Warn("audit PG write failed (best-effort)", "error", err, "action", event.Action, "session", event.SessionID)
		}
	}

	return nil
}

// writeJSONL appends a single event as a JSON line.
func (w *AuditWriter) writeJSONL(event AuditEvent) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Rotate file if date changed
	expected := auditFileName()
	current := filepath.Base(w.file.Name())
	if current != expected {
		w.file.Close()
		logPath := filepath.Join(w.dir, expected)
		f, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			return err
		}
		w.file = f
		w.encoder = json.NewEncoder(f)
	}

	return w.encoder.Encode(event)
}

// writePG inserts an audit event into bridge_audit_events (best-effort).
func (w *AuditWriter) writePG(event AuditEvent) error {
	detailJSON, err := json.Marshal(event.Detail)
	if err != nil {
		detailJSON = []byte("{}")
	}

	_, err = w.db.Exec(
		`INSERT INTO bridge_audit_events (owner_id, session_id, actor_id, action, risk_mode, detail, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		event.OwnerID, event.SessionID, event.ActorID, event.Action, event.RiskMode, detailJSON, event.CreatedAt,
	)
	return err
}

// Close flushes and closes the JSONL file.
func (w *AuditWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file != nil {
		return w.file.Close()
	}
	return nil
}

// Writable returns nil if the audit directory and file are writable.
func (w *AuditWriter) Writable() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return fmt.Errorf("audit file not open")
	}
	// Test write
	info, err := w.file.Stat()
	if err != nil {
		return fmt.Errorf("stat audit file: %w", err)
	}
	if info.Mode().Perm()&0200 == 0 {
		return fmt.Errorf("audit file not writable: %s", w.file.Name())
	}
	return nil
}
