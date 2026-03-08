package routing

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"

	"github.com/Minh-Tam-Solution/MTClaw/internal/store"
)

// BuildTeamContext generates the team context prompt for team-routed messages.
// Returns empty string if no team routing is active.
// CTO-13 FIX: Uses cached teamID/teamName to avoid redundant ListTeams() call.
func BuildTeamContext(ctx context.Context, teamName string, teamID *uuid.UUID, teamStore store.TeamStore) string {
	if teamName == "" || teamStore == nil || teamID == nil {
		return ""
	}

	// Look up full team name from mention map.
	fullName, ok := TeamMentionMap[teamName]
	if !ok {
		fullName = teamName
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("## Team Context\nYou are responding as the **lead** of the **%s** team.\n", fullName))
	sb.WriteString("Team members available for delegation:\n")

	members, _ := teamStore.ListMembers(ctx, *teamID)
	for _, m := range members {
		sb.WriteString(fmt.Sprintf("- @%s\n", m.AgentKey))
	}

	return sb.String()
}
