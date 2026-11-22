# Contributing to aghelm

aghelm is a fork of the project named in [NOTICE](../NOTICE). Bug reports and pull requests go to [Doremi203/aghelm](https://github.com/Doremi203/aghelm/issues/new); a fix that applies to the upstream project is worth sending there as well.

## What a change covers

A rough patch that works is welcome. For a large change (new UI, new keybinding, reworked status detection), open an issue first to agree on the approach.

Follow the [Thin Wrapper Principle](../PRODUCT.md#thin-wrapper-principle): preserve
each agent TUI's defaults and user configuration, keep shared features generic,
and avoid integrations that need updates whenever an upstream tool changes.
For example, a model picker needs reliable dynamic discovery across supported
tools; a maintained list of model names does not belong in aghelm.

New shared features must work for every applicable CLI in the same PR.
Notifications, copying replies, and session controls are shared workspace
features even when the first implementation is easiest for one tool. Reuse
the existing shared paths before building a separate provider integration.
Keeping the old behavior for the other tools does not cover the new feature.

A bug fix specific to one tool or support for a new CLI can stay narrow.
Other partial features need evidence that the scope makes sense and an explicit
maintainer decision before merge. A documented upstream capability may justify
an exception. Listing exclusions or promising follow-up PRs does not.
See [tool coverage](../REVIEW.md#tool-coverage) for the evidence required.

If you want a feature for one tool but cannot or do not plan to build it for
the other applicable tools, [open an issue](https://github.com/Doremi203/aghelm/issues/new)
first. You may also open a draft PR linked to that issue as a reference
implementation. Mark it as reference only, name the missing tool coverage, and
keep it in draft until the coverage is complete or the maintainer accepts an
exception.

Build against the current release of each agent CLI. None of them publishes a
long-term-support line, so the latest stable release is the one we support.
When an upstream release changes a flag, a subcommand, or a screen, move to
the new behavior and drop the old one. Set that tool's `minimum_version` in
`internal/config/builtin.go` to the first release with the new behavior, in
the same pull request, so someone on an older release gets the update dialog.

A change is done when it holds on the whole matrix: every supported CLI, every
platform we build plus WSL2, every terminal including over SSH, keyboard and
mouse. A setting a user can change lives on a Settings row that the manager
stores, never in a file or variable the user edits by hand. [AGENTS.md](../AGENTS.md#what-every-change-covers) spells out
each of these with the reason behind it.

## Setup

You need Go 1.27.4+, tmux 3.1+, and git.

```bash
git clone https://github.com/Doremi203/aghelm.git
cd aghelm
go run .
```

aghelm runs its sessions on a dedicated tmux socket (`-L aghelm`), so a development build stays clear of the tmux server your own shell is attached to and of an upstream install's.

`make build` writes `./aghelm` (`aghelm.exe` on Windows) and `make install` puts it in your Go bin directory. Both stamp the commit from `git describe`, such as `0.47.1-10-g1656b508-dirty`, which Settings and `--version` show. A build without the stamp, such as `go run .` or a plain `go build`, reports `dev`, except a plain `go build` on a clean checkout of a tag: Go stamps that tag as the module version. Add `-buildvcs=false` to keep it `dev`. Without `make`, pass the stamp yourself: `go build -ldflags "-X main.version=$(git describe --tags --always --dirty --long)" .`

## Checks

There is no CI. Run `make check` before every commit:

```bash
make check
```

It runs, in order, and stops at the first failure:

- `gofmt -l .`, which must print nothing
- `go vet ./...`
- staticcheck, at the version pinned in the `Makefile`
- the architecture checks in `tools/architecture` (`check-ui-boundaries`, `check-file-names`, `check-file-sizes`, `check-identity`)
- `go test -race ./...` with `TMUX` unset, `TMUX_TMPDIR=/tmp/amtest` and `SHELL=/bin/sh`

The test suite includes end-to-end tests against a real tmux server on its own socket. Install tmux to run them; tmux-dependent tests are skipped when tmux is unavailable. `make check` unsets `TMUX` itself, so the tests never land on the server your shell is in.

`make vulncheck` runs govulncheck against the Go vulnerability database. It needs the network, so it stays out of `make check`; run it after a dependency change. The real-terminal scenarios in `tools/e2e` and the side-by-side checks in `tools/compatibility` run by hand; their READMEs say how.

`make check` runs shell commands (`test`, `mkdir -p`, `env -u`), so on Windows it needs a POSIX shell such as the one Git for Windows ships. Without one, run the steps by hand. On Windows the suite needs psmux 3.4.0 or newer, PowerShell 7.3 or newer, and `cat`, `echo`, `printf` and `true` from Git for Windows (`C:\Program Files\Git\usr\bin`) on `PATH`. The `ui` package outruns go test's default timeout, and `-race` needs cgo:

```powershell
go test -timeout 40m ./...
```

## Code style

- Match the surrounding code. Names describe intent; comments explain a non-obvious *why* and stay rare.
- Keep changes focused on one thing. A change that fixes a bug and refactors three packages is two commits.
- Add tests for behavior you change, next to the file you touched.

## Commits and pull requests

Commit subjects follow Conventional Commits, matching the existing history:

```
feat(ui): add mouse support to the sessions rail
fix(status): treat an Esc interrupt as waiting
docs: document the review-base subcommand
```

In a pull request description, say what changed and why, and how you verified
it. For a shared feature, account for every CLI in `builtinTools` with its
behavior and verification, or evidence that the feature does not apply. For a
tool-specific fix, name the affected tools and why the others are unaffected.
Name excluded platforms and what covering them would take. Include before and
after frames when the change is visible; see below.

## Capturing a TUI frame

The manager draws full screen, so a photo of your own terminal carries
everything else on your screen and cannot show a mouse gesture at all. Drive a
built binary on its own tmux socket instead and capture the frames it paints.

```bash
go build -o /tmp/amcap/bin/aghelm .   # build first, before HOME moves

mkdir -p /tmp/amcap/home /tmp/amcap/sock
export HOME=/tmp/amcap/home          # throwaway config and store
export TMUX_TMPDIR=/tmp/amcap/sock   # keep it short, see AGENTS.md
unset TMUX
tmux -L outer new-session -d -x 110 -y 30 /tmp/amcap/bin/aghelm
```

`unset TMUX` and a short `TMUX_TMPDIR` matter for the same reason they matter
to the suite; see Build and test in [AGENTS.md](../AGENTS.md). The socket
directory also has to exist before tmux starts. When it does not, tmux falls
back to your default socket without saying so, and the run lands on your real
sessions. The manager starts its own `aghelm` server under that directory,
so `tmux -L aghelm` reaches the sessions it spawns without touching your
own. A fresh `HOME` opens the first-run setup screen once; `esc` skips it.

Keys go in with `send-keys`, mouse events as raw SGR bytes through `send-keys -H`:

```bash
sgr() { printf "\033[<%s;%s;%s%s" "$1" "$2" "$3" "$4" | od -An -tx1 | tr -d '\n'; }

tmux -L outer send-keys -H $(sgr 0 34 5 M)    # left press at column 34, row 5
tmux -L outer send-keys -H $(sgr 32 48 5 M)   # motion, dragging to column 48
tmux -L outer send-keys -H $(sgr 0 48 5 m)    # release
tmux -L outer send-keys -H $(sgr 65 10 6 M)   # wheel down, 64 for up
```

SGR columns and rows are 1-based while the model's are 0-based, which is the
one thing that will cost you a run.

Read a frame back with `tmux -L outer capture-pane -p`, or `-p -e` to keep the
colors. Paste the before and after into the PR. Kill both servers before you
delete the directory, or the daemon is left stranded.

## Licensing

Contributions come in under the project's [Apache-2.0 license](../LICENSE), same as everything else here. There is nothing extra to sign: section 5 places any contribution you intentionally submit for inclusion under those terms unless you state otherwise.

Two clauses are worth knowing as a contributor. Section 3 licenses the patent claims you can license that your own contribution necessarily infringes, on its own or combined with the project, and it withdraws that grant from anyone who sues over the work. Section 9 lets a redistributor sell support or a warranty only on its own behalf, and requires it to indemnify, defend and hold contributors harmless for what it took on.

[REVIEW.md](../REVIEW.md) states what a review here covers.
