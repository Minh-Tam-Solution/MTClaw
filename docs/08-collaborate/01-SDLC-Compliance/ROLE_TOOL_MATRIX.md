# ROLE_TOOL_MATRIX — MTClaw SASE 12-Role Model

**Version**: 2.1.0
**Framework**: SDLC 6.1.1 SASE (Software Agent Standard Edition)
**Active by default**: 5 SOULs | **Available on-demand**: 12 SOULs

## Matrix

| # | SOUL | Type | Category | Active | Tools / Rails | Trigger |
|---|------|------|----------|--------|---------------|---------|
| 1 | assistant | Router | Universal | Default (`is_default`) | General Q&A, meeting notes, routing to specialized SOULs | Default entry point |
| 2 | pm | SE4A | Executor | Default | /spec (Rail #1), requirements, user stories | `@pm` or requirements context |
| 3 | architect | SE4A | Executor | Default | ADRs, system design, G2 gate review | `@architect` or design context |
| 4 | coder | SE4A | Executor | Default | Implementation, tests, code generation | `@coder` or code context |
| 5 | reviewer | SE4A | Executor | Default | PR Gate (Rail #2), code review, quality scoring | `@reviewer` or PR context |
| 6 | enghelp | Business | Business | On-demand | Engineering daily tasks, code review, debugging | `@enghelp` or engineering context |
| 7 | cto | SE4H | Advisor | On-demand | Architecture guard, P0 blocking decisions | `@cto` explicit |
| 8 | cpo | SE4H | Advisor | On-demand | Product guard, strategic decisions | `@cpo` explicit |
| 9 | ceo | SE4H | Advisor | On-demand | Executive decisions, governance override | `@ceo` explicit |
| 10 | researcher | SE4A | Executor | On-demand | User research, data analysis | `@researcher` |
| 11 | writer | SE4A | Executor | On-demand | Documentation, guides | `@writer` |
| 12 | pjm | SE4A | Executor | On-demand | Sprint planning, task breakdown | `@pjm` |
| 13 | devops | SE4A | Executor | On-demand | Infrastructure, deployment | `@devops` |
| 14 | tester | SE4A | Executor | On-demand | Test strategy, QA | `@tester` |
| 15 | sales | Business | Executor | On-demand | Proposals, pricing, case studies | `@sales` |
| 16 | cs | Business | Executor | On-demand | Customer support, ticket handling, onboarding | `@cs` |
| 17 | itadmin | Operations | Executor | On-demand | Infrastructure, AI models, ports, security, monitoring | `@itadmin` |

## Rail Ownership

| Rail | Primary SOUL | Supporting SOULs | Phase |
|------|-------------|-----------------|-------|
| #1 Spec Factory | pm | architect, cpo | Sprint 4 (prototype) → Sprint 7 (full) |
| #2 PR Gate | reviewer | coder, cto | Sprint 5 (WARNING) → Sprint 8 (ENFORCE) |
| #3 Knowledge | assistant | all domain SOULs | Sprint 6 (RAG per domain) |

## Notes
- **5 active default**: assistant (universal router) + 4 SDLC core (pm, architect, coder, reviewer)
- **On-demand**: Activated via explicit `@role` mention or context detection by assistant
- **SE4A**: Software Engineer for AI (autonomous executor)
- **SE4H**: Software Engineer for Human (advisory, human-in-loop)
- **Tenant-agnostic**: SOUL names have no tenant prefix — same SOULs work for MTS, NQH, or future tenants
- **v2.1.0**: Added `itadmin` SOUL (Operations executor) — infrastructure, AI models, ports, security
- **Retired**: `mts-general` merged into `assistant` (v2.0.0); `mts-dev/mts-sales/mts-cs` renamed to `enghelp/sales/cs`
