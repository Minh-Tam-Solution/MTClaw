package claudecode

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

const gitCmdTimeout = 5 * time.Second

// ProjectRegistry manages registered project directories for bridge sessions.
type ProjectRegistry struct {
	mu       sync.RWMutex
	projects map[string]*BridgeProject // keyed by "ownerID:name"
}

// NewProjectRegistry creates an empty project registry.
func NewProjectRegistry() *ProjectRegistry {
	return &ProjectRegistry{
		projects: make(map[string]*BridgeProject),
	}
}

// Register adds or updates a project.
func (r *ProjectRegistry) Register(ownerID, name, path string, agentType AgentProviderType) (*BridgeProject, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve path: %w", err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("path does not exist: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("path is not a directory: %s", absPath)
	}

	key := projectKey(ownerID, name)

	r.mu.Lock()
	defer r.mu.Unlock()

	id, err := GenerateSessionID(ownerID)
	if err != nil {
		return nil, err
	}

	p := &BridgeProject{
		ID:        "proj:" + id[3:], // reuse the tenant:rand portion
		OwnerID:   ownerID,
		Name:      name,
		Path:      absPath,
		AgentType: agentType,
		CreatedAt: time.Now(),
	}
	r.projects[key] = p
	return p, nil
}

// Get returns a project by owner and name.
func (r *ProjectRegistry) Get(ownerID, name string) (*BridgeProject, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.projects[projectKey(ownerID, name)]
	return p, ok
}

// List returns all projects for an owner.
func (r *ProjectRegistry) List(ownerID string) []*BridgeProject {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*BridgeProject
	prefix := ownerID + ":"
	for key, p := range r.projects {
		if strings.HasPrefix(key, prefix) {
			result = append(result, p)
		}
	}
	return result
}

// Delete removes a project by owner and name.
func (r *ProjectRegistry) Delete(ownerID, name string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	key := projectKey(ownerID, name)
	if _, ok := r.projects[key]; ok {
		delete(r.projects, key)
		return true
	}
	return false
}

func projectKey(ownerID, name string) string {
	return ownerID + ":" + name
}

// ComputeWorkspaceFingerprint creates a composite fingerprint for a project path.
// Components: canonical_path + device_inode + git_root + remote_url_normalized + tenant_salt
// This prevents workspace confusion on symlinks, bind mounts, worktree clones,
// and multi-tenant shared hosts.
func ComputeWorkspaceFingerprint(projectPath, tenantID string) (string, error) {
	// 1. Canonical absolute path
	canonicalPath, err := filepath.EvalSymlinks(projectPath)
	if err != nil {
		canonicalPath = projectPath // fallback to raw path
	}
	canonicalPath, err = filepath.Abs(canonicalPath)
	if err != nil {
		return "", fmt.Errorf("resolve absolute path: %w", err)
	}

	// 2. Device + inode (detects bind mounts, hardlinks)
	deviceInode := "0:0"
	if info, err := os.Stat(canonicalPath); err == nil {
		if stat, ok := info.Sys().(*syscall.Stat_t); ok {
			deviceInode = fmt.Sprintf("%d:%d", stat.Dev, stat.Ino)
		}
	}

	// 3. Git root (detects worktree clones)
	gitRoot := gitRootDir(canonicalPath)

	// 4. Git remote URL (normalized — strip .git suffix, lowercase)
	remoteURL := normalizeGitRemote(canonicalPath)

	// 5. Tenant salt
	tenantSalt := fmt.Sprintf("%x", sha256.Sum256([]byte("tenant:"+tenantID)))[:16]

	// Compose and hash
	composite := strings.Join([]string{canonicalPath, deviceInode, gitRoot, remoteURL, tenantSalt}, ":")
	hash := sha256.Sum256([]byte(composite))
	return fmt.Sprintf("%x", hash), nil
}

// gitRootDir returns the git repository root, or empty string if not a git repo.
func gitRootDir(dir string) string {
	ctx, cancel := context.WithTimeout(context.Background(), gitCmdTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--show-toplevel")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// normalizeGitRemote returns the normalized origin remote URL.
func normalizeGitRemote(dir string) string {
	ctx, cancel := context.WithTimeout(context.Background(), gitCmdTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", "remote", "get-url", "origin")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	url := strings.TrimSpace(string(out))
	url = strings.TrimSuffix(url, ".git")
	url = strings.ToLower(url)
	return url
}
