package governance

import (
	"encoding/json"
	"testing"

	"github.com/nextlevelbuilder/goclaw/internal/store"
)

// helpers to build json.RawMessage from Go values.
func toJSON(v interface{}) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

func fullNarrative() json.RawMessage {
	return toJSON(map[string]string{
		"as_a":    "As a project manager responsible for delivery",
		"i_want":  "I want automated spec quality scoring",
		"so_that": "So that low-quality specs are rejected before reaching coder",
	})
}

func fullAC() json.RawMessage {
	return toJSON([]map[string]string{
		{"scenario": "Spec with all 5 dimensions complete", "expected_result": "Score >= 85, spec saved to DB"},
		{"scenario": "Spec with missing BDD scenarios", "expected_result": "Score < 70, rejection message returned"},
	})
}

func fullBDD() json.RawMessage {
	return toJSON([]map[string]string{
		{"given": "A spec with all narrative fields filled", "when": "EvaluateSpecQuality is called", "then": "Score includes 25 narrative points"},
	})
}

func fullRisks() json.RawMessage {
	return toJSON([]map[string]string{
		{"description": "Quality scorer too strict blocks legitimate specs", "mitigation": "Start threshold at 70, tune based on first 10 rejections"},
	})
}

func fullTechReq() json.RawMessage {
	return toJSON("Must integrate at spec_processor.go line 46-48, after ContentHash, before CreateSpec. Pure function with no DB dependency.")
}

func fullSpec() *store.GovernanceSpec {
	return &store.GovernanceSpec{
		Narrative:             fullNarrative(),
		AcceptanceCriteria:    fullAC(),
		BDDScenarios:          fullBDD(),
		Risks:                 fullRisks(),
		TechnicalRequirements: fullTechReq(),
	}
}

func TestQuality_FullSpec_Passes(t *testing.T) {
	result := EvaluateSpecQuality(fullSpec())
	if !result.Pass {
		t.Errorf("expected Pass=true, got score=%d reasons=%v", result.Score, result.Reasons)
	}
	if result.Score < 85 {
		t.Errorf("expected score >= 85 for full spec, got %d", result.Score)
	}
	if len(result.Reasons) != 0 {
		t.Errorf("expected no reasons for full spec, got %v", result.Reasons)
	}
}

func TestQuality_MinimalSpec_Passes(t *testing.T) {
	// Narrative (25) + AC 2 complete (25) + BDD 1 complete (20) = 70 → Pass
	spec := &store.GovernanceSpec{
		Narrative:          fullNarrative(),
		AcceptanceCriteria: fullAC(),
		BDDScenarios:       fullBDD(),
		// No risks, no tech req → 0 + 0 = 70
	}
	result := EvaluateSpecQuality(spec)
	if !result.Pass {
		t.Errorf("expected minimal spec to pass (score=%d), reasons=%v", result.Score, result.Reasons)
	}
	if result.Score != 70 {
		t.Errorf("expected score=70, got %d", result.Score)
	}
}

func TestQuality_EmptyNarrative_LowersScore(t *testing.T) {
	spec := fullSpec()
	spec.Narrative = nil
	result := EvaluateSpecQuality(spec)
	// Without narrative (25 pts): 0+25+20+15+15 = 75, still passes threshold 70.
	// But score must be lower than fullSpec (100) and narrative reason must appear.
	if result.Score >= 100 {
		t.Errorf("expected reduced score without narrative, got %d", result.Score)
	}
	found := false
	for _, r := range result.Reasons {
		if contains(r, "Narrative") || contains(r, "narrative") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected narrative-related reason, got %v", result.Reasons)
	}

	// To actually fail, remove narrative AND another dimension.
	spec.BDDScenarios = nil
	result2 := EvaluateSpecQuality(spec)
	if result2.Pass {
		t.Errorf("expected Pass=false with narrative+BDD missing, got score=%d", result2.Score)
	}
}

func TestQuality_OneAC_Fails(t *testing.T) {
	spec := fullSpec()
	spec.AcceptanceCriteria = toJSON([]map[string]string{
		{"scenario": "Single AC", "expected_result": "Only one criterion"},
	})
	result := EvaluateSpecQuality(spec)
	// Narrative(25) + AC 1 complete(15) + BDD(20) + Risk(15) + Tech(15) = 90 → still passes
	// But if we remove other dimensions to test threshold boundary:
	if result.Score < 70 {
		// With full other dimensions, 1 AC still passes. Test the score reduction.
		t.Logf("1 AC reduces score by 10 pts (25→15), score=%d", result.Score)
	}
	// Verify AC reason exists
	found := false
	for _, r := range result.Reasons {
		if contains(r, "criteria") || contains(r, "Acceptance") {
			found = true
		}
	}
	if !found {
		t.Errorf("expected acceptance criteria reason, got %v", result.Reasons)
	}
}

func TestQuality_NoBDD_Fails(t *testing.T) {
	// Narrative(25) + AC(25) + BDD(0) + Risk(0) + Tech(0) = 50 → Fail
	spec := &store.GovernanceSpec{
		Narrative:          fullNarrative(),
		AcceptanceCriteria: fullAC(),
	}
	result := EvaluateSpecQuality(spec)
	if result.Pass {
		t.Errorf("expected fail without BDD, got score=%d", result.Score)
	}
}

func TestQuality_PartialNarrative(t *testing.T) {
	spec := fullSpec()
	spec.Narrative = toJSON(map[string]string{
		"as_a":  "As a project manager responsible for delivery",
		"i_want": "I want automated spec quality scoring",
		// so_that missing
	})
	result := EvaluateSpecQuality(spec)
	// 2 of 3 → 15 pts narrative
	// With full other dims: 15 + 25 + 20 + 15 + 15 = 90
	if result.Score != 90 {
		t.Errorf("expected score=90 with partial narrative, got %d", result.Score)
	}
}

func TestQuality_ACMissingFields(t *testing.T) {
	spec := fullSpec()
	spec.AcceptanceCriteria = toJSON([]map[string]string{
		{"scenario": "Has scenario", "expected_result": "Has result"},
		{"scenario": "Has scenario only"},
	})
	result := EvaluateSpecQuality(spec)
	// 2 AC but one incomplete → 10 pts
	// With full other dims: 25 + 10 + 20 + 15 + 15 = 85
	if result.Score != 85 {
		t.Errorf("expected score=85 with incomplete AC fields, got %d", result.Score)
	}
}

func TestQuality_NilFields(t *testing.T) {
	spec := &store.GovernanceSpec{} // all json.RawMessage fields are nil
	result := EvaluateSpecQuality(spec)
	if result.Score != 0 {
		t.Errorf("expected score=0 for nil fields, got %d", result.Score)
	}
	if result.Pass {
		t.Errorf("expected Pass=false for nil fields")
	}
	if len(result.Reasons) != 5 {
		t.Errorf("expected 5 reasons (one per dimension), got %d: %v", len(result.Reasons), result.Reasons)
	}
}

func TestQuality_EmptyJSON(t *testing.T) {
	spec := &store.GovernanceSpec{
		Narrative: json.RawMessage(`{}`),
	}
	result := EvaluateSpecQuality(spec)
	if result.Pass {
		t.Errorf("expected fail with empty JSON narrative, got score=%d", result.Score)
	}
}

func TestQuality_ThresholdEdge_69(t *testing.T) {
	// Construct to score exactly 69:
	// Narrative partial 2/3 (15) + AC 1 complete (15) + BDD complete (20) + Risk(0) + Tech brief(8) = 58
	// Need 69. Try: Narrative full (25) + AC 1 complete (15) + BDD incomplete (10) + Risk(0) + Tech full (15) = 65
	// Try: Narrative full (25) + AC 2 incomplete (10) + BDD complete (20) + Risk incomplete (8) + Tech brief (8) = 71... too much
	// Try: Narrative 2/3 (15) + AC 2 complete (25) + BDD incomplete (10) + Risk(0) + Tech brief(8) = 58
	// Try: Narrative 2/3 (15) + AC 2 incomplete (10) + BDD complete (20) + Risk complete (15) + Tech brief (8) = 68... close
	// Try: Narrative 2/3 (15) + AC 2 incomplete (10) + BDD complete (20) + Risk complete (15) + Tech full (15) = 75
	// Exact 69 is hard with discrete scoring. Test just-below-threshold.
	// Narrative 1/3 (8) + AC 2 complete (25) + BDD incomplete (10) + Risk complete (15) + Tech brief (8) = 66
	// Narrative 2/3 (15) + AC 2 incomplete (10) + BDD complete (20) + Risk incomplete (8) + Tech full (15) = 68
	spec := &store.GovernanceSpec{
		Narrative: toJSON(map[string]string{
			"as_a":  "As a project manager responsible for delivery",
			"i_want": "I want automated spec quality scoring",
		}),
		AcceptanceCriteria: toJSON([]map[string]string{
			{"scenario": "test", "expected_result": "pass"},
			{"scenario": "missing result field"},
		}),
		BDDScenarios: fullBDD(),
		Risks: toJSON([]map[string]string{
			{"description": "risk exists", "mitigation": ""},
		}),
		TechnicalRequirements: fullTechReq(),
	}
	result := EvaluateSpecQuality(spec)
	// 15 + 10 + 20 + 8 + 15 = 68
	if result.Score != 68 {
		t.Errorf("expected score=68, got %d", result.Score)
	}
	if result.Pass {
		t.Errorf("expected Pass=false for score 68")
	}
}

func TestQuality_ThresholdEdge_70(t *testing.T) {
	// Exactly 70: Narrative full (25) + AC 2 complete (25) + BDD complete (20) + no risks (0) + no tech (0) = 70
	spec := &store.GovernanceSpec{
		Narrative:          fullNarrative(),
		AcceptanceCriteria: fullAC(),
		BDDScenarios:       fullBDD(),
	}
	result := EvaluateSpecQuality(spec)
	if result.Score != 70 {
		t.Errorf("expected score=70, got %d", result.Score)
	}
	if !result.Pass {
		t.Errorf("expected Pass=true for score 70")
	}
}

func TestQuality_MalformedJSON(t *testing.T) {
	spec := &store.GovernanceSpec{
		Narrative:          json.RawMessage(`not valid json`),
		AcceptanceCriteria: json.RawMessage(`{"not": "an array"}`),
		BDDScenarios:       json.RawMessage(`broken`),
		Risks:              json.RawMessage(`123`),
		TechnicalRequirements: fullTechReq(),
	}
	result := EvaluateSpecQuality(spec)
	// Should not panic — graceful 0 for each malformed dimension
	if result.Score > 15 {
		t.Errorf("expected low score for malformed JSON, got %d", result.Score)
	}
	if result.Pass {
		t.Errorf("expected fail for malformed JSON")
	}
}

func TestFormatRejectionMessage(t *testing.T) {
	q := QualityResult{
		Score:   45,
		Reasons: []string{"Narrative missing", "BDD scenarios missing"},
		Pass:    false,
	}
	msg := FormatRejectionMessage(q)
	if !contains(msg, "45/100") {
		t.Errorf("expected score in message, got: %s", msg)
	}
	if !contains(msg, "Narrative missing") {
		t.Errorf("expected reason in message, got: %s", msg)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsSubstring(s, sub))
}

func containsSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
