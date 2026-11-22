---
name: aghelm-reference
description: Find aghelm's documentation, the coding CLIs it manages, the tools its MCP server exposes, and its shell subcommands. Use instead of guessing URLs or flags.
license: Apache-2.0
homepage: https://github.com/Doremi203/aghelm/tree/main/docs
repository: https://github.com/Doremi203/aghelm
---

<!--
Why this skill exists: it tells an AI agent where aghelm's reference material lives, so
it reads the documentation and the binary's own help instead of guessing.
-->

# Look up aghelm reference material

aghelm has no website or reference API. Its documentation is Markdown in the repository,
and the binary describes its own subcommands.

## When to use this

- Answering "does aghelm support X" or "how do I install it".
- Pulling a documentation page as Markdown for context.
- Listing the subcommands or MCP tools before telling a developer what an agent can do.

## Documentation as Markdown

Every page is a file under `docs/` on `main`:

```bash
curl https://raw.githubusercontent.com/Doremi203/aghelm/main/docs/install.md
curl https://raw.githubusercontent.com/Doremi203/aghelm/main/docs/usage.md
```

| Page | Covers |
| --- | --- |
| `README.md` | What aghelm is, the supported CLIs, install, the main keys |
| `docs/install.md` | Building from source, dependencies, Windows, uninstalling |
| `docs/usage.md` | Every key, quick prompt, review, groups, the MCP tools and the shell commands |
| `docs/configuration.md` | Settings, key bindings, the supported CLIs and their versions |
| `docs/ssh-connections.md` | Sessions on another machine over SSH |
| `docs/notifications.md` | The messages panel |

## From the binary

```bash
aghelm --help            # every subcommand
aghelm settings list     # every setting with its current value
```

The MCP server is `aghelm mcp`; every session of an MCP-capable CLI carries it, and its tools
are listed under "MCP: how agents discover these commands" in `docs/usage.md`.
