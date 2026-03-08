package claudecode

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const defaultTmuxTimeout = 5 * time.Second

// TmuxSession represents a running tmux session.
type TmuxSession struct {
	Name      string
	WindowID  string
	Created   time.Time
	Attached  bool
}

// TmuxBridge manages tmux sessions for the Claude Code bridge.
type TmuxBridge struct {
	tmuxPath string
}

// NewTmuxBridge creates a bridge, verifying tmux is available.
func NewTmuxBridge() (*TmuxBridge, error) {
	path, err := exec.LookPath("tmux")
	if err != nil {
		return nil, fmt.Errorf("tmux not found in PATH: %w", err)
	}
	return &TmuxBridge{tmuxPath: path}, nil
}

// NewTmuxBridgeWithPath creates a bridge with a specific tmux binary path (for testing).
func NewTmuxBridgeWithPath(path string) *TmuxBridge {
	return &TmuxBridge{tmuxPath: path}
}

// CreateSession starts a new detached tmux session with the given name and working directory.
func (t *TmuxBridge) CreateSession(ctx context.Context, name, workdir string) error {
	if err := validateSessionName(name); err != nil {
		return err
	}
	args := []string{"new-session", "-d", "-s", name}
	if workdir != "" {
		args = append(args, "-c", workdir)
	}
	_, err := t.run(ctx, args...)
	if err != nil {
		return fmt.Errorf("create session %q: %w", name, err)
	}
	return nil
}

// KillSession terminates a tmux session.
func (t *TmuxBridge) KillSession(ctx context.Context, target string) error {
	_, err := t.run(ctx, "kill-session", "-t", target)
	if err != nil {
		return fmt.Errorf("kill session %q: %w", target, err)
	}
	return nil
}

// CapturePane captures the last N lines from a tmux pane.
func (t *TmuxBridge) CapturePane(ctx context.Context, target string, lines int) (string, error) {
	if lines <= 0 {
		lines = 30
	}
	startLine := fmt.Sprintf("-%d", lines)
	output, err := t.run(ctx, "capture-pane", "-p", "-t", target, "-S", startLine)
	if err != nil {
		return "", fmt.Errorf("capture pane %q: %w", target, err)
	}
	return output, nil
}

// SendKeys sends text to a tmux pane using paste-buffer for reliability.
// Uses tmux load-buffer + paste-buffer instead of send-keys to avoid
// character-by-character issues and escape sequence problems.
func (t *TmuxBridge) SendKeys(ctx context.Context, target, text string) error {
	if text == "" {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, defaultTmuxTimeout)
	defer cancel()

	// Load text into tmux buffer via stdin
	loadCmd := t.command(ctx, "load-buffer", "-")
	loadCmd.Stdin = strings.NewReader(text)
	if err := loadCmd.Run(); err != nil {
		return fmt.Errorf("load buffer for %q: %w", target, err)
	}

	// Paste buffer into target pane (reuse same context with remaining timeout)
	pasteCmd := t.command(ctx, "paste-buffer", "-t", target)
	if err := pasteCmd.Run(); err != nil {
		return fmt.Errorf("paste buffer to %q: %w", target, err)
	}
	return nil
}

// SendEnter sends an Enter keypress to the target pane.
func (t *TmuxBridge) SendEnter(ctx context.Context, target string) error {
	_, err := t.run(ctx, "send-keys", "-t", target, "Enter")
	if err != nil {
		return fmt.Errorf("send enter to %q: %w", target, err)
	}
	return nil
}

// ListSessions returns all running tmux sessions.
func (t *TmuxBridge) ListSessions(ctx context.Context) ([]TmuxSession, error) {
	output, err := t.run(ctx, "list-sessions", "-F", "#{session_name}|#{session_created}|#{session_attached}")
	if err != nil {
		// "no server running" means no sessions
		if strings.Contains(err.Error(), "no server running") || strings.Contains(err.Error(), "no sessions") {
			return nil, nil
		}
		return nil, fmt.Errorf("list sessions: %w", err)
	}

	var sessions []TmuxSession
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 3)
		if len(parts) < 3 {
			continue
		}
		sessions = append(sessions, TmuxSession{
			Name:     parts[0],
			Attached: parts[2] == "1",
		})
	}
	return sessions, nil
}

// SessionExists checks if a tmux session with the given name exists.
func (t *TmuxBridge) SessionExists(ctx context.Context, target string) (bool, error) {
	_, err := t.run(ctx, "has-session", "-t", target)
	if err != nil {
		// Exit code 1 = session doesn't exist (not an error)
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

// run executes a tmux command with timeout and returns stdout.
func (t *TmuxBridge) run(ctx context.Context, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, defaultTmuxTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, t.tmuxPath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		errMsg := strings.TrimSpace(stderr.String())
		if errMsg != "" {
			return "", fmt.Errorf("%w: %s", err, errMsg)
		}
		return "", err
	}
	return stdout.String(), nil
}

// command creates an exec.Cmd for tmux with context timeout.
// Caller is responsible for canceling the context when using this directly.
func (t *TmuxBridge) command(ctx context.Context, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, t.tmuxPath, args...)
}

// BuildSessionName creates a tmux session name from tenant and random parts.
// Format: cc-{tenant8}-{rand8} (max 21 chars, well under tmux 256 limit).
func BuildSessionName(tenantHash, randPart string) string {
	return fmt.Sprintf("cc-%s-%s", tenantHash, randPart)
}

// validateSessionName checks that a session name is safe for tmux.
func validateSessionName(name string) error {
	if name == "" {
		return fmt.Errorf("session name cannot be empty")
	}
	if len(name) > 256 {
		return fmt.Errorf("session name too long: %d chars (max 256)", len(name))
	}
	if strings.ContainsAny(name, ".:") {
		return fmt.Errorf("session name contains invalid characters (period or colon): %q", name)
	}
	return nil
}
