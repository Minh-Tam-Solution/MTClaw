# Baseline Metrics — MTS Internal Platform

**US**: US-029-004
**Status**: ✅ done — consolidated 2026-03-04 (all interviews complete)
**Owner**: [@pm]
**Source**: Synthesized from US-029-001 (Engineering), US-029-002 (Sales + CS), US-029-003 (Back Office)
**Purpose**: Establish pre-AI baseline for ROI calculation + G0.1 evidence

---

## Data Quality Rules ([@cto] directive — Sprint 29 Day 1)

| Situation | Rule |
|-----------|------|
| n ≥ 3 interviews | Dùng median — reportable |
| n = 2 interviews | Ghi rõ: `___ phút (n=2 — small sample, interpret with caution)` |
| n = 1 interview | Ghi rõ: `___ phút (n=1 — single data point, NOT extrapolate)` |
| n = 0 | Ghi: `TBD — interview not yet conducted` |

**Principle**: G0.1 evidence phải honest về confidence level. Đừng extrapolate từ n=1 hay n=2 như thể đó là fact.

---

## Team Size Snapshot (MTS, 2026-03-01)

| Team | Headcount (est.) | Interview Coverage |
|------|-----------------|-------------------|
| Engineering (Dev/QA/DevOps) | ~8-12 | 3 / 3+ interviews ✅ |
| Sales | ~3-5 | 2 / 3 interviews (n=2 — small sample) |
| Customer Success | ~2-4 | 1 / 3 interviews (n=1 — single data point) |
| Back Office + Marketing | ~3-6 | 2 / 2 interviews ✅ |
| **Total MTS** | **~16-27** | **8 interviews total** |

---

## Engineering Baselines

*Source: [interviews-engineering.md](interviews-engineering.md) — n=3, MEDIUM confidence*

| Metric | Baseline (no AI) | Notes |
|--------|-----------------|-------|
| Code review — avg PR | **20 min** (median n=3) | Range: 15-25 min |
| Code review — complex PR (cross-module, AGPL, tenant isolation) | **75 min** (median n=3) | Range: 60-90 min |
| PR description (full, with context) | **15 min** (median n=3) | |
| ADR draft (from scratch) | **3 giờ** (median n=3) | Range: 2.5-4h |
| Bflow docs lookup (per question) | **10 min** (n=2 cited explicitly — small sample) | Phú + Dương |
| chat.nhatquangholding.com adoption | **~1/3 dùng thường xuyên** | 1/3 dùng, 2/3 barrier vì generic |

**Root cause for low AI adoption**: No persistent Bflow/NQH-Bot knowledge → phải rebuild context mỗi session (5-10 phút overhead)

**Target (Bflow-aware AI)**:
| Metric | Target | Estimated Saving |
|--------|--------|-----------------|
| Code review — avg PR | 10-12 min | -8-10 min/PR |
| Code review — complex PR | 30-35 min | -40 min/PR |
| PR description | 5-8 min | -7-10 min |
| ADR draft | 1-1.5 giờ | -1.5-2 giờ |

---

## Sales Baselines

*Source: [interviews-sales-cs.md](interviews-sales-cs.md) — n=2 (small sample), interpret with caution*

| Metric | Baseline (no AI) | Notes |
|--------|-----------------|-------|
| B2B proposal draft (brief → first draft) | **2.75 giờ** (n=2 — small sample) | Range: 2.5-3h |
| RFP response (1 technical question, without escalation) | **45 phút** (n=1 — single data point) | With escalation to dev: +2-4h wait |
| Case study search + compile | **30 phút** (n=1 — single data point) | |
| Follow-up email sau meeting | **17 phút** (n=2 — small sample) | Range: 15-20 min |
| Proposals written per week (avg) | **1.5-2.5 proposals** (n=2) | 1 enterprise + 2-3 SME |
| Pricing/features lookup per session | **20-30 phút** (n=2 — small sample) | Gather from Notion + email + ask manager |

**Root cause for low AI adoption**: Generic AI không biết Bflow pricing tiers, case studies, product specs → phải paste toàn bộ context vào mỗi session

**Target (Bflow-aware AI)**:
| Metric | Target | Estimated Saving |
|--------|--------|-----------------|
| B2B proposal draft | ~1 giờ | ~1.75 giờ/proposal |
| RFP response (1 question) | ~5-10 phút | ~35-40 phút |
| Case study search | ~5 phút | ~25 phút |
| Follow-up email | ~5 phút | ~12 phút |

---

## Customer Success Baselines

*Source: [interviews-sales-cs.md](interviews-sales-cs.md) — n=1 (single data point), **do NOT extrapolate***

| Metric | Baseline (no AI) | Notes |
|--------|-----------------|-------|
| Ticket response — simple FAQ | **10 phút** (n=1 — single data point) | |
| Ticket response — technical issue (Bflow config/API) | **45 phút** (n=1 — single data point) | Search docs (20 min) + hỏi dev (20-25 min wait) |
| Tickets per week (per CS person) | **20-30 tickets** (n=1 — single data point) | |
| Escalation rate to dev team | **35%** (n=1 — single data point) | ~7-10 tickets/tuần escalate |
| Onboarding checklist creation (per client) | **2 giờ** (n=1 — single data point) | |
| Bflow docs search (per technical question) | **15-20 phút** (n=1 — single data point) | |

**Note**: CS n=1 — aim for CS Interview 2 before finalizing ROI calculations. Current data is directional only.

**Root cause**: Bflow technical knowledge gap → phải escalate 35% tickets lên dev team; dev team bị interrupt

**Target (Bflow-aware AI)**:
| Metric | Target | Estimated Saving |
|--------|--------|-----------------|
| Ticket response (technical) | ~15-20 phút | ~25-30 phút/ticket |
| Escalation rate | ~15-20% | ~15-20% reduction |
| Onboarding checklist | ~30 phút | ~1.5 giờ/client |

---

## Back Office + Marketing Baselines

*Source: [interviews-back-office.md](interviews-back-office.md) — n=2 (small sample), interpret with caution*

| Metric | Baseline (no AI) | Notes |
|--------|-----------------|-------|
| Meeting notes → full biên bản | **23 phút** (n=2 — small sample) | From rough notes post-meeting |
| Action items extraction from notes | **10 phút** (n=1 — An only) | |
| LinkedIn post draft | **50 phút** (n=1 — An only, single data point) | Research + draft + edit |
| Internal announcement | **19 phút** (n=2 — small sample) | |
| HR policy Q&A lookup | **10 phút** (n=1 — Thảo only, single data point) | Search doc + draft reply |
| Meetings per week (avg) | **3-4** (n=2 — small sample) | |
| HR questions received per week | **8-12** (n=1 — Thảo only) | Via Telegram |

**Root cause**: Không có company-specific knowledge base → AI generic output không usable; HR policy không searchable

**Target (với AI)**:
| Metric | Target | Estimated Saving |
|--------|--------|-----------------|
| Meeting notes | ~5 phút (paste rough → AI formats) | ~18 phút/cuộc họp |
| HR policy Q&A | ~2 phút (AI answers from RAG) | ~8 phút/câu |
| LinkedIn post | ~15 phút | ~35 phút/post |

---

## ROI Estimate

### Assumptions

| Input | Value | Source |
|-------|-------|--------|
| Avg hourly cost (blended MTS — all roles) | ~150,000 VND/giờ | PM estimate (~$6/h blended) |
| Working hours/day | 8h | |
| Working days/year | 240 | |
| AI response time target | <3s p95 | ADR-011 |
| Exchange rate | 25,000 VND/USD | Estimate |

**Note**: Hourly cost estimate chưa được confirm bởi [@cfo] — cần verify trước G0.1 submission nếu cần ROI chính xác.

### Time Savings per Role per Day (conservative estimate)

| Role | Time Saved/Day (est.) | People | Hours/Year | Value/Year (VND) | Confidence |
|------|----------------------|--------|------------|-----------------|-----------|
| Engineering | **30 min/day** (code review + PR desc + ADR) | **10** | **1,200h** | **180M VND** | Medium (n=3) |
| Sales | **45 min/day** (proposal + RFP + case study) | **4** | **720h** | **108M VND** | Low (n=2) |
| CS | **25 min/day** (ticket response + docs search) | **3** | **300h** | **45M VND** | Very Low (n=1) |
| Back Office | **20 min/day** (meeting notes + HR Q&A) | **5** | **400h** | **60M VND** | Low (n=2) |
| **Total (conservative)** | | **~22** | **~2,620h** | **~393M VND** | |
| **Total (optimistic)** | | **~27** | **~3,500h** | **~525M VND** | |

**Conservative total**: ~393M VND/năm (~$15,700 USD/năm)
**Optimistic total**: ~525M VND/năm (~$21,000 USD/năm)

*Note: Estimates based on small samples — treat as directional, not precise. Conservative estimate used for business case.*

### MTS-OpenClaw Operating Cost

| Item | Monthly | Annual |
|------|---------|--------|
| VPS (2-4 vCPU, 4-8GB) | ~$20-40 | ~$240-480 |
| Bflow AI-Platform API | $0/query (existing infra) | $0 |
| Claude API (Dev opt-in, shared Max 200 Plan) | ~$50-100 (shared) | ~$600-1,200 |
| **Total** | **~$70-140** | **~$840-1,680** |

**Payback period**: **< 1 month** (annual value $15,700 vs annual cost $1,680)
**ROI**: ~935% (conservative) to ~1,250% (optimistic)

*Note: Even at 50% of estimated savings, ROI remains strongly positive. Low-risk investment.*

---

## Adoption Signal

| Team | "Would use Telegram AI" | Primary use case | Confidence |
|------|------------------------|-----------------|-----------|
| Engineering | **1/3 Yes, 2/3 Maybe** | Code review với Bflow context | Medium (n=3) |
| Sales | **2/2 Yes** | Proposal draft + RFP | Low (n=2 — small sample) |
| CS | **1/1 Yes** | Technical ticket response | Very Low (n=1) |
| Back Office | **2/2 Yes** | Meeting notes + HR FAQ | Low (n=2 — small sample) |
| **Overall** | **6/8 Yes, 2/8 Maybe** | | |

**Accuracy concern**: Engineering (2/3) và CS (1/1) có accuracy concern — "nếu AI sai thì nguy hiểm hơn không dùng." Sales + Back Office ít concern hơn (lower stakes tasks).

---

## G0.1 Evidence Status

- [x] Engineering interviews complete (≥3) → baselines filled ✅ (n=3, medium confidence)
- [x] Sales + CS interviews complete (≥3 combined) → baselines filled ✅ (n=2 Sales + n=1 CS, noted)
- [x] Back Office interviews complete (≥2) → baselines filled ✅ (n=2, small sample noted)
- [x] ROI calculation complete (conservative estimate with confidence caveats)
- [x] Adoption signal clear — overall 6/8 Yes, 2/8 Maybe
- **Confidence level**: **Medium overall** (Engineering n=3 solid; Sales/CS/Back Office small samples noted per CTO directive)

---

*Owner: [@pm] | US-029-004 | Completed: 2026-03-04 | Data quality per [@cto] directive (Sprint 29 Day 1)*
