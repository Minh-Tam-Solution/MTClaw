package claudecode

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAuditWriter_WriteJSONL(t *testing.T) {
	dir := t.TempDir()
	w, err := NewAuditWriter(dir, nil)
	if err != nil {
		t.Fatalf("NewAuditWriter: %v", err)
	}
	defer w.Close()

	event := AuditEvent{
		OwnerID:   "tenant-1",
		SessionID: "br:abc12345:def67890",
		ActorID:   "user-123",
		Action:    "session.created",
		RiskMode:  "read",
		Detail:    map[string]interface{}{"project": "/tmp/myproject"},
		CreatedAt: time.Now(),
	}

	if err := w.Write(event); err != nil {
		t.Fatalf("Write: %v", err)
	}

	// Read the file back
	files, _ := filepath.Glob(filepath.Join(dir, "bridge-audit-*.jsonl"))
	if len(files) != 1 {
		t.Fatalf("expected 1 audit file, got %d", len(files))
	}

	data, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatalf("read audit file: %v", err)
	}

	lines := strings.TrimSpace(string(data))
	if lines == "" {
		t.Fatal("audit file is empty")
	}

	var decoded AuditEvent
	if err := json.Unmarshal([]byte(lines), &decoded); err != nil {
		t.Fatalf("unmarshal audit line: %v", err)
	}

	if decoded.Action != "session.created" {
		t.Errorf("action: got %q, want %q", decoded.Action, "session.created")
	}
	if decoded.OwnerID != "tenant-1" {
		t.Errorf("owner_id: got %q, want %q", decoded.OwnerID, "tenant-1")
	}
}

func TestAuditWriter_MultipleEvents(t *testing.T) {
	dir := t.TempDir()
	w, err := NewAuditWriter(dir, nil)
	if err != nil {
		t.Fatalf("NewAuditWriter: %v", err)
	}
	defer w.Close()

	for i := 0; i < 5; i++ {
		event := AuditEvent{
			OwnerID:  "tenant-1",
			ActorID:  "user-123",
			Action:   "test.event",
			Detail:   map[string]interface{}{"index": i},
		}
		if err := w.Write(event); err != nil {
			t.Fatalf("Write[%d]: %v", i, err)
		}
	}

	files, _ := filepath.Glob(filepath.Join(dir, "bridge-audit-*.jsonl"))
	data, _ := os.ReadFile(files[0])
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 5 {
		t.Errorf("expected 5 lines, got %d", len(lines))
	}
}

func TestAuditWriter_Writable(t *testing.T) {
	dir := t.TempDir()
	w, err := NewAuditWriter(dir, nil)
	if err != nil {
		t.Fatalf("NewAuditWriter: %v", err)
	}
	defer w.Close()

	if err := w.Writable(); err != nil {
		t.Errorf("should be writable: %v", err)
	}
}

func TestAuditWriter_DefaultDir(t *testing.T) {
	// Empty dir should resolve to ~/.mtclaw/bridge-audit/
	w, err := NewAuditWriter("", nil)
	if err != nil {
		t.Fatalf("NewAuditWriter with empty dir: %v", err)
	}
	defer w.Close()

	home, _ := os.UserHomeDir()
	expected := filepath.Join(home, ".mtclaw", "bridge-audit")
	if !strings.HasPrefix(w.dir, expected) {
		t.Errorf("dir: got %q, want prefix %q", w.dir, expected)
	}
}

func TestAuditWriter_NilDB_NoPanic(t *testing.T) {
	dir := t.TempDir()
	w, err := NewAuditWriter(dir, nil)
	if err != nil {
		t.Fatalf("NewAuditWriter: %v", err)
	}
	defer w.Close()

	// Should write JSONL without PG, no panic
	event := AuditEvent{
		OwnerID: "tenant-1",
		ActorID: "user-123",
		Action:  "test.nil_db",
	}
	if err := w.Write(event); err != nil {
		t.Fatalf("Write with nil DB should succeed: %v", err)
	}
}
