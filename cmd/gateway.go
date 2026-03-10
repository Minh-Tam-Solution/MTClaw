package cmd

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/Minh-Tam-Solution/MTClaw/internal/agent"
	"github.com/Minh-Tam-Solution/MTClaw/internal/bootstrap"
	"github.com/Minh-Tam-Solution/MTClaw/internal/bus"
	"github.com/Minh-Tam-Solution/MTClaw/internal/channels"
	"github.com/Minh-Tam-Solution/MTClaw/internal/claudecode"
	"github.com/Minh-Tam-Solution/MTClaw/extensions/msteams"
	"github.com/Minh-Tam-Solution/MTClaw/internal/channels/telegram"
	"github.com/Minh-Tam-Solution/MTClaw/internal/channels/discord"
	"github.com/Minh-Tam-Solution/MTClaw/internal/channels/zalo"
	"github.com/Minh-Tam-Solution/MTClaw/internal/config"
	"github.com/Minh-Tam-Solution/MTClaw/internal/cron"
	"github.com/Minh-Tam-Solution/MTClaw/internal/evidence"
	"github.com/Minh-Tam-Solution/MTClaw/internal/gateway"
	"github.com/Minh-Tam-Solution/MTClaw/internal/gateway/methods"
	mcpbridge "github.com/Minh-Tam-Solution/MTClaw/internal/mcp"
	"github.com/Minh-Tam-Solution/MTClaw/internal/pairing"
	"github.com/Minh-Tam-Solution/MTClaw/internal/permissions"
	"github.com/Minh-Tam-Solution/MTClaw/internal/providers"
	"github.com/Minh-Tam-Solution/MTClaw/internal/sandbox"
	"github.com/Minh-Tam-Solution/MTClaw/internal/scheduler"
	"github.com/Minh-Tam-Solution/MTClaw/internal/sessions"
	"github.com/Minh-Tam-Solution/MTClaw/internal/skills"
	"github.com/Minh-Tam-Solution/MTClaw/internal/rag"
	"github.com/Minh-Tam-Solution/MTClaw/internal/store"
	"github.com/Minh-Tam-Solution/MTClaw/internal/store/file"
	"github.com/Minh-Tam-Solution/MTClaw/internal/store/pg"
	"github.com/Minh-Tam-Solution/MTClaw/internal/tools"
	"github.com/Minh-Tam-Solution/MTClaw/internal/tracing"
	httpapi "github.com/Minh-Tam-Solution/MTClaw/internal/http"
	"github.com/Minh-Tam-Solution/MTClaw/pkg/browser"
	"github.com/Minh-Tam-Solution/MTClaw/pkg/protocol"
)

func runGateway() {
	// Setup structured logging
	logLevel := slog.LevelInfo
	if verbose {
		logLevel = slog.LevelDebug
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	})))

	// Load config
	cfgPath := resolveConfigPath()

	cfg, err := config.Load(cfgPath)
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Auto-detect: if no provider API key is configured, help the user.
	// Also trigger auto-onboard when config file doesn't exist (first run),
	// even if env vars provide API keys — managed mode needs DB seeding.
	_, cfgStatErr := os.Stat(cfgPath)
	configMissing := os.IsNotExist(cfgStatErr)
	if !cfg.HasAnyProvider() || configMissing {
		// Docker / CI: env vars provide API keys → non-interactive auto-onboard.
		if canAutoOnboard() {
			if runAutoOnboard(cfgPath) {
				cfg, _ = config.Load(cfgPath)
			} else {
				os.Exit(1)
			}
		} else if _, statErr := os.Stat(cfgPath); statErr == nil {
			// Config file exists — user already onboarded but forgot to source .env.local.
			envPath := filepath.Join(filepath.Dir(cfgPath), ".env.local")
			fmt.Println("No AI provider API key found. Did you forget to load your secrets?")
			fmt.Println()
			fmt.Printf("  source %s && ./mtclaw\n", envPath)
			fmt.Println()
			fmt.Println("Or re-run the setup wizard:  ./mtclaw onboard")
			os.Exit(1)
		} else {
			// No config file at all → first time, redirect to onboard wizard.
			fmt.Println("No configuration found. Starting setup wizard...")
			fmt.Println()
			runOnboard()
			return
		}
	}

	// Create core components
	msgBus := bus.New()

	// Create provider registry
	providerRegistry := providers.NewRegistry()
	registerProviders(providerRegistry, cfg)

	// Resolve workspace (must be absolute for system prompt + file tool path resolution)
	workspace := config.ExpandHome(cfg.Agents.Defaults.Workspace)
	if !filepath.IsAbs(workspace) {
		workspace, _ = filepath.Abs(workspace)
	}
	os.MkdirAll(workspace, 0755)

	// Seed bootstrap templates to disk (standalone mode only).
	// In managed mode, bootstrap files live in Postgres — not on disk.
	if cfg.Database.Mode != "managed" {
		seededFiles, seedErr := bootstrap.EnsureWorkspaceFiles(workspace)
		if seedErr != nil {
			slog.Warn("bootstrap template seeding failed", "error", seedErr)
		} else if len(seededFiles) > 0 {
			slog.Info("seeded workspace templates", "files", seededFiles)
		}
	}

	// Create tool registry with all tools
	toolsReg := tools.NewRegistry()
	agentCfg := cfg.ResolveAgent("default")

	// Sandbox manager (optional — routes tools through Docker containers)
	var sandboxMgr sandbox.Manager
	if sbCfg := cfg.Agents.Defaults.Sandbox; sbCfg != nil && sbCfg.Mode != "" && sbCfg.Mode != "off" {
		if err := sandbox.CheckDockerAvailable(context.Background()); err != nil {
			slog.Warn("sandbox disabled: Docker not available",
				"configured_mode", sbCfg.Mode,
				"error", err,
			)
		} else {
			resolved := sbCfg.ToSandboxConfig()
			sandboxMgr = sandbox.NewDockerManager(resolved)
			slog.Info("sandbox enabled", "mode", string(resolved.Mode), "image", resolved.Image, "scope", string(resolved.Scope))
		}
	}

	// Register file tools + exec tool (with sandbox routing via FsBridge if enabled)
	if sandboxMgr != nil {
		toolsReg.Register(tools.NewSandboxedReadFileTool(workspace, agentCfg.RestrictToWorkspace, sandboxMgr))
		toolsReg.Register(tools.NewSandboxedWriteFileTool(workspace, agentCfg.RestrictToWorkspace, sandboxMgr))
		toolsReg.Register(tools.NewSandboxedListFilesTool(workspace, agentCfg.RestrictToWorkspace, sandboxMgr))
		toolsReg.Register(tools.NewSandboxedEditTool(workspace, agentCfg.RestrictToWorkspace, sandboxMgr))
		toolsReg.Register(tools.NewSandboxedExecTool(workspace, agentCfg.RestrictToWorkspace, sandboxMgr))
	} else {
		toolsReg.Register(tools.NewReadFileTool(workspace, agentCfg.RestrictToWorkspace))
		toolsReg.Register(tools.NewWriteFileTool(workspace, agentCfg.RestrictToWorkspace))
		toolsReg.Register(tools.NewListFilesTool(workspace, agentCfg.RestrictToWorkspace))
		toolsReg.Register(tools.NewEditTool(workspace, agentCfg.RestrictToWorkspace))
		toolsReg.Register(tools.NewExecTool(workspace, agentCfg.RestrictToWorkspace))
	}

	// Memory system
	memMgr := setupMemory(workspace, cfg)
	if memMgr != nil {
		defer memMgr.Close()
		toolsReg.Register(tools.NewMemorySearchTool(memMgr))
		toolsReg.Register(tools.NewMemoryGetTool(memMgr))
		slog.Info("memory system enabled", "tools", []string{"memory_search", "memory_get"})
	}

	// Browser automation tool
	var browserMgr *browser.Manager
	if cfg.Tools.Browser.Enabled {
		browserMgr = browser.New(
			browser.WithHeadless(cfg.Tools.Browser.Headless),
		)
		toolsReg.Register(browser.NewBrowserTool(browserMgr))
		defer browserMgr.Close()
		slog.Info("browser tool enabled", "headless", cfg.Tools.Browser.Headless)
	}

	// Web tools (web_search + web_fetch)
	webSearchTool := tools.NewWebSearchTool(tools.WebSearchConfig{
		BraveEnabled: cfg.Tools.Web.Brave.Enabled,
		BraveAPIKey:  cfg.Tools.Web.Brave.APIKey,
		DDGEnabled:   cfg.Tools.Web.DuckDuckGo.Enabled,
	})
	if webSearchTool != nil {
		toolsReg.Register(webSearchTool)
		slog.Info("web_search tool enabled")
	}
	webFetchTool := tools.NewWebFetchTool(tools.WebFetchConfig{})
	toolsReg.Register(webFetchTool)
	slog.Info("web_fetch tool enabled")

	// Vision fallback tool (for non-vision providers like MiniMax)
	toolsReg.Register(tools.NewReadImageTool(providerRegistry))
	toolsReg.Register(tools.NewCreateImageTool(providerRegistry))

	// TTS (text-to-speech) system
	ttsMgr := setupTTS(cfg)
	if ttsMgr != nil {
		toolsReg.Register(tools.NewTtsTool(ttsMgr))
		slog.Info("tts enabled", "provider", ttsMgr.PrimaryProvider(), "auto", string(ttsMgr.AutoMode()))
	}

	// Tool rate limiting (per session, sliding window)
	if cfg.Tools.RateLimitPerHour > 0 {
		toolsReg.SetRateLimiter(tools.NewToolRateLimiter(cfg.Tools.RateLimitPerHour))
		slog.Info("tool rate limiting enabled", "per_hour", cfg.Tools.RateLimitPerHour)
	}

	// Credential scrubbing (enabled by default, can be disabled via config)
	if cfg.Tools.ScrubCredentials != nil && !*cfg.Tools.ScrubCredentials {
		toolsReg.SetScrubbing(false)
		slog.Info("credential scrubbing disabled")
	}

	// MCP servers (standalone mode: shared across all agents)
	var mcpMgr *mcpbridge.Manager
	if len(cfg.Tools.McpServers) > 0 {
		mcpMgr = mcpbridge.NewManager(toolsReg, mcpbridge.WithConfigs(cfg.Tools.McpServers))
		if err := mcpMgr.Start(context.Background()); err != nil {
			slog.Warn("mcp.startup_errors", "error", err)
		}
		defer mcpMgr.Stop()
		slog.Info("MCP servers initialized", "configured", len(cfg.Tools.McpServers), "tools", len(mcpMgr.ToolNames()))
	}

	// Subagent system
	subagentMgr := setupSubagents(providerRegistry, cfg, msgBus, toolsReg, workspace, sandboxMgr)
	if subagentMgr != nil {
		// Wire announce queue for batched subagent result delivery (matching TS debounce pattern)
		announceQueue := tools.NewAnnounceQueue(1000, 20,
			func(sessionKey string, items []tools.AnnounceQueueItem, meta tools.AnnounceMetadata) {
				remainingActive := subagentMgr.CountRunningForParent(meta.ParentAgent)
				content := tools.FormatBatchedAnnounce(items, remainingActive)
				senderID := fmt.Sprintf("subagent:batch-%d", len(items))
				label := items[0].Label
				if len(items) > 1 {
					label = fmt.Sprintf("%d tasks", len(items))
				}
				batchMeta := map[string]string{
					"origin_channel":      meta.OriginChannel,
					"origin_peer_kind":    meta.OriginPeerKind,
					"parent_agent":        meta.ParentAgent,
					"subagent_label":      label,
					"origin_trace_id":     meta.OriginTraceID,
					"origin_root_span_id": meta.OriginRootSpanID,
				}
				if meta.OriginLocalKey != "" {
					batchMeta["origin_local_key"] = meta.OriginLocalKey
				}
				msgBus.PublishInbound(bus.InboundMessage{
					Channel:  "system",
					SenderID: senderID,
					ChatID:   meta.OriginChatID,
					Content:  content,
					UserID:   meta.OriginUserID,
					Metadata: batchMeta,
				})
			},
			func(parentID string) int {
				return subagentMgr.CountRunningForParent(parentID)
			},
		)
		subagentMgr.SetAnnounceQueue(announceQueue)

		toolsReg.Register(tools.NewSpawnTool(subagentMgr, "default", 0))
		slog.Info("subagent system enabled", "tools", []string{"spawn"})
	}

	// Exec approval system — always active (deny patterns + safe bins + configurable ask mode)
	var execApprovalMgr *tools.ExecApprovalManager
	{
		approvalCfg := tools.DefaultExecApprovalConfig()
		// Override from user config (backward compat: explicit values take precedence)
		if eaCfg := cfg.Tools.ExecApproval; eaCfg.Security != "" {
			approvalCfg.Security = tools.ExecSecurity(eaCfg.Security)
		}
		if eaCfg := cfg.Tools.ExecApproval; eaCfg.Ask != "" {
			approvalCfg.Ask = tools.ExecAskMode(eaCfg.Ask)
		}
		if len(cfg.Tools.ExecApproval.Allowlist) > 0 {
			approvalCfg.Allowlist = cfg.Tools.ExecApproval.Allowlist
		}
		execApprovalMgr = tools.NewExecApprovalManager(approvalCfg)

		// Wire approval to exec tools in the registry
		if execTool, ok := toolsReg.Get("exec"); ok {
			if aa, ok := execTool.(tools.ApprovalAware); ok {
				aa.SetApprovalManager(execApprovalMgr, "default")
			}
		}
		slog.Info("exec approval enabled", "security", string(approvalCfg.Security), "ask", string(approvalCfg.Ask))
	}

	// --- Enforcement: Policy engines ---

	// Permission policy engine (role-based RPC access control)
	permPE := permissions.NewPolicyEngine(cfg.Gateway.OwnerIDs)

	// Tool policy engine (7-step tool filtering pipeline)
	toolPE := tools.NewPolicyEngine(&cfg.Tools)

	// Data directory for Phase 2 services
	dataDir := os.Getenv("MTCLAW_DATA_DIR")
	if dataDir == "" {
		dataDir = config.ExpandHome("~/.mtclaw/data")
	}
	os.MkdirAll(dataDir, 0755)

	// Block exec from accessing sensitive directories (data dir, .mtclaw, config file).
	// Prevents `cp /app/data/config.json workspace/` and similar exfiltration.
	if execTool, ok := toolsReg.Get("exec"); ok {
		if et, ok := execTool.(*tools.ExecTool); ok {
			et.DenyPaths(dataDir, ".mtclaw/")
			if cfgPath := os.Getenv("MTCLAW_CONFIG"); cfgPath != "" {
				et.DenyPaths(cfgPath)
			}
		}
	}

	// --- Mode-based store creation ---
	// Standalone: file-based adapters wrapping sessions/cron/pairing packages.
	// Managed: Postgres stores from pg.NewPGStores.
	var sessStore store.SessionStore
	var cronStore store.CronStore
	var pairingStore store.PairingStore
	var managedStores *store.Stores
	var traceCollector *tracing.Collector

	if cfg.Database.Mode == "managed" && cfg.Database.PostgresDSN != "" {
		// Schema compatibility check: ensure DB schema matches this binary.
		if err := checkSchemaOrAutoUpgrade(cfg.Database.PostgresDSN); err != nil {
			slog.Error("schema compatibility check failed", "error", err)
			os.Exit(1)
		}

		storeCfg := store.StoreConfig{
			PostgresDSN:   cfg.Database.PostgresDSN,
			Mode:          cfg.Database.Mode,
			EncryptionKey: os.Getenv("MTCLAW_ENCRYPTION_KEY"),
		}
		pgStores, pgErr := pg.NewPGStores(storeCfg)
		if pgErr != nil {
			slog.Error("failed to create PG stores", "error", pgErr)
			os.Exit(1)
		}
		managedStores = pgStores
		sessStore = pgStores.Sessions
		cronStore = pgStores.Cron
		pairingStore = pgStores.Pairing
		if pgStores.Tracing != nil {
			traceCollector = tracing.NewCollector(pgStores.Tracing)
			traceCollector.Start()
			slog.Info("LLM tracing enabled")
		}
	} else {
		// Standalone mode: file-based stores
		sessStore = file.NewFileSessionStore(sessions.NewManager(config.ExpandHome(cfg.Sessions.Storage)))
		cronStorePath := filepath.Join(dataDir, "cron", "jobs.json")
		cronStore = file.NewFileCronStore(cron.NewService(cronStorePath, nil))
		pairingStorePath := filepath.Join(dataDir, "pairing.json")
		pairingStore = file.NewFilePairingStore(pairing.NewService(pairingStorePath))
	}
	if traceCollector != nil {
		defer traceCollector.Stop()
		// OTel OTLP export: compiled via build tags. Build with 'go build -tags otel' to enable.
		initOTelExporter(context.Background(), cfg, traceCollector)
	}

	// Wire cron retry config from config.json
	cronRetryCfg := cfg.Cron.ToRetryConfig()
	if svc, ok := cronStore.(interface{ SetRetryConfig(cron.RetryConfig) }); ok {
		svc.SetRetryConfig(cronRetryCfg)
	}

	// Managed mode: load secrets from config_secrets table before env overrides.
	// Precedence: config.json → DB secrets → env vars (highest).
	if managedStores != nil && managedStores.ConfigSecrets != nil {
		if secrets, err := managedStores.ConfigSecrets.GetAll(context.Background()); err == nil && len(secrets) > 0 {
			cfg.ApplyDBSecrets(secrets)
			cfg.ApplyEnvOverrides()
			slog.Info("managed mode: config secrets loaded from DB", "count", len(secrets))
		}
	}

	// Managed mode: register providers from DB (overrides config providers).
	if managedStores != nil && managedStores.Providers != nil {
		registerProvidersFromDB(providerRegistry, managedStores.Providers)
	}

	// Managed mode: wire embedding provider to PGMemoryStore so IndexDocument generates vectors.
	if managedStores != nil && managedStores.Memory != nil {
		memCfg := cfg.Agents.Defaults.Memory
		if embProvider := resolveEmbeddingProvider(cfg, memCfg); embProvider != nil {
			managedStores.Memory.SetEmbeddingProvider(embProvider)
			slog.Info("managed mode: memory embeddings enabled", "provider", embProvider.Name(), "model", embProvider.Model())

			// Backfill embeddings for existing chunks that were stored without vectors.
			type backfiller interface {
				BackfillEmbeddings(ctx context.Context) (int, error)
			}
			if bf, ok := managedStores.Memory.(backfiller); ok {
				go func() {
					bgCtx := context.Background()
					count, err := bf.BackfillEmbeddings(bgCtx)
					if err != nil {
						slog.Warn("memory embeddings backfill failed", "error", err)
					} else if count > 0 {
						slog.Info("memory embeddings backfill complete", "chunks_updated", count)
					}
				}()
			}
		} else {
			slog.Warn("managed mode: memory embeddings disabled (no API key), chunks stored without vectors")
		}
	}

	// Load bootstrap files for default agent's system prompt.
	// Managed mode: load from DB first, seed if empty, fallback to filesystem.
	// Standalone mode: load from workspace filesystem.
	var contextFiles []bootstrap.ContextFile

	if managedStores != nil && managedStores.Agents != nil {
		bgCtx := context.Background()
		defaultAgent, agErr := managedStores.Agents.GetByKey(bgCtx, "default")
		if agErr == nil {
			dbFiles := bootstrap.LoadFromStore(bgCtx, managedStores.Agents, defaultAgent.ID)
			if len(dbFiles) > 0 {
				contextFiles = dbFiles
				slog.Info("bootstrap loaded from store", "count", len(dbFiles))
			} else {
				// DB empty → seed templates, then load
				if _, seedErr := bootstrap.SeedToStore(bgCtx, managedStores.Agents, defaultAgent.ID, defaultAgent.AgentType); seedErr != nil {
					slog.Warn("failed to seed bootstrap to store", "error", seedErr)
				} else {
					contextFiles = bootstrap.LoadFromStore(bgCtx, managedStores.Agents, defaultAgent.ID)
					slog.Info("bootstrap seeded and loaded from store", "count", len(contextFiles))
				}
			}
		}
	}

	if len(contextFiles) == 0 {
		// Standalone mode or DB fallback
		rawFiles := bootstrap.LoadWorkspaceFiles(workspace)
		truncCfg := bootstrap.TruncateConfig{
			MaxCharsPerFile: agentCfg.BootstrapMaxChars,
			TotalMaxChars:   agentCfg.BootstrapTotalMaxChars,
		}
		if truncCfg.MaxCharsPerFile <= 0 {
			truncCfg.MaxCharsPerFile = bootstrap.DefaultMaxCharsPerFile
		}
		if truncCfg.TotalMaxChars <= 0 {
			truncCfg.TotalMaxChars = bootstrap.DefaultTotalMaxChars
		}
		contextFiles = bootstrap.BuildContextFiles(rawFiles, truncCfg)
		slog.Info("bootstrap loaded from filesystem", "count", len(contextFiles))
	}

	// Debug: log bootstrap file loading results
	{
		var loadedNames []string
		for _, cf := range contextFiles {
			loadedNames = append(loadedNames, fmt.Sprintf("%s(%d)", cf.Path, len(cf.Content)))
		}
		slog.Info("bootstrap context files", "count", len(contextFiles), "files", loadedNames)
	}

	// Skills loader + search tool
	// Global skills live under ~/.mtclaw/skills/ (user-managed), not data/skills/.
	globalSkillsDir := os.Getenv("MTCLAW_SKILLS_DIR")
	if globalSkillsDir == "" {
		globalSkillsDir = filepath.Join(config.ExpandHome("~/.mtclaw"), "skills")
	}
	skillsLoader := skills.NewLoader(workspace, globalSkillsDir, "")
	skillSearchTool := tools.NewSkillSearchTool(skillsLoader)
	toolsReg.Register(skillSearchTool)
	slog.Info("skill_search tool registered", "skills", len(skillsLoader.ListSkills()))

	// Managed mode: wire embedding-based skill search
	if managedStores != nil && managedStores.Skills != nil {
		if pgSkills, ok := managedStores.Skills.(*pg.PGSkillStore); ok {
			memCfg := cfg.Agents.Defaults.Memory
			if embProvider := resolveEmbeddingProvider(cfg, memCfg); embProvider != nil {
				pgSkills.SetEmbeddingProvider(embProvider)
				skillSearchTool.SetEmbeddingSearcher(pgSkills, embProvider)
				slog.Info("managed mode: skill embeddings enabled", "provider", embProvider.Name())

				// Backfill embeddings for existing skills
				go func() {
					count, err := pgSkills.BackfillSkillEmbeddings(context.Background())
					if err != nil {
						slog.Warn("skill embeddings backfill failed", "error", err)
					} else if count > 0 {
						slog.Info("skill embeddings backfill complete", "skills_updated", count)
					}
				}()
			}
		}
	}

	// Cron tool (agent-facing, matching TS cron-tool.ts)
	toolsReg.Register(tools.NewCronTool(cronStore))
	slog.Info("cron tool registered")

	// Session tools (list, status, history, send)
	toolsReg.Register(tools.NewSessionsListTool())
	toolsReg.Register(tools.NewSessionStatusTool())
	toolsReg.Register(tools.NewSessionsHistoryTool())
	toolsReg.Register(tools.NewSessionsSendTool())

	// Message tool (send to channels)
	toolsReg.Register(tools.NewMessageTool())
	slog.Info("session + message tools registered")

	// Allow read_file to access skills directories (outside workspace).
	// Skills can live in ~/.mtclaw/skills/, ~/.agents/skills/, etc.
	homeDir, _ := os.UserHomeDir()
	if readTool, ok := toolsReg.Get("read_file"); ok {
		if pa, ok := readTool.(tools.PathAllowable); ok {
			pa.AllowPaths(globalSkillsDir)
			if homeDir != "" {
				pa.AllowPaths(filepath.Join(homeDir, ".agents", "skills"))
			}
		}
	}

	// Memory detection: SQLite (standalone) or PG (managed) — either enables memory.
	hasMemory := memMgr != nil
	if !hasMemory && managedStores != nil && managedStores.Memory != nil {
		hasMemory = true
		// PG memory is available but SQLite failed or wasn't created.
		// Ensure memory tools are registered so wireManagedExtras can wire PG store to them.
		if _, exists := toolsReg.Get("memory_search"); !exists {
			toolsReg.Register(tools.NewMemorySearchTool(nil))
			toolsReg.Register(tools.NewMemoryGetTool(nil))
			slog.Info("memory tools registered for managed mode (PG-backed)")
		}
	}

	// Wire SessionStoreAware + BusAware on tools that need them
	for _, name := range []string{"sessions_list", "session_status", "sessions_history", "sessions_send"} {
		if t, ok := toolsReg.Get(name); ok {
			if sa, ok := t.(tools.SessionStoreAware); ok {
				sa.SetSessionStore(sessStore)
			}
			if ba, ok := t.(tools.BusAware); ok {
				ba.SetMessageBus(msgBus)
			}
		}
	}
	// Wire BusAware on message tool
	if t, ok := toolsReg.Get("message"); ok {
		if ba, ok := t.(tools.BusAware); ok {
			ba.SetMessageBus(msgBus)
		}
	}

	// Standalone mode: wire FileAgentStore + interceptors + callbacks.
	// Must happen after tool registration (wires interceptors to read_file, write_file, edit).
	var fileAgentStore store.AgentStore
	var ensureUserFiles agent.EnsureUserFilesFunc
	var contextFileLoader agent.ContextFileLoaderFunc
	if cfg.Database.Mode != "managed" {
		var standaloneCleanup func()
		fileAgentStore, ensureUserFiles, contextFileLoader, standaloneCleanup =
			wireStandaloneExtras(cfg, toolsReg, dataDir, workspace)
		if standaloneCleanup != nil {
			defer standaloneCleanup()
		}
	}

	// Create all agents
	agentRouter := agent.NewRouter()

	isManaged := managedStores != nil

	// In managed mode, agents are created lazily by the resolver (from DB).
	// In standalone mode, create agents eagerly from config (no resolver to re-create on TTL expiry).
	if !isManaged {
		agentRouter.DisableTTL()

		// Standalone: inject DELEGATION.md listing all available SOULs.
		// In managed mode this is built from agent_links DB table; in standalone
		// we derive it from SOUL files on disk so the router agent knows its peers.
		const defaultSoulsDir = "docs/08-collaborate/souls"
		soulRoles := scanSOULRoles(defaultSoulsDir)
		if len(soulRoles) > 0 {
			// Build active agents map from config
			activeAgents := map[string]string{"default": "Assistant — router agent"}
			for agentID, spec := range cfg.Agents.List {
				if agentID == "default" {
					continue
				}
				name := spec.DisplayName
				if name == "" {
					name = agentID
				}
				activeAgents[agentID] = name
			}
			delegationMD := bootstrap.BuildSOULDelegationMD(soulRoles, activeAgents)
			if delegationMD != "" {
				contextFiles = append(contextFiles, bootstrap.ContextFile{
					Path:    bootstrap.DelegationFile,
					Content: delegationMD,
				})
				slog.Info("DELEGATION.md injected", "chars", len(delegationMD), "roles", len(soulRoles))
			} else {
				slog.Warn("DELEGATION.md is empty despite having SOUL roles", "roles", len(soulRoles))
			}
		}

		// Always create "default" agent
		if err := createAgentLoop("default", cfg, agentRouter, providerRegistry, msgBus, sessStore, toolsReg, toolPE, contextFiles, skillsLoader, hasMemory, sandboxMgr, fileAgentStore, ensureUserFiles, contextFileLoader); err != nil {
			slog.Error("failed to create default agent", "error", err)
			os.Exit(1)
		}

		// Create additional agents from agents.list.
		// Each agent gets its own context files loaded from its workspace (Sprint 26 — per-agent SOUL fix).
		for agentID := range cfg.Agents.List {
			if agentID == "default" {
				continue
			}
			agentSpecificCfg := cfg.ResolveAgent(agentID)
			agentWS := config.ExpandHome(agentSpecificCfg.Workspace)
			if !filepath.IsAbs(agentWS) {
				agentWS, _ = filepath.Abs(agentWS)
			}

			// Load per-agent context files if workspace differs from default
			agentContextFiles := contextFiles
			if agentWS != workspace {
				rawFiles := bootstrap.LoadWorkspaceFiles(agentWS)
				truncCfg := bootstrap.TruncateConfig{
					MaxCharsPerFile: agentSpecificCfg.BootstrapMaxChars,
					TotalMaxChars:   agentSpecificCfg.BootstrapTotalMaxChars,
				}
				if truncCfg.MaxCharsPerFile <= 0 {
					truncCfg.MaxCharsPerFile = bootstrap.DefaultMaxCharsPerFile
				}
				if truncCfg.TotalMaxChars <= 0 {
					truncCfg.TotalMaxChars = bootstrap.DefaultTotalMaxChars
				}
				loaded := bootstrap.BuildContextFiles(rawFiles, truncCfg)
				if len(loaded) > 0 {
					agentContextFiles = loaded
					slog.Info("per-agent context files loaded", "agent", agentID, "workspace", agentWS, "count", len(loaded))
				}
			}

			if err := createAgentLoop(agentID, cfg, agentRouter, providerRegistry, msgBus, sessStore, toolsReg, toolPE, agentContextFiles, skillsLoader, hasMemory, sandboxMgr, fileAgentStore, ensureUserFiles, contextFileLoader); err != nil {
				slog.Error("failed to create agent", "agent", agentID, "error", err)
			}
		}

		// Set standalone resolver: lazy-create agent Loops on-demand from SOUL files.
		// When @coder or @reviewer is mentioned but not in cfg.Agents.List, the resolver
		// validates against known SOULs and creates a Loop inheriting the default agent's
		// provider, workspace, and tool config (CTO-3). Cached forever (DisableTTL).
		knownSOULs := make(map[string]bool, len(soulRoles))
		for _, r := range soulRoles {
			knownSOULs[r.Role] = true
		}
		agentRouter.SetResolver(func(agentKey string) (agent.Agent, error) {
			if !knownSOULs[agentKey] {
				return nil, fmt.Errorf("unknown agent: %s (not a known SOUL)", agentKey)
			}
			// Build per-agent context files: replace SOUL.md with role-specific SOUL content.
			// On-demand agents inherit default's bootstrap files but get their own persona.
			agentCtxFiles := make([]bootstrap.ContextFile, 0, len(contextFiles)+1)
			for _, cf := range contextFiles {
				if cf.Path == bootstrap.SoulFile {
					continue // skip default SOUL.md — replaced below
				}
				agentCtxFiles = append(agentCtxFiles, cf)
			}
			// Load role-specific SOUL content from disk
			soul, err := claudecode.LoadSOUL(defaultSoulsDir, agentKey)
			if err == nil && soul.Body != "" {
				agentCtxFiles = append(agentCtxFiles, bootstrap.ContextFile{
					Path:    bootstrap.SoulFile,
					Content: soul.Body,
				})
				slog.Info("standalone resolver: loaded SOUL persona", "agent", agentKey, "chars", len(soul.Body))
			}
			loop, loopErr := buildAgentLoop(agentKey, cfg, providerRegistry, msgBus, sessStore, toolsReg, toolPE, agentCtxFiles, skillsLoader, hasMemory, sandboxMgr, fileAgentStore, ensureUserFiles, contextFileLoader)
			if loopErr != nil {
				return nil, fmt.Errorf("failed to create agent %s: %w", agentKey, loopErr)
			}
			slog.Info("standalone resolver: created agent on-demand", "agent", agentKey)
			return loop, nil
		})
	} else {
		slog.Info("managed mode: agents will be resolved lazily from database")
	}

	// Create gateway server and wire enforcement
	server := gateway.NewServer(cfg, msgBus, agentRouter, sessStore, toolsReg)
	server.SetPolicyEngine(permPE)
	server.SetPairingService(pairingStore)

	// contextFileInterceptor is created inside wireManagedExtras (managed mode only).
	// Declared here so it can be passed to registerAllMethods → AgentsMethods
	// for immediate cache invalidation on agents.files.set.
	var contextFileInterceptor *tools.ContextFileInterceptor

	// Managed mode: set agent store for tools_invoke context injection + wire extras
	if managedStores != nil && managedStores.Agents != nil {
		server.SetAgentStore(managedStores.Agents)
	}
	if managedStores != nil {
		// Dynamic custom tools: load global tools from DB before resolver
		var dynamicLoader *tools.DynamicToolLoader
		if managedStores.CustomTools != nil {
			dynamicLoader = tools.NewDynamicToolLoader(managedStores.CustomTools, workspace)
			if err := dynamicLoader.LoadGlobal(context.Background(), toolsReg); err != nil {
				slog.Warn("failed to load global custom tools", "error", err)
			}
		}

		contextFileInterceptor = wireManagedExtras(managedStores, agentRouter, providerRegistry, msgBus, sessStore, toolsReg, toolPE, skillsLoader, hasMemory, traceCollector, workspace, cfg.Gateway.InjectionAction, cfg, sandboxMgr, dynamicLoader)
		agentsH, skillsH, tracesH, mcpH, customToolsH, channelInstancesH, providersH, delegationsH, builtinToolsH := wireManagedHTTP(managedStores, cfg.Gateway.Token, msgBus, toolsReg, providerRegistry, permPE.IsOwner)
		if agentsH != nil {
			server.SetAgentsHandler(agentsH)
		}
		if skillsH != nil {
			server.SetSkillsHandler(skillsH)
		}
		if tracesH != nil {
			server.SetTracesHandler(tracesH)
		}
		if mcpH != nil {
			server.SetMCPHandler(mcpH)
		}
		if customToolsH != nil {
			server.SetCustomToolsHandler(customToolsH)
		}
		if channelInstancesH != nil {
			server.SetChannelInstancesHandler(channelInstancesH)
		}
		if providersH != nil {
			server.SetProvidersHandler(providersH)
		}
		if delegationsH != nil {
			server.SetDelegationsHandler(delegationsH)
		}
		if builtinToolsH != nil {
			server.SetBuiltinToolsHandler(builtinToolsH)
		}

		// Seed + apply builtin tool disables
		if managedStores.BuiltinTools != nil {
			seedBuiltinTools(context.Background(), managedStores.BuiltinTools)
			applyBuiltinToolDisables(context.Background(), managedStores.BuiltinTools, toolsReg)
		}
	}

	// Sprint 8: GitHub PR Gate webhook handler + API client (CTO-23: construct once at startup)
	var ghClient *tools.GitHubClient
	if cfg.GitHub.WebhookSecret != "" {
		webhookHandler := httpapi.NewWebhookGitHubHandler(cfg.GitHub.WebhookSecret, msgBus)
		server.SetWebhookGitHubHandler(webhookHandler)
		slog.Info("github webhook handler enabled")
	}
	if cfg.GitHub.AppToken != "" {
		ghClient = tools.NewGitHubClient(cfg.GitHub.AppToken)
		slog.Info("github API client enabled")
	}

	// Sprint 8: Evidence export API (uses specStore + prGateStore for audit export)
	// Sprint 11: PDF audit trail export added via SetEvidenceChain (wired below with chainBuilder)
	var evidenceHandler *httpapi.EvidenceExportHandler
	if managedStores != nil && (managedStores.Specs != nil || managedStores.PRGate != nil) {
		evidenceHandler = httpapi.NewEvidenceExportHandler(managedStores.Specs, managedStores.PRGate, cfg.Gateway.Token)
		server.SetEvidenceExportHandler(evidenceHandler)
		slog.Info("evidence export handler enabled")
	}

	// Register all RPC methods
	var agentStoreForRPC store.AgentStore
	if isManaged {
		agentStoreForRPC = managedStores.Agents
	}

	// SkillStore for RPC methods: PG in managed mode, file wrapper in standalone.
	var skillStore store.SkillStore
	if managedStores != nil && managedStores.Skills != nil {
		skillStore = managedStores.Skills
	} else {
		skillStore = file.NewFileSkillStore(skillsLoader)
	}

	var configSecretsStore store.ConfigSecretsStore
	if managedStores != nil {
		configSecretsStore = managedStores.ConfigSecrets
	}

	var teamStoreForRPC store.TeamStore
	if managedStores != nil {
		teamStoreForRPC = managedStores.Teams
	}

	pairingMethods := registerAllMethods(server, agentRouter, sessStore, cronStore, pairingStore, cfg, cfgPath, workspace, dataDir, msgBus, execApprovalMgr, agentStoreForRPC, isManaged, skillStore, configSecretsStore, teamStoreForRPC, contextFileInterceptor)

	// Channel manager
	channelMgr := channels.NewManager(msgBus)

	// Wire channel sender on message tool (now that channelMgr exists)
	if t, ok := toolsReg.Get("message"); ok {
		if cs, ok := t.(tools.ChannelSenderAware); ok {
			cs.SetChannelSender(channelMgr.SendToChannel)
		}
	}

	// Managed mode: load channel instances from DB first.
	var instanceLoader *channels.InstanceLoader
	if managedStores != nil && managedStores.ChannelInstances != nil {
		instanceLoader = channels.NewInstanceLoader(managedStores.ChannelInstances, managedStores.Agents, channelMgr, msgBus, pairingStore)
		instanceLoader.RegisterFactory("telegram", telegram.FactoryWithStores(managedStores.Agents, managedStores.Teams, managedStores.Specs))
		instanceLoader.RegisterFactory("zalo_oa", zalo.Factory)
		instanceLoader.RegisterFactory("msteams", msteams.Factory)
		instanceLoader.RegisterFactory("discord", discord.Factory)
		if err := instanceLoader.LoadAll(context.Background()); err != nil {
			slog.Error("failed to load channel instances from DB", "error", err)
		}
	}

	// Register config-based channels as fallback (standalone mode only).
	// In managed mode, channels are loaded from DB via instanceLoader — skip config-based registration.
	if cfg.Channels.Telegram.Enabled && cfg.Channels.Telegram.Token != "" && instanceLoader == nil {
		tg, err := telegram.New(cfg.Channels.Telegram, msgBus, pairingStore, nil, nil, nil)
		if err != nil {
			slog.Error("failed to initialize telegram channel", "error", err)
		} else {
			channelMgr.RegisterChannel("telegram", tg)
			slog.Info("telegram channel enabled (config)")
		}
	}

	if cfg.Channels.Zalo.Enabled && cfg.Channels.Zalo.Token != "" && instanceLoader == nil {
		z, err := zalo.New(cfg.Channels.Zalo, msgBus, pairingStore)
		if err != nil {
			slog.Error("failed to initialize zalo channel", "error", err)
		} else {
			channelMgr.RegisterChannel("zalo", z)
			slog.Info("zalo channel enabled (config)")
		}
	}

	if cfg.Channels.MSTeams.Enabled && cfg.Channels.MSTeams.AppID != "" && instanceLoader == nil {
		ms, err := msteams.New(cfg.Channels.MSTeams, msgBus)
		if err != nil {
			slog.Error("failed to initialize msteams channel", "error", err)
		} else {
			channelMgr.RegisterChannel("msteams", ms)
			server.AddMuxHandler(ms.RegisterRoutes)
			slog.Info("msteams channel enabled (config)", "webhook_path", cfg.Channels.MSTeams.WebhookPath)
		}
	}

	if cfg.Channels.Discord.Enabled && cfg.Channels.Discord.Token != "" && instanceLoader == nil {
		dc, err := discord.New(cfg.Channels.Discord, msgBus, pairingStore)
		if err != nil {
			slog.Error("failed to initialize discord channel", "error", err)
		} else {
			channelMgr.RegisterChannel("discord", dc)
			slog.Info("discord channel enabled (config)")
		}
	}

	// Wire Claude Code Bridge (ADR-010, CTO I2) — inject into all Telegram channels.
	if cfg.Bridge.Enabled {
		bridgeCfg := claudecode.DefaultBridgeConfig()
		bridgeCfg.Enabled = cfg.Bridge.Enabled
		bridgeCfg.HookPort = cfg.Bridge.HookPort
		bridgeCfg.HookBind = cfg.Bridge.HookBind
		if cfg.Bridge.AuditDir != "" {
			bridgeCfg.AuditDir = cfg.Bridge.AuditDir
		}
		if cfg.Bridge.StandaloneDir != "" {
			bridgeCfg.StandaloneDir = cfg.Bridge.StandaloneDir
		}
		// Apply admission overrides from config.json if present.
		if adm := cfg.Bridge.Admission; len(adm) > 0 {
			if v, ok := adm["max_sessions_per_agent"].(float64); ok && v > 0 {
				bridgeCfg.Admission.MaxSessionsPerAgent = int(v)
			}
			if v, ok := adm["max_total_sessions"].(float64); ok && v > 0 {
				bridgeCfg.Admission.MaxTotalSessions = int(v)
			}
			if v, ok := adm["max_cpu_percent"].(float64); ok && v > 0 {
				bridgeCfg.Admission.MaxCPUPercent = v
			}
			if v, ok := adm["max_memory_percent"].(float64); ok && v > 0 {
				bridgeCfg.Admission.MaxMemoryPercent = v
			}
			if v, ok := adm["per_tenant_session_cap"].(float64); ok && v > 0 {
				bridgeCfg.Admission.PerTenantSessionCap = int(v)
			}
		}
		tmuxBridge, err := claudecode.NewTmuxBridge()
		if err != nil {
			slog.Warn("claude code bridge: tmux not available, sessions will not launch terminal", "error", err)
		}
		bridgeMgr := claudecode.NewSessionManager(bridgeCfg, tmuxBridge)

		// Wire PG persistence for bridge sessions (Sprint 26 — dual-write pattern)
		if managedStores != nil && managedStores.BridgeSessions != nil {
			bridgeMgr.SetStore(managedStores.BridgeSessions)
			if n, err := bridgeMgr.LoadFromStore(context.Background()); err != nil {
				slog.Warn("bridge session recovery from PG failed", "error", err)
			} else if n > 0 {
				slog.Info("bridge sessions recovered from PG on startup", "count", n)
			}
		}

		// Wire audit dual-write (Sprint 26 — JSONL primary + PG secondary)
		{
			var auditDB *sql.DB
			if cfg.Database.Mode == "managed" && cfg.Database.PostgresDSN != "" {
				if db, err := pg.OpenDB(cfg.Database.PostgresDSN); err != nil {
					slog.Warn("audit PG connection failed, JSONL-only mode", "error", err)
				} else {
					auditDB = db
				}
			}
			auditWriter, err := claudecode.NewAuditWriter(bridgeCfg.AuditDir, auditDB)
			if err != nil {
				slog.Warn("audit writer init failed, audit disabled", "error", err)
			} else {
				bridgeMgr.SetAuditWriter(auditWriter)
				defer auditWriter.Close()
				slog.Info("bridge audit writer enabled", "dir", bridgeCfg.AuditDir, "pg", auditDB != nil)
			}
		}

		// Pre-register projects from config.json
		for _, proj := range cfg.Bridge.Projects {
			if _, err := bridgeMgr.Projects().Register("global", proj.Name, proj.Path, claudecode.AgentClaudeCode); err != nil {
				slog.Warn("bridge project registration failed", "name", proj.Name, "path", proj.Path, "error", err)
			} else {
				slog.Info("bridge project registered", "name", proj.Name, "path", proj.Path)
			}
		}

		// Inject into all registered Telegram channels
		for _, name := range channelMgr.GetEnabledChannels() {
			if ch, ok := channelMgr.GetChannel(name); ok {
				if tgCh, ok := ch.(*telegram.Channel); ok {
					tgCh.SetBridgeManager(bridgeMgr)
					slog.Info("claude code bridge wired into telegram channel", "channel", name)
				}
			}
		}

		// Start HookServer (Sprint 15/B) — localhost-only HTTP server for signed webhooks
		notifier := claudecode.NewNotifier(func(ctx context.Context, channel, chatID, message string) error {
			return channelMgr.SendToChannel(ctx, channel, chatID, message)
		})
		var hookOpts []claudecode.HookServerOption
		if bridgeCfg.HookBind != "" {
			hookOpts = append(hookOpts, claudecode.WithHookBind(bridgeCfg.HookBind))
		}
		hookServer := claudecode.NewHookServer(bridgeCfg.HookPort, bridgeMgr, notifier, hookOpts...)

		// Inject HookServer into Telegram channels for permission callbacks (Sprint 16/C)
		for _, name := range channelMgr.GetEnabledChannels() {
			if ch, ok := channelMgr.GetChannel(name); ok {
				if tgCh, ok := ch.(*telegram.Channel); ok {
					tgCh.SetHookServer(hookServer)
				}
			}
		}

		bridgeCtx, bridgeCancel := context.WithCancel(context.Background())
		go func() {
			if err := hookServer.Start(bridgeCtx); err != nil {
				slog.Error("hook server stopped", "error", err)
			}
		}()

		// Start HealthMonitor (Sprint 15/B) — 30s check interval
		healthMon := claudecode.NewHealthMonitor(bridgeMgr, tmuxBridge, 0)
		go healthMon.Start(bridgeCtx)

		// Session cleanup ticker: remove stopped sessions older than 24h every 10 minutes.
		go func() {
			ticker := time.NewTicker(10 * time.Minute)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					if n := bridgeMgr.CleanupStopped(24 * time.Hour); n > 0 {
						slog.Info("bridge session cleanup", "removed", n)
					}
				case <-bridgeCtx.Done():
					return
				}
			}
		}()

		defer bridgeCancel()
		_ = healthMon // accessed via bridge status CLI
	}

	// TODO: create_forum_topic tool — disabled for now, re-enable when needed.
	// toolsReg.Register(tools.NewCreateForumTopicTool(func() tools.ForumTopicCreator {
	// 	for _, name := range channelMgr.GetEnabledChannels() {
	// 		ch, ok := channelMgr.GetChannel(name)
	// 		if !ok { continue }
	// 		if fc, ok := ch.(tools.ForumTopicCreator); ok { return fc }
	// 	}
	// 	return nil
	// }))

	// Register channels RPC methods (after channelMgr is initialized with all channels)
	methods.NewChannelsMethods(channelMgr).Register(server.Router())

	// Register channel instances WS RPC methods (managed mode only)
	if managedStores != nil && managedStores.ChannelInstances != nil {
		methods.NewChannelInstancesMethods(managedStores.ChannelInstances, msgBus).Register(server.Router())
	}

	// Register agent links WS RPC methods (managed mode only)
	if managedStores != nil && managedStores.AgentLinks != nil && managedStores.Agents != nil {
		methods.NewAgentLinksMethods(managedStores.AgentLinks, managedStores.Agents, agentRouter, msgBus).Register(server.Router())
	}

	// Register agent teams WS RPC methods (managed mode only)
	if managedStores != nil && managedStores.Teams != nil {
		methods.NewTeamsMethods(managedStores.Teams, managedStores.Agents, managedStores.AgentLinks, agentRouter, msgBus).Register(server.Router())
	}

	// Register evidence chain WS RPC methods (Sprint 11, ADR-009)
	if managedStores != nil && managedStores.EvidenceLinks != nil && managedStores.Specs != nil {
		chainBuilder := evidence.NewChainBuilder(managedStores.EvidenceLinks, managedStores.Specs, managedStores.PRGate)
		methods.NewEvidenceMethods(chainBuilder, managedStores.EvidenceLinks, managedStores.Specs).Register(server.Router())

		// Sprint 11 T11-03: Wire PDF audit trail into evidence export handler.
		if evidenceHandler != nil {
			evidenceHandler.SetEvidenceChain(managedStores.EvidenceLinks, chainBuilder)
			slog.Info("audit trail PDF endpoint enabled")
		}
	}

	// Cache invalidation: reload channel instances on changes.
	// Runs in a goroutine because Reload() is heavy (stops channels, waits for polling exit,
	// sleeps 500ms, reloads from DB, starts new channels) and Broadcast handlers must be non-blocking.
	if instanceLoader != nil {
		msgBus.Subscribe(bus.TopicCacheChannelInstances, func(event bus.Event) {
			if event.Name != protocol.EventCacheInvalidate {
				return
			}
			payload, ok := event.Payload.(bus.CacheInvalidatePayload)
			if !ok || payload.Kind != bus.CacheKindChannelInstances {
				return
			}
			go instanceLoader.Reload(context.Background())
		})
	}

	// Wire pairing approval notification → channel (matching TS notifyPairingApproved).
	botName := cfg.ResolveDisplayName("default")
	pairingMethods.SetOnApprove(func(ctx context.Context, channel, chatID string) {
		msg := fmt.Sprintf("✅ %s access approved. Send a message to start chatting.", botName)
		if err := channelMgr.SendToChannel(ctx, channel, chatID, msg); err != nil {
			slog.Warn("failed to send pairing approval notification", "channel", channel, "chatID", chatID, "error", err)
		}
	})

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Skills directory watcher — auto-detect new/removed/modified skills at runtime.
	if skillsWatcher, err := skills.NewWatcher(skillsLoader); err != nil {
		slog.Warn("skills watcher unavailable", "error", err)
	} else {
		if err := skillsWatcher.Start(ctx); err != nil {
			slog.Warn("skills watcher start failed", "error", err)
		} else {
			defer skillsWatcher.Stop()
		}
	}

	// Start channels
	if err := channelMgr.StartAll(ctx); err != nil {
		slog.Error("failed to start channels", "error", err)
	}

	// Create lane-based scheduler (matching TS CommandLane pattern).
	// The RunFunc resolves the agent from the RunRequest metadata.
	// Must be created before cron setup so cron jobs route through the scheduler.
	sched := scheduler.NewScheduler(
		scheduler.DefaultLanes(),
		scheduler.DefaultQueueConfig(),
		makeSchedulerRunFunc(agentRouter, cfg),
	)
	defer sched.Stop()

	// Start cron service with job handler (routes through scheduler's cron lane)
	cronStore.SetOnJob(makeCronJobHandler(sched, msgBus, cfg))
	if err := cronStore.Start(); err != nil {
		slog.Warn("cron service failed to start", "error", err)
	}

	// Start heartbeat service (matching TS heartbeat-runner.ts).
	heartbeatSvc := setupHeartbeat(cfg, agentRouter, sessStore, msgBus, workspace, managedStores)
	if heartbeatSvc != nil {
		heartbeatSvc.Start()
	}

	// Adaptive throttle: reduce per-session concurrency when nearing the summary threshold.
	// This prevents concurrent runs from racing with summarization.
	// Uses calibrated token estimation (actual prompt tokens from last LLM call)
	// and the agent's real context window (cached on session by the Loop).
	sched.SetTokenEstimateFunc(func(sessionKey string) (int, int) {
		history := sessStore.GetHistory(sessionKey)
		lastPT, lastMC := sessStore.GetLastPromptTokens(sessionKey)
		tokens := agent.EstimateTokensWithCalibration(history, lastPT, lastMC)
		cw := sessStore.GetContextWindow(sessionKey)
		if cw <= 0 {
			cw = 200000 // fallback for sessions not yet processed
		}
		return tokens, cw
	})

	// Subscribe to agent events for channel streaming/reaction forwarding.
	// Events emitted by agent loops are broadcast to the bus; we forward them
	// to the channel manager which routes to StreamingChannel/ReactionChannel.
	msgBus.Subscribe(bus.TopicChannelStreaming, func(event bus.Event) {
		if event.Name != protocol.EventAgent {
			return
		}
		agentEvent, ok := event.Payload.(agent.AgentEvent)
		if !ok {
			return
		}
		channelMgr.HandleAgentEvent(agentEvent.Type, agentEvent.RunID, agentEvent.Payload)
	})

	// Start inbound message consumer (channel → scheduler → agent → channel)
	var consumerTeamStore store.TeamStore
	var consumerTracingStore store.TracingStore
	if managedStores != nil {
		consumerTeamStore = managedStores.Teams
		consumerTracingStore = managedStores.Tracing
	}
	// Sprint 6: RAG client for SOUL-Aware RAG routing (US-034).
	var ragClient *rag.Client
	if cfg.Providers.BflowAI.APIKey != "" {
		ragClient = rag.NewClient(cfg.Providers.BflowAI.APIBase, cfg.Providers.BflowAI.APIKey, "")
	}
	// Sprint 7: Wire specStore for governance spec processing (Rail #1).
	var consumerSpecStore store.SpecStore
	if managedStores != nil {
		consumerSpecStore = managedStores.Specs
	}
	// Sprint 8: Wire prGateStore for PR Gate evaluation persistence (Rail #2).
	var consumerPRGateStore store.PRGateStore
	if managedStores != nil {
		consumerPRGateStore = managedStores.PRGate
	}
	// Sprint 11 (ADR-009): Wire evidence linker for cross-rail auto-linking.
	var consumerEvidenceLinker *evidence.Linker
	if managedStores != nil && managedStores.EvidenceLinks != nil {
		consumerEvidenceLinker = evidence.NewLinker(managedStores.EvidenceLinks)
	}
	go consumeInboundMessages(ctx, msgBus, agentRouter, cfg, sched, channelMgr, consumerTeamStore, consumerTracingStore, ragClient, consumerSpecStore, ghClient, consumerPRGateStore, consumerEvidenceLinker)

	go func() {
		sig := <-sigCh
		slog.Info("graceful shutdown initiated", "signal", sig)

		// Broadcast shutdown event
		server.BroadcastEvent(*protocol.NewEvent(protocol.EventShutdown, nil))

		// Stop channels, cron, and heartbeat
		channelMgr.StopAll(context.Background())
		cronStore.Stop()
		if heartbeatSvc != nil {
			heartbeatSvc.Stop()
		}

		// Stop sandbox pruning + release containers
		if sandboxMgr != nil {
			sandboxMgr.Stop()
			slog.Info("releasing sandbox containers...")
			sandboxMgr.ReleaseAll(context.Background())
		}

		cancel()
	}()

	gatewayMode := "standalone"
	if cfg.Database.Mode == "managed" {
		gatewayMode = "managed"
	}
	slog.Info("mtclaw gateway starting",
		"version", Version,
		"protocol", protocol.ProtocolVersion,
		"mode", gatewayMode,
		"agents", agentRouter.List(),
		"tools", toolsReg.Count(),
		"channels", channelMgr.GetEnabledChannels(),
	)

	// Tailscale listener: build the mux first, then pass it to initTailscale
	// so the same routes are served on both the main listener and Tailscale.
	// Compiled via build tags: `go build -tags tsnet` to enable.
	mux := server.BuildMux()
	tsCleanup := initTailscale(ctx, cfg, mux)
	if tsCleanup != nil {
		defer tsCleanup()
	}

	// Phase 1: suggest localhost binding when Tailscale is active
	if cfg.Tailscale.Hostname != "" && cfg.Gateway.Host == "0.0.0.0" {
		slog.Info("Tailscale enabled. Consider setting MTCLAW_HOST=127.0.0.1 for localhost-only + Tailscale access")
	}

	if err := server.Start(ctx); err != nil {
		slog.Error("gateway error", "error", err)
		os.Exit(1)
	}
}
