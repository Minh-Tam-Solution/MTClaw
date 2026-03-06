package msteams

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/nextlevelbuilder/goclaw/internal/bus"
)

// activity is the minimal Bot Framework Activity schema we need to process.
type activity struct {
	Type         string       `json:"type"`
	ID           string       `json:"id"`
	ServiceURL   string       `json:"serviceUrl"`
	ChannelID    string       `json:"channelId"`
	From         activityFrom `json:"from"`
	Conversation activityConv `json:"conversation"`
	Text         string       `json:"text"`
}

type activityFrom struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type activityConv struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// webhookHandler processes inbound Bot Framework Activity POST requests.
type webhookHandler struct {
	appID    string
	keyCache *jwksCache
	msgBus   *bus.MessageBus
	agentID  string // explicit agent routing (empty = default route)
}

// ServeHTTP handles incoming activity POST requests from Bot Framework.
func (h *webhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Verify Bot Framework JWT
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		slog.Warn("msteams: missing Authorization header")
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
	if err := ValidateBotFrameworkJWT(tokenStr, h.appID, h.keyCache); err != nil {
		slog.Warn("msteams: JWT validation failed", "error", err)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Parse Activity (max 1MB)
	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	var act activity
	if err := json.Unmarshal(body, &act); err != nil {
		slog.Warn("msteams: failed to parse activity", "error", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	switch act.Type {
	case "message":
		h.handleMessage(w, &act)
	case "conversationUpdate":
		// Acknowledge without action (member added events, etc.)
		w.WriteHeader(http.StatusOK)
	default:
		// All other activity types: acknowledge, no-op
		w.WriteHeader(http.StatusOK)
	}
}

// handleMessage processes an inbound text message activity.
func (h *webhookHandler) handleMessage(w http.ResponseWriter, act *activity) {
	text := strings.TrimSpace(act.Text)
	if text == "" {
		slog.Debug("msteams: empty message text, skipping")
		w.WriteHeader(http.StatusOK)
		return
	}

	msg := bus.InboundMessage{
		Channel:    "msteams",
		SenderID:   act.From.ID,
		ChatID:     act.Conversation.ID,
		Content:    text,
		PeerKind:   "direct",
		UserID:     act.From.ID,
		ServiceURL: act.ServiceURL,
		AgentID:    h.agentID,
	}

	h.msgBus.PublishInbound(msg)
	slog.Debug("msteams: published inbound message", "from", act.From.ID, "conv", act.Conversation.ID)
	w.WriteHeader(http.StatusOK)
}
