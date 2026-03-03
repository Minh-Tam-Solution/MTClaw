# Sprint 5 — MTS Pilot + PR Gate WARNING (Rail #2)

**SDLC Stage**: 04-Build
**Version**: 1.0.0
**Date**: 2026-03-03
**Author**: [@pm] + [@architect]
**Framework**: SDLC Enterprise Framework 6.1.1
**Tier**: STANDARD

---

## Sprint Summary

| Field | Value |
|-------|-------|
| Sprint | 5 |
| Goal | MTS Pilot live (10 employees) + PR Gate WARNING (Rail #2) + G3 gate |
| Duration | 5 days |
| Owner | [@coder] (implementation) + [@pm] (pilot ops) + [@devops] (deploy) |
| Points | ~13 |
| Gate | **G3 — Build Ready** |
| Predecessor | Sprint 4 ✅ (CTO 9.0/10 APPROVED) |

---

## Entry Criteria

All met:

- [x] Sprint 4 complete — CTO 9.0/10 APPROVED
- [x] /spec command working (Rail #1 prototype)
- [x] Context Anchoring Layer A implemented
- [x] @mention SOUL routing implemented
- [x] Evidence metadata (TraceName + TraceTags) flowing to traces
- [x] IT Admin SOUL seeded (17 SOULs total)
- [x] `make souls-validate` passing
- [x] Bflow AI-Platform verified (aip_c786 key active)
- [ ] Telegram bot registered via BotFather (operational — Day 1 prerequisite)

---

## Sprint Goal

> **MTS employees using MTClaw daily via Telegram** + **PR Gate processes first real PR in WARNING mode** + **G3 Build Ready** gate approval.

Sprint 5 bridges prototype → production pilot. Three parallel tracks:
1. **PR Gate (Rail #2)** — `/review` command, reviewer SOUL, WARNING report
2. **Staging Deploy** — Docker Compose on MTS VPS, ai-net bridge
3. **MTS Pilot** — 10 employees onboarded, measuring WAU

---

## User Stories

### US-027: PR Gate Skill + /review Command (P0, 4 pts)

**As a** MTS developer,
**I want to** send `/review <PR_URL>` in Telegram,
**So that** the reviewer SOUL evaluates the PR and reports issues.

**Acceptance Criteria**:
- Given `/review` with no URL → usage example displayed
- Given `/review https://github.com/org/repo/pull/123` → "🔍 Reviewing PR..." ack
- Given valid PR URL → reviewer SOUL fetches diff, evaluates policy rules, reports findings
- Given findings → report includes: issues (WARN/INFO), suggestions, summary table
- Given any review → trace record created with name='pr-gate', tags=['rail:pr-gate']
- WARNING mode: report only, never blocks merge

**Tasks**:
1. Create `docs/08-collaborate/skills/pr-gate/SKILL.md` — reviewer SOUL instruction
2. Add `/review` case to `commands.go` — same pattern as `/spec`
3. Add `/review` to `/help` text
4. Validate: trace record created with correct name/tags

**Design**: [PR Gate Design](../../02-design/pr-gate-design.md) Section 3

---

### US-028: MTS Staging Deployment (P0, 2 pts)

**As a** MTS DevOps,
**I want to** deploy MTClaw to MTS VPS via Docker Compose,
**So that** the Telegram bot is accessible 24/7 for pilot users.

**Acceptance Criteria**:
- Given Docker Compose → `goclaw` container starts successfully
- Given ai-net bridge → Bflow AI-Platform reachable at `http://ai-platform:8120`
- Given Telegram token → bot appears online and responds to `/start`
- Given PostgreSQL → all 12 migrations applied, 17 agents queryable
- Given Prometheus → basic metrics exported (optional, stretch)

**Tasks**:
1. Register Telegram bot via BotFather (manual, Day 1)
2. Configure `.env` on VPS with real credentials
3. Run `docker compose -f ... up -d --build`
4. Verify: `/start`, `/help`, `/spec`, `/review` all respond
5. Verify: `GET /v1/agents` returns 17 agents
6. (Stretch) Add Prometheus scrape config to `docker-compose.otel.yml`

**Note**: `docker-compose.mts.yml` already exists from Sprint 3. No new compose file needed — just deploy.

---

### US-029: Integration Tests — Critical Paths (P1, 3 pts)

**As a** CTO,
**I want** integration tests for critical paths,
**So that** we have automated regression before G3.

**Acceptance Criteria**:
- Given test suite → `make test-integration` runs scenarios against real PostgreSQL
- Given tenant isolation → User A query CANNOT return User B data
- Given SOUL routing → `@reviewer` routes to reviewer agent, `@pm` to PM agent
- Given /spec flow → command metadata flows through to trace record
- Given /review flow → reviewer SOUL produces valid review report format
- Given AI-Platform unavailable → graceful degradation (error message, not crash)

**Tasks**:
1. Create `internal/integration_test.go` (or `cmd/integration_test.go`)
2. Test: tenant isolation via RLS (`SET LOCAL app.tenant_id`)
3. Test: @mention routing resolves correct agent
4. Test: /spec metadata → TraceName='spec-factory'
5. Test: /review metadata → TraceName='pr-gate'
6. Test: AI-Platform timeout → graceful error

**Target**: 70% unit coverage (per test strategy Sprint 4-5 tier)

---

### US-030: Token Cost Tracking Verification (P1, 1 pt)

**As a** CTO,
**I want** token usage queryable per tenant per SOUL,
**So that** we can track cost before G3.

**Acceptance Criteria**:
- Given /spec invocation → `traces.total_input_tokens` > 0
- Given /review invocation → `traces.total_output_tokens` > 0
- Given query → `SELECT agent_key, SUM(total_input_tokens + total_output_tokens) FROM traces WHERE tenant_id = 'mts' GROUP BY agent_key` returns results
- CTO ISSUE-B from Sprint 3: validate traces table is sufficient (defer token_usage table if volume low)

**Tasks**:
1. After staging deploy, run 5+ /spec and /review invocations
2. Query traces table for token counts
3. Document findings (sufficient vs need dedicated table)

---

### US-031: MTS Pilot Onboarding — 10 Employees (P0, 1 pt)

**As a** PM,
**I want** 10 MTS employees using MTClaw via Telegram,
**So that** we have real usage data for G3.

**Acceptance Criteria**:
- Given 10 invitations sent → at least 5 join within 2 days
- Given onboarding guide → users can `/start`, `/help`, `/spec`, `@soul_name`
- Given 5 days pilot → measure WAU (target: 3/10 = 30%)
- Given feedback → collect qualitative notes (happy/friction/requests)

**Tasks**:
1. Create onboarding guide (Telegram-friendly format, not doc)
2. Recruit: 3 Engineering + 3 Sales/CS + 2 Back Office + 2 Management
3. Send invite with bot link + onboarding instructions
4. Track: daily active users, commands used, SOUL preferences
5. Day 5: compile pilot report

**Note**: [@pm] scope, not [@coder]. Listed here for completeness.

---

### US-032: Sprint 4 Feedback Incorporation (P1, 1 pt)

**As a** PM,
**I want** Sprint 4 SOUL feedback session findings addressed,
**So that** known UX issues are fixed before pilot.

**Acceptance Criteria**:
- Given Sprint 4 US-023 feedback → document findings
- If blocking UX issues found → fix before pilot launch
- If no blocking issues → proceed as planned

**Note**: This depends on whether US-023 feedback session was conducted. If not yet done, combine with pilot kickoff (Day 1-2).

---

### US-033: G3 Gate Proposal (P0, 1 pt)

**As a** PM,
**I want** a G3 Build Ready proposal,
**So that** [@cto] and [@cpo] can approve Phase 1 completion.

**Acceptance Criteria**:
- Given proposal → includes: Sprint 1-5 summary, pilot metrics, evidence
- Given evidence → /spec trace records, /review trace records, deployment logs
- Given criteria → matches G3 Build Ready definition from test strategy

**G3 Success Criteria**:
- [ ] All 5 sprints complete (G0.1 → G0.2 → G2 → Sprint 4 → Sprint 5)
- [ ] 2 Rails operational (Spec Factory + PR Gate WARNING)
- [ ] 17 SOULs seeded and routable
- [ ] MTS staging deployed and running
- [ ] Pilot: ≥3/10 WAU (30% adoption)
- [ ] Token cost queryable per tenant
- [ ] Integration tests passing
- [ ] Zero P0 bugs in production

---

## Sprint Schedule

| Day | Track 1: PR Gate | Track 2: Deploy | Track 3: Pilot |
|-----|-----------------|-----------------|----------------|
| 1 | pr-gate SKILL.md + /review cmd | BotFather + .env setup | Recruit 10 users |
| 2 | /review handler testing | Docker Compose deploy | Send invites |
| 3 | Integration tests (tenant, routing) | Verify all endpoints | Pilot Day 1 |
| 4 | Integration tests (AI, traces) | Token cost verify | Pilot Day 2 |
| 5 | Sprint 4 feedback fixes (if any) | Prometheus (stretch) | G3 proposal |

---

## Risk Register

| # | Risk | Prob | Impact | Mitigation |
|---|------|------|--------|------------|
| R4 | MTS adoption <30% WAU | Med | High | Onboarding guide + daily nudge + quick wins |
| R7 | PR Gate false positives | Med | Med | WARNING mode = no blocking, collect data |
| R12 | web_fetch can't access private GitHub repos | High | Med | Test with public repos; PAT for Sprint 8 |
| R13 | MTS VPS Docker resources insufficient | Low | Med | Monitor RAM/CPU; goclaw has 1G limit |
| R14 | Sprint 4 feedback reveals blocking UX | Med | High | Day 1 review + hotfix before pilot |

---

## Tech Debt (from Sprint 4 Review)

| Item | Severity | Sprint |
|------|----------|--------|
| ISSUE-2: `len(goal) > 200` counts bytes not runes (Vietnamese) | LOW | 5 (fix in this sprint) |
| ISSUE-3: Migration 000012 SELECT INTO without error guard | LOW | Defer |

---

## Exit Criteria (G3 Proposal Readiness)

- [ ] 2 Rails operational: /spec + /review
- [ ] Staging deployed, bot online 24/7
- [ ] Pilot running with ≥3 WAU
- [ ] Token cost queryable
- [ ] Integration tests: ≥5 scenarios passing
- [ ] 70% unit test coverage (test strategy tier)
- [ ] Zero P0 bugs
- [ ] CTO + CPO sign-off

---

## References

- [Roadmap Sprint 5](../../01-planning/roadmap.md#sprint-5--mts-pilot--pr-gate-warning-rail-2)
- [PR Gate Design](../../02-design/pr-gate-design.md)
- [Requirements FR-003](../../01-planning/requirements.md)
- [Test Strategy](../../01-planning/test-strategy.md)
- [Sprint 4 Coder Handoff](../SPRINT-004-CODER-HANDOFF.md)
