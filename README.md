# kira

A git-native, terminal-first ticket tracker. Tickets are Markdown files with
YAML frontmatter under `.kira/`, committed alongside the code they track — no
server, no database, no account. Drive it from the CLI, a terminal UI, or a
local web dashboard.

## Install

Release archives for linux/darwin × amd64/arm64 are on GitHub Releases. Or:

```sh
go install github.com/shivamshivanshu/kira/cmd/kira@latest
```

## Quickstart

```sh
kira init --key DEMO                    # in an existing git repo; commits .kira/
kira hooks install                      # per clone: field-level merge driver + hooks
kira create ticket "Fix flaky merge test" --no-edit
kira list                               # or: kira board, kira tree
kira                                    # terminal UI; `kira server` for the web dashboard
```

## Development

Run `just` to list recipes. The lint recipe uses the golangci-lint version
pinned in [CI](.github/workflows/ci.yml).

```sh
just build                         # standard build, ./kira
just release 0.1.0                 # stripped release build, ./kira
just install                       # install to /usr/local/bin/kira
just debug                         # unoptimized build, ./kira-debug
just clean                         # remove local binaries and the Kira index cache
just run --help                    # run the CLI from source
just test                          # all tests
just test ./internal/core -run TestStats
just test-race                     # all tests with the race detector
just check                         # build, vet, race tests, and lint
just perf                          # CI performance checks
```

`just install` requires write access to the destination. Pass a different
directory with `just install "$HOME/.local/bin"`.

`just clean` keeps the installed binary, tickets, config, and local active
ticket and sprint selections.

## Documentation

- [User manual](docs/user_manual/README.md) — concepts, CLI and config
  reference, TUI guide, recipes.
- `kira <command> --help`; `kira schema` prints the JSON Schema of every
  `--json` output.
- [Web UI development](internal/webui/README.md).
