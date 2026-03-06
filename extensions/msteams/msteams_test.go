package msteams

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/nextlevelbuilder/goclaw/internal/bus"
	"github.com/nextlevelbuilder/goclaw/internal/config"
)

// ─── Config validation (4 tests) ─────────────────────────────────────────────

func TestMSTeamsFactory_TenantIDRequired(t *testing.T) {
	_, err := New(config.MSTeamsConfig{
		AppID:       "app-id",
		AppPassword: "app-secret",
		TenantID:    "",
	}, bus.New())
	if err == nil {
		t.Fatal("expected error for empty TenantID, got nil")
	}
	if !strings.Contains(err.Error(), "MSTEAMS_TENANT_ID") {
		t.Errorf("error should mention MSTEAMS_TENANT_ID, got: %v", err)
	}
}

func TestMSTeamsFactory_CommonTenantRejected(t *testing.T) {
	_, err := New(config.MSTeamsConfig{
		AppID:       "app-id",
		AppPassword: "app-secret",
		TenantID:    "common", // ADR-007 CTO decision: must never be "common"
	}, bus.New())
	if err == nil {
		t.Fatal("expected error for TenantID='common', got nil")
	}
	if !strings.Contains(err.Error(), "common") {
		t.Errorf("error should mention 'common', got: %v", err)
	}
}

func TestMSTeamsFactory_AppIDAndSecretRequired(t *testing.T) {
	_, err := New(config.MSTeamsConfig{
		AppID:       "",
		AppPassword: "app-secret",
		TenantID:    "mts-tenant-id",
	}, bus.New())
	if err == nil {
		t.Fatal("expected error for empty AppID, got nil")
	}

	_, err = New(config.MSTeamsConfig{
		AppID:       "app-id",
		AppPassword: "",
		TenantID:    "mts-tenant-id",
	}, bus.New())
	if err == nil {
		t.Fatal("expected error for empty AppPassword, got nil")
	}
}

func TestMSTeamsFactory_DefaultWebhookPath(t *testing.T) {
	ch, err := New(config.MSTeamsConfig{
		AppID:       "app-id",
		AppPassword: "app-secret",
		TenantID:    "mts-tenant-id",
	}, bus.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ch.cfg.WebhookPath != "/v1/channels/msteams/webhook" {
		t.Errorf("expected default webhook path, got %q", ch.cfg.WebhookPath)
	}
}

// ─── JWT helpers ─────────────────────────────────────────────────────────────

func newTestRSAKey(t *testing.T) *rsa.PrivateKey {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}
	return key
}

// injectTestKey populates the jwksCache directly with a test key, bypassing HTTP fetch.
func injectTestKey(cache *jwksCache, kid string, pub *rsa.PublicKey) {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	cache.keys = map[string]*rsa.PublicKey{kid: pub}
	cache.expiry = time.Now().Add(1 * time.Hour)
}

// signTestToken creates a signed Bot Framework-style JWT using the test RSA key.
func signTestToken(t *testing.T, priv *rsa.PrivateKey, kid, appID string, expired bool) string {
	t.Helper()
	expiry := time.Now().Add(1 * time.Hour)
	if expired {
		expiry = time.Now().Add(-1 * time.Hour)
	}
	claims := jwt.MapClaims{
		"iss": botFrameworkIssuer,
		"aud": appID,
		"exp": expiry.Unix(),
		"iat": time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid
	signed, err := token.SignedString(priv)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return signed
}

// ─── JWT middleware (3 tests) ─────────────────────────────────────────────────

func TestJWTMiddleware_MissingAuthHeader_Returns401(t *testing.T) {
	keyCache := newJWKSCache()
	h := &webhookHandler{appID: "app-id", keyCache: keyCache, msgBus: bus.New()}

	req := httptest.NewRequest(http.MethodPost, "/v1/channels/msteams/webhook", bytes.NewBufferString(`{}`))
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestJWTMiddleware_InvalidToken_Returns401(t *testing.T) {
	keyCache := newJWKSCache()
	priv := newTestRSAKey(t)
	injectTestKey(keyCache, "kid1", &priv.PublicKey)

	h := &webhookHandler{appID: "app-id", keyCache: keyCache, msgBus: bus.New()}

	req := httptest.NewRequest(http.MethodPost, "/v1/channels/msteams/webhook", bytes.NewBufferString(`{}`))
	req.Header.Set("Authorization", "Bearer not-a-valid-jwt")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rr.Code)
	}
}

func TestJWTMiddleware_ValidToken_CallsNext(t *testing.T) {
	const appID = "test-app-id"
	const kid = "test-kid"

	priv := newTestRSAKey(t)
	keyCache := newJWKSCache()
	injectTestKey(keyCache, kid, &priv.PublicKey)

	msgBus := bus.New()
	h := &webhookHandler{appID: appID, keyCache: keyCache, msgBus: msgBus}

	body, _ := json.Marshal(activity{
		Type:         "message",
		ServiceURL:   "https://smba.trafficmanager.net/us/",
		From:         activityFrom{ID: "user-1"},
		Conversation: activityConv{ID: "conv-1"},
		Text:         "hello",
	})

	tokenStr := signTestToken(t, priv, kid, appID, false)
	req := httptest.NewRequest(http.MethodPost, "/v1/channels/msteams/webhook", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}
}

// ─── Activity parsing (3 tests) ──────────────────────────────────────────────

func TestWebhookHandler_MessageActivity_PublishesToBus(t *testing.T) {
	const appID = "app-id"
	const kid = "kid"
	priv := newTestRSAKey(t)
	keyCache := newJWKSCache()
	injectTestKey(keyCache, kid, &priv.PublicKey)

	msgBus := bus.New()
	h := &webhookHandler{appID: appID, keyCache: keyCache, msgBus: msgBus}

	act := activity{
		Type:         "message",
		ServiceURL:   "https://smba.trafficmanager.net/us/",
		From:         activityFrom{ID: "user-42"},
		Conversation: activityConv{ID: "conv-99"},
		Text:         "run /spec",
	}
	body, _ := json.Marshal(act)
	tokenStr := signTestToken(t, priv, kid, appID, false)

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	// Verify message published to bus (100ms timeout — synchronous handler)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	msg, ok := msgBus.ConsumeInbound(ctx)
	if !ok {
		t.Fatal("expected inbound message to be published to bus")
	}
	if msg.SenderID != "user-42" {
		t.Errorf("expected SenderID=user-42, got %q", msg.SenderID)
	}
	if msg.ChatID != "conv-99" {
		t.Errorf("expected ChatID=conv-99, got %q", msg.ChatID)
	}
	if msg.ServiceURL != "https://smba.trafficmanager.net/us/" {
		t.Errorf("expected ServiceURL to be set, got %q", msg.ServiceURL)
	}
	if msg.Channel != "msteams" {
		t.Errorf("expected channel=msteams, got %q", msg.Channel)
	}
}

func TestWebhookHandler_EmptyText_Skipped(t *testing.T) {
	const appID = "app-id"
	const kid = "kid"
	priv := newTestRSAKey(t)
	keyCache := newJWKSCache()
	injectTestKey(keyCache, kid, &priv.PublicKey)

	msgBus := bus.New()
	h := &webhookHandler{appID: appID, keyCache: keyCache, msgBus: msgBus}

	act := activity{
		Type:         "message",
		From:         activityFrom{ID: "user-1"},
		Conversation: activityConv{ID: "conv-1"},
		Text:         "",
	}
	body, _ := json.Marshal(act)
	tokenStr := signTestToken(t, priv, kid, appID, false)

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rr.Code)
	}

	// No message should be published for empty text — check with 10ms timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	_, ok := msgBus.ConsumeInbound(ctx)
	if ok {
		t.Error("expected no message published for empty text, but one was received")
	}
}

func TestWebhookHandler_ConversationUpdate_Acknowledged(t *testing.T) {
	const appID = "app-id"
	const kid = "kid"
	priv := newTestRSAKey(t)
	keyCache := newJWKSCache()
	injectTestKey(keyCache, kid, &priv.PublicKey)

	h := &webhookHandler{appID: appID, keyCache: keyCache, msgBus: bus.New()}

	act := activity{
		Type:         "conversationUpdate",
		From:         activityFrom{ID: "user-1"},
		Conversation: activityConv{ID: "conv-1"},
	}
	body, _ := json.Marshal(act)
	tokenStr := signTestToken(t, priv, kid, appID, false)

	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewBuffer(body))
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 for conversationUpdate, got %d", rr.Code)
	}
}

// ─── Send (3 tests) ──────────────────────────────────────────────────────────

func TestMSTeamsChannel_Send_AcquiresTokenFirst(t *testing.T) {
	var tokenRequested bool
	// Mock Bot Framework token endpoint
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tokenRequested = true
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tokenResponse{
			AccessToken: "test-bearer-token",
			ExpiresIn:   3600,
			TokenType:   "Bearer",
		})
	}))
	defer tokenSrv.Close()

	// Mock Bot Framework API server (TLS — CTO-47 SSRF requires HTTPS)
	apiSrv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	defer apiSrv.Close()

	ch, err := New(config.MSTeamsConfig{
		AppID:       "app-id",
		AppPassword: "app-secret",
		TenantID:    "mts-tenant",
	}, bus.New())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// Point token provider at test server endpoint and expire cache to force refresh
	ch.tokenProv.tokenEndpoint = tokenSrv.URL
	ch.tokenProv.cache.expiry = time.Time{}

	// Point API client at TLS test server; allow test server URL for SSRF validation
	ch.httpClient = apiSrv.Client()
	ch.allowedPrefixes = []string{apiSrv.URL}

	err = ch.Send(context.Background(), bus.OutboundMessage{
		Channel:    "msteams",
		ChatID:     "conv-1",
		Content:    "hello",
		ServiceURL: apiSrv.URL,
	})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}

	if !tokenRequested {
		t.Error("expected token to be requested from token provider before sending")
	}
}

func TestMSTeamsChannel_Send_CorrectEndpointURL(t *testing.T) {
	var capturedPath string
	apiSrv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		w.WriteHeader(http.StatusCreated)
	}))
	defer apiSrv.Close()

	ch, err := New(config.MSTeamsConfig{
		AppID:       "app-id",
		AppPassword: "app-secret",
		TenantID:    "mts-tenant",
	}, bus.New())
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// Pre-fill token cache so we skip acquisition
	ch.tokenProv.cache.token = "test-token"
	ch.tokenProv.cache.expiry = time.Now().Add(1 * time.Hour)
	ch.httpClient = apiSrv.Client()
	ch.allowedPrefixes = []string{apiSrv.URL}

	err = ch.Send(context.Background(), bus.OutboundMessage{
		Channel:    "msteams",
		ChatID:     "conv-abc",
		Content:    "test",
		ServiceURL: apiSrv.URL,
	})
	if err != nil {
		t.Fatalf("Send: %v", err)
	}

	expected := "/v3/conversations/conv-abc/activities"
	if capturedPath != expected {
		t.Errorf("expected endpoint path %q, got %q", expected, capturedPath)
	}
}

func TestMSTeamsChannel_Send_HTTPErrorReturnsError(t *testing.T) {
	apiSrv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer apiSrv.Close()

	ch, err := New(config.MSTeamsConfig{
		AppID:       "app-id",
		AppPassword: "app-secret",
		TenantID:    "mts-tenant",
	}, bus.New())
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ch.tokenProv.cache.token = "test-token"
	ch.tokenProv.cache.expiry = time.Now().Add(1 * time.Hour)
	ch.httpClient = apiSrv.Client()
	ch.allowedPrefixes = []string{apiSrv.URL}

	err = ch.Send(context.Background(), bus.OutboundMessage{
		Channel:    "msteams",
		ChatID:     "conv-1",
		Content:    "test",
		ServiceURL: apiSrv.URL,
	})
	if err == nil {
		t.Fatal("expected error for HTTP 403, got nil")
	}
	if !strings.Contains(err.Error(), "403") {
		t.Errorf("error should mention status 403, got: %v", err)
	}
}

// ─── CTO-33 regression (3 tests) ─────────────────────────────────────────────

func TestNoDiscordReferenceInChannelGo(t *testing.T) {
	content, err := os.ReadFile("channel.go")
	if err != nil {
		t.Fatalf("read channel.go: %v", err)
	}
	lower := strings.ToLower(string(content))
	if strings.Contains(lower, "discord") {
		t.Error("channel.go contains 'discord' reference — CTO-33 regression")
	}
}

func TestNoDiscordReferenceInWebhookGo(t *testing.T) {
	content, err := os.ReadFile("webhook.go")
	if err != nil {
		t.Fatalf("read webhook.go: %v", err)
	}
	lower := strings.ToLower(string(content))
	if strings.Contains(lower, "discord") {
		t.Error("webhook.go contains 'discord' reference — CTO-33 regression")
	}
}

func TestNoDiscordReferenceInMSTeamsGo(t *testing.T) {
	content, err := os.ReadFile("msteams.go")
	if err != nil {
		t.Fatalf("read msteams.go: %v", err)
	}
	lower := strings.ToLower(string(content))
	if strings.Contains(lower, "discord") {
		t.Error("msteams.go contains 'discord' reference — CTO-33 regression")
	}
}
