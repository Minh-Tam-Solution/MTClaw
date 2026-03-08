package agent

import (
	"context"
	"testing"

	"github.com/Minh-Tam-Solution/MTClaw/internal/providers"
)

// stubProvider is a test double that returns a fixed response or error.
type stubProvider struct {
	name     string
	model    string
	chatResp *providers.ChatResponse
	chatErr  error
	called   bool
}

func (p *stubProvider) Name() string        { return p.name }
func (p *stubProvider) DefaultModel() string { return p.model }
func (p *stubProvider) Chat(_ context.Context, _ providers.ChatRequest) (*providers.ChatResponse, error) {
	p.called = true
	return p.chatResp, p.chatErr
}
func (p *stubProvider) ChatStream(ctx context.Context, req providers.ChatRequest, cb func(providers.StreamChunk)) (*providers.ChatResponse, error) {
	return p.Chat(ctx, req)
}

func TestNewLoop_FallbackProviderWired(t *testing.T) {
	fb := &stubProvider{name: "fallback", model: "fb-model"}
	loop := NewLoop(LoopConfig{
		ID:               "test",
		FallbackProvider: fb,
	})
	if loop.fallbackProvider == nil {
		t.Fatal("expected fallback provider to be wired")
	}
	if loop.fallbackProvider.Name() != "fallback" {
		t.Errorf("expected fallback name 'fallback', got %q", loop.fallbackProvider.Name())
	}
}

func TestNewLoop_NoFallbackByDefault(t *testing.T) {
	loop := NewLoop(LoopConfig{ID: "test"})
	if loop.fallbackProvider != nil {
		t.Error("expected no fallback provider by default")
	}
}

func TestFallbackProvider_IsRetryableError_Triggers(t *testing.T) {
	// Verify that HTTPError 500 is retryable (drives fallback in loop)
	err500 := &providers.HTTPError{Status: 500, Body: "internal server error"}
	if !providers.IsRetryableError(err500) {
		t.Error("expected HTTP 500 to be retryable")
	}

	err429 := &providers.HTTPError{Status: 429, Body: "rate limited"}
	if !providers.IsRetryableError(err429) {
		t.Error("expected HTTP 429 to be retryable")
	}

	err400 := &providers.HTTPError{Status: 400, Body: "bad request"}
	if providers.IsRetryableError(err400) {
		t.Error("expected HTTP 400 to NOT be retryable")
	}
}

func TestFallbackProvider_LoopConfigPreservesBothProviders(t *testing.T) {
	primary := &stubProvider{name: "primary", model: "p-model"}
	fallback := &stubProvider{name: "claude-cli", model: "sonnet"}

	loop := NewLoop(LoopConfig{
		ID:               "test-agent",
		Provider:         primary,
		FallbackProvider: fallback,
	})

	if loop.provider == nil {
		t.Fatal("expected primary provider")
	}
	if loop.provider.Name() != "primary" {
		t.Errorf("expected primary name 'primary', got %q", loop.provider.Name())
	}
	if loop.fallbackProvider == nil {
		t.Fatal("expected fallback provider")
	}
	if loop.fallbackProvider.Name() != "claude-cli" {
		t.Errorf("expected fallback name 'claude-cli', got %q", loop.fallbackProvider.Name())
	}
}

func TestFallbackProvider_StubChatResponse(t *testing.T) {
	// Verify fallback provider can produce a valid ChatResponse
	fb := &stubProvider{
		name:  "claude-cli",
		model: "sonnet",
		chatResp: &providers.ChatResponse{
			Content:      "fallback answer",
			FinishReason: "stop",
			Usage:        &providers.Usage{PromptTokens: 10, CompletionTokens: 20, TotalTokens: 30},
		},
	}
	resp, err := fb.Chat(context.Background(), providers.ChatRequest{
		Messages: []providers.Message{{Role: "user", Content: "test"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Content != "fallback answer" {
		t.Errorf("expected 'fallback answer', got %q", resp.Content)
	}
	if !fb.called {
		t.Error("expected called=true")
	}
}
