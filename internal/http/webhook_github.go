package http

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/nextlevelbuilder/goclaw/internal/bus"
)

// WebhookGitHubHandler handles GitHub webhook events for PR Gate ENFORCE.
// Verifies HMAC-SHA256 signature before processing.
type WebhookGitHubHandler struct {
	secret string           // HMAC-SHA256 webhook secret
	msgBus *bus.MessageBus  // publishes inbound messages to the processing pipeline
}

// NewWebhookGitHubHandler creates a handler for GitHub PR webhook events.
func NewWebhookGitHubHandler(secret string, msgBus *bus.MessageBus) *WebhookGitHubHandler {
	return &WebhookGitHubHandler{secret: secret, msgBus: msgBus}
}

// RegisterRoutes registers the GitHub webhook endpoint on the given mux.
func (h *WebhookGitHubHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /github/webhook", h.handleWebhook)
}

func (h *WebhookGitHubHandler) handleWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(io.LimitReader(r.Body, 10<<20)) // 10MB max
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// Verify HMAC-SHA256 signature
	signature := r.Header.Get("X-Hub-Signature-256")
	if !verifySignature(h.secret, body, signature) {
		slog.Warn("github.webhook_signature_invalid")
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return
	}

	// Only process pull_request events
	eventType := r.Header.Get("X-GitHub-Event")
	if eventType != "pull_request" {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ignored","event":"%s"}`, eventType)
		return
	}

	// Parse PR payload
	var payload ghWebhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		slog.Warn("github.webhook_parse_error", "error", err)
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	// Only process opened, synchronize, reopened actions
	switch payload.Action {
	case "opened", "synchronize", "reopened":
		// process
	default:
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ignored","action":"%s"}`, payload.Action)
		return
	}

	prURL := payload.PullRequest.HTMLURL
	prNumber := payload.PullRequest.Number
	headSHA := payload.PullRequest.Head.SHA
	repoFullName := payload.Repository.FullName

	slog.Info("github.webhook_pr_event",
		"action", payload.Action,
		"repo", repoFullName,
		"pr", prNumber,
		"sha", headSHA,
	)

	// Publish to message bus — the consumer routes to reviewer SOUL
	h.msgBus.PublishInbound(bus.InboundMessage{
		Channel:  "github",
		SenderID: "github-webhook",
		ChatID:   fmt.Sprintf("%s#%d", repoFullName, prNumber),
		Content:  prURL, // reviewer SOUL uses web_fetch to get diff
		Metadata: map[string]string{
			"command":   "review",
			"rail":      "pr-gate",
			"pr_url":    prURL,
			"pr_number": strconv.Itoa(prNumber),
			"head_sha":  headSHA,
			"repo":      repoFullName,
			"mode":      "enforce",
		},
	})

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"status":"accepted","pr":%d,"repo":"%s"}`, prNumber, repoFullName)
}

// verifySignature checks the HMAC-SHA256 signature from GitHub webhook.
func verifySignature(secret string, body []byte, signature string) bool {
	if !strings.HasPrefix(signature, "sha256=") {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

// ghWebhookPayload is a minimal struct for GitHub webhook PR events.
type ghWebhookPayload struct {
	Action      string       `json:"action"`
	PullRequest ghPR         `json:"pull_request"`
	Repository  ghRepository `json:"repository"`
}

type ghPR struct {
	Number  int    `json:"number"`
	HTMLURL string `json:"html_url"`
	DiffURL string `json:"diff_url"`
	Head    ghHead `json:"head"`
}

type ghHead struct {
	SHA string `json:"sha"`
}

type ghRepository struct {
	FullName string `json:"full_name"`
}
