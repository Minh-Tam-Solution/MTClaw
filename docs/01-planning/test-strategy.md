# Test Strategy — MTClaw

**SDLC Stage**: 01-Planning
**Version**: 2.1.0
**Date**: 2026-03-06 (Sprint 12: governance engine, evidence chain, spec quality, workspace commands)
**Author**: [@pm], [@cto] (tiered targets), [@tester] (Sprint 8-12 update)

---

## Philosophy

- Test real logic, not mocks (Zero Mock Policy)
- Tiered coverage targets — earn trust incrementally
- Integration tests = scenario checklists, not line percentage
- E2E = critical paths only

## Tiered Coverage Targets

| Phase | Sprint | Unit Coverage | Rationale |
|-------|--------|-------------|-----------|
| Foundation | 1-3 | **60%** | Establish baseline, learn Go patterns |
| Core Rails | 4-5 | **70%** | Critical paths for Spec Factory + PR Gate |
| Governance | 8-9 | **80%** | Mature codebase, full 3 Rails |
| Evidence + Quality | 10-12 | **80%** | Governance engine, evidence chain, spec quality |

## Unit Tests

- **Framework**: Go `testing` package + `testify/assert`
- **Naming**: `*_test.go` colocated with source
- **Run**: `make test` or `go test ./... -v`
- **Coverage**: `make test-coverage` → `coverage.html`

### What to Unit Test
- SOUL loading and caching logic
- Tenant context injection
- Cost guardrail calculations
- Request routing to SOUL
- Token usage tracking

### What NOT to Unit Test
- Database queries (integration test)
- Bflow AI-Platform responses (integration test)
- Telegram/Zalo message delivery (E2E test)
- MS Teams live Bot Framework API (integration/E2E — requires Azure AD credentials)

## Integration Tests

Scenario-based checklist (not line coverage):

| Scenario | Sprint | Priority |
|----------|--------|----------|
| Tenant isolation: User A cannot see User B data | 3 | P0 |
| SOUL loading: All 16 SOULs load at startup | 1 | P0 |
| SOUL cache: Checksum mismatch triggers reload | 2 | P1 |
| Bflow AI: Request → AI-Platform → response | 4 | P0 |
| Bflow AI: Fallback on AI-Platform timeout | 4 | P1 |
| Cost guardrail: Reject at 100% monthly limit | 3 | P1 |
| Spec Factory: `/spec` → JSON output | 4 | P0 |
| PR Gate: WARNING mode evaluation | 5 | P0 |
| Evidence: Governance action creates audit record | 5 | P0 |
| Multi-tenant concurrent: 2 tenants simultaneous | 5 | P1 |
| MS Teams: inbound message → bus publish → SOUL routing | 10 | P0 |
| MS Teams: JWT verification (valid/expired/wrong iss/wrong aud) | 10 | P0 |
| MS Teams: channel column written to governance tables | 10 | P1 |
| MS Teams: `MSTEAMS_APP_PASSWORD` not in logs | 10 | P0 (security) |
| MS Teams + Telegram: cross-channel /spec produces same output | 10 | P1 |
| PR Gate ENFORCE: fail verdict blocks merge | 8 | P0 |
| GitHub webhook: HMAC signature verification + PR inbound | 8 | P0 |
| Spec quality: 5-dimension scoring threshold (70) | 12 | P0 |
| Design-first gate: coder blocked without approved spec | 12 | P0 |
| Evidence chain: spec -> pr_gate link -> chain build | 12 | P0 |
| Evidence linker: auto-link spec to PR by session key | 12 | P0 |
| Audit PDF: spec + evidence chain -> valid PDF export | 8 | P1 |
| Workspace show: `/workspace` returns current agent directory | 12 | P1 |
| Workspace switch: `/workspace <path>` updates agent + cache invalidation | 12 | P0 |
| Workspace invalid path: returns error, no state change | 12 | P1 |
| Projects list: `/projects` lists siblings, marks current | 12 | P1 |

## E2E Tests (Critical Paths Only)

| Path | Description | Sprint |
|------|-------------|--------|
| Onboarding | New user → Telegram → first AI response | 4 |
| Delegation | User → @pm → /spec → JSON output | 4 |
| Multi-tenant | MTS user + NQH user concurrent | 6 |
| MS Teams full flow | Teams message → SOUL → Adaptive Card reply | 10 (manual, requires Azure AD) |
| PR Gate flow | GitHub PR → webhook → @reviewer → verdict → evidence link | 12 |
| Spec quality gate | /spec → quality scoring → accept/reject → evidence chain | 12 |
| Design gate | @coder task → design gate check → spec required | 12 |
| Audit trail | Spec → PR review → chain build → PDF export | 12 |
| Channel cleanup | Verify Discord/Feishu/WhatsApp removed cleanly | 9 |
| Workspace flow | /workspace show -> /projects -> /workspace switch -> tools use new dir | 12 |

## CI/CD Integration

```yaml
# GitHub Actions gate
- make test           # Unit tests
- make test-coverage  # Coverage report
- make souls-validate # SOUL frontmatter check
- make build          # Binary compiles
```

## Zero Mock Exception

Per [@cto] directive: Unit tests for RAG client may use mocked HTTP responses (documented CI exception) since Bflow AI-Platform is external dependency. All mocks must be:
- Documented in test file header
- Based on real API response format
- Tagged with `// CI_MOCK_EXCEPTION: Bflow AI-Platform`

**MS Teams exception** (Sprint 10): Unit tests use `httptest.NewServer` mock for Bot Framework token endpoint and API endpoint. This is not a mock — it is a real HTTP server in the test process (Zero Mock Policy compliant). Bot Framework OpenID metadata fetch is bypassed by injecting RSA keys directly into `jwksCache` via `injectTestKey()`. Tagged with `// CI_MOCK_EXCEPTION: Bot Framework live endpoint (Azure AD creds required for E2E)`.

**Test plan locations**:
- `docs/05-test/MASTER-TEST-PLAN.md` (Sprint 1-12 cumulative, includes workspace commands)
- `docs/05-test/test-plan-msteams-sprint10.md` (MS Teams detail)

---

**References**: [ADR-003: Observability](../02-design/01-ADRs/SPEC-0003-ADR-003-Observability-Architecture.md), [Requirements](requirements.md)
