package bootstrap

import (
	"fmt"
	"strings"
)

// SOULRoleInfo describes a SOUL role for delegation markdown generation.
// Callers scan SOUL files via claudecode.KnownRoles()/LoadSOUL() and pass
// the results here — keeping bootstrap free of claudecode imports (CTO-2).
type SOULRoleInfo struct {
	Role     string // e.g. "coder", "pm"
	Title    string // first heading from SOUL body
	Category string // frontmatter category: advisor/executor/router/business/operations
}

// BuildSOULDelegationMD generates a DELEGATION.md listing all available SOULs
// grouped by category. This is mode-agnostic and serves as:
//   - Standalone: primary delegation source (no DB)
//   - Managed: fallback when agent_links is incomplete
//
// activeAgents maps agentID→displayName for currently active/pre-loaded agents.
// Pass nil if not applicable (e.g. managed-mode fallback).
func BuildSOULDelegationMD(roles []SOULRoleInfo, activeAgents map[string]string) string {
	if len(roles) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("# Available Agents\n\n")
	sb.WriteString("Use @agent_name to route messages to a specific agent (e.g. @pm, @coder, @reviewer).\n")
	sb.WriteString("The agent list below is complete and authoritative — answer questions about available agents directly from it.\n\n")

	// Group by category
	type catGroup struct {
		name  string
		key   string
		roles []SOULRoleInfo
	}
	categories := []catGroup{
		{name: "Advisors (C-Level)", key: "advisor"},
		{name: "Executors (SDLC Roles)", key: "executor"},
		{name: "Router", key: "router"},
		{name: "Business", key: "business"},
		{name: "Operations", key: "operations"},
	}

	categorized := make(map[string]bool)
	for i := range categories {
		for _, r := range roles {
			cat := r.Category
			if cat == "" {
				cat = "executor"
			}
			if cat == categories[i].key {
				categories[i].roles = append(categories[i].roles, r)
				categorized[r.Role] = true
			}
		}
	}

	// Catch uncategorized
	var uncategorized []SOULRoleInfo
	for _, r := range roles {
		if !categorized[r.Role] {
			uncategorized = append(uncategorized, r)
		}
	}
	if len(uncategorized) > 0 {
		categories = append(categories, catGroup{name: "Other", key: "other", roles: uncategorized})
	}

	for _, cat := range categories {
		if len(cat.roles) == 0 {
			continue
		}
		sb.WriteString(fmt.Sprintf("## %s\n\n", cat.name))
		for _, r := range cat.roles {
			title := r.Title
			if title == "" {
				title = r.Role
			}
			sb.WriteString(fmt.Sprintf("- **@%s** — %s\n", r.Role, title))
		}
		sb.WriteString("\n")
	}

	// Note which agents are currently active (standalone mode)
	if len(activeAgents) > 0 {
		sb.WriteString("## Currently Active Agents\n\n")
		sb.WriteString("These agents are pre-loaded and respond immediately:\n")
		for agentID, name := range activeAgents {
			if name == "" {
				name = agentID
			}
			sb.WriteString(fmt.Sprintf("- **@%s** (%s)\n", agentID, name))
		}
		sb.WriteString("\nOther agents listed above can be routed to via @mention.\n")
	}

	return sb.String()
}
