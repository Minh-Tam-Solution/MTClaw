package claudecode

import (
	"strings"
	"testing"
)

func TestParseTranscript_Valid(t *testing.T) {
	ndjson := `{"type":"user","content":"Fix the bug"}
{"type":"assistant","content":"I'll fix the bug in main.go"}
{"type":"tool_use","tool_name":"Edit","tool_input":"main.go"}
{"type":"tool_result","tool_name":"Edit"}
{"type":"assistant","content":"Done! The bug is fixed."}
`
	entries, err := ParseTranscript(strings.NewReader(ndjson))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(entries) != 5 {
		t.Errorf("expected 5 entries, got %d", len(entries))
	}
}

func TestParseTranscript_SkipsMalformed(t *testing.T) {
	ndjson := `{"type":"user","content":"hello"}
not valid json
{"type":"assistant","content":"hi"}
also bad {{}
`
	entries, err := ParseTranscript(strings.NewReader(ndjson))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(entries) != 2 {
		t.Errorf("expected 2 valid entries (skipping 2 malformed), got %d", len(entries))
	}
}

func TestParseTranscript_EmptyInput(t *testing.T) {
	entries, err := ParseTranscript(strings.NewReader(""))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

func TestSummarizeTranscript(t *testing.T) {
	entries := []TranscriptEntry{
		{Type: "user", Content: "Fix the bug"},
		{Type: "assistant", Content: "Looking at the code..."},
		{Type: "tool_use", ToolName: "Read"},
		{Type: "tool_result", ToolName: "Read"},
		{Type: "tool_use", ToolName: "Edit"},
		{Type: "tool_result", ToolName: "Edit"},
		{Type: "tool_use", ToolName: "Read"},
		{Type: "tool_result", ToolName: "Read", IsError: true},
		{Type: "assistant", Content: "Done fixing!"},
	}

	summary := SummarizeTranscript(entries)

	if summary.TotalEntries != 9 {
		t.Errorf("total: got %d, want 9", summary.TotalEntries)
	}
	if summary.UserMessages != 1 {
		t.Errorf("user msgs: got %d, want 1", summary.UserMessages)
	}
	if summary.AssistantMessages != 2 {
		t.Errorf("assistant msgs: got %d, want 2", summary.AssistantMessages)
	}
	if summary.ToolCalls != 3 {
		t.Errorf("tool calls: got %d, want 3", summary.ToolCalls)
	}
	if summary.ToolErrors != 1 {
		t.Errorf("tool errors: got %d, want 1", summary.ToolErrors)
	}
	if len(summary.ToolNames) != 2 {
		t.Errorf("unique tools: got %d, want 2 (Read, Edit)", len(summary.ToolNames))
	}
	if summary.LastAssistantMsg != "Done fixing!" {
		t.Errorf("last msg: got %q, want %q", summary.LastAssistantMsg, "Done fixing!")
	}
}

func TestTranscriptSummary_Brief(t *testing.T) {
	s := TranscriptSummary{
		AssistantMessages: 3,
		UserMessages:      2,
		ToolCalls:         5,
		ToolErrors:        1,
		ToolNames:         []string{"Read", "Edit", "Bash"},
	}
	brief := s.Brief()
	if !strings.Contains(brief, "5 msgs") {
		t.Errorf("brief should contain message count: %q", brief)
	}
	if !strings.Contains(brief, "5 tool calls") {
		t.Errorf("brief should contain tool calls: %q", brief)
	}
}
