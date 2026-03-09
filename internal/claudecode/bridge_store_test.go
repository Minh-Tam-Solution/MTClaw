package claudecode

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/Minh-Tam-Solution/MTClaw/internal/store"
)

// mockBridgeSessionStore is an in-memory implementation of BridgeSessionStore for testing.
type mockBridgeSessionStore struct {
	mu      sync.Mutex
	records map[string]*store.BridgeSessionRecord
}

func newMockBridgeSessionStore() *mockBridgeSessionStore {
	return &mockBridgeSessionStore{records: make(map[string]*store.BridgeSessionRecord)}
}

func (m *mockBridgeSessionStore) Upsert(_ context.Context, rec *store.BridgeSessionRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.records[rec.ID] = rec
	return nil
}

func (m *mockBridgeSessionStore) Get(_ context.Context, id string) (*store.BridgeSessionRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	rec, ok := m.records[id]
	if !ok {
		return nil, nil
	}
	return rec, nil
}

func (m *mockBridgeSessionStore) ListByTenant(_ context.Context, tenantID string) ([]*store.BridgeSessionRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []*store.BridgeSessionRecord
	for _, rec := range m.records {
		if rec.OwnerID == tenantID && rec.Status != "stopped" {
			result = append(result, rec)
		}
	}
	return result, nil
}

func (m *mockBridgeSessionStore) ListActive(_ context.Context) ([]*store.BridgeSessionRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var result []*store.BridgeSessionRecord
	for _, rec := range m.records {
		if rec.Status != "stopped" {
			result = append(result, rec)
		}
	}
	return result, nil
}

func (m *mockBridgeSessionStore) UpdateStatus(_ context.Context, id string, status string, stoppedAt *time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if rec, ok := m.records[id]; ok {
		rec.Status = status
		rec.StoppedAt = stoppedAt
	}
	return nil
}

func (m *mockBridgeSessionStore) UpdateRiskMode(_ context.Context, id string, riskMode string, capabilities json.RawMessage) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if rec, ok := m.records[id]; ok {
		rec.RiskMode = riskMode
		rec.Capabilities = capabilities
	}
	return nil
}

func (m *mockBridgeSessionStore) DeleteOlderThan(_ context.Context, cutoff time.Time) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var deleted int64
	for id, rec := range m.records {
		if rec.Status == "stopped" && rec.StoppedAt != nil && rec.StoppedAt.Before(cutoff) {
			delete(m.records, id)
			deleted++
		}
	}
	return deleted, nil
}

func (m *mockBridgeSessionStore) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.records)
}

func (m *mockBridgeSessionStore) get(id string) *store.BridgeSessionRecord {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.records[id]
}

func TestSessionManager_DualWrite_CreateSession(t *testing.T) {
	m := testManager()
	mockStore := newMockBridgeSessionStore()
	m.SetStore(mockStore)

	ctx := testContext("tenant-1")
	opts := testOpts("tenant-1")

	session, err := m.CreateSession(ctx, opts)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	// Verify PG store received the session
	rec := mockStore.get(session.ID)
	if rec == nil {
		t.Fatal("expected session to be persisted to PG store")
	}
	if rec.OwnerID != "tenant-1" {
		t.Errorf("OwnerID: got %q, want tenant-1", rec.OwnerID)
	}
	if rec.Status != "active" {
		t.Errorf("Status: got %q, want active", rec.Status)
	}
	if rec.AgentType != string(AgentClaudeCode) {
		t.Errorf("AgentType: got %q, want %q", rec.AgentType, AgentClaudeCode)
	}
}

func TestSessionManager_DualWrite_KillSession(t *testing.T) {
	m := testManager()
	mockStore := newMockBridgeSessionStore()
	m.SetStore(mockStore)

	ctx := testContext("tenant-1")
	opts := testOpts("tenant-1")

	session, err := m.CreateSession(ctx, opts)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	if err := m.KillSession(ctx, session.ID, opts.OwnerActorID); err != nil {
		t.Fatalf("KillSession: %v", err)
	}

	rec := mockStore.get(session.ID)
	if rec == nil {
		t.Fatal("expected session in PG store")
	}
	if rec.Status != "stopped" {
		t.Errorf("Status after kill: got %q, want stopped", rec.Status)
	}
	if rec.StoppedAt == nil {
		t.Error("StoppedAt should be set after kill")
	}
}

func TestSessionManager_DualWrite_UpdateRiskMode(t *testing.T) {
	m := testManager()
	mockStore := newMockBridgeSessionStore()
	m.SetStore(mockStore)

	ctx := testContext("tenant-1")
	opts := testOpts("tenant-1")

	session, err := m.CreateSession(ctx, opts)
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}

	if err := m.UpdateRiskMode(ctx, session.ID, RiskModePatch, opts.OwnerActorID); err != nil {
		t.Fatalf("UpdateRiskMode: %v", err)
	}

	rec := mockStore.get(session.ID)
	if rec == nil {
		t.Fatal("expected session in PG store")
	}
	if rec.RiskMode != "patch" {
		t.Errorf("RiskMode: got %q, want patch", rec.RiskMode)
	}
}

func TestSessionManager_LoadFromStore(t *testing.T) {
	mockStore := newMockBridgeSessionStore()

	// Seed the mock store with a record
	capsJSON, _ := json.Marshal(CapabilitiesForRisk(RiskModeRead))
	mockStore.Upsert(context.Background(), &store.BridgeSessionRecord{
		ID:                   "br:abc12345:def67890",
		OwnerID:              "tenant-1",
		AgentType:            string(AgentClaudeCode),
		TmuxTarget:           "cc-abc12345-def67890",
		ProjectPath:          "/tmp",
		WorkspaceFingerprint: "sha256:test",
		Status:               "active",
		RiskMode:             "read",
		Capabilities:         capsJSON,
		OwnerActorID:         "actor-1",
		ApproverACL:          json.RawMessage(`[]`),
		NotifyACL:            json.RawMessage(`["actor-1"]`),
		Channel:              "telegram",
		ChatID:               "chat-1",
		CreatedAt:            time.Now().UTC(),
		LastActivityAt:       time.Now().UTC(),
	})

	m := testManager()
	m.SetStore(mockStore)

	n, err := m.LoadFromStore(context.Background())
	if err != nil {
		t.Fatalf("LoadFromStore: %v", err)
	}
	if n != 1 {
		t.Errorf("loaded: got %d, want 1", n)
	}

	// Verify session is in memory with disconnected state
	ctx := testContext("tenant-1")
	sessions, err := m.ListSessions(ctx, "tenant-1")
	if err != nil {
		t.Fatalf("ListSessions: %v", err)
	}
	if len(sessions) != 1 {
		t.Fatalf("sessions count: got %d, want 1", len(sessions))
	}
	if sessions[0].Status != SessionStateDisconnected {
		t.Errorf("recovered session status: got %q, want disconnected", sessions[0].Status)
	}
}

func TestSessionManager_NoStore_NoPanic(t *testing.T) {
	// Verify all operations work without a PG store (standalone mode)
	m := testManager()
	ctx := testContext("tenant-1")
	opts := testOpts("tenant-1")

	session, err := m.CreateSession(ctx, opts)
	if err != nil {
		t.Fatalf("CreateSession without store: %v", err)
	}

	if err := m.UpdateRiskMode(ctx, session.ID, RiskModePatch, opts.OwnerActorID); err != nil {
		t.Fatalf("UpdateRiskMode without store: %v", err)
	}

	if err := m.KillSession(ctx, session.ID, opts.OwnerActorID); err != nil {
		t.Fatalf("KillSession without store: %v", err)
	}

	n, err := m.LoadFromStore(context.Background())
	if err != nil {
		t.Fatalf("LoadFromStore without store: %v", err)
	}
	if n != 0 {
		t.Errorf("LoadFromStore without store: got %d, want 0", n)
	}
}
