# ROLE_TOOL_MATRIX — MTClaw SASE 12-Role Model

**Version**: 1.0.0
**Framework**: SDLC 6.1.1 SASE (Software Agent Standard Edition)
**Active by default**: 6 SOULs | **Available on-demand**: 10 SOULs

## Matrix

| # | SOUL | Type | Category | Active | Tools / Rails | Trigger |
|---|------|------|----------|--------|---------------|---------|
| 1 | pm | SE4A | Executor | Default | /spec (Rail #1), requirements, user stories | `@pm` or requirements context |
| 2 | architect | SE4A | Executor | Default | ADRs, system design, G2 gate review | `@architect` or design context |
| 3 | coder | SE4A | Executor | Default | Implementation, tests, code generation | `@coder` or code context |
| 4 | reviewer | SE4A | Executor | Default | PR Gate (Rail #2), code review, quality scoring | `@reviewer` or PR context |
| 5 | assistant | Router | Router | Default | General Q&A, routing to specialized SOULs | Default fallback |
| 6 | mts-dev | MTS | Business | Default | MTS dev daily tasks, Bflow API, internal tools | `@mts-dev` or dev context |
| 7 | cto | SE4H | Advisor | On-demand | Architecture guard, P0 blocking decisions | `@cto` explicit |
| 8 | cpo | SE4H | Advisor | On-demand | Product guard, strategic decisions | `@cpo` explicit |
| 9 | ceo | SE4H | Advisor | On-demand | Executive decisions, governance override | `@ceo` explicit |
| 10 | researcher | SE4A | Executor | On-demand | User research, data analysis | `@researcher` |
| 11 | writer | SE4A | Executor | On-demand | Documentation, guides | `@writer` |
| 12 | pjm | SE4A | Executor | On-demand | Sprint planning, task breakdown | `@pjm` |
| 13 | devops | SE4A | Executor | On-demand | Infrastructure, deployment | `@devops` |
| 14 | tester | SE4A | Executor | On-demand | Test strategy, QA | `@tester` |
| 15 | mts-sales | MTS | Business | On-demand | Sales playbooks, proposals, CRM queries | `@mts-sales` |
| 16 | mts-cs | MTS | Business | On-demand | Customer support, ticket handling | `@mts-cs` |
| 17 | mts-general | MTS | Business | On-demand | General MTS employee tasks | `@mts-general` |

## Rail Ownership

| Rail | Primary SOUL | Supporting SOULs | Phase |
|------|-------------|-----------------|-------|
| #1 Spec Factory | pm | architect, cpo | Sprint 4 (prototype) → Sprint 7 (full) |
| #2 PR Gate | reviewer | coder, cto | Sprint 5 (WARNING) → Sprint 8 (ENFORCE) |
| #3 Knowledge | assistant | all domain SOULs | Sprint 6 (RAG per domain) |

## Notes
- **6 active default**: Keeps token cost low while covering daily workflows
- **On-demand**: Activated via explicit `@role` mention or context detection
- **SE4A**: Software Engineer for AI (autonomous executor)
- **SE4H**: Software Engineer for Human (advisory, human-in-loop)
