package providers

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestClaudeCLIProvider_Name(t *testing.T) {
	p := NewClaudeCLIProvider(ClaudeCLIConfig{})
	if p.Name() != "claude-cli" {
		t.Errorf("expected name claude-cli, got %s", p.Name())
	}
}

func TestClaudeCLIProvider_DefaultModel(t *testing.T) {
	p := NewClaudeCLIProvider(ClaudeCLIConfig{Model: "opus"})
	if p.DefaultModel() != "opus" {
		t.Errorf("expected model opus, got %s", p.DefaultModel())
	}
}

func TestClaudeCLIProvider_DefaultModelFallback(t *testing.T) {
	p := NewClaudeCLIProvider(ClaudeCLIConfig{})
	if p.DefaultModel() != "sonnet" {
		t.Errorf("expected default model sonnet, got %s", p.DefaultModel())
	}
}

func TestClaudeCLIProvider_DefaultTimeout(t *testing.T) {
	p := NewClaudeCLIProvider(ClaudeCLIConfig{})
	if p.timeout != 120*time.Second {
		t.Errorf("expected default timeout 120s, got %s", p.timeout)
	}
}

func TestClaudeCLIProvider_CustomTimeout(t *testing.T) {
	p := NewClaudeCLIProvider(ClaudeCLIConfig{Timeout: 60 * time.Second})
	if p.timeout != 60*time.Second {
		t.Errorf("expected timeout 60s, got %s", p.timeout)
	}
}

func TestBuildCLIPrompt_Simple(t *testing.T) {
	msgs := []Message{
		{Role: "user", Content: "Hello"},
	}
	prompt := buildCLIPrompt(msgs)
	if prompt != "Hello" {
		t.Errorf("expected 'Hello', got %q", prompt)
	}
}

func TestBuildCLIPrompt_WithSystem(t *testing.T) {
	msgs := []Message{
		{Role: "system", Content: "You are helpful."},
		{Role: "user", Content: "Hello"},
	}
	prompt := buildCLIPrompt(msgs)
	if prompt == "" {
		t.Error("expected non-empty prompt")
	}
	if !strings.Contains(prompt, "[System]") || !strings.Contains(prompt, "You are helpful.") || !strings.Contains(prompt, "Hello") {
		t.Errorf("prompt missing expected content: %q", prompt)
	}
}

func TestBuildCLIPrompt_Empty(t *testing.T) {
	prompt := buildCLIPrompt(nil)
	if prompt != "" {
		t.Errorf("expected empty prompt, got %q", prompt)
	}
}

func TestParseCLIResponse_Valid(t *testing.T) {
	data := []byte(`{
		"type": "result",
		"role": "assistant",
		"model": "claude-sonnet-4-5-20250929",
		"content": [{"type": "text", "text": "Hello world"}],
		"stop_reason": "end_turn",
		"usage": {"input_tokens": 10, "output_tokens": 5}
	}`)
	resp, err := parseCLIResponse(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "Hello world" {
		t.Errorf("expected 'Hello world', got %q", resp.Content)
	}
	if resp.FinishReason != "stop" {
		t.Errorf("expected finish_reason 'stop', got %q", resp.FinishReason)
	}
	if resp.Usage == nil {
		t.Fatal("expected usage, got nil")
	}
	if resp.Usage.PromptTokens != 10 || resp.Usage.CompletionTokens != 5 {
		t.Errorf("unexpected usage: %+v", resp.Usage)
	}
}

func TestParseCLIResponse_MaxTokens(t *testing.T) {
	data := []byte(`{
		"type": "result",
		"content": [{"type": "text", "text": "truncated"}],
		"stop_reason": "max_tokens"
	}`)
	resp, err := parseCLIResponse(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.FinishReason != "length" {
		t.Errorf("expected finish_reason 'length', got %q", resp.FinishReason)
	}
}

func TestParseCLIResponse_RawText(t *testing.T) {
	// If CLI returns non-JSON text, treat as plain text response
	data := []byte("Just a plain text response")
	resp, err := parseCLIResponse(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "Just a plain text response" {
		t.Errorf("expected plain text, got %q", resp.Content)
	}
}

func TestParseCLIResponse_Empty(t *testing.T) {
	_, err := parseCLIResponse([]byte(""))
	if err == nil {
		t.Error("expected error for empty response")
	}
}

func TestFilterEnv_StripsAPIKeys(t *testing.T) {
	env := []string{
		"HOME=/home/user",
		"ANTHROPIC_API_KEY=sk-ant-secret",
		"CLAUDE_API_KEY=sk-another-secret",
		"PATH=/usr/bin",
	}
	filtered := filterEnv(env)
	for _, e := range filtered {
		if strings.Contains(e, "ANTHROPIC_API_KEY") || strings.Contains(e, "CLAUDE_API_KEY") {
			t.Errorf("expected API key to be stripped, found: %s", e)
		}
	}
	if len(filtered) != 2 {
		t.Errorf("expected 2 env vars, got %d", len(filtered))
	}
}

func TestFilterEnv_PreservesOther(t *testing.T) {
	env := []string{
		"HOME=/home/user",
		"MTCLAW_PROVIDER=bflow-ai-platform",
	}
	filtered := filterEnv(env)
	if len(filtered) != 2 {
		t.Errorf("expected 2 env vars preserved, got %d", len(filtered))
	}
}

func TestClaudeCLIProvider_ChatEmptyPrompt(t *testing.T) {
	p := NewClaudeCLIProvider(ClaudeCLIConfig{})
	_, err := p.Chat(context.Background(), ChatRequest{Messages: nil})
	if err == nil {
		t.Error("expected error for empty prompt")
	}
}

func TestClaudeCLIProvider_ChatStreamDelegatesToChat(t *testing.T) {
	// Use a non-existent binary to verify ChatStream delegates to Chat
	p := NewClaudeCLIProvider(ClaudeCLIConfig{Path: "/nonexistent/claude"})
	_, err := p.ChatStream(context.Background(), ChatRequest{
		Messages: []Message{{Role: "user", Content: "hi"}},
	}, func(chunk StreamChunk) {})
	if err == nil {
		t.Error("expected error from non-existent binary")
	}
}

func TestParseCLIResponse_MultipleContentBlocks(t *testing.T) {
	data := []byte(`{
		"type": "result",
		"content": [
			{"type": "text", "text": "First part."},
			{"type": "text", "text": "Second part."}
		],
		"stop_reason": "end_turn"
	}`)
	resp, err := parseCLIResponse(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "First part.\nSecond part." {
		t.Errorf("expected joined content, got %q", resp.Content)
	}
}

