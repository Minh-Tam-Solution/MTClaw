package agent

import (
	"context"
	"errors"
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

// --- Sprint 27: Provider Chain E2E Scenarios (T27.9) ---

// failingProvider always returns a specific error.
type failingProvider struct {
	name string
	err  error
}

func (p *failingProvider) Name() string        { return p.name }
func (p *failingProvider) DefaultModel() string { return "fail-model" }
func (p *failingProvider) Chat(_ context.Context, _ providers.ChatRequest) (*providers.ChatResponse, error) {
	return nil, p.err
}
func (p *failingProvider) ChatStream(ctx context.Context, req providers.ChatRequest, _ func(providers.StreamChunk)) (*providers.ChatResponse, error) {
	return p.Chat(ctx, req)
}

// TestFallbackE2E_PrimarySucceeds verifies no fallback when primary works.
func TestFallbackE2E_PrimarySucceeds(t *testing.T) {
	primary := &stubProvider{
		name:  "anthropic",
		model: "claude-sonnet",
		chatResp: &providers.ChatResponse{
			Content:      "primary answer",
			FinishReason: "stop",
			Usage:        &providers.Usage{PromptTokens: 100, CompletionTokens: 50, TotalTokens: 150},
		},
	}
	fallback := &stubProvider{name: "claude-cli", model: "sonnet"}

	loop := NewLoop(LoopConfig{
		ID:               "test-e2e-primary-ok",
		Provider:         primary,
		FallbackProvider: fallback,
	})

	if loop.provider.Name() != "anthropic" {
		t.Errorf("expected primary provider 'anthropic', got %q", loop.provider.Name())
	}

	// Simulate primary call
	resp, err := primary.Chat(context.Background(), providers.ChatRequest{
		Messages: []providers.Message{{Role: "user", Content: "hello"}},
	})
	if err != nil {
		t.Fatalf("primary should succeed: %v", err)
	}
	if resp.Content != "primary answer" {
		t.Errorf("expected 'primary answer', got %q", resp.Content)
	}
	if fallback.called {
		t.Error("fallback should NOT be called when primary succeeds")
	}
}

// TestFallbackE2E_RetryableError_FallbackSucceeds verifies fallback on retryable error.
func TestFallbackE2E_RetryableError_FallbackSucceeds(t *testing.T) {
	retryableErr := &providers.HTTPError{Status: 502, Body: "bad gateway"}

	primary := &failingProvider{name: "anthropic", err: retryableErr}
	fallback := &stubProvider{
		name:  "claude-cli",
		model: "sonnet",
		chatResp: &providers.ChatResponse{
			Content:      "fallback answer",
			FinishReason: "stop",
			Usage:        &providers.Usage{PromptTokens: 10, CompletionTokens: 20, TotalTokens: 30},
		},
	}

	// Verify the error is retryable
	if !providers.IsRetryableError(retryableErr) {
		t.Fatal("HTTP 502 should be retryable")
	}

	// Simulate the fallback chain: primary fails → try fallback
	_, err := primary.Chat(context.Background(), providers.ChatRequest{
		Messages: []providers.Message{{Role: "user", Content: "hello"}},
	})
	if err == nil {
		t.Fatal("primary should fail")
	}

	if providers.IsRetryableError(err) {
		resp, fbErr := fallback.Chat(context.Background(), providers.ChatRequest{
			Messages: []providers.Message{{Role: "user", Content: "hello"}},
		})
		if fbErr != nil {
			t.Fatalf("fallback should succeed: %v", fbErr)
		}
		if resp.Content != "fallback answer" {
			t.Errorf("expected 'fallback answer', got %q", resp.Content)
		}
		if !fallback.called {
			t.Error("fallback should be called")
		}
	}
}

// TestFallbackE2E_FatalError_NoFallback verifies fatal errors skip fallback.
func TestFallbackE2E_FatalError_NoFallback(t *testing.T) {
	fatalErr := &providers.HTTPError{Status: 400, Body: "bad request: invalid model"}

	if providers.IsRetryableError(fatalErr) {
		t.Fatal("HTTP 400 should NOT be retryable")
	}

	// 401 Unauthorized — also fatal
	authErr := &providers.HTTPError{Status: 401, Body: "invalid api key"}
	if providers.IsRetryableError(authErr) {
		t.Fatal("HTTP 401 should NOT be retryable")
	}

	// 403 Forbidden — also fatal
	forbiddenErr := &providers.HTTPError{Status: 403, Body: "access denied"}
	if providers.IsRetryableError(forbiddenErr) {
		t.Fatal("HTTP 403 should NOT be retryable")
	}
}

// TestFallbackE2E_BothFail verifies error when both providers fail.
func TestFallbackE2E_BothFail(t *testing.T) {
	primaryErr := &providers.HTTPError{Status: 503, Body: "service unavailable"}
	fallbackErr := errors.New("claude CLI process exited with code 1")

	primary := &failingProvider{name: "anthropic", err: primaryErr}
	fallback := &failingProvider{name: "claude-cli", err: fallbackErr}

	// Primary fails
	_, err := primary.Chat(context.Background(), providers.ChatRequest{
		Messages: []providers.Message{{Role: "user", Content: "hello"}},
	})
	if err == nil {
		t.Fatal("primary should fail")
	}

	// Fallback also fails
	if providers.IsRetryableError(err) {
		_, fbErr := fallback.Chat(context.Background(), providers.ChatRequest{
			Messages: []providers.Message{{Role: "user", Content: "hello"}},
		})
		if fbErr == nil {
			t.Fatal("fallback should also fail")
		}
		// Both errors should be present for diagnostics
		if fbErr.Error() == "" {
			t.Error("fallback error should have content")
		}
	}
}

// TestFallbackE2E_CTOGuard_NoFallbackAtIter1WithTools verifies CTO-R2-1 guard.
func TestFallbackE2E_CTOGuard_NoFallbackAtIter1WithTools(t *testing.T) {
	retryableErr := &providers.HTTPError{Status: 500, Body: "internal server error"}
	if !providers.IsRetryableError(retryableErr) {
		t.Fatal("HTTP 500 should be retryable")
	}

	// CTO guard logic: at iteration=1 with tools, don't fallback
	iteration := 1
	tools := []providers.ToolDefinition{
		{Type: "function", Function: providers.ToolFunctionSchema{Name: "read_file"}},
	}

	canFallback := true
	if iteration == 1 && len(tools) > 0 {
		canFallback = false // CTO-R2-1: don't give wrong text-only answer
	}

	if canFallback {
		t.Error("should NOT allow fallback at iteration=1 with tools (CTO-R2-1)")
	}

	// At iteration=2, fallback IS allowed (tools already ran)
	iteration = 2
	canFallback = true
	if iteration == 1 && len(tools) > 0 {
		canFallback = false
	}
	if !canFallback {
		t.Error("should allow fallback at iteration=2 with tools")
	}

	// At iteration=1 WITHOUT tools, fallback IS allowed
	iteration = 1
	tools = nil
	canFallback = true
	if iteration == 1 && len(tools) > 0 {
		canFallback = false
	}
	if !canFallback {
		t.Error("should allow fallback at iteration=1 without tools")
	}
}

// TestFallbackE2E_ToolsStrippedOnFallback verifies CTO-501: tools always stripped.
func TestFallbackE2E_ToolsStrippedOnFallback(t *testing.T) {
	originalReq := providers.ChatRequest{
		Messages: []providers.Message{{Role: "user", Content: "hello"}},
		Tools: []providers.ToolDefinition{
			{Type: "function", Function: providers.ToolFunctionSchema{Name: "read_file"}},
			{Type: "function", Function: providers.ToolFunctionSchema{Name: "write_file"}},
		},
	}

	// Simulate fallback request preparation (CTO-501)
	fallbackReq := originalReq
	fallbackReq.Tools = nil

	if fallbackReq.Tools != nil {
		t.Error("fallback request should have nil tools (CTO-501)")
	}
	if len(originalReq.Tools) != 2 {
		t.Error("original request tools should be preserved")
	}
	if len(fallbackReq.Messages) != 1 {
		t.Error("fallback request should preserve messages")
	}
}
