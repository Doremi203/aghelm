package app

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/Doremi203/aghelm/internal/config"
	"github.com/Doremi203/aghelm/internal/execution"
	"github.com/Doremi203/aghelm/internal/git"
	"github.com/Doremi203/aghelm/internal/hooks"
	"github.com/Doremi203/aghelm/internal/notify"
	"github.com/Doremi203/aghelm/internal/sessioncmd"
	"github.com/Doremi203/aghelm/internal/status"
	"github.com/Doremi203/aghelm/internal/store"
	"github.com/Doremi203/aghelm/internal/tmux"
)

type Local struct {
	ProfileDir string
	Runtime    sessioncmd.Runtime
	Engine     *status.Engine
	Backend    *sessioncmd.Backend
	Lifecycle  *sessioncmd.Lifecycle
	Execution  *execution.Runner
}

func OpenLocal(profileDir string, driver *tmux.Driver) (*Local, error) {
	if driver == nil {
		return nil, errors.New("local application requires an explicit tmux driver")
	}
	cfg, err := config.Default()
	if err != nil {
		return nil, err
	}
	engine, err := status.NewEngine(cfg)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(profileDir, 0o755); err != nil {
		return nil, err
	}
	st, err := store.Open(filepath.Join(profileDir, "state.db"))
	if err != nil {
		return nil, err
	}
	if err := sessioncmd.ApplyBindingSettings(st, driver); err != nil {
		st.Close()
		return nil, err
	}
	optionalGitDriver, _ := git.New()
	runtime := sessioncmd.Runtime{
		Config: cfg, Store: st, Driver: driver, Hooks: hooks.NewManager(profileDir),
		Git: optionalGitDriver, Snapshot: st.SetSnapshot,
	}
	backend, err := sessioncmd.BorrowBackend(runtime)
	if err != nil {
		st.Close()
		return nil, err
	}
	lifecycle, err := backend.Lifecycle()
	if err != nil {
		st.Close()
		return nil, err
	}
	runner := execution.New(execution.Dependencies{
		Store: st, TMux: driver, Engine: engine, Hooks: runtime.Hooks, Git: optionalGitDriver,
		Inbox: sessioncmd.NewInboxOwner(st), Notify: notify.Notify, ShowCommand: notify.ShowCommand,
		TakeFocus: func() (string, bool) { return notify.TakeFocus(profileDir) },
	}, execution.OptionsFromConfig(cfg))
	return &Local{ProfileDir: profileDir, Runtime: runtime, Engine: engine, Backend: backend, Lifecycle: lifecycle, Execution: runner}, nil
}

func (l *Local) Close() error {
	return l.Runtime.Store.Close()
}
