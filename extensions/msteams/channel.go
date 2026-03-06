package msteams

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/nextlevelbuilder/goclaw/internal/bus"
	"github.com/nextlevelbuilder/goclaw/internal/channels"
	"github.com/nextlevelbuilder/goclaw/internal/config"
)

// MSTeamsChannel implements channels.Channel for Microsoft Teams via Bot Framework.
type MSTeamsChannel struct {
	channels.BaseChannel
	cfg             config.MSTeamsConfig
	tokenProv       *tokenProvider
	keyCache        *jwksCache
	httpClient      *http.Client
	handler         *webhookHandler
	allowedPrefixes []string // override allowedServiceURLPrefixes for testing; nil = use default
}

var _ channels.Channel = (*MSTeamsChannel)(nil)

// New creates a new MSTeamsChannel from config.
// Returns error if TenantID is empty or "common" (ADR-007 CTO decision).
func New(cfg config.MSTeamsConfig, msgBus *bus.MessageBus) (*MSTeamsChannel, error) {
	if cfg.AppID == "" {
		return nil, fmt.Errorf("msteams: MSTEAMS_APP_ID is required")
	}
	if cfg.AppPassword == "" {
		return nil, fmt.Errorf("msteams: MSTEAMS_APP_PASSWORD is required")
	}
	if cfg.TenantID == "" {
		return nil, fmt.Errorf("msteams: MSTEAMS_TENANT_ID is required — use your organization's specific Azure tenant ID")
	}
	if cfg.TenantID == "common" {
		return nil, fmt.Errorf("msteams: MSTEAMS_TENANT_ID must not be 'common' — this would allow any Microsoft 365 user to reach the bot (ADR-007 CTO decision)")
	}

	webhookPath := cfg.WebhookPath
	if webhookPath == "" {
		webhookPath = "/v1/channels/msteams/webhook"
		cfg.WebhookPath = webhookPath
	}

	keyCache := newJWKSCache()
	tp := newTokenProvider(cfg.AppID, cfg.AppPassword)

	ch := &MSTeamsChannel{
		cfg:        cfg,
		tokenProv:  tp,
		keyCache:   keyCache,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
	ch.BaseChannel = *channels.NewBaseChannel("msteams", msgBus, nil)

	ch.handler = &webhookHandler{
		appID:    cfg.AppID,
		keyCache: keyCache,
		msgBus:   msgBus,
	}

	return ch, nil
}

// Start validates config and pre-warms the JWKS cache (non-blocking background).
func (c *MSTeamsChannel) Start(ctx context.Context) error {
	c.SetRunning(true)
	slog.Info("msteams channel started", "webhook_path", c.cfg.WebhookPath, "tenant_id", c.cfg.TenantID)

	// Pre-warm JWKS cache in background — first request may be slower without this
	go func() {
		if _, err := c.keyCache.GetKey(""); err != nil {
			// No-op: error just means cache miss on first real request; that path handles the error
			slog.Debug("msteams: JWKS pre-warm attempt completed (miss expected on empty kid)", "info", err)
		}
	}()

	return nil
}

// Stop marks the channel as stopped.
func (c *MSTeamsChannel) Stop(_ context.Context) error {
	c.SetRunning(false)
	slog.Info("msteams channel stopped")
	return nil
}

// allowedServiceURLPrefixes is the allowlist of Bot Framework service URL prefixes.
// CTO-47 / PT-07: defense-in-depth against SSRF via crafted serviceURL.
var allowedServiceURLPrefixes = []string{
	"https://smba.trafficmanager.net/",
	"https://smba-1.trafficmanager.net/",
}

// ValidateServiceURL checks that a service URL is a valid Bot Framework endpoint.
// Returns error if the URL is not HTTPS or does not match the allowlist.
func ValidateServiceURL(rawURL string) error {
	return validateServiceURLWithPrefixes(rawURL, allowedServiceURLPrefixes)
}

func validateServiceURLWithPrefixes(rawURL string, prefixes []string) error {
	if rawURL == "" {
		return fmt.Errorf("msteams: service_url is empty")
	}
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("msteams: invalid service_url: %w", err)
	}
	// Check prefix match first — custom prefixes (testing) may allow http://.
	for _, prefix := range prefixes {
		if strings.HasPrefix(rawURL, prefix) {
			return nil
		}
	}
	// HTTPS enforcement only for production (default) prefixes.
	if parsed.Scheme != "https" {
		return fmt.Errorf("msteams: service_url must use HTTPS, got %q", parsed.Scheme)
	}
	return fmt.Errorf("msteams: service_url %q not in allowed Bot Framework prefixes", rawURL)
}

// Send delivers an outbound message to a Teams conversation via Bot Framework REST API.
// msg.Metadata["service_url"] must be set (populated from inbound activity's ServiceURL).
func (c *MSTeamsChannel) Send(ctx context.Context, msg bus.OutboundMessage) error {
	serviceURL := msg.Metadata["service_url"]
	if msg.ServiceURL != "" {
		serviceURL = msg.ServiceURL
	}
	if serviceURL == "" {
		return fmt.Errorf("msteams: cannot send to %s: service_url not set (required for Bot Framework reply)", msg.ChatID)
	}

	// CTO-47 / PT-07: SSRF defense-in-depth — validate serviceURL against allowlist.
	prefixes := c.allowedPrefixes
	if prefixes == nil {
		prefixes = allowedServiceURLPrefixes
	}
	if err := validateServiceURLWithPrefixes(serviceURL, prefixes); err != nil {
		slog.Warn("msteams: SSRF blocked", "service_url", serviceURL, "error", err)
		return err
	}

	token, err := c.tokenProv.Token()
	if err != nil {
		return fmt.Errorf("msteams: failed to acquire token: %w", err)
	}

	endpoint := fmt.Sprintf("%s/v3/conversations/%s/activities", serviceURL, msg.ChatID)

	var payload interface{}
	if msg.Format == "adaptive_card" && msg.Content != "" {
		// Adaptive Card: wrap as attachment
		payload = map[string]interface{}{
			"type": "message",
			"attachments": []map[string]interface{}{
				{
					"contentType": "application/vnd.microsoft.card.adaptive",
					"content":     json.RawMessage(msg.Content),
				},
			},
		}
	} else {
		payload = map[string]interface{}{
			"type": "message",
			"text": msg.Content,
		}
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("msteams: marshal send payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("msteams: build send request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("msteams: send request failed: %w", err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("msteams: send returned HTTP %d for conversation %s", resp.StatusCode, msg.ChatID)
	}

	slog.Debug("msteams: message sent", "conv_id", msg.ChatID, "status", resp.StatusCode)
	return nil
}

// SetAgentID sets the explicit agent ID for routing (used by InstanceLoader).
func (c *MSTeamsChannel) SetAgentID(id string) {
	c.BaseChannel.SetAgentID(id)
	c.handler.agentID = id
}

// SetTokenEndpoint overrides the token endpoint URL (for testing).
func (c *MSTeamsChannel) SetTokenEndpoint(url string) {
	c.tokenProv.tokenEndpoint = url
}

// SetHTTPClient overrides the HTTP client (for testing).
func (c *MSTeamsChannel) SetHTTPClient(client *http.Client) {
	c.httpClient = client
}

// SetAllowedPrefixes overrides the service URL allowlist (for testing with http:// test servers).
func (c *MSTeamsChannel) SetAllowedPrefixes(prefixes []string) {
	c.allowedPrefixes = prefixes
}

// RegisterRoutes registers the MS Teams webhook endpoint on the given mux.
// Called from cmd/gateway.go via server.AddMuxHandler(ch.RegisterRoutes).
func (c *MSTeamsChannel) RegisterRoutes(mux *http.ServeMux) {
	mux.Handle("POST "+c.cfg.WebhookPath, c.handler)
	slog.Info("msteams webhook registered", "path", c.cfg.WebhookPath)
}
