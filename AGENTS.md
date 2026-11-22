# Agent notes

Project context for coding agents. Human setup and PR conventions live in
[.github/CONTRIBUTING.md](.github/CONTRIBUTING.md), and what a review looks
for in [REVIEW.md](REVIEW.md). aghelm is a fork with no CI and no releases:
`make check` is the gate, and users build from source.

## Product boundary

Keep aghelm a thin wrapper around the supported TUIs. Preserve upstream
defaults and user configuration; avoid hardcoded models, version-specific
behavior, and recreating upstream settings. Apply the
[Thin Wrapper Principle](PRODUCT.md#thin-wrapper-principle) to every integration,
including shared controls, capability discovery, and status detection.
Provider-specific features multiply ongoing maintenance. Supporting a TUI does
not mean mirroring its feature set; prefer workspace features shared across tools.

## What every change covers

A feature, a tool, or a fix is done when it holds on the whole matrix. The
matrix is every CLI in `builtinTools` (`internal/config/config.go`), macOS,
Linux and Windows on amd64 and arm64 plus WSL2, every terminal that reaches
the manager, local or over SSH, and both input methods. A change that
covers a subset names what it leaves out in the PR description and what
covering the rest would take. Disclosure alone does not approve partial tool
coverage. [REVIEW.md](REVIEW.md#tool-coverage) defines the allowed exceptions.

- **Keyboard and mouse.** Every action a key performs is reachable with the
  mouse, and every clickable surface has a key. A new row, panel, button,
  or handle ships with both, and the footer and the `?` key map name the key.
- **Dynamic over hardcoded.** Anything that varies by tool, version, or user
  is discovered at runtime or read from the tool itself: models, thinking
  modes, capabilities, prompts, screens. A maintained list of values that an
  upstream release can change is a defect. Where discovery has no stable,
  documented interface, the feature stays in the underlying TUI.
- **The current CLI release.** Each CLI is supported at its latest stable
  release. None publishes a long-term-support line. When an upstream release
  changes a flag, a subcommand, a file layout, or a screen, the adapter moves
  to the new behavior and drops the old one, with no branch for older
  releases. The same change sets the tool's `minimum_version` in
  `internal/config/builtin.go` to the first release it needs, so a spawn on
  an older CLI opens the update dialog instead of failing.
- **Configurable in the UI, or not configurable.** A setting a user can
  change lives in Settings (`s`), is stored by the manager, and takes effect
  from the picker, the way the keybindings and editor rows do. The value
  lives in `state.db`, where the manager, the CLI and the MCP server all
  read it, and `aghelm settings` reads and writes it from a shell.
  Every key is named in `ui.SettingSpecs` with the values it takes, so the
  shell refuses what the modal would never write. A value that is a
  command, a token or a URL stays off the modal and is set from the shell
  alone, so the modal does not grow a row per knob. A new per-user file,
  environment variable, or hand-edited block is the wrong shape; put the
  choice on a Settings row, behind `aghelm settings`, or pick a
  default in code.
- **Every tool.** New shared behavior covers every applicable tool in the
  same PR. A field, status rule, or command added to one `[tools.*]` block
  is added to every tool it applies to. A fix to one tool's adapter or support
  for a new CLI can stay narrow. Any other partial feature needs evidence
  and an explicit maintainer decision before merge. Reuse shared paths first.
  A working first adapter or an unchanged fallback for other tools is not
  coverage of the new behavior. If the contributor cannot or does not plan to
  cover the other applicable tools, start with an issue. An optional draft PR
  can provide a reference implementation linked to that issue. Mark it as
  reference only with its missing coverage. Keep it in draft until coverage
  is complete or the maintainer accepts an exception.
- **Every platform.** A `runtime.GOOS` branch, a platform-only command, or
  a path built from one platform's layout needs its counterpart, or the
  description names the platform it skips. WSL2 is a platform of its own:
  the binary is the Linux one, but the clipboard, notifications, and host
  stats go through Windows interop (`internal/wsl` detects it), so a Linux
  branch is checked there too.
- **Every terminal.** The manager draws through tmux into whatever terminal
  the user runs, and over SSH that terminal is on another machine. A
  feature that talks to the terminal (clipboard, links, notifications,
  themes, mouse) works on the plain path first, and a branch keyed on
  `TERM_PROGRAM`, `TERM`, or `SSH_CONNECTION` is an addition to that path,
  never the only way the feature works. A tmux version gate names the
  version and keeps the feature usable on tmux 3.1, the oldest
  [docs/install.md](docs/install.md) supports.

The reason is maintenance. One binary serves every user, and a value frozen
in code or in a user's file stops receiving fixes the day it is written.

## Build and test

```bash
go run .                                        # run the TUI
go build ./...
make build                                      # ./aghelm stamped with git describe
make check                                      # everything below, in one target
mkdir -p /tmp/amtest && env -u TMUX TMUX_TMPDIR=/tmp/amtest SHELL=/bin/sh go test -race ./...
```

The suite drives a real tmux server. When your own shell already runs inside
tmux, `$TMUX` overrides `TMUX_TMPDIR` and the tests land on the live socket,
so `env -u TMUX` is mandatory, never a bare `go test`. Keep the socket dir
short: `TMUX_TMPDIR/tmux-<uid>/default` must stay under 104 characters or
tmux silently falls back to the default socket. The directory must exist
before tmux starts, for the same reason. Never run `tmux kill-server` or
`kill-session` against the default socket; kill stray processes by PID.

Panes fall back to `$SHELL` when their agent exits, and a shell whose startup
files fork or print throws off the timing tests. `internal/ui` pins
`SHELL=/bin/sh` in its `TestMain`; for the other packages that drive tmux, run
with `SHELL=/bin/sh` as `make check` does.

On Windows, see Checks in [CONTRIBUTING.md](.github/CONTRIBUTING.md) for
what the suite needs.

`config.Dir` is the directory that holds `state.db`. It is
`AGHELM_CONFIG_DIR` when that variable is set, on every OS, and
otherwise `$XDG_CONFIG_HOME/aghelm` when `XDG_CONFIG_HOME` is set.
`os.UserConfigDir` ignores `XDG_CONFIG_HOME` on macOS, so a probe that
must not open the user's store sets the first:

```bash
AGHELM_CONFIG_DIR=/tmp/amprobe aghelm
```

`go test` reads a temp directory while the process is still on the user
config it started with, including an `XDG_CONFIG_HOME` the environment
already had. A test that sets `AGHELM_CONFIG_DIR`,
`XDG_CONFIG_HOME`, `HOME`, or `APPDATA` still selects its own. The
substitution stays in the test process.

Before finishing: `make check` passes. It runs gofmt, `go vet`, staticcheck,
the checks in `tools/architecture` (including `check-identity`, which rejects
the upstream project's names outside its exception list) and the race suite.
`make vulncheck` runs govulncheck and needs the network.

A change to the UI, the poller, or a tool's status rules is verified against
real sessions before the commit says so: build the binary, run it under a
throwaway `HOME` on its own socket (see Capturing a TUI frame in
[CONTRIBUTING.md](.github/CONTRIBUTING.md)), drive a real CLI in it, and read
the frames back. A green suite is not that.

## Invariants

Rules the code depends on that no compiler checks. Each one has broken a
release or a user's sessions before.

- **Every tmux call names its socket.** Go through `Driver` in
  `internal/tmux`, whose `args` pins `-L aghelm`. A bare
  `exec.Command("tmux", ...)` resolves through `$TMUX` to the developer's own
  server. A global option, `kill-session`, or `set-hook` on a socket this
  process does not own is data loss.
- **`pane-base-index` stays 0.** `EnsureBindings` pins it so `.0` is the
  agent's pane. A CLI that splits its window (Claude Code teammates run as
  splits of the managed window) leaves the agent in pane 0, and a target
  that names the session instead of the pane lands on the wrong one.
- **Read panes through `Engine.Plain`, not `capture-pane -e`.** A running
  step's marker blinks; its off frame is a styled blank cell only the
  escape-preserving capture shows, and a rule matched on it flaps.
- **`Update` never blocks.** Bubble Tea runs one `Model`; synchronous I/O,
  exec, or a sleep on the update path goes into a `tea.Cmd`.
- **The migrations slice in `internal/store` is append-only.** The loop
  swallows "duplicate column", so an edited or reordered entry reaches fresh
  installs only while upgraded databases keep the old shape. New schema is a
  new entry at the end, written so re-running it is harmless.
- **`builtinTools` is the only source of tool definitions.** It loads on
  every start, so an edited pattern reaches every install built from that commit.
  A default belongs in code.
- **The key tables and the editor are read from the store, by every
  process.** The manager, the CLI and the MCP server each install the tmux
  bindings, so a table read from anywhere else gives a session the wrong
  keys. A `config.toml` from an earlier release is read once, by the import
  in `internal/store`.
- **No speculative branches.** A fallback or nil-guard for a shape that
  cannot occur here is a defect, not safety. Verify the real shape first.
  Guards on external input are correct.

## Pitfalls

- `git stash` is repo-wide across worktrees. With several sessions on one
  checkout, a pop lands on someone else's work. Commit to the branch instead.
- A tmux `send-keys` stops around 1024 bytes; paste through `load-buffer`,
  which `Driver` already does, with a buffer name that carries the pid so
  the manager, the MCP server, and the CLI never swap text between panes.

## Concurrent sessions

Multiple agent sessions share this checkout. Do feature work in an isolated
git worktree (`git worktree add`), or your edits get swept into someone
else's commit. Branch from `origin/main` after a fetch, not from the local
`main` ref.

## Layout

- `main.go` dispatches subcommands (`rename`, `review-repo`, `sessions`,
  `spawn`, `mcp`, and the rest of the workspace CLI) and boots the TUI.
- `internal/ui` is the Bubble Tea program: one `Model` that routes messages
  and composes the frame. Each feature is a child package
  (`internal/ui/<feature>`: help, review, focus, rail, settings, spawn, quick,
  dialogs, notices, onboarding, connections, geometry, composer, pathcomplete, preview,
  frame, statusbar, startup, session, launchhint, terminal, workspace;
  `presentation`, `theme` and `surface` are shared). Cards, resize and
  focus are surfaces on a stack over the base screen (list or Review), read
  from the top for input; there is no mode enum
  ([docs/architecture.md](docs/architecture.md#decisions)). A package never imports the
  root. It changes root state by returning values (`rail.Decision`,
  `rail.Intent`), or, where a surface's signature returns only a `tea.Cmd`,
  through `Root` calls that each carry out one decision
  (`settings.Root.Apply`, `dialogs.ForkRoot.Submit`). Store, tmux
  and Git work runs on the ordered effect lane: `ui/effects` holds the
  contract, each package declares its own requests, and the root only binds
  and applies them in `effects_adapter_executor.go`. Root files are the
  per-feature adapters, `<feature>_adapter*.go`, `<feature>_*_effect.go` and
  `<feature>_*_surface.go`, so `ls internal/ui/settings_*` is the whole
  Settings adapter; only `keys.go`, `mouse.go`, `updates.go` and `ids.go`
  route. A new feature is a package and adds its name to
  `tools/architecture/check-file-names`. The package map, the checks `make check` runs and
  the reasons are in [docs/architecture.md](docs/architecture.md).
- `internal/tmux` owns the dedicated tmux socket and control-mode client;
  `internal/store` is the SQLite state, Settings included; `internal/status`
  classifies pane output into agent states; `internal/config` holds the
  built-in tool rules.

## Style

- Comments are rare and explain a non-obvious why, never what the code does.
- Tests live next to the file they cover: a test for `listview.go` belongs
  in `listview_test.go`, or `listview_<aspect>_test.go` when it splits.
  `tools/architecture/check-file-names` fails on any other name; shared
  fixtures go in `*_helpers_test.go`.
- A Go source file stays under 1,000 lines;
  `tools/architecture/check-file-sizes` fails on one that reaches it.
