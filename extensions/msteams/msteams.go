// Package msteams implements the MS Teams channel extension for MTClaw.
// Uses Microsoft Bot Framework v3 REST API for inbound (webhook) and outbound (reply) messaging.
//
// Architecture: extensions/msteams follows the established channel factory pattern.
// Only one line changes in cmd/gateway.go — RegisterFactory("msteams", msteams.Factory).
//
// ADR-007 (APPROVED 2026-03-17): Bot Framework REST, MTS tenant only, app password auth.
// CTO decisions: TenantID must not be "common"; respond in conversation (not DM).
package msteams

import (
	"encoding/json"
	"fmt"

	"github.com/nextlevelbuilder/goclaw/internal/bus"
	"github.com/nextlevelbuilder/goclaw/internal/channels"
	"github.com/nextlevelbuilder/goclaw/internal/config"
	"github.com/nextlevelbuilder/goclaw/internal/store"
)

// msteamsCreds maps the credentials JSON from the channel_instances table.
type msteamsCreds struct {
	AppID       string `json:"app_id"`
	AppPassword string `json:"app_password"`
	TenantID    string `json:"tenant_id"`
}

// msteamsInstanceConfig maps the non-secret config JSONB from the channel_instances table.
type msteamsInstanceConfig struct {
	WebhookPath string `json:"webhook_path,omitempty"`
}

// Factory creates an MSTeamsChannel from DB instance data (managed mode).
// Registered in cmd/gateway.go: instanceLoader.RegisterFactory("msteams", msteams.Factory)
func Factory(name string, creds json.RawMessage, cfgJSON json.RawMessage,
	msgBus *bus.MessageBus, _ store.PairingStore) (channels.Channel, error) {

	var c msteamsCreds
	if len(creds) > 0 {
		if err := json.Unmarshal(creds, &c); err != nil {
			return nil, fmt.Errorf("msteams: decode credentials: %w", err)
		}
	}

	var ic msteamsInstanceConfig
	if len(cfgJSON) > 0 {
		if err := json.Unmarshal(cfgJSON, &ic); err != nil {
			return nil, fmt.Errorf("msteams: decode config: %w", err)
		}
	}

	cfg := config.MSTeamsConfig{
		Enabled:     true,
		AppID:       c.AppID,
		AppPassword: c.AppPassword,
		TenantID:    c.TenantID,
		WebhookPath: ic.WebhookPath,
	}

	ch, err := New(cfg, msgBus)
	if err != nil {
		return nil, err
	}

	ch.SetName(name)
	return ch, nil
}
