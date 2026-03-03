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

```json
{
  "spec_version": "0.1.0",
  "title": "Short descriptive title",
  "narrative": {
    "as_a": "role",
    "i_want": "feature/capability",
    "so_that": "business value"
  },
  "acceptance_criteria": [
    "Given X, When Y, Then Z"
  ],
  "priority": "P0|P1|P2|P3",
  "estimated_effort": "S|M|L|XL",
  "soul_author": "pm",
  "created_at": "ISO 8601 timestamp"
}
```

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
