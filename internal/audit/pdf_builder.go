// Package audit provides compliance-ready PDF export for governance evidence chains.
// Sprint 11, ADR-008: Uses johnfercher/maroto v2 (MIT, zero CGO, pure Go).
package audit

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/johnfercher/maroto/v2"
	"github.com/johnfercher/maroto/v2/pkg/components/col"
	"github.com/johnfercher/maroto/v2/pkg/components/row"
	"github.com/johnfercher/maroto/v2/pkg/components/text"
	"github.com/johnfercher/maroto/v2/pkg/config"
	"github.com/johnfercher/maroto/v2/pkg/consts/align"
	"github.com/johnfercher/maroto/v2/pkg/consts/fontstyle"
	"github.com/johnfercher/maroto/v2/pkg/core"
	"github.com/johnfercher/maroto/v2/pkg/props"

	"github.com/nextlevelbuilder/goclaw/internal/store"
)

// Version is injected at build time. Fallback for dev builds.
var Version = "dev"

// AuditTrailPDF builds a compliance-ready PDF for a spec's evidence chain.
// Sections: Header, Spec Summary, PR Gate table, Evidence Timeline, Footer with SHA256.
// Returns PDF bytes or error if chain is empty.
func AuditTrailPDF(spec *store.GovernanceSpec, chain []store.ChainNode) ([]byte, error) {
	if spec == nil {
		return nil, fmt.Errorf("audit: spec is nil")
	}
	if len(chain) == 0 {
		return nil, fmt.Errorf("audit: evidence chain is empty — cannot generate audit trail")
	}

	cfg := config.NewBuilder().
		WithPageNumber().
		WithLeftMargin(15).
		WithRightMargin(15).
		WithTopMargin(15).
		Build()

	m := maroto.New(cfg)

	// Section 1: Header
	addHeader(m, spec)

	// Section 2: Spec Summary
	addSpecSummary(m, spec)

	// Section 3: PR Gate Evaluations (from chain nodes with type "pr_gate")
	addPRGateSection(m, chain)

	// Section 4: Evidence Timeline
	addTimeline(m, chain)

	// Section 5: Footer with SHA256 integrity hash.
	// Hash is computed over spec+chain metadata (not the PDF structure, avoids circular dependency).
	addFooter(m, spec, chain)

	document, err := m.Generate()
	if err != nil {
		return nil, fmt.Errorf("audit: PDF generation failed: %w", err)
	}

	return document.GetBytes(), nil
}

func addHeader(m core.Maroto, spec *store.GovernanceSpec) {
	m.AddRows(
		row.New(12).Add(
			col.New(12).Add(
				text.New("MTClaw Audit Trail Report", props.Text{
					Size:  16,
					Style: fontstyle.Bold,
					Align: align.Center,
				}),
			),
		),
	)

	m.AddRows(
		row.New(6).Add(
			col.New(6).Add(
				text.New(fmt.Sprintf("Spec: %s", spec.SpecID), props.Text{Size: 10, Style: fontstyle.Bold}),
			),
			col.New(6).Add(
				text.New(fmt.Sprintf("Tenant: %s", spec.OwnerID), props.Text{Size: 10, Align: align.Right}),
			),
		),
	)

	m.AddRows(
		row.New(6).Add(
			col.New(6).Add(
				text.New(fmt.Sprintf("Created: %s", spec.CreatedAt.Format("2006-01-02")), props.Text{Size: 9}),
			),
			col.New(6).Add(
				text.New(fmt.Sprintf("Generated: %s", time.Now().UTC().Format(time.RFC3339)), props.Text{Size: 9, Align: align.Right}),
			),
		),
	)

	// Divider
	m.AddRows(row.New(4))
}

func addSpecSummary(m core.Maroto, spec *store.GovernanceSpec) {
	m.AddRows(
		row.New(8).Add(
			col.New(12).Add(
				text.New("1. SPECIFICATION", props.Text{Size: 12, Style: fontstyle.Bold}),
			),
		),
	)

	fields := []struct {
		label string
		value string
	}{
		{"Spec ID", spec.SpecID},
		{"Title", spec.Title},
		{"Version", spec.SpecVersion},
		{"Status", spec.Status},
		{"Priority", spec.Priority},
		{"Effort", spec.EstimatedEffort},
		{"Tier", spec.Tier},
		{"Author", spec.SoulAuthor},
		{"Channel", spec.Channel},
	}

	// Count BDD scenarios
	if len(spec.BDDScenarios) > 0 {
		var scenarios []interface{}
		if json.Unmarshal(spec.BDDScenarios, &scenarios) == nil {
			fields = append(fields, struct {
				label string
				value string
			}{"BDD Scenarios", fmt.Sprintf("%d scenarios", len(scenarios))})
		}
	}

	for _, f := range fields {
		if f.value == "" {
			continue
		}
		m.AddRows(
			row.New(5).Add(
				col.New(3).Add(
					text.New(f.label, props.Text{Size: 9, Style: fontstyle.Bold}),
				),
				col.New(9).Add(
					text.New(f.value, props.Text{Size: 9}),
				),
			),
		)
	}

	m.AddRows(row.New(4))
}

func addPRGateSection(m core.Maroto, chain []store.ChainNode) {
	m.AddRows(
		row.New(8).Add(
			col.New(12).Add(
				text.New("2. PR GATE EVALUATIONS", props.Text{Size: 12, Style: fontstyle.Bold}),
			),
		),
	)

	// Table header
	m.AddRows(
		row.New(6).Add(
			col.New(3).Add(text.New("Verdict", props.Text{Size: 9, Style: fontstyle.Bold})),
			col.New(4).Add(text.New("PR URL", props.Text{Size: 9, Style: fontstyle.Bold})),
			col.New(3).Add(text.New("Date", props.Text{Size: 9, Style: fontstyle.Bold})),
			col.New(2).Add(text.New("ID", props.Text{Size: 9, Style: fontstyle.Bold})),
		),
	)

	prCount := 0
	for _, node := range chain {
		if node.Type != "pr_gate" {
			continue
		}
		prCount++

		verdict := node.Verdict
		if verdict == "" {
			verdict = "N/A"
		}
		prURL := node.PRURL
		if prURL == "" {
			prURL = "-"
		}
		// Truncate long URLs for PDF readability
		if len(prURL) > 40 {
			prURL = prURL[:37] + "..."
		}

		m.AddRows(
			row.New(5).Add(
				col.New(3).Add(text.New(strings.ToUpper(verdict), props.Text{Size: 9})),
				col.New(4).Add(text.New(prURL, props.Text{Size: 8})),
				col.New(3).Add(text.New(node.CreatedAt.Format("2006-01-02"), props.Text{Size: 9})),
				col.New(2).Add(text.New(node.ID.String()[:8], props.Text{Size: 8})),
			),
		)
	}

	if prCount == 0 {
		m.AddRows(
			row.New(5).Add(
				col.New(12).Add(text.New("No PR Gate evaluations linked.", props.Text{Size: 9})),
			),
		)
	}

	m.AddRows(row.New(4))
}

func addTimeline(m core.Maroto, chain []store.ChainNode) {
	m.AddRows(
		row.New(8).Add(
			col.New(12).Add(
				text.New("3. EVIDENCE TIMELINE", props.Text{Size: 12, Style: fontstyle.Bold}),
			),
		),
	)

	// Table header
	m.AddRows(
		row.New(6).Add(
			col.New(3).Add(text.New("Date", props.Text{Size: 9, Style: fontstyle.Bold})),
			col.New(3).Add(text.New("Type", props.Text{Size: 9, Style: fontstyle.Bold})),
			col.New(3).Add(text.New("Status", props.Text{Size: 9, Style: fontstyle.Bold})),
			col.New(3).Add(text.New("ID", props.Text{Size: 9, Style: fontstyle.Bold})),
		),
	)

	for _, node := range chain {
		status := node.Status
		if node.Type == "pr_gate" && node.Verdict != "" {
			status = node.Verdict
		}
		if status == "" {
			status = "-"
		}

		m.AddRows(
			row.New(5).Add(
				col.New(3).Add(text.New(node.CreatedAt.Format("2006-01-02 15:04"), props.Text{Size: 8})),
				col.New(3).Add(text.New(node.Type, props.Text{Size: 9})),
				col.New(3).Add(text.New(status, props.Text{Size: 9})),
				col.New(3).Add(text.New(node.ID.String()[:8], props.Text{Size: 8})),
			),
		)
	}

	m.AddRows(row.New(4))
}

func addFooter(m core.Maroto, spec *store.GovernanceSpec, chain []store.ChainNode) {
	// Compute SHA256 over spec+chain metadata for integrity verification.
	hashInput := fmt.Sprintf("%s|%s|%s|%d", spec.SpecID, spec.ContentHash, spec.Status, len(chain))
	for _, n := range chain {
		hashInput += fmt.Sprintf("|%s:%s", n.ID, n.Type)
	}
	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(hashInput)))

	m.AddRows(
		row.New(8).Add(
			col.New(8).Add(
				text.New(fmt.Sprintf("SHA256: %s", hash[:32]+"..."), props.Text{Size: 7}),
			),
			col.New(4).Add(
				text.New(fmt.Sprintf("MTClaw SDLC Gateway %s", Version), props.Text{
					Size:  7,
					Align: align.Right,
				}),
			),
		),
	)
}
