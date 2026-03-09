package providers

import (
	"testing"
)

// Benchmark the non-subprocess parts of Claude CLI provider.
// These are the fast paths that execute on every fallback request.
// The actual subprocess (exec.Command) is not benchmarked as it requires
// the claude binary (CI_MOCK_EXCEPTION: Claude CLI binary).

func BenchmarkClaudeCLI_BuildPrompt(b *testing.B) {
	messages := []Message{
		{Role: "system", Content: "You are a helpful assistant. Follow the SOUL persona injection below."},
		{Role: "user", Content: "Hello, can you help me with a coding task? I need to implement a provider health tracker with circuit breaker pattern in Go."},
		{Role: "assistant", Content: "I'd be happy to help you implement a provider health tracker with circuit breaker pattern in Go. Here's a basic structure..."},
		{Role: "user", Content: "Can you also add sliding window support for the success rate calculation?"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = buildCLIPrompt(messages)
	}
}

func BenchmarkClaudeCLI_ParseResponse(b *testing.B) {
	// Typical Claude CLI JSON response
	data := []byte(`{
		"type": "result",
		"role": "assistant",
		"model": "claude-sonnet-4-5-20250929",
		"content": [
			{"type": "text", "text": "Here is the implementation of the provider health tracker with sliding window and circuit breaker pattern:\n\n` + "`" + "`" + "`" + `go\ntype ProviderHealthTracker struct {\n\tmu       sync.RWMutex\n\trecords  map[string]*providerHealth\n\twindow   int\n\twindowDur time.Duration\n}\n` + "`" + "`" + "`" + `\n\nThe circuit breaker has three states: closed (normal), open (tripped), and half-open (probe)."}
		],
		"stop_reason": "end_turn",
		"usage": {
			"input_tokens": 150,
			"output_tokens": 200,
			"cache_creation_input_tokens": 0,
			"cache_read_input_tokens": 0
		}
	}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = parseCLIResponse(data)
	}
}

func BenchmarkClaudeCLI_FilterEnv(b *testing.B) {
	// Typical container environment (~30 vars)
	env := []string{
		"HOME=/app",
		"PATH=/usr/local/bin:/usr/bin:/bin",
		"HOSTNAME=mtclaw-container",
		"MTCLAW_POSTGRES_DSN=postgres://user:pass@db:5432/mtclaw",
		"MTCLAW_BFLOW_API_KEY=bflow-key-123",
		"MTCLAW_BFLOW_BASE_URL=https://api.bflow.ai",
		"MTCLAW_TELEGRAM_TOKEN=telegram-token",
		"MTCLAW_ENCRYPTION_KEY=enc-key",
		"BFLOW_TENANT_ID=tenant-1",
		"MTCLAW_BRIDGE_ENABLED=true",
		"MTCLAW_BRIDGE_HOOK_PORT=18792",
		"MTCLAW_CLAUDE_CLI_ENABLED=true",
		"MTCLAW_CLAUDE_CLI_MODEL=sonnet",
		"ANTHROPIC_API_KEY=sk-ant-should-be-stripped",
		"CLAUDE_API_KEY=claude-key-should-be-stripped",
		"LANG=C.UTF-8",
		"TERM=xterm",
		"USER=root",
		"SHELL=/bin/sh",
		"PWD=/app",
		"GOPATH=/go",
		"GOROOT=/usr/local/go",
		"CGO_ENABLED=0",
		"TMPDIR=/tmp",
		"TZ=UTC",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = filterEnv(env)
	}
}
