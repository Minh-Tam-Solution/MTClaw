package file

import (
	"fmt"

	"github.com/Minh-Tam-Solution/MTClaw/internal/cron"
	"github.com/Minh-Tam-Solution/MTClaw/internal/memory"
	"github.com/Minh-Tam-Solution/MTClaw/internal/pairing"
	"github.com/Minh-Tam-Solution/MTClaw/internal/sessions"
	"github.com/Minh-Tam-Solution/MTClaw/internal/skills"
	"github.com/Minh-Tam-Solution/MTClaw/internal/store"
)

// NewFileStores creates all stores backed by filesystem/in-memory managers (standalone mode).
func NewFileStores(cfg store.StoreConfig) (*store.Stores, error) {
	sessMgr := sessions.NewManager(cfg.SessionsDir)

	memCfg := memory.DefaultManagerConfig(cfg.Workspace)
	memMgr, err := memory.NewManager(memCfg)
	if err != nil {
		return nil, fmt.Errorf("create memory manager: %w", err)
	}

	cronSvc := cron.NewService(cfg.CronStorePath, nil)
	pairingSvc := pairing.NewService(cfg.PairingStorePath)
	skillsLoader := skills.NewLoader(cfg.Workspace, cfg.GlobalSkillsDir, cfg.BuiltinSkillsDir)

	return &store.Stores{
		Sessions: NewFileSessionStore(sessMgr),
		Memory:   NewFileMemoryStore(memMgr),
		Cron:     NewFileCronStore(cronSvc),
		Pairing:  NewFilePairingStore(pairingSvc),
		Skills:   NewFileSkillStore(skillsLoader),
	}, nil
}
