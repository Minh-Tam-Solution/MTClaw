package cmd

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/spf13/cobra"

	"github.com/Minh-Tam-Solution/MTClaw/internal/config"
	"github.com/Minh-Tam-Solution/MTClaw/internal/cost"
	"github.com/Minh-Tam-Solution/MTClaw/internal/store/pg"
	"github.com/Minh-Tam-Solution/MTClaw/internal/upgrade"
	"github.com/Minh-Tam-Solution/MTClaw/pkg/protocol"
)

func doctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check system environment and configuration health",
		Run: func(cmd *cobra.Command, args []string) {
			runDoctor()
		},
	}
}

func runDoctor() {
	fmt.Println("mtclaw doctor")
	fmt.Printf("  Version:  %s (protocol %d)\n", Version, protocol.ProtocolVersion)
	fmt.Printf("  OS:       %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf("  Go:       %s\n", runtime.Version())
	fmt.Println()

	// Config
	cfgPath := resolveConfigPath()
	fmt.Printf("  Config:   %s", cfgPath)
	if _, err := os.Stat(cfgPath); err != nil {
		fmt.Println(" (NOT FOUND)")
	} else {
		fmt.Println(" (OK)")
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		fmt.Printf("  Config load error: %s\n", err)
		return
	}

	// Database (managed mode only) — open early so we can show DB providers.
	var db *sql.DB
	isManaged := cfg.Database.Mode == "managed" && cfg.Database.PostgresDSN != ""
	if isManaged {
		fmt.Println()
		fmt.Println("  Database:")
		fmt.Printf("    %-12s managed\n", "Mode:")
		var dbErr error
		db, dbErr = sql.Open("pgx", cfg.Database.PostgresDSN)
		if dbErr != nil {
			fmt.Printf("    %-12s CONNECT FAILED (%s)\n", "Status:", dbErr)
			db = nil
		} else if pingErr := db.Ping(); pingErr != nil {
			fmt.Printf("    %-12s CONNECT FAILED (%s)\n", "Status:", pingErr)
			db.Close()
			db = nil
		} else {
			defer db.Close()
			s, schemaErr := upgrade.CheckSchema(db)
			if schemaErr != nil {
				fmt.Printf("    %-12s CHECK FAILED (%s)\n", "Schema:", schemaErr)
			} else if s.Dirty {
				fmt.Printf("    %-12s v%d (DIRTY — run: mtclaw migrate force %d)\n", "Schema:", s.CurrentVersion, s.CurrentVersion-1)
			} else if s.Compatible {
				fmt.Printf("    %-12s v%d (up to date)\n", "Schema:", s.CurrentVersion)
			} else if s.CurrentVersion > s.RequiredVersion {
				fmt.Printf("    %-12s v%d (binary too old, requires v%d)\n", "Schema:", s.CurrentVersion, s.RequiredVersion)
			} else {
				fmt.Printf("    %-12s v%d (upgrade needed — run: mtclaw upgrade)\n", "Schema:", s.CurrentVersion)
			}

			pending, hookErr := upgrade.PendingHooks(context.Background(), db)
			if hookErr == nil && len(pending) > 0 {
				fmt.Printf("    %-12s %d pending\n", "Data hooks:", len(pending))
			} else if hookErr == nil {
				fmt.Printf("    %-12s all applied\n", "Data hooks:")
			}
		}
	}

	// Providers — show DB providers in managed mode, config providers otherwise.
	fmt.Println()
	fmt.Println("  Providers:")
	if isManaged && db != nil {
		checkDBProviders(db)
		// Also show config-only providers (env vars) not in DB.
		checkProvider("Anthropic (env)", cfg.Providers.Anthropic.APIKey)
		checkProvider("OpenAI (env)", cfg.Providers.OpenAI.APIKey)
		checkProvider("OpenRouter (env)", cfg.Providers.OpenRouter.APIKey)
	} else {
		checkProvider("Anthropic", cfg.Providers.Anthropic.APIKey)
		checkProvider("OpenAI", cfg.Providers.OpenAI.APIKey)
		checkProvider("OpenRouter", cfg.Providers.OpenRouter.APIKey)
		checkProvider("Gemini", cfg.Providers.Gemini.APIKey)
		checkProvider("Groq", cfg.Providers.Groq.APIKey)
		checkProvider("DeepSeek", cfg.Providers.DeepSeek.APIKey)
		checkProvider("Mistral", cfg.Providers.Mistral.APIKey)
		checkProvider("XAI", cfg.Providers.XAI.APIKey)
	}

	// Channels — show DB channels in managed mode, config channels otherwise.
	fmt.Println()
	fmt.Println("  Channels:")
	if isManaged && db != nil {
		checkDBChannels(db)
	} else {
		checkChannel("Telegram", cfg.Channels.Telegram.Enabled, cfg.Channels.Telegram.Token != "")
		checkChannel("Zalo", cfg.Channels.Zalo.Enabled, cfg.Channels.Zalo.Token != "")
	}

	// Claude CLI (fallback provider)
	if cfg.Providers.ClaudeCLI.Enabled {
		fmt.Println()
		fmt.Println("  Claude CLI (fallback):")
		cliPath := cfg.Providers.ClaudeCLI.Path
		if cliPath == "" {
			cliPath = "claude"
		}
		if path, err := exec.LookPath(cliPath); err != nil {
			fmt.Printf("    %-12s NOT FOUND (%s)\n", "Binary:", cliPath)
		} else {
			fmt.Printf("    %-12s %s\n", "Binary:", path)
			// Check version
			out, err := exec.Command(path, "--version").Output()
			if err != nil {
				fmt.Printf("    %-12s UNKNOWN (--version failed)\n", "Version:")
			} else {
				fmt.Printf("    %-12s %s\n", "Version:", strings.TrimSpace(string(out)))
			}
		}
		model := cfg.Providers.ClaudeCLI.Model
		if model == "" {
			model = "sonnet"
		}
		fmt.Printf("    %-12s %s\n", "Model:", model)
		timeout := cfg.Providers.ClaudeCLI.Timeout
		if timeout <= 0 {
			timeout = 120
		}
		fmt.Printf("    %-12s %ds\n", "Timeout:", timeout)
		// Check OAuth token persistence (claude login state)
		home := os.Getenv("HOME")
		if home == "" {
			home = "/app"
		}
		oauthDir := home + "/.claude"
		if info, err := os.Stat(oauthDir); err != nil {
			fmt.Printf("    %-12s NOT FOUND (%s) — run: claude login\n", "OAuth:", oauthDir)
		} else if !info.IsDir() {
			fmt.Printf("    %-12s NOT A DIRECTORY (%s)\n", "OAuth:", oauthDir)
		} else {
			fmt.Printf("    %-12s %s (OK)\n", "OAuth:", oauthDir)
		}
	}

	// Provider chain
	if len(cfg.ProviderChain.Chain) > 0 {
		fmt.Println()
		fmt.Printf("  Provider Chain: %s\n", strings.Join(cfg.ProviderChain.Chain, " → "))
	}

	// External tools
	fmt.Println()
	fmt.Println("  External Tools:")
	checkBinary("docker")
	checkBinary("curl")
	checkBinary("git")

	// Adoption Metrics (managed mode only — Sprint 27)
	if isManaged && db != nil {
		checkAdoptionMetrics(db)
	}

	// Cost Status (managed mode only — Sprint 27)
	if isManaged && db != nil {
		checkCostStatus(db)
	}

	// Workspace
	fmt.Println()
	ws := config.ExpandHome(cfg.Agents.Defaults.Workspace)
	fmt.Printf("  Workspace: %s", ws)
	if _, err := os.Stat(ws); err != nil {
		fmt.Println(" (NOT FOUND)")
	} else {
		fmt.Println(" (OK)")
	}

	fmt.Println()
	fmt.Println("Doctor check complete.")
}

func checkProvider(name, apiKey string) {
	if apiKey != "" {
		maskedKey := apiKey[:4] + strings.Repeat("*", len(apiKey)-8) + apiKey[len(apiKey)-4:]
		fmt.Printf("    %-12s %s\n", name+":", maskedKey)
	} else {
		fmt.Printf("    %-12s (not configured)\n", name+":")
	}
}

func checkChannel(name string, enabled, hasCredentials bool) {
	status := "disabled"
	if enabled && hasCredentials {
		status = "enabled"
	} else if enabled {
		status = "enabled (missing credentials)"
	}
	fmt.Printf("    %-12s %s\n", name+":", status)
}

func checkDBChannels(db *sql.DB) {
	rows, err := db.QueryContext(context.Background(),
		"SELECT name, channel_type, enabled FROM channel_instances ORDER BY channel_type, name")
	if err != nil {
		fmt.Printf("    (could not query channels: %s)\n", err)
		return
	}
	defer rows.Close()

	found := false
	for rows.Next() {
		var name, channelType string
		var enabled bool
		if err := rows.Scan(&name, &channelType, &enabled); err != nil {
			continue
		}
		found = true
		status := "enabled"
		if !enabled {
			status = "disabled"
		}
		label := fmt.Sprintf("%s/%s", channelType, name)
		fmt.Printf("    %-24s %s\n", label+":", status)
	}
	if !found {
		fmt.Println("    (none configured in database)")
	}
}

func checkDBProviders(db *sql.DB) {
	rows, err := db.QueryContext(context.Background(),
		"SELECT name, COALESCE(display_name, name), enabled, (api_key IS NOT NULL AND api_key != '') AS has_key FROM llm_providers ORDER BY name")
	if err != nil {
		fmt.Printf("    (could not query providers: %s)\n", err)
		return
	}
	defer rows.Close()

	found := false
	for rows.Next() {
		var name, displayName string
		var enabled, hasKey bool
		if err := rows.Scan(&name, &displayName, &enabled, &hasKey); err != nil {
			continue
		}
		found = true
		status := "enabled"
		if !enabled {
			status = "disabled"
		}
		if !hasKey {
			status += " (no API key)"
		}
		fmt.Printf("    %-16s %s\n", displayName+":", status)
	}
	if !found {
		fmt.Println("    (none configured in database)")
	}
}

func checkBinary(name string) {
	path, err := exec.LookPath(name)
	if err != nil {
		fmt.Printf("    %-12s NOT FOUND\n", name+":")
	} else {
		fmt.Printf("    %-12s %s\n", name+":", path)
	}
}

func checkAdoptionMetrics(db *sql.DB) {
	ctx := context.Background()
	tracingStore := pg.NewPGTracingStore(db)
	since := time.Now().AddDate(0, 0, -7)

	fmt.Println()
	fmt.Println("  Adoption Metrics (last 7 days):")

	// WAU
	wau, err := tracingStore.CountDistinctUsers(ctx, since)
	if err != nil {
		fmt.Printf("    %-12s ERROR (%s)\n", "WAU:", err)
		return
	}
	fmt.Printf("    %-12s %d\n", "WAU:", wau)

	// By SOUL/agent
	byAgent, err := tracingStore.CountByAgent(ctx, since)
	if err == nil && len(byAgent) > 0 {
		parts := sortedMapEntries(byAgent)
		fmt.Printf("    %-12s %s\n", "By SOUL:", strings.Join(parts, ", "))
	} else if err == nil {
		fmt.Printf("    %-12s (no data)\n", "By SOUL:")
	}

	// By channel
	byChannel, err := tracingStore.CountByChannel(ctx, since)
	if err == nil && len(byChannel) > 0 {
		parts := sortedMapEntries(byChannel)
		fmt.Printf("    %-12s %s\n", "By Channel:", strings.Join(parts, ", "))
	} else if err == nil {
		fmt.Printf("    %-12s (no data)\n", "By Channel:")
	}

	// Tokens by provider
	byProvider, err := tracingStore.SumTokensByProvider(ctx, since)
	if err == nil && len(byProvider) > 0 {
		var parts []string
		for provider, usage := range byProvider {
			parts = append(parts, fmt.Sprintf("%s: %s in / %s out",
				provider, formatTokenCount(usage.InputTokens), formatTokenCount(usage.OutputTokens)))
		}
		sort.Strings(parts)
		fmt.Printf("    %-12s %s\n", "Tokens:", strings.Join(parts, ", "))
	} else if err == nil {
		fmt.Printf("    %-12s (no data)\n", "Tokens:")
	}
}

func checkCostStatus(db *sql.DB) {
	ctx := context.Background()
	tracingStore := pg.NewPGTracingStore(db)

	fmt.Println()
	fmt.Println("  Cost Status:")

	// Daily request count
	_, dailyCount, dailyLimit, err := cost.CheckDailyLimit(ctx, tracingStore)
	if err != nil {
		fmt.Printf("    %-12s ERROR (%s)\n", "Daily:", err)
	} else {
		pct := 0
		if dailyLimit > 0 {
			pct = dailyCount * 100 / dailyLimit
		}
		fmt.Printf("    %-12s %d / %d requests (%d%%)\n", "Daily:", dailyCount, dailyLimit, pct)
	}

	// Monthly token count
	_, monthlyTokens, monthlyLimit, mErr := cost.CheckMonthlyTokenLimit(ctx, tracingStore)
	if mErr != nil {
		fmt.Printf("    %-12s ERROR (%s)\n", "Monthly:", mErr)
	} else {
		pct := 0
		if monthlyLimit > 0 {
			pct = monthlyTokens * 100 / monthlyLimit
		}
		fmt.Printf("    %-12s %s / %s tokens (%d%%)\n", "Monthly:",
			formatTokenCount(monthlyTokens), formatTokenCount(monthlyLimit), pct)
	}

	// Overall status
	status := "OK"
	if err == nil && dailyLimit > 0 && dailyCount*100/dailyLimit >= 80 {
		status = "WARNING (daily usage ≥80%)"
	}
	if mErr == nil && monthlyLimit > 0 && monthlyTokens*100/monthlyLimit >= 80 {
		status = "WARNING (monthly tokens ≥80%)"
	}
	if err == nil && dailyCount >= dailyLimit {
		status = "EXCEEDED (daily limit reached)"
	}
	if mErr == nil && monthlyTokens >= monthlyLimit {
		status = "EXCEEDED (monthly token limit reached)"
	}
	fmt.Printf("    %-12s %s\n", "Status:", status)
}

// sortedMapEntries formats a map[string]int as "key: count" entries sorted by count descending.
func sortedMapEntries(m map[string]int) []string {
	type kv struct {
		key   string
		count int
	}
	var entries []kv
	for k, v := range m {
		entries = append(entries, kv{k, v})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].count > entries[j].count })

	parts := make([]string, len(entries))
	for i, e := range entries {
		parts[i] = fmt.Sprintf("%s: %d", e.key, e.count)
	}
	return parts
}

// formatTokenCount formats a token count with K/M suffixes for readability.
func formatTokenCount(n int) string {
	if n >= 1_000_000 {
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	}
	if n >= 1_000 {
		return fmt.Sprintf("%.1fK", float64(n)/1_000)
	}
	return fmt.Sprintf("%d", n)
}
