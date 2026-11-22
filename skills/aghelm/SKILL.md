---
name: aghelm
description: Run a fleet of AI coding agents as live tmux sessions with aghelm. Use when a developer is running more than one coding agent, needs to see which one is working or blocked, wants to spawn another on an independent task, or wants to review an agent's diff without leaving the terminal.
license: Apache-2.0
homepage: https://github.com/Doremi203/aghelm
repository: https://github.com/Doremi203/aghelm
---

<!--
Why this skill exists: it packages the operating procedure for aghelm (spawn, status,
answer-without-attaching, diff review) so an AI agent can install the knowledge once
instead of re-deriving it from the docs on every session.
-->

# Run a fleet of coding agents with aghelm

aghelm is a single Go binary on top of tmux. Every row in its list is a real tmux
session running a real coding CLI: the developer's own installation, login, config and MCP
servers. It never wraps or re-implements an agent; it reads the same pane a human would and
colours the row by what that pane is doing.

## When to use this

- The developer is running more than one coding agent and cannot tell which is blocked.
- They want to spawn an agent on an independent task without losing the one already running.
- They want an agent's conversation to survive a crash, a reboot, or a closed terminal.
- They want to review a diff and send line comments back into the agent's prompt.

Do not reach for it to *replace* a coding CLI. It starts the CLIs that are already installed.

## Install

macOS and Linux, amd64 and arm64; Windows on psmux or inside WSL2. Needs Go 1.27.4+,
tmux 3.1+ and git. aghelm has no releases; it builds from source:

```bash
go install github.com/Doremi203/aghelm@latest
```

or `make install` from a checkout. See `docs/install.md` in the repository.

## The keys that matter

| Key | What it does |
| --- | --- |
| `space` | Opens quick prompt mode, a prompt bar docked in the footer. On a group row it spawns a new agent with that prompt; on a session row it answers the agent already running there. |
| `tab` / `alt+m` | Cycles which CLI the next spawn starts. |
| `shift+tab` | Steps the CLI back one. |
| `ctrl+l` / `ctrl+x` / `ctrl+y` | Picks the spawn's model, reasoning effort and profile from the ones the CLI reports. |
| `ctrl+g` | Steps the environment profile the spawn launches under, starting from the target group's. |
| `ctrl+t` / `alt+w` | Toggles spawning the agent into a fresh git worktree and branch. |
| `ctrl+r` | Opens a full-screen whole-file diff whose line comments go back to the agent as one review prompt. |
| `v` | Revives a dead session on the conversation it held. |
| `y` | Copies the selected session's newest reply to the system clipboard, without attaching. |

## Status colours

A row is coloured `working`, `waiting`, `finished`, `idle` or `errored`. Claude Code
reports through hook events; every other CLI is read from its pane with regexes. Fourteen
CLIs ship with working profiles: Claude Code, OpenCode, Codex, Grok Build, Gemini CLI,
Antigravity CLI, Pi, Command Code, Hermes Agent, Muse Code, Oh My Pi, Cursor CLI, Kiro CLI and GitHub Copilot CLI. Every profile
ships in the binary, so a rebuild brings the current rules for all of them. A CLI
outside that list is a feature request, and the `CLIs` row in Settings ends with
`request CLI support`, which opens one prefilled.

## Driving it from inside an agent

Sessions of MCP-capable CLIs carry aghelm's own MCP server over stdio, so an agent can
`create_session` to spawn a peer, `send_session` to message one, `wait_for_session` to block
on one, `task` to claim shared work, and `reserve_files` to declare what it is editing. Once a
peer's work is saved, `delete_session` (or `aghelm delete <id>`) removes it with its
worktree and branch. A worktree with uncommitted or unpushed work, or one another session
still uses, stays with its branch, and the result's notice says so.

## Reference

- Documentation: https://github.com/Doremi203/aghelm/tree/main/docs
- Every subcommand: `aghelm --help`
