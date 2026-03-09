// Package souls — Sprint 8 Task 4: SOUL behavioral test suite.
// 5 critical SOULs × 5 structural checks = 25 tests.
// Tests validate SOUL file structure/content, NOT LLM behavioral output.
// CPO Condition 3: remaining 11 SOULs deferred to Sprint 9.
package souls

import (
	"os"
	"strings"
	"testing"
)

const soulDir = "../../docs/08-collaborate/souls"

// loadSOUL reads a SOUL file by role name.
func loadSOUL(t *testing.T, role string) string {
	t.Helper()
	path := soulDir + "/SOUL-" + role + ".md"
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read SOUL file %s: %v", path, err)
	}
	content := string(data)
	if len(content) < 50 {
		t.Fatalf("SOUL %q content suspiciously short (%d bytes)", role, len(content))
	}
	return content
}

// hasYAMLFrontmatter checks for YAML frontmatter delimiters.
func hasYAMLFrontmatter(content string) bool {
	return strings.HasPrefix(content, "---\n") && strings.Count(content, "---") >= 2
}

// extractFrontmatter returns the YAML frontmatter block.
func extractFrontmatter(content string) string {
	if !strings.HasPrefix(content, "---\n") {
		return ""
	}
	rest := content[4:] // skip first "---\n"
	idx := strings.Index(rest, "---")
	if idx == -1 {
		return ""
	}
	return rest[:idx]
}

// hasSection checks if content contains a markdown heading.
func hasSection(content, heading string) bool {
	return strings.Contains(content, heading)
}

// =============================================================================
// SOUL: PM (Product Manager) — 5 tests
// =============================================================================

func TestSOUL_PM_HasYAMLFrontmatter(t *testing.T) {
	content := loadSOUL(t, "pm")
	if !hasYAMLFrontmatter(content) {
		t.Error("PM SOUL missing YAML frontmatter (required for SOUL lifecycle)")
	}
	fm := extractFrontmatter(content)
	if !strings.Contains(fm, "role: pm") {
		t.Error("PM SOUL frontmatter missing 'role: pm'")
	}
	if !strings.Contains(fm, "category:") {
		t.Error("PM SOUL frontmatter missing 'category' field")
	}
	if !strings.Contains(fm, "version:") {
		t.Error("PM SOUL frontmatter missing 'version' field")
	}
}

func TestSOUL_PM_IdentitySection(t *testing.T) {
	content := loadSOUL(t, "pm")
	if !hasSection(content, "## Identity") {
		t.Fatal("PM SOUL missing '## Identity' section")
	}
	lower := strings.ToLower(content)
	if !strings.Contains(lower, "product manager") {
		t.Error("PM Identity should mention 'Product Manager' role")
	}
}

func TestSOUL_PM_CapabilitiesSection(t *testing.T) {
	content := loadSOUL(t, "pm")
	if !hasSection(content, "## Capabilities") {
		t.Fatal("PM SOUL missing '## Capabilities' section")
	}
	lower := strings.ToLower(content)
	// PM must mention spec/requirement capabilities
	if !strings.Contains(lower, "requirement") && !strings.Contains(lower, "spec") {
		t.Error("PM Capabilities should mention requirements or specifications")
	}
}

func TestSOUL_PM_ConstraintsSection(t *testing.T) {
	content := loadSOUL(t, "pm")
	if !hasSection(content, "## Constraints") {
		t.Fatal("PM SOUL missing '## Constraints' section")
	}
}

func TestSOUL_PM_ChecksumDeterministic(t *testing.T) {
	content := loadSOUL(t, "pm")
	h1 := ChecksumContent(content)
	h2 := ChecksumContent(content)
	if h1 != h2 {
		t.Error("PM SOUL checksum not deterministic")
	}
	if len(h1) != 64 {
		t.Errorf("expected 64-char SHA-256 hex, got %d chars", len(h1))
	}
}

// =============================================================================
// SOUL: Reviewer (Code Reviewer) — 5 tests
// =============================================================================

func TestSOUL_Reviewer_HasYAMLFrontmatter(t *testing.T) {
	content := loadSOUL(t, "reviewer")
	if !hasYAMLFrontmatter(content) {
		t.Error("Reviewer SOUL missing YAML frontmatter")
	}
	fm := extractFrontmatter(content)
	if !strings.Contains(fm, "role: reviewer") {
		t.Error("Reviewer SOUL frontmatter missing 'role: reviewer'")
	}
}

func TestSOUL_Reviewer_IdentitySection(t *testing.T) {
	content := loadSOUL(t, "reviewer")
	if !hasSection(content, "## Identity") {
		t.Fatal("Reviewer SOUL missing '## Identity' section")
	}
	lower := strings.ToLower(content)
	if !strings.Contains(lower, "review") {
		t.Error("Reviewer Identity should mention 'review'")
	}
}

func TestSOUL_Reviewer_CapabilitiesMentionCodeReview(t *testing.T) {
	content := loadSOUL(t, "reviewer")
	if !hasSection(content, "## Capabilities") {
		t.Fatal("Reviewer SOUL missing '## Capabilities' section")
	}
	lower := strings.ToLower(content)
	if !strings.Contains(lower, "code") && !strings.Contains(lower, "review") {
		t.Error("Reviewer Capabilities should mention code review")
	}
}

func TestSOUL_Reviewer_ConstraintsSection(t *testing.T) {
	content := loadSOUL(t, "reviewer")
	if !hasSection(content, "## Constraints") {
		t.Fatal("Reviewer SOUL missing '## Constraints' section")
	}
}

func TestSOUL_Reviewer_ChecksumDeterministic(t *testing.T) {
	content := loadSOUL(t, "reviewer")
	h1 := ChecksumContent(content)
	h2 := ChecksumContent(content)
	if h1 != h2 {
		t.Error("Reviewer SOUL checksum not deterministic")
	}
}

// =============================================================================
// SOUL: Coder (Developer) — 5 tests
// =============================================================================

func TestSOUL_Coder_HasYAMLFrontmatter(t *testing.T) {
	content := loadSOUL(t, "coder")
	if !hasYAMLFrontmatter(content) {
		t.Error("Coder SOUL missing YAML frontmatter")
	}
	fm := extractFrontmatter(content)
	if !strings.Contains(fm, "role: coder") {
		t.Error("Coder SOUL frontmatter missing 'role: coder'")
	}
}

func TestSOUL_Coder_IdentitySection(t *testing.T) {
	content := loadSOUL(t, "coder")
	if !hasSection(content, "## Identity") {
		t.Fatal("Coder SOUL missing '## Identity' section")
	}
	lower := strings.ToLower(content)
	if !strings.Contains(lower, "developer") && !strings.Contains(lower, "coder") {
		t.Error("Coder Identity should mention 'Developer' or 'Coder'")
	}
}

func TestSOUL_Coder_CapabilitiesMentionCode(t *testing.T) {
	content := loadSOUL(t, "coder")
	if !hasSection(content, "## Capabilities") {
		t.Fatal("Coder SOUL missing '## Capabilities' section")
	}
	lower := strings.ToLower(content)
	if !strings.Contains(lower, "code") && !strings.Contains(lower, "implement") {
		t.Error("Coder Capabilities should mention code or implementation")
	}
}

func TestSOUL_Coder_ConstraintsSection(t *testing.T) {
	content := loadSOUL(t, "coder")
	if !hasSection(content, "## Constraints") {
		t.Fatal("Coder SOUL missing '## Constraints' section")
	}
}

func TestSOUL_Coder_ChecksumDeterministic(t *testing.T) {
	content := loadSOUL(t, "coder")
	h1 := ChecksumContent(content)
	h2 := ChecksumContent(content)
	if h1 != h2 {
		t.Error("Coder SOUL checksum not deterministic")
	}
}

// =============================================================================
// SOUL: Enghelp (Engineering Helper) — 4 tests
// =============================================================================

func TestSOUL_Enghelp_HasYAMLFrontmatter(t *testing.T) {
	content := loadSOUL(t, "enghelp")
	if !hasYAMLFrontmatter(content) {
		t.Error("Enghelp SOUL missing YAML frontmatter")
	}
	fm := extractFrontmatter(content)
	if !strings.Contains(fm, "role: enghelp") {
		t.Error("Enghelp SOUL frontmatter missing 'role: enghelp'")
	}
}

func TestSOUL_Enghelp_IdentitySection(t *testing.T) {
	content := loadSOUL(t, "enghelp")
	if !hasSection(content, "## Identity") {
		t.Fatal("Enghelp SOUL missing '## Identity' section")
	}
}

func TestSOUL_Enghelp_CapabilitiesMentionBackend(t *testing.T) {
	content := loadSOUL(t, "enghelp")
	if !hasSection(content, "## Capabilities") {
		t.Fatal("Enghelp SOUL missing '## Capabilities' section")
	}
	lower := strings.ToLower(content)
	if !strings.Contains(lower, "code") && !strings.Contains(lower, "review") &&
		!strings.Contains(lower, "debug") && !strings.Contains(lower, "adr") {
		t.Error("Enghelp Capabilities should mention code, review, debug, or ADR")
	}
}

func TestSOUL_Enghelp_ChecksumDeterministic(t *testing.T) {
	content := loadSOUL(t, "enghelp")
	h1 := ChecksumContent(content)
	h2 := ChecksumContent(content)
	if h1 != h2 {
		t.Error("Enghelp SOUL checksum not deterministic")
	}
}

// =============================================================================
// SOUL: Sales — 5 tests
// =============================================================================

func TestSOUL_Sales_HasYAMLFrontmatter(t *testing.T) {
	content := loadSOUL(t, "sales")
	if !hasYAMLFrontmatter(content) {
		t.Error("Sales SOUL missing YAML frontmatter")
	}
	fm := extractFrontmatter(content)
	if !strings.Contains(fm, "role: sales") {
		t.Error("Sales SOUL frontmatter missing 'role: sales'")
	}
}

func TestSOUL_Sales_IdentitySection(t *testing.T) {
	content := loadSOUL(t, "sales")
	if !hasSection(content, "## Identity") {
		t.Fatal("Sales SOUL missing '## Identity' section")
	}
	lower := strings.ToLower(content)
	if !strings.Contains(lower, "sales") {
		t.Error("Sales Identity should mention 'Sales'")
	}
}

func TestSOUL_Sales_CapabilitiesMentionSalesTasks(t *testing.T) {
	content := loadSOUL(t, "sales")
	if !hasSection(content, "## Capabilities") {
		t.Fatal("Sales SOUL missing '## Capabilities' section")
	}
	lower := strings.ToLower(content)
	// Sales should mention proposals, clients, or business content
	if !strings.Contains(lower, "proposal") && !strings.Contains(lower, "client") &&
		!strings.Contains(lower, "sales") {
		t.Error("Sales Capabilities should mention proposals, clients, or sales tasks")
	}
}

func TestSOUL_Sales_RAGCollectionField(t *testing.T) {
	content := loadSOUL(t, "sales")
	fm := extractFrontmatter(content)
	if !strings.Contains(fm, "rag_collections:") {
		t.Error("Sales SOUL frontmatter missing 'rag_collections' field")
	}
	if !strings.Contains(fm, "sales") {
		t.Error("Sales SOUL rag_collections should include 'sales'")
	}
}

func TestSOUL_Sales_ChecksumDeterministic(t *testing.T) {
	content := loadSOUL(t, "sales")
	h1 := ChecksumContent(content)
	h2 := ChecksumContent(content)
	if h1 != h2 {
		t.Error("Sales SOUL checksum not deterministic")
	}
}

// =============================================================================
// Sprint 9 — T9-03: 12 Governance SOULs × 5 tests = 60 tests
// CTO-32: CEO included (SE4H advisor, sdlc_gates=[G0.1,G4])
//         assistant excluded (category=router, sdlc_gates=[])
// =============================================================================

// =============================================================================
// SOUL: Architect (SE4H advisor) — 5 tests
// =============================================================================

func TestSOUL_Architect_HasYAMLFrontmatter(t *testing.T) {
	content := loadSOUL(t, "architect")
	if !hasYAMLFrontmatter(content) {
		t.Error("Architect SOUL missing YAML frontmatter")
	}
	fm := extractFrontmatter(content)
	if !strings.Contains(fm, "role: architect") {
		t.Error("Architect SOUL frontmatter missing 'role: architect'")
	}
	if !strings.Contains(fm, "category:") {
		t.Error("Architect SOUL frontmatter missing 'category' field")
	}
	if !strings.Contains(fm, "version:") {
		t.Error("Architect SOUL frontmatter missing 'version' field")
	}
}

func TestSOUL_Architect_IdentitySection(t *testing.T) {
	content := loadSOUL(t, "architect")
	if !hasSection(content, "## Identity") {
		t.Fatal("Architect SOUL missing '## Identity' section")
	}
	lower := strings.ToLower(content)
	if !strings.Contains(lower, "architect") {
		t.Error("Architect Identity should mention 'architect'")
	}
}

func TestSOUL_Architect_CapabilitiesSection(t *testing.T) {
	content := loadSOUL(t, "architect")
	if !hasSection(content, "## Capabilities") {
		t.Fatal("Architect SOUL missing '## Capabilities' section")
	}
	lower := strings.ToLower(content)
	if !strings.Contains(lower, "architect") && !strings.Contains(lower, "architecture") && !strings.Contains(lower, "adr") {
		t.Error("Architect Capabilities should mention architecture or ADR")
	}
}

func TestSOUL_Architect_ConstraintsSection(t *testing.T) {
	content := loadSOUL(t, "architect")
	if !hasSection(content, "## Constraints") {
		t.Fatal("Architect SOUL missing '## Constraints' section")
	}
}

func TestSOUL_Architect_ChecksumDeterministic(t *testing.T) {
	content := loadSOUL(t, "architect")
	h1 := ChecksumContent(content)
	h2 := ChecksumContent(content)
	if h1 != h2 {
		t.Error("Architect SOUL checksum not deterministic")
	}
	if len(h1) != 64 {
		t.Errorf("expected 64-char SHA-256 hex, got %d chars", len(h1))
	}
}

// =============================================================================
// SOUL: CEO (SE4H advisor) — 5 tests
// =============================================================================

func TestSOUL_CEO_HasYAMLFrontmatter(t *testing.T) {
	content := loadSOUL(t, "ceo")
	if !hasYAMLFrontmatter(content) {
		t.Error("CEO SOUL missing YAML frontmatter")
	}
	fm := extractFrontmatter(content)
	if !strings.Contains(fm, "role: ceo") {
		t.Error("CEO SOUL frontmatter missing 'role: ceo'")
	}
	if !strings.Contains(fm, "category:") {
		t.Error("CEO SOUL frontmatter missing 'category' field")
	}
	if !strings.Contains(fm, "version:") {
		t.Error("CEO SOUL frontmatter missing 'version' field")
	}
}

func TestSOUL_CEO_IdentitySection(t *testing.T) {
	content := loadSOUL(t, "ceo")
	if !hasSection(content, "## Identity") {
		t.Fatal("CEO SOUL missing '## Identity' section")
	}
	lower := strings.ToLower(content)
	if !strings.Contains(lower, "ceo") && !strings.Contains(lower, "executive") && !strings.Contains(lower, "chief") {
		t.Error("CEO Identity should mention 'ceo', 'executive', or 'chief'")
	}
}

func TestSOUL_CEO_CapabilitiesSection(t *testing.T) {
	content := loadSOUL(t, "ceo")
	if !hasSection(content, "## Capabilities") {
		t.Fatal("CEO SOUL missing '## Capabilities' section")
	}
	lower := strings.ToLower(content)
	if !strings.Contains(lower, "strategic") && !strings.Contains(lower, "decision") && !strings.Contains(lower, "gate") && !strings.Contains(lower, "ceo") {
		t.Error("CEO Capabilities should mention strategic decisions or gates")
	}
}

func TestSOUL_CEO_ConstraintsSection(t *testing.T) {
	content := loadSOUL(t, "ceo")
	if !hasSection(content, "## Constraints") {
		t.Fatal("CEO SOUL missing '## Constraints' section")
	}
}

func TestSOUL_CEO_ChecksumDeterministic(t *testing.T) {
	content := loadSOUL(t, "ceo")
	h1 := ChecksumContent(content)
	h2 := ChecksumContent(content)
	if h1 != h2 {
		t.Error("CEO SOUL checksum not deterministic")
	}
	if len(h1) != 64 {
		t.Errorf("expected 64-char SHA-256 hex, got %d chars", len(h1))
	}
}

// =============================================================================
// SOUL: CPO (SE4H advisor) — 5 tests
// =============================================================================

func TestSOUL_CPO_HasYAMLFrontmatter(t *testing.T) {
	content := loadSOUL(t, "cpo")
	if !hasYAMLFrontmatter(content) {
		t.Error("CPO SOUL missing YAML frontmatter")
	}
	fm := extractFrontmatter(content)
	if !strings.Contains(fm, "role: cpo") {
		t.Error("CPO SOUL frontmatter missing 'role: cpo'")
	}
}

func TestSOUL_CPO_IdentitySection(t *testing.T) {
	content := loadSOUL(t, "cpo")
	if !hasSection(content, "## Identity") {
		t.Fatal("CPO SOUL missing '## Identity' section")
	}
	lower := strings.ToLower(content)
	if !strings.Contains(lower, "product") && !strings.Contains(lower, "cpo") {
		t.Error("CPO Identity should mention 'product' or 'cpo'")
	}
}

func TestSOUL_CPO_CapabilitiesSection(t *testing.T) {
	content := loadSOUL(t, "cpo")
	if !hasSection(content, "## Capabilities") {
		t.Fatal("CPO SOUL missing '## Capabilities' section")
	}
	lower := strings.ToLower(content)
	if !strings.Contains(lower, "product") && !strings.Contains(lower, "strategy") && !strings.Contains(lower, "gate") {
		t.Error("CPO Capabilities should mention product strategy or gates")
	}
}

func TestSOUL_CPO_ConstraintsSection(t *testing.T) {
	content := loadSOUL(t, "cpo")
	if !hasSection(content, "## Constraints") {
		t.Fatal("CPO SOUL missing '## Constraints' section")
	}
}

func TestSOUL_CPO_ChecksumDeterministic(t *testing.T) {
	content := loadSOUL(t, "cpo")
	h1 := ChecksumContent(content)
	h2 := ChecksumContent(content)
	if h1 != h2 {
		t.Error("CPO SOUL checksum not deterministic")
	}
	if len(h1) != 64 {
		t.Errorf("expected 64-char SHA-256 hex, got %d chars", len(h1))
	}
}

// =============================================================================
// SOUL: CTO (SE4H advisor) — 5 tests
// =============================================================================

func TestSOUL_CTO_HasYAMLFrontmatter(t *testing.T) {
	content := loadSOUL(t, "cto")
	if !hasYAMLFrontmatter(content) {
		t.Error("CTO SOUL missing YAML frontmatter")
	}
	fm := extractFrontmatter(content)
	if !strings.Contains(fm, "role: cto") {
		t.Error("CTO SOUL frontmatter missing 'role: cto'")
	}
}

func TestSOUL_CTO_IdentitySection(t *testing.T) {
	content := loadSOUL(t, "cto")
	if !hasSection(content, "## Identity") {
		t.Fatal("CTO SOUL missing '## Identity' section")
	}
	lower := strings.ToLower(content)
	if !strings.Contains(lower, "technical") && !strings.Contains(lower, "cto") {
		t.Error("CTO Identity should mention 'technical' or 'cto'")
	}
}

func TestSOUL_CTO_CapabilitiesSection(t *testing.T) {
	content := loadSOUL(t, "cto")
	if !hasSection(content, "## Capabilities") {
		t.Fatal("CTO SOUL missing '## Capabilities' section")
	}
	lower := strings.ToLower(content)
	if !strings.Contains(lower, "technical") && !strings.Contains(lower, "standard") && !strings.Contains(lower, "performance") {
		t.Error("CTO Capabilities should mention technical standards or performance")
	}
}

func TestSOUL_CTO_ConstraintsSection(t *testing.T) {
	content := loadSOUL(t, "cto")
	if !hasSection(content, "## Constraints") {
		t.Fatal("CTO SOUL missing '## Constraints' section")
	}
}

func TestSOUL_CTO_ChecksumDeterministic(t *testing.T) {
	content := loadSOUL(t, "cto")
	h1 := ChecksumContent(content)
	h2 := ChecksumContent(content)
	if h1 != h2 {
		t.Error("CTO SOUL checksum not deterministic")
	}
	if len(h1) != 64 {
		t.Errorf("expected 64-char SHA-256 hex, got %d chars", len(h1))
	}
}

// =============================================================================
// SOUL: CS (Customer Support, SE4A executor) — 5 tests
// =============================================================================

func TestSOUL_CS_HasYAMLFrontmatter(t *testing.T) {
	content := loadSOUL(t, "cs")
	if !hasYAMLFrontmatter(content) {
		t.Error("CS SOUL missing YAML frontmatter")
	}
	fm := extractFrontmatter(content)
	if !strings.Contains(fm, "role: cs") {
		t.Error("CS SOUL frontmatter missing 'role: cs'")
	}
}

func TestSOUL_CS_IdentitySection(t *testing.T) {
	content := loadSOUL(t, "cs")
	if !hasSection(content, "## Identity") {
		t.Fatal("CS SOUL missing '## Identity' section")
	}
	lower := strings.ToLower(content)
	if !strings.Contains(lower, "customer") && !strings.Contains(lower, "support") && !strings.Contains(lower, "service") {
		t.Error("CS Identity should mention 'customer', 'support', or 'service'")
	}
}

func TestSOUL_CS_CapabilitiesSection(t *testing.T) {
	content := loadSOUL(t, "cs")
	if !hasSection(content, "## Capabilities") {
		t.Fatal("CS SOUL missing '## Capabilities' section")
	}
	lower := strings.ToLower(content)
	if !strings.Contains(lower, "customer") && !strings.Contains(lower, "support") && !strings.Contains(lower, "service") {
		t.Error("CS Capabilities should mention customer support tasks")
	}
}

func TestSOUL_CS_ConstraintsSection(t *testing.T) {
	content := loadSOUL(t, "cs")
	if !hasSection(content, "## Constraints") {
		t.Fatal("CS SOUL missing '## Constraints' section")
	}
}

func TestSOUL_CS_ChecksumDeterministic(t *testing.T) {
	content := loadSOUL(t, "cs")
	h1 := ChecksumContent(content)
	h2 := ChecksumContent(content)
	if h1 != h2 {
		t.Error("CS SOUL checksum not deterministic")
	}
	if len(h1) != 64 {
		t.Errorf("expected 64-char SHA-256 hex, got %d chars", len(h1))
	}
}

// =============================================================================
// SOUL: DevOps (SE4A executor) — 5 tests
// =============================================================================

func TestSOUL_DevOps_HasYAMLFrontmatter(t *testing.T) {
	content := loadSOUL(t, "devops")
	if !hasYAMLFrontmatter(content) {
		t.Error("DevOps SOUL missing YAML frontmatter")
	}
	fm := extractFrontmatter(content)
	if !strings.Contains(fm, "role: devops") {
		t.Error("DevOps SOUL frontmatter missing 'role: devops'")
	}
}

func TestSOUL_DevOps_IdentitySection(t *testing.T) {
	content := loadSOUL(t, "devops")
	if !hasSection(content, "## Identity") {
		t.Fatal("DevOps SOUL missing '## Identity' section")
	}
	lower := strings.ToLower(content)
	if !strings.Contains(lower, "devops") && !strings.Contains(lower, "infrastructure") && !strings.Contains(lower, "ci/cd") {
		t.Error("DevOps Identity should mention 'devops', 'infrastructure', or 'ci/cd'")
	}
}

func TestSOUL_DevOps_CapabilitiesSection(t *testing.T) {
	content := loadSOUL(t, "devops")
	if !hasSection(content, "## Capabilities") {
		t.Fatal("DevOps SOUL missing '## Capabilities' section")
	}
	lower := strings.ToLower(content)
	if !strings.Contains(lower, "devops") && !strings.Contains(lower, "ci") && !strings.Contains(lower, "infrastructure") && !strings.Contains(lower, "deploy") {
		t.Error("DevOps Capabilities should mention CI/CD or infrastructure")
	}
}

func TestSOUL_DevOps_ConstraintsSection(t *testing.T) {
	content := loadSOUL(t, "devops")
	if !hasSection(content, "## Constraints") {
		t.Fatal("DevOps SOUL missing '## Constraints' section")
	}
}

func TestSOUL_DevOps_ChecksumDeterministic(t *testing.T) {
	content := loadSOUL(t, "devops")
	h1 := ChecksumContent(content)
	h2 := ChecksumContent(content)
	if h1 != h2 {
		t.Error("DevOps SOUL checksum not deterministic")
	}
	if len(h1) != 64 {
		t.Errorf("expected 64-char SHA-256 hex, got %d chars", len(h1))
	}
}

// =============================================================================
// SOUL: Fullstack (SE4A executor) — 5 tests
// =============================================================================

func TestSOUL_Fullstack_HasYAMLFrontmatter(t *testing.T) {
	content := loadSOUL(t, "fullstack")
	if !hasYAMLFrontmatter(content) {
		t.Error("Fullstack SOUL missing YAML frontmatter")
	}
	fm := extractFrontmatter(content)
	if !strings.Contains(fm, "role: fullstack") {
		t.Error("Fullstack SOUL frontmatter missing 'role: fullstack'")
	}
}

func TestSOUL_Fullstack_IdentitySection(t *testing.T) {
	content := loadSOUL(t, "fullstack")
	if !hasSection(content, "## Identity") {
		t.Fatal("Fullstack SOUL missing '## Identity' section")
	}
	lower := strings.ToLower(content)
	if !strings.Contains(lower, "fullstack") && !strings.Contains(lower, "full-stack") && !strings.Contains(lower, "full stack") {
		t.Error("Fullstack Identity should mention 'fullstack' or 'full-stack'")
	}
}

func TestSOUL_Fullstack_CapabilitiesSection(t *testing.T) {
	content := loadSOUL(t, "fullstack")
	if !hasSection(content, "## Capabilities") {
		t.Fatal("Fullstack SOUL missing '## Capabilities' section")
	}
	lower := strings.ToLower(content)
	if !strings.Contains(lower, "typescript") && !strings.Contains(lower, "frontend") && !strings.Contains(lower, "backend") && !strings.Contains(lower, "fullstack") {
		t.Error("Fullstack Capabilities should mention TypeScript, frontend, or backend")
	}
}

func TestSOUL_Fullstack_ConstraintsSection(t *testing.T) {
	content := loadSOUL(t, "fullstack")
	if !hasSection(content, "## Constraints") {
		t.Fatal("Fullstack SOUL missing '## Constraints' section")
	}
}

func TestSOUL_Fullstack_ChecksumDeterministic(t *testing.T) {
	content := loadSOUL(t, "fullstack")
	h1 := ChecksumContent(content)
	h2 := ChecksumContent(content)
	if h1 != h2 {
		t.Error("Fullstack SOUL checksum not deterministic")
	}
	if len(h1) != 64 {
		t.Errorf("expected 64-char SHA-256 hex, got %d chars", len(h1))
	}
}

// =============================================================================
// SOUL: ITAdmin (SE4A executor) — 5 tests
// =============================================================================

func TestSOUL_ITAdmin_HasYAMLFrontmatter(t *testing.T) {
	content := loadSOUL(t, "itadmin")
	if !hasYAMLFrontmatter(content) {
		t.Error("ITAdmin SOUL missing YAML frontmatter")
	}
	fm := extractFrontmatter(content)
	if !strings.Contains(fm, "role: itadmin") {
		t.Error("ITAdmin SOUL frontmatter missing 'role: itadmin'")
	}
}

func TestSOUL_ITAdmin_IdentitySection(t *testing.T) {
	content := loadSOUL(t, "itadmin")
	if !hasSection(content, "## Identity") {
		t.Fatal("ITAdmin SOUL missing '## Identity' section")
	}
	lower := strings.ToLower(content)
	if !strings.Contains(lower, "itadmin") && !strings.Contains(lower, "it admin") && !strings.Contains(lower, "infrastructure") && !strings.Contains(lower, "security") {
		t.Error("ITAdmin Identity should mention IT admin, infrastructure, or security")
	}
}

func TestSOUL_ITAdmin_CapabilitiesSection(t *testing.T) {
	content := loadSOUL(t, "itadmin")
	if !hasSection(content, "## Capabilities") {
		t.Fatal("ITAdmin SOUL missing '## Capabilities' section")
	}
	lower := strings.ToLower(content)
	if !strings.Contains(lower, "infrastructure") && !strings.Contains(lower, "security") && !strings.Contains(lower, "itadmin") && !strings.Contains(lower, "it admin") {
		t.Error("ITAdmin Capabilities should mention infrastructure ops or security hardening")
	}
}

func TestSOUL_ITAdmin_ConstraintsSection(t *testing.T) {
	content := loadSOUL(t, "itadmin")
	if !hasSection(content, "## Constraints") {
		t.Fatal("ITAdmin SOUL missing '## Constraints' section")
	}
}

func TestSOUL_ITAdmin_ChecksumDeterministic(t *testing.T) {
	content := loadSOUL(t, "itadmin")
	h1 := ChecksumContent(content)
	h2 := ChecksumContent(content)
	if h1 != h2 {
		t.Error("ITAdmin SOUL checksum not deterministic")
	}
	if len(h1) != 64 {
		t.Errorf("expected 64-char SHA-256 hex, got %d chars", len(h1))
	}
}

// =============================================================================
// SOUL: PJM (Project Manager, SE4A executor) — 5 tests
// =============================================================================

func TestSOUL_PJM_HasYAMLFrontmatter(t *testing.T) {
	content := loadSOUL(t, "pjm")
	if !hasYAMLFrontmatter(content) {
		t.Error("PJM SOUL missing YAML frontmatter")
	}
	fm := extractFrontmatter(content)
	if !strings.Contains(fm, "role: pjm") {
		t.Error("PJM SOUL frontmatter missing 'role: pjm'")
	}
}

func TestSOUL_PJM_IdentitySection(t *testing.T) {
	content := loadSOUL(t, "pjm")
	if !hasSection(content, "## Identity") {
		t.Fatal("PJM SOUL missing '## Identity' section")
	}
	lower := strings.ToLower(content)
	if !strings.Contains(lower, "project") && !strings.Contains(lower, "pjm") && !strings.Contains(lower, "sprint") {
		t.Error("PJM Identity should mention 'project', 'pjm', or 'sprint'")
	}
}

func TestSOUL_PJM_CapabilitiesSection(t *testing.T) {
	content := loadSOUL(t, "pjm")
	if !hasSection(content, "## Capabilities") {
		t.Fatal("PJM SOUL missing '## Capabilities' section")
	}
	lower := strings.ToLower(content)
	if !strings.Contains(lower, "project") && !strings.Contains(lower, "sprint") && !strings.Contains(lower, "timeline") && !strings.Contains(lower, "task") {
		t.Error("PJM Capabilities should mention project management or sprint planning")
	}
}

func TestSOUL_PJM_ConstraintsSection(t *testing.T) {
	content := loadSOUL(t, "pjm")
	if !hasSection(content, "## Constraints") {
		t.Fatal("PJM SOUL missing '## Constraints' section")
	}
}

func TestSOUL_PJM_ChecksumDeterministic(t *testing.T) {
	content := loadSOUL(t, "pjm")
	h1 := ChecksumContent(content)
	h2 := ChecksumContent(content)
	if h1 != h2 {
		t.Error("PJM SOUL checksum not deterministic")
	}
	if len(h1) != 64 {
		t.Errorf("expected 64-char SHA-256 hex, got %d chars", len(h1))
	}
}

// =============================================================================
// SOUL: Researcher (SE4A executor) — 5 tests
// =============================================================================

func TestSOUL_Researcher_HasYAMLFrontmatter(t *testing.T) {
	content := loadSOUL(t, "researcher")
	if !hasYAMLFrontmatter(content) {
		t.Error("Researcher SOUL missing YAML frontmatter")
	}
	fm := extractFrontmatter(content)
	if !strings.Contains(fm, "role: researcher") {
		t.Error("Researcher SOUL frontmatter missing 'role: researcher'")
	}
}

func TestSOUL_Researcher_IdentitySection(t *testing.T) {
	content := loadSOUL(t, "researcher")
	if !hasSection(content, "## Identity") {
		t.Fatal("Researcher SOUL missing '## Identity' section")
	}
	lower := strings.ToLower(content)
	if !strings.Contains(lower, "research") {
		t.Error("Researcher Identity should mention 'research'")
	}
}

func TestSOUL_Researcher_CapabilitiesSection(t *testing.T) {
	content := loadSOUL(t, "researcher")
	if !hasSection(content, "## Capabilities") {
		t.Fatal("Researcher SOUL missing '## Capabilities' section")
	}
	lower := strings.ToLower(content)
	if !strings.Contains(lower, "research") && !strings.Contains(lower, "methodology") && !strings.Contains(lower, "analysis") && !strings.Contains(lower, "evidence") {
		t.Error("Researcher Capabilities should mention research methodology or evidence")
	}
}

func TestSOUL_Researcher_ConstraintsSection(t *testing.T) {
	content := loadSOUL(t, "researcher")
	if !hasSection(content, "## Constraints") {
		t.Fatal("Researcher SOUL missing '## Constraints' section")
	}
}

func TestSOUL_Researcher_ChecksumDeterministic(t *testing.T) {
	content := loadSOUL(t, "researcher")
	h1 := ChecksumContent(content)
	h2 := ChecksumContent(content)
	if h1 != h2 {
		t.Error("Researcher SOUL checksum not deterministic")
	}
	if len(h1) != 64 {
		t.Errorf("expected 64-char SHA-256 hex, got %d chars", len(h1))
	}
}

// =============================================================================
// SOUL: Tester (QA, SE4A executor) — 5 tests
// =============================================================================

func TestSOUL_Tester_HasYAMLFrontmatter(t *testing.T) {
	content := loadSOUL(t, "tester")
	if !hasYAMLFrontmatter(content) {
		t.Error("Tester SOUL missing YAML frontmatter")
	}
	fm := extractFrontmatter(content)
	if !strings.Contains(fm, "role: tester") {
		t.Error("Tester SOUL frontmatter missing 'role: tester'")
	}
}

func TestSOUL_Tester_IdentitySection(t *testing.T) {
	content := loadSOUL(t, "tester")
	if !hasSection(content, "## Identity") {
		t.Fatal("Tester SOUL missing '## Identity' section")
	}
	lower := strings.ToLower(content)
	if !strings.Contains(lower, "test") && !strings.Contains(lower, "qa") && !strings.Contains(lower, "quality") {
		t.Error("Tester Identity should mention 'test', 'qa', or 'quality'")
	}
}

func TestSOUL_Tester_CapabilitiesSection(t *testing.T) {
	content := loadSOUL(t, "tester")
	if !hasSection(content, "## Capabilities") {
		t.Fatal("Tester SOUL missing '## Capabilities' section")
	}
	lower := strings.ToLower(content)
	if !strings.Contains(lower, "test") && !strings.Contains(lower, "coverage") && !strings.Contains(lower, "qa") {
		t.Error("Tester Capabilities should mention QA patterns or test coverage")
	}
}

func TestSOUL_Tester_ConstraintsSection(t *testing.T) {
	content := loadSOUL(t, "tester")
	if !hasSection(content, "## Constraints") {
		t.Fatal("Tester SOUL missing '## Constraints' section")
	}
}

func TestSOUL_Tester_ChecksumDeterministic(t *testing.T) {
	content := loadSOUL(t, "tester")
	h1 := ChecksumContent(content)
	h2 := ChecksumContent(content)
	if h1 != h2 {
		t.Error("Tester SOUL checksum not deterministic")
	}
	if len(h1) != 64 {
		t.Errorf("expected 64-char SHA-256 hex, got %d chars", len(h1))
	}
}

// =============================================================================
// SOUL: Writer (Documentation, SE4A executor) — 5 tests
// =============================================================================

func TestSOUL_Writer_HasYAMLFrontmatter(t *testing.T) {
	content := loadSOUL(t, "writer")
	if !hasYAMLFrontmatter(content) {
		t.Error("Writer SOUL missing YAML frontmatter")
	}
	fm := extractFrontmatter(content)
	if !strings.Contains(fm, "role: writer") {
		t.Error("Writer SOUL frontmatter missing 'role: writer'")
	}
}

func TestSOUL_Writer_IdentitySection(t *testing.T) {
	content := loadSOUL(t, "writer")
	if !hasSection(content, "## Identity") {
		t.Fatal("Writer SOUL missing '## Identity' section")
	}
	lower := strings.ToLower(content)
	if !strings.Contains(lower, "writer") && !strings.Contains(lower, "document") && !strings.Contains(lower, "writing") {
		t.Error("Writer Identity should mention 'writer', 'document', or 'writing'")
	}
}

func TestSOUL_Writer_CapabilitiesSection(t *testing.T) {
	content := loadSOUL(t, "writer")
	if !hasSection(content, "## Capabilities") {
		t.Fatal("Writer SOUL missing '## Capabilities' section")
	}
	lower := strings.ToLower(content)
	if !strings.Contains(lower, "document") && !strings.Contains(lower, "writing") && !strings.Contains(lower, "guide") && !strings.Contains(lower, "writer") {
		t.Error("Writer Capabilities should mention documentation or writing")
	}
}

func TestSOUL_Writer_ConstraintsSection(t *testing.T) {
	content := loadSOUL(t, "writer")
	if !hasSection(content, "## Constraints") {
		t.Fatal("Writer SOUL missing '## Constraints' section")
	}
}

func TestSOUL_Writer_ChecksumDeterministic(t *testing.T) {
	content := loadSOUL(t, "writer")
	h1 := ChecksumContent(content)
	h2 := ChecksumContent(content)
	if h1 != h2 {
		t.Error("Writer SOUL checksum not deterministic")
	}
	if len(h1) != 64 {
		t.Errorf("expected 64-char SHA-256 hex, got %d chars", len(h1))
	}
}
