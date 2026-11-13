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

## Documentation

- [User manual](docs/user_manual/README.md) — concepts, CLI and config
  reference, TUI guide, recipes.
- `kira <command> --help`; `kira schema` prints the JSON Schema of every
  `--json` output.
- [Web UI development](internal/webui/README.md).
