package claudecode

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHookServer_HealthEndpoint(t *testing.T) {
	mgr := testManager()
	hs := NewHookServer(0, mgr, nil)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	hs.server.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("health: got %d, want 200", w.Code)
	}
}

func TestHookServer_HookMissingHeaders(t *testing.T) {
	mgr := testManager()
	hs := NewHookServer(0, mgr, nil)

	body := `{"event":"stop"}`
	req := httptest.NewRequest(http.MethodPost, "/hook", bytes.NewBufferString(body))
	w := httptest.NewRecorder()
	hs.server.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("missing headers: got %d, want 401", w.Code)
	}
}

func TestHookServer_HookWrongMethod(t *testing.T) {
	mgr := testManager()
	hs := NewHookServer(0, mgr, nil)

	req := httptest.NewRequest(http.MethodGet, "/hook", nil)
	w := httptest.NewRecorder()
	hs.server.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("wrong method: got %d, want 405", w.Code)
	}
}

func TestHookServer_HookInvalidSession(t *testing.T) {
	mgr := testManager()
	hs := NewHookServer(0, mgr, nil)

	body := `{"event":"stop"}`
	ts := time.Now().Unix()
	sig := SignHook("fake-secret", body, ts)

	req := httptest.NewRequest(http.MethodPost, "/hook", bytes.NewBufferString(body))
	req.Header.Set("X-Hook-Signature", sig)
	req.Header.Set("X-Hook-Timestamp", fmt.Sprintf("%d", ts))
	req.Header.Set("X-Hook-Session", "br:nonexist:session")
	w := httptest.NewRecorder()
	hs.server.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("invalid session: got %d, want 404", w.Code)
	}
}

func TestHookServer_HookValidStop(t *testing.T) {
	mgr := testManager()
	ctx := testContext("tenant-1")

	session, err := mgr.CreateSession(ctx, testOpts("tenant-1"))
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	var notified bool
	notifier := NewNotifier(func(_ context.Context, _, _, _ string) error {
		notified = true
		return nil
	})
	hs := NewHookServer(0, mgr, notifier)

	hookReq := HookRequest{
		SessionID: session.ID,
		Event:     "stop",
		ExitCode:  0,
		Summary:   "Task complete",
	}
	body, _ := json.Marshal(hookReq)
	ts := time.Now().Unix()
	sig := SignHook(session.HookSecret, string(body), ts)

	req := httptest.NewRequest(http.MethodPost, "/hook", bytes.NewBuffer(body))
	req.Header.Set("X-Hook-Signature", sig)
	req.Header.Set("X-Hook-Timestamp", fmt.Sprintf("%d", ts))
	req.Header.Set("X-Hook-Session", session.ID)
	w := httptest.NewRecorder()
	hs.server.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("valid stop: got %d, want 200", w.Code)
	}
	if !notified {
		t.Error("notifier should have been called")
	}
}

func TestHookServer_HookWrongSignature(t *testing.T) {
	mgr := testManager()
	ctx := testContext("tenant-1")

	session, err := mgr.CreateSession(ctx, testOpts("tenant-1"))
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	hs := NewHookServer(0, mgr, nil)

	body := `{"event":"stop"}`
	ts := time.Now().Unix()
	wrongSig := SignHook("wrong-secret", body, ts)

	req := httptest.NewRequest(http.MethodPost, "/hook", bytes.NewBufferString(body))
	req.Header.Set("X-Hook-Signature", wrongSig)
	req.Header.Set("X-Hook-Timestamp", fmt.Sprintf("%d", ts))
	req.Header.Set("X-Hook-Session", session.ID)
	w := httptest.NewRecorder()
	hs.server.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("wrong signature: got %d, want 403", w.Code)
	}
}

func TestHookServer_PermissionEvent_Returns202(t *testing.T) {
	mgr := testManager()
	ctx := testContext("tenant-1")

	session, err := mgr.CreateSession(ctx, testOpts("tenant-1"))
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	hs := NewHookServer(0, mgr, nil)

	hookReq := HookRequest{
		SessionID: session.ID,
		Event:     "permission",
		Tool:      "Bash",
		ToolInput: json.RawMessage(`{"command":"rm -rf /tmp/test"}`),
	}
	body, _ := json.Marshal(hookReq)
	ts := time.Now().Unix()
	sig := SignHook(session.HookSecret, string(body), ts)

	req := httptest.NewRequest(http.MethodPost, "/hook", bytes.NewBuffer(body))
	req.Header.Set("X-Hook-Signature", sig)
	req.Header.Set("X-Hook-Timestamp", fmt.Sprintf("%d", ts))
	req.Header.Set("X-Hook-Session", session.ID)
	w := httptest.NewRecorder()
	hs.server.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("permission event: got %d, want 202", w.Code)
	}

	var resp PermissionResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Decision != PermissionPending {
		t.Errorf("decision: got %q, want %q", resp.Decision, PermissionPending)
	}
	if resp.Tool != "Bash" {
		t.Errorf("tool: got %q, want %q", resp.Tool, "Bash")
	}
	if resp.ID == "" {
		t.Error("permission ID should not be empty")
	}
}

func TestHookServer_PermissionEvent_MissingTool(t *testing.T) {
	mgr := testManager()
	ctx := testContext("tenant-1")

	session, err := mgr.CreateSession(ctx, testOpts("tenant-1"))
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	hs := NewHookServer(0, mgr, nil)

	hookReq := HookRequest{
		SessionID: session.ID,
		Event:     "permission",
		// Tool intentionally omitted
	}
	body, _ := json.Marshal(hookReq)
	ts := time.Now().Unix()
	sig := SignHook(session.HookSecret, string(body), ts)

	req := httptest.NewRequest(http.MethodPost, "/hook", bytes.NewBuffer(body))
	req.Header.Set("X-Hook-Signature", sig)
	req.Header.Set("X-Hook-Timestamp", fmt.Sprintf("%d", ts))
	req.Header.Set("X-Hook-Session", session.ID)
	w := httptest.NewRecorder()
	hs.server.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("missing tool: got %d, want 400", w.Code)
	}
}

func TestHookServer_PermissionEvent_Dedup(t *testing.T) {
	mgr := testManager()
	ctx := testContext("tenant-1")

	session, err := mgr.CreateSession(ctx, testOpts("tenant-1"))
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	hs := NewHookServer(0, mgr, nil)

	hookReq := HookRequest{
		SessionID: session.ID,
		Event:     "permission",
		Tool:      "Edit",
	}
	body, _ := json.Marshal(hookReq)
	ts := time.Now().Unix()
	sig := SignHook(session.HookSecret, string(body), ts)

	// Send twice — should get the same permission ID (dedup)
	var ids [2]string
	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/hook", bytes.NewBuffer(body))
		req.Header.Set("X-Hook-Signature", sig)
		req.Header.Set("X-Hook-Timestamp", fmt.Sprintf("%d", ts))
		req.Header.Set("X-Hook-Session", session.ID)
		w := httptest.NewRecorder()
		hs.server.Handler.ServeHTTP(w, req)

		if w.Code != http.StatusAccepted {
			t.Fatalf("request %d: got %d, want 202", i, w.Code)
		}
		var resp PermissionResponse
		json.NewDecoder(w.Body).Decode(&resp)
		ids[i] = resp.ID
	}

	if ids[0] != ids[1] {
		t.Errorf("dedup failed: got different IDs %q and %q", ids[0], ids[1])
	}
}

func TestHookServer_PermissionPoll(t *testing.T) {
	mgr := testManager()
	ctx := testContext("tenant-1")

	session, err := mgr.CreateSession(ctx, testOpts("tenant-1"))
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	hs := NewHookServer(0, mgr, nil)

	// Create a permission request via hook
	hookReq := HookRequest{
		SessionID: session.ID,
		Event:     "permission",
		Tool:      "Bash",
	}
	body, _ := json.Marshal(hookReq)
	ts := time.Now().Unix()
	sig := SignHook(session.HookSecret, string(body), ts)

	createReq := httptest.NewRequest(http.MethodPost, "/hook", bytes.NewBuffer(body))
	createReq.Header.Set("X-Hook-Signature", sig)
	createReq.Header.Set("X-Hook-Timestamp", fmt.Sprintf("%d", ts))
	createReq.Header.Set("X-Hook-Session", session.ID)
	createW := httptest.NewRecorder()
	hs.server.Handler.ServeHTTP(createW, createReq)

	var createResp PermissionResponse
	json.NewDecoder(createW.Body).Decode(&createResp)

	// Poll for the permission
	pollReq := httptest.NewRequest(http.MethodGet, "/hook/permission/"+createResp.ID, nil)
	pollW := httptest.NewRecorder()
	hs.server.Handler.ServeHTTP(pollW, pollReq)

	if pollW.Code != http.StatusOK {
		t.Errorf("poll: got %d, want 200", pollW.Code)
	}

	var pollResp PermissionResponse
	json.NewDecoder(pollW.Body).Decode(&pollResp)
	if pollResp.Decision != PermissionPending {
		t.Errorf("poll decision: got %q, want %q", pollResp.Decision, PermissionPending)
	}
}

func TestHookServer_PermissionPoll_NotFound(t *testing.T) {
	mgr := testManager()
	hs := NewHookServer(0, mgr, nil)

	req := httptest.NewRequest(http.MethodGet, "/hook/permission/nonexist", nil)
	w := httptest.NewRecorder()
	hs.server.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("poll not found: got %d, want 404", w.Code)
	}
}

func TestHookServer_PermissionPoll_WrongMethod(t *testing.T) {
	mgr := testManager()
	hs := NewHookServer(0, mgr, nil)

	req := httptest.NewRequest(http.MethodPost, "/hook/permission/some-id", nil)
	w := httptest.NewRecorder()
	hs.server.Handler.ServeHTTP(w, req)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("wrong method: got %d, want 405", w.Code)
	}
}

func TestHookServer_PermissionPoll_AfterDecision(t *testing.T) {
	mgr := testManager()
	ctx := testContext("tenant-1")

	session, err := mgr.CreateSession(ctx, testOpts("tenant-1"))
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	hs := NewHookServer(0, mgr, nil)

	// Create permission
	hookReq := HookRequest{SessionID: session.ID, Event: "permission", Tool: "Bash"}
	body, _ := json.Marshal(hookReq)
	ts := time.Now().Unix()
	sig := SignHook(session.HookSecret, string(body), ts)

	createReq := httptest.NewRequest(http.MethodPost, "/hook", bytes.NewBuffer(body))
	createReq.Header.Set("X-Hook-Signature", sig)
	createReq.Header.Set("X-Hook-Timestamp", fmt.Sprintf("%d", ts))
	createReq.Header.Set("X-Hook-Session", session.ID)
	createW := httptest.NewRecorder()
	hs.server.Handler.ServeHTTP(createW, createReq)

	var createResp PermissionResponse
	json.NewDecoder(createW.Body).Decode(&createResp)

	// Approve via store directly (simulating Telegram callback)
	err = hs.Permissions().Decide(createResp.ID, PermissionApproved, session.OwnerActorID, []string{session.OwnerActorID})
	if err != nil {
		t.Fatalf("decide: %v", err)
	}

	// Poll — should show approved
	pollReq := httptest.NewRequest(http.MethodGet, "/hook/permission/"+createResp.ID, nil)
	pollW := httptest.NewRecorder()
	hs.server.Handler.ServeHTTP(pollW, pollReq)

	var pollResp PermissionResponse
	json.NewDecoder(pollW.Body).Decode(&pollResp)
	if pollResp.Decision != PermissionApproved {
		t.Errorf("after approval: got %q, want %q", pollResp.Decision, PermissionApproved)
	}
}

func TestHookServer_RateLimiter(t *testing.T) {
	mgr := testManager()
	ctx := testContext("tenant-1")

	session, err := mgr.CreateSession(ctx, testOpts("tenant-1"))
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	hs := NewHookServer(0, mgr, nil)

	body := `{"event":"stop"}`
	ts := time.Now().Unix()
	sig := SignHook(session.HookSecret, body, ts)

	// Send 11 requests (limit is 10/sec)
	var lastCode int
	for i := 0; i < 11; i++ {
		req := httptest.NewRequest(http.MethodPost, "/hook", bytes.NewBufferString(body))
		req.Header.Set("X-Hook-Signature", sig)
		req.Header.Set("X-Hook-Timestamp", fmt.Sprintf("%d", ts))
		req.Header.Set("X-Hook-Session", session.ID)
		w := httptest.NewRecorder()
		hs.server.Handler.ServeHTTP(w, req)
		lastCode = w.Code
	}

	if lastCode != http.StatusTooManyRequests {
		t.Errorf("11th request: got %d, want 429", lastCode)
	}
}
