package claudecode

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

// SoulContent holds loaded SOUL data with integrity hash.
type SoulContent struct {
	Role        string // "pm", "coder", etc.
	Category    string // from YAML frontmatter: "executor", "advisor", "router", "business"
	Body        string // markdown body (after frontmatter stripped)
	ContentHash string // SHA-256 of full file content
	SourcePath  string // absolute path to SOUL file
}

// soulRolesCache caches KnownRoles result for hot-path performance (CTO-L4).
// Protected by soulRolesMu to prevent race between KnownRoles reads and
// InvalidateRolesCache writes (CTO-100).
var (
	soulRolesMu    sync.RWMutex
	soulRolesCache struct {
		once  sync.Once
		dir   string
		roles []string
		err   error
	}
)

// KnownRoles returns valid SOUL role names by scanning soulsDir for SOUL-*.md files.
// Results are cached after the first call for the same directory (CTO-L4).
func KnownRoles(soulsDir string) ([]string, error) {
	soulRolesMu.RLock()
	soulRolesCache.once.Do(func() {
		// Upgrade to write within Do — safe because sync.Once serializes.
		// Fields are only written inside Do, reads outside are protected by RLock.
		soulRolesCache.dir = soulsDir
		soulRolesCache.roles, soulRolesCache.err = scanSoulRoles(soulsDir)
	})

	// If directory changed, re-scan (shouldn't happen in normal use).
	if soulRolesCache.dir != soulsDir {
		soulRolesMu.RUnlock()
		return scanSoulRoles(soulsDir)
	}

	roles, err := soulRolesCache.roles, soulRolesCache.err
	soulRolesMu.RUnlock()
	return roles, err
}

// InvalidateRolesCache resets the cached roles. Called after install-agents.
func InvalidateRolesCache() {
	soulRolesMu.Lock()
	soulRolesCache = struct {
		once  sync.Once
		dir   string
		roles []string
		err   error
	}{}
	soulRolesMu.Unlock()
}

func scanSoulRoles(soulsDir string) ([]string, error) {
	pattern := filepath.Join(soulsDir, "SOUL-*.md")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("scan SOUL files: %w", err)
	}

	roles := make([]string, 0, len(matches))
	for _, m := range matches {
		base := filepath.Base(m)
		// Extract role from "SOUL-{role}.md"
		role := strings.TrimPrefix(base, "SOUL-")
		role = strings.TrimSuffix(role, ".md")
		if role != "" {
			roles = append(roles, role)
		}
	}

	if len(roles) == 0 {
		return nil, fmt.Errorf("no SOUL files found in %s", soulsDir)
	}

	return roles, nil
}

// LoadSOUL reads SOUL-{role}.md from soulsDir, parses frontmatter, computes hash.
// Returns error if role unknown or file missing.
//
// Path traversal guard (CTO-B3):
//  1. Validate role against KnownRoles() allowlist
//  2. Construct path via filepath.Join
//  3. Verify resolved path is within soulsDir
func LoadSOUL(soulsDir, role string) (*SoulContent, error) {
	// Step 1: Validate role is known (prevents arbitrary path components)
	known, err := KnownRoles(soulsDir)
	if err != nil {
		return nil, fmt.Errorf("load SOUL: %w", err)
	}
	if !containsString(known, role) {
		return nil, fmt.Errorf("unknown role %q (known: %s)", role, strings.Join(known, ", "))
	}

	// Step 2: Construct path safely
	resolved := filepath.Join(soulsDir, "SOUL-"+role+".md")

	// Step 3: Double-check resolved is within soulsDir (defense-in-depth)
	cleanResolved := filepath.Clean(resolved)
	cleanDir := filepath.Clean(soulsDir)
	if !strings.HasPrefix(cleanResolved, cleanDir+string(filepath.Separator)) && cleanResolved != cleanDir {
		return nil, fmt.Errorf("path traversal detected for role %q", role)
	}

	// Read file
	data, err := os.ReadFile(resolved)
	if err != nil {
		return nil, fmt.Errorf("read SOUL file %s: %w", resolved, err)
	}

	// Compute hash of full content
	hash := sha256.Sum256(data)
	contentHash := hex.EncodeToString(hash[:])

	// Parse frontmatter
	content := string(data)
	fm := soulExtractFrontmatter(content)
	body := soulStripFrontmatter(content)

	kv := soulParseSimpleYAML(fm)

	return &SoulContent{
		Role:        kv["role"],
		Category:    kv["category"],
		Body:        strings.TrimSpace(body),
		ContentHash: contentHash,
		SourcePath:  resolved,
	}, nil
}

// HashFileContent computes SHA-256 hash of a file's contents.
func HashFileContent(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

func containsString(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

// --- Frontmatter parsing (duplicated from skills/loader.go per ADR-011 D13) ---

var soulFrontmatterRe = regexp.MustCompile(`(?s)^---\n(.*?)\n---\n?`)

func soulExtractFrontmatter(content string) string {
	match := soulFrontmatterRe.FindStringSubmatch(content)
	if len(match) > 1 {
		return match[1]
	}
	return ""
}

func soulStripFrontmatter(content string) string {
	return soulFrontmatterRe.ReplaceAllString(content, "")
}

func soulParseSimpleYAML(content string) map[string]string {
	result := make(map[string]string)
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			val := strings.TrimSpace(parts[1])
			val = strings.Trim(val, "\"'")
			result[key] = val
		}
	}
	return result
}
