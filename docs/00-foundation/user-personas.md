# User Personas — MTClaw

**SDLC Stage**: 00-Foundation
**Version**: 1.0.0
**Date**: 2026-03-02
**Author**: [@pm]
**Source**: Sprint 29 interviews (n=8) + User Journey Map
**Framework**: SDLC 6.1.1 — Stage 00 Required Artifact (STANDARD tier: min 2 personas)

---

## Persona 1: Phú — Senior Developer (Engineering)

### Demographics

| Attribute | Value |
|-----------|-------|
| Name | Phú (composite persona) |
| Role | Senior Developer, MTS Engineering |
| Team | Engineering (4 people) |
| Experience | 3+ years at MTS, full-stack |
| Tech literacy | High (CLI, Git, Docker, APIs) |
| AI usage today | GitHub Copilot (code), chat.nqh occasionally |
| Channel preference | Telegram (daily active) |

### Goals

1. **Primary**: Get code review context fast — PR checklist, SQL injection check, RLS compliance
2. **Secondary**: Generate specs for features before implementation (avoid rework)
3. **Tertiary**: Quick lookup of Bflow API docs without leaving Telegram

### Pain Points (from interviews, n=3)

1. "chat.nhatquangholding.com không biết gì về Bflow API — trả lời generic" (3/3)
2. "Review PR mất 30-45 phút, AI có thể giúp scan trước" (2/3)
3. "Viết spec/user story không phải sở trường, nhưng phải làm" (2/3)
4. "Nếu AI trả lời sai về code thì nguy hiểm hơn không dùng" (2/3 — accuracy is adoption gate)

### Behavior Pattern

- Daily Telegram user (tech channels, bot interactions)
- Prefers **explicit** SOUL switching: `@reviewer`, `@coder`, `@pm`
- Expects <5s response time for AI — slower = abandon
- Will test accuracy aggressively before trusting
- High tolerance for English; switches Vietnamese/English naturally

### SOUL Assignment

| SOUL | Mode | Use Case |
|------|------|----------|
| `dev` | On-demand (via `@dev` or auto-detect) | Daily engineering tasks, Bflow API lookup |
| `reviewer` | On-demand (`@reviewer`) | PR review with structured checklist |
| `coder` | On-demand (`@coder`) | Code generation, bug fix assistance |
| `pm` | On-demand (`@pm` or `/spec`) | Spec writing, user stories |

### Success Criteria

| Metric | Baseline | Target (Sprint 6) |
|--------|----------|--------------------|
| Time for code review prep | 30-45 min | 10-15 min |
| Spec writing time | 2-3 hours | 30 min |
| AI accuracy trust | 0% (no AI review) | >80% (uses regularly) |
| WAU | 0 | Weekly active |

---

## Persona 2: Hương — Sales Executive

### Demographics

| Attribute | Value |
|-----------|-------|
| Name | Hương (composite persona) |
| Role | Sales Executive, MTS Sales Team |
| Team | Sales (3 people) |
| Experience | 2 years at MTS, B2B SaaS sales |
| Tech literacy | Medium (uses apps, not CLI; no Git) |
| AI usage today | ChatGPT occasionally for email drafts |
| Channel preference | Telegram + Zalo |

### Goals

1. **Primary**: Draft client proposals with correct Bflow pricing in minutes, not hours
2. **Secondary**: Find relevant case studies for client pitches
3. **Tertiary**: Quick answers about Bflow features for client questions

### Pain Points (from interviews, n=2, small sample)

1. "Mỗi proposal mất 1-2 giờ vì phải tìm pricing, features, case study từ nhiều nguồn" (2/2)
2. "Copy-paste từ proposal cũ rồi sửa — hay sai giá" (2/2)
3. "Khách hỏi feature mới mà mình chưa biết có hay không" (1/2)

### Behavior Pattern

- Telegram for internal, Zalo for some clients
- Prefers **automatic** SOUL routing — doesn't want to think about `@sales`
- Needs copy-paste friendly output (proposal → email → client)
- Vietnamese primary language for proposals
- Low tolerance for wrong pricing — one mistake = lost deal

### SOUL Assignment

| SOUL | Mode | Use Case |
|------|------|----------|
| `sales` | Auto-detect (via assistant delegation) | Proposals, pricing, case studies |
| `assistant` | Default entry point | Non-sales questions, general Q&A |

### Success Criteria

| Metric | Baseline | Target (Sprint 6) |
|--------|----------|--------------------|
| Proposal draft time | 1-2 hours | 15-20 min |
| Pricing accuracy | Manual (error-prone) | RAG-verified |
| Case study lookup | 25-30 min in Notion | <1 min |
| WAU | 0 | Weekly active |

### RAG Dependency

- **sales**: Bflow pricing tiers, feature comparison, packaging, case studies
- ⚠️ RAG collections required by Sprint 6 for full value delivery

---

## Persona 3: Thảo — HR Admin (Back Office)

### Demographics

| Attribute | Value |
|-----------|-------|
| Name | Thảo (composite persona) |
| Role | HR Admin, MTS Back Office |
| Team | Back Office (1 person handling HR + Admin) |
| Experience | 1.5 years at MTS |
| Tech literacy | Low (uses basic apps: email, spreadsheet, chat) |
| AI usage today | None — tried chat.nqh once, found it confusing |
| Channel preference | Telegram (follows colleague links) |

### Goals

1. **Primary**: Quick HR policy answers without searching through Google Drive
2. **Secondary**: Generate meeting minutes from rough notes
3. **Tertiary**: Draft routine documents (contracts, announcements)

### Pain Points (from interviews, n=2, small sample)

1. "Tìm chính sách HR trong Google Drive mất 15-20 phút" (2/2)
2. "Viết biên bản họp tốn thời gian, phải format đúng mẫu" (2/2)
3. "Thử chat.nqh nhưng không biết cách dùng, trả lời tiếng Anh nhiều quá" (1/2)
4. "Lo AI trả lời sai chính sách → nhân viên hiểu lầm" (1/2)

### Behavior Pattern

- Telegram user (messages colleagues, follows group channels)
- Needs **zero ceremony** — type question, get answer
- **Must** respond in Vietnamese — English = confusion
- Needs source citation ("theo tài liệu HR-Policy-2026.md") to build trust
- Will consult HR Manager for verification if answer seems off
- Session persistence critical — "nó nhớ tôi là HR Admin" across visits

### SOUL Assignment

| SOUL | Mode | Use Case |
|------|------|----------|
| `assistant` | Default entry point (`is_default=true`) | HR Q&A, meeting notes, general office — handles directly |
| `dev` | Auto-delegate (rare) | If Thảo asks technical question, assistant delegates to dev |

> **Note (CPO-OBS-1)**: 95%+ of Thảo's interactions stay within assistant. Delegation to dev is edge case but documented for completeness.

### Success Criteria

| Metric | Baseline | Target (Sprint 6) |
|--------|----------|--------------------|
| HR policy lookup time | 15-20 min | <1 min |
| Meeting minutes time | 30 min | 5 min |
| Vietnamese response rate | N/A | 100% |
| Trust (would-use-again) | No | Yes |
| WAU | 0 | Bi-weekly active |

### RAG Dependency

- **hr-policies**: HR policies, leave policy, benefits, org chart
- ⚠️ RAG accuracy = adoption gate — one wrong policy answer breaks trust

---

## Cross-Persona Insights

### Adoption Risk Matrix

| Persona | Adoption Risk | Blocker | Mitigation |
|---------|--------------|---------|------------|
| Phú (Engineering) | Low | AI accuracy for code | PR Gate WARNING mode first |
| Hương (Sales) | Medium | Wrong pricing | RAG curation + "confirm with manager" |
| Thảo (Back Office) | Medium-High | Complexity, language | Zero-ceremony UX, Vietnamese-only |

### SOUL Routing Priority

```
New user → /start → welcome message
  │
  ├─ Engineering (detected by context) → assistant delegates to dev
  ├─ Sales (detected by context: proposal, pricing) → assistant delegates to sales
  └─ Others (default) → assistant handles directly
```

### Design Implications for Sprint 3-4

1. **Onboarding must be <30 seconds** — especially for Thảo (non-tech)
2. **Vietnamese as default** — detect language from input, respond in same language
3. **Source citation mandatory** for RAG answers — builds trust across all personas
4. **SOUL switch invisible** — users don't know about SOULs, they just ask questions
5. **First 30 seconds define adoption** — if first response is wrong or slow, user abandons

---

## Evidence Trail

| Persona | Interview Source | Sample Size | Confidence |
|---------|-----------------|-------------|------------|
| Phú (Engineering) | `interviews-engineering.md` | n=3 | High |
| Hương (Sales) | `interviews-sales-cs.md` | n=2 | Medium (small sample) |
| Thảo (Back Office) | `interviews-back-office.md` | n=2 | Medium (small sample) |

---

## References

- [User Research: Engineering](user-research/interviews-engineering.md)
- [User Research: Sales/CS](user-research/interviews-sales-cs.md)
- [User Research: Back Office](user-research/interviews-back-office.md)
- [User Journey Map](../01-planning/user-journey-map.md)
- [Baseline Metrics](user-research/baseline-metrics.md)
