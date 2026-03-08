package claudecode

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

func TestPermissionStore_Create(t *testing.T) {
	ps := NewPermissionStore()

	req := &PermissionRequest{
		ID:          "perm-001",
		OwnerID:     "tenant-1",
		SessionID:   "br:abc:def",
		Tool:        "Bash",
		RiskLevel:   "high",
		RequestHash: "hash-001",
		ActorID:     "user-1",
	}

	created, err := ps.Create(req)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if created.Decision != PermissionPending {
		t.Errorf("decision: got %q, want %q", created.Decision, PermissionPending)
	}
	if created.ExpiresAt.IsZero() {
		t.Error("ExpiresAt should be set")
	}
	if ps.Count() != 1 {
		t.Errorf("count: got %d, want 1", ps.Count())
	}
}

func TestPermissionStore_Create_MissingFields(t *testing.T) {
	ps := NewPermissionStore()

	tests := []struct {
		name string
		req  *PermissionRequest
	}{
		{"no ID", &PermissionRequest{SessionID: "s", Tool: "t", RequestHash: "h"}},
		{"no session", &PermissionRequest{ID: "i", Tool: "t", RequestHash: "h"}},
		{"no tool", &PermissionRequest{ID: "i", SessionID: "s", RequestHash: "h"}},
		{"no hash", &PermissionRequest{ID: "i", SessionID: "s", Tool: "t"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ps.Create(tt.req)
			if err == nil {
				t.Error("expected error for missing fields")
			}
		})
	}
}

func TestPermissionStore_Dedup(t *testing.T) {
	ps := NewPermissionStore()

	req1 := &PermissionRequest{
		ID: "perm-001", OwnerID: "t1", SessionID: "s1", Tool: "Bash",
		RiskLevel: "high", RequestHash: "same-hash", ActorID: "u1",
	}
	req2 := &PermissionRequest{
		ID: "perm-002", OwnerID: "t1", SessionID: "s1", Tool: "Bash",
		RiskLevel: "high", RequestHash: "same-hash", ActorID: "u1",
	}

	first, err := ps.Create(req1)
	if err != nil {
		t.Fatalf("Create first: %v", err)
	}
	second, err := ps.Create(req2)
	if err != nil {
		t.Fatalf("Create second: %v", err)
	}

	// Dedup: second should return the first request
	if second.ID != first.ID {
		t.Errorf("dedup: got ID %q, want %q", second.ID, first.ID)
	}
	if ps.Count() != 1 {
		t.Errorf("count after dedup: got %d, want 1", ps.Count())
	}
}

func TestPermissionStore_Get(t *testing.T) {
	ps := NewPermissionStore()
	req := &PermissionRequest{
		ID: "perm-001", OwnerID: "t1", SessionID: "s1", Tool: "Read",
		RiskLevel: "low", RequestHash: "h1", ActorID: "u1",
	}
	ps.Create(req)

	got := ps.Get("perm-001")
	if got == nil {
		t.Fatal("expected to find request")
	}
	if got.Tool != "Read" {
		t.Errorf("tool: got %q, want %q", got.Tool, "Read")
	}
}

func TestPermissionStore_Get_NotFound(t *testing.T) {
	ps := NewPermissionStore()
	if ps.Get("nonexist") != nil {
		t.Error("expected nil for nonexistent request")
	}
}

func TestPermissionStore_Decide_Approve(t *testing.T) {
	ps := NewPermissionStore()
	req := &PermissionRequest{
		ID: "perm-001", OwnerID: "t1", SessionID: "s1", Tool: "Edit",
		RiskLevel: "high", RequestHash: "h1", ActorID: "u1",
	}
	ps.Create(req)

	err := ps.Decide("perm-001", PermissionApproved, "approver-1", []string{"approver-1"})
	if err != nil {
		t.Fatalf("Decide: %v", err)
	}

	got := ps.Get("perm-001")
	if got.Decision != PermissionApproved {
		t.Errorf("decision: got %q, want %q", got.Decision, PermissionApproved)
	}
	if got.DecidedBy != "approver-1" {
		t.Errorf("decided_by: got %q, want %q", got.DecidedBy, "approver-1")
	}
}

func TestPermissionStore_Decide_Deny(t *testing.T) {
	ps := NewPermissionStore()
	req := &PermissionRequest{
		ID: "perm-001", OwnerID: "t1", SessionID: "s1", Tool: "Bash",
		RiskLevel: "high", RequestHash: "h1", ActorID: "u1",
	}
	ps.Create(req)

	err := ps.Decide("perm-001", PermissionDenied, "approver-1", []string{"approver-1"})
	if err != nil {
		t.Fatalf("Decide: %v", err)
	}

	got := ps.Get("perm-001")
	if got.Decision != PermissionDenied {
		t.Errorf("decision: got %q, want %q", got.Decision, PermissionDenied)
	}
}

func TestPermissionStore_Decide_DoubleApply(t *testing.T) {
	ps := NewPermissionStore()
	req := &PermissionRequest{
		ID: "perm-001", OwnerID: "t1", SessionID: "s1", Tool: "Bash",
		RiskLevel: "high", RequestHash: "h1", ActorID: "u1",
	}
	ps.Create(req)

	// First decision
	err := ps.Decide("perm-001", PermissionApproved, "approver-1", []string{"approver-1"})
	if err != nil {
		t.Fatalf("first Decide: %v", err)
	}

	// Second decision — must be rejected (acceptance #5)
	err = ps.Decide("perm-001", PermissionDenied, "approver-1", []string{"approver-1"})
	if err == nil {
		t.Fatal("expected error on double-apply")
	}
	if got := ps.Get("perm-001"); got.Decision != PermissionApproved {
		t.Errorf("decision should remain approved, got %q", got.Decision)
	}
}

func TestPermissionStore_Decide_ACLMismatch(t *testing.T) {
	ps := NewPermissionStore()
	req := &PermissionRequest{
		ID: "perm-001", OwnerID: "t1", SessionID: "s1", Tool: "Edit",
		RiskLevel: "high", RequestHash: "h1", ActorID: "u1",
	}
	ps.Create(req)

	// Actor not in ACL — must be rejected (acceptance #6)
	err := ps.Decide("perm-001", PermissionApproved, "rogue-actor", []string{"approver-1", "approver-2"})
	if err == nil {
		t.Fatal("expected ACL mismatch error")
	}
	if got := ps.Get("perm-001"); got.Decision != PermissionPending {
		t.Errorf("decision should remain pending, got %q", got.Decision)
	}
}

func TestPermissionStore_Timeout_HighRisk_Deny(t *testing.T) {
	ps := NewPermissionStore()
	req := &PermissionRequest{
		ID: "perm-001", OwnerID: "t1", SessionID: "s1", Tool: "Bash",
		RiskLevel: "high", RequestHash: "h1", ActorID: "u1",
		ExpiresAt: time.Now().Add(-1 * time.Second), // already expired
	}
	ps.Create(req)

	got := ps.Get("perm-001")
	if got.Decision != PermissionDenied {
		t.Errorf("high-risk timeout: got %q, want %q", got.Decision, PermissionDenied)
	}
	if got.DecidedBy != "system:timeout" {
		t.Errorf("decided_by: got %q, want %q", got.DecidedBy, "system:timeout")
	}
}

func TestPermissionStore_Timeout_LowRisk_Approve(t *testing.T) {
	ps := NewPermissionStore()
	req := &PermissionRequest{
		ID: "perm-001", OwnerID: "t1", SessionID: "s1", Tool: "Read",
		RiskLevel: "low", RequestHash: "h1", ActorID: "u1",
		ExpiresAt: time.Now().Add(-1 * time.Second), // already expired
	}
	ps.Create(req)

	got := ps.Get("perm-001")
	if got.Decision != PermissionApproved {
		t.Errorf("low-risk timeout: got %q, want %q", got.Decision, PermissionApproved)
	}
	if got.DecidedBy != "system:timeout" {
		t.Errorf("decided_by: got %q, want %q", got.DecidedBy, "system:timeout")
	}
}

func TestPermissionStore_Decide_AfterExpiry(t *testing.T) {
	ps := NewPermissionStore()
	req := &PermissionRequest{
		ID: "perm-001", OwnerID: "t1", SessionID: "s1", Tool: "Bash",
		RiskLevel: "high", RequestHash: "h1", ActorID: "u1",
		ExpiresAt: time.Now().Add(-1 * time.Second),
	}
	ps.Create(req)

	// Try to decide after expiry — should fail
	err := ps.Decide("perm-001", PermissionApproved, "approver-1", []string{"approver-1"})
	if err == nil {
		t.Fatal("expected error deciding after expiry")
	}
}

func TestPermissionStore_ListPending(t *testing.T) {
	ps := NewPermissionStore()
	for i := 0; i < 3; i++ {
		ps.Create(&PermissionRequest{
			ID: fmt.Sprintf("perm-%d", i), OwnerID: "t1", SessionID: "s1",
			Tool: "Bash", RiskLevel: "high",
			RequestHash: fmt.Sprintf("h%d", i), ActorID: "u1",
		})
	}
	// One for a different session
	ps.Create(&PermissionRequest{
		ID: "perm-other", OwnerID: "t1", SessionID: "s2",
		Tool: "Read", RiskLevel: "low",
		RequestHash: "h-other", ActorID: "u1",
	})

	pending := ps.ListPending("s1")
	if len(pending) != 3 {
		t.Errorf("pending for s1: got %d, want 3", len(pending))
	}
}

func TestPermissionStore_Cleanup(t *testing.T) {
	ps := NewPermissionStore()
	req := &PermissionRequest{
		ID: "perm-old", OwnerID: "t1", SessionID: "s1", Tool: "Read",
		RiskLevel: "low", RequestHash: "h-old", ActorID: "u1",
		CreatedAt: time.Now().Add(-2 * time.Hour),
	}
	ps.Create(req)
	ps.Decide("perm-old", PermissionApproved, "u1", []string{"u1"})

	removed := ps.Cleanup(1 * time.Hour)
	if removed != 1 {
		t.Errorf("cleanup: removed %d, want 1", removed)
	}
	if ps.Count() != 0 {
		t.Errorf("count after cleanup: got %d, want 0", ps.Count())
	}
}

func TestPermissionStore_GetByHash(t *testing.T) {
	ps := NewPermissionStore()
	ps.Create(&PermissionRequest{
		ID: "perm-001", OwnerID: "t1", SessionID: "s1", Tool: "Bash",
		RiskLevel: "high", RequestHash: "unique-hash", ActorID: "u1",
	})

	got := ps.GetByHash("unique-hash")
	if got == nil || got.ID != "perm-001" {
		t.Errorf("GetByHash: expected perm-001, got %v", got)
	}
	if ps.GetByHash("nonexist") != nil {
		t.Error("GetByHash should return nil for missing hash")
	}
}

func TestComputeRequestHash_Deterministic(t *testing.T) {
	ts := time.Date(2026, 3, 7, 10, 30, 0, 0, time.UTC)
	input := json.RawMessage(`{"file":"main.go"}`)

	h1 := ComputeRequestHash("br:abc:def", "Edit", input, ts)
	h2 := ComputeRequestHash("br:abc:def", "Edit", input, ts)
	if h1 != h2 {
		t.Errorf("non-deterministic hash: %q != %q", h1, h2)
	}
}

func TestComputeRequestHash_MinuteBucket(t *testing.T) {
	input := json.RawMessage(`{}`)

	// Same minute bucket
	t1 := time.Date(2026, 3, 7, 10, 30, 0, 0, time.UTC)
	t2 := time.Date(2026, 3, 7, 10, 30, 45, 0, time.UTC)
	h1 := ComputeRequestHash("s1", "Bash", input, t1)
	h2 := ComputeRequestHash("s1", "Bash", input, t2)
	if h1 != h2 {
		t.Error("same-minute timestamps should produce same hash")
	}

	// Different minute bucket
	t3 := time.Date(2026, 3, 7, 10, 31, 0, 0, time.UTC)
	h3 := ComputeRequestHash("s1", "Bash", input, t3)
	if h1 == h3 {
		t.Error("different-minute timestamps should produce different hashes")
	}
}

func TestComputeRequestHash_DifferentTools(t *testing.T) {
	ts := time.Now()
	input := json.RawMessage(`{}`)
	h1 := ComputeRequestHash("s1", "Bash", input, ts)
	h2 := ComputeRequestHash("s1", "Edit", input, ts)
	if h1 == h2 {
		t.Error("different tools should produce different hashes")
	}
}

func TestIsHighRisk(t *testing.T) {
	tests := []struct {
		tool     string
		highRisk bool
	}{
		{"Bash", true},
		{"Edit", true},
		{"Write", true},
		{"Agent", true},
		{"Delete", true},
		{"Read", false},
		{"Glob", false},
		{"Grep", false},
		{"Unknown", false},
	}
	for _, tt := range tests {
		if got := IsHighRisk(tt.tool); got != tt.highRisk {
			t.Errorf("IsHighRisk(%q): got %v, want %v", tt.tool, got, tt.highRisk)
		}
	}
}
