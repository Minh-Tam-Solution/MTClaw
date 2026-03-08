package http

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Minh-Tam-Solution/MTClaw/internal/bus"
)

const testWebhookSecret = "test-secret-key"

func computeHMAC(secret string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func TestVerifySignature_Valid(t *testing.T) {
	body := []byte(`{"test":"data"}`)
	sig := computeHMAC(testWebhookSecret, body)
	if !verifySignature(testWebhookSecret, body, sig) {
		t.Error("expected valid signature to pass verification")
	}
}

func TestVerifySignature_Invalid(t *testing.T) {
	body := []byte(`{"test":"data"}`)
	sig := "sha256=0000000000000000000000000000000000000000000000000000000000000000"
	if verifySignature(testWebhookSecret, body, sig) {
		t.Error("expected invalid signature to fail verification")
	}
}

func TestVerifySignature_MalformedPrefix(t *testing.T) {
	body := []byte(`{"test":"data"}`)
	mac := hmac.New(sha256.New, []byte(testWebhookSecret))
	mac.Write(body)
	sig := hex.EncodeToString(mac.Sum(nil)) // missing "sha256=" prefix
	if verifySignature(testWebhookSecret, body, sig) {
		t.Error("expected signature without sha256= prefix to fail")
	}
}

func makePRPayload(action, repo string, prNumber int, sha, htmlURL string) []byte {
	payload := ghWebhookPayload{
		Action: action,
		PullRequest: ghPR{
			Number:  prNumber,
			HTMLURL: htmlURL,
			Head:    ghHead{SHA: sha},
		},
		Repository: ghRepository{FullName: repo},
	}
	b, _ := json.Marshal(payload)
	return b
}

func TestHandleWebhook_PROpened(t *testing.T) {
	msgBus := bus.New()
	handler := NewWebhookGitHubHandler(testWebhookSecret, msgBus)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	body := makePRPayload("opened", "org/repo", 42, "abc123", "https://github.com/org/repo/pull/42")
	sig := computeHMAC(testWebhookSecret, body)

	req := httptest.NewRequest(http.MethodPost, "/github/webhook", bytes.NewReader(body))
	req.Header.Set("X-Hub-Signature-256", sig)
	req.Header.Set("X-GitHub-Event", "pull_request")

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"status":"accepted"`)) {
		t.Errorf("expected accepted response, got %s", rec.Body.String())
	}
}

func TestHandleWebhook_PRSynchronize(t *testing.T) {
	msgBus := bus.New()
	handler := NewWebhookGitHubHandler(testWebhookSecret, msgBus)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	body := makePRPayload("synchronize", "org/repo", 42, "def456", "https://github.com/org/repo/pull/42")
	sig := computeHMAC(testWebhookSecret, body)

	req := httptest.NewRequest(http.MethodPost, "/github/webhook", bytes.NewReader(body))
	req.Header.Set("X-Hub-Signature-256", sig)
	req.Header.Set("X-GitHub-Event", "pull_request")

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"status":"accepted"`)) {
		t.Errorf("expected accepted response, got %s", rec.Body.String())
	}
}

func TestHandleWebhook_IgnoredAction(t *testing.T) {
	msgBus := bus.New()
	handler := NewWebhookGitHubHandler(testWebhookSecret, msgBus)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	body := makePRPayload("closed", "org/repo", 42, "abc123", "https://github.com/org/repo/pull/42")
	sig := computeHMAC(testWebhookSecret, body)

	req := httptest.NewRequest(http.MethodPost, "/github/webhook", bytes.NewReader(body))
	req.Header.Set("X-Hub-Signature-256", sig)
	req.Header.Set("X-GitHub-Event", "pull_request")

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"status":"ignored"`)) {
		t.Errorf("expected ignored response, got %s", rec.Body.String())
	}
}

func TestHandleWebhook_NonPREvent(t *testing.T) {
	msgBus := bus.New()
	handler := NewWebhookGitHubHandler(testWebhookSecret, msgBus)

	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	body := []byte(`{"ref":"refs/heads/main"}`)
	sig := computeHMAC(testWebhookSecret, body)

	req := httptest.NewRequest(http.MethodPost, "/github/webhook", bytes.NewReader(body))
	req.Header.Set("X-Hub-Signature-256", sig)
	req.Header.Set("X-GitHub-Event", "push")

	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
	if !bytes.Contains(rec.Body.Bytes(), []byte(`"status":"ignored"`)) {
		t.Errorf("expected ignored response, got %s", rec.Body.String())
	}
}
