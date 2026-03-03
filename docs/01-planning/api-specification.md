# API Specification — MTClaw

**SDLC Stage**: 01-Planning
**Version**: 1.0.0
**Date**: 2026-03-02
**Author**: [@pm] + [@researcher]
**Base**: GoClaw existing API + MTClaw governance extensions

---

## 1. Overview

MTClaw's API consists of two layers:

1. **GoClaw Core API** — inherited from upstream (agent management, sessions, traces, etc.)
2. **MTClaw Governance API** — new endpoints for 3 Rails governance

### Base URL

```
http://localhost:8080/v1/
```

### Authentication

| Method | Header | Usage |
|--------|--------|-------|
| Bearer Token | `Authorization: Bearer {token}` | All API endpoints |
| User ID | `X-GoClaw-User-Id: {user_id}` | User context injection |
| Agent ID | `X-GoClaw-Agent-Id: {agent_id}` | Agent context override |

---

## 2. GoClaw Core API (Inherited)

### 2.1 Agents (SOUL Management)

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/v1/agents` | List all agents (SOULs) for current owner |
| `POST` | `/v1/agents` | Create new agent (SOUL) |
| `GET` | `/v1/agents/{id}` | Get agent detail |
| `PUT` | `/v1/agents/{id}` | Update agent config |
| `DELETE` | `/v1/agents/{id}` | Soft-delete agent |
| `GET` | `/v1/agents/{id}/shares` | List agent share grants |
| `POST` | `/v1/agents/{id}/shares` | Share agent with user |
| `DELETE` | `/v1/agents/{id}/shares/{userID}` | Revoke agent share |
| `POST` | `/v1/agents/{id}/regenerate` | Regenerate agent context |
| `POST` | `/v1/agents/{id}/resummon` | Re-summon agent |

**MTClaw usage**: Agents table = 16 SOULs. CRUD via this API for SOUL management.

### 2.2 Providers (LLM Configuration)

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/v1/providers` | List LLM providers |
| `POST` | `/v1/providers` | Register new provider |
| `GET` | `/v1/providers/{id}` | Get provider detail |
| `PUT` | `/v1/providers/{id}` | Update provider config |
| `DELETE` | `/v1/providers/{id}` | Remove provider |
| `GET` | `/v1/providers/{id}/models` | List available models |
| `POST` | `/v1/providers/{id}/verify` | Verify provider connectivity |

**MTClaw usage**: Register Bflow AI-Platform as primary provider.

### 2.3 Skills

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/v1/skills` | List all skills |
| `POST` | `/v1/skills/upload` | Upload new skill |
| `GET` | `/v1/skills/{id}` | Get skill detail |
| `PUT` | `/v1/skills/{id}` | Update skill |
| `DELETE` | `/v1/skills/{id}` | Delete skill |
| `POST` | `/v1/skills/{id}/grants/agent` | Grant skill to agent |
| `DELETE` | `/v1/skills/{id}/grants/agent/{agentID}` | Revoke agent skill |
| `POST` | `/v1/skills/{id}/grants/user` | Grant skill to user |
| `DELETE` | `/v1/skills/{id}/grants/user/{userID}` | Revoke user skill |

**MTClaw usage**: Register spec-factory skill, governance skills.

### 2.4 Tools

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/v1/tools/builtin` | List built-in tools |
| `GET` | `/v1/tools/builtin/{name}` | Get built-in tool detail |
| `PUT` | `/v1/tools/builtin/{name}` | Update built-in tool config |
| `GET` | `/v1/tools/custom` | List custom tools |
| `POST` | `/v1/tools/custom` | Create custom tool |
| `GET` | `/v1/tools/custom/{id}` | Get custom tool detail |
| `PUT` | `/v1/tools/custom/{id}` | Update custom tool |
| `DELETE` | `/v1/tools/custom/{id}` | Delete custom tool |

### 2.5 MCP Servers (External Tools)

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/v1/mcp/servers` | List MCP servers |
| `POST` | `/v1/mcp/servers` | Register MCP server |
| `GET` | `/v1/mcp/servers/{id}` | Get MCP server detail |
| `PUT` | `/v1/mcp/servers/{id}` | Update MCP server |
| `DELETE` | `/v1/mcp/servers/{id}` | Delete MCP server |
| `POST` | `/v1/mcp/servers/{id}/grants/agent` | Grant MCP access to agent |
| `DELETE` | `/v1/mcp/servers/{id}/grants/agent/{agentID}` | Revoke agent MCP access |
| `GET` | `/v1/mcp/grants/agent/{agentID}` | List agent MCP grants |
| `POST` | `/v1/mcp/servers/{id}/grants/user` | Grant MCP access to user |
| `DELETE` | `/v1/mcp/servers/{id}/grants/user/{userID}` | Revoke user MCP access |
| `POST` | `/v1/mcp/requests` | Create MCP tool request |
| `GET` | `/v1/mcp/requests` | List pending MCP requests |
| `POST` | `/v1/mcp/requests/{id}/review` | Review MCP request |

### 2.6 Channels

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/v1/channels/instances` | List channel instances |
| `POST` | `/v1/channels/instances` | Create channel instance |
| `GET` | `/v1/channels/instances/{id}` | Get channel detail |
| `PUT` | `/v1/channels/instances/{id}` | Update channel config |
| `DELETE` | `/v1/channels/instances/{id}` | Delete channel |

**MTClaw usage**: Telegram (Phase 1), Zalo (Phase 2).

### 2.7 Traces & Observability

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/v1/traces` | List traces (filterable by agent, user, session, status) |
| `GET` | `/v1/traces/{traceID}` | Get trace detail with spans |

**Query params**: `agent_id`, `user_id`, `session_key`, `status`, `limit`, `offset`

### 2.8 Delegations

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/v1/delegations` | List delegation history |
| `GET` | `/v1/delegations/{id}` | Get delegation detail |

### 2.9 Chat Completions (OpenAI-Compatible)

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/v1/chat/completions` | OpenAI-compatible chat endpoint |

**MTClaw usage**: Bflow AI-Platform forwards to this format.

---

## 3. MTClaw Governance API (New — Sprint 4+)

### 3.1 Spec Factory (Rail #1) — Sprint 4

| Method | Endpoint | Description | Sprint |
|--------|----------|-------------|--------|
| `POST` | `/v1/governance/specs` | Generate spec from natural language | Sprint 4 |
| `GET` | `/v1/governance/specs` | List generated specs | Sprint 4 |
| `GET` | `/v1/governance/specs/{specID}` | Get spec detail | Sprint 4 |
| `PUT` | `/v1/governance/specs/{specID}` | Update spec (approve/modify) | Sprint 7 |
| `POST` | `/v1/governance/specs/{specID}/attach` | Attach evidence to spec | Sprint 7 |

#### POST /v1/governance/specs

**Request**:
```json
{
  "description": "Create login feature for Bflow mobile app",
  "soul": "pm",
  "language": "vi",
  "format": "prototype"
}
```

**Response (Sprint 4 Prototype)**:
```json
{
  "spec_version": "0.1.0",
  "title": "Đăng nhập Bflow Mobile",
  "narrative": {
    "as_a": "nhân viên Bflow",
    "i_want": "đăng nhập vào mobile app",
    "so_that": "truy cập ERP từ điện thoại"
  },
  "acceptance_criteria": [
    "Given có tài khoản Bflow, When nhập đúng email/password, Then đăng nhập thành công",
    "Given sai password 3 lần, When thử lần thứ 4, Then tài khoản bị khóa 15 phút"
  ],
  "priority": "P1",
  "estimated_effort": "M",
  "soul_author": "pm",
  "trace_id": "abc-123-def",
  "created_at": "2026-03-15T10:30:00Z"
}
```

### 3.2 PR Gate (Rail #2) — Sprint 5

| Method | Endpoint | Description | Sprint |
|--------|----------|-------------|--------|
| `POST` | `/v1/governance/pr-reviews` | Submit PR for review | Sprint 5 |
| `GET` | `/v1/governance/pr-reviews` | List PR reviews | Sprint 5 |
| `GET` | `/v1/governance/pr-reviews/{id}` | Get review detail | Sprint 5 |

#### POST /v1/governance/pr-reviews

**Request**:
```json
{
  "pr_url": "https://github.com/org/repo/pull/42",
  "soul": "reviewer",
  "mode": "WARNING",
  "checks": ["sql_injection", "rls_compliance", "test_coverage"]
}
```

**Response**:
```json
{
  "review_id": "rev-001",
  "pr_url": "https://github.com/org/repo/pull/42",
  "mode": "WARNING",
  "verdict": "PASS_WITH_WARNINGS",
  "score": 75,
  "findings": [
    {
      "severity": "WARNING",
      "category": "security",
      "message": "No RLS policy on new table",
      "file": "migrations/008_new_feature.sql",
      "line": 15
    }
  ],
  "soul_author": "reviewer",
  "trace_id": "def-456-ghi",
  "created_at": "2026-04-01T14:00:00Z"
}
```

### 3.3 Knowledge & Answering (Rail #3) — Sprint 6

| Method | Endpoint | Description | Sprint |
|--------|----------|-------------|--------|
| `POST` | `/v1/governance/knowledge/query` | RAG query with SOUL context | Sprint 6 |
| `POST` | `/v1/governance/knowledge/ingest` | Ingest documents for RAG | Sprint 6 |
| `GET` | `/v1/governance/knowledge/collections` | List RAG collections | Sprint 6 |

#### POST /v1/governance/knowledge/query

**Request**:
```json
{
  "query": "Quy trình xử lý khiếu nại khách hàng",
  "soul": "cs",
  "collections": ["hr-policies", "engineering"],
  "max_results": 5
}
```

**Response**:
```json
{
  "answer": "Quy trình xử lý khiếu nại gồm 5 bước...",
  "sources": [
    {
      "document": "SOP-CS-001.md",
      "section": "Quy trình khiếu nại",
      "relevance_score": 0.92
    }
  ],
  "soul_author": "cs",
  "trace_id": "ghi-789-jkl"
}
```

### 3.4 Evidence Trail — Sprint 6

| Method | Endpoint | Description | Sprint |
|--------|----------|-------------|--------|
| `GET` | `/v1/governance/evidence` | List evidence records | Sprint 6 |
| `GET` | `/v1/governance/evidence/{id}` | Get evidence detail | Sprint 6 |
| `POST` | `/v1/governance/evidence/export` | Export evidence for audit | Sprint 8 |

### 3.5 SOUL Management (Extended) — Sprint 4

| Method | Endpoint | Description | Sprint |
|--------|----------|-------------|--------|
| `GET` | `/v1/governance/souls` | List SOULs with quality scores | Sprint 4 |
| `GET` | `/v1/governance/souls/{key}/metrics` | Get SOUL quality metrics | Sprint 4 |
| `POST` | `/v1/governance/souls/validate` | Validate SOUL frontmatter | Sprint 3 |

### 3.6 Tenant Cost (ADR-003) — Sprint 3

| Method | Endpoint | Description | Sprint |
|--------|----------|-------------|--------|
| `GET` | `/v1/governance/costs` | Get tenant cost summary | Sprint 3 |
| `GET` | `/v1/governance/costs/daily` | Daily cost breakdown | Sprint 3 |
| `GET` | `/v1/governance/costs/by-soul` | Cost per SOUL | Sprint 4 |
| `PUT` | `/v1/governance/costs/limits` | Update tenant cost limits | Sprint 3 |

---

## 4. API Endpoint Summary

### Total Endpoint Count

| Category | Inherited | New (Governance) | Total |
|----------|-----------|-----------------|-------|
| Agents (SOULs) | 10 | 3 | 13 |
| Providers | 7 | 0 | 7 |
| Skills | 9 | 0 | 9 |
| Tools | 8 | 0 | 8 |
| MCP | 11 | 0 | 11 |
| Channels | 5 | 0 | 5 |
| Traces | 2 | 0 | 2 |
| Delegations | 2 | 0 | 2 |
| Chat | 1 | 0 | 1 |
| **Governance** | 0 | **15** | **15** |
| **Total** | **55** | **18** | **73** |

### Phased Delivery

| Sprint | New Endpoints | Category |
|--------|--------------|----------|
| Sprint 3 | 4 | Tenant costs, SOUL validation |
| Sprint 4 | 6 | Spec factory, SOUL metrics |
| Sprint 5 | 3 | PR Gate |
| Sprint 6 | 5 | Knowledge/RAG, Evidence |

---

## 5. Error Codes

| Code | Meaning | Example |
|------|---------|---------|
| 400 | Bad Request | Invalid spec format, missing required fields |
| 401 | Unauthorized | Missing or invalid Bearer token |
| 403 | Forbidden | Cross-tenant access attempt (RLS) |
| 404 | Not Found | Agent/spec/review not found |
| 409 | Conflict | Duplicate agent_key, spec_id collision |
| 422 | Unprocessable | SOUL validation failed |
| 429 | Rate Limited | Tenant daily request limit exceeded |
| 500 | Internal Error | LLM provider timeout, DB error |
| 503 | Service Unavailable | Bflow AI-Platform unreachable (graceful degradation) |

---

## 6. Performance Targets (NFR)

| Endpoint Category | p95 Target | Notes |
|-------------------|-----------|-------|
| Non-AI endpoints | <100ms | Agent CRUD, traces list |
| AI endpoints | <5s | Spec generation, PR review, RAG query |
| Cost queries | <200ms | Aggregate queries with caching |
| Evidence export | <10s | Depends on data volume |

---

## References

- GoClaw HTTP handlers: `internal/http/*.go`
- FR-003: 3 Rails Governance (`docs/01-planning/requirements.md`)
- ADR-003: Observability (`docs/02-design/01-ADRs/SPEC-0003-ADR-003-Observability-Architecture.md`)
- Spec Command Design: `docs/02-design/spec-command-design.md`
- RLS Design: `docs/02-design/rls-tenant-isolation-design.md`
