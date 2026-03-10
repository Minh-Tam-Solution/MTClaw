// Package discord implements the Discord Bot channel.
// Uses discordgo Gateway WebSocket for inbound, REST for outbound.
// Re-added in Sprint 30 (ADR-006-Amendment) for Vietnamese dev team accessibility.
//
// Follows Zalo channel pattern (simplest existing channel).
// Streaming and reactions deferred to Sprint 31.
package discord

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"

	"github.com/Minh-Tam-Solution/MTClaw/internal/bus"
	"github.com/Minh-Tam-Solution/MTClaw/internal/channels"
	"github.com/Minh-Tam-Solution/MTClaw/internal/config"
	"github.com/Minh-Tam-Solution/MTClaw/internal/store"
)

const (
	maxTextLength   = 2000
	pairingDebounce = 60 * time.Second
)

// Channel connects to the Discord Bot API via Gateway WebSocket.
type Channel struct {
	*channels.BaseChannel
	token           string
	session         *discordgo.Session
	dmPolicy        string
	groupPolicy     string
	requireMention  bool
	guildIDs        map[string]bool // allowlisted guild IDs (empty = no guilds allowed)
	pairingService  store.PairingStore
	pairingDebounce sync.Map // senderID → time.Time
	stopCh          chan struct{}
	botUserID       string // populated on Start, for mention detection
}

// New creates a new Discord channel.
func New(cfg config.DiscordConfig, msgBus *bus.MessageBus, pairingSvc store.PairingStore) (*Channel, error) {
	if cfg.Token == "" {
		return nil, fmt.Errorf("discord token is required")
	}

	base := channels.NewBaseChannel("discord", msgBus, cfg.AllowFrom)
	base.ValidatePolicy(cfg.DMPolicy, cfg.GroupPolicy)

	dmPolicy := cfg.DMPolicy
	if dmPolicy == "" {
		dmPolicy = "pairing"
	}

	groupPolicy := cfg.GroupPolicy
	if groupPolicy == "" {
		groupPolicy = "disabled"
	}

	guildIDs := make(map[string]bool, len(cfg.GuildIDs))
	for _, id := range cfg.GuildIDs {
		guildIDs[id] = true
	}

	// Create discordgo session (not yet opened).
	dg, err := discordgo.New("Bot " + cfg.Token)
	if err != nil {
		return nil, fmt.Errorf("create discord session: %w", err)
	}

	// Required intents: guild messages, DMs, and message content (privileged).
	dg.Identify.Intents = discordgo.IntentsGuildMessages |
		discordgo.IntentsDirectMessages |
		discordgo.IntentMessageContent

	return &Channel{
		BaseChannel:    base,
		token:          cfg.Token,
		session:        dg,
		dmPolicy:       dmPolicy,
		groupPolicy:    groupPolicy,
		requireMention: cfg.RequireMention,
		guildIDs:       guildIDs,
		pairingService: pairingSvc,
		stopCh:         make(chan struct{}),
	}, nil
}

// Start opens the Discord Gateway WebSocket and begins receiving messages.
func (c *Channel) Start(_ context.Context) error {
	slog.Info("starting discord bot (gateway)")

	// Register message handler before opening.
	c.session.AddHandler(c.handleMessageCreate)

	if err := c.session.Open(); err != nil {
		return fmt.Errorf("discord gateway open: %w", err)
	}

	// Store bot user ID for mention detection.
	if c.session.State != nil && c.session.State.User != nil {
		c.botUserID = c.session.State.User.ID
		slog.Info("discord bot connected",
			"bot_id", c.botUserID,
			"bot_name", c.session.State.User.Username,
		)
	}

	c.SetRunning(true)
	return nil
}

// Stop gracefully closes the Discord session.
func (c *Channel) Stop(_ context.Context) error {
	slog.Info("stopping discord bot")
	close(c.stopCh)
	if c.session != nil {
		_ = c.session.Close()
	}
	c.SetRunning(false)
	return nil
}

// Send delivers an outbound message to a Discord channel.
func (c *Channel) Send(_ context.Context, msg bus.OutboundMessage) error {
	if !c.IsRunning() {
		return fmt.Errorf("discord bot not running")
	}
	return c.sendChunkedText(msg.ChatID, msg.Content)
}

// --- Message handling ---

func (c *Channel) handleMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Skip bot messages (including our own).
	if m.Author == nil || m.Author.Bot {
		return
	}

	// Determine peer kind: DM vs guild.
	peerKind := "group"
	if m.GuildID == "" {
		peerKind = "direct"
	}

	senderID := m.Author.ID + "|" + m.Author.Username
	chatID := m.ChannelID

	// Policy enforcement.
	if peerKind == "direct" {
		if !c.checkDMPolicy(senderID, chatID) {
			return
		}
	} else {
		// Guild message: check guild allowlist.
		if !c.isGuildAllowed(m.GuildID) {
			slog.Debug("discord message from non-allowlisted guild",
				"guild_id", m.GuildID, "sender_id", senderID)
			return
		}

		// Group policy check.
		if !c.CheckPolicy(peerKind, c.dmPolicy, c.groupPolicy, senderID) {
			return
		}

		// Mention gating for guild channels.
		if c.requireMention && !c.isMentioned(m.Message) {
			return
		}
	}

	content := m.Content
	if content == "" {
		content = "[empty message]"
	}

	// Strip bot mention from content for cleaner input.
	if c.botUserID != "" {
		content = strings.ReplaceAll(content, "<@"+c.botUserID+">", "")
		content = strings.ReplaceAll(content, "<@!"+c.botUserID+">", "")
		content = strings.TrimSpace(content)
		if content == "" {
			content = "[mention only]"
		}
	}

	slog.Debug("discord message received",
		"sender_id", senderID,
		"chat_id", chatID,
		"peer_kind", peerKind,
		"guild_id", m.GuildID,
		"preview", channels.Truncate(content, 50),
	)

	metadata := map[string]string{
		"platform":   "discord",
		"message_id": m.ID,
	}
	if m.GuildID != "" {
		metadata["guild_id"] = m.GuildID
	}

	c.HandleMessage(senderID, chatID, content, nil, metadata, peerKind)
}

// --- DM Policy ---

func (c *Channel) checkDMPolicy(senderID, chatID string) bool {
	switch c.dmPolicy {
	case "disabled":
		slog.Debug("discord DM rejected: DMs disabled", "sender_id", senderID)
		return false

	case "open":
		return true

	case "allowlist":
		if !c.IsAllowed(senderID) {
			slog.Debug("discord DM rejected by allowlist", "sender_id", senderID)
			return false
		}
		return true

	default: // "pairing"
		paired := false
		if c.pairingService != nil {
			paired = c.pairingService.IsPaired(senderID, c.Name())
		}
		inAllowList := c.HasAllowList() && c.IsAllowed(senderID)

		if paired || inAllowList {
			return true
		}

		c.sendPairingReply(senderID, chatID)
		return false
	}
}

func (c *Channel) sendPairingReply(senderID, chatID string) {
	if c.pairingService == nil {
		return
	}

	// Debounce.
	if lastSent, ok := c.pairingDebounce.Load(senderID); ok {
		if time.Since(lastSent.(time.Time)) < pairingDebounce {
			return
		}
	}

	code, err := c.pairingService.RequestPairing(senderID, c.Name(), chatID, "default")
	if err != nil {
		slog.Debug("discord pairing request failed", "sender_id", senderID, "error", err)
		return
	}

	replyText := fmt.Sprintf(
		"MTClaw: access not configured.\n\nYour Discord user: %s\n\nPairing code: %s\n\nAsk the bot owner to approve with:\n  mtclaw pairing approve %s",
		senderID, code, code,
	)

	if _, err := c.session.ChannelMessageSend(chatID, replyText); err != nil {
		slog.Warn("failed to send discord pairing reply", "error", err)
	} else {
		c.pairingDebounce.Store(senderID, time.Now())
		slog.Info("discord pairing reply sent", "sender_id", senderID, "code", code)
	}
}

// --- Guild allowlist ---

func (c *Channel) isGuildAllowed(guildID string) bool {
	if len(c.guildIDs) == 0 {
		return false // Security: no guilds allowed by default
	}
	return c.guildIDs[guildID]
}

// --- Mention detection ---

func (c *Channel) isMentioned(m *discordgo.Message) bool {
	if c.botUserID == "" {
		return false
	}
	for _, u := range m.Mentions {
		if u.ID == c.botUserID {
			return true
		}
	}
	return false
}

// --- Chunked text sending ---

func (c *Channel) sendChunkedText(channelID, text string) error {
	for len(text) > 0 {
		chunk := text
		if len(chunk) > maxTextLength {
			cutAt := maxTextLength
			if idx := strings.LastIndex(text[:maxTextLength], "\n"); idx > maxTextLength/2 {
				cutAt = idx + 1
			}
			chunk = text[:cutAt]
			text = text[cutAt:]
		} else {
			text = ""
		}

		if _, err := c.session.ChannelMessageSend(channelID, chunk); err != nil {
			return fmt.Errorf("discord send: %w", err)
		}
	}
	return nil
}
