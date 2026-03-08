package claudecode

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testSoulsDir = "../../docs/08-collaborate/souls"

func TestKnownRoles(t *testing.T) {
	InvalidateRolesCache()

	roles, err := KnownRoles(testSoulsDir)
	if err != nil {
		t.Fatalf("KnownRoles: %v", err)
	}
	if len(roles) == 0 {
		t.Fatal("expected at least 1 role")
	}

	// Verify known roles include coder and pm
	found := map[string]bool{}
	for _, r := range roles {
		found[r] = true
	}
	for _, want := range []string{"coder", "pm", "architect", "cto"} {
		if !found[want] {
			t.Errorf("expected role %q in KnownRoles, got %v", want, roles)
		}
	}
}

func TestKnownRoles_MissingDir(t *testing.T) {
	InvalidateRolesCache()

	_, err := KnownRoles("/nonexistent/path")
	if err == nil {
		t.Fatal("expected error for missing directory")
	}
	if !strings.Contains(err.Error(), "no SOUL files found") {
		t.Errorf("expected 'no SOUL files found' error, got: %v", err)
	}
}

func TestKnownRoles_Cache(t *testing.T) {
	InvalidateRolesCache()

	roles1, err := KnownRoles(testSoulsDir)
	if err != nil {
		t.Fatalf("first call: %v", err)
	}
	roles2, err := KnownRoles(testSoulsDir)
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if len(roles1) != len(roles2) {
		t.Errorf("cache mismatch: %d vs %d", len(roles1), len(roles2))
	}
}

func TestLoadSOUL_Coder(t *testing.T) {
	InvalidateRolesCache()

	soul, err := LoadSOUL(testSoulsDir, "coder")
	if err != nil {
		t.Fatalf("LoadSOUL(coder): %v", err)
	}
	if soul.Role != "coder" {
		t.Errorf("role: got %q, want coder", soul.Role)
	}
	if soul.Category != "executor" {
		t.Errorf("category: got %q, want executor", soul.Category)
	}
	if soul.Body == "" {
		t.Error("body should not be empty")
	}
	if soul.ContentHash == "" {
		t.Error("content hash should not be empty")
	}
	if len(soul.ContentHash) != 64 {
		t.Errorf("hash length: got %d, want 64 (SHA-256 hex)", len(soul.ContentHash))
	}
	if soul.SourcePath == "" {
		t.Error("source path should not be empty")
	}
}

func TestLoadSOUL_UnknownRole(t *testing.T) {
	InvalidateRolesCache()

	_, err := LoadSOUL(testSoulsDir, "nonexistent-role")
	if err == nil {
		t.Fatal("expected error for unknown role")
	}
	if !strings.Contains(err.Error(), "unknown role") {
		t.Errorf("expected 'unknown role' error, got: %v", err)
	}
}

func TestLoadSOUL_PathTraversal(t *testing.T) {
	InvalidateRolesCache()

	_, err := LoadSOUL(testSoulsDir, "../../../etc/passwd")
	if err == nil {
		t.Fatal("expected error for path traversal attempt")
	}
}

func TestLoadSOUL_AllRoles(t *testing.T) {
	InvalidateRolesCache()

	roles, err := KnownRoles(testSoulsDir)
	if err != nil {
		t.Fatalf("KnownRoles: %v", err)
	}

	for _, role := range roles {
		soul, err := LoadSOUL(testSoulsDir, role)
		if err != nil {
			t.Errorf("LoadSOUL(%s): %v", role, err)
			continue
		}
		if soul.Body == "" {
			t.Errorf("LoadSOUL(%s): empty body", role)
		}
		if soul.ContentHash == "" {
			t.Errorf("LoadSOUL(%s): empty hash", role)
		}
	}
}

func TestHashFileContent(t *testing.T) {
	tmpFile := filepath.Join(t.TempDir(), "test.txt")
	if err := os.WriteFile(tmpFile, []byte("hello world"), 0600); err != nil {
		t.Fatal(err)
	}

	hash, err := HashFileContent(tmpFile)
	if err != nil {
		t.Fatalf("HashFileContent: %v", err)
	}
	if len(hash) != 64 {
		t.Errorf("hash length: got %d, want 64", len(hash))
	}

	// Same content = same hash
	hash2, _ := HashFileContent(tmpFile)
	if hash != hash2 {
		t.Error("same file should produce same hash")
	}
}

func TestHashFileContent_MissingFile(t *testing.T) {
	_, err := HashFileContent("/nonexistent/file")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestSoulFrontmatterParsing(t *testing.T) {
	content := "---\nrole: pm\ncategory: advisor\nversion: 1.0.0\n---\n\n# PM SOUL\n\nBody content here."

	fm := soulExtractFrontmatter(content)
	if fm == "" {
		t.Fatal("expected frontmatter")
	}

	kv := soulParseSimpleYAML(fm)
	if kv["role"] != "pm" {
		t.Errorf("role: got %q, want pm", kv["role"])
	}
	if kv["category"] != "advisor" {
		t.Errorf("category: got %q, want advisor", kv["category"])
	}

	body := soulStripFrontmatter(content)
	if strings.Contains(body, "---") {
		t.Error("body should not contain frontmatter delimiters")
	}
	if !strings.Contains(body, "Body content here") {
		t.Error("body should contain markdown content")
	}
}

func TestInvalidateRolesCache(t *testing.T) {
	InvalidateRolesCache()
	// Should be able to call KnownRoles again without error
	_, err := KnownRoles(testSoulsDir)
	if err != nil {
		t.Fatalf("KnownRoles after invalidate: %v", err)
	}
	InvalidateRolesCache() // cleanup
}
