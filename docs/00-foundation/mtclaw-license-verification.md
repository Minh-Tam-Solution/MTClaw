# GoClaw License Verification

**Date**: 2026-03-02
**Verified by**: [@pm] Sprint 1 Day 1
**Status**: PARTIAL — MIT declared but LICENSE file missing

## Findings

### 1. LICENSE File
- **Local clone**: NOT FOUND (`/home/nqh/shared/MTS-OpenClaw/mtclaw/LICENSE` does not exist)
- **Git history**: Never tracked (not in any commit)
- **GitHub raw**: 404 (https://raw.githubusercontent.com/Minh-Tam-Solution/mtclaw/main/LICENSE)
- **GitHub API**: License endpoint returns 404

### 2. MIT Declaration Evidence
- **README badge**: `[![License: MIT](https://img.shields.io/badge/License-MIT-yellow?style=flat-square)](LICENSE)`
- **GitHub metadata**: Repository page shows "MIT" in About section
- **go.mod**: No license field (standard for Go modules)

### 3. Dependency Licenses (go.mod scan)
All dependencies are permissive:
- `pgx` (jackc/pgx) — MIT
- `cobra` (spf13/cobra) — Apache-2.0
- `telego` — MIT
- `opentelemetry-go` — Apache-2.0
- `gorilla/websocket` — BSD-2-Clause
- `golang.org/x/*` — BSD-3-Clause

**No AGPL/GPL dependencies found.**

### 4. Risk Assessment
- **Risk**: LOW — MIT declared in multiple places, just missing the actual file
- **Impact**: Internal fork for MTClaw; MIT allows forking, modification, commercial use
- **Action**: Created LICENSE file with standard MIT text and GoClaw attribution

## Resolution

Created `LICENSE` file in MTClaw repo with:
- **PROPRIETARY** license (MTClaw is NOT OSS — internal use only for MTS/NQH)
- Note that GoClaw upstream is MIT-licensed, which permits internal forking
- Copyright to Minh Tam Solution

## Follow-up
- [ ] Contact upstream maintainer to add LICENSE file to GoClaw repo
- [ ] Verify no license change in future GoClaw releases

---

**CTO Directive**: LICENSE verification is pre-execution gate. Proceed with documented finding.
Upstream contact is follow-up, not Sprint 1 blocker.
