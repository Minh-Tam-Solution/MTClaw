package governance

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/nextlevelbuilder/goclaw/internal/store"
)

// mockSpecStore implements store.SpecStore for design gate testing.
// Only ListSpecs is used by DesignFirstGate.
type mockSpecStore struct {
	specs []store.GovernanceSpec
}

func (m *mockSpecStore) CreateSpec(ctx context.Context, spec *store.GovernanceSpec) error {
	return nil
}
func (m *mockSpecStore) GetSpec(ctx context.Context, specID string) (*store.GovernanceSpec, error) {
	return nil, nil
}
func (m *mockSpecStore) ListSpecs(ctx context.Context, opts store.SpecListOpts) ([]store.GovernanceSpec, error) {
	var result []store.GovernanceSpec
	for _, s := range m.specs {
		if opts.Status != "" && s.Status != opts.Status {
			continue
		}
		result = append(result, s)
		if opts.Limit > 0 && len(result) >= opts.Limit {
			break
		}
	}
	return result, nil
}
func (m *mockSpecStore) CountSpecs(ctx context.Context, opts store.SpecListOpts) (int, error) {
	return len(m.specs), nil
}
func (m *mockSpecStore) UpdateSpecStatus(ctx context.Context, specID string, status string) error {
	return nil
}
func (m *mockSpecStore) NextSpecID(ctx context.Context, year int) (string, error) {
	return "SPEC-2026-0001", nil
}

func emptyStore() *mockSpecStore {
	return &mockSpecStore{}
}

func storeWithApprovedSpec() *mockSpecStore {
	return &mockSpecStore{
		specs: []store.GovernanceSpec{
			{
				ID:     uuid.New(),
				SpecID: "SPEC-2026-0001",
				Title:  "Auth system spec",
				Status: store.SpecStatusApproved,
				CreatedAt: time.Now(),
			},
		},
	}
}

func TestDesignGate_NonCoder_Passes(t *testing.T) {
	pass, reason := DesignFirstGate(context.Background(), "pm", "implement user auth", emptyStore())
	if !pass {
		t.Errorf("expected pass for non-coder agent, got blocked: %s", reason)
	}
}

func TestDesignGate_CoderAdHocQuestion_How_Passes(t *testing.T) {
	pass, _ := DesignFirstGate(context.Background(), "coder", "how do I implement auth?", emptyStore())
	if !pass {
		t.Error("expected pass for 'how' question")
	}
}

func TestDesignGate_CoderExplain_Passes(t *testing.T) {
	pass, _ := DesignFirstGate(context.Background(), "coder", "explain the routing logic", emptyStore())
	if !pass {
		t.Error("expected pass for 'explain' question")
	}
}

func TestDesignGate_CoderDebug_Passes(t *testing.T) {
	pass, _ := DesignFirstGate(context.Background(), "coder", "debug this error in auth module", emptyStore())
	if !pass {
		t.Error("expected pass for 'debug' question")
	}
}

func TestDesignGate_CoderQuestionMark_Passes(t *testing.T) {
	pass, _ := DesignFirstGate(context.Background(), "coder", "what does this function do?", emptyStore())
	if !pass {
		t.Error("expected pass for question-mark content")
	}
}

func TestDesignGate_CoderCan_Passes(t *testing.T) {
	pass, _ := DesignFirstGate(context.Background(), "coder", "can you explain the error handling?", emptyStore())
	if !pass {
		t.Error("expected pass for 'can' prefix (CTO Decision D3)")
	}
}

func TestDesignGate_CoderShould_Passes(t *testing.T) {
	pass, _ := DesignFirstGate(context.Background(), "coder", "should we use context propagation here?", emptyStore())
	if !pass {
		t.Error("expected pass for 'should' prefix (CTO Decision D3)")
	}
}

func TestDesignGate_CoderIs_Passes(t *testing.T) {
	pass, _ := DesignFirstGate(context.Background(), "coder", "is it possible to use generics here?", emptyStore())
	if !pass {
		t.Error("expected pass for 'is' prefix (CTO Decision D3)")
	}
}

func TestDesignGate_CoderCodeTask_NoSpec_Blocks(t *testing.T) {
	pass, reason := DesignFirstGate(context.Background(), "coder", "implement user authentication", emptyStore())
	if pass {
		t.Error("expected block for code task with no approved spec")
	}
	if !containsSubstring(reason, "Design-First Gate") {
		t.Errorf("expected 'Design-First Gate' in reason, got: %s", reason)
	}
}

func TestDesignGate_CoderBuildTask_NoSpec_Blocks(t *testing.T) {
	pass, _ := DesignFirstGate(context.Background(), "coder", "build the dashboard component", emptyStore())
	if pass {
		t.Error("expected block for 'build' task with no approved spec")
	}
}

func TestDesignGate_CoderCodeTask_WithSpec_Passes(t *testing.T) {
	pass, reason := DesignFirstGate(context.Background(), "coder", "implement user authentication", storeWithApprovedSpec())
	if !pass {
		t.Errorf("expected pass with approved spec, got blocked: %s", reason)
	}
}

func TestDesignGate_NilSpecStore_Passes(t *testing.T) {
	pass, _ := DesignFirstGate(context.Background(), "coder", "implement something", nil)
	if !pass {
		t.Error("expected pass with nil specStore (graceful degradation)")
	}
}

func TestDesignGate_EmptyContent_Passes(t *testing.T) {
	pass, _ := DesignFirstGate(context.Background(), "coder", "", emptyStore())
	if !pass {
		t.Error("expected pass for empty content")
	}
}

func TestDesignGate_WhitespaceContent_Passes(t *testing.T) {
	pass, _ := DesignFirstGate(context.Background(), "coder", "   ", emptyStore())
	if !pass {
		t.Error("expected pass for whitespace-only content")
	}
}

func TestIsAdHocQuestion(t *testing.T) {
	cases := []struct {
		input    string
		expected bool
	}{
		{"how do I implement auth?", true},
		{"explain the routing logic", true},
		{"debug this error", true},
		{"what is the purpose of this?", true},
		{"why does this fail?", true},
		{"where is the config?", true},
		{"help me understand this", true},
		{"can you review this?", true},
		{"should we use generics?", true},
		{"is this correct?", true},
		{"does this work?", true},
		{"could you check this?", true},
		{"something ending with question mark?", true},
		{"implement user auth", false},
		{"build the dashboard", false},
		{"create a new endpoint", false},
		{"refactor the auth module", false},
		{"", false},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got := isAdHocQuestion(tc.input)
			if got != tc.expected {
				t.Errorf("isAdHocQuestion(%q) = %v, want %v", tc.input, got, tc.expected)
			}
		})
	}
}
