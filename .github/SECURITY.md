# Security Policy

aghelm has no releases. Fixes land on `main`; update your checkout (`git pull && make install`) before reporting, in case the issue is already fixed.

## Reporting a vulnerability

Report privately through GitHub: **[Report a vulnerability](https://github.com/Doremi203/aghelm/security/advisories/new)**. That opens a private advisory visible only to the owner of this fork.

Please include:

- What an attacker can do, and what they need to already have (local shell, a crafted repo, a malicious config, a hostile agent).
- Steps to reproduce, with the smallest input that triggers it.
- Version (`aghelm --version`), OS, and tmux version.

Keep the report private until a fix is on `main`. A vulnerability that also affects the upstream project belongs with its maintainers too.

## What aghelm touches

Useful context when judging whether something is a vulnerability:

- **tmux sessions.** The manager creates and drives sessions on its own tmux socket, and reads the visible text of each pane to derive status and show each session's last messages in the list. Pane content is treated as data, not commands.
- **Process spawning.** Sessions launch the CLI commands built into the binary, and the editor key runs the command set in Settings. Both run as you.
- **Local state.** Settings and state (`state.db`, SQLite) live in your OS user config directory. Session names, group paths, working directories, and review targets are stored there.
- **Generated agent config.** Launching Claude Code, Codex, OpenCode, Grok, Gemini, Antigravity, Command Code, Muse Code, Cursor CLI, Kiro CLI, GitHub Copilot CLI, or an MCP-enabled Hermes tool writes or registers generated settings or MCP configuration so the agent gets the aghelm MCP tools. Cursor CLI's entry is written to `~/.cursor/mcp.json`. Kiro CLI's entry is added with `kiro-cli mcp add --scope global`, which writes `~/.kiro/settings/mcp.json`. GitHub Copilot CLI reads its entry from a file in the manager's own directory, passed with `--additional-mcp-config`, and loads a plugin from that directory with `--plugin-dir` whose hooks report the conversation it is on. Claude Code also gets generated hook settings that report its status.
- **MCP server.** `aghelm mcp` speaks stdio to the agent in its own session and exposes session, group, task, reservation, terminal, messaging, and review operations. It identifies the calling session from its environment and scopes destructive terminal operations to the caller's own nested terminals.
- **Git repositories.** Review mode runs read-only git commands against the repo a session declares or that ranking picks.
- **Network.** aghelm makes no requests of its own to the internet: it has no updater and no message feed. Model discovery talks to a server an agent CLI starts on `127.0.0.1`, and links you click open in your browser.

Findings that involve one of these crossing a trust boundary in a way the docs do not describe are exactly what this policy is for.

## Out of scope

- Behavior of the AI agents themselves (Claude Code, Codex, OpenCode, Grok, Gemini, Antigravity, Pi, Command Code, Hermes, Muse Code, Oh My Pi, Cursor CLI, Kiro CLI, GitHub Copilot CLI). Report those to their maintainers.
- Anything requiring an attacker to already have write access to your config file or your shell.
