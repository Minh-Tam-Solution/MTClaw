package claudecode

import (
	"os"
	"strings"
	"testing"
)

func TestProjectRegistry_RegisterAndGet(t *testing.T) {
	r := NewProjectRegistry()
	tmpDir := t.TempDir()

	p, err := r.Register("owner-1", "myproject", tmpDir, AgentClaudeCode)
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if p.Name != "myproject" {
		t.Errorf("Name: got %q, want %q", p.Name, "myproject")
	}
	if p.OwnerID != "owner-1" {
		t.Errorf("OwnerID: got %q, want %q", p.OwnerID, "owner-1")
	}
	if !strings.HasPrefix(p.ID, "proj:") {
		t.Errorf("ID should start with 'proj:', got %q", p.ID)
	}

	got, ok := r.Get("owner-1", "myproject")
	if !ok {
		t.Fatal("Get: project not found")
	}
	if got.Path != p.Path {
		t.Errorf("Path mismatch: got %q, want %q", got.Path, p.Path)
	}
}

func TestProjectRegistry_GetNotFound(t *testing.T) {
	r := NewProjectRegistry()
	_, ok := r.Get("owner-1", "nonexistent")
	if ok {
		t.Error("expected project not found")
	}
}

func TestProjectRegistry_List(t *testing.T) {
	r := NewProjectRegistry()
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	r.Register("owner-1", "proj-a", dir1, AgentClaudeCode)
	r.Register("owner-1", "proj-b", dir2, AgentClaudeCode)
	r.Register("owner-2", "proj-c", dir1, AgentClaudeCode)

	list := r.List("owner-1")
	if len(list) != 2 {
		t.Errorf("List owner-1: got %d projects, want 2", len(list))
	}

	list2 := r.List("owner-2")
	if len(list2) != 1 {
		t.Errorf("List owner-2: got %d projects, want 1", len(list2))
	}

	list3 := r.List("owner-3")
	if len(list3) != 0 {
		t.Errorf("List owner-3: got %d projects, want 0", len(list3))
	}
}

func TestProjectRegistry_Delete(t *testing.T) {
	r := NewProjectRegistry()
	tmpDir := t.TempDir()
	r.Register("owner-1", "proj-a", tmpDir, AgentClaudeCode)

	if !r.Delete("owner-1", "proj-a") {
		t.Error("Delete should return true for existing project")
	}
	if r.Delete("owner-1", "proj-a") {
		t.Error("Delete should return false for already-deleted project")
	}
	_, ok := r.Get("owner-1", "proj-a")
	if ok {
		t.Error("project should not exist after delete")
	}
}

func TestProjectRegistry_RegisterInvalidPath(t *testing.T) {
	r := NewProjectRegistry()
	_, err := r.Register("owner-1", "bad", "/nonexistent/path/12345", AgentClaudeCode)
	if err == nil {
		t.Error("expected error for nonexistent path")
	}
}

func TestProjectRegistry_RegisterFilePath(t *testing.T) {
	r := NewProjectRegistry()
	tmpFile := t.TempDir() + "/file.txt"
	os.WriteFile(tmpFile, []byte("test"), 0o644)

	_, err := r.Register("owner-1", "bad", tmpFile, AgentClaudeCode)
	if err == nil {
		t.Error("expected error for file path (not directory)")
	}
}

func TestComputeWorkspaceFingerprint_Deterministic(t *testing.T) {
	tmpDir := t.TempDir()

	fp1, err := ComputeWorkspaceFingerprint(tmpDir, "tenant-1")
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	fp2, err := ComputeWorkspaceFingerprint(tmpDir, "tenant-1")
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if fp1 != fp2 {
		t.Errorf("fingerprint should be deterministic: %q != %q", fp1, fp2)
	}
	if len(fp1) != 64 {
		t.Errorf("fingerprint should be 64 hex chars (sha256), got %d", len(fp1))
	}
}

func TestComputeWorkspaceFingerprint_DifferentTenants(t *testing.T) {
	tmpDir := t.TempDir()

	fp1, _ := ComputeWorkspaceFingerprint(tmpDir, "tenant-1")
	fp2, _ := ComputeWorkspaceFingerprint(tmpDir, "tenant-2")
	if fp1 == fp2 {
		t.Error("different tenants should produce different fingerprints")
	}
}

func TestComputeWorkspaceFingerprint_DifferentPaths(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()

	fp1, _ := ComputeWorkspaceFingerprint(dir1, "tenant-1")
	fp2, _ := ComputeWorkspaceFingerprint(dir2, "tenant-1")
	if fp1 == fp2 {
		t.Error("different paths should produce different fingerprints")
	}
}
