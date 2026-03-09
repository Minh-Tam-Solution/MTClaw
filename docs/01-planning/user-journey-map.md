# User Journey Map — MTClaw

**SDLC Stage**: 01-Planning
**Version**: 1.0.0
**Date**: 2026-03-02
**Author**: [@pm] + [@researcher]
**CPO Directive**: CONCERN-1 — "Khi nhân viên mở Telegram lần đầu, họ thấy gì?"

---

## Research Methodology

[@researcher]: User journeys derived from:
1. Sprint 29 interview data (n=8) — pain points and desired workflows
2. GoClaw Telegram implementation (`internal/channels/telegram/`) — actual commands and message flow
3. GoClaw agent loop (`internal/agent/`) — SOUL injection and routing
4. Confidence: **Medium** — based on interview evidence + GoClaw code analysis, not yet user-tested

---

## Journey 1: Engineering — Code Review Request

**Persona**: Phú (Senior Dev, Engineering team)
**Context**: Cần review PR trước khi merge, muốn Bflow API context
**SOUL**: `assistant` (default router) → delegates to `enghelp` or `reviewer` based on context

### Flow

```
Step 1: Open Telegram → find MTClaw bot → /start
        ┌──────────────────────────────────────────┐
        │ 👋 MTClaw — Governance Assistant          │
        │                                          │
        │ Xin chào Phú! Tôi là MTClaw assistant.   │
        │ Bạn có thể:                              │
        │ • Gõ câu hỏi trực tiếp                   │
        │ • @pm để yêu cầu spec                    │
        │ • @reviewer để code review               │
        │ • /help để xem danh sách commands         │
        │                                          │
        │ SOUL: assistant (universal router)        │
        └──────────────────────────────────────────┘
        ⏱ Response: <2s (welcome = cached, no AI call)

Step 2: Gõ câu hỏi — "Review PR #42: thêm tenant isolation cho agent_shares table"
        → assistant detects engineering context → delegates to `enghelp`
        → AI-Platform call: POST /v1/chat/completions
           system_prompt = SOUL-enghelp.md content
           user_message = PR review request
        ⏱ Response: 3-5s (AI generation)

Step 3: Muốn deep review → "@reviewer PR #42 check SQL injection risk"
        → SOUL switch: enghelp → reviewer (via @mention detection)
        → reviewer SOUL has Rail #2 (PR Gate) context
        → Response includes structured checklist
        ⏱ Response: 3-5s

Step 4: Hài lòng → merge PR
        → Evidence: conversation logged in sessions table
        → trace_id links all interactions
```

### Key Moments

| Moment | User Feeling | Design Implication |
|--------|-------------|-------------------|
| `/start` response | "Ồ, nó biết tôi là ai" | Welcome phải personalized (tên + role) |
| First AI response | "Nhanh hơn chat.nqh" | Must be <5s — faster than generic AI |
| `@reviewer` switch | "Chuyên gia review" | SOUL switch phải seamless, không cần restart |
| Evidence trail | (invisible) | User không cần biết — nhưng audit log phải complete |

---

## Journey 2: Sales — Proposal Draft

**Persona**: Hương (Sales, 2 years at MTS)
**Context**: Cần draft proposal cho khách hàng SME, muốn Bflow pricing context
**SOUL**: `assistant` (default) → delegates to `sales` on-demand

### Flow

```
Step 1: Open Telegram → send message trực tiếp (không cần /start lần 2)
        "Tạo proposal cho khách hàng ABC Corp, gói Bflow Professional"
        → SOUL `assistant` (default router) detects sales context
        → Delegates to `sales` SOUL automatically
        ⏱ Response: 3-5s

Step 2: AI responds với draft proposal
        ┌──────────────────────────────────────────┐
        │ 📋 Proposal Draft — ABC Corp             │
        │                                          │
        │ Gói: Bflow Professional                  │
        │ Giá: [từ RAG: pricing tiers]             │
        │ Tính năng: [từ RAG: feature list]        │
        │ Case study tương tự: [từ RAG]            │
        │                                          │
        │ ⚠️ Lưu ý: Giá chưa bao gồm VAT.        │
        │ Cần confirm với manager trước khi gửi.   │
        └──────────────────────────────────────────┘
        → AI-Platform RAG call: POST /v1/rag/query
           collection = "sales"
           query = "Bflow Professional pricing features"

Step 3: "Thêm case study ngành F&B cho proposal này"
        → RAG query: collection = "mts-case-studies"
        → AI enriches proposal with relevant case study
        ⏱ Response: 3-5s

Step 4: "OK, format lại thành email gửi cho khách"
        → AI formats → user copy-paste vào email
        → Evidence: proposal generation logged
```

### Key Moments

| Moment | User Feeling | Design Implication |
|--------|-------------|-------------------|
| First message (no /start) | "Tiện, không cần ceremony" | Returning users skip onboarding |
| RAG-enriched pricing | "Nó biết giá Bflow!" | RAG collections phải accurate + up-to-date |
| Case study | "Đỡ phải tìm trong Notion" | Save 25 phút (baseline: 30 phút search) |
| Format to email | "Từ draft → gửi nhanh" | Output formatting matters for copy-paste |

---

## Journey 3: General — First-Time User (Back Office)

**Persona**: Thảo (HR Admin, Back Office team)
**Context**: Lần đầu dùng MTClaw, không rành tech, cần HR policy lookup
**SOUL**: `assistant` (default, handles directly — no delegation needed for general tasks)

### Flow

```
Step 1: Nhận link bot từ đồng nghiệp → click → /start
        ┌──────────────────────────────────────────┐
        │ 👋 Chào Thảo!                            │
        │                                          │
        │ MTClaw giúp bạn:                         │
        │ • Tra cứu chính sách HR nhanh            │
        │ • Soạn biên bản họp từ ghi chú           │
        │ • Trả lời câu hỏi về quy trình công ty   │
        │                                          │
        │ Gõ câu hỏi bất kỳ để bắt đầu!           │
        └──────────────────────────────────────────┘
        ⏱ Response: <2s

Step 2: "Chính sách nghỉ phép năm 2026 như thế nào?"
        → assistant SOUL handles directly (general Q&A = no delegation needed)
        → RAG query: collection = "hr-policies"
        ⏱ Response: 3-5s
        ┌──────────────────────────────────────────┐
        │ 📋 Chính sách nghỉ phép 2026              │
        │                                          │
        │ • Nghỉ phép năm: 12 ngày (FTE)           │
        │ • Nghỉ phép tích lũy: tối đa 5 ngày...  │
        │ • [nguồn: HR-Policy-2026.md]             │
        │                                          │
        │ ℹ️ Tôi trả lời dựa trên tài liệu HR.    │
        │ Nếu cần xác nhận, liên hệ HR Manager.   │
        └──────────────────────────────────────────┘

Step 3: "Tạo biên bản họp: họp sprint review 10/3, thành viên: Phú, Hương, An"
        → assistant SOUL handles directly (meeting notes = direct capability)
        → Output: formatted minutes
        ⏱ Response: 3-5s

Step 4: (1 tuần sau) Thảo quay lại → gõ câu hỏi mới
        → Session history preserved → AI has context
        → Không cần nhắc lại "tôi là HR Admin"
```

### Key Moments

| Moment | User Feeling | Design Implication |
|--------|-------------|-------------------|
| /start (first time) | "Dễ hiểu, không phức tạp" | Onboarding phải <30 giây, Vietnamese, simple |
| HR policy answer | "Nó biết chính sách MTS!" | RAG accuracy = adoption gate (nếu sai → mất tin tưởng) |
| Source citation | "Tôi verify được" | Always cite source doc → build trust |
| Returning user | "Nó nhớ tôi" | Session persistence critical cho non-tech users |

---

## Cross-Journey Insights ([@researcher] synthesis)

### Pattern 1: First 30 Seconds Define Adoption

| User Type | First 30s Must Deliver | Failure Mode |
|-----------|----------------------|-------------|
| Tech (Engineering) | AI response with Bflow context | Generic ChatGPT answer → abandon |
| Sales | Proposal with real pricing | Wrong pricing → distrust |
| Non-tech (Back Office) | Simple answer in Vietnamese | Complex UI → confusion |

### Pattern 2: SOUL Switching Must Be Invisible

Users don't think in "SOULs" — they think in tasks. SOUL routing should be:
- **Automatic** for context detection (HR question → assistant handles directly)
- **Explicit** only for power users (`@reviewer`, `@pm`)
- **Never require restart** — switch mid-conversation

### Pattern 3: RAG Accuracy = Adoption Gate

From interviews (Sprint 29):
- Engineering: "Nếu AI sai thì nguy hiểm hơn không dùng" (2/3 respondents)
- Sales: Wrong pricing = lost deal
- HR: Wrong policy answer = employee complaints

**Implication**: Phase 1 RAG collections must be manually curated and verified before deployment. Automated ingestion is Phase 2.

### Pattern 4: Evidence Trail = Invisible to User

Users should never interact with audit trail directly. It must be:
- Automatic (every session, every AI call logged)
- Queryable by admins only
- Linked via trace_id for full request chain

---

## Technical Mapping (GoClaw → Journey)

| Journey Step | GoClaw Component | Code Path |
|-------------|-----------------|-----------|
| `/start` | Telegram command handler | `internal/channels/telegram/commands.go:70` |
| Text message | Agent loop | `internal/agent/loop.go` |
| SOUL injection | System prompt builder | `internal/agent/systemprompt.go` |
| `@role` detection | Agent resolver/router | `internal/agent/router.go` |
| RAG query | Memory search | `internal/memory/search.go` |
| Session persist | Session manager | `internal/sessions/manager.go` |
| Evidence/trace | Tracing collector | `internal/tracing/collector.go` |

---

## Sprint 4 Validation Plan (CPO CONCERN-3)

SOUL feedback session:
- **Who**: 3-4 MTS users (1 Engineering, 1 Sales, 1 Back Office)
- **Method**: 15-minute live session, user tries their assigned SOUL
- **Measure**: Time to first useful answer, satisfaction (1-5), would-use-again (Y/N)
- **Output**: Findings doc → SOUL tuning for Sprint 5

---

**Confidence**: Medium — grounded in interview data + GoClaw code, not yet user-tested
**Next validation**: Sprint 4 SOUL feedback session
