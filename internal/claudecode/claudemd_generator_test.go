package claudecode

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectProjectProfile_Go(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test"), 0644)

	profile := DetectProjectProfile(dir)
	if profile.Language != "go" {
		t.Errorf("expected go, got %q", profile.Language)
	}
	if profile.TestCmd != "go test ./... -v -count=1" {
		t.Errorf("unexpected test cmd: %q", profile.TestCmd)
	}
}

func TestDetectProjectProfile_TypeScript(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "package.json"), []byte("{}"), 0644)

	profile := DetectProjectProfile(dir)
	if profile.Language != "typescript" {
		t.Errorf("expected typescript, got %q", profile.Language)
	}
}

func TestDetectProjectProfile_Python(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte("[project]"), 0644)

	profile := DetectProjectProfile(dir)
	if profile.Language != "python" {
		t.Errorf("expected python, got %q", profile.Language)
	}
	if profile.TestCmd != "pytest -v" {
		t.Errorf("unexpected test cmd: %q", profile.TestCmd)
	}
}

func TestDetectProjectProfile_Makefile(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test"), 0644)
	os.WriteFile(filepath.Join(dir, "Makefile"), []byte("build:\n\tgo build\ntest:\n\tgo test ./..."), 0644)

	profile := DetectProjectProfile(dir)
	if !profile.HasMakefile {
		t.Error("expected HasMakefile=true")
	}
	if profile.BuildCmd != "make build" {
		t.Errorf("expected make build, got %q", profile.BuildCmd)
	}
	if profile.TestCmd != "make test" {
		t.Errorf("expected make test, got %q", profile.TestCmd)
	}
}

func TestDetectProjectProfile_Docker(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM alpine"), 0644)

	profile := DetectProjectProfile(dir)
	if !profile.HasDocker {
		t.Error("expected HasDocker=true")
	}
}

func TestGenerateClaudeMD_Content(t *testing.T) {
	profile := &ProjectProfile{
		Name:        "myproject",
		Language:    "go",
		BuildCmd:    "make build",
		TestCmd:     "make test",
		HasMakefile: true,
		HasDocker:   true,
		GitRemote:   "github.com/org/myproject",
	}

	content := GenerateClaudeMD(profile)
	if !strings.HasPrefix(content, claudeMDGeneratedHeader) {
		t.Error("should start with generated header")
	}
	if !strings.Contains(content, "# myproject") {
		t.Error("should contain project name heading")
	}
	if !strings.Contains(content, "make build") {
		t.Error("should contain build command")
	}
	if !strings.Contains(content, "make test") {
		t.Error("should contain test command")
	}
	if !strings.Contains(content, "Language: go") {
		t.Error("should contain language")
	}
	if !strings.Contains(content, "Docker") {
		t.Error("should contain Docker reference")
	}

	// Under 100 lines
	lines := strings.Count(content, "\n")
	if lines > 100 {
		t.Errorf("CLAUDE.md should be under 100 lines, got %d", lines)
	}
}

func TestInitProject_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test"), 0644)

	created, err := InitProject(dir, false)
	if err != nil {
		t.Fatalf("InitProject: %v", err)
	}
	if !created {
		t.Error("expected file to be created")
	}

	data, err := os.ReadFile(filepath.Join(dir, "CLAUDE.md"))
	if err != nil {
		t.Fatalf("read CLAUDE.md: %v", err)
	}
	if !strings.HasPrefix(string(data), claudeMDGeneratedHeader) {
		t.Error("should have generated header")
	}
}

func TestInitProject_SkipsUserFile(t *testing.T) {
	dir := t.TempDir()
	claudeFile := filepath.Join(dir, "CLAUDE.md")
	os.WriteFile(claudeFile, []byte("# My custom CLAUDE.md"), 0644)

	created, err := InitProject(dir, false)
	if err != nil {
		t.Fatalf("InitProject: %v", err)
	}
	if created {
		t.Error("should skip user-owned file")
	}

	data, _ := os.ReadFile(claudeFile)
	if !strings.Contains(string(data), "My custom") {
		t.Error("user file should be preserved")
	}
}

func TestInitProject_ForceOverwrite(t *testing.T) {
	dir := t.TempDir()
	claudeFile := filepath.Join(dir, "CLAUDE.md")
	os.WriteFile(claudeFile, []byte("# My custom CLAUDE.md"), 0644)

	created, err := InitProject(dir, true)
	if err != nil {
		t.Fatalf("InitProject force: %v", err)
	}
	if !created {
		t.Error("force should create/overwrite")
	}

	data, _ := os.ReadFile(claudeFile)
	if !strings.HasPrefix(string(data), claudeMDGeneratedHeader) {
		t.Error("force should overwrite with generated content")
	}
}
