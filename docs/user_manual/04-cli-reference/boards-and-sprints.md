# Boards and sprints

`board` (the kanban view and board management) and `sprint`. The model —
boards as numbering namespaces, sprints as date ranges, the per-clone active
sprint — is in [Concepts](../03-concepts.md#boards). Config keys:
[`boards`](../05-configuration-reference.md#boards),
[`sprints`](../05-configuration-reference.md#sprints). Every board/sprint
mutation is a config change, committed under `commit.mode: auto`. Global flags:
[Setup and schema commands](setup-and-schema-commands.md#global-flags).

Samples: default board `APP`, plus `WEB` and `API`; people `alice`, `bob`.

## `kira board`

```
kira board [<parent-id>] [flags]
```

One column per state of the type, in workflow order. On a terminal, with no
`<parent-id>` and no filter, it opens the interactive board (the
[TUI](../07-tui-guide.md#board) board screen). Otherwise — or with `--plain`,
`--json`, a non-terminal stdout, or `ui.auto_tui: false` — it prints a static
table sized to the terminal (`COLUMNS`, else 80), eliding long titles with `…`.

| Flag | Meaning |
| --- | --- |
| `--plain` | Static table even on a terminal. |
| `--type` | Board for `epic` instead of `ticket`. |
| `--board`, `--state`, `--priority`, `--label`, `--owner`, `--sprint`, `--filter`, `--query` | Filters, as on [`kira list`](views-and-queries.md#kira-list). |
| `--at` | Ref or date; static, read-only. |

A column with a WIP limit shows `count/limit`. Glyphs: `[ ]` todo, `[~]` doing,
`[x]` done. A filter flag narrows the counts to the filtered set; a
`<parent-id>` scope keeps whole-column counts.

```console
$ kira board --plain
TODO                          │IN_PROGRESS  2/3             │REVIEW  1/2                  │DONE                         │WONT_DO
──────────────────────────────│─────────────────────────────│─────────────────────────────│─────────────────────────────│─────────────────────────────
     [ ] APP-1  Ship 1.0 cha… │     [~] API-1  Rate-limit … │     [~] WEB-3  Dark mode t… │                             │
     [ ] WEB-4  Fix session … │     [~] WEB-2  Wire up log… │                             │                             │
$ kira board --plain --owner alice
TODO                          │IN_PROGRESS  1/3             │REVIEW  0/2                  │DONE                         │WONT_DO
──────────────────────────────│─────────────────────────────│─────────────────────────────│─────────────────────────────│─────────────────────────────
     [ ] WEB-4  Fix session … │     [~] WEB-2  Wire up log… │                             │                             │
$ kira board --plain WEB-1
TODO                          │IN_PROGRESS  2/3             │REVIEW  1/2                  │DONE                         │WONT_DO
──────────────────────────────│─────────────────────────────│─────────────────────────────│─────────────────────────────│─────────────────────────────
                              │     [~] WEB-2  Wire up log… │     [~] WEB-3  Dark mode t… │                             │
```

## `kira board list`

Boards in config order with `(default)` / `(archived)` markers; `--json` adds
`description`.

```console
$ kira board list --json
{
  "boards": [
    {
      "key": "APP",
      "name": "boards",
      "default": true
    },
    {
      "key": "WEB",
      "name": "Web Frontend",
      "description": "Browser client work"
    },
    {
      "key": "API",
      "name": "Public API"
    }
  ]
}
```

The default board is changed only by editing `default: true` in config.

## `kira board create`

```
kira board create <KEY> [--name <text>] [--description <text>]
```

`KEY` matches `^[A-Z][A-Z0-9]{1,9}$`; `--name` defaults to the key.

```console
$ kira board create WEB --name "Web Frontend" --description "Browser client work"
Created board WEB
$ kira board create docs
kira: board key "docs" must match ^[A-Z][A-Z0-9]{1,9}$
$ kira board create API
kira: board "API" already exists
```

## `kira board rename`

```
kira board rename <KEY> <name>
```

Changes the display name; the key is immutable.

```console
$ kira board rename APP "Application"
Renamed board APP to Application
$ kira board rename NOPE X
kira: no such board "NOPE"
  hint: boards: APP, WEB, API
```

## `kira board move`

```
kira board move <item> <KEY>
```

Renumbers the item onto the target board's next number and appends the old
number to `aliases`, so every old number keeps resolving to the same id.

```console
$ kira board move WEB-4 API
Moved WEB-4 -> API-2
$ kira show WEB-4
API-2  Fix session cookie expiry  [TODO]
...
$ kira board move API-2 APP
Moved API-2 -> APP-2
$ grep -E "^(number|aliases):" $(grep -l "Fix session cookie" .kira/tickets/*.md)
number: APP-2
aliases: [WEB-4, API-2]
$ kira board move APP-2 APP
kira: APP-2 is already on board APP
```

## `kira board archive` / `unarchive`

```
kira board archive <KEY>
kira board unarchive <KEY>
```

An archived board stays in `board list` as `(archived)`; its items still list,
show and resolve; it is no longer a target for `create --board` or
`board move`, and hints omit it.

```console
$ kira board archive WEB
Archived board WEB
$ kira show WEB-2 --format number-title
WEB-2 Wire up login form
$ kira create ticket "New on archived" --no-edit --board WEB
kira: no such board "WEB"
  hint: boards: APP, API
$ kira board unarchive WEB
Unarchived board WEB
```

## `kira sprint create`

```
kira sprint create --key KEY --name NAME --start YYYY-MM-DD --end YYYY-MM-DD
```

All four are required; `start` must be before `end`.

```console
$ kira sprint create --key S1 --name "Sprint 1" --start 2026-11-02 --end 2026-11-13
Created sprint S1 (2026-11-02 -> 2026-11-13)
$ kira sprint create --name "Sprint 3"
kira: config: sprints: sprint with empty key
$ kira sprint create --key S3 --start 2026-11-30 --end 2026-12-11
kira: config: sprints[S3]: empty name
$ kira sprint create --key S3 --name "Sprint 3" --start 30-11-2026 --end 2026-12-11
kira: config: sprints[S3].start: invalid date "30-11-2026", want YYYY-MM-DD
$ kira sprint create --key S3 --name "Sprint 3" --start 2026-12-11 --end 2026-11-30
kira: config: sprints[S3]: start 2026-12-11 is not before end 2026-11-30
```

## `kira sprint list`

Config order; `*` marks the active sprint; `done/total` counts items whose
`sprint` names it, "done" being any `done`-category state.

```console
$ kira sprint list
* S1  Sprint 1  2026-11-02 -> 2026-11-13  1/4 done
  S2  Sprint 2  2026-11-16 -> 2026-11-27  0/0 done
```

With none: ``No sprints configured (use `kira sprint create`)``.

## `kira sprint activate`

```
kira sprint activate <key>
```

Writes the per-clone pointer `.kira/.cache/active-sprint` (git-ignored, never
committed) that `active` resolves to.

```console
$ kira sprint activate S1
Activated sprint S1
$ git status --short
$ kira sprint activate S2
Activated sprint S2 (was S1)
$ kira sprint activate nope
kira: "nope" is not a key in the configured sprints
```

With no pointer set, `active` matches nothing and warns:

```console
$ kira list --sprint active
kira: no active sprint set; sprint=active matches nothing (run `kira sprint activate <key>`)
no items
```

## `kira sprint close`

```
kira sprint close <key> [--move-to <key>]
```

Lists items in the sprint not in a `done` state and clears the active pointer
if it named this sprint. The sprint stays in config. `--move-to` reassigns
each unfinished item to another sprint, one commit per item. States are not
changed — move abandoned work to `WONT_DO` first.

```console
$ kira sprint close S1 --move-to S1
kira: --move-to target "S1" is the sprint being closed
$ kira sprint close S1 --move-to ZZ
kira: --move-to: "ZZ" is not a key in the configured sprints
$ kira sprint close S1 --move-to S2
Closed sprint S1
  unfinished: WEB-2
  unfinished: APP-2
  unfinished: API-1
Moved 3 unfinished item(s) to S2
Cleared the active-sprint pointer
$ git log --oneline -4
2f666c8 API-1 sprint S1 -> S2
c4a68d2 APP-2 sprint S1 -> S2
ffb31de WEB-2 sprint S1 -> S2
ada2360 WEB-3 state REVIEW -> DONE
$ kira sprint list
  S1  Sprint 1  2026-11-02 -> 2026-11-13  1/1 done
  S2  Sprint 2  2026-11-16 -> 2026-11-27  0/3 done
```
