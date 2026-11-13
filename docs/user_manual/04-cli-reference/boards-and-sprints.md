# Boards and sprints

Boards and sprints are the two ways kira groups tickets across the whole
project. A **board** is a permanent namespace: it owns the display numbers of
the tickets filed under it (`WEB-1`, `API-7`), and every ticket belongs to
exactly one. A **sprint** is a time box: a named date range that any ticket can
be tagged with, and that tickets move in and out of freely.

Both are declared in `.kira/config.yaml`, so both are committed and shared with
your teammates — with one deliberate exception, the *active sprint* pointer,
which is local to your clone. That distinction is covered in detail under
[`sprint activate`](#sprint-activate).

All commands accept the global flags `-C`, `--json` and `--no-color`; see the
global flags reference.

---

## `kira board` — the kanban view

Renders your tickets as a kanban board: one column per workflow state, in the
order the workflow declares them. With no arguments and no filters, on a
terminal, it launches the **interactive board** — a full-screen view you can
navigate and act from. Anything that narrows the board (a positional epic id,
or any of the filter flags) switches it to the **static table** printed to
stdout, as does `--plain`, `--json`, `--non-interactive`, a non-terminal stdout
(a pipe or a file), and `ui.auto_tui: false` in config.

Passing an epic id scopes the board to that epic's children, which is the usual
way to look at one workstream in isolation.

| Flag | Argument | Meaning |
| --- | --- | --- |
| `--at` | git ref | Render the board as it stood at a git ref. Static and read-only. |
| `--filter` | name | Apply a named saved query from the `filters:` block in config. |
| `--label` | label | Show only tickets carrying this label. |
| `--owner` | name | Show only this owner's tickets. `@me` resolves to your git user. |
| `--plain` | — | Force the static table instead of launching the interactive board. |
| `--query` | expression | Filter by a query expression. ANDed with the other flags. |

Column headers show `STATE  count/wip` when the workflow sets an advisory WIP
limit on that state, and the bare state name when it does not. The glyph before
each ticket number reflects its state category: `[ ]` for to-do, `[~]` for
in-progress, `[x]` for done.

```console
$ kira board --plain
TODO                            │IN_PROGRESS  2/3               │REVIEW  1/2                    │DONE                           │WONT_DO
────────────────────────────────│───────────────────────────────│───────────────────────────────│───────────────────────────────│───────────────────────────────
     [ ] APP-1  Ship 1.0 chang… │     [~] API-1  Rate-limit /v… │     [~] WEB-2  Dark mode tog… │                               │
     [ ] WEB-3  Fix session co… │     [~] WEB-1  Wire up login… │                               │                               │
```

Here `IN_PROGRESS  2/3` means two tickets against a WIP limit of three;
`DONE` and `WONT_DO` carry no limit, so they show no counter.

Filters compose. To see only what one person is carrying:

```console
$ kira board --plain --owner alice
TODO                            │IN_PROGRESS  1/3               │REVIEW  0/2                    │DONE                           │WONT_DO
────────────────────────────────│───────────────────────────────│───────────────────────────────│───────────────────────────────│───────────────────────────────
     [ ] WEB-3  Fix session co… │     [~] WEB-1  Wire up login… │                               │                               │
```

Note that the counters track the *filtered* set, not the whole column — with
`--owner alice` the `IN_PROGRESS` header reads `1/3`, because only one of the
two in-progress tickets is hers. Scoping to an epic behaves the same way:

```console
$ kira board --plain WEB-4
TODO                            │IN_PROGRESS  2/3               │REVIEW  0/2                    │DONE                           │WONT_DO
────────────────────────────────│───────────────────────────────│───────────────────────────────│───────────────────────────────│───────────────────────────────
                                │     [~] WEB-1  Wire up login… │                               │     [x] WEB-2  Dark mode tog… │
```

`--at` reads the ticket set out of a git tree rather than your working copy,
which is how you answer "what did the board look like before the release?":

```console
$ kira board --plain --at HEAD~20
```

The static table sizes itself to your terminal width, falling back to the
`COLUMNS` environment variable and then to 80 columns when stdout is not a
terminal. Long titles are elided with `…`; widen the terminal (or set
`COLUMNS`) to see more.

---

## `kira board list`

Lists every configured board, in config order, as `KEY  Name` plus any status
markers. The board tagged `(default)` is the one `kira create` files new
tickets under when you do not pass `--board`. `(archived)` marks a board that
has been retired — see [`board archive`](#kira-board-archive).

This command takes no flags beyond the global ones.

```console
$ kira board list
APP  Application (default)
WEB  Web Frontend (archived)
API  Public API
```

With `--json` you get the same data structurally, including each board's
optional description:

```console
$ kira board list --json
{
  "boards": [
    {
      "key": "APP",
      "name": "Application",
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

Which board is the default is a config setting, not something `board create`
or `board rename` can change; edit the `default: true` marker in the `boards:`
block of `.kira/config.yaml` to move it.

---

## `kira board create`

Adds a board to the config. Board keys are short uppercase codes that become
the prefix of every ticket number on that board, so they are worth choosing
carefully — you will type them constantly. kira's own repository is a good
illustration: it splits its work across `KIRA` (the default), `CORE`, `TUI`,
`CLI`, `DATA`, `INFRA` and `DX`, which is roughly one board per subsystem.

The key must match `^[A-Z][A-Z0-9]{1,9}$`: an uppercase letter followed by one
to nine more uppercase letters or digits. Creating a board is a config
mutation, so under the default `commit.mode: auto` it is committed to `.kira/`
immediately.

| Flag | Argument | Meaning |
| --- | --- | --- |
| `--name` | text | Display name. Defaults to the key. |
| `--description` | text | Optional longer description, stored in config. |

```console
$ kira board create WEB --name "Web Frontend" --description "Browser client work"
Created board WEB
$ kira board create API --name "Public API"
Created board API
```

Both failure modes are reported up front rather than producing a broken config:

```console
$ kira board create docs
kira: board key "docs" must match ^[A-Z][A-Z0-9]{1,9}$
$ kira board create API
kira: board "API" already exists
```

---

## `kira board rename`

Changes a board's display name. The key is immutable — it is baked into every
ticket number on the board — so renaming only touches the human-readable label
shown in `board list` and in the interactive pickers.

```console
$ kira board rename APP "Application"
Renamed board APP to Application
$ kira board list
APP  Application (default)
WEB  Web Frontend
API  Public API
```

An unknown key is rejected, with the valid keys offered as a hint:

```console
$ kira board rename NOPE "X"
kira: no such board "NOPE"
  hint: boards: APP, WEB, API
```

---

## `kira board move`

Moves one item to a different board. This is the one board operation that
changes a ticket's identity: because the board key *is* the number prefix, the
item is renumbered to the next free number on the destination board.

To keep every existing reference working — commit messages, branch names, links
in other tickets, a URL you pasted into chat last week — kira records the old
number as an **alias** on the item. Aliases accumulate, so an item moved twice
still answers to both of its previous numbers. Nothing you have already written
down goes stale.

This command takes no flags beyond the global ones. Moving an item to the board
it is already on is an error, as is naming a board that does not exist or has
been archived.

Here is the whole behaviour end to end. Start with a ticket on `WEB`:

```console
$ kira show WEB-3
WEB-3  Fix session cookie expiry  [TODO]
id:        01KYHX9WFXT90K03ZTE05RBGJ6
type:      ticket
category:  todo
owner:     alice
```

Move it to `API` — note the reported renumbering:

```console
$ kira board move WEB-3 API
Moved WEB-3 -> API-2
```

The item now lists under its new number:

```console
$ kira list
NUMBER  STATE        TYPE    PRIORITY  TITLE
API-1   IN_PROGRESS  ticket  -         Rate-limit /v1/search
APP-1   TODO         ticket  -         Ship 1.0 changelog
WEB-1   IN_PROGRESS  ticket  -         Wire up login form
API-2   TODO         ticket  -         Fix session cookie expiry
WEB-2   REVIEW       ticket  -         Dark mode toggle
```

But the **old number still resolves**, and resolves to the same item — same
ULID, now displayed under its current number:

```console
$ kira show WEB-3
API-2  Fix session cookie expiry  [TODO]
id:        01KYHX9WFXT90K03ZTE05RBGJ6
type:      ticket
category:  todo
owner:     alice
```

The alias lives in the ticket's own frontmatter, so it travels with the file
and is shared with everyone who pulls:

```yaml
id: 01KYHX9WFXT90K03ZTE05RBGJ6
number: API-2
aliases: [WEB-3]
type: ticket
title: "Fix session cookie expiry"
state: TODO
```

Move it again and the alias list grows rather than being replaced — every
number the item has ever had keeps working:

```console
$ kira board move API-2 APP
Moved API-2 -> APP-2
$ kira show WEB-3
APP-2  Fix session cookie expiry  [TODO]
```

```yaml
number: APP-2
aliases: [WEB-3, API-2]
```

Rejected cases:

```console
$ kira board move API-2 API
kira: API-2 is already on board API
$ kira board move APP-1 WEB          # WEB is archived
kira: no such board "WEB"
  hint: boards: APP, API
```

---

## `kira board archive`

Retires a board without deleting anything. Archiving is purely about *new*
work: an archived board disappears from board pickers and stops being a valid
target, while everything already filed under it keeps working exactly as
before.

Concretely, after archiving `WEB`:

- `board list` still shows the board, flagged `(archived)`.
- Its tickets still appear in `kira list` and on `kira board`.
- Its tickets still resolve by number in `show`, `edit`, `move` and everywhere else.
- `kira create --board WEB` is rejected.
- `kira board move <item> WEB` is rejected.

Verified:

```console
$ kira board archive WEB
Archived board WEB
$ kira board list
APP  Application (default)
WEB  Web Frontend (archived)
API  Public API
$ kira list
NUMBER  STATE        TYPE    PRIORITY  TITLE
API-1   IN_PROGRESS  ticket  -         Rate-limit /v1/search
APP-1   TODO         ticket  -         Ship 1.0 changelog
WEB-1   IN_PROGRESS  ticket  -         Wire up login form
API-2   TODO         ticket  -         Fix session cookie expiry
WEB-2   REVIEW       ticket  -         Dark mode toggle
$ kira show WEB-1
WEB-1  Wire up login form  [IN_PROGRESS]
$ kira create ticket --title "New on archived" --no-edit --board WEB
kira: no such board "WEB"
  hint: boards: APP, API
```

Note that the hint lists only the live boards — archived keys are omitted from
the set of things you can file against.

## `kira board unarchive`

Restores an archived board to normal status. It becomes a valid `--board`
target again and reappears in pickers.

```console
$ kira board unarchive WEB
Unarchived board WEB
$ kira board list
APP  Application (default)
WEB  Web Frontend
API  Public API
```

---

## `kira sprint create`

Appends a sprint to the `sprints:` block in config. A sprint is just a keyed
date range; tickets join it by setting their `sprint` field (`kira create
--sprint`, `kira edit --sprint`). Like every config mutation it is committed
immediately under `commit.mode: auto`, so sprints are shared with your team.

All four flags are effectively required — the resulting sprint is validated
against the config schema, and an empty or malformed value fails the whole
command.

| Flag | Argument | Meaning |
| --- | --- | --- |
| `--key` | KEY | Sprint key. This is what item `sprint` fields reference. |
| `--name` | text | Display name. |
| `--start` | YYYY-MM-DD | Start date. |
| `--end` | YYYY-MM-DD | End date. Must be strictly after `--start`. |

```console
$ kira sprint create --key S1 --name "Sprint 1" --start 2026-07-06 --end 2026-07-17
Created sprint S1 (2026-07-06 -> 2026-07-17)
$ kira sprint create --key S2 --name "Sprint 2" --start 2026-07-20 --end 2026-07-31
Created sprint S2 (2026-07-20 -> 2026-07-31)
```

Every omission or malformed value is caught before anything is written:

```console
$ kira sprint create --name "Sprint 1"
kira: config: sprints: sprint with empty key
$ kira sprint create --key S0 --start 2026-06-22 --end 2026-07-03
kira: config: sprints[S0]: empty name
$ kira sprint create --key S1 --name "Sprint 1" --start 2026-07-06
kira: config: sprints[S1].end: invalid date "", want YYYY-MM-DD
$ kira sprint create --key S1 --name "Sprint 1" --start 06-07-2026 --end 2026-07-17
kira: config: sprints[S1].start: invalid date "06-07-2026", want YYYY-MM-DD
$ kira sprint create --key S1 --name "Sprint 1" --start 2026-07-20 --end 2026-07-06
kira: config: sprints[S1]: start 2026-07-20 is not before end 2026-07-06
```

---

## `kira sprint list`

Lists every configured sprint with its date range and completion counts, in
config order. A leading `*` marks the active sprint. The counts are
`done/total` over items whose `sprint` field names that sprint, where "done"
means any state in the `done` category — so both `DONE` and `WONT_DO` count as
finished.

This command takes no flags beyond the global ones.

```console
$ kira sprint list
* S1  Sprint 1  2026-07-06 -> 2026-07-17  1/4 done
  S2  Sprint 2  2026-07-20 -> 2026-07-31  0/0 done
```

With nothing configured yet:

```console
$ kira sprint list
No sprints configured (use `kira sprint create`)
```

---

## `kira sprint activate`

<a id="sprint-activate"></a>

Sets your **active sprint**: the sprint that the shorthand `active` resolves
to, as in `kira list --sprint active` or a query expression like `sprint =
active`. It saves you retyping the current sprint key a hundred times a
fortnight.

The important thing to understand is that this pointer is **local to your
clone and not shared**. It is written to `.kira/.cache/active-sprint`, which
sits under the git-ignored `.kira/.cache/` directory — the same disposable
cache that holds the index. It is never committed, never pushed, and your
teammates cannot see it.

That is deliberate. Which sprint is "active" is a statement about *your current
working set*, not a fact about the project. Two people on the same repo can
legitimately want different answers at the same moment: one grinding through
the sprint that is closing, another already grooming the next one. A committed
"active sprint" flag would make one of them wrong, and would churn the config
history with a commit every time anyone changed focus. The sprints themselves —
their keys, names and dates — *are* shared, because those are facts about the
project; only the cursor into them is personal.

This command takes no flags beyond the global ones.

Here is the locality proved directly. The tree is clean before activating:

```console
$ git status --short
$ kira sprint activate S1
Activated sprint S1
$ git status --short
$
```

Still clean — no config change, no new tracked file, nothing to commit. The
pointer does exist, but git is ignoring it:

```console
$ git status --short --ignored
!! .kira/.cache/active-sprint
!! .kira/.cache/index.db
!! .kira/.cache/lock
!! .kira/.cache/meta.json
```

Once set, `active` works as a sprint name anywhere a sprint is accepted:

```console
$ kira sprint list
* S1  Sprint 1  2026-07-06 -> 2026-07-17  0/4 done
  S2  Sprint 2  2026-07-20 -> 2026-07-31  0/0 done
$ kira list --sprint active
NUMBER  STATE        TYPE    PRIORITY  TITLE
API-1   IN_PROGRESS  ticket  -         Rate-limit /v1/search
WEB-1   IN_PROGRESS  ticket  -         Wire up login form
API-2   TODO         ticket  -         Fix session cookie expiry
WEB-2   REVIEW       ticket  -         Dark mode toggle
$ kira list --query 'sprint = active'
NUMBER  STATE        TYPE    PRIORITY  TITLE
API-1   IN_PROGRESS  ticket  -         Rate-limit /v1/search
WEB-1   IN_PROGRESS  ticket  -         Wire up login form
API-2   TODO         ticket  -         Fix session cookie expiry
WEB-2   REVIEW       ticket  -         Dark mode toggle
```

Activating a different sprint reports what it displaced, and an unknown key is
refused:

```console
$ kira sprint activate S2
Activated sprint S2 (was S1)
$ kira sprint activate nope
kira: "nope" is not a key in the configured sprints
```

Because the pointer lives in the cache directory, a fresh clone starts with no
active sprint. Asking for `active` before setting one is a warning rather than
a hard failure — the filter simply matches nothing:

```console
$ kira list --sprint active
kira: no active sprint set; sprint=active matches nothing (run `kira sprint activate <key>`)
no items
```

---

## `kira sprint close`

Wraps up a sprint. Closing is a **reporting and spillover** operation, not a
deletion: it lists every item in the sprint that has not reached a done-category
state, optionally reassigns those items to a successor sprint, and clears the
active-sprint pointer if the sprint being closed was the active one. The sprint
itself stays in config with its dates intact, so its history remains
inspectable and `sprint list` keeps showing it.

| Flag | Argument | Meaning |
| --- | --- | --- |
| `--move-to` | KEY | Reassign every unfinished item to this sprint. One commit per moved item. |

`--move-to` must name a configured sprint, and cannot be the sprint being
closed:

```console
$ kira sprint close S2 --move-to S2
kira: --move-to target "S2" is the sprint being closed
$ kira sprint close S2 --move-to ZZ
kira: --move-to: "ZZ" is not a key in the configured sprints
```

Closing without `--move-to` is a pure dry look at what did not land:

```console
$ kira sprint close S1
Closed sprint S1
  unfinished: WEB-1
  unfinished: API-1
  unfinished: API-2
```

Adding `--move-to` carries the spillover forward. Each item is moved as its own
commit, so the reassignment shows up as a per-ticket line in history rather
than one opaque bulk change — you can revert, blame or cherry-pick a single
item's spillover:

```console
$ git log --oneline | head -3
1f82ea8 WEB-2 state REVIEW -> DONE
a26bdc4 API-2 edit sprint
c117b39 API-1 edit sprint

$ kira sprint close S1 --move-to S2
Closed sprint S1
  unfinished: WEB-1
  unfinished: API-1
  unfinished: API-2
Moved 3 unfinished item(s) to S2
Cleared the active-sprint pointer

$ git log --oneline | head -8
2fcbf21 API-2 sprint S1 -> S2
a726a25 API-1 sprint S1 -> S2
eeba50a WEB-1 sprint S1 -> S2
1f82ea8 WEB-2 state REVIEW -> DONE
a26bdc4 API-2 edit sprint
c117b39 API-1 edit sprint
1ae7be1 WEB-2 edit sprint
d4d00f6 WEB-1 edit sprint
```

Three unfinished items, three commits, each naming the item and the sprint
transition it performed.

Afterwards the counts settle: `S1` keeps only the work that actually finished
in it, and `S2` has inherited the rest.

```console
$ kira sprint list
  S1  Sprint 1  2026-07-06 -> 2026-07-17  1/1 done
  S2  Sprint 2  2026-07-20 -> 2026-07-31  0/3 done
```

Note that the `*` is gone: `S1` had been the active sprint, and closing it
cleared the local pointer, as the `Cleared the active-sprint pointer` line
reported. `kira sprint activate S2` is the natural next step.

Closing does not change any item's *state* — an unfinished ticket stays
`TODO` or `IN_PROGRESS`, it just belongs to a different time box now. If you
want to abandon spillover work rather than carry it, move those items to
`WONT_DO` before closing.
