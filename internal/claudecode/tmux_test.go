package claudecode

import (
	"context"
	"testing"
)

func TestValidateSessionName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid short", "cc-abcd1234-efgh5678", false},
		{"valid minimal", "a", false},
		{"empty", "", true},
		{"too long", string(make([]byte, 257)), true},
		{"contains period", "my.session", true},
		{"contains colon", "my:session", true},
		{"hyphens ok", "cc-test-123", false},
		{"underscores ok", "cc_test_123", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSessionName(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateSessionName(%q): got err=%v, wantErr=%v", tt.input, err, tt.wantErr)
			}
		})
	}
}

func TestBuildSessionName(t *testing.T) {
	name := BuildSessionName("abcd1234", "efgh5678")
	want := "cc-abcd1234-efgh5678"
	if name != want {
		t.Errorf("BuildSessionName: got %q, want %q", name, want)
	}
}

func TestBuildSessionName_Length(t *testing.T) {
	// "cc-" (3) + 8-char tenant hash + "-" (1) + 8-char random = 20 chars
	name := BuildSessionName("12345678", "abcdefgh")
	if len(name) != 20 {
		t.Errorf("session name length: got %d, want 20", len(name))
	}
	if err := validateSessionName(name); err != nil {
		t.Errorf("generated name should be valid: %v", err)
	}
}

func TestNewTmuxBridgeWithPath(t *testing.T) {
	b := NewTmuxBridgeWithPath("/usr/bin/tmux")
	if b.tmuxPath != "/usr/bin/tmux" {
		t.Errorf("tmuxPath: got %q, want /usr/bin/tmux", b.tmuxPath)
	}
}

func TestCapturePane_DefaultLines(t *testing.T) {
	// Verify that lines <= 0 defaults to 30
	// We can't run a real tmux command, but we can verify the logic path
	// by checking the function doesn't panic with 0 lines
	b := NewTmuxBridgeWithPath("/nonexistent/tmux")
	_, err := b.CapturePane(context.Background(), "fake-target", 0)
	if err == nil {
		t.Error("expected error with nonexistent tmux binary")
	}
	_, err = b.CapturePane(context.Background(), "fake-target", -5)
	if err == nil {
		t.Error("expected error with nonexistent tmux binary")
	}
}

func TestSendKeys_EmptyText(t *testing.T) {
	b := NewTmuxBridgeWithPath("/nonexistent/tmux")
	err := b.SendKeys(context.Background(), "fake-target", "")
	if err != nil {
		t.Errorf("SendKeys with empty text should be no-op, got: %v", err)
	}
}

func TestSessionExists_NonexistentBinary(t *testing.T) {
	b := NewTmuxBridgeWithPath("/nonexistent/tmux")
	_, err := b.SessionExists(context.Background(), "fake")
	if err == nil {
		t.Error("expected error with nonexistent tmux binary")
	}
}

func TestListSessions_NonexistentBinary(t *testing.T) {
	b := NewTmuxBridgeWithPath("/nonexistent/tmux")
	_, err := b.ListSessions(context.Background())
	if err == nil {
		t.Error("expected error with nonexistent tmux binary")
	}
}
