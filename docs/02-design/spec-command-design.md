# /spec Command Design — MTClaw Rail #1: Spec Factory

**SDLC Stage**: 02-Design
**Version**: 1.0.0
**Date**: 2026-03-02
**Author**: [@pm] + [@researcher]
**Implements**: FR-003 Rail #1 (Spec Factory), US-012

---

## 1. Overview

The `/spec` command is MTClaw's first governance rail — transforming natural language requirements into structured, auditable specifications with evidence attachment.

### Phased Delivery

| Sprint | Scope | Output |
|--------|-------|--------|
| **Sprint 4** (prototype) | `/spec` → structured JSON | Basic spec with title, narrative, acceptance criteria |
| **Sprint 7** (full) | Full spec factory | spec_id, BDD scenarios, risk scoring, evidence vault link |

---

## 2. Architecture Decision: Skill-Based Approach

[@researcher]: GoClaw analysis reveals two implementation paths:

| Approach | Pros | Cons |
|----------|------|------|
| **A: Telegram command handler** | Fast, simple | Channel-coupled, no hot-reload, hard to version |
| **B: Skill-based** | Reusable, hot-reload, Git-versioned, SOUL-routable | Slightly more setup |

**Decision**: **Skill-based approach (B)** — aligns with governance-first principle. The skill lives as a Git-tracked markdown file, version-controlled, auditable.

### How It Works

```
User types: /spec Create login feature for Bflow mobile app
  │
  ▼
1. Telegram command handler detects /spec prefix
   → Publishes InboundMessage with metadata: {command: "spec", soul: "pm"}
  │
  ▼
2. Agent loop receives message
   → Routes to PM SOUL (or current SOUL delegates to PM)
   → Skills system detects "spec-factory" skill applies
  │
  ▼
3. PM SOUL reads skill instructions (SKILL.md)
   → Follows structured output template
   → Generates JSON spec
  │
  ▼
4. Output:
   a) User sees: Formatted spec summary in Telegram
   b) Agent generates: JSON spec artifact
   c) Evidence: Spec attached to traces table with trace_id
```

---

## 3. Skill Definition

### 3.1 File Location

```
docs/08-collaborate/skills/spec-factory/SKILL.md
```

(Also deployable to GoClaw skills directory at runtime)

### 3.2 SKILL.md Content

```markdown
---
name: spec-factory
description: Generate structured specifications from natural language requirements. Governance Rail #1.
---

# Spec Factory — Governance Rail #1

## When This Skill Activates

- User sends `/spec` command
- User asks to "create a spec", "write requirements", "generate a user story"
- PM SOUL receives a requirements-related request

## Output Format

Generate a JSON specification following this schema:

### Sprint 4 (Prototype)

{
  "spec_version": "0.1.0",
  "title": "Short descriptive title",
  "narrative": {
    "as_a": "role",
    "i_want": "feature/capability",
    "so_that": "business value"
  },
  "acceptance_criteria": [
    "Given X, When Y, Then Z",
    "Given A, When B, Then C"
  ],
  "priority": "P0|P1|P2|P3",
  "estimated_effort": "S|M|L|XL",
  "soul_author": "pm",
  "created_at": "ISO 8601 timestamp"
}

### Sprint 7 (Full)

{
  "spec_version": "1.0.0",
  "spec_id": "SPEC-YYYY-NNNN",
  "title": "Short descriptive title",
  "narrative": {
    "as_a": "role",
    "i_want": "feature/capability",
    "so_that": "business value"
  },
  "acceptance_criteria": [
    {
      "scenario": "Happy path",
      "given": "precondition",
      "when": "action",
      "then": "expected result"
    }
  ],
  "technical_requirements": ["requirement 1", "requirement 2"],
  "risks": [
    {
      "description": "risk description",
      "probability": "low|medium|high",
      "impact": "low|medium|high",
      "mitigation": "mitigation plan"
    }
  ],
  "dependencies": ["SPEC-YYYY-NNNN"],
  "bdd_scenarios": ["Feature: ...\n  Scenario: ..."],
  "priority": "P0|P1|P2|P3",
  "estimated_effort": "S|M|L|XL",
  "soul_author": "pm",
  "evidence_id": "trace_id reference",
  "created_at": "ISO 8601 timestamp"
}

## Process Steps

1. **Clarify** (if input is vague):
   - Ask 1-2 targeted questions maximum
   - Do NOT ask more than 2 questions — generate best-effort spec instead

2. **Generate**:
   - Create spec JSON following the schema above
   - Use Vietnamese for narrative if user wrote in Vietnamese
   - Use BDD format (Given/When/Then) for acceptance criteria

3. **Present**:
   - Show formatted summary to user (not raw JSON)
   - Format: Title, Narrative, Acceptance Criteria list
   - Ask: "Approve, modify, or discard?"

4. **Record**:
   - On approval: Save spec as evidence (write_file to workspace)
   - Link to trace_id for audit trail

## Boundaries

- This skill generates SPECS only — not code, not designs, not test plans
- If user asks for implementation → delegate to @coder
- If user asks for architecture → delegate to @architect
- If user asks for test cases → delegate to @tester

## Vietnamese Support

- Input in Vietnamese → output in Vietnamese
- Input in English → output in English
- Mixed → follow user's primary language

## Quality Criteria (from SOUL Quality Rubric)

- Correctness: Does the spec answer what the user asked? (weight: 30%)
- Completeness: Are all acceptance criteria covered? (weight: 20%)
- Role Alignment: PM-style output, not code/design? (weight: 20%)
- Efficiency: Reasonable length, no filler? (weight: 15%)
- Safety: No hallucinated requirements? Sources cited? (weight: 15%)
```

---

## 4. Command Handler Integration

### 4.1 Telegram Command (/spec)

In `internal/channels/telegram/commands.go`, the `/spec` command follows the "publish to agent loop" pattern:

```
case "/spec":
    // Extract task description after /spec
    taskText = strings.TrimPrefix(text, "/spec ")

    // Publish to agent loop with spec metadata
    Publish InboundMessage:
      Channel:  "telegram"
      Content:  taskText
      Metadata: { "command": "spec", "rail": "spec-factory" }

    // Send acknowledgment
    Reply: "Generating spec..."
    return true
```

### 4.2 Agent Loop Processing

The agent loop receives the message and:

1. Detects `metadata.command == "spec"` → ensures PM SOUL is active (or delegates to PM)
2. Skills system matches "spec-factory" skill
3. Agent reads SKILL.md instructions
4. Agent generates structured output following the schema
5. Agent presents formatted summary to user

### 4.3 SOUL Routing for /spec

| Current SOUL | Action |
|-------------|--------|
| `pm` | Handle directly (primary owner of /spec) |
| `assistant` | Delegate to `pm` via spawn(agent="pm") |
| `coder` | Delegate to `pm` |
| `reviewer` | Delegate to `pm` |
| Any other | Delegate to `pm` |

---

## 5. Evidence Attachment

### 5.1 Evidence Record

Every /spec invocation produces an evidence record linked via `trace_id`:

```
traces table:
  trace_id: UUID
  agent_id: pm SOUL agent UUID
  name: "spec-factory"
  input_preview: "Create login feature for..."
  output_preview: "{spec JSON first 500 chars}"
  metadata: {
    "command": "spec",
    "spec_version": "0.1.0",
    "spec_title": "Login Feature for Bflow Mobile"
  }
```

### 5.2 Spec Storage (Sprint 7)

```
Spec files stored as workspace files:
  specs/SPEC-2026-0001.json  → Full JSON spec
  specs/SPEC-2026-0001.md    → Human-readable markdown version

Queryable via:
  SELECT * FROM traces
  WHERE name = 'spec-factory'
  AND metadata->>'spec_id' = 'SPEC-2026-0001'
```

---

## 6. Example Interaction

### 6.1 Sprint 4 Prototype

```
User: /spec Tạo tính năng đăng nhập cho Bflow mobile app

MTClaw (PM SOUL):

📋 Spec: Đăng nhập Bflow Mobile

Narrative:
  Với tư cách là nhân viên Bflow
  Tôi muốn đăng nhập vào mobile app
  Để truy cập ERP từ điện thoại

Acceptance Criteria:
  ✅ Given có tài khoản Bflow, When nhập đúng email/password, Then đăng nhập thành công
  ✅ Given sai password 3 lần, When thử lần thứ 4, Then tài khoản bị khóa 15 phút
  ✅ Given đăng nhập thành công, When mở app lần sau, Then auto-login (JWT refresh)

Priority: P1 | Effort: M

Approve, modify, or discard?
```

### 6.2 Sprint 7 Full

```
User: /spec Add SSO integration for enterprise clients

MTClaw (PM SOUL):

📋 SPEC-2026-0042: SSO Integration for Enterprise Clients

Narrative:
  As an enterprise admin
  I want SSO login via SAML 2.0 / OIDC
  So that employees use corporate credentials

Acceptance Criteria:
  ✅ Given SAML 2.0 IDP configured, When user clicks SSO login, Then redirected to IDP
  ✅ Given valid SAML assertion, When IDP redirects back, Then user authenticated + session created
  ✅ Given OIDC provider configured, When user clicks SSO, Then OAuth 2.0 flow completes
  ✅ Given SSO disabled for tenant, When user visits login, Then standard login shown

Risks:
  ⚠️ SAML assertion replay attack (Medium/High) → Validate assertion timestamps + one-time use
  ⚠️ IDP downtime (Low/Medium) → Fallback to standard login with admin override

BDD Scenarios: 4 generated
Evidence: Attached to trace abc-123-def

Priority: P1 | Effort: L | Dependencies: SPEC-2026-0038

Approve, modify, or discard?
```

---

## 7. Implementation Timeline

| Sprint | Task | Effort |
|--------|------|--------|
| **Sprint 4 Day 1** | Create SKILL.md for spec-factory | 0.5 day |
| **Sprint 4 Day 1-2** | /spec command handler in Telegram | 1 day |
| **Sprint 4 Day 2-3** | PM SOUL enhancement (spec generation instructions) | 1 day |
| **Sprint 4 Day 3-4** | Evidence attachment via trace_id | 1 day |
| **Sprint 4 Day 5** | Integration test: /spec → JSON → evidence | 0.5 day |
| **Sprint 7** | Full spec: spec_id, BDD, risk, evidence vault | 3 days |
| **Sprint 7** | Spec query API + dashboard widget | 2 days |

---

## 8. Success Metrics

| Metric | Sprint 4 Target | Sprint 7 Target |
|--------|----------------|----------------|
| Spec generation time | <30s (p95) | <30s (p95) |
| JSON schema validity | 90% valid on first attempt | 99% valid |
| User approval rate | 50% approve without modification | 70% |
| Evidence capture | 100% of /spec invocations traced | 100% |
| Vietnamese support | Basic (narrative only) | Full (all fields) |

---

## References

- FR-003: 3 Rails Governance (`docs/01-planning/requirements.md`)
- SOUL Quality Rubric: `docs/01-planning/soul-quality-rubric.md`
- GoClaw Skills System: `internal/skills/loader.go`, `internal/skills/search.go`
- GoClaw Command Handler: `internal/channels/telegram/commands.go`
- GoClaw Tool Interface: `internal/tools/registry.go`
- SOUL Loading Plan: `docs/02-design/soul-loading-implementation-plan.md`
