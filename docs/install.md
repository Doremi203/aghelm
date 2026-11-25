# Install

aghelm has no releases or packages. It builds from source, and the result is one binary, `aghelm`.

## Dependencies

| Tool | What it powers |
| --- | --- |
| Go 1.27.4+ | building the binary |
| tmux 3.1+ | every agent session |
| git | diff review and worktree sessions |
| wl-clipboard or xclip | pasting images into a prompt on Linux |
| notify-send | desktop notifications on Linux (`libnotify-bin` with apt, `libnotify-tools` with zypper, `libnotify` with the others) |

Install them with your package manager. When aghelm starts without tmux, or the diff view cannot find git, it names the install command it detected, or tells you to use your package manager.

## From a checkout

```bash
git clone https://github.com/Doremi203/aghelm.git
cd aghelm
make install
```

`make install` runs `go install .` with the version stamped from `git describe`, so `aghelm --version` names the commit. `make build` writes `./aghelm` in the checkout instead. The binary lands in `$(go env GOBIN)`, or `$(go env GOPATH)/bin` when `GOBIN` is unset; put that directory on your `PATH`.

To update, pull and install again:

```bash
git pull && make install
```

Agent sessions keep running while the binary is replaced. Restart aghelm to load the new build.

## Without a checkout

```bash
go install github.com/Doremi203/aghelm@latest
```

`aghelm --version` prints the tag Go resolved, or `dev` for a pseudo-version. Run the same command again to update.

## Next to the upstream install

aghelm keeps its own binary name, tmux socket (`aghelm`), config directory, MCP server name, hook files and notifier, so it installs and runs beside an upstream install without either one seeing the other's sessions or settings. Nothing is imported from the upstream config directory: aghelm starts with its own first-run setup. aghelm drops the upstream's session variables from its environment at startup, so launching it from an upstream session does not hand that session's identity to aghelm's sessions.

## Windows

aghelm runs natively on Windows on [psmux](https://github.com/psmux/psmux) 3.4.0 or newer, a tmux-compatible multiplexer, with PowerShell as the pane shell. Build it with `go install .` from a checkout, or with the `go install` line above; `make install` also works under cmd.exe. `make check` needs a POSIX shell.

The Windows dependencies:

| Tool | Install |
| --- | --- |
| psmux | `winget install --id marlocarlo.psmux -e` (or `cargo install psmux`). 3.4.0 or newer. Older builds misbehave and the manager refuses them |
| git | `winget install --id Git.Git -e` |
| PowerShell 7.3 or newer (`pwsh`) | `winget install --id Microsoft.PowerShell -e`. The manager refuses Windows PowerShell 5.1 and PowerShell before 7.3, because they split a prompt containing `"` into several arguments to the agent CLI |

Differences from macOS and Linux:

- Each session runs on its own psmux server.
- Inside an attached session, the review and editor keys detach through a short PowerShell `run-shell`, so they take a moment longer than on tmux.
- Command Code (`cmd`) shares its name with `cmd.exe`, so it is not supported natively: it is left out of the tool pickers, and a session asking for it is refused.
- An agent CLI installed only as a `.cmd` or `.bat` file is refused, since cmd.exe would run a prompt holding `"` as commands. CLIs installed with npm work, because npm adds a `.ps1` beside each `.cmd`.
- A CLI the setup dialog installs with npm lands in `%APPDATA%\npm`. When that directory was not on `Path` before the manager started, the manager keeps its old `Path`, so start it again before spawning that CLI.

aghelm also runs inside [WSL2](https://learn.microsoft.com/windows/wsl/install), on tmux: build it in a WSL shell as on Linux. The agent CLIs then belong in the distro too. WSL appends the Windows `PATH` to the distro's, so a CLI installed on the Windows side is visible in a WSL shell, and running it there starts a Windows process or fails without a Linux runtime. A spawn finding only that copy stops on the setup dialog, which names where the Windows copy is and the command that installs the CLI in the distro. Paste and toasts under WSL go through the Windows host.

## Uninstalling

Agent sessions keep running on their own server after aghelm quits. Stop them first with `X` in aghelm, which kills every live session in view. On macOS and Linux, `tmux -L aghelm kill-server` ends all of them at once. It touches only aghelm's private `aghelm` server, never your own tmux or an upstream install's.

Then remove the binary:

```bash
rm "$(go env GOPATH)/bin/aghelm"      # or "$(go env GOBIN)/aghelm" when GOBIN is set
```

On Windows, delete `aghelm.exe` from the same directory.

### Settings and state

Removing the binary keeps your settings, sessions and caches. They live in one directory. It is `AGHELM_CONFIG_DIR` when that is set, otherwise `aghelm` under `XDG_CONFIG_HOME` when that is set, otherwise `aghelm` under your OS user config directory. It holds `state.db` and the generated hook and MCP files the launches load. On macOS it also holds the notification helper app. Delete it to remove everything:

```bash
rm -rf ~/Library/Application\ Support/aghelm        # macOS
rm -rf "${XDG_CONFIG_HOME:-$HOME/.config}/aghelm"    # Linux
```

```powershell
Remove-Item -LiteralPath (Join-Path $env:APPDATA 'aghelm') -Recurse -Force
```

On macOS the notification sounds are copied to `~/Library/Sounds` as `aghelm-*.aiff`. Remove them with `rm ~/Library/Sounds/aghelm-*.aiff`.

Some agent CLIs keep an `aghelm` MCP server entry in their own user config, which aghelm adds when it first launches them. Grok, Gemini CLI, Command Code, Antigravity and Hermes register it through their own `mcp add` command, so remove it with that CLI's MCP commands. Muse Code keeps it in `muse/settings.json` under `XDG_CONFIG_HOME` (`~/.config` by default), and Cursor CLI in `~/.cursor/mcp.json`. Delete the `aghelm` entry from those files. The other CLIs load aghelm's files from its own directory on each launch, so deleting that directory removes them. Launching Grok also sets `theme = "terminal"` under `[ui]` and `terminal_theme = true` under `[features]` in `~/.grok/config.toml` (or `config.toml` in `GROK_HOME`). Change those back if you want Grok's own theme.
