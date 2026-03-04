// Package routing extracts mention and team routing logic from gateway_consumer.
// Sprint 7: CTO-14 refactoring — better testability + maintainability.
package routing

import (
	"context"
	"log/slog"
	"strings"

	"github.com/google/uuid"

	"github.com/nextlevelbuilder/goclaw/internal/agent"
	"github.com/nextlevelbuilder/goclaw/internal/store"
)

// TeamMentionMap maps short mention keys to full team names in DB.
// CTO-8: DB names are "SDLC Engineering" but users type @engineering.
// Hardcoded for Sprint 6 (3 static teams). Sprint 9+: add mention_key column to teams table.
var TeamMentionMap = map[string]string{
	"engineering": "SDLC Engineering",
	"business":    "Business Operations",
	"advisory":    "Advisory Board",
}

// MentionResult contains the resolved routing from an @mention.
type MentionResult struct {
	AgentKey  string
	TeamName  string
	TeamID    *uuid.UUID
	IsTeam    bool
	Content   string // message content with @mention stripped
}

// ResolveMention resolves @mention routing from message content.
// Returns a MentionResult with the resolved agent key and team info.
// Resolution order: agent-first, then team-second.
func ResolveMention(ctx context.Context, content string, agents *agent.Router, teamStore store.TeamStore, channel string) MentionResult {
	result := MentionResult{Content: content}

	if !strings.HasPrefix(content, "@") {
		return result
	}

	parts := strings.SplitN(content, " ", 2)
	candidate := strings.TrimPrefix(parts[0], "@")
	candidate = strings.ToLower(candidate)

	// Agent-first resolution (Sprint 4)
	if _, err := agents.Get(candidate); err == nil {
		result.AgentKey = candidate
		if len(parts) > 1 {
			result.Content = strings.TrimSpace(parts[1])
		}
		slog.Info("routing: @mention agent resolved",
			"mention", candidate, "channel", channel)
		return result
	}

	// Team-second resolution (Sprint 6, CTO-8)
	if teamStore != nil {
		if fullName, ok := TeamMentionMap[candidate]; ok {
			teams, _ := teamStore.ListTeams(ctx)
			for _, t := range teams {
				if t.Name == fullName {
					result.AgentKey = t.LeadAgentKey
					result.TeamName = candidate
					teamID := t.ID
					result.TeamID = &teamID
					result.IsTeam = true
					if len(parts) > 1 {
						result.Content = strings.TrimSpace(parts[1])
					}
					slog.Info("routing: @mention team resolved",
						"team", t.Name, "mention", candidate,
						"lead", t.LeadAgentKey, "channel", channel)
					return result
				}
			}
		}
	}

	return result
}
