package governance

import (
	"testing"
)

func TestParseSpecJSON_FencedBlock(t *testing.T) {
	output := "Here is the spec:\n\n```json\n{\"spec_version\":\"1.0.0\",\"title\":\"Login Feature\",\"narrative\":{\"as_a\":\"user\"},\"acceptance_criteria\":[{\"scenario\":\"Happy path\"}],\"priority\":\"P0\",\"estimated_effort\":\"M\"}\n```\n\nPlease review."
	spec, err := ParseSpecJSON(output)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if spec.Title != "Login Feature" {
		t.Errorf("expected title 'Login Feature', got %q", spec.Title)
	}
	if spec.Priority != "P0" {
		t.Errorf("expected priority P0, got %q", spec.Priority)
	}
}

func TestParseSpecJSON_RawJSON(t *testing.T) {
	output := "Generated spec: {\"spec_version\":\"1.0.0\",\"title\":\"Payment API\",\"narrative\":{},\"acceptance_criteria\":[]}"
	spec, err := ParseSpecJSON(output)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if spec.Title != "Payment API" {
		t.Errorf("expected title 'Payment API', got %q", spec.Title)
	}
	// Should default priority and effort
	if spec.Priority != "P1" {
		t.Errorf("expected default priority P1, got %q", spec.Priority)
	}
	if spec.EstimatedEffort != "M" {
		t.Errorf("expected default effort M, got %q", spec.EstimatedEffort)
	}
}

func TestParseSpecJSON_NoJSON(t *testing.T) {
	output := "Sorry, I couldn't generate a spec from that."
	_, err := ParseSpecJSON(output)
	if err == nil {
		t.Error("expected error for no-JSON output, got nil")
	}
}

func TestParseSpecJSON_MissingRequiredFields(t *testing.T) {
	output := `{"spec_version":"","title":"","narrative":{}}`
	_, err := ParseSpecJSON(output)
	if err == nil {
		t.Error("expected error for missing required fields, got nil")
	}
}

func TestExtractJSONBlock_Fenced(t *testing.T) {
	text := "prefix\n```json\n{\"key\":\"value\"}\n```\nsuffix"
	got := extractJSONBlock(text)
	if got != `{"key":"value"}` {
		t.Errorf("expected JSON block, got %q", got)
	}
}

func TestExtractJSONBlock_Bare(t *testing.T) {
	text := "result: {\"a\":1}"
	got := extractJSONBlock(text)
	if got != `{"a":1}` {
		t.Errorf("expected JSON block, got %q", got)
	}
}

func TestExtractJSONBlock_Nested(t *testing.T) {
	text := `{"outer":{"inner":"value"}}`
	got := extractJSONBlock(text)
	if got != `{"outer":{"inner":"value"}}` {
		t.Errorf("expected full nested JSON, got %q", got)
	}
}

func TestSha256Hex(t *testing.T) {
	hash := sha256Hex("hello")
	if len(hash) != 64 {
		t.Errorf("expected 64-char hex string, got %d chars", len(hash))
	}
	// Same input should produce same hash
	if sha256Hex("hello") != hash {
		t.Error("same input should produce same hash")
	}
	if sha256Hex("world") == hash {
		t.Error("different input should produce different hash")
	}
}
