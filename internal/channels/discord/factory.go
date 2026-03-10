package discord

import (
	"encoding/json"
	"fmt"

	"github.com/Minh-Tam-Solution/MTClaw/internal/bus"
	"github.com/Minh-Tam-Solution/MTClaw/internal/channels"
	"github.com/Minh-Tam-Solution/MTClaw/internal/config"
	"github.com/Minh-Tam-Solution/MTClaw/internal/store"
)

// discordCreds maps the credentials JSON from the channel_instances table.
type discordCreds struct {
	Token string `json:"token"`
}

// discordInstanceConfig maps the non-secret config JSONB from the channel_instances table.
type discordInstanceConfig struct {
	DMPolicy       string   `json:"dm_policy,omitempty"`
	GroupPolicy    string   `json:"group_policy,omitempty"`
	RequireMention bool     `json:"require_mention,omitempty"`
	GuildIDs       []string `json:"guild_ids,omitempty"`
	AllowFrom      []string `json:"allow_from,omitempty"`
}

// Factory creates a Discord channel from DB instance data.
func Factory(name string, creds json.RawMessage, cfg json.RawMessage,
	msgBus *bus.MessageBus, pairingSvc store.PairingStore) (channels.Channel, error) {

	var c discordCreds
	if len(creds) > 0 {
		if err := json.Unmarshal(creds, &c); err != nil {
			return nil, fmt.Errorf("decode discord credentials: %w", err)
		}
	}
	if c.Token == "" {
		return nil, fmt.Errorf("discord token is required")
	}

	var ic discordInstanceConfig
	if len(cfg) > 0 {
		if err := json.Unmarshal(cfg, &ic); err != nil {
			return nil, fmt.Errorf("decode discord config: %w", err)
		}
	}

	dCfg := config.DiscordConfig{
		Enabled:        true,
		Token:          c.Token,
		AllowFrom:      ic.AllowFrom,
		DMPolicy:       ic.DMPolicy,
		GroupPolicy:    ic.GroupPolicy,
		RequireMention: ic.RequireMention,
		GuildIDs:       ic.GuildIDs,
	}

	ch, err := New(dCfg, msgBus, pairingSvc)
	if err != nil {
		return nil, err
	}

	ch.SetName(name)
	return ch, nil
}
