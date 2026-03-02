# Test Strategy — MTClaw

**SDLC Stage**: 01-Planning
**Version**: 1.0.0
**Date**: 2026-03-02
**Author**: [@pm], [@cto] (tiered targets)

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
| Governance | 8+ | **80%** | Mature codebase, full 3 Rails |

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

## E2E Tests (Critical Paths Only)

| Path | Description | Sprint |
|------|-------------|--------|
| Onboarding | New user → Telegram → first AI response | 4 |
| Delegation | User → @pm → /spec → JSON output | 4 |
| Multi-tenant | MTS user + NQH user concurrent | 6 |

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

---

**References**: [ADR-003: Observability](../02-design/01-ADRs/SPEC-0003-ADR-003-Observability-Architecture.md), [Requirements](requirements.md)
