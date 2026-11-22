# Plain make functions and GOFLAGS keep the recipes free of shell syntax, so
# they run under cmd.exe as well as sh, and a tag name never reaches a shell.
# .git marks this directory as a repo root, so source inside another repository
# does not read that repository's tags. --long keeps a build on a tag reading
# as a commit, never as that release.
VERSION := $(if $(wildcard .git),$(shell git describe --tags --always --dirty --long))
VERSION := $(or $(VERSION),dev)
# override keeps the stamp when GOFLAGS is set on the make command line; it goes
# first so a caller's own -ldflags, which Go applies last, still wins. go env
# supplies flags stored with go env -w when GOFLAGS is not set.
override GOFLAGS := -ldflags=-X=main.version=$(VERSION) $(or $(GOFLAGS),$(shell go env GOFLAGS))
export GOFLAGS

.PHONY: build install check vulncheck

build:
	go build -o aghelm$(shell go env GOEXE) .

install:
	go install .

check:
	test -z "$$(gofmt -l .)"
	go vet ./...
	go run honnef.co/go/tools/cmd/staticcheck@2026.2.2 ./...
	go run ./tools/architecture/check-ui-boundaries
	go run ./tools/architecture/check-file-names
	go run ./tools/architecture/check-file-sizes
	go run ./tools/architecture/check-identity
	mkdir -p /tmp/amtest
	env -u TMUX TMUX_TMPDIR=/tmp/amtest SHELL=/bin/sh go test -race ./...

vulncheck:
	go run golang.org/x/vuln/cmd/govulncheck@v1.10.0 ./...
