---
sprint: 16
title: Claude Code Bridge — Permission Approval via Telegram (C)
status: PLANNED
date: 2026-05-05
version: "1.0.0"
author: "[@pm] + [@architect]"
framework: SDLC Enterprise Framework 6.1.1
tier: STANDARD
---

# Sprint 16 — Claude Code Bridge: Permission Approval via Telegram (C)

**Sprint**: 16 of 17 (bridge track: C of A1/A2/B/C/D)
**Phase**: 5 (Claude Code Bridge — ADR-010)
**Duration**: 4 days
**Owner**: [@coder] (implementation)
**Points**: ~12 (security-critical: fail-closed, async permission)
**Gate**: Full approval works, fail-closed verified, double-callback safe
**Entry**: Sprint 15 gate passed

---

## 1. Sprint Goal

**Permission requests -> Telegram Approve/Reject -> fail-safe fallback (D6).**

### Key Outcomes

1. Async permission store with TTL + request_hash dedup (L2)
2. `POST /hook/permission` -> 202 -> poll `GET /hook/permission/{id}`
3. Telegram inline keyboard (Approve/Reject with tool details)
4. Actor validation against ApproverACL (D8)
5. Timeout: auto-approve low-risk, **deny** high-risk (fail closed)
6. Native dialog fallback only if `LocalInteractive = true`
7. Hook script: `permission-request.sh`

---

## 2. New Files

| File | Purpose |
|------|---------|
| `internal/claudecode/permission_store.go` | Async state + TTL + request_hash UNIQUE dedup |
| `internal/claudecode/handlers/permission_handler.go` | POST -> 202 -> poll GET + fail-safe (D6) |

### Modified Files

| File | Change |
|------|--------|
| `internal/channels/telegram/commands_cc.go` | Inline keyboard for approval, callback_query handler |
| `internal/claudecode/hook_server.go` | Add permission endpoints |

---

## 3. Acceptance Criteria

| # | Criterion | Verification |
|---|-----------|-------------|
| 1 | Approve button in Telegram resolves pending permission | Integration test |
| 2 | Reject button denies tool execution | Integration test |
| 3 | Timeout auto-rejects high-risk tools (Bash, Edit, Write) | Unit test |
| 4 | Timeout auto-approves low-risk tools (Read, Glob, Grep) | Unit test |
| 5 | Duplicate callback (race condition) doesn't double-apply | Unit test (request_hash UNIQUE) |
| 6 | Non-approver actor cannot approve (ACL check) | Unit test |
| 7 | HookServer unreachable + high-risk = deny + audit + Telegram notify | Unit test |
| 8 | `reason_code=hook_unreachable_fail_closed` in audit event | Unit test |
| 9 | `make build && make test` passes | CI gate |

---

## 4. External Security Review (Post-Sprint 16)

**2 days** external Go security review of:
- `hook_auth.go` — HMAC implementation
- `input_sanitizer.go` — deny pattern completeness
- `permission_store.go` — race conditions, TTL enforcement
- `session_manager.go` — tenant isolation, admission control

**Blocking**: Sprint 17 (free-text) cannot start until security review passes.

---

## 5. NOT in Sprint 16

| Item | Sprint |
|------|--------|
| Free-text relay | 17 (D) |
| `/cc risk interactive` | 17 (D) |
| Installer (`mtclaw bridge setup/uninstall`) | 17 (D) |
| PG bridge store | 17 (D) |
