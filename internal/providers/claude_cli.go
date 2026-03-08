package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"time"
)

// ClaudeCLIProvider wraps the Claude CLI binary as an LLM provider.
// Uses Claude Max subscription (OAuth) — NOT API key billing.
// CLI invocation: claude -p --output-format json --model <model> --max-turns 1 "<prompt>"
type ClaudeCLIProvider struct {
	cliPath string        // absolute path to claude binary
	model   string        // e.g. "sonnet", "opus", "haiku"
	timeout time.Duration // subprocess timeout (default 120s)
}

// ClaudeCLIConfig holds configuration for the Claude CLI provider.
type ClaudeCLIConfig struct {
	Path    string        // path to claude binary
	Model   string        // model name (default "sonnet")
	Timeout time.Duration // subprocess timeout
	Enabled bool          // whether provider is enabled
}

// NewClaudeCLIProvider creates a new Claude CLI provider.
func NewClaudeCLIProvider(cfg ClaudeCLIConfig) *ClaudeCLIProvider {
	path := cfg.Path
	if path == "" {
		path = "claude" // rely on PATH
	}
	model := cfg.Model
	if model == "" {
		model = "sonnet"
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	return &ClaudeCLIProvider{
		cliPath: path,
		model:   model,
		timeout: timeout,
	}
}

// claudeCLIResponse is the JSON output from `claude -p --output-format json`.
type claudeCLIResponse struct {
	Type         string `json:"type"`
	Role         string `json:"role"`
	Model        string `json:"model"`
	Content      []claudeCLIContentBlock `json:"content"`
	StopReason   string `json:"stop_reason"`
	Usage        *claudeCLIUsage `json:"usage,omitempty"`
}

type claudeCLIContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type claudeCLIUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	CacheCreationInputTokens int `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     int `json:"cache_read_input_tokens,omitempty"`
}

func (p *ClaudeCLIProvider) Name() string { return "claude-cli" }

func (p *ClaudeCLIProvider) DefaultModel() string { return p.model }

// Chat sends messages to Claude CLI and returns the response.
// Combines all messages into a single prompt string for the CLI.
func (p *ClaudeCLIProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	prompt := buildCLIPrompt(req.Messages)
	if prompt == "" {
		return nil, fmt.Errorf("claude-cli: empty prompt")
	}

	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	args := []string{"-p", "--output-format", "json", "--model", p.model, "--max-turns", "1"}

	cmd := exec.CommandContext(ctx, p.cliPath, args...)
	cmd.Env = filterEnv(os.Environ())
	cmd.Stdin = strings.NewReader(prompt)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	slog.Debug("claude-cli: executing", "path", p.cliPath, "model", p.model, "prompt_len", len(prompt))

	if err := cmd.Run(); err != nil {
		stderrStr := strings.TrimSpace(stderr.String())
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("claude-cli: timeout after %s: %s", p.timeout, stderrStr)
		}
		return nil, fmt.Errorf("claude-cli: process error: %w (stderr: %s)", err, stderrStr)
	}

	return parseCLIResponse(stdout.Bytes())
}

// ChatStream delegates to Chat and emits a single chunk (CTO-500/502).
// Claude CLI doesn't support native streaming — this satisfies the Provider interface.
func (p *ClaudeCLIProvider) ChatStream(ctx context.Context, req ChatRequest, onChunk func(StreamChunk)) (*ChatResponse, error) {
	resp, err := p.Chat(ctx, req)
	if err != nil {
		return nil, err
	}

	if onChunk != nil {
		if resp.Content != "" {
			onChunk(StreamChunk{Content: resp.Content})
		}
		onChunk(StreamChunk{Done: true})
	}

	return resp, nil
}

// buildCLIPrompt converts message history into a single prompt string for the CLI.
func buildCLIPrompt(messages []Message) string {
	var sb strings.Builder
	for _, msg := range messages {
		switch msg.Role {
		case "system":
			sb.WriteString("[System]\n")
			sb.WriteString(msg.Content)
			sb.WriteString("\n\n")
		case "user":
			sb.WriteString(msg.Content)
			sb.WriteString("\n\n")
		case "assistant":
			sb.WriteString("[Assistant]\n")
			sb.WriteString(msg.Content)
			sb.WriteString("\n\n")
		case "tool":
			sb.WriteString("[Tool Result]\n")
			sb.WriteString(msg.Content)
			sb.WriteString("\n\n")
		}
	}
	return strings.TrimSpace(sb.String())
}

// parseCLIResponse parses the JSON output from claude CLI into a ChatResponse.
func parseCLIResponse(data []byte) (*ChatResponse, error) {
	var cliResp claudeCLIResponse
	if err := json.Unmarshal(data, &cliResp); err != nil {
		// If JSON parsing fails, treat raw output as text response
		text := strings.TrimSpace(string(data))
		if text == "" {
			return nil, fmt.Errorf("claude-cli: empty response")
		}
		return &ChatResponse{
			Content:      text,
			FinishReason: "stop",
		}, nil
	}

	// Extract text content from content blocks
	var content strings.Builder
	for _, block := range cliResp.Content {
		if block.Type == "text" {
			if content.Len() > 0 {
				content.WriteString("\n")
			}
			content.WriteString(block.Text)
		}
	}

	resp := &ChatResponse{
		Content:      content.String(),
		FinishReason: "stop",
	}

	if cliResp.StopReason == "max_tokens" {
		resp.FinishReason = "length"
	}

	if cliResp.Usage != nil {
		resp.Usage = &Usage{
			PromptTokens:     cliResp.Usage.InputTokens,
			CompletionTokens: cliResp.Usage.OutputTokens,
			TotalTokens:      cliResp.Usage.InputTokens + cliResp.Usage.OutputTokens,
			CacheCreationTokens: cliResp.Usage.CacheCreationInputTokens,
			CacheReadTokens:     cliResp.Usage.CacheReadInputTokens,
		}
	}

	return resp, nil
}

// filterEnv strips sensitive env vars from the subprocess environment.
// Forces OAuth billing (Claude Max subscription) instead of API key billing (CTO-R2-2).
func filterEnv(env []string) []string {
	filtered := make([]string, 0, len(env))
	for _, e := range env {
		key := e
		if idx := strings.IndexByte(e, '='); idx >= 0 {
			key = e[:idx]
		}
		switch key {
		case "ANTHROPIC_API_KEY", "CLAUDE_API_KEY":
			continue // strip to force OAuth
		default:
			filtered = append(filtered, e)
		}
	}
	return filtered
}
