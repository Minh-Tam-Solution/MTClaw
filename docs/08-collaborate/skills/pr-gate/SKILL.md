---
name: pr-gate
description: Code review for pull requests. Governance Rail #2 (WARNING mode).
---

# PR Gate — Governance Rail #2

## When This Skill Activates

- User sends `/review <PR_URL>` command
- User asks to "review this PR", "check this code", "evaluate this pull request"
- Reviewer SOUL receives a code review request

## Review Process

1. **Fetch**: Use web_fetch to retrieve the PR diff from the URL
2. **Analyze**: Evaluate the diff against policy rules below
3. **Report**: Format findings as a structured review report

## Policy Rules (WARNING Mode)

Evaluate and report — do NOT block merge.

| Rule | Severity | Check |
|------|----------|-------|
| Missing tests | WARN | PR adds .go/.ts files but no corresponding test files |
| Large diff | WARN | >500 lines changed — suggest splitting |
| Security patterns | WARN | Hardcoded secrets, SQL injection, XSS patterns |
| Missing spec reference | WARN | No SPEC- or issue # in PR title/body |
| TODO/FIXME | INFO | New TODO/FIXME comments added |

## Report Format

Format your review as:

🔍 **PR Review — WARNING Mode**

**PR**: {title} (#{number})
**Files**: {count} files, +{additions}/-{deletions} lines

### Issues Found
- ⚠️ WARN: {description}
- ℹ️ INFO: {description}

### Suggestions
1. {actionable suggestion}

### Summary
| Category | Status |
|----------|--------|
| Tests | ⚠️/✅ |
| Size | ⚠️/✅ |
| Security | ⚠️/✅ |
| Spec ref | ⚠️/✅ |

**Mode**: WARNING (report only — merge not blocked)

## Boundaries

- This skill reviews code only — not architecture, not specs, not deployment
- If PR requires architecture review → delegate to @architect
- If PR lacks a spec → suggest creating one via @pm /spec
- WARNING mode: NEVER say "merge blocked" or "PR rejected"

## Vietnamese Support

- Review in the language of PR content
- If mixed → use English (code convention)
