# aghelm

aghelm is a modified version of [agent-manager](https://github.com/YoanWai/agent-manager).

A local terminal app that runs every AI coding agent CLI you already have in one workspace. It is a fork that builds from source, keeps its own binary, tmux socket, config directory and MCP server name, and makes no requests to the upstream project, so it runs side by side with an installed upstream copy. See [NOTICE](NOTICE) for what changed.

![three agents spawned from the quick prompt bar, one each on Claude Code, Codex and Gemini CLI, tab picking the CLI each time, each new agent landing in the list](docs/demo.gif)

Claude Code, Codex, OpenCode, Grok, Gemini CLI, Antigravity CLI, Pi, Command Code, Hermes Agent, Muse Code, Oh My Pi, Cursor CLI, Kiro CLI, and GitHub Copilot CLI run side by side. Each tool runs in its own persistent tmux session.

aghelm is a thin layer over the CLIs you already have. Each session launches your own installed tool as-is: your login, your subscription, your config files, your MCP servers, and every feature the tool ships all carry over, exactly as they behave in a plain terminal.

Instead of hunting through terminal tabs to see which agent is done and which is stuck, every session shows up in one list with live status, grouped into a project tree you can fold and reorder. You answer any of them without attaching: `space` sends a prompt straight into a session's pane, or spawns a new agent in the selected group. A dead session revives on its own conversation with `v`. And `ctrl+r` opens a full-file diff of what an agent changed, syntax-highlighted, where the comments you leave on lines go back to the agent's pane as one review prompt when you press `C`.

Press `f` on a session to continue its conversation in a separate named fork.

The mouse works across the whole manager. The wheel moves the cursor, a click selects a row, and a double click attaches to a session or folds a group. Drag a row by its handle (`⠿` in Advanced, `=` in Compatible) to reorder it or move it into another group, and the `[…]` at the end of every row opens its actions. While any message is left, a click on `messages` in the key legend opens it. Settings has a mouse toggle for terminals where you want native selection back. Settings → **symbols** switches between **Advanced** geometric marks (the default) and **Compatible** ASCII marks, with a live preview.

![Mouse demo with the geometric marks used by Advanced mode: the wheel moves the cursor, clicks focus and leave sessions, the ⠿ handle drags rows, and […] opens row actions](docs/demo-mouse.gif)

The tools you use alongside agents live in the same list: `T` opens a shell under the selected agent, or in the selected group, for builds, Git, and one-off commands. An agent can spawn another, send it a message, and wait until it is done: every MCP-capable session carries those tools on launch.

Save named instructions in Settings and reuse them from the New Session form, the quick prompt, `spawn --saved`, `send --saved`, or `create_session`. See [Saved prompts](docs/usage.md#saved-prompts).

Not here yet: cost tracking.

**Jump to:** [Install](#install) · [Usage](#usage) · [Keys](docs/usage.md#keys) · [Diff review](docs/usage.md#diff-review) · [Configuration](docs/configuration.md) · [Docs](docs/)

## Supported tools

Status detection supports **Claude Code**, **OpenCode**, **Codex**, **Grok Build**, **Gemini CLI**, **Antigravity CLI**, **Pi**, **Command Code**, **Hermes Agent**, **Muse Code**, **Oh My Pi**, **Cursor CLI**, **Kiro CLI**, and **GitHub Copilot CLI**. Each one's launch, revive, fork, and status rules ship in the binary (Hermes, Antigravity, Oh My Pi, Cursor CLI, Kiro CLI, and GitHub Copilot CLI have no fork), so an upgrade brings the current version of all of them (see [Configuration](docs/configuration.md#agent-clis)). A CLI that is not on the list is a [feature request](https://github.com/Doremi203/aghelm/issues/new).

## Install

Runs on macOS and Linux, natively on Windows with [psmux](docs/install.md#windows), and inside WSL2. aghelm has no releases; it builds from source with Go 1.27.4+ and needs tmux 3.1+ and git.

```bash
git clone https://github.com/Doremi203/aghelm.git
cd aghelm
make install            # go install ., stamped with git describe
```

or, without a checkout:

```bash
go install github.com/Doremi203/aghelm@latest
```

Either one installs `aghelm` to `$(go env GOBIN)`, or `$(go env GOPATH)/bin` when `GOBIN` is unset. To update a checkout, run `git pull && make install`. Dependencies, Windows, and uninstalling: [docs/install.md](docs/install.md).

The MCP server that every session carries is named `aghelm`, so the agent CLIs ask again for permission to use its tools (`mcp__aghelm__*` in Claude Code) even where they had allowed the upstream ones.

## Usage

```bash
aghelm
```

Sessions run inside tmux (`am_*` namespace), so they survive the manager quitting. Inside a session, **Ctrl+Q** detaches back to the manager when your terminal and tmux leave it available; **Ctrl+\\** is an alternate under the same rule. **Ctrl+C** reaches the agent as its cancel key inside a session; in the list it quits the manager. **Ctrl+R** opens the session's diff review and **F3** opens its directory in your editor; the **keybindings** row in Settings (`s`) moves any of them, or hands one back to the agent, and moves or turns off any action of the manager's own list, with `esc` and `ctrl+c` staying as they are (see [Key bindings](docs/configuration.md#key-bindings)). In an attach, the session footer also shows an inner tmux prefix followed by `d` when configured. When nested inside another tmux, send the inner prefix shown in the footer, then press `d`. If both tmux servers use the same prefix, invoke the outer tmux's `send-prefix` binding; if the outer tmux otherwise captures the inner prefix, configure it to forward that key. `aghelm --version` prints the version.

Agent sessions live on a private tmux server named `aghelm`, so they never mix with the tmux you run yourself and a `kill-server` on your own socket leaves them alone. To reach one from a plain shell, name that server: `tmux -L aghelm ls`, then `tmux -L aghelm attach -t am_<id>`.

The full reference, every key, quick prompt mode, killing and reviving, diff review, groups, status detection, stats, and themes, lives in [docs/usage.md](docs/usage.md). Tell your agent what you want to review in aghelm, and your agent will set it up for you to view in the review panel. You can also tell your agent to manage sessions and terminals in aghelm. The short version:

| Key | Action |
|-----|--------|
| `n` | New session (name, tool, the model, effort and profile the tool reports, environment profile, directory, optional starting prompt, group) |
| `space` | Quick prompt mode: answer the selected session, or spawn an agent in the selected group |
| `enter` | Attach the session full screen / fold or unfold a group |
| click / double click | Select the row / attach the session, or fold or unfold a group |
| drag `⠿` / `=` | Reorder a row, drop it into another group, or nest a terminal under an agent |
| `[…]` / right click | The row's actions: attach, prompt, review, fork, rename, move, archive, kill, delete and more |
| `→` / `←` | Open / close a group |
| `ctrl+r` | Review the session's changes as full-file diffs; `c` comments a line, `C` sends a numbered review round, and sent comments stay visible as open or handled |
| `x` / `v` | Kill a session to free its RAM / revive it on its own conversation |
| `R` | Restart a session on an empty context: same name, group, directory and tool, fresh conversation |
| `s` | Settings (default tool, theme or follow the OS light/dark mode, list density, mouse, review layout, desktop notifications, editor, keybindings) |
| `?` | The key map for the current screen; review shows only review bindings |

A session can spawn into its own git worktree (`<repo>-worktrees/<name>`, branch `am/<name>`), toggled on the `n` form, with `ctrl+t` in quick prompt mode, or by default in Settings.

A session can start on a model, reasoning effort and profile of its own, picked on the `n` form, with `ctrl+l`, `ctrl+x` and `ctrl+y` in quick prompt mode, or with `spawn --model --effort --profile`. The lists come from each CLI at runtime, so a new model shows up the day the CLI ships it, and restart, revive and fork keep the choice (see [Model, effort and profile](docs/usage.md#model-effort-and-profile)).

A session can also launch under an environment profile, a named set of environment variables such as a second `CLAUDE_CONFIG_DIR`, defined with `aghelm env set` and picked per session or as a group's default (see [Environment profiles](docs/usage.md#environment-profiles)).

![the session tree, with a Claude Code session's latest answer in the preview](docs/screenshot-sessions.png)

![the session list taking the whole terminal, one-line rows with the computer stats on the foot, then a session focused full screen](docs/demo-fullscreen.gif)

![the same full-screen list at the comfortable density: every session on three lines, its task and the agent's last message under the name](docs/demo-fullscreen-comfortable.gif)

![review, side by side, with the changed lines tinted in full file context](docs/screenshot-review.png)

Configuration (Settings, the editor `o` opens, and the key tables) is in [docs/configuration.md](docs/configuration.md).

## Development

```bash
go run .
make check     # gofmt, vet, staticcheck, the architecture checks, and the race suite against a real tmux server
```

See [CONTRIBUTING.md](.github/CONTRIBUTING.md) for the checks, and [AGENTS.md](AGENTS.md) if you point a coding agent at this repo. Security reports: [SECURITY.md](.github/SECURITY.md).

## License

[Apache-2.0](LICENSE). See [NOTICE](NOTICE).
