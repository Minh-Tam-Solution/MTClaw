package bootstrap

import (
	"strings"
	"testing"
)

func TestBuildSOULDelegationMD_Empty(t *testing.T) {
	result := BuildSOULDelegationMD(nil, nil)
	if result != "" {
		t.Errorf("expected empty string for nil roles, got %q", result)
	}
}

func TestBuildSOULDelegationMD_GroupsByCategory(t *testing.T) {
	roles := []SOULRoleInfo{
		{Role: "cto", Title: "Chief Technology Officer", Category: "advisor"},
		{Role: "coder", Title: "Software Coder", Category: "executor"},
		{Role: "pm", Title: "Product Manager", Category: "executor"},
		{Role: "assistant", Title: "Router Agent", Category: "router"},
		{Role: "sales", Title: "Sales Lead", Category: "business"},
		{Role: "itadmin", Title: "IT Admin", Category: "operations"},
	}

	md := BuildSOULDelegationMD(roles, nil)

	// Verify header
	if !strings.Contains(md, "# Available Agents") {
		t.Error("missing header")
	}

	// Verify category sections exist
	for _, section := range []string{
		"## Advisors (C-Level)",
		"## Executors (SDLC Roles)",
		"## Router",
		"## Business",
		"## Operations",
	} {
		if !strings.Contains(md, section) {
			t.Errorf("missing section: %s", section)
		}
	}

	// Verify roles appear in correct sections
	advisorIdx := strings.Index(md, "## Advisors (C-Level)")
	executorIdx := strings.Index(md, "## Executors (SDLC Roles)")
	ctoIdx := strings.Index(md, "**@cto**")
	coderIdx := strings.Index(md, "**@coder**")

	if ctoIdx < advisorIdx || ctoIdx > executorIdx {
		t.Error("@cto should appear in Advisors section")
	}
	if coderIdx < executorIdx {
		t.Error("@coder should appear after Executors heading")
	}
}

func TestBuildSOULDelegationMD_DefaultCategory(t *testing.T) {
	// Roles with empty category default to "executor"
	roles := []SOULRoleInfo{
		{Role: "newagent", Title: "New Agent", Category: ""},
	}

	md := BuildSOULDelegationMD(roles, nil)
	if !strings.Contains(md, "## Executors (SDLC Roles)") {
		t.Error("empty category should default to executor section")
	}
	if !strings.Contains(md, "**@newagent**") {
		t.Error("missing @newagent in output")
	}
}

func TestBuildSOULDelegationMD_UncategorizedRoles(t *testing.T) {
	roles := []SOULRoleInfo{
		{Role: "custom", Title: "Custom Agent", Category: "exotic"},
	}

	md := BuildSOULDelegationMD(roles, nil)
	if !strings.Contains(md, "## Other") {
		t.Error("unknown category should appear under 'Other' section")
	}
	if !strings.Contains(md, "**@custom**") {
		t.Error("missing @custom in output")
	}
}

func TestBuildSOULDelegationMD_ActiveAgents(t *testing.T) {
	roles := []SOULRoleInfo{
		{Role: "pm", Title: "Product Manager", Category: "executor"},
	}
	active := map[string]string{
		"default": "Assistant — router agent",
		"pm":      "PM",
	}

	md := BuildSOULDelegationMD(roles, active)

	if !strings.Contains(md, "## Currently Active Agents") {
		t.Error("missing Currently Active Agents section")
	}
	if !strings.Contains(md, "**@default**") {
		t.Error("missing @default in active agents")
	}
	if !strings.Contains(md, "**@pm**") {
		t.Error("missing @pm in active agents")
	}
}

func TestBuildSOULDelegationMD_NoActiveAgentsSection(t *testing.T) {
	roles := []SOULRoleInfo{
		{Role: "pm", Title: "Product Manager", Category: "executor"},
	}

	// nil activeAgents → no "Currently Active Agents" section
	md := BuildSOULDelegationMD(roles, nil)
	if strings.Contains(md, "## Currently Active Agents") {
		t.Error("should not have Currently Active Agents section when activeAgents is nil")
	}
}

func TestBuildSOULDelegationMD_RoleTitleFallback(t *testing.T) {
	roles := []SOULRoleInfo{
		{Role: "test", Title: "", Category: "executor"},
	}

	md := BuildSOULDelegationMD(roles, nil)
	// Empty title should fall back to role name
	if !strings.Contains(md, "**@test** — test") {
		t.Error("empty title should fall back to role name")
	}
}

func TestBuildSOULDelegationMD_ContainsUsageInstructions(t *testing.T) {
	roles := []SOULRoleInfo{
		{Role: "coder", Title: "Coder", Category: "executor"},
	}

	md := BuildSOULDelegationMD(roles, nil)
	if !strings.Contains(md, "@agent_name") {
		t.Error("should contain @mention usage instructions")
	}
	if !strings.Contains(md, "complete and authoritative") {
		t.Error("should contain authoritative instruction")
	}
}
