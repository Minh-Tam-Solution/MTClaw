---
sprint: 15
title: Claude Code Bridge — HookServer + Stop Notification + Doctor (B)
status: PLANNED
date: 2026-04-28
version: "1.0.0"
author: "[@pm] + [@architect]"
framework: SDLC Enterprise Framework 6.1.1
tier: STANDARD
---

# Sprint 15 — Claude Code Bridge: HookServer + Stop Notification + Doctor (B)

**Sprint**: 15 of 17 (bridge track: B of A1/A2/B/C/D)
**Phase**: 5 (Claude Code Bridge — ADR-010)
**Duration**: 5 days
**Owner**: [@coder] (implementation)
**Points**: ~15 (security-critical: HMAC, circuit breaker)
**Gate**: HMAC replay rejected, stop notification delivered to Telegram, doctor shows all green
**Entry**: Sprint 14 gate + CEO validation sign-off

---

## 1. Entry Criteria

| Criterion | Status | Owner |
|-----------|--------|-------|
| Sprint 14 gate passed | Pending | [@cto] |
| CEO validation sign-off (2-3 day usage) | Pending | [@ceo] |
| All /cc commands working in Telegram | Pending | [@coder] |
| Cross-tenant isolation test passing | Pending | [@coder] |

---

## 2. Sprint Goal

**Agent completion triggers Telegram notification. Health monitoring ensures reliability.**

When Claude Code finishes a task, the bridge:
1. Receives a signed webhook from the hook script
2. Verifies HMAC-SHA256 signature (rejects replay attacks)
3. Scrubs output, captures git diff
4. Delivers formatted summary to Telegram

### Key Outcomes

1. HookServer on 127.0.0.1:18792 (localhost only, configurable)
2. HMAC-SHA256 authentication with 30s timestamp window
3. Stop notification: scrubbed summary + git diff -> Telegram
4. Health monitor: 30s ticker, admission control runtime checks
5. Circuit breaker: 3 failures -> degraded -> native dialog fallback
6. Transcript parser: NDJSON -> summarizer
7. Full doctor: all health checks + connectivity
8. Hook shell script: `stop.sh` (HMAC-signed)

---

## 3. New Files

| File | Purpose |
|------|---------|
| `internal/claudecode/hook_auth.go` | HMAC-SHA256 signing + verification (D5) |
| `internal/claudecode/hook_server.go` | HTTP server 127.0.0.1:18792 + rate limiting |
| `internal/claudecode/handlers/stop_handler.go` | Stop event: scrub + git diff + Telegram |
| `internal/claudecode/notifier.go` | Bus integration, circuit breaker |
| `internal/claudecode/health.go` | Periodic health check + admission runtime |
| `internal/claudecode/transcript.go` | NDJSON parser + summarizer |

### Modified Files

| File | Change |
|------|--------|
| `cmd/gateway.go` | Start HookServer goroutine if bridge enabled |
| `cmd/bridge.go` | `mtclaw bridge serve` (standalone) + full `status` doctor |

---

## 4. Task Breakdown (5 Days)

### Day 1: HMAC Auth + HookServer skeleton
### Day 2: Stop handler + notifier + circuit breaker
### Day 3: Health monitor + transcript parser
### Day 4: Hook shell scripts + gateway wiring
### Day 5: Integration tests + stress test (6 concurrent sessions)

**Stress test**: 1 session spam permission requests + 5 simultaneous stop hooks — verify no out-of-order routing.

---

## 5. Acceptance Criteria

| # | Criterion | Verification |
|---|-----------|-------------|
| 1 | HMAC replay with old timestamp rejected | Unit test |
| 2 | HMAC with wrong secret rejected | Unit test |
| 3 | Stop notification arrives in Telegram within 5s | Integration test |
| 4 | Git diff included in stop notification (truncated to 2000 chars) | Integration test |
| 5 | Circuit breaker trips after 3 notification failures | Unit test |
| 6 | Health check detects dead tmux session | Unit test |
| 7 | Doctor shows all-green for healthy setup | CLI test |
| 8 | Rate limiter blocks >10 hooks/sec per session | Unit test |
| 9 | `make build && make test` passes | CI gate |

---

## 6. NOT in Sprint 15

| Item | Sprint |
|------|--------|
| Permission approval (D6) | 16 (C) |
| Free-text relay | 17 (D) |
| PG bridge store | 17 (D) |
| Installer (`mtclaw bridge setup`) | 17 (D) |
