package claudecode

import (
	"testing"
	"time"
)

func TestSignHook_Deterministic(t *testing.T) {
	sig1 := SignHook("secret123", `{"event":"stop"}`, 1000)
	sig2 := SignHook("secret123", `{"event":"stop"}`, 1000)
	if sig1 != sig2 {
		t.Error("same input should produce same signature")
	}
	if len(sig1) != 64 {
		t.Errorf("signature length: got %d, want 64 hex chars", len(sig1))
	}
}

func TestSignHook_DifferentSecrets(t *testing.T) {
	sig1 := SignHook("secret-a", "body", 1000)
	sig2 := SignHook("secret-b", "body", 1000)
	if sig1 == sig2 {
		t.Error("different secrets should produce different signatures")
	}
}

func TestSignHook_DifferentTimestamps(t *testing.T) {
	sig1 := SignHook("secret", "body", 1000)
	sig2 := SignHook("secret", "body", 1001)
	if sig1 == sig2 {
		t.Error("different timestamps should produce different signatures")
	}
}

func TestVerifyHook_Valid(t *testing.T) {
	secret := "test-hook-secret-0123456789abcdef"
	body := `{"session_id":"br:abc12345:def67890","exit_code":0}`
	ts := time.Now().Unix()
	sig := SignHook(secret, body, ts)

	if err := VerifyHook(secret, body, sig, ts); err != nil {
		t.Errorf("valid hook should verify: %v", err)
	}
}

func TestVerifyHook_WrongSecret(t *testing.T) {
	body := `{"event":"stop"}`
	ts := time.Now().Unix()
	sig := SignHook("correct-secret", body, ts)

	err := VerifyHook("wrong-secret", body, sig, ts)
	if err == nil {
		t.Error("wrong secret should fail verification")
	}
}

func TestVerifyHook_TamperedBody(t *testing.T) {
	secret := "my-secret"
	ts := time.Now().Unix()
	sig := SignHook(secret, `{"exit_code":0}`, ts)

	err := VerifyHook(secret, `{"exit_code":1}`, sig, ts)
	if err == nil {
		t.Error("tampered body should fail verification")
	}
}

func TestVerifyHook_ReplayRejected(t *testing.T) {
	secret := "my-secret"
	body := `{"event":"stop"}`
	oldTS := time.Now().Add(-60 * time.Second).Unix() // 60s ago — beyond 30s window
	sig := SignHook(secret, body, oldTS)

	err := VerifyHook(secret, body, sig, oldTS)
	if err == nil {
		t.Error("expired timestamp should be rejected (replay attack)")
	}
}

func TestVerifyHook_FutureTimestamp(t *testing.T) {
	secret := "my-secret"
	body := `{"event":"stop"}`
	futureTS := time.Now().Add(60 * time.Second).Unix() // 60s in future
	sig := SignHook(secret, body, futureTS)

	err := VerifyHook(secret, body, sig, futureTS)
	if err == nil {
		t.Error("future timestamp beyond window should be rejected")
	}
}

func TestVerifyHook_WithinWindow(t *testing.T) {
	secret := "my-secret"
	body := `{"event":"stop"}`
	recentTS := time.Now().Add(-15 * time.Second).Unix() // 15s ago — within 30s window
	sig := SignHook(secret, body, recentTS)

	if err := VerifyHook(secret, body, sig, recentTS); err != nil {
		t.Errorf("recent timestamp should pass: %v", err)
	}
}
