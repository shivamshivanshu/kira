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

- `kira --help` and `kira <command> --help` — the command surface.
- [schema/kira.json](schema/kira.json) — published JSON schema for every `--json` output and the automation hook-stdin payload.
- `.kira/config.yaml` — the annotated project configuration; `kira config init` scaffolds personal preferences.
