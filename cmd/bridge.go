package cmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/Minh-Tam-Solution/MTClaw/internal/claudecode"
	"github.com/Minh-Tam-Solution/MTClaw/internal/config"
)

func bridgeCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "bridge",
		Short: "Claude Code terminal bridge management",
	}
	cmd.AddCommand(bridgeStatusCmd())
	cmd.AddCommand(bridgeSetupCmd())
	cmd.AddCommand(bridgeUninstallCmd())
	cmd.AddCommand(bridgeInstallAgentsCmd())
	cmd.AddCommand(bridgeInitProjectCmd())
	return cmd
}

func bridgeStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Doctor-lite: check bridge health (tmux, config, audit)",
		Run: func(cmd *cobra.Command, args []string) {
			runBridgeStatus()
		},
	}
}

func runBridgeStatus() {
	cfg, err := config.Load(resolveConfigPath())
	if err != nil {
		fmt.Printf("FAIL  config load: %v\n", err)
		os.Exit(1)
	}

	passed := 0
	failed := 0

	check := func(name string, fn func() error) {
		if err := fn(); err != nil {
			fmt.Printf("FAIL  %s: %v\n", name, err)
			failed++
		} else {
			fmt.Printf("OK    %s\n", name)
			passed++
		}
	}

	// 1. Bridge enabled
	check("bridge.enabled", func() error {
		if !cfg.Bridge.Enabled {
			return fmt.Errorf("bridge is disabled in config (set bridge.enabled=true)")
		}
		return nil
	})

	// 2. tmux binary present
	check("tmux binary", func() error {
		path, err := exec.LookPath("tmux")
		if err != nil {
			return fmt.Errorf("tmux not found in PATH")
		}
		fmt.Printf("       path: %s\n", path)
		return nil
	})

	// 3. Hook port config valid
	check("hook port", func() error {
		port := cfg.Bridge.HookPort
		if port == 0 {
			port = 18792
		}
		if port < 1024 || port > 65535 {
			return fmt.Errorf("hook port %d out of range [1024, 65535]", port)
		}
		fmt.Printf("       port: %d\n", port)
		return nil
	})

	// 4. Active tmux sessions
	check("tmux sessions", func() error {
		bridge, err := claudecode.NewTmuxBridge()
		if err != nil {
			fmt.Printf("       (tmux not available — ok for first launch)\n")
			return nil // not fatal
		}
		sessions, err := bridge.ListSessions(context.Background())
		if err != nil {
			fmt.Printf("       (no tmux server running — ok for first launch)\n")
			return nil
		}
		count := 0
		for _, s := range sessions {
			if strings.HasPrefix(s.Name, "cc-") {
				count++
			}
		}
		fmt.Printf("       bridge sessions: %d\n", count)
		return nil
	})

	// 5. Audit directory writable
	check("audit dir writable", func() error {
		dir := cfg.Bridge.AuditDir
		if dir == "" {
			home, _ := os.UserHomeDir()
			dir = filepath.Join(home, ".mtclaw", "bridge-audit")
		}
		if err := os.MkdirAll(dir, 0700); err != nil {
			return fmt.Errorf("cannot create audit dir: %v", err)
		}
		testFile := filepath.Join(dir, ".write-test")
		if err := os.WriteFile(testFile, []byte("test"), 0600); err != nil {
			return fmt.Errorf("audit dir not writable: %v", err)
		}
		os.Remove(testFile)
		fmt.Printf("       dir: %s\n", dir)
		return nil
	})

	// 6. Store path writable (standalone)
	check("standalone store dir", func() error {
		dir := cfg.Bridge.StandaloneDir
		if dir == "" {
			home, _ := os.UserHomeDir()
			dir = filepath.Join(home, ".mtclaw")
		}
		if err := os.MkdirAll(dir, 0700); err != nil {
			return fmt.Errorf("cannot create store dir: %v", err)
		}
		fmt.Printf("       dir: %s\n", dir)
		return nil
	})

	// 7. HealthMonitor last status (if gateway is running with bridge)
	check("health monitor", func() error {
		// HealthMonitor runs inside the gateway process. In CLI mode we can't
		// query it directly, so we verify the prerequisites are met.
		// When the gateway is running, healthMon.LastStatus() is available via
		// the /health endpoint (Sprint D). Here we just confirm the monitor
		// could be constructed.
		_, tmuxErr := claudecode.NewTmuxBridge()
		if tmuxErr != nil {
			fmt.Printf("       health monitor would skip tmux checks (tmux unavailable)\n")
		} else {
			fmt.Printf("       health monitor prerequisites met\n")
		}
		return nil
	})

	fmt.Printf("\nBridge status: %d passed, %d failed\n", passed, failed)
	if failed > 0 {
		os.Exit(1)
	}
}

func bridgeSetupCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "setup",
		Short: "Generate hook scripts and HMAC secret for Claude Code integration",
		Run: func(cmd *cobra.Command, args []string) {
			runBridgeSetup()
		},
	}
}

func runBridgeSetup() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("FAIL  cannot determine home directory: %v\n", err)
		os.Exit(1)
	}

	hooksDir := filepath.Join(home, ".claude", "hooks")
	if err := os.MkdirAll(hooksDir, 0700); err != nil {
		fmt.Printf("FAIL  cannot create hooks dir: %v\n", err)
		os.Exit(1)
	}

	// Generate HMAC secret
	secret, err := claudecode.GenerateHookSecret()
	if err != nil {
		fmt.Printf("FAIL  cannot generate hook secret: %v\n", err)
		os.Exit(1)
	}

	// Secret file path (used by hook scripts at runtime — CTO-96)
	secretFile := filepath.Join(home, ".mtclaw", "bridge-hook-secret")

	// Write stop hook script (CTO-96: read secret from file at runtime, not embedded)
	stopHook := filepath.Join(hooksDir, "stop.sh")
	stopScript := fmt.Sprintf(`#!/bin/bash
# Generated by mtclaw bridge setup — Claude Code stop hook
# Sends session stop event to MTClaw bridge

SECRET_FILE="%s"
if [ ! -f "$SECRET_FILE" ]; then
  echo "Hook secret not found: $SECRET_FILE" >&2
  exit 0  # don't block Claude Code on missing secret
fi
HOOK_SECRET=$(cat "$SECRET_FILE")

HOOK_URL="http://127.0.0.1:18792/hook"
SESSION_ID="${CC_SESSION_ID:-unknown}"
TIMESTAMP=$(date +%%s)
BODY='{"event":"stop","exit_code":'${CC_EXIT_CODE:-0}',"summary":"session ended"}'
SIGNATURE=$(echo -n "${TIMESTAMP}.${BODY}" | openssl dgst -sha256 -hmac "$HOOK_SECRET" -hex 2>/dev/null | awk '{print $NF}')

curl -s -X POST "$HOOK_URL" \
  -H "Content-Type: application/json" \
  -H "X-Hook-Signature: $SIGNATURE" \
  -H "X-Hook-Timestamp: $TIMESTAMP" \
  -H "X-Hook-Session: $SESSION_ID" \
  -d "$BODY" >/dev/null 2>&1 || true
`, secretFile)

	if err := os.WriteFile(stopHook, []byte(stopScript), 0700); err != nil {
		fmt.Printf("FAIL  cannot write stop hook: %v\n", err)
		os.Exit(1)
	}

	// Write permission hook script (CTO-96: read secret from file at runtime)
	permHook := filepath.Join(hooksDir, "permission-request.sh")
	permScript := fmt.Sprintf(`#!/bin/bash
# Generated by mtclaw bridge setup — Claude Code permission hook
# Sends permission request to MTClaw bridge, polls for decision

SECRET_FILE="%s"
if [ ! -f "$SECRET_FILE" ]; then
  echo "Hook secret not found: $SECRET_FILE" >&2
  exit 1  # fail-closed for permission hooks
fi
HOOK_SECRET=$(cat "$SECRET_FILE")

HOOK_URL="http://127.0.0.1:18792/hook"
POLL_URL="http://127.0.0.1:18792/hook/permission"
SESSION_ID="${CC_SESSION_ID:-unknown}"
TOOL="${CC_TOOL_NAME:-unknown}"
TOOL_INPUT="${CC_TOOL_INPUT:-{}}"
TIMESTAMP=$(date +%%s)
BODY="{\"event\":\"permission\",\"tool\":\"${TOOL}\",\"tool_input\":${TOOL_INPUT}}"
SIGNATURE=$(echo -n "${TIMESTAMP}.${BODY}" | openssl dgst -sha256 -hmac "$HOOK_SECRET" -hex 2>/dev/null | awk '{print $NF}')

# Submit permission request (202 Accepted)
RESPONSE=$(curl -s -X POST "$HOOK_URL" \
  -H "Content-Type: application/json" \
  -H "X-Hook-Signature: $SIGNATURE" \
  -H "X-Hook-Timestamp: $TIMESTAMP" \
  -H "X-Hook-Session: $SESSION_ID" \
  -d "$BODY")

PERM_ID=$(echo "$RESPONSE" | grep -o '"id":"[^"]*"' | head -1 | cut -d'"' -f4)
if [ -z "$PERM_ID" ]; then
  echo "Failed to create permission request" >&2
  exit 1
fi

# Poll for decision (max 3 minutes = 180 seconds)
for i in $(seq 1 90); do
  POLL=$(curl -s "${POLL_URL}/${PERM_ID}")
  DECISION=$(echo "$POLL" | grep -o '"decision":"[^"]*"' | head -1 | cut -d'"' -f4)
  case "$DECISION" in
    approved) exit 0 ;;
    denied)   exit 1 ;;
    expired)  exit 1 ;;
  esac
  sleep 2
done

# Timeout — exit with error (fail-closed)
exit 1
`, secretFile)

	if err := os.WriteFile(permHook, []byte(permScript), 0700); err != nil {
		fmt.Printf("FAIL  cannot write permission hook: %v\n", err)
		os.Exit(1)
	}

	// Write secret to file (0600)
	if err := os.MkdirAll(filepath.Dir(secretFile), 0700); err != nil {
		fmt.Printf("FAIL  cannot create mtclaw dir: %v\n", err)
		os.Exit(1)
	}
	if err := os.WriteFile(secretFile, []byte(secret), 0600); err != nil {
		fmt.Printf("FAIL  cannot write secret file: %v\n", err)
		os.Exit(1)
	}

	// Check for existing ccpoke hooks
	ccpokeDir := filepath.Join(home, ".claude", "hooks")
	if entries, err := os.ReadDir(ccpokeDir); err == nil {
		for _, e := range entries {
			if strings.Contains(e.Name(), "ccpoke") {
				fmt.Printf("WARN  existing ccpoke hook detected: %s\n", e.Name())
				fmt.Printf("       Consider migrating to mtclaw bridge hooks.\n")
			}
		}
	}

	fmt.Printf("OK    Bridge setup complete.\n")
	fmt.Printf("       hooks dir: %s\n", hooksDir)
	fmt.Printf("       stop hook: %s\n", stopHook)
	fmt.Printf("       permission hook: %s\n", permHook)
	fmt.Printf("       secret file: %s (0600)\n", secretFile)
	fmt.Printf("\n       Next: Add bridge.enabled=true to config.json\n")
}

func bridgeUninstallCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "uninstall",
		Short: "Remove bridge hook scripts",
		Run: func(cmd *cobra.Command, args []string) {
			runBridgeUninstall()
		},
	}
}

func bridgeInstallAgentsCmd() *cobra.Command {
	var soulsDir string
	var roles string
	var force bool

	cmd := &cobra.Command{
		Use:   "install-agents <project-path>",
		Short: "Generate .claude/agents/*.md from SOUL files (Sprint 18)",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			projectPath := args[0]

			// Validate project path
			cleaned := filepath.Clean(projectPath)
			if !filepath.IsAbs(cleaned) {
				fmt.Printf("FAIL  project path must be absolute: %s\n", projectPath)
				os.Exit(1)
			}

			if soulsDir == "" {
				soulsDir = "docs/08-collaborate/souls"
			}

			var roleFilter []string
			if roles != "" {
				roleFilter = strings.Split(roles, ",")
				for i := range roleFilter {
					roleFilter[i] = strings.TrimSpace(roleFilter[i])
				}
			}

			// Invalidate cache in case install-agents was run before in this process
			claudecode.InvalidateRolesCache()

			result, err := claudecode.InstallAgents(cleaned, soulsDir, roleFilter, force)
			if err != nil {
				fmt.Printf("FAIL  install-agents: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("OK    install-agents complete.\n")
			fmt.Printf("       installed: %d\n", result.Installed)
			fmt.Printf("       updated:   %d\n", result.Updated)
			fmt.Printf("       skipped:   %d\n", result.Skipped)
			fmt.Printf("       target:    %s/.claude/agents/\n", cleaned)
			if result.Skills != nil {
				fmt.Printf("       skills:    %d installed, %d updated, %d skipped\n",
					result.Skills.Installed, result.Skills.Updated, result.Skills.Skipped)
			}
		},
	}

	cmd.Flags().StringVar(&soulsDir, "souls-dir", "", "Path to SOUL files (default: docs/08-collaborate/souls)")
	cmd.Flags().StringVar(&roles, "roles", "", "Comma-separated roles to install (default: all)")
	cmd.Flags().BoolVar(&force, "force", false, "Overwrite user-created agent files")

	return cmd
}

func bridgeInitProjectCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "init-project <project-path>",
		Short: "Generate CLAUDE.md for a project (Sprint 20B)",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			projectPath := filepath.Clean(args[0])
			if !filepath.IsAbs(projectPath) {
				fmt.Printf("FAIL  project path must be absolute: %s\n", projectPath)
				os.Exit(1)
			}

			created, err := claudecode.InitProject(projectPath, force)
			if err != nil {
				fmt.Printf("FAIL  init-project: %v\n", err)
				os.Exit(1)
			}

			if created {
				fmt.Printf("OK    CLAUDE.md generated at %s/CLAUDE.md\n", projectPath)
			} else {
				fmt.Printf("OK    CLAUDE.md already exists (user-owned). Use --force to overwrite.\n")
			}
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Overwrite user-created CLAUDE.md")
	return cmd
}

func runBridgeUninstall() {
	home, err := os.UserHomeDir()
	if err != nil {
		fmt.Printf("FAIL  cannot determine home directory: %v\n", err)
		os.Exit(1)
	}

	files := []string{
		filepath.Join(home, ".claude", "hooks", "stop.sh"),
		filepath.Join(home, ".claude", "hooks", "permission-request.sh"),
		filepath.Join(home, ".mtclaw", "bridge-hook-secret"),
	}

	removed := 0
	for _, f := range files {
		if err := os.Remove(f); err == nil {
			fmt.Printf("OK    removed %s\n", f)
			removed++
		} else if !os.IsNotExist(err) {
			fmt.Printf("WARN  cannot remove %s: %v\n", f, err)
		}
	}

	if removed == 0 {
		fmt.Printf("OK    No bridge hooks found to remove.\n")
	} else {
		fmt.Printf("\nRemoved %d hook files.\n", removed)
	}
}
