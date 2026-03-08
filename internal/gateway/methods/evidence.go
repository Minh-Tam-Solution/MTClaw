package methods

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"

	"github.com/Minh-Tam-Solution/MTClaw/internal/evidence"
	"github.com/Minh-Tam-Solution/MTClaw/internal/gateway"
	"github.com/Minh-Tam-Solution/MTClaw/internal/store"
	"github.com/Minh-Tam-Solution/MTClaw/pkg/protocol"
)

// EvidenceMethods handles evidence.chain and evidence.link (Sprint 11, ADR-009).
type EvidenceMethods struct {
	chainBuilder  *evidence.ChainBuilder
	evidenceStore store.EvidenceLinkStore
	specStore     store.SpecStore
}

func NewEvidenceMethods(chainBuilder *evidence.ChainBuilder, evidenceStore store.EvidenceLinkStore, specStore store.SpecStore) *EvidenceMethods {
	return &EvidenceMethods{
		chainBuilder:  chainBuilder,
		evidenceStore: evidenceStore,
		specStore:     specStore,
	}
}

func (m *EvidenceMethods) Register(router *gateway.MethodRouter) {
	router.Register(protocol.MethodEvidenceChain, m.handleChain)
	router.Register(protocol.MethodEvidenceLink, m.handleLink)
}

// handleChain returns the evidence chain for a given spec.
// Params: { "specId": "SPEC-2026-0042" }
func (m *EvidenceMethods) handleChain(ctx context.Context, client *gateway.Client, req *protocol.RequestFrame) {
	var params struct {
		SpecID string `json:"specId"`
	}
	if req.Params != nil {
		json.Unmarshal(req.Params, &params)
	}
	if params.SpecID == "" {
		client.SendResponse(protocol.NewErrorResponse(req.ID, protocol.ErrInvalidRequest, "specId is required"))
		return
	}

	chain, err := m.chainBuilder.BuildChainBySpecID(ctx, params.SpecID)
	if err != nil {
		client.SendResponse(protocol.NewErrorResponse(req.ID, protocol.ErrNotFound, "spec not found: "+err.Error()))
		return
	}

	client.SendResponse(protocol.NewOKResponse(req.ID, chain))
}

// handleLink creates a manual evidence link.
// Params: { "specId": "SPEC-2026-0042", "toType": "pr_gate", "toId": "<uuid>", "linkReason": "manual" }
func (m *EvidenceMethods) handleLink(ctx context.Context, client *gateway.Client, req *protocol.RequestFrame) {
	var params struct {
		SpecID     string `json:"specId"`
		ToType     string `json:"toType"`
		ToID       string `json:"toId"`
		LinkReason string `json:"linkReason"`
	}
	if req.Params != nil {
		json.Unmarshal(req.Params, &params)
	}
	if params.SpecID == "" || params.ToType == "" || params.ToID == "" {
		client.SendResponse(protocol.NewErrorResponse(req.ID, protocol.ErrInvalidRequest, "specId, toType, and toId are required"))
		return
	}

	toUUID, err := uuid.Parse(params.ToID)
	if err != nil {
		client.SendResponse(protocol.NewErrorResponse(req.ID, protocol.ErrInvalidRequest, "invalid toId UUID"))
		return
	}

	spec, err := m.specStore.GetSpec(ctx, params.SpecID)
	if err != nil {
		client.SendResponse(protocol.NewErrorResponse(req.ID, protocol.ErrNotFound, "spec not found: "+err.Error()))
		return
	}

	reason := params.LinkReason
	if reason == "" {
		reason = "manual"
	}

	link := &store.EvidenceLink{
		OwnerID:    spec.OwnerID,
		FromType:   "spec",
		FromID:     spec.ID,
		ToType:     params.ToType,
		ToID:       toUUID,
		LinkReason: reason,
	}
	if err := m.evidenceStore.CreateLink(ctx, link); err != nil {
		client.SendResponse(protocol.NewErrorResponse(req.ID, protocol.ErrInternal, "failed to create link: "+err.Error()))
		return
	}

	client.SendResponse(protocol.NewOKResponse(req.ID, map[string]interface{}{
		"id":         link.ID,
		"fromType":   link.FromType,
		"fromId":     link.FromID,
		"toType":     link.ToType,
		"toId":       link.ToID,
		"linkReason": link.LinkReason,
	}))
}
