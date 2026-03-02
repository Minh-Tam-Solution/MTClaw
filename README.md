# MTClaw

[![Go](https://img.shields.io/badge/Go_1.25-00ADD8?style=flat-square&logo=go&logoColor=white)](https://go.dev/) [![PostgreSQL](https://img.shields.io/badge/PostgreSQL-316192?style=flat-square&logo=postgresql&logoColor=white)](https://www.postgresql.org/) [![SDLC 6.1.1](https://img.shields.io/badge/SDLC-6.1.1-blue?style=flat-square)](SDLC-Enterprise-Framework/) [![License: Proprietary](https://img.shields.io/badge/License-Proprietary-red?style=flat-square)](LICENSE)

**MTClaw** is a governance-first company assistant platform built on GoClaw runtime. It provides 3 governance rails (Spec Factory, PR Gate, Knowledge & Answering) with 16 AI personas (SOULs) for engineering, sales, customer service, and general business tasks.

## Architecture

```
EndiorBot (reference only)     SDLC-Orchestrator (pattern reference)
       \                              /
        \--- zero runtime coupling ---/
                    |
              MTClaw (runtime)
              GoClaw + 16 SOULs
              Bflow AI-Platform
              Telegram (P1) / Zalo (P2)
```

- **Runtime**: GoClaw (Go 1.25, single binary, PostgreSQL multi-tenant)
- **AI Backend**: Bflow AI-Platform ONLY (single source of truth)
- **Governance**: 3 Rails (Spec Factory + PR Gate + Knowledge)
- **Personas**: 16 SOULs (12 SDLC + 4 MTS business)
- **Channels**: Telegram (Phase 1), Zalo (Phase 2)
- **Framework**: SDLC Enterprise Framework 6.1.1 (STANDARD tier)

## Quick Start

```bash
# Build
make build

# Configure
cp .env.example .env
# Edit .env with your credentials

# Database
make migrate-up

# Run
make run
```

## Project Structure

```
MTClaw/
  cmd/                  # CLI commands (cobra)
  internal/             # Core business logic
  migrations/           # PostgreSQL migrations
  pkg/                  # Shared packages
  docs/
    00-foundation/      # Problem statement, business case, user research
    01-planning/        # Requirements, test strategy, user stories
    02-design/          # ADRs, system architecture
    04-build/           # Sprint plans
    08-collaborate/     # AGENTS.md, SDLC compliance, SOULs
  SDLC-Enterprise-Framework/  # Framework (symlink)
```

## 16 SOULs

| Category | SOULs |
|----------|-------|
| SDLC Executors (SE4A) | pm, architect, coder, reviewer, researcher, writer, pjm, devops, tester |
| SDLC Advisors (SE4H) | cto, cpo, ceo |
| Router | assistant |
| MTS Business | mts-dev, mts-sales, mts-cs, mts-general |

## Documentation

- [AGENTS.md](docs/08-collaborate/AGENTS.md) — AI assistant guide
- [Problem Statement](docs/00-foundation/problem-statement.md)
- [Requirements](docs/01-planning/requirements.md)
- [ADRs](docs/02-design/01-ADRs/)

## License

Internal use only. GoClaw runtime is MIT-licensed (upstream).
MTClaw is a proprietary internal platform for Minh Tam Solution.

---

*MTClaw = Governance backbone for AI-first transformation.*
*Built with SDLC Enterprise Framework 6.1.1.*
