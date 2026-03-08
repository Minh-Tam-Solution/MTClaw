// Package governance — Sprint 12: Spec Quality Scoring (T12-GOV-01).
// CTO Governance Audit GAP 1: enforce quality at spec creation boundary.
package governance

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Minh-Tam-Solution/MTClaw/internal/store"
)

// QualityThreshold is the minimum score for a spec to be accepted.
// CTO Decision 2: start at 70, raise to 75 after T13-GOV-05 ships.
const QualityThreshold = 70

// QualityResult holds the scoring output from EvaluateSpecQuality.
type QualityResult struct {
	Score   int      `json:"score"`   // 0-100
	Reasons []string `json:"reasons"` // failure reasons if Score < threshold
	Pass    bool     `json:"pass"`
}

// SpecResult is the structured return type for ProcessSpecOutput.
// CTO Sprint 12 Decision D2: replaces fragile string prefix convention.
type SpecResult struct {
	SpecID   string        `json:"spec_id"`  // populated on success
	Rejected bool          `json:"rejected"` // true if quality gate failed
	Quality  QualityResult `json:"quality"`  // scoring details (always populated)
}

// EvaluateSpecQuality scores a GovernanceSpec across 5 dimensions.
//
// Scoring rubric (CTO-approved, Governance Audit 2026-03-06):
//
//	Narrative completeness    25 pts
//	Acceptance criteria       25 pts
//	BDD scenarios             20 pts
//	Risk assessment           15 pts
//	Technical requirements    15 pts
//
// Threshold: score < 70 → QualityResult.Pass = false
func EvaluateSpecQuality(spec *store.GovernanceSpec) QualityResult {
	var score int
	var reasons []string

	// Dimension 1: Narrative completeness (25 pts)
	narScore, narReason := scoreNarrative(spec.Narrative)
	score += narScore
	if narReason != "" {
		reasons = append(reasons, narReason)
	}

	// Dimension 2: Acceptance criteria (25 pts)
	acScore, acReason := scoreAcceptanceCriteria(spec.AcceptanceCriteria)
	score += acScore
	if acReason != "" {
		reasons = append(reasons, acReason)
	}

	// Dimension 3: BDD scenarios (20 pts)
	bddScore, bddReason := scoreBDDScenarios(spec.BDDScenarios)
	score += bddScore
	if bddReason != "" {
		reasons = append(reasons, bddReason)
	}

	// Dimension 4: Risk assessment (15 pts)
	riskScore, riskReason := scoreRisks(spec.Risks)
	score += riskScore
	if riskReason != "" {
		reasons = append(reasons, riskReason)
	}

	// Dimension 5: Technical requirements (15 pts)
	techScore, techReason := scoreTechnicalRequirements(spec.TechnicalRequirements)
	score += techScore
	if techReason != "" {
		reasons = append(reasons, techReason)
	}

	return QualityResult{
		Score:   score,
		Reasons: reasons,
		Pass:    score >= QualityThreshold,
	}
}

// FormatRejectionMessage builds a user-facing rejection message from a QualityResult.
func FormatRejectionMessage(q QualityResult) string {
	msg := fmt.Sprintf("Quality Gate: spec scored %d/100 (minimum: %d). Issues:\n", q.Score, QualityThreshold)
	for _, r := range q.Reasons {
		msg += fmt.Sprintf("- %s\n", r)
	}
	msg += "\nPlease improve the spec and try again."
	return msg
}

// --- Dimension scorers ---

// scoreNarrative evaluates as_a + i_want + so_that fields.
// 25 pts: all 3 present + >20 chars each.
// 15 pts: 2 of 3 present + >20 chars.
// 8 pts: 1 of 3 present + >20 chars.
// 0 pts: none present.
func scoreNarrative(raw json.RawMessage) (int, string) {
	if len(raw) == 0 {
		return 0, "Narrative missing: add as_a, i_want, so_that fields (each >20 chars)"
	}

	var fields struct {
		AsA    string `json:"as_a"`
		IWant  string `json:"i_want"`
		SoThat string `json:"so_that"`
	}
	if err := json.Unmarshal(raw, &fields); err != nil {
		return 0, "Narrative: invalid JSON structure"
	}

	count := 0
	if len(strings.TrimSpace(fields.AsA)) > 20 {
		count++
	}
	if len(strings.TrimSpace(fields.IWant)) > 20 {
		count++
	}
	if len(strings.TrimSpace(fields.SoThat)) > 20 {
		count++
	}

	switch count {
	case 3:
		return 25, ""
	case 2:
		return 15, "Narrative incomplete: all 3 fields (as_a, i_want, so_that) must be >20 chars"
	case 1:
		return 8, "Narrative incomplete: only 1 of 3 fields meets minimum length"
	default:
		return 0, "Narrative empty: as_a, i_want, so_that all missing or too short (<20 chars)"
	}
}

// scoreAcceptanceCriteria evaluates acceptance criteria list.
// 25 pts: len >= 2 + all have scenario + expected_result.
// 15 pts: len == 1 with both fields.
// 10 pts: len >= 2 but missing fields.
// 0 pts: none.
func scoreAcceptanceCriteria(raw json.RawMessage) (int, string) {
	if len(raw) == 0 {
		return 0, "Acceptance criteria missing: add at least 2 criteria with scenario + expected_result"
	}

	var criteria []struct {
		Scenario       string `json:"scenario"`
		ExpectedResult string `json:"expected_result"`
	}
	if err := json.Unmarshal(raw, &criteria); err != nil {
		return 0, "Acceptance criteria: invalid JSON structure (expected array)"
	}

	if len(criteria) == 0 {
		return 0, "Acceptance criteria empty: add at least 2 criteria"
	}

	allComplete := true
	for _, c := range criteria {
		if strings.TrimSpace(c.Scenario) == "" || strings.TrimSpace(c.ExpectedResult) == "" {
			allComplete = false
			break
		}
	}

	switch {
	case len(criteria) >= 2 && allComplete:
		return 25, ""
	case len(criteria) == 1 && allComplete:
		return 15, "Acceptance criteria: need at least 2 (found 1)"
	case len(criteria) >= 2:
		return 10, "Acceptance criteria: some missing scenario or expected_result fields"
	default:
		return 5, "Acceptance criteria: 1 criterion with incomplete fields"
	}
}

// scoreBDDScenarios evaluates BDD scenarios.
// 20 pts: len >= 1 + all have given/when/then.
// 10 pts: len >= 1 but missing fields.
// 0 pts: none.
func scoreBDDScenarios(raw json.RawMessage) (int, string) {
	if len(raw) == 0 {
		return 0, "BDD scenarios missing: add at least 1 scenario with given/when/then"
	}

	var scenarios []struct {
		Given string `json:"given"`
		When  string `json:"when"`
		Then  string `json:"then"`
	}
	if err := json.Unmarshal(raw, &scenarios); err != nil {
		return 0, "BDD scenarios: invalid JSON structure (expected array)"
	}

	if len(scenarios) == 0 {
		return 0, "BDD scenarios empty: add at least 1 scenario"
	}

	allComplete := true
	for _, s := range scenarios {
		if strings.TrimSpace(s.Given) == "" || strings.TrimSpace(s.When) == "" || strings.TrimSpace(s.Then) == "" {
			allComplete = false
			break
		}
	}

	if allComplete {
		return 20, ""
	}
	return 10, "BDD scenarios: some missing given, when, or then fields"
}

// scoreRisks evaluates risk assessment.
// 15 pts: len >= 1 + all have description + mitigation.
// 8 pts: len >= 1 but missing fields.
// 0 pts: none.
func scoreRisks(raw json.RawMessage) (int, string) {
	if len(raw) == 0 {
		return 0, "Risk assessment missing: add at least 1 risk with description + mitigation"
	}

	var risks []struct {
		Description string `json:"description"`
		Mitigation  string `json:"mitigation"`
	}
	if err := json.Unmarshal(raw, &risks); err != nil {
		return 0, "Risk assessment: invalid JSON structure (expected array)"
	}

	if len(risks) == 0 {
		return 0, "Risk assessment empty: add at least 1 risk"
	}

	allComplete := true
	for _, r := range risks {
		if strings.TrimSpace(r.Description) == "" || strings.TrimSpace(r.Mitigation) == "" {
			allComplete = false
			break
		}
	}

	if allComplete {
		return 15, ""
	}
	return 8, "Risk assessment: some missing description or mitigation fields"
}

// scoreTechnicalRequirements evaluates technical requirements.
// 15 pts: non-null + >50 chars.
// 8 pts: non-null + >0 chars but <=50.
// 0 pts: null or empty.
func scoreTechnicalRequirements(raw json.RawMessage) (int, string) {
	if len(raw) == 0 {
		return 0, "Technical requirements missing: add requirements text (>50 chars)"
	}

	// Technical requirements can be a string or a structured object.
	// Try string first, then fall back to measuring raw JSON length.
	var text string
	if err := json.Unmarshal(raw, &text); err != nil {
		// Not a simple string — use raw length as proxy (minus JSON overhead).
		trimmed := strings.TrimSpace(string(raw))
		if len(trimmed) > 50 {
			return 15, ""
		}
		if len(trimmed) > 0 {
			return 8, "Technical requirements too brief: expand to >50 chars"
		}
		return 0, "Technical requirements empty"
	}

	text = strings.TrimSpace(text)
	if len(text) > 50 {
		return 15, ""
	}
	if len(text) > 0 {
		return 8, "Technical requirements too brief: expand to >50 chars"
	}
	return 0, "Technical requirements empty"
}
