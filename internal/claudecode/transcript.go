package claudecode

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// TranscriptEntry represents a single NDJSON line from Claude Code's transcript.
type TranscriptEntry struct {
	Type      string `json:"type"`                // "assistant", "user", "tool_use", "tool_result"
	Content   string `json:"content,omitempty"`    // text content
	ToolName  string `json:"tool_name,omitempty"`  // for tool_use/tool_result
	ToolInput string `json:"tool_input,omitempty"` // for tool_use
	IsError   bool   `json:"is_error,omitempty"`   // for tool_result errors
}

// ParseTranscript reads NDJSON lines from a reader and returns parsed entries.
// Malformed lines are skipped (best-effort parsing for robustness).
func ParseTranscript(r io.Reader) ([]TranscriptEntry, error) {
	var entries []TranscriptEntry
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 256*1024), 1<<20) // up to 1MB per line

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var entry TranscriptEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue // skip malformed lines
		}
		entries = append(entries, entry)
	}

	if err := scanner.Err(); err != nil {
		return entries, fmt.Errorf("transcript scan: %w", err)
	}
	return entries, nil
}

// SummarizeTranscript creates a human-readable summary from transcript entries.
// Returns: assistant message count, tool calls, errors, and a short text summary.
func SummarizeTranscript(entries []TranscriptEntry) TranscriptSummary {
	s := TranscriptSummary{}

	for _, e := range entries {
		switch e.Type {
		case "assistant":
			s.AssistantMessages++
			// Capture last assistant message as the summary
			if e.Content != "" {
				s.LastAssistantMsg = e.Content
			}
		case "user":
			s.UserMessages++
		case "tool_use":
			s.ToolCalls++
			s.ToolNames = appendUnique(s.ToolNames, e.ToolName)
		case "tool_result":
			if e.IsError {
				s.ToolErrors++
			}
		}
	}

	s.TotalEntries = len(entries)
	return s
}

// TranscriptSummary holds aggregated statistics from a transcript.
type TranscriptSummary struct {
	TotalEntries      int      `json:"total_entries"`
	AssistantMessages int      `json:"assistant_messages"`
	UserMessages      int      `json:"user_messages"`
	ToolCalls         int      `json:"tool_calls"`
	ToolErrors        int      `json:"tool_errors"`
	ToolNames         []string `json:"tool_names"`
	LastAssistantMsg  string   `json:"last_assistant_msg,omitempty"`
}

// Brief returns a one-line summary string.
func (s TranscriptSummary) Brief() string {
	return fmt.Sprintf("%d msgs, %d tool calls (%d errors), tools: [%s]",
		s.AssistantMessages+s.UserMessages, s.ToolCalls, s.ToolErrors, strings.Join(s.ToolNames, ", "))
}

func appendUnique(slice []string, val string) []string {
	if val == "" {
		return slice
	}
	for _, s := range slice {
		if s == val {
			return slice
		}
	}
	return append(slice, val)
}
