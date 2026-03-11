package agent

import (
	"strings"
	"testing"

	"github.com/Minh-Tam-Solution/MTClaw/internal/config"
)

func TestResolveMemoryFlushSettings_Defaults(t *testing.T) {
	settings := ResolveMemoryFlushSettings(nil)
	if settings == nil {
		t.Fatal("expected non-nil settings for nil compaction config")
	}
	if !settings.Enabled {
		t.Error("expected enabled by default")
	}
	if settings.SoftThresholdTokens != DefaultSoftThresholdTokens {
		t.Errorf("expected soft threshold %d, got %d", DefaultSoftThresholdTokens, settings.SoftThresholdTokens)
	}
}

func TestResolveMemoryFlushSettings_Disabled(t *testing.T) {
	disabled := false
	settings := ResolveMemoryFlushSettings(&config.CompactionConfig{
		MemoryFlush: &config.MemoryFlushConfig{Enabled: &disabled},
	})
	if settings != nil {
		t.Error("expected nil settings when disabled")
	}
}

func TestResolveMemoryFlushSettings_CustomPrompt(t *testing.T) {
	custom := "Custom flush prompt"
	settings := ResolveMemoryFlushSettings(&config.CompactionConfig{
		MemoryFlush: &config.MemoryFlushConfig{Prompt: custom},
	})
	if settings == nil {
		t.Fatal("expected non-nil settings")
	}
	if settings.Prompt != custom {
		t.Errorf("expected custom prompt, got %q", settings.Prompt)
	}
}

func TestDefaultMemoryFlushPrompt_StructuredFormat(t *testing.T) {
	// Phase 0 requirement: prompt must request structured output
	requiredPatterns := []string{
		"[decision]",
		"[fact]",
		"[preference]",
		"[lesson]",
		"[entity]",
		"entity",
		"relation",
		"value",
	}
	for _, pattern := range requiredPatterns {
		if !strings.Contains(DefaultMemoryFlushPrompt, pattern) &&
			!strings.Contains(DefaultMemoryFlushSystemPrompt, pattern) {
			t.Errorf("structured flush prompt missing required pattern: %q", pattern)
		}
	}
}

func TestDefaultMemoryFlushSystemPrompt_StructuredRequirement(t *testing.T) {
	if !strings.Contains(DefaultMemoryFlushSystemPrompt, "STRUCTURED OUTPUT REQUIRED") {
		t.Error("system prompt must contain STRUCTURED OUTPUT REQUIRED directive")
	}
}
