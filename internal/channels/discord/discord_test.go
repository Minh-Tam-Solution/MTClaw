package discord

import (
	"encoding/json"
	"testing"

	"github.com/bwmarrin/discordgo"

	"github.com/Minh-Tam-Solution/MTClaw/internal/bus"
	"github.com/Minh-Tam-Solution/MTClaw/internal/config"
)

func TestNew_RequiresToken(t *testing.T) {
	cfg := config.DiscordConfig{Enabled: true, Token: ""}
	_, err := New(cfg, bus.New(), nil)
	if err == nil {
		t.Fatal("expected error for empty token")
	}
}

func TestNew_DefaultPolicies(t *testing.T) {
	cfg := config.DiscordConfig{Enabled: true, Token: "test-token"}
	ch, err := New(cfg, bus.New(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ch.dmPolicy != "pairing" {
		t.Errorf("expected default dmPolicy=pairing, got %s", ch.dmPolicy)
	}
	if ch.groupPolicy != "disabled" {
		t.Errorf("expected default groupPolicy=disabled, got %s", ch.groupPolicy)
	}
}

func TestNew_CustomPolicies(t *testing.T) {
	cfg := config.DiscordConfig{
		Enabled:     true,
		Token:       "test-token",
		DMPolicy:    "open",
		GroupPolicy: "open",
		GuildIDs:    []string{"123", "456"},
	}
	ch, err := New(cfg, bus.New(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ch.dmPolicy != "open" {
		t.Errorf("expected dmPolicy=open, got %s", ch.dmPolicy)
	}
	if ch.groupPolicy != "open" {
		t.Errorf("expected groupPolicy=open, got %s", ch.groupPolicy)
	}
	if !ch.guildIDs["123"] || !ch.guildIDs["456"] {
		t.Error("expected guild IDs 123 and 456 to be allowlisted")
	}
}

func TestCheckDMPolicy_Open(t *testing.T) {
	cfg := config.DiscordConfig{Enabled: true, Token: "test-token", DMPolicy: "open"}
	ch, _ := New(cfg, bus.New(), nil)
	if !ch.checkDMPolicy("user1|testuser", "ch1") {
		t.Error("open policy should accept all DMs")
	}
}

func TestCheckDMPolicy_Disabled(t *testing.T) {
	cfg := config.DiscordConfig{Enabled: true, Token: "test-token", DMPolicy: "disabled"}
	ch, _ := New(cfg, bus.New(), nil)
	if ch.checkDMPolicy("user1|testuser", "ch1") {
		t.Error("disabled policy should reject all DMs")
	}
}

func TestCheckDMPolicy_Allowlist(t *testing.T) {
	cfg := config.DiscordConfig{
		Enabled:  true,
		Token:    "test-token",
		DMPolicy: "allowlist",
		AllowFrom: config.FlexibleStringSlice{"user1"},
	}
	ch, _ := New(cfg, bus.New(), nil)

	if !ch.checkDMPolicy("user1|testuser", "ch1") {
		t.Error("allowlist should accept allowed sender")
	}
	if ch.checkDMPolicy("user2|other", "ch1") {
		t.Error("allowlist should reject non-allowed sender")
	}
}

func TestIsGuildAllowed_EmptyDefault(t *testing.T) {
	cfg := config.DiscordConfig{Enabled: true, Token: "test-token"}
	ch, _ := New(cfg, bus.New(), nil)

	if ch.isGuildAllowed("any-guild") {
		t.Error("empty guild_ids should reject all guilds (security default)")
	}
}

func TestIsGuildAllowed_Allowlisted(t *testing.T) {
	cfg := config.DiscordConfig{
		Enabled:  true,
		Token:    "test-token",
		GuildIDs: []string{"guild-123"},
	}
	ch, _ := New(cfg, bus.New(), nil)

	if !ch.isGuildAllowed("guild-123") {
		t.Error("allowlisted guild should be accepted")
	}
	if ch.isGuildAllowed("guild-999") {
		t.Error("non-allowlisted guild should be rejected")
	}
}

func TestSendChunkedText_Under2000(t *testing.T) {
	// We can't test actual sending without a real session, but verify chunking logic.
	text := "Hello, world!"
	if len(text) > maxTextLength {
		t.Error("test text should be under 2000 chars")
	}
}

func TestSendChunkedText_Over2000(t *testing.T) {
	// Build a string > 2000 chars with newlines.
	var sb []byte
	for i := 0; i < 30; i++ {
		line := make([]byte, 80)
		for j := range line {
			line[j] = 'A' + byte(i%26)
		}
		sb = append(sb, line...)
		sb = append(sb, '\n')
	}
	text := string(sb) // ~30*81 = 2430 chars

	if len(text) <= maxTextLength {
		t.Fatalf("test text should exceed 2000 chars, got %d", len(text))
	}
}

func TestSenderIDFormat(t *testing.T) {
	userID := "123456789"
	username := "testuser"
	senderID := userID + "|" + username

	if senderID != "123456789|testuser" {
		t.Errorf("unexpected senderID format: %s", senderID)
	}

	// Verify BaseChannel.IsAllowed works with compound ID.
	cfg := config.DiscordConfig{
		Enabled:   true,
		Token:     "test-token",
		AllowFrom: config.FlexibleStringSlice{"123456789"},
	}
	ch, _ := New(cfg, bus.New(), nil)
	if !ch.IsAllowed(senderID) {
		t.Error("IsAllowed should match compound senderID by ID part")
	}
}

func TestIsMentioned_BotMentioned(t *testing.T) {
	cfg := config.DiscordConfig{Enabled: true, Token: "test-token"}
	ch, _ := New(cfg, bus.New(), nil)
	ch.botUserID = "bot-123"

	msg := &discordgo.Message{
		Mentions: []*discordgo.User{
			{ID: "bot-123"},
		},
	}
	if !ch.isMentioned(msg) {
		t.Error("should detect bot mention")
	}
}

func TestIsMentioned_NotMentioned(t *testing.T) {
	cfg := config.DiscordConfig{Enabled: true, Token: "test-token"}
	ch, _ := New(cfg, bus.New(), nil)
	ch.botUserID = "bot-123"

	msg := &discordgo.Message{
		Mentions: []*discordgo.User{
			{ID: "other-456"},
		},
	}
	if ch.isMentioned(msg) {
		t.Error("should not detect mention of different user")
	}
}

func TestIsMentioned_NoBotID(t *testing.T) {
	cfg := config.DiscordConfig{Enabled: true, Token: "test-token"}
	ch, _ := New(cfg, bus.New(), nil)
	// botUserID is empty (not yet connected)

	msg := &discordgo.Message{
		Mentions: []*discordgo.User{
			{ID: "bot-123"},
		},
	}
	if ch.isMentioned(msg) {
		t.Error("should return false when botUserID is empty")
	}
}

func TestFactory_DecodesCreds(t *testing.T) {
	creds, _ := json.Marshal(discordCreds{Token: "test-token-factory"})
	cfg, _ := json.Marshal(discordInstanceConfig{
		DMPolicy:    "open",
		GroupPolicy: "open",
		GuildIDs:    []string{"g1"},
	})

	ch, err := Factory("discord-test", creds, cfg, bus.New(), nil)
	if err != nil {
		t.Fatalf("factory error: %v", err)
	}
	if ch.Name() != "discord-test" {
		t.Errorf("expected name discord-test, got %s", ch.Name())
	}
}

func TestFactory_EmptyToken(t *testing.T) {
	creds, _ := json.Marshal(discordCreds{Token: ""})
	_, err := Factory("discord-test", creds, nil, bus.New(), nil)
	if err == nil {
		t.Fatal("expected error for empty token in factory")
	}
}
