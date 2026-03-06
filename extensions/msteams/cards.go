package msteams

import (
	"encoding/json"
	"strings"
)

// SpecCard builds an Adaptive Card for a governance spec output.
// Used by the spec processor when channel == "msteams".
// Returns the card JSON suitable for use as OutboundMessage.Content with Format="adaptive_card".
func SpecCard(specID, title, status string, scenarios []string) json.RawMessage {
	statusColor := "default"
	switch strings.ToUpper(status) {
	case "APPROVED", "PASS":
		statusColor = "good"
	case "REJECTED", "FAIL", "BLOCKED":
		statusColor = "attention"
	case "PROPOSED", "REVIEW":
		statusColor = "warning"
	}

	scenarioFacts := make([]map[string]string, 0, len(scenarios))
	for _, s := range scenarios {
		scenarioFacts = append(scenarioFacts, map[string]string{
			"title": "•",
			"value": s,
		})
	}

	card := map[string]interface{}{
		"$schema": "http://adaptivecards.io/schemas/adaptive-card.json",
		"type":    "AdaptiveCard",
		"version": "1.3",
		"body": []interface{}{
			map[string]interface{}{
				"type":   "TextBlock",
				"text":   "📋 Governance Spec",
				"weight": "Bolder",
				"size":   "Medium",
			},
			map[string]interface{}{
				"type": "FactSet",
				"facts": []map[string]string{
					{"title": "Spec ID", "value": specID},
					{"title": "Title", "value": title},
					{"title": "Status", "value": strings.ToUpper(status)},
				},
			},
			buildStatusBadge(strings.ToUpper(status), statusColor),
			buildScenariosBlock(scenarioFacts),
		},
	}

	data, _ := json.Marshal(card)
	return json.RawMessage(data)
}

// PRReviewCard builds an Adaptive Card for a PR Gate evaluation result.
// Used by the PR gate processor when channel == "msteams".
// Returns the card JSON suitable for use as OutboundMessage.Content with Format="adaptive_card".
func PRReviewCard(prURL, verdict string, blockRules, warnRules []string) json.RawMessage {
	verdictUpper := strings.ToUpper(verdict)
	verdictColor := "default"
	verdictIcon := "ℹ️"
	switch verdictUpper {
	case "PASS", "APPROVED":
		verdictColor = "good"
		verdictIcon = "✅"
	case "BLOCK", "BLOCKED", "FAIL", "REJECTED":
		verdictColor = "attention"
		verdictIcon = "🚫"
	case "WARN", "WARNING":
		verdictColor = "warning"
		verdictIcon = "⚠️"
	}

	blockFacts := rulesToFacts(blockRules, "🚫")
	warnFacts := rulesToFacts(warnRules, "⚠️")

	body := []interface{}{
		map[string]interface{}{
			"type":   "TextBlock",
			"text":   verdictIcon + " PR Gate Review",
			"weight": "Bolder",
			"size":   "Medium",
		},
		map[string]interface{}{
			"type": "FactSet",
			"facts": []map[string]string{
				{"title": "PR URL", "value": prURL},
				{"title": "Verdict", "value": verdictUpper},
			},
		},
		buildStatusBadge(verdictUpper, verdictColor),
	}

	if len(blockFacts) > 0 {
		body = append(body,
			map[string]interface{}{
				"type":  "TextBlock",
				"text":  "**Block Rules Triggered:**",
				"wrap":  true,
				"color": "attention",
			},
			map[string]interface{}{
				"type":  "FactSet",
				"facts": blockFacts,
			},
		)
	}

	if len(warnFacts) > 0 {
		body = append(body,
			map[string]interface{}{
				"type":  "TextBlock",
				"text":  "**Warnings:**",
				"wrap":  true,
				"color": "warning",
			},
			map[string]interface{}{
				"type":  "FactSet",
				"facts": warnFacts,
			},
		)
	}

	card := map[string]interface{}{
		"$schema": "http://adaptivecards.io/schemas/adaptive-card.json",
		"type":    "AdaptiveCard",
		"version": "1.3",
		"body":    body,
	}

	data, _ := json.Marshal(card)
	return json.RawMessage(data)
}

// buildStatusBadge returns an Adaptive Card TextBlock styled as a status badge.
func buildStatusBadge(text, color string) map[string]interface{} {
	return map[string]interface{}{
		"type":   "TextBlock",
		"text":   text,
		"color":  color,
		"weight": "Bolder",
	}
}

// buildScenariosBlock returns an Adaptive Card FactSet of BDD scenarios (or empty TextBlock).
func buildScenariosBlock(facts []map[string]string) interface{} {
	if len(facts) == 0 {
		return map[string]interface{}{
			"type":  "TextBlock",
			"text":  "_No BDD scenarios defined._",
			"isSubtle": true,
		}
	}
	return map[string]interface{}{
		"type":  "FactSet",
		"facts": facts,
	}
}

// rulesToFacts converts a list of rule names into Adaptive Card FactSet entries.
func rulesToFacts(rules []string, prefix string) []map[string]string {
	facts := make([]map[string]string, 0, len(rules))
	for _, r := range rules {
		facts = append(facts, map[string]string{
			"title": prefix,
			"value": r,
		})
	}
	return facts
}
