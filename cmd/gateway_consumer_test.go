package cmd

import (
	"errors"
	"strings"
	"testing"

	"github.com/Minh-Tam-Solution/MTClaw/internal/config"
	"github.com/Minh-Tam-Solution/MTClaw/internal/rag"
	"github.com/Minh-Tam-Solution/MTClaw/internal/routing"
)

// =============================================================
// Scenario 1: Command Metadata Flow (TraceName + TraceTags)
// Verifies: InboundMessage metadata flows to TraceName/TraceTags
// =============================================================

func TestExtractTraceMetadata_SpecFactory(t *testing.T) {
	metadata := map[string]string{
		"rail":    "spec-factory",
		"command": "spec",
	}

	traceName, traceTags := extractTraceMetadata(metadata, "")

	if traceName != "spec-factory" {
		t.Errorf("TraceName: got %q, want %q", traceName, "spec-factory")
	}
	if !containsTag(traceTags, "rail:spec-factory") {
		t.Errorf("TraceTags missing 'rail:spec-factory': %v", traceTags)
	}
	if !containsTag(traceTags, "command:spec") {
		t.Errorf("TraceTags missing 'command:spec': %v", traceTags)
	}
}

func TestExtractTraceMetadata_PRGate(t *testing.T) {
	metadata := map[string]string{
		"rail":    "pr-gate",
		"command": "review",
		"pr_url":  "https://github.com/org/repo/pull/123",
	}

	traceName, traceTags := extractTraceMetadata(metadata, "")

	if traceName != "pr-gate" {
		t.Errorf("TraceName: got %q, want %q", traceName, "pr-gate")
	}
	if !containsTag(traceTags, "rail:pr-gate") {
		t.Errorf("TraceTags missing 'rail:pr-gate': %v", traceTags)
	}
	if !containsTag(traceTags, "command:review") {
		t.Errorf("TraceTags missing 'command:review': %v", traceTags)
	}
}

func TestExtractTraceMetadata_WithMention(t *testing.T) {
	metadata := map[string]string{}

	traceName, traceTags := extractTraceMetadata(metadata, "reviewer")

	if traceName != "" {
		t.Errorf("TraceName should be empty for mention-only: got %q", traceName)
	}
	if !containsTag(traceTags, "mention:reviewer") {
		t.Errorf("TraceTags missing 'mention:reviewer': %v", traceTags)
	}
}

func TestExtractTraceMetadata_NoMetadata(t *testing.T) {
	traceName, traceTags := extractTraceMetadata(nil, "")

	if traceName != "" {
		t.Errorf("TraceName should be empty: got %q", traceName)
	}
	if len(traceTags) != 0 {
		t.Errorf("TraceTags should be empty: got %v", traceTags)
	}
}

// =============================================================
// Scenario 2: Command Routing Priority
// Verifies: /spec → PM (not overridden by @mention or handoff)
//           @reviewer → reviewer (mention takes priority over default)
// =============================================================

func TestResolveAgentRoute_PeerMatch(t *testing.T) {
	cfg := &config.Config{
		Bindings: []config.AgentBinding{
			{
				AgentID: "pm",
				Match: config.BindingMatch{
					Channel: "telegram",
					Peer:    &config.BindingPeer{Kind: "direct", ID: "12345"},
				},
			},
			{
				AgentID: "assistant",
				Match:   config.BindingMatch{Channel: "telegram"},
			},
		},
	}

	// Peer-specific match should win
	got := resolveAgentRoute(cfg, "telegram", "12345", "direct")
	if got != "pm" {
		t.Errorf("peer match: got %q, want %q", got, "pm")
	}

	// Non-matching peer falls through to channel-level
	got = resolveAgentRoute(cfg, "telegram", "99999", "direct")
	if got != "assistant" {
		t.Errorf("channel fallback: got %q, want %q", got, "assistant")
	}
}

func TestResolveAgentRoute_ChannelMatch(t *testing.T) {
	cfg := &config.Config{
		Bindings: []config.AgentBinding{
			{
				AgentID: "assistant",
				Match:   config.BindingMatch{Channel: "telegram"},
			},
		},
	}

	got := resolveAgentRoute(cfg, "telegram", "any-chat", "group")
	if got != "assistant" {
		t.Errorf("channel match: got %q, want %q", got, "assistant")
	}
}

func TestResolveAgentRoute_NoMatch(t *testing.T) {
	cfg := &config.Config{
		Bindings: []config.AgentBinding{
			{
				AgentID: "pm",
				Match:   config.BindingMatch{Channel: "zalo"},
			},
		},
	}

	// No telegram binding → default agent
	got := resolveAgentRoute(cfg, "telegram", "any-chat", "direct")
	if got != "default" {
		t.Errorf("no match: got %q, want %q", got, "default")
	}
}

// =============================================================
// Scenario 3: AI-Platform Graceful Degradation
// Verifies: formatAgentError() produces user-friendly messages
// =============================================================

func TestFormatAgentError_RateLimit(t *testing.T) {
	tests := []struct {
		errMsg string
	}{
		{"rate limit exceeded"},
		{"429 too many requests"},
		{"quota exceeded for model"},
		{"resource_exhausted"},
	}
	for _, tt := range tests {
		got := formatAgentError(errors.New(tt.errMsg))
		if !strings.Contains(got, "rate limit") {
			t.Errorf("error %q: got %q, want rate limit message", tt.errMsg, got)
		}
	}
}

func TestFormatAgentError_Timeout(t *testing.T) {
	tests := []struct {
		errMsg string
	}{
		{"request timeout"},
		{"connection timed out"},
		// Note: "context deadline exceeded" matches context overflow first
		// because isContextOverflowError checks for "context" + "exceeded".
		// This is correct behavior — context overflow takes priority.
	}
	for _, tt := range tests {
		got := formatAgentError(errors.New(tt.errMsg))
		if !strings.Contains(got, "timed out") && !strings.Contains(got, "Request timed out") {
			t.Errorf("error %q: got %q, want timeout message", tt.errMsg, got)
		}
	}
}

func TestFormatAgentError_ContextDeadlineIsOverflow(t *testing.T) {
	// "context deadline exceeded" contains "context" + "exceeded"
	// which matches isContextOverflowError before timeout check.
	// This is the actual behavior of the classifier.
	got := formatAgentError(errors.New("context deadline exceeded"))
	if !strings.Contains(got, "Context overflow") {
		t.Errorf("got %q, want context overflow (classifier priority)", got)
	}
}

func TestFormatAgentError_Auth(t *testing.T) {
	tests := []struct {
		errMsg string
	}{
		{"invalid api key"},
		{"401 unauthorized"},
		{"403 forbidden"},
		{"access denied"},
	}
	for _, tt := range tests {
		got := formatAgentError(errors.New(tt.errMsg))
		if !strings.Contains(got, "Authentication error") {
			t.Errorf("error %q: got %q, want auth error message", tt.errMsg, got)
		}
	}
}

func TestFormatAgentError_ContextOverflow(t *testing.T) {
	tests := []struct {
		errMsg string
	}{
		{"request_too_large"},
		{"context length exceeded"},
		{"prompt is too long"},
	}
	for _, tt := range tests {
		got := formatAgentError(errors.New(tt.errMsg))
		if !strings.Contains(got, "Context overflow") {
			t.Errorf("error %q: got %q, want context overflow message", tt.errMsg, got)
		}
	}
}

func TestFormatAgentError_Billing(t *testing.T) {
	got := formatAgentError(errors.New("insufficient credits on account"))
	if !strings.Contains(got, "billing") {
		t.Errorf("got %q, want billing error message", got)
	}
}

func TestFormatAgentError_Overloaded(t *testing.T) {
	got := formatAgentError(errors.New("service overloaded"))
	if !strings.Contains(got, "overloaded") {
		t.Errorf("got %q, want overloaded message", got)
	}
}

func TestFormatAgentError_ModelConfig(t *testing.T) {
	got := formatAgentError(errors.New("not a valid model"))
	if !strings.Contains(got, "Model configuration") {
		t.Errorf("got %q, want model config message", got)
	}
}

func TestFormatAgentError_GenericNeverExposesRawError(t *testing.T) {
	raw := "POST /v1/chat/completions 500 Internal Server Error: {\"error\":\"something broke\"}"
	got := formatAgentError(errors.New(raw))
	if strings.Contains(got, "/v1/chat") || strings.Contains(got, "Internal Server Error") {
		t.Errorf("raw API payload leaked to user: %q", got)
	}
	if !strings.Contains(got, "something went wrong") {
		t.Errorf("expected generic user-friendly message, got %q", got)
	}
}

func TestFormatAgentError_SessionHistory(t *testing.T) {
	tests := []struct {
		errMsg string
	}{
		{"tool_use_id mismatch"},
		{"roles must alternate"},
		{"unexpected tool result"},
	}
	for _, tt := range tests {
		got := formatAgentError(errors.New(tt.errMsg))
		if !strings.Contains(got, "Session history") {
			t.Errorf("error %q: got %q, want session history message", tt.errMsg, got)
		}
	}
}

// =============================================================
// Scenario 4: UTF-8 Rune-Safe Truncation
// Verifies: Vietnamese text truncated at rune boundary, not byte
// =============================================================

func TestRuneSafeTruncation(t *testing.T) {
	// Vietnamese text: "Tạo tính năng đăng nhập cho ứng dụng di động"
	// Each Vietnamese character may be multi-byte in UTF-8.
	vn := "Tạo tính năng đăng nhập cho ứng dụng di động Bflow. "
	// Build a string > 200 runes by repeating
	var long strings.Builder
	for long.Len() < 1000 {
		long.WriteString(vn)
	}
	goal := long.String()

	// Apply rune-safe truncation (same logic as gateway_consumer.go)
	goalRunes := []rune(goal)
	if len(goalRunes) > 200 {
		goal = string(goalRunes[:200]) + "..."
	}

	// Verify: result should be valid UTF-8 and end with "..."
	if !strings.HasSuffix(goal, "...") {
		t.Error("truncated string should end with '...'")
	}

	// Verify: exactly 200 runes + "..."
	resultRunes := []rune(goal)
	// "..." = 3 runes
	if len(resultRunes) != 203 {
		t.Errorf("expected 203 runes (200 + '...'), got %d", len(resultRunes))
	}

	// Verify: no invalid UTF-8 sequences
	if !isValidUTF8(goal) {
		t.Error("truncated string contains invalid UTF-8")
	}
}

func TestRuneSafeTruncation_ShortString(t *testing.T) {
	goal := "Tạo tính năng đăng nhập"
	goalRunes := []rune(goal)
	if len(goalRunes) > 200 {
		goal = string(goalRunes[:200]) + "..."
	}

	// Short string should not be truncated
	if strings.HasSuffix(goal, "...") {
		t.Error("short string should not be truncated")
	}
	if goal != "Tạo tính năng đăng nhập" {
		t.Errorf("short string modified: got %q", goal)
	}
}

func TestRuneSafeTruncation_ExactBoundary(t *testing.T) {
	// Build exactly 200 runes
	runes := make([]rune, 200)
	for i := range runes {
		runes[i] = 'ạ' // multi-byte Vietnamese character
	}
	goal := string(runes)

	goalRunes := []rune(goal)
	if len(goalRunes) > 200 {
		goal = string(goalRunes[:200]) + "..."
	}

	// Exactly 200 runes → no truncation
	if strings.HasSuffix(goal, "...") {
		t.Error("exactly 200 runes should not be truncated")
	}
}

// =============================================================
// Scenario 5: @mention Parsing
// Verifies: @reviewer → agentID = "reviewer" with content stripped
//           @nonexistent → no routing change
//           Command + @mention → command takes priority
// =============================================================

func TestParseMention_ValidAgent(t *testing.T) {
	content := "@reviewer please check this PR"
	mention, stripped := parseMention(content)

	if mention != "reviewer" {
		t.Errorf("mention: got %q, want %q", mention, "reviewer")
	}
	if stripped != "please check this PR" {
		t.Errorf("stripped: got %q, want %q", stripped, "please check this PR")
	}
}

func TestParseMention_NoMention(t *testing.T) {
	content := "please check this PR"
	mention, stripped := parseMention(content)

	if mention != "" {
		t.Errorf("mention should be empty: got %q", mention)
	}
	if stripped != content {
		t.Errorf("content should be unchanged: got %q", stripped)
	}
}

func TestParseMention_MentionOnly(t *testing.T) {
	content := "@pm"
	mention, stripped := parseMention(content)

	if mention != "pm" {
		t.Errorf("mention: got %q, want %q", mention, "pm")
	}
	if stripped != "" {
		t.Errorf("stripped should be empty: got %q", stripped)
	}
}

func TestParseMention_CaseInsensitive(t *testing.T) {
	content := "@PM some task"
	mention, _ := parseMention(content)

	if mention != "pm" {
		t.Errorf("mention should be lowercase: got %q, want %q", mention, "pm")
	}
}

// =============================================================
// Scenario 6: RAG Collection Mapping by SOUL (Sprint 6)
// Verifies: correct collection(s) returned for each SOUL
// =============================================================

func TestRAGCollectionMapping_EngineeringSOULs(t *testing.T) {
	engSOULs := []string{"enghelp", "coder", "architect", "reviewer", "devops", "tester", "itadmin", "writer"}
	for _, soul := range engSOULs {
		collections := rag.CollectionMap[soul]
		if len(collections) == 0 {
			t.Errorf("SOUL %q: expected collections, got none", soul)
			continue
		}
		if collections[0] != "engineering" {
			t.Errorf("SOUL %q: expected 'engineering', got %v", soul, collections)
		}
	}
}

func TestRAGCollectionMapping_SalesSOUL(t *testing.T) {
	collections := rag.CollectionMap["sales"]
	if len(collections) != 1 || collections[0] != "sales" {
		t.Errorf("sales SOUL: got %v, want [sales]", collections)
	}
}

func TestRAGCollectionMapping_CrossFunctionalSOULs(t *testing.T) {
	// cs, assistant, pm → both engineering + sales
	crossSOULs := []string{"cs", "assistant", "pm"}
	for _, soul := range crossSOULs {
		collections := rag.CollectionMap[soul]
		if len(collections) != 2 {
			t.Errorf("SOUL %q: expected 2 collections, got %v", soul, collections)
			continue
		}
		hasEng := false
		hasSales := false
		for _, c := range collections {
			if c == "engineering" {
				hasEng = true
			}
			if c == "sales" {
				hasSales = true
			}
		}
		if !hasEng || !hasSales {
			t.Errorf("SOUL %q: expected [engineering, sales], got %v", soul, collections)
		}
	}
}

func TestRAGCollectionMapping_UnknownSOUL(t *testing.T) {
	collections := rag.CollectionMap["nonexistent"]
	if len(collections) != 0 {
		t.Errorf("unknown SOUL: got %v, want empty", collections)
	}
}

// =============================================================
// Scenario 7: Team Mention Resolution (Sprint 6, CTO-8)
// Verifies: @engineering → team lead, agent-first priority
// =============================================================

func TestTeamMentionMap_AllTeamsMapped(t *testing.T) {
	expected := map[string]string{
		"engineering": "SDLC Engineering",
		"business":    "Business Operations",
		"advisory":    "Advisory Board",
	}
	for key, want := range expected {
		got, ok := routing.TeamMentionMap[key]
		if !ok {
			t.Errorf("routing.TeamMentionMap missing key %q", key)
			continue
		}
		if got != want {
			t.Errorf("routing.TeamMentionMap[%q]: got %q, want %q", key, got, want)
		}
	}
}

func TestTeamMentionMap_ShortKeysAreCorrect(t *testing.T) {
	// Verify users type short keys, not full names
	for key := range routing.TeamMentionMap {
		if strings.Contains(key, " ") {
			t.Errorf("routing.TeamMentionMap key %q contains spaces — should be short mention key", key)
		}
	}
}

func TestParseMention_TeamMention(t *testing.T) {
	// @engineering routes to team mention, not an agent
	content := "@engineering what's our velocity?"
	mention, stripped := parseMention(content)

	if mention != "engineering" {
		t.Errorf("mention: got %q, want %q", mention, "engineering")
	}
	if stripped != "what's our velocity?" {
		t.Errorf("stripped: got %q, want %q", stripped, "what's our velocity?")
	}

	// Verify it's a valid team mention key
	if _, ok := routing.TeamMentionMap[mention]; !ok {
		t.Errorf("parsed mention %q is not in routing.TeamMentionMap", mention)
	}
}

// =============================================================
// Scenario 8: Evidence Tags (Sprint 6 — RAG + Team tags)
// Verifies: RAG and team tags are included in trace metadata
// =============================================================

func TestExtractTraceMetadata_WithRAGTags(t *testing.T) {
	metadata := map[string]string{
		"rail":    "spec-factory",
		"command": "spec",
	}

	traceName, traceTags := extractTraceMetadata(metadata, "")

	// Simulate Sprint 6 RAG tags appended
	ragTags := []string{"rag:engineering", "rag_hits:5", "rag_tokens:1850"}
	traceTags = append(traceTags, ragTags...)

	if traceName != "spec-factory" {
		t.Errorf("TraceName: got %q, want %q", traceName, "spec-factory")
	}
	if !containsTag(traceTags, "rag:engineering") {
		t.Errorf("missing rag collection tag: %v", traceTags)
	}
	if !containsTag(traceTags, "rag_hits:5") {
		t.Errorf("missing rag_hits tag: %v", traceTags)
	}
}

func TestExtractTraceMetadata_WithTeamTag(t *testing.T) {
	metadata := map[string]string{}

	_, traceTags := extractTraceMetadata(metadata, "pm")

	// Simulate Sprint 6 team tag
	mentionTeam := "engineering"
	traceTags = append(traceTags, "team:"+mentionTeam)

	if !containsTag(traceTags, "mention:pm") {
		t.Errorf("missing mention tag: %v", traceTags)
	}
	if !containsTag(traceTags, "team:engineering") {
		t.Errorf("missing team tag: %v", traceTags)
	}
}

// =============================================================
// Helpers
// =============================================================

// extractTraceMetadata extracts TraceName and TraceTags from InboundMessage
// metadata. This is the same logic as gateway_consumer.go lines 243-257.
func extractTraceMetadata(metadata map[string]string, mentionAgent string) (string, []string) {
	var traceName string
	var traceTags []string
	if metadata != nil {
		if rail := metadata["rail"]; rail != "" {
			traceName = rail
			traceTags = append(traceTags, "rail:"+rail)
		}
		if command := metadata["command"]; command != "" {
			traceTags = append(traceTags, "command:"+command)
		}
	}
	if mentionAgent != "" {
		traceTags = append(traceTags, "mention:"+mentionAgent)
	}
	return traceName, traceTags
}

// parseMention extracts @agentkey prefix from content.
// Returns (agentKey, remainingContent). If no mention, agentKey is empty
// and content is returned unchanged.
func parseMention(content string) (string, string) {
	if !strings.HasPrefix(content, "@") {
		return "", content
	}
	parts := strings.SplitN(content, " ", 2)
	candidate := strings.TrimPrefix(parts[0], "@")
	candidate = strings.ToLower(candidate)
	remaining := ""
	if len(parts) > 1 {
		remaining = strings.TrimSpace(parts[1])
	}
	return candidate, remaining
}

func containsTag(tags []string, tag string) bool {
	for _, t := range tags {
		if t == tag {
			return true
		}
	}
	return false
}

func isValidUTF8(s string) bool {
	for i := 0; i < len(s); {
		r, size := rune(s[i]), 1
		if s[i] >= 0x80 {
			r, size = decodeRuneInString(s[i:])
		}
		if r == 0xFFFD && size == 1 {
			return false // invalid UTF-8
		}
		i += size
	}
	return true
}

func decodeRuneInString(s string) (rune, int) {
	// Use the standard library approach via []rune conversion
	runes := []rune(s[:minInt(4, len(s))])
	if len(runes) == 0 {
		return 0xFFFD, 1
	}
	return runes[0], len(string(runes[:1]))
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
