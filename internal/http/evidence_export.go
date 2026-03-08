package http

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/Minh-Tam-Solution/MTClaw/internal/audit"
	"github.com/Minh-Tam-Solution/MTClaw/internal/evidence"
	"github.com/Minh-Tam-Solution/MTClaw/internal/store"
)

// EvidenceExportHandler exports governance evidence (specs + PR evaluations) for audit.
type EvidenceExportHandler struct {
	specStore     store.SpecStore
	prGateStore   store.PRGateStore
	evidenceStore store.EvidenceLinkStore
	chainBuilder  *evidence.ChainBuilder
	token         string
}

// NewEvidenceExportHandler creates a handler for evidence export endpoints.
func NewEvidenceExportHandler(specStore store.SpecStore, prGateStore store.PRGateStore, token string) *EvidenceExportHandler {
	return &EvidenceExportHandler{
		specStore:   specStore,
		prGateStore: prGateStore,
		token:       token,
	}
}

// SetEvidenceChain configures the handler for PDF audit trail export (Sprint 11, T11-03).
func (h *EvidenceExportHandler) SetEvidenceChain(evidenceStore store.EvidenceLinkStore, chainBuilder *evidence.ChainBuilder) {
	h.evidenceStore = evidenceStore
	h.chainBuilder = chainBuilder
}

// RegisterRoutes registers the evidence export endpoints.
func (h *EvidenceExportHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/evidence/export", h.authMiddleware(h.handleExport))
	mux.HandleFunc("GET /v1/evidence/audit-trail.pdf", h.authMiddleware(h.handleAuditTrailPDF))
}

func (h *EvidenceExportHandler) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if h.token != "" {
			if extractBearerToken(r) != h.token {
				writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
				return
			}
		}
		next(w, r)
	}
}

func (h *EvidenceExportHandler) handleExport(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	format := q.Get("format")
	if format == "" {
		format = "json"
	}
	rail := q.Get("rail")

	// Parse date range (default: last 30 days)
	now := time.Now().UTC()
	from := now.AddDate(0, 0, -30)
	to := now
	if v := q.Get("from"); v != "" {
		if parsed, err := time.Parse("2006-01-02", v); err == nil {
			from = parsed
		}
	}
	if v := q.Get("to"); v != "" {
		if parsed, err := time.Parse("2006-01-02", v); err == nil {
			to = parsed.Add(24*time.Hour - time.Nanosecond) // end of day
		}
	}

	ctx := r.Context()

	// Collect specs (Rail #1)
	var specs []store.GovernanceSpec
	if (rail == "" || rail == "spec-factory") && h.specStore != nil {
		var err error
		const specLimit = 1000
		specs, err = h.specStore.ListSpecs(ctx, store.SpecListOpts{
			Since: &from,
			Limit: specLimit,
		})
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load specs"})
			return
		}
		if len(specs) == specLimit {
			slog.Warn("evidence_export: spec result capped at limit — export may be incomplete; pagination not yet implemented",
				"limit", specLimit, "from", from, "to", to)
		}
		// Filter by date range
		var filtered []store.GovernanceSpec
		for _, s := range specs {
			if !s.CreatedAt.Before(from) && !s.CreatedAt.After(to) {
				filtered = append(filtered, s)
			}
		}
		specs = filtered
	}

	// Collect PR evaluations (Rail #2)
	var prEvals []store.PRGateEvaluation
	if (rail == "" || rail == "pr-gate") && h.prGateStore != nil {
		var err error
		prEvals, err = h.prGateStore.ListEvaluations(ctx, store.PRGateFilter{
			Since: &from,
			Until: &to,
			Limit: 1000,
		})
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to load PR evaluations"})
			return
		}
	}

	// Calculate stats
	passCount := 0
	for _, e := range prEvals {
		if e.Verdict == "pass" {
			passCount++
		}
	}
	passRate := float64(0)
	if len(prEvals) > 0 {
		passRate = float64(passCount) / float64(len(prEvals))
	}

	if format == "csv" {
		h.writeCSV(w, specs, prEvals)
		return
	}

	// JSON response
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"export_date": now.Format(time.RFC3339),
		"period": map[string]string{
			"from": from.Format("2006-01-02"),
			"to":   to.Format("2006-01-02"),
		},
		"specs":          specs,
		"pr_evaluations": prEvals,
		"stats": map[string]interface{}{
			"total_specs":      len(specs),
			"total_pr_reviews": len(prEvals),
			"pass_rate":        passRate,
		},
	})
}

func (h *EvidenceExportHandler) writeCSV(w http.ResponseWriter, specs []store.GovernanceSpec, prEvals []store.PRGateEvaluation) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="evidence-export.csv"`)
	w.WriteHeader(http.StatusOK)

	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Header row
	writer.Write([]string{
		"type", "id", "created_at", "title_or_repo", "status_or_verdict",
		"author", "priority_or_mode", "detail",
	})

	// Spec rows
	for _, s := range specs {
		writer.Write([]string{
			"spec",
			s.SpecID,
			s.CreatedAt.Format(time.RFC3339),
			s.Title,
			s.Status,
			s.SoulAuthor,
			s.Priority,
			fmt.Sprintf("tier=%s effort=%s", s.Tier, s.EstimatedEffort),
		})
	}

	// PR evaluation rows
	for _, e := range prEvals {
		detail := fmt.Sprintf("pr=#%d sha=%s", e.PRNumber, e.HeadSHA)
		rulesJSON, _ := json.Marshal(json.RawMessage(e.RulesEvaluated))
		if len(rulesJSON) > 2 { // not empty "[]"
			detail += " rules=" + string(rulesJSON)
		}
		writer.Write([]string{
			"pr_gate",
			e.ID.String(),
			e.CreatedAt.Format(time.RFC3339),
			e.Repo,
			e.Verdict,
			e.SoulAuthor,
			e.Mode,
			detail,
		})
	}
}

// handleAuditTrailPDF generates a compliance-ready PDF for a spec's evidence chain.
// GET /v1/evidence/audit-trail.pdf?specId=SPEC-2026-0042
func (h *EvidenceExportHandler) handleAuditTrailPDF(w http.ResponseWriter, r *http.Request) {
	specID := r.URL.Query().Get("specId")
	if specID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "specId query parameter is required"})
		return
	}

	if h.chainBuilder == nil || h.specStore == nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "audit trail PDF not available — evidence chain not configured"})
		return
	}

	ctx := r.Context()

	spec, err := h.specStore.GetSpec(ctx, specID)
	if err != nil {
		slog.Warn("audit_trail_pdf: spec not found", "spec_id", specID, "error", err)
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "spec not found"})
		return
	}

	chainResult, err := h.chainBuilder.BuildChain(ctx, spec)
	if err != nil {
		slog.Error("audit_trail_pdf: failed to build chain", "spec_id", specID, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "failed to build evidence chain"})
		return
	}

	pdfBytes, err := audit.AuditTrailPDF(spec, chainResult.Chain)
	if err != nil {
		slog.Error("audit_trail_pdf: PDF generation failed", "spec_id", specID, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "PDF generation failed: " + err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="audit-trail-%s.pdf"`, specID))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(pdfBytes)))
	w.WriteHeader(http.StatusOK)
	w.Write(pdfBytes)
}
