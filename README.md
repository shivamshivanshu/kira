# kira

A git-native, local-first, terminal-first ticket tracker. Tickets live as files in
`.kira/` inside your repository, version-controlled alongside your code — no server,
no database, no account. Work from the command line or the interactive TUI.

## Quickstart

```sh
kira init                              # create a .kira/ store in the current git repo
kira create ticket --title "Fix flaky merge test"
kira list                              # list tickets and epics
kira board                             # kanban board grouped by workflow state
kira                                   # launch the interactive TUI (default with no command)
```

## Documentation

- [DESIGN.md](DESIGN.md) — architecture overview and the canonical entry point.
- [docs/design/](docs/design/) — detailed specs: data model, storage/git, CLI, TUI, telemetry, testing.
- [docs/ROADMAP.md](docs/ROADMAP.md) — milestone execution plan.
- [docs/STRETCH_GOALS.md](docs/STRETCH_GOALS.md) — proposed and deferred ideas.
