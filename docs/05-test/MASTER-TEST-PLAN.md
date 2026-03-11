# Master Test Plan — MTClaw

**SDLC Stage**: 05-Test
**Version**: 7.0.0
**Date**: 2026-03-11
**Author**: [@tester], [@cto] (tiered targets)
**Coverage**: Sprint 1-34 cumulative (includes Claude Code Bridge A-D + Intelligence Upgrade + Provider Fallback + Observability + Health Routing + Discord Channel + Unified Command Routing)

---

## 1. Scope

This plan covers all automated testing for MTClaw gateway across Sprint 1-25, organized by test tier (Unit, Integration, E2E, Security, Performance).

### Sprint Feature Map

| Sprint | Features | Risk Level |
|--------|----------|------------|
| 1-3 | Foundation: tenant isolation, SOUL loading, config, sessions | P0 |
| 4 | Bflow AI-Platform, /spec prototype, 16 SOULs seeded | P0 |
| 5 | PR Gate WARNING, context anchoring, @mention routing | P0 |
| 6 | SOUL-Aware RAG, team routing, cost guardrails | P1 |
| 7 | Rebrand mtclaw->mtclaw, /spec governance, SOUL drift, RAG evidence | P1 |
| 8 | PR Gate ENFORCE, GitHub webhook, context drift prevention, audit PDF | P0 |
| 9 | Channel cleanup (Discord/Feishu/WhatsApp removed), SOUL completion | P1 |
| 10 | MSTeams Bot Framework extension (Adaptive Cards, SSRF) | P0 |
| 11 | Security pentest hardening, performance baseline | P0 |
| 12 | Spec quality scoring, design-first gate, evidence linking, workspace/projects | P0 |
| 13 (A1) | Bridge local session core: types, tmux, project registry, state machine | P0 |
| 14 (A2) | Bridge security: sanitizer, redactor, policy, audit, /cc commands | P0 |
| 15 (B) | Bridge HookServer: HMAC auth, stop notify, circuit breaker, health monitor | P0 |
| 16 (C) | Bridge permission: async store, fail-safe timeout, Telegram keyboard | P0 |
| 17 (D) | Bridge interactive: SendText relay, setup CLI, cleanup cron, drain queue | P0 |
| **18** | **SOUL-Aware Launch: --as flag, Strategy A/B/C, install-agents, soul loader** | **P0** |
| **19** | **Intelligence Envelope: SessionIntelligenceEnvelope, /cc info, turn context** | **P1** |
| **20A** | **Skills Integration: SDLC Framework skill, install-skills, agent template skills** | **P1** |
| **20B** | **Project Context: CLAUDE.md generator, /cc context set, turn-time injection** | **P1** |
| **21** | **Role-Aware Defaults: role->RiskMode defaults (UX only), --allowedTools** | **P1** |
| **22** | **Agent Teams Research Spike: ADR-012 NO-GO decision (no production code)** | **P2** |
| **23** | **Provider Persona Projection: capability matrix, Cursor POC adapter, ADR-013** | **P1** |
| **24** | **Provider Fallback Chain: Claude CLI provider, fallback logic, env/config, doctor** | **P0** |
| **25** | **Fallback Deploy + Observability: Docker npm install, OAuth volume, fallback tracing metadata, OTEL propagation** | **P0** |
| **26** | **Bridge Production Readiness: PG session persistence, Docker bridge, audit dual-write, standalone SOUL fix** | **P0** |
| **27** | **Metrics + Integration + Hardening: Adoption metrics (PG traces), cost guardrails (monthly tokens + warning), OpenAPI spec, provider chain E2E tests, integration specs** | **P1** |
| **28** | **Observability + Hardening + Test Realignment: Health-based provider routing (circuit breaker), bridge session metrics, fallback loop E2E tests, deployment runbooks, fallback benchmarks** | **P0** |
| **29** | **Zalo Channel: OAuth 2.0, webhook verification, DM/group support** | **P1** |
| **30** | **Discord Channel: DM/group policy, guild allowlist, mention gate, env-based init** | **P0** |
| **31** | **Memory Enhancement Phase 0: improved memory flush prompt** | **P1** |
| **32** | **Memory Phase 1: Discord reactions** | **P1** |
| **33** | **Unified Command Routing Sprint A: shared `internal/commands/` package, Responder interface, CommandMetadata struct, ResolveAgentUUID extraction, PublishReset/Stop/StopAll shared helpers** | **P0** |
| **34** | **Discord Command Parity Sprint B: FactoryWithStores expansion (3 stores), 11 new Discord commands (/spec, /review, /teams, /status, /spec_list, /spec_detail, /tasks, /task_detail, /writers, /addwriter, /removewriter), shared formatters (specs, tasks, writers), Telegram refactored to shared package** | **P0** |

---

## 2. Test Tiers

### 2.1 Unit Tests (target: 80%+ coverage)

| Package | Test File | Tests | Sprint | Status |
|---------|-----------|-------|--------|--------|
| agent | input_guard_test.go | Input sanitization | 2 | PASS |
| agent | toolloop_test.go | Tool execution loop | 3 | PASS |
| agent | loop_history_test.go | Conversation history | 3 | PASS |
| bootstrap | seed_store_test.go | SOUL seeding | 4 | PASS |
| bus | (covered by consumer_test) | Dedup, debounce | 3 | PASS |
| channels/telegram | format_test.go | Message formatting | 2 | PASS |
| channels/telegram | media_test.go | Media handling | 2 | PASS |
| channels/telegram | stt_test.go | Speech-to-text | 4 | PASS |
| channels/telegram | topic_config_test.go | Topic routing | 5 | PASS |
| channels/telegram | commands_workspace_test.go | Workspace show/switch/projects | 12 | PASS |
| channels/typing | controller_test.go | Typing indicators | 3 | PASS |
| cron | retry_test.go | Retry with backoff | 3 | PASS |
| gateway/methods | agents_create_owner_test.go | Agent ownership | 4 | PASS |
| governance | spec_processor_test.go | Spec processing | 7 | PASS |
| governance | spec_quality_test.go | 5-dimension quality scoring | 12 | PASS |
| governance | pr_processor_test.go | PR verdict parsing | 8 | PASS |
| governance | design_gate_test.go | Design-first gate | 12 | PASS |
| http | webhook_github_test.go | GitHub HMAC verification | 8 | PASS |
| audit | pdf_builder_test.go | Audit trail PDF | 8 | PASS |
| mcp | bridge_tool_test.go | MCP bridge | 3 | PASS |
| memory | memory_test.go | Vector memory | 4 | PASS |
| providers | schema_cleaner_test.go | Schema normalization | 3 | PASS |
| rag | evidence_test.go | RAG evidence retrieval | 7 | PASS |
| sandbox | docker_test.go | Docker sandbox | 4 | PASS |
| scheduler | scheduler_test.go | Lane scheduling | 3 | PASS |
| security | pentest_test.go | 7 security vectors | 11 | PASS |
| souls | drift_test.go | SOUL drift detection | 7 | PASS |
| souls | behavioral_test.go | 16 SOULs x5 structural | 9 | PASS |
| store | validate_test.go | Input validation | 2 | PASS |
| tools | boundary_test.go | Tool boundaries | 3 | PASS |
| tools | scrub_test.go | Output scrubbing | 4 | PASS |
| tools | context_file_interceptor_test.go | Context injection | 5 | PASS |
| tools | context_keys_test.go | Context key routing | 7 | PASS |
| tools | registry_test.go | Tool registry | 3 | PASS |
| tools | policy_mcp_test.go | MCP policy | 5 | PASS |
| tools | rate_limiter_test.go | Rate limiting | 6 | PASS |
| tracing | exporter_test.go | OTel export | 5 | PASS |
| extensions/msteams | msteams_test.go | Config, JWT, send, cards | 10 | PASS |
| cmd | gateway_consumer_test.go | Consumer routing | 5 | PASS |
| **claudecode** | **types_test.go** | SessionID gen, HookSecret, Capabilities, Admission | 13 | **PASS** |
| **claudecode** | **tmux_test.go** | Tmux session names, capture, sendKeys, list | 13 | **PASS** |
| **claudecode** | **project_test.go** | Project registry CRUD, workspace fingerprint | 13 | **PASS** |
| **claudecode** | **session_test.go** | State machine, risk mode, queue, ACL, turn context | 13+17+19 | **PASS** |
| **claudecode** | **session_manager_test.go** | Create, tenant isolation, admission, kill, risk, SendText, CaptureOutput, CleanupStopped, TransitionSession drain, persona strategy A/B/C, stale agent file, role defaults | 13+17+18+21 | **PASS** |
| **claudecode** | **bridge_policy_test.go** | Capability model, risk escalation, role defaults (executor/advisor/unknown/missing), AllowedTools per role, FormatAllowedTools, role overridable | 14+21 | **PASS** |
| **claudecode** | **input_sanitizer_test.go** | 87+ deny patterns, shell + bridge-specific | 14 | **PASS** |
| **claudecode** | **output_redactor_test.go** | API keys, tokens, DSN, JWT, PEM, env vars, heavy redact | 14 | **PASS** |
| **claudecode** | **bridge_audit_test.go** | JSONL write, multiple events, writable dir, nil DB | 14 | **PASS** |
| **claudecode** | **hook_auth_test.go** | HMAC-SHA256 sign/verify, replay rejection, timestamp window | 15 | **PASS** |
| **claudecode** | **hook_server_test.go** | Health, missing headers, wrong method, invalid session, valid stop, wrong sig, permission 202, dedup, poll, after-decision, rate limiter | 15+16 | **PASS** |
| **claudecode** | **notifier_test.go** | Stop message, redaction, circuit breaker, nil sendFn | 15 | **PASS** |
| **claudecode** | **transcript_test.go** | NDJSON parse, malformed skip, summarize | 15 | **PASS** |
| **claudecode** | **health_test.go** | Initial check, no dead sessions, active detection, default interval | 15 | **PASS** |
| **claudecode** | **permission_store_test.go** | Create, dedup, get, decide (approve/deny/double-apply/ACL), timeout (high/low risk), expiry, list, cleanup, hash | 16 | **PASS** |
| **claudecode** | **soul_loader_test.go** | KnownRoles (scan/missing/cache), LoadSOUL (coder/unknown/path traversal/all roles), HashFileContent, frontmatter parsing, InvalidateRolesCache | 18 | **PASS** |
| **claudecode** | **agent_installer_test.go** | LoadAgentTemplates, InstallAgents (create/idempotent/skip user/force/role overrides), IncludesSkills, AgentFileHasSkills | 18+20A | **PASS** |
| **claudecode** | **provider_test.go** | LaunchCommand Strategy A/B/C/Precedence, AllowedTools, NoAllowedTools, StubAdapter, EnvSanitization | 18+21 | **PASS** |
| **claudecode** | **intelligence_test.go** | BuildPersonaEnvelope (StrategyA/B/Bare/EmptySource/EmptySourceWithRole), BuildIntelligenceEnvelope, JSON serialization, OmitEmpty, StrategyFromPersonaSource, TurnContext (marshal/format markdown/nil/empty/omit empty) | 19 | **PASS** |
| **claudecode** | **skills_generator_test.go** | SDLCFrameworkSkill (under budget/content), InstallSkills (create/idempotent/skip user/force) | 20A | **PASS** |
| **claudecode** | **claudemd_generator_test.go** | DetectProjectProfile (Go/TS/Python/Makefile/Docker), GenerateClaudeMD content, InitProject (create/skip user/force) | 20B | **PASS** |
| **claudecode** | **provider_cursor_test.go** | CursorProjectionAdapter (name/launch nil/hooks unsupported/parse unsupported/capabilities/transcript empty), CursorRule FormatMDC (with/without alwaysApply), GenerateCursorRules (create/idempotent/skip user/force/filtered/invalid), ProjectionInfo, Registry integration | 23 | **PASS** |
| **providers** | **claude_cli_test.go** | Name, DefaultModel (custom/fallback), Timeout (default/custom), buildCLIPrompt (simple/system/empty), parseCLIResponse (valid/maxTokens/rawText/empty/multiBlock), filterEnv (strips/preserves), ChatEmptyPrompt, ChatStreamDelegatesToChat | 24 | **PASS** |
| **agent** | **fallback_test.go** | FallbackProviderWired, NoFallbackByDefault, IsRetryableError_Triggers (500/429/400), LoopConfigPreservesBothProviders, StubChatResponse, E2E_PrimarySucceeds, E2E_RetryableError_FallbackSucceeds, E2E_FatalError_NoFallback, E2E_BothFail, E2E_CTOGuard_NoFallbackAtIter1WithTools, E2E_ToolsStrippedOnFallback | 25+27 | **PASS** |
| **agent** | **fallback_loop_test.go** | HealthTrackerWired, NoHealthTrackerByDefault, HealthTrackerRecordsViaProvider, CircuitBreakerTripsOnRepeatedFailure, FailOpenWhenBothCircuitsOpen, HealthTrackerScoreAccuracy | 28 | **PASS** |
| **cost** | **guardrails_test.go** | CheckDailyLimit (under/at/fail-open), CheckMonthlyTokenLimit (under/exceeded/fail-open), CheckWarningThreshold (below/above/fail-open) | 27 | **PASS** |
| **store/pg** | **tracing_adoption_test.go** | TokenUsage struct, SQL patterns, time range, empty map, aggregation math | 27 | **PASS** |
| **providers** | **health_tracker_test.go** | InitialHealthy, RecordSuccess, RecordFailure, CircuitBreaker_Trip, CircuitBreaker_Cooldown, CircuitBreaker_Recovery, SlidingWindow, Score_Empty, Stats, Concurrent, FailOpenOnDoubleCircuitOpen | 28 | **PASS** |
| **claudecode** | **session_metrics_test.go** | BridgeMetrics_Empty, BridgeMetrics_ActiveCount, BridgeMetrics_ByRiskMode, BridgeMetrics_ByRole, BridgeMetrics_Lifetime | 28 | **PASS** |
| **channels/discord** | **discord_test.go** | New() constructor (4→6 params), guild allowlist, mention gate, DM policy, group policy, message routing | 30+34 | **PASS** |
| **commands** | **resolver_test.go** | ResolveAgentUUID_ParsesUUID, _FallbackToStore, _EmptyKey | 33 | **PASS** |
| **commands** | **(inline in workspace/tasks/specs/writers)** | Shared formatters: FormatTaskList, FormatTaskDetail, FormatSpecList, FormatSpecDetail, ListWriters, AddWriter, RemoveWriter, TruncateStr, TaskStatusIcon, PublishSpec, PublishReview, PublishReset, CommandMetadata.ToMap() | 33+34 | **PASS** |

### 2.2 Integration Tests

| ID | Scenario | Sprint | Priority | Status |
|----|----------|--------|----------|--------|
| INT-001 | Tenant isolation: User A cannot see User B data | 3 | P0 | PASS |
| INT-002 | SOUL loading: All 16 SOULs load at startup | 4 | P0 | PASS |
| INT-003 | SOUL cache: Checksum mismatch triggers reload | 5 | P1 | PASS |
| INT-004 | Bflow AI: Request -> AI-Platform -> response | 4 | P0 | PASS |
| INT-005 | Bflow AI: Fallback on AI-Platform timeout | 4 | P1 | PASS |
| INT-006 | Cost guardrail: Reject at 100% monthly limit | 6 | P1 | PASS |
| INT-007 | Spec Factory: /spec -> JSON output | 7 | P0 | PASS |
| INT-008 | PR Gate: WARNING mode evaluation | 5 | P0 | PASS |
| INT-009 | Evidence: Governance action creates audit record | 5 | P0 | PASS |
| INT-010 | Multi-tenant concurrent: 2 tenants simultaneous | 6 | P1 | PASS |
| INT-011 | MSTeams: inbound message -> bus publish -> SOUL routing | 10 | P0 | PASS |
| INT-012 | MSTeams: JWT verification (valid/expired/wrong iss/aud) | 10 | P0 | PASS |
| INT-013 | MSTeams: channel column written to governance tables | 10 | P1 | PASS |
| INT-014 | MSTeams: MSTEAMS_APP_PASSWORD not in logs | 10 | P0 | PASS |
| INT-015 | MSTeams + Telegram: cross-channel /spec same output | 10 | P1 | PASS |
| INT-016 | PR Gate ENFORCE: fail verdict blocks merge | 8 | P0 | PASS |
| INT-017 | GitHub webhook: HMAC signature verification | 8 | P0 | PASS |
| INT-018 | GitHub webhook: PR opened -> inbound publish | 8 | P0 | PASS |
| INT-019 | Spec quality: 5-dimension scoring >= 70 pass | 12 | P0 | PASS |
| INT-020 | Design-first gate: coder blocked without spec | 12 | P0 | PASS |
| INT-021 | Evidence chain: spec -> pr_gate -> chain build | 12 | P0 | NEW |
| INT-022 | Evidence linker: auto-link spec to PR by session | 12 | P0 | NEW |
| INT-023 | Audit PDF: spec + chain -> valid PDF | 8 | P1 | PASS |
| INT-024 | SOUL behavioral: 16 SOULs x 5 checks pass | 9 | P0 | PASS |
| INT-025 | Context drift: 5 layers verified | 8 | P0 | PASS |
| INT-026 | Workspace show: /workspace returns current dir | 12 | P1 | NEW |
| INT-027 | Workspace switch: /workspace <path> updates agent + invalidates cache | 12 | P0 | NEW |
| INT-028 | Workspace invalid: /workspace <bad_path> returns error, no state change | 12 | P1 | NEW |
| INT-029 | Projects list: /projects lists siblings, marks current | 12 | P1 | NEW |
| **INT-030** | **Bridge tenant isolation: Actor A cannot see/switch/capture sessions of tenant B** | **13** | **P0** | **PASS** |
| **INT-031** | **Bridge provider downgrade: Cursor stub blocks interactive escalation** | **13** | **P0** | **PASS** |
| **INT-032** | **Bridge HMAC: replay attack rejected (30s window)** | **15** | **P0** | **PASS** |
| **INT-033** | **Bridge HookServer: stop notification delivered with circuit breaker** | **15** | **P0** | **PASS** |
| **INT-034** | **Bridge permission: POST 202 -> poll GET -> approve via Decide()** | **16** | **P0** | **PASS** |
| **INT-035** | **Bridge fail-closed: high-risk tool auto-denied on 3min timeout** | **16** | **P0** | **PASS** |
| **INT-036** | **Bridge fail-open: low-risk tool auto-approved on timeout** | **16** | **P1** | **PASS** |
| **INT-037** | **Bridge dedup: same permission request returns same ID** | **16** | **P0** | **PASS** |
| **INT-038** | **Bridge double-apply: approve 2x returns "already decided"** | **16** | **P0** | **PASS** |
| **INT-039** | **Bridge SendText: 4-layer defense (tenant, capability, state, sanitizer)** | **17** | **P0** | **PASS** |
| **INT-040** | **Bridge busy queue: messages enqueued and drained on transition** | **17** | **P0** | **PASS** |
| **INT-041** | **Bridge cleanup: stopped sessions >24h removed, recent kept** | **17** | **P1** | **PASS** |
| **INT-042** | **Bridge rate limiter: 11th request in 1s returns 429** | **15** | **P1** | **PASS** |
| **INT-043** | **Bridge SOUL-aware launch: /cc launch --as coder -> persona injected, strategy resolved** | **18** | **P0** | **PASS** |
| **INT-044** | **Bridge install-agents: generates .claude/agents/*.md from SOUL files, idempotent** | **18** | **P0** | **PASS** |
| **INT-045** | **Bridge intelligence envelope: CreateSession populates PersonaEnvelope with strategy/hash** | **19** | **P1** | **PASS** |
| **INT-046** | **Bridge skills: install-agents also creates .claude/skills/sdlc-framework/SKILL.md** | **20A** | **P1** | **PASS** |
| **INT-047** | **Bridge CLAUDE.md: init-project generates project context file** | **20B** | **P1** | **PASS** |
| **INT-048** | **Bridge context sanitization: SetContext rejects malicious content via CheckInputSafe** | **20B** | **P0** | **PASS** |
| **INT-049** | **Bridge role defaults: executor role starts at patch, advisor at read** | **21** | **P1** | **PASS** |
| **INT-050** | **Bridge Cursor projection: GenerateCursorRules creates .cursor/rules/*.mdc files** | **23** | **P1** | **PASS** |
| **INT-051** | **Bridge provider registry: CursorProjectionAdapter registered, replaces StubAdapter** | **23** | **P1** | **PASS** |
| **INT-052** | **Claude CLI provider: subprocess execution + JSON parsing** | **24** | **P0** | **PASS** |
| **INT-053** | **Claude CLI filterEnv: ANTHROPIC_API_KEY + CLAUDE_API_KEY stripped from subprocess** | **24** | **P0** | **PASS** |
| **INT-054** | **Fallback chain: resolver picks first non-primary provider from MTCLAW_PROVIDER_CHAIN** | **24** | **P0** | **PASS** |
| **INT-055** | **Fallback guard: iteration=1 + tools blocks fallback (CTO-R2-1)** | **24** | **P0** | **PASS** |
| **INT-056** | **Fallback tracing: 2-span pattern — primary fail span + fallback success span** | **25** | **P0** | **PASS** |
| **INT-057** | **OTEL metadata propagation: fallback=true + primary_provider + primary_error in mtclaw.meta.*** | **25** | **P1** | **PASS** |
| **INT-058** | **Doctor: Claude CLI binary check + OAuth dir check + provider chain display** | **25** | **P1** | **PASS** |
| **INT-059** | **Discord channel: guild allowlist rejects non-allowed guilds** | **30** | **P0** | **PASS** |
| **INT-060** | **Discord channel: DM policy enforcement (open/disabled/pairing/allowlist)** | **30** | **P0** | **PASS** |
| **INT-061** | **Discord mention gate: require_mention=true filters unmentioned messages** | **30** | **P1** | **PASS** |
| **INT-062** | **Shared commands: ResolveAgentUUID parses UUID directly or falls back to store** | **33** | **P0** | **PASS** |
| **INT-063** | **Shared commands: PublishSpec routes to PM SOUL with rail=spec-factory metadata** | **33** | **P0** | **NEW** |
| **INT-064** | **Shared commands: PublishReview routes to reviewer SOUL with rail=pr-gate + pr_url** | **33** | **P0** | **NEW** |
| **INT-065** | **Shared commands: CommandMetadata.ToMap() skips empty fields** | **33** | **P1** | **PASS** |
| **INT-066** | **Discord /spec_list: shared ListSpecs returns formatted spec list** | **34** | **P1** | **NEW** |
| **INT-067** | **Discord /tasks: shared ListTasks returns formatted task list via teamStore** | **34** | **P1** | **NEW** |
| **INT-068** | **Discord /writers: shared WritersCmd.ListWriters returns writer list** | **34** | **P1** | **NEW** |
| **INT-069** | **Discord /addwriter: mention regex parses `<@123>` and `<@!123>` formats** | **34** | **P0** | **NEW** |
| **INT-070** | **Discord FactoryWithStores: expanded to (agentStore, teamStore, specStore)** | **34** | **P0** | **PASS** |
| **INT-071** | **Telegram /spec refactored: uses shared PublishSpec (same behavior, no regression)** | **33** | **P0** | **NEW** |
| **INT-072** | **Telegram /tasks refactored: uses shared FormatTaskDetail + MaxTasksInList (preserves inline keyboard UX)** | **34** | **P0** | **NEW** |

### 2.3 E2E Tests

| ID | Path | Description | Sprint | Status |
|----|------|-------------|--------|--------|
| E2E-001 | Onboarding | Telegram DM -> pairing -> first AI response | 4 | PASS |
| E2E-002 | Delegation | User -> @pm -> /spec -> JSON output | 7 | PASS |
| E2E-003 | Multi-tenant | MTS + NQH concurrent sessions | 6 | PASS |
| E2E-004 | MSTeams flow | Teams msg -> SOUL -> Adaptive Card | 10 | BLOCKED (Azure AD) |
| E2E-005 | PR Gate flow | GitHub PR -> webhook -> @reviewer -> verdict -> evidence link | 12 | NEW |
| E2E-006 | Spec quality | /spec -> quality gate -> accept/reject -> evidence chain | 12 | NEW |
| E2E-007 | Design gate | @coder task -> design gate check -> spec required | 12 | NEW |
| E2E-008 | Audit trail | Spec -> PR review -> chain build -> PDF export | 12 | NEW |
| E2E-009 | Channel cleanup | Verify Discord/Feishu/WhatsApp removed cleanly | 9 | NEW |
| E2E-010 | Workspace flow | /workspace show -> /projects list -> /workspace switch -> tool uses new dir | 12 | NEW |
| **E2E-011** | **Bridge launch** | **/cc link -> /cc launch -> /cc sessions -> /cc capture -> /cc kill** | **14** | **MANUAL** |
| **E2E-012** | **Bridge notify** | **Agent completes task -> stop hook -> HMAC verify -> Telegram notification** | **15** | **MANUAL** |
| **E2E-013** | **Bridge permission** | **Agent requests Bash -> hook -> Telegram keyboard -> Approve -> poll returns approved** | **16** | **MANUAL** |
| **E2E-014** | **Bridge interactive** | **/cc risk interactive -> /cc send "text" -> tmux receives -> /cc capture shows output** | **17** | **MANUAL** |
| **E2E-015** | **Bridge setup CLI** | **mtclaw bridge setup -> hooks created -> mtclaw bridge status -> all green** | **17** | **MANUAL** |
| **E2E-016** | **Bridge uninstall** | **mtclaw bridge uninstall -> hooks removed -> mtclaw bridge status -> warns** | **17** | **MANUAL** |
| **E2E-017** | **Bridge SOUL launch** | **/cc launch myproject --as coder -> session shows AgentRole=coder, PersonaSource=agent_file** | **18** | **MANUAL** |
| **E2E-018** | **Bridge install-agents** | **mtclaw bridge install-agents <path> -> .claude/agents/ created with 17 SOUL files** | **18** | **MANUAL** |
| **E2E-019** | **Bridge Cursor projection** | **mtclaw bridge install-agents --provider cursor -> .cursor/rules/*.mdc created** | **23** | **NEW** |
| **E2E-020** | **Fallback deploy** | **docker compose build -> claude --version in container -> mtclaw doctor shows Claude CLI** | **25** | **AUTOMATED (Sprint 28 — fallback_loop_test.go validates chain wiring)** |
| **E2E-021** | **Fallback E2E** | **Primary fails (429/500) -> fallback to claude-cli -> user gets response via Telegram** | **25** | **AUTOMATED (Sprint 28 — fallback_loop_test.go + fallback_test.go E2E scenarios)** |
| **E2E-022** | **Discord channel E2E** | **Discord DM -> SOUL routing -> AI response -> Discord send** | **30** | **MANUAL** |
| **E2E-023** | **Discord command parity** | **All 17 commands tested on Discord: /help, /status, /teams, /workspace, /projects, /reset, /stop, /stopall, /spec, /review, /spec_list, /spec_detail, /tasks, /task_detail, /writers, /addwriter, /removewriter** | **34** | **MANUAL** |
| **E2E-024** | **Telegram regression** | **All Telegram commands unchanged after Sprint 33 refactoring to shared package** | **33** | **MANUAL** |
| **E2E-025** | **Cross-channel parity** | **Same command on Telegram + Discord produces equivalent output (format may differ per platform)** | **34** | **MANUAL** |

### 2.4 Security Tests

| ID | Vector | Description | Sprint | Status |
|----|--------|-------------|--------|--------|
| SEC-001 | RLS bypass | All queries use owner_id filtering | 11 | PASS (structural) |
| SEC-002 | Cross-tenant API | Returns 404 not 403 | 11 | PASS (structural) |
| SEC-003 | SOUL injection | System prompt precedence | 11 | PASS (structural) |
| SEC-004 | JWT forgery | RS256 signature validation | 11 | PASS (structural) |
| SEC-005 | SOUL drift | RLS + content hashing | 11 | PASS (structural) |
| SEC-006 | Token exhaustion | 3-layer defense | 11 | PASS (structural) |
| SEC-007 | SSRF ServiceURL | URL allowlist (CTO-47) | 11 | PASS |
| SEC-008 | HMAC webhook | GitHub signature spoofing | 8 | PASS |
| **SEC-009** | **Bridge HMAC replay** | **Hook signature with expired timestamp rejected** | **15** | **PASS** |
| **SEC-010** | **Bridge cross-tenant** | **Session lookup returns "not found" (not 403) for wrong tenant** | **13** | **PASS** |
| **SEC-011** | **Bridge input sanitizer** | **87+ dangerous patterns blocked (rm -rf, curl pipe, env dump)** | **14** | **PASS** |
| **SEC-012** | **Bridge output redactor** | **API keys, tokens, DSN, JWT, PEM redacted from capture** | **14** | **PASS** |
| **SEC-013** | **Bridge fail-closed** | **High-risk tool denied on HookServer unreachable/timeout** | **16** | **PASS** |
| **SEC-014** | **Bridge ACL enforcement** | **Non-approver cannot Decide() on permission request** | **16** | **PASS** |
| **SEC-015** | **Bridge capability gate** | **Free text blocked in read/patch mode (structured_only)** | **17** | **PASS** |
| **SEC-016** | **Bridge provider guard** | **Interactive mode blocked if provider lacks permission hooks** | **13** | **PASS** |
| **SEC-017** | **Bridge path traversal** | **LoadSOUL rejects role with path traversal (../etc/passwd)** | **18** | **PASS** |
| **SEC-018** | **Bridge context injection** | **SetContext sanitizes via CheckInputSafe before storing (CTO-118)** | **20B** | **PASS** |
| **SEC-019** | **Bridge context length** | **SetContext enforces per-field (500) and total (2000) char caps (CTO-120)** | **20B** | **PASS** |
| **SEC-020** | **Bridge agent file override** | **Agent file permissionMode cannot bypass bridge D2 capability model** | **21** | **PASS (structural)** |
| **SEC-021** | **Claude CLI env isolation** | **ANTHROPIC_API_KEY + CLAUDE_API_KEY stripped from subprocess env (forces OAuth billing)** | **24** | **PASS** |
| **SEC-022** | **Docker read-only + Claude CLI** | **Container read_only:true, tmpfs without noexec, cap_drop:ALL, no-new-privileges** | **25** | **PASS (structural)** |

### 2.5 Performance Baseline (Sprint 11)

| Metric | Target | Measured | Status |
|--------|--------|----------|--------|
| API p95 latency | <100ms | ~80ms | PASS |
| Gateway startup | <5s | ~2s | PASS |
| SOUL load time | <1s | ~0.5s | PASS |
| Telegram polling connect | <3s | ~1s | PASS |
| **Fallback BuildPrompt** | **<1ms** | **~247ns** | **PASS (Sprint 28)** |
| **Fallback ParseResponse** | **<1ms** | **~3μs** | **PASS (Sprint 28)** |
| **Fallback FilterEnv** | **<1ms** | **~131ns** | **PASS (Sprint 28)** |
| **Fallback total overhead** | **<200ms** | **~50ms** | **PASS (Sprint 28, non-subprocess only)** |

---

## 3. Test Execution Matrix

### 3.1 By Sprint (cumulative)

| Sprint | Unit | Integration | E2E | Security | Total | Delta |
|--------|------|-------------|-----|----------|-------|-------|
| 1-3 | 45 | 3 | 0 | 0 | 48 | +48 |
| 4-5 | 78 | 10 | 2 | 0 | 90 | +42 |
| 6-7 | 112 | 15 | 3 | 0 | 130 | +40 |
| 8 | 149 | 19 | 3 | 1 | 172 | +42 |
| 9 | 229 | 20 | 3 | 1 | 253 | +81 |
| 10 | 250 | 25 | 3 | 8 | 286 | +33 |
| 11 | 257 | 25 | 3 | 8 | 293 | +7 |
| 12 | 280 | 31 | 9 | 8 | 328 | +35 |
| **13 (A1)** | **310** | **33** | **10** | **11** | **364** | **+36** |
| **14 (A2)** | **345** | **33** | **11** | **13** | **402** | **+38** |
| **15 (B)** | **385** | **37** | **12** | **14** | **448** | **+46** |
| **16 (C)** | **410** | **41** | **13** | **16** | **480** | **+32** |
| **17 (D)** | **430** | **43** | **16** | **16** | **505** | **+25** |
| **18** | **458** | **45** | **18** | **17** | **538** | **+33** |
| **19** | **470** | **46** | **18** | **17** | **551** | **+13** |
| **20A** | **478** | **47** | **18** | **17** | **560** | **+9** |
| **20B** | **487** | **48** | **18** | **19** | **572** | **+12** |
| **21** | **495** | **49** | **18** | **20** | **582** | **+10** |
| **22** | **495** | **49** | **18** | **20** | **582** | **+0** |
| **23** | **512** | **51** | **19** | **20** | **602** | **+20** |
| **24** | **529** | **55** | **19** | **21** | **624** | **+22** |
| **25** | **534** | **58** | **21** | **22** | **635** | **+11** |
| **26** | **544** | **60** | **21** | **22** | **647** | **+12** |
| **27** | **564** | **60** | **21** | **22** | **667** | **+20** |
| **28** | **586** | **60** | **21** | **22** | **689** | **+22** |
| **29** | **586** | **60** | **21** | **22** | **689** | **+0** |
| **30** | **599** | **63** | **22** | **22** | **706** | **+17** |
| **31** | **599** | **63** | **22** | **22** | **706** | **+0** |
| **32** | **599** | **63** | **22** | **22** | **706** | **+0** |
| **33** | **608** | **67** | **24** | **22** | **721** | **+15** |
| **34** | **618** | **72** | **25** | **22** | **737** | **+16** |

### 3.2 Traceability: Sprint 8-34 Features -> Tests

| Feature | Unit Tests | Integration | E2E | Security |
|---------|-----------|-------------|-----|----------|
| PR Gate ENFORCE (S8) | pr_processor_test | INT-016, INT-017, INT-018 | E2E-005 | SEC-008 |
| GitHub Webhook (S8) | webhook_github_test | INT-017, INT-018 | E2E-005 | SEC-008 |
| Context Drift (S8) | drift_e2e_test (25) | INT-025 | - | - |
| Audit PDF (S8) | pdf_builder_test (5) | INT-023 | E2E-008 | - |
| SOUL Behavioral (S9) | behavioral_test (80) | INT-024 | - | - |
| Channel Cleanup (S9) | - | - | E2E-009 | - |
| MSTeams Extension (S10) | msteams_test (21) | INT-011..015 | E2E-004 | SEC-007 |
| Pentest Vectors (S11) | pentest_test (7) | - | - | SEC-001..007 |
| Spec Quality (S12) | spec_quality_test (18) | INT-019 | E2E-006 | - |
| Design Gate (S12) | design_gate_test (17) | INT-020 | E2E-007 | - |
| Evidence Chain (S12) | (NEW: e2e test) | INT-021, INT-022 | E2E-005, E2E-008 | - |
| Workspace/Projects (S12) | commands_workspace_test | INT-026..029 | E2E-010 | - |
| **Bridge Session Core (S13)** | types, tmux, project, session, session_manager (30) | INT-030, INT-031 | E2E-011 | SEC-010, SEC-016 |
| **Bridge Security (S14)** | policy, sanitizer, redactor, audit (35) | - | E2E-011 | SEC-011, SEC-012 |
| **Bridge HookServer (S15)** | hook_auth, hook_server, notifier, health, transcript (40) | INT-032..033, INT-042 | E2E-012 | SEC-009 |
| **Bridge Permission (S16)** | permission_store (19), hook_server +7 | INT-034..038 | E2E-013 | SEC-013, SEC-014 |
| **Bridge Interactive (S17)** | session_manager +13 (SendText, Capture, Cleanup, Drain) | INT-039..041 | E2E-014..016 | SEC-015 |
| **SOUL-Aware Launch (S18)** | soul_loader (10), agent_installer (6), provider (6), session_manager +6 | INT-043, INT-044 | E2E-017, E2E-018 | SEC-017 |
| **Intelligence Envelope (S19)** | intelligence (12), session +3 | INT-045 | - | - |
| **Skills Integration (S20A)** | skills_generator (6), agent_installer +2 | INT-046 | - | - |
| **Project Context (S20B)** | claudemd_generator (5), session_manager +4 | INT-047, INT-048 | - | SEC-018, SEC-019 |
| **Role-Aware Defaults (S21)** | bridge_policy +8, provider +2, session_manager +3 | INT-049 | - | SEC-020 |
| **Agent Teams Spike (S22)** | (no production code — ADR-012 only) | - | - | - |
| **Provider Projection (S23)** | provider_cursor (17), provider registry +1 | INT-050, INT-051 | E2E-019 | - |
| **Provider Fallback Chain (S24)** | claude_cli_test (17) | INT-052..055 | - | SEC-021 |
| **Fallback Deploy + Observability (S25)** | fallback_test (5) | INT-056..058 | E2E-020, E2E-021 | SEC-022 |
| **Bridge Production Readiness (S26)** | bridge store tests (~10) | - | - | - |
| **Metrics + Integration + Hardening (S27)** | fallback_test +6 E2E, guardrails_test (9), tracing_adoption_test (5) | - | - | - |
| **Observability + Hardening (S28)** | health_tracker_test (11), fallback_loop_test (6), session_metrics_test (5), claude_cli_bench (3) | - | E2E-020 (AUTO), E2E-021 (AUTO) | - |
| **Discord Channel (S30)** | discord_test (13) | INT-059..061 | E2E-022 | - |
| **Shared Commands Extraction (S33)** | resolver_test (3), commands shared formatters | INT-062..065, INT-071 | E2E-024 | - |
| **Discord Command Parity (S34)** | discord_test updated, commands formatters | INT-066..070, INT-072 | E2E-023, E2E-025 | - |

### 3.3 Claude Code Bridge — 240 Unit Tests Breakdown

| Test File | Count | Sprint | Coverage Area |
|-----------|-------|--------|---------------|
| types_test.go | 10 | 13 | SessionID, HookSecret, Capabilities, Admission |
| tmux_test.go | 7 | 13 | Session names, validation, capture, sendKeys |
| project_test.go | 7 | 13 | Registry CRUD, fingerprint (deterministic, tenant, path) |
| session_test.go | 17 | 13+17+19 | State machine (13 transitions), risk mode (3), queue (2), ACL (3), touch, turn context (set/consume/accumulate/clear) |
| session_manager_test.go | 32 | 13+17+18+21 | Create, tenant isolation (3), kill (2), admission (4), risk guard, mismatch, SendText (6), CaptureOutput (3), Cleanup (2), TransitionDrain (2), persona strategy A/B/bare/invalid/stale, role defaults |
| bridge_policy_test.go | 16 | 14+21 | InputAllowed, CaptureAllowed, ToolAllowed (3 policies), RiskEscalation (7), role defaults (executor/advisor/unknown/missing), AllowedTools (executor/advisor), FormatAllowedTools, RoleDefaultOverridable |
| input_sanitizer_test.go | 5 | 14 | Pattern count, empty, safe, shell deny, bridge deny |
| output_redactor_test.go | 14 | 14 | Empty, no secrets, API keys, bearer, AWS, GitHub, DSN, password, env vars, JWT, PEM, heavy redact (2), truncate, pattern count |
| bridge_audit_test.go | 5 | 14 | JSONL write, multiple events, writable, default dir, nil DB |
| hook_auth_test.go | 7 | 15 | Deterministic sign, diff secrets, diff timestamps, verify (valid/wrong/tampered/replay/future/within-window) |
| hook_server_test.go | 11 | 15+16 | Health, missing headers, wrong method, invalid session, valid stop, wrong sig, permission (202, missing tool, dedup, poll, not found, wrong method, after decision), rate limiter |
| notifier_test.go | 4 | 15 | Stop message, redaction, circuit breaker, nil sendFn |
| transcript_test.go | 5 | 15 | Parse valid, malformed skip, empty, summarize, brief |
| health_test.go | 5 | 15 | Initial check, no dead, active detection, default interval, last status |
| permission_store_test.go | 19 | 16 | Create, missing fields, dedup, get, not found, decide (approve/deny/double/ACL), timeout (high/low), expiry, list, cleanup, get-by-hash, hash (deterministic/bucket/tools), isHighRisk |
| soul_loader_test.go | 10 | 18 | KnownRoles (scan/missing/cache), LoadSOUL (coder/unknown/path traversal/all roles), HashFileContent (2), frontmatter parsing, InvalidateRolesCache |
| agent_installer_test.go | 8 | 18+20A | LoadAgentTemplates, InstallAgents (create/idempotent/skip user/force/role overrides), IncludesSkills, AgentFileHasSkills |
| provider_test.go | 8 | 18+21 | LaunchCommand (StrategyA/B/C/Precedence/AllowedTools/NoAllowedTools), StubAdapter, EnvSanitization |
| intelligence_test.go | 12 | 19 | BuildPersonaEnvelope (5 cases), BuildIntelligenceEnvelope, JSON, OmitEmpty, StrategyFromPersonaSource, TurnContext (marshal/format/nil/empty/omit) |
| skills_generator_test.go | 6 | 20A | SDLCFrameworkSkill (under budget/content), InstallSkills (create/idempotent/skip user/force) |
| claudemd_generator_test.go | 9 | 20B | DetectProjectProfile (Go/TS/Python/Makefile/Docker), GenerateClaudeMD content, InitProject (create/skip user/force) |
| provider_cursor_test.go | 17 | 23 | CursorProjectionAdapter (7 methods), CursorRule FormatMDC (2), GenerateCursorRules (5), ProjectionInfo, Registry integration |
| **Total** | **240** | **13-23** | **All race-clean** |

---

## 4. CI/CD Gates

```yaml
# GitHub Actions gate (all must pass)
- make build            # Binary compiles
- make test             # Unit + integration tests
- make test-coverage    # Coverage report (target: 80%)
- make souls-validate   # SOUL YAML frontmatter + char budget
```

### Bridge-Specific CI Command

```bash
# Run bridge tests with race detector (recommended for CI)
go test ./internal/claudecode/... -race -count=1 -timeout=120s
```

---

## 5. Zero Mock Policy Exceptions

| Exception | Justification | Tag |
|-----------|---------------|-----|
| Bflow AI-Platform HTTP | External dependency, real response format used | `CI_MOCK_EXCEPTION: Bflow AI-Platform` |
| Bot Framework token endpoint | Azure AD creds required for live test | `CI_MOCK_EXCEPTION: Bot Framework live endpoint` |
| Bot Framework JWKS | RSA keys injected directly via injectTestKey() | `CI_MOCK_EXCEPTION: Bot Framework live endpoint` |
| Bridge tmux calls | tmux binary not available in CI containers | `CI_MOCK_EXCEPTION: tmux binary` |

| Claude CLI binary | claude binary not available in CI; fallback_test uses stubProvider test double | `CI_MOCK_EXCEPTION: Claude CLI binary` |

All exceptions use `httptest.NewServer` (real HTTP servers in test process), not mock objects. Bridge tests use `nil` tmux (no process spawning) to test all logic layers above tmux. Fallback tests use `stubProvider` (implements full Provider interface) to test wiring and retryable error classification without requiring the `claude` binary.

---

## 6. Blocked Items

| Item | Blocker | Owner | Impact |
|------|---------|-------|--------|
| E2E-004: MSTeams full flow | Azure AD provisioning | @devops | Medium (unit+integration cover core logic) |
| SEC-001: Live RLS test | Requires test PostgreSQL with RLS | @devops | Low (structural assertion covers policy) |
| E2E-011..016: Bridge manual tests | Requires live Telegram + tmux + Claude Code | @ceo | Medium (unit tests cover all logic) |
| E2E-017..018: SOUL launch manual tests | Requires live Telegram + tmux + Claude Code + SOUL files | @ceo | Medium (unit tests cover strategy resolution) |
| E2E-019: Cursor projection | Requires Cursor IDE installed | @devops | Low (unit tests cover file generation) |
| E2E-020: Fallback deploy | Docker build + claude login required | @devops | Medium (unit tests cover all logic) |
| E2E-021: Fallback E2E via Telegram | Requires primary provider failure (429/500) | @ceo | Medium (unit + structural tests cover chain) |
| E2E-022: Discord channel E2E | Requires live Discord bot + managed mode | @devops | Medium (unit tests cover routing logic) |
| E2E-023: Discord command parity | Requires live Discord + Telegram bots + managed mode + test data | @tester | High (manual test plan covers all 17 commands) |
| E2E-024: Telegram regression | Requires live Telegram bot + managed mode | @tester | High (ensures Sprint 33 refactoring didn't break) |
| E2E-025: Cross-channel parity | Requires both bots running simultaneously | @tester | Medium (verifies shared package produces consistent output) |

---

## 7. Risk Register

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Azure AD provisioning delays E2E-004 | Medium | Medium | Unit + integration cover 90% of MSTeams logic |
| Channel cleanup breaks existing flows | Low | High | E2E-009 verifies clean removal |
| Evidence chain data integrity | Low | High | E2E-005 + E2E-008 cover full flow |
| Spec quality threshold too strict/lenient | Medium | Medium | E2E-006 validates boundary (69/70) |
| Workspace switch breaks active agent session | Low | Medium | Cache invalidation broadcast + E2E-010 |
| **Bridge busy queue never drained** | **Low** | **Medium** | **CTO-94 fixed: TransitionSession + drainQueue** |
| **Bridge hook secret exposed in scripts** | **Low** | **Medium** | **CTO-96 fixed: scripts read from file at runtime** |
| **Bridge permission double-apply** | **Low** | **High** | **request_hash UNIQUE + Decide() check (19 tests)** |
| **Bridge tmux unavailable in production** | **Low** | **High** | **Graceful nil-tmux handling, health monitor detects** |
| **SOUL path traversal in bridge** | **Low** | **High** | **KnownRoles() pre-validation + filepath prefix check (SEC-017)** |
| **Context injection via sendKeys** | **Low** | **Medium** | **CheckInputSafe at SetContext time (CTO-118, SEC-018)** |
| **Agent file overrides bridge governance** | **Low** | **High** | **D2 is the only security boundary — VerifyBridgeOverridesAgentFile documents invariant (SEC-020)** |
| **Experimental Agent Teams API instability** | **Medium** | **Medium** | **ADR-012 NO-GO decision — deferred until API exits experimental** |
| **Provider parity illusion** | **Medium** | **Low** | **ADR-013: per-provider projection, not unified abstraction** |
| **Claude CLI npm on Alpine** | **Low** | **Medium** | **npm install in Dockerfile (not host binary mount); validated by E2E-020** |
| **OAuth token expiry in container** | **Low** | **Medium** | **Named Docker volume (claude-oauth) persists token; doctor warns if missing** |
| **Fallback fires excessively** | **Low** | **Medium** | **2-span tracing + OTEL metadata enables Grafana alerting (Sprint 26 backlog)** |
| **Health tracker false positives** | **Medium** | **Medium** | **Requires 3 consecutive failures to trip (not single), sliding window with time expiry (Sprint 28)** |
| **Circuit breaker cooldown too short** | **Low** | **Low** | **Configurable via MTCLAW_PROVIDER_CB_COOLDOWN env var, 30s default (Sprint 28)** |
| **Both circuits open (primary + fallback)** | **Low** | **Medium** | **Fail-open: still attempt fallback when no other option exists (OBS-028-1, Sprint 28)** |
| **Sprint 33 refactoring breaks Telegram commands** | **Low** | **High** | **Pure extraction — shared functions return `(string, error)`, channels format independently. TC-26 regression suite covers all 10 Telegram commands.** |
| **Discord command parity incomplete** | **Low** | **Medium** | **17 commands verified in PJM review (9.8/10) + CTO verification. TC-01..TC-19 manual test matrix.** |
| **Discord /addwriter mention parsing fails** | **Low** | **Medium** | **Regex `<@!?(\d+)>` tested for both `<@123>` and `<@!123>` formats (TC-17.4, TC-17.5)** |
| **Shared PublishSpec/Review SOUL routing incorrect** | **Low** | **High** | **AgentID hardcoded ("pm" / "reviewer") — cannot be overridden by callers. TC-13.5, TC-14.5 verify routing.** |
| **Discord FactoryWithStores expansion breaks existing** | **Low** | **Medium** | **discord_test.go updated (13 tests). All `New()` calls use 6 params. `make test` passes.** |

---

## 8. Manual Test Plan — Claude Code Bridge via Telegram

This section provides step-by-step instructions for manual E2E validation of the Bridge feature set (E2E-011 through E2E-019).

### Prerequisites

```bash
# 1. Ensure bridge is enabled in config.json
# Add to your config.json:
#   "bridge": { "enabled": true, "hookPort": 18792 }

# 2. Ensure tmux is installed
tmux -V   # expect: tmux 3.x

# 3. Run bridge setup to generate hook scripts
./mtclaw bridge setup

# 4. Verify bridge health
./mtclaw bridge status

# 5. Start gateway
./mtclaw
```

### TEST-M01: Bridge Status (E2E-015)

**Goal**: Verify `mtclaw bridge status` reports all checks.

```
Step 1: Run ./mtclaw bridge status
Expected:
  OK    bridge.enabled
  OK    tmux binary
         path: /usr/bin/tmux
  OK    hook port
         port: 18792
  OK    tmux sessions
         bridge sessions: 0
  OK    audit dir writable
         dir: ~/.mtclaw/bridge-audit
  OK    standalone store dir
         dir: ~/.mtclaw
  OK    health monitor
         health monitor prerequisites met

  Bridge status: 7 passed, 0 failed
```

### TEST-M02: Identity Linking (E2E-011)

**Goal**: Verify `/cc link` binds Telegram identity.

```
Step 1: In Telegram, send: /cc link
Expected: Confirmation message with your actor ID

Step 2: Send: /cc link (again)
Expected: "Already linked" or updated confirmation
```

### TEST-M03: Session Launch + List + Capture + Kill (E2E-011)

**Goal**: Full session lifecycle via Telegram.

```
Step 1: Send: /cc launch /home/nqh/shared/MTClaw
Expected: Session created message with session ID (br:XXXX:YYYY)
  - Shows project path
  - Shows risk mode: read
  - Shows status: active

Step 2: Send: /cc sessions
Expected: List of sessions showing:
  - Session ID
  - Status: active
  - Risk: read
  - Project path

Step 3: Send: /cc capture
Expected: Last 30 lines of tmux pane output
  - Any secrets should be [REDACTED]
  - Line count limited to 30 (read mode)

Step 4: Send: /cc kill
Expected: Session terminated message
  - Status changes to stopped

Step 5: Send: /cc sessions
Expected: No active sessions (or session shows stopped)
```

### TEST-M04: Risk Escalation (E2E-014)

**Goal**: Verify capability model and provider guard.

```
Step 1: Launch a session: /cc launch /home/nqh/shared/MTClaw
Step 2: Send: /cc risk patch
Expected: Risk mode changed to "patch"
  - /cc capture now shows up to 50 lines (standard redaction)

Step 3: Send: /cc risk interactive
Expected: Risk mode changed to "interactive"
  - Provider capability check passes (Claude Code supports hooks)

Step 4: Send: /cc risk read
Expected: Risk mode downgraded to "read"
  - Anyone can downgrade

Step 5: Kill the session: /cc kill
```

### TEST-M05: Free-Text Relay (E2E-014)

**Goal**: Verify `/cc send` delivers text to tmux pane.

```
Step 1: Launch a session: /cc launch /home/nqh/shared/MTClaw
Step 2: Escalate: /cc risk interactive
Step 3: Send: /cc send echo hello from telegram
Expected: "Sent to session br:XXXX:YYYY"

Step 4: Send: /cc capture
Expected: Output shows "hello from telegram" in the tmux capture

Step 5: Test rejection in read mode:
  /cc risk read
  /cc send echo test
Expected: Error about structured_only mode

Step 6: Test dangerous input:
  /cc risk interactive
  /cc send rm -rf /
Expected: Error about input blocked by sanitizer

Step 7: Kill: /cc kill
```

### TEST-M06: Project Registration (E2E-011)

**Goal**: Verify project CRUD via Telegram.

```
Step 1: Send: /cc register mtclaw /home/nqh/shared/MTClaw
Expected: Project registered message

Step 2: Send: /cc projects
Expected: List showing "mtclaw" with path

Step 3: Launch with project name: /cc launch mtclaw
Expected: Session created for /home/nqh/shared/MTClaw

Step 4: Kill: /cc kill
```

### TEST-M07: Bridge Setup + Uninstall CLI (E2E-015, E2E-016)

**Goal**: Verify hook script lifecycle.

```
Step 1: Run: ./mtclaw bridge setup
Expected:
  OK    Bridge setup complete.
         hooks dir: ~/.claude/hooks
         stop hook: ~/.claude/hooks/stop.sh
         permission hook: ~/.claude/hooks/permission-request.sh
         secret file: ~/.mtclaw/bridge-hook-secret (0600)

Step 2: Verify hook scripts read secret from file (CTO-96):
  grep 'SECRET_FILE=' ~/.claude/hooks/stop.sh
Expected: SECRET_FILE="/home/<user>/.mtclaw/bridge-hook-secret"
  (NOT an inline HMAC secret)

Step 3: Verify permissions:
  ls -la ~/.mtclaw/bridge-hook-secret
Expected: -rw------- (0600)

  ls -la ~/.claude/hooks/stop.sh
Expected: -rwx------ (0700)

Step 4: Run: ./mtclaw bridge uninstall
Expected:
  OK    removed ~/.claude/hooks/stop.sh
  OK    removed ~/.claude/hooks/permission-request.sh
  OK    removed ~/.mtclaw/bridge-hook-secret
  Removed 3 hook files.

Step 5: Run: ./mtclaw bridge uninstall (again)
Expected:
  OK    No bridge hooks found to remove.
```

### TEST-M08: Stop Notification (E2E-012)

**Goal**: Verify stop hook delivers Telegram notification.

```
Prerequisites: Gateway running with bridge.enabled=true

Step 1: Launch session: /cc launch /home/nqh/shared/MTClaw
Step 2: Note the session ID from the response

Step 3: From another terminal, simulate a stop hook:
  SESSION_ID="<session-id-from-step-2>"
  SECRET=$(cat ~/.mtclaw/bridge-hook-secret)
  TIMESTAMP=$(date +%s)
  BODY='{"event":"stop","exit_code":0,"summary":"Task complete"}'
  SIGNATURE=$(echo -n "${TIMESTAMP}.${BODY}" | openssl dgst -sha256 -hmac "$SECRET" -hex | awk '{print $NF}')

  curl -s -X POST http://127.0.0.1:18792/hook \
    -H "Content-Type: application/json" \
    -H "X-Hook-Signature: $SIGNATURE" \
    -H "X-Hook-Timestamp: $TIMESTAMP" \
    -H "X-Hook-Session: $SESSION_ID" \
    -d "$BODY"

Expected in Telegram: Stop notification with:
  - Session ID
  - Status: completed
  - Project path
  - Summary: "Task complete"
```

### TEST-M09: Permission Approval (E2E-013)

**Goal**: Verify Telegram inline keyboard approval flow.

```
Prerequisites: Session running, gateway with bridge enabled

Step 1: Simulate a permission request hook:
  BODY='{"event":"permission","tool":"Bash","tool_input":{"command":"npm install"}}'
  TIMESTAMP=$(date +%s)
  SIGNATURE=$(echo -n "${TIMESTAMP}.${BODY}" | openssl dgst -sha256 -hmac "$SECRET" -hex | awk '{print $NF}')

  curl -s -X POST http://127.0.0.1:18792/hook \
    -H "Content-Type: application/json" \
    -H "X-Hook-Signature: $SIGNATURE" \
    -H "X-Hook-Timestamp: $TIMESTAMP" \
    -H "X-Hook-Session: $SESSION_ID" \
    -d "$BODY"

Expected response: HTTP 202 with JSON {"id":"<perm-id>","decision":"pending",...}

Step 2: Check Telegram for inline keyboard message with:
  - Tool: Bash
  - Risk level
  - [Approve] [Deny] buttons

Step 3: Tap [Approve] in Telegram
Expected: Message updates to "Approved by <actor>"

Step 4: Poll the permission endpoint:
  curl -s http://127.0.0.1:18792/hook/permission/<perm-id>

Expected: {"decision":"approved",...}
```

### TEST-M10: SOUL-Aware Launch (E2E-017)

**Goal**: Verify `/cc launch --as` injects SOUL persona.

```
Prerequisites: Agent files installed via mtclaw bridge install-agents

Step 1: Install agent files:
  ./mtclaw bridge install-agents /home/nqh/shared/MTClaw

Step 2: Verify agent files exist:
  ls .claude/agents/
Expected: coder.md, pm.md, architect.md, etc.

Step 3: Launch with role:
  /cc launch /home/nqh/shared/MTClaw --as coder
Expected: Session created with:
  - AgentRole: coder
  - PersonaSource: agent_file (Strategy A)
  - RiskMode: patch (executor default)

Step 4: Check /cc sessions
Expected: Shows role=coder, source=agent_file, strategy=A

Step 5: Launch without role:
  /cc kill
  /cc launch /home/nqh/shared/MTClaw
Expected: Session created with:
  - AgentRole: (empty)
  - PersonaSource: bare (Strategy C)
  - RiskMode: read (default)

Step 6: Kill: /cc kill
```

### TEST-M11: Install Agents CLI (E2E-018)

**Goal**: Verify install-agents creates files correctly.

```
Step 1: Run install-agents:
  ./mtclaw bridge install-agents /tmp/test-project --souls-dir docs/08-collaborate/souls

Step 2: Verify files:
  ls /tmp/test-project/.claude/agents/
Expected: 17 .md files (one per SOUL)

  ls /tmp/test-project/.claude/skills/sdlc-framework/
Expected: SKILL.md

Step 3: Verify generated header:
  head -1 /tmp/test-project/.claude/agents/coder.md
Expected: # Generated by mtclaw bridge install-agents (claude-code >= 2.x) — do not edit manually

Step 4: Run again (idempotent):
  ./mtclaw bridge install-agents /tmp/test-project --souls-dir docs/08-collaborate/souls
Expected: 0 installed, 0 updated, 17 skipped (no changes)

Step 5: Create a user file and verify it's preserved:
  echo "My custom agent" > /tmp/test-project/.claude/agents/custom.md
  ./mtclaw bridge install-agents /tmp/test-project --souls-dir docs/08-collaborate/souls
Expected: custom.md not overwritten (skipped)
```

### TEST-M12: Fallback Docker Deploy (E2E-020)

**Goal**: Verify Claude CLI installs and runs inside Alpine container.

```
Step 1: Build with Claude CLI enabled:
  ENABLE_CLAUDE_CLI=true docker compose build --no-cache mtclaw

Step 2: Start container:
  docker compose up -d mtclaw

Step 3: Verify claude binary:
  docker compose exec mtclaw claude --version
Expected: @anthropic-ai/claude-code/2.x.x

Step 4: Verify doctor output:
  docker compose exec mtclaw ./mtclaw doctor
Expected:
  Claude CLI (fallback):
    Binary:      /usr/local/bin/claude (or /usr/bin/claude)
    Version:     2.x.x
    Model:       sonnet
    Timeout:     120s
    OAuth:       /app/.claude (OK)
  Provider Chain: bflow-ai-platform → claude-cli

Step 5: Verify OAuth persistence across restart:
  docker compose exec mtclaw claude login
  docker compose restart mtclaw
  docker compose exec mtclaw ls /app/.claude/
Expected: OAuth config files persist (claude-oauth volume)
```

### TEST-M13: Fallback E2E via Telegram (E2E-021)

**Goal**: Verify fallback activates on primary provider failure.

```
Prerequisites: Container running with Claude CLI enabled + logged in

Step 1: Send a message to Telegram bot
Expected: Normal response from bflow-ai-platform (primary)

Step 2: Simulate primary failure by temporarily stopping bflow gateway:
  docker compose stop bflow-ai-gateway-staging
  (or set MTCLAW_BFLOW_BASE_URL to invalid host)

Step 3: Send another message to Telegram bot
Expected:
  - Response arrives (from claude-cli fallback)
  - slog shows: "primary provider failed, trying fallback"
  - slog shows: "fallback provider succeeded"

Step 4: Check traces in dashboard or OTEL:
  - Primary fail span visible (status=error, provider=bflow-ai-platform)
  - Fallback success span visible (status=completed, provider=claude-cli)
  - Fallback span metadata: fallback=true, primary_provider=bflow-ai-platform

Step 5: Restore primary:
  docker compose start bflow-ai-gateway-staging
  Send another message
Expected: Response from bflow-ai-platform again (primary restored)

Step 6: Record latency:
  Primary response time: ___ms
  Fallback response time: ___ms (expected: higher due to claude CLI subprocess)
```

### Test Result Tracking

| Test ID | Description | Tester | Date | Result | Notes |
|---------|-------------|--------|------|--------|-------|
| TEST-M01 | Bridge status | | | | |
| TEST-M02 | Identity linking | | | | |
| TEST-M03 | Session lifecycle | | | | |
| TEST-M04 | Risk escalation | | | | |
| TEST-M05 | Free-text relay | | | | |
| TEST-M06 | Project registration | | | | |
| TEST-M07 | Setup + uninstall CLI | | | | |
| TEST-M08 | Stop notification | | | | |
| TEST-M09 | Permission approval | | | | |
| TEST-M10 | SOUL-aware launch | | | | |
| TEST-M11 | Install agents CLI | | | | |
| TEST-M12 | Fallback Docker deploy | | | | |
| TEST-M13 | Fallback E2E Telegram | | | | |

---

## 9. Manual Test Plan — Sprint 33+34: Unified Command Routing (E2E-023, E2E-024, E2E-025)

**Target**: MTClaw repo (`/home/nqh/shared/MTClaw`)
**Test Type**: Manual testing against live Discord + Telegram bots
**Scope**: Sprint 33 (Shared Commands Package) + Sprint 34 (Discord Command Parity)

### Prerequisites

| Item | Requirement |
|------|-------------|
| MTClaw binary | Built from current branch (`make build`) |
| PostgreSQL | Running with migrations applied (`make migrate-up`) |
| Discord bot | Token configured in `.env` or DB `channel_instances` |
| Telegram bot | Token configured in `.env` or DB `channel_instances` |
| Managed mode | `MTCLAW_POSTGRES_DSN` set, stores wired via `FactoryWithStores` |
| Agent configured | At least 1 agent with workspace set |
| Team configured | At least 1 team with tasks |
| Spec store | At least 1 governance spec in DB |

### Test Data Setup

```sql
-- Verify agent exists with workspace
SELECT id, key, workspace FROM agents WHERE owner_id = '<tenant_id>' LIMIT 1;

-- Verify team exists with tasks
SELECT t.id, t.name, COUNT(tt.id) as task_count
FROM agent_teams t
LEFT JOIN agent_team_tasks tt ON t.id = tt.team_id
WHERE t.owner_id = '<tenant_id>'
GROUP BY t.id, t.name;

-- Verify specs exist
SELECT spec_id, title, status FROM governance_specs
WHERE owner_id = '<tenant_id>'
ORDER BY created_at DESC LIMIT 5;
```

### Legend

| Symbol | Meaning |
|--------|---------|
| P | PASS |
| F | FAIL |
| S | SKIP (precondition not met) |
| N/A | Not applicable for this channel |

### TC-CMD-01: /help

| # | Step | Expected Result | TG | DC |
|---|------|-----------------|----|----|
| 01.1 | Send `/help` in DM | Shows full command list | | |
| 01.2 | Send `/help` in group | Shows full command list | | |
| 01.3 | Verify Telegram `/help` includes `/cc` commands | /cc section present | | N/A |
| 01.4 | Verify Discord `/help` does NOT include `/cc` | No /cc section | N/A | |
| 01.5 | Verify Discord `/help` lists `/addwriter @user` | Mention-based syntax shown | N/A | |
| 01.6 | Verify Telegram `/help` lists `/addwriter` reply-to syntax | Reply-to syntax shown | | N/A |

### TC-CMD-02: /status

| # | Step | Expected Result | TG | DC |
|---|------|-----------------|----|----|
| 02.1 | Send `/status` in DM | Shows "Bot status: Running" + channel name | | |
| 02.2 | Telegram shows bot username | `Bot: @<bot_username>` present | | N/A |
| 02.3 | Discord shows "Channel: Discord" | Channel name correct | N/A | |

### TC-CMD-03: /teams

| # | Step | Expected Result | TG | DC |
|---|------|-----------------|----|----|
| 03.1 | Send `/teams` | Shows 3 teams: engineering, business, advisory | | |
| 03.2 | Verify team leads shown | @pm, @assistant, @cto | | |
| 03.3 | Verify usage hint | "Use @team_name" instruction present | | |

### TC-CMD-04: /workspace

| # | Step | Expected Result | TG | DC |
|---|------|-----------------|----|----|
| 04.1 | Send `/workspace` (no arg) | Shows current workspace path | | |
| 04.2 | Send `/workspace /home/nqh/shared/MTClaw` | Updates workspace, confirms new path | | |
| 04.3 | Send `/workspace` again | Shows updated path from step 04.2 | | |
| 04.4 | Send `/workspace /nonexistent/path` | Error: path does not exist | | |
| 04.5 | Send `/workspace /etc/passwd` | Error: not a directory | | |
| 04.6 | Without agentStore (standalone) | "Workspace commands require managed mode." | | |

### TC-CMD-05: /projects

| # | Step | Expected Result | TG | DC |
|---|------|-----------------|----|----|
| 05.1 | Send `/projects` (workspace set) | Lists sibling directories | | |
| 05.2 | Current workspace marked with `>` | `> N. <current_dir>` visible | | |
| 05.3 | Hidden dirs (`.xxx`) excluded | No dot-prefixed entries | | |
| 05.4 | Switch hint shown | `Switch: /workspace <parent>/<name>` | | |
| 05.5 | Without agentStore (standalone) | "Project commands require managed mode." | | |

### TC-CMD-06: /spec_list

| # | Step | Expected Result | TG | DC |
|---|------|-----------------|----|----|
| 06.1 | Send `/spec_list` (specs exist) | "Recent Specifications:" header + numbered list | | |
| 06.2 | Each spec shows: `N. SPEC-ID — Title [Status]` | Format matches | | |
| 06.3 | Max 10 specs shown | List capped at 10 | | |
| 06.4 | Footer shows usage hint | "Use /spec_detail <SPEC-ID>" | | |
| 06.5 | Send `/spec-list` (hyphenated) | Same result as `/spec_list` | | |
| 06.6 | No specs in DB | "No specs found. Use /spec..." | | |
| 06.7 | Without specStore | "Spec features are not available." | | |

### TC-CMD-07: /spec_detail

| # | Step | Expected Result | TG | DC |
|---|------|-----------------|----|----|
| 07.1 | Send `/spec_detail SPEC-2026-0001` | Full spec detail shown | | |
| 07.2 | Verify header: `SPEC-ID — Title` | Present | | |
| 07.3 | Verify fields: Status, Priority, Effort, Author, Version, Created | All present | | |
| 07.4 | Verify narrative: As a / I want / So that | If narrative exists in spec | | |
| 07.5 | Verify acceptance criteria | If criteria exist in spec | | |
| 07.6 | Verify risks section | If risks exist in spec | | |
| 07.7 | Send `/spec_detail` (no ID) | Usage message shown | | |
| 07.8 | Send `/spec_detail NONEXISTENT` | "spec not found" error | | |
| 07.9 | Send `/spec-detail SPEC-2026-0001` | Same result as underscore variant | | |
| 07.10 | Case-insensitive: `/spec_detail spec-2026-0001` | Finds spec (uppercased internally) | | |

### TC-CMD-08: /tasks

| # | Step | Expected Result | TG | DC |
|---|------|-----------------|----|----|
| 08.1 | Send `/tasks` (team has tasks) | "Tasks for team \"name\" (N):" header | | |
| 08.2 | Each task shows: `N. [icon] Subject — @owner` | Format matches | | |
| 08.3 | Telegram uses emoji icons (✅🔄⛔⏳) | Emoji visible | | N/A |
| 08.4 | Discord uses text icons (done, >>, !!, ..) | Text icons visible | N/A | |
| 08.5 | Max 30 tasks shown | "showing 30 of N" if >30 tasks | | |
| 08.6 | Telegram shows inline keyboard buttons | Tap-to-view buttons present | | N/A |
| 08.7 | Discord shows "Use /task_detail <id>" footer | Text footer present | N/A | |
| 08.8 | No tasks | "No tasks for team \"name\"." | | |
| 08.9 | Agent not in team | "This agent is not part of any team." | | |
| 08.10 | Without teamStore | "Team features are not available." | | |

### TC-CMD-09: /task_detail

| # | Step | Expected Result | TG | DC |
|---|------|-----------------|----|----|
| 09.1 | Send `/task_detail <full-uuid>` | Full task detail shown | | |
| 09.2 | Verify fields: Subject, ID, Status, Owner, Priority, Created | All present | | |
| 09.3 | Verify Description section | If task has description | | |
| 09.4 | Verify Result section | If task has result | | |
| 09.5 | Verify BlockedBy section | If task has blockers | | |
| 09.6 | UUID prefix match: `/task_detail <first-8-chars>` | Finds task by prefix | | |
| 09.7 | Send `/task_detail` (no ID) | "Usage: /task_detail <task_id>" | | |
| 09.8 | Send `/task_detail nonexistent` | "task not found" error | | |
| 09.9 | Telegram callback button tap (td:uuid) | Shows same task detail | | N/A |

### TC-CMD-10: /reset

| # | Step | Expected Result | TG | DC |
|---|------|-----------------|----|----|
| 10.1 | Send `/reset` in DM | "Conversation history has been reset." | | |
| 10.2 | Send a follow-up message | Agent has no prior context | | |
| 10.3 | Send `/reset` in group | Same confirmation message | | |
| 10.4 | Verify metadata: `command=reset, platform=<channel>` | Check server logs for InboundMessage metadata | | |

### TC-CMD-11: /stop

| # | Step | Expected Result | TG | DC |
|---|------|-----------------|----|----|
| 11.1 | Start a long task, then send `/stop` | Task cancelled, feedback from consumer | | |
| 11.2 | Send `/stop` with no running task | No error (silent, no crash) | | |
| 11.3 | Verify metadata: `command=stop, platform=<channel>` | Check server logs | | |

### TC-CMD-12: /stopall

| # | Step | Expected Result | TG | DC |
|---|------|-----------------|----|----|
| 12.1 | Start multiple tasks, send `/stopall` | All tasks cancelled | | |
| 12.2 | Verify metadata: `command=stopall, platform=<channel>` | Check server logs | | |

### TC-CMD-13: /spec (Bus-Routed)

| # | Step | Expected Result | TG | DC |
|---|------|-----------------|----|----|
| 13.1 | Send `/spec Create login feature` | Ack message shown, then PM SOUL generates spec | | |
| 13.2 | Telegram ack: "📋 Generating spec..." | Emoji ack visible | | N/A |
| 13.3 | Discord ack: "Generating spec..." | Text ack visible (no emoji) | N/A | |
| 13.4 | Verify bus metadata: `command=spec, rail=spec-factory` | Check logs for metadata | | |
| 13.5 | Verify AgentID routes to "pm" | PM SOUL processes the request | | |
| 13.6 | Send `/spec` (no description) | Usage message with example | | |
| 13.7 | Generated spec appears in `/spec_list` | New spec listed after generation | | |

### TC-CMD-14: /review (Bus-Routed)

| # | Step | Expected Result | TG | DC |
|---|------|-----------------|----|----|
| 14.1 | Send `/review https://github.com/org/repo/pull/123` | Ack message, reviewer SOUL processes | | |
| 14.2 | Telegram ack: "🔍 Reviewing PR..." | Emoji ack visible | | N/A |
| 14.3 | Discord ack: "Reviewing PR..." | Text ack visible (no emoji) | N/A | |
| 14.4 | Verify bus metadata: `command=review, rail=pr-gate, pr_url=<url>` | Check logs | | |
| 14.5 | Verify AgentID routes to "reviewer" | Reviewer SOUL processes | | |
| 14.6 | Send `/review` (no URL) | Usage message with example | | |
| 14.7 | Send `/review not-a-pr-url` | Usage message (must contain /pull/) | | |

### TC-CMD-15: /writers (Group-Only)

| # | Step | Expected Result | TG | DC |
|---|------|-----------------|----|----|
| 15.1 | Send `/writers` in group (writers configured) | "File writers for this group (N):" | | |
| 15.2 | Each writer: `N. @username (ID: xxx)` | Format matches | | |
| 15.3 | Send `/writers` in DM | "This command only works in group chats." | | |
| 15.4 | No writers configured | "No file writers configured..." auto-add message | | |
| 15.5 | Without agentStore | "File writer management is not available." | | |

### TC-CMD-16: /addwriter (Telegram — reply-to)

| # | Step | Expected Result | TG | DC |
|---|------|-----------------|----|----|
| 16.1 | Reply to user's message with `/addwriter` | "Added @username as a file writer." | | N/A |
| 16.2 | Send `/addwriter` without reply | Instruction to reply to a message | | N/A |
| 16.3 | Non-writer tries to add | "Only existing file writers can manage..." | | N/A |
| 16.4 | Verify cache invalidation event broadcast | Check logs for `EventCacheInvalidate` | | N/A |

### TC-CMD-17: /addwriter (Discord — mention)

| # | Step | Expected Result | TG | DC |
|---|------|-----------------|----|----|
| 17.1 | Send `/addwriter @user` in group | "Added @user as a file writer." (or ID-based) | N/A | |
| 17.2 | Send `/addwriter` without mention | "To add a writer, mention them: /addwriter @user" | N/A | |
| 17.3 | Non-writer tries to add | "only existing file writers can manage..." | N/A | |
| 17.4 | Verify mention regex parses `<@123456>` | Writer added with correct numeric ID | N/A | |
| 17.5 | Verify mention regex parses `<@!123456>` (nickname) | Writer added with correct numeric ID | N/A | |

### TC-CMD-18: /removewriter (Telegram — reply-to)

| # | Step | Expected Result | TG | DC |
|---|------|-----------------|----|----|
| 18.1 | Reply to writer's message with `/removewriter` | "Removed @username from file writers." | | N/A |
| 18.2 | Try to remove last writer | "Cannot remove the last file writer." | | N/A |
| 18.3 | Non-writer tries to remove | "Only existing file writers can manage..." | | N/A |

### TC-CMD-19: /removewriter (Discord — mention)

| # | Step | Expected Result | TG | DC |
|---|------|-----------------|----|----|
| 19.1 | Send `/removewriter @user` in group | "Removed @user from file writers." | N/A | |
| 19.2 | Try to remove last writer | "cannot remove the last file writer" | N/A | |
| 19.3 | Non-writer tries to remove | "only existing file writers can manage..." | N/A | |

### TC-CMD-20: Cross-Cutting — Metadata Correctness

| # | Step | Expected Result | TG | DC |
|---|------|-----------------|----|----|
| 20.1 | Telegram `/reset`: Check metadata has `platform=telegram` | Correct platform | | N/A |
| 20.2 | Discord `/reset`: Check metadata has `platform=discord` | Correct platform | N/A | |
| 20.3 | Telegram `/spec`: Check metadata has `rail=spec-factory` | Rail auto-set | | N/A |
| 20.4 | Discord `/spec`: Check metadata has `rail=spec-factory` | Rail auto-set | N/A | |
| 20.5 | Telegram `/review`: Check `pr_url` in metadata | PR URL present | | N/A |
| 20.6 | Discord `/review`: Check `pr_url` in metadata | PR URL present | N/A | |
| 20.7 | Telegram forum: Check `local_key`, `is_forum`, `message_thread_id` | Forum-specific fields present | | N/A |
| 20.8 | Discord: Verify NO `local_key`/`is_forum`/`message_thread_id` | Empty fields skipped by ToMap() | N/A | |

### TC-CMD-21: Cross-Cutting — Message Chunking

| # | Step | Expected Result | TG | DC |
|---|------|-----------------|----|----|
| 21.1 | Command response >4096 chars (Telegram limit) | Split into multiple messages | | N/A |
| 21.2 | Command response >2000 chars (Discord limit) | Split into chunks at newline boundaries | N/A | |
| 21.3 | Verify chunk split prefers newline boundary | No mid-word splits | N/A | |

### TC-CMD-22: Telegram Regression (Sprint 33 Refactoring)

| # | Step | Expected Result | TG | DC |
|---|------|-----------------|----|----|
| 22.1 | `/workspace` GET returns same format as pre-Sprint 33 | Output format unchanged | | N/A |
| 22.2 | `/workspace <path>` SET updates workspace + reloads PROJECT.md | Cache invalidation broadcast in logs | | N/A |
| 22.3 | `/projects` returns same format as pre-Sprint 33 | Output format unchanged | | N/A |
| 22.4 | `/spec_list` returns same format as pre-Sprint 33 | Output format unchanged | | N/A |
| 22.5 | `/spec_detail` returns same format as pre-Sprint 33 | Narrative, criteria, risks sections unchanged | | N/A |
| 22.6 | `/tasks` inline keyboard still works | Buttons appear and respond to taps | | N/A |
| 22.7 | `/task_detail` returns same format as pre-Sprint 33 | All fields present and formatted correctly | | N/A |
| 22.8 | `/addwriter` via reply-to still works | Writer added, cache invalidated | | N/A |
| 22.9 | `/writers` returns same format as pre-Sprint 33 | List format unchanged | | N/A |
| 22.10 | `/cc launch` still works | Claude Code session starts normally | | N/A |

### TC-CMD-23: Discord Regression (Sprint 33 Refactoring)

| # | Step | Expected Result | TG | DC |
|---|------|-----------------|----|----|
| 23.1 | `/workspace` GET works on Discord | Current workspace shown | N/A | |
| 23.2 | `/projects` works on Discord | Project list shown | N/A | |
| 23.3 | `/reset` works on Discord | Confirmation + history cleared | N/A | |
| 23.4 | `/stop` works on Discord | No crash, task stopped if running | N/A | |
| 23.5 | `/stopall` works on Discord | No crash, all tasks stopped if running | N/A | |

### TC-CMD-24: Discord Platform-Specific

| # | Step | Expected Result | TG | DC |
|---|------|-----------------|----|----|
| 24.1 | Guild allowlist: command from allowlisted guild | Command processed normally | N/A | |
| 24.2 | Guild allowlist: command from non-allowlisted guild | Message silently ignored | N/A | |
| 24.3 | `require_mention=true`: `/help` without @bot | Message ignored | N/A | |
| 24.4 | `require_mention=true`: `@bot /help` | Command processed, bot mention stripped | N/A | |
| 24.5 | `require_mention=false`: `/help` without @bot | Command processed normally | N/A | |

### TC-CMD-25: DM Policy Enforcement

| # | Step | Expected Result | TG | DC |
|---|------|-----------------|----|----|
| 25.1 | `dm_policy=open`: Send DM command | Command processed | | |
| 25.2 | `dm_policy=disabled`: Send DM command | DM rejected silently | | |
| 25.3 | `dm_policy=pairing`: Unpaired user sends DM | Pairing code message sent | | |
| 25.4 | `dm_policy=allowlist`: Allowed user sends DM | Command processed | | |
| 25.5 | `dm_policy=allowlist`: Non-allowed user sends DM | DM rejected silently | | |

### TC-CMD-26: Error Handling & Edge Cases

| # | Step | Expected Result | TG | DC |
|---|------|-----------------|----|----|
| 26.1 | `/workspace` without agentStore | "Workspace commands require managed mode." | | |
| 26.2 | `/projects` without agentStore | "Project commands require managed mode." | | |
| 26.3 | `/spec_list` without specStore | "Spec features are not available." | | |
| 26.4 | `/tasks` without teamStore | "Team features are not available." | | |
| 26.5 | `/writers` without agentStore | "File writer management is not available." | | |
| 26.6 | Send `/` alone | Not handled (returns false, passes to agent) | | |
| 26.7 | Send `/unknowncommand` | Not handled, passes to agent loop | | |
| 26.8 | Send `/WORKSPACE` (uppercase) | Handled (case-insensitive) | | |
| 26.9 | Send `/workspace   ` (trailing spaces) | Treated as GET (trimmed to empty) | | |

### Pre-Test Verification

```bash
# Must pass before manual testing
make build
go test ./internal/commands/ ./internal/channels/discord/ ./internal/channels/telegram/ -v -count=1
```

### Test Execution Log

| Field | Value |
|-------|-------|
| Tester | |
| Date | |
| MTClaw version | `git rev-parse --short HEAD` |
| Branch | |
| Telegram bot | @________________ |
| Discord bot | ________________ |
| Managed mode | Yes / No |
| Agent key | |
| Team name | |

### Summary

| Category | Total | Pass | Fail | Skip |
|----------|-------|------|------|------|
| TC-CMD-01..09: Channel-local commands | 60 | | | |
| TC-CMD-10..14: Bus-routed commands | 22 | | | |
| TC-CMD-15..19: Writers commands | 19 | | | |
| TC-CMD-20..21: Cross-cutting | 11 | | | |
| TC-CMD-22..23: Regression | 15 | | | |
| TC-CMD-24..25: Platform-specific | 10 | | | |
| TC-CMD-26: Error handling | 9 | | | |
| **Total** | **146** | | | |

### Defects Found

| # | TC | Severity | Description | Status |
|---|-----|----------|-------------|--------|
| | | | | |

### Sprint 33 AC Cross-Reference

| AC | Description | Test Cases |
|----|-------------|------------|
| AC-1 | `make test` passes | Pre-test verification |
| AC-2 | `make build` produces binary | Pre-test verification |
| AC-3 | 7 extracted functions in commands/ | Code review (verified in PJM review) |
| AC-4 | Responder interface by both channels | Code review (verified in PJM review) |
| AC-5 | CommandMetadata with ToMap() | TC-CMD-20 |
| AC-6 | ResolveAgentUUID 3 tests | Unit test suite |
| AC-7 | No duplicated reloadProjectContext | Code review (verified in PJM review) |
| AC-8 | Manual testing Telegram | TC-CMD-22 |
| AC-9 | Manual testing Discord | TC-CMD-23 |

### Sprint 34 AC Cross-Reference

| AC | Description | Test Cases |
|----|-------------|------------|
| AC-1 | `make test` passes | Pre-test verification |
| AC-2 | /spec routes to PM SOUL | TC-CMD-13 |
| AC-3 | /review routes to reviewer | TC-CMD-14 |
| AC-4 | /spec_list shows specs | TC-CMD-06 |
| AC-5 | /tasks shows tasks | TC-CMD-08 |
| AC-6 | /writers lists writers | TC-CMD-15 |
| AC-7 | /addwriter @user parses mention | TC-CMD-17 |
| AC-8 | /help lists all commands | TC-CMD-01 |
| AC-9 | Telegram unchanged | TC-CMD-22 |
| AC-10 | Shared formatters by both | TC-CMD-06..09 (same format both channels) |
| AC-11 | PublishSpec/Review tested | TC-CMD-13, TC-CMD-14, TC-CMD-20 |

### Test Result Tracking (Sprint 33+34)

| Test ID | Description | Tester | Date | Result | Notes |
|---------|-------------|--------|------|--------|-------|
| TEST-M01 | Bridge status | | | | |
| TEST-M02 | Identity linking | | | | |
| TEST-M03 | Session lifecycle | | | | |
| TEST-M04 | Risk escalation | | | | |
| TEST-M05 | Free-text relay | | | | |
| TEST-M06 | Project registration | | | | |
| TEST-M07 | Setup + uninstall CLI | | | | |
| TEST-M08 | Stop notification | | | | |
| TEST-M09 | Permission approval | | | | |
| TEST-M10 | SOUL-aware launch | | | | |
| TEST-M11 | Install agents CLI | | | | |
| TEST-M12 | Fallback Docker deploy | | | | |
| TEST-M13 | Fallback E2E Telegram | | | | |

---

**Next review**: After Sprint 34 manual testing (E2E-023, E2E-024, E2E-025)
**Owner**: [@tester]
**Approved by**: [@cto] (pending Sprint 34 manual test sign-off)
