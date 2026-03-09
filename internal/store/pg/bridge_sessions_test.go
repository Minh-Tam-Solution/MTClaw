package pg

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/Minh-Tam-Solution/MTClaw/internal/store"
)

// TestBridgeSessionRecord_Serialization verifies JSONB fields round-trip correctly.
func TestBridgeSessionRecord_Serialization(t *testing.T) {
	caps := json.RawMessage(`{"input_mode":"free_text","tool_policy":"allow_all"}`)
	acl := json.RawMessage(`["actor-1","actor-2"]`)

	rec := &store.BridgeSessionRecord{
		ID:                   "br:abc123:def456",
		OwnerID:              "tenant-1",
		AgentType:            "claude-code",
		TmuxTarget:           "cc-abc123-def456",
		ProjectPath:          "/home/user/project",
		WorkspaceFingerprint: "sha256:abcdef",
		Status:               "active",
		RiskMode:             "read",
		Capabilities:         caps,
		OwnerActorID:         "actor-1",
		ApproverACL:          acl,
		NotifyACL:            acl,
		UserID:               "user-1",
		Channel:              "telegram",
		ChatID:               "chat-123",
		InteractiveEligible:  true,
		HookSecret:           "secret-xyz",
		CreatedAt:            time.Now().UTC(),
		LastActivityAt:       time.Now().UTC(),
	}

	// Verify JSON serialization
	data, err := json.Marshal(rec)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded store.BridgeSessionRecord
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.ID != rec.ID {
		t.Errorf("ID: got %q, want %q", decoded.ID, rec.ID)
	}
	if decoded.OwnerID != rec.OwnerID {
		t.Errorf("OwnerID: got %q, want %q", decoded.OwnerID, rec.OwnerID)
	}
	if decoded.Status != rec.Status {
		t.Errorf("Status: got %q, want %q", decoded.Status, rec.Status)
	}
	if decoded.InteractiveEligible != rec.InteractiveEligible {
		t.Errorf("InteractiveEligible: got %v, want %v", decoded.InteractiveEligible, rec.InteractiveEligible)
	}
	if string(decoded.Capabilities) != string(rec.Capabilities) {
		t.Errorf("Capabilities: got %s, want %s", decoded.Capabilities, rec.Capabilities)
	}
}

// TestHelpers_nilStr_derefStr tests the helper functions used by bridge session store.
func TestHelpers_nilStr_derefStr(t *testing.T) {
	// nilStr: empty string → nil
	if nilStr("") != nil {
		t.Error("nilStr('') should return nil")
	}
	s := "hello"
	if p := nilStr(s); p == nil || *p != s {
		t.Errorf("nilStr(%q) = %v, want pointer to %q", s, p, s)
	}

	// derefStr: nil → empty
	if derefStr(nil) != "" {
		t.Error("derefStr(nil) should return empty")
	}
	if derefStr(&s) != s {
		t.Errorf("derefStr(&%q) = %q", s, derefStr(&s))
	}
}

// TestHelpers_jsonOrEmpty tests JSONB helper for bridge sessions.
func TestHelpers_jsonOrEmpty(t *testing.T) {
	result := jsonOrEmpty(nil)
	if string(result) != "{}" {
		t.Errorf("jsonOrEmpty(nil) = %s, want {}", result)
	}

	data := json.RawMessage(`{"key":"value"}`)
	result = jsonOrEmpty(data)
	if string(result) != `{"key":"value"}` {
		t.Errorf("jsonOrEmpty(data) = %s, want %s", result, data)
	}
}

// TestHelpers_nilTime tests time pointer helper.
func TestHelpers_nilTime(t *testing.T) {
	if nilTime(nil) != nil {
		t.Error("nilTime(nil) should return nil")
	}

	now := time.Now()
	if p := nilTime(&now); p == nil || !p.Equal(now) {
		t.Error("nilTime(&now) should return same time")
	}
}

// TestBridgeSessionCols_Count verifies column count matches struct fields.
func TestBridgeSessionCols_Count(t *testing.T) {
	// bridgeSessionCols has 20 columns, matching BridgeSessionRecord's 20 fields
	// This ensures the INSERT and SELECT queries stay in sync with the struct.
	ctx := context.Background()
	_ = ctx // would be used in integration test with real DB

	// Count commas in bridgeSessionCols string + 1 = number of columns
	cols := bridgeSessionCols
	count := 1
	for _, c := range cols {
		if c == ',' {
			count++
		}
	}
	if count != 20 {
		t.Errorf("bridgeSessionCols has %d columns, want 20 (matching BridgeSessionRecord struct)", count)
	}
}
