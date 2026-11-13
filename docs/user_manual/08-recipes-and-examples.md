# Recipes and Examples

Short, working answers to "how do I do X". Every command sequence on this page
was run against a real repository — the output is what you will see.

Part 1 covers the things you will do every day. Part 2 surfaces options that
are genuinely useful but easy to never discover.

This page trades depth for breadth. Where a recipe only shows one flag of many,
it links to the reference page that documents the rest.

---

## Part 1 — Everyday workflows

### Take a ticket from idea to done

```console
$ kira create ticket --title 'Fix flaky retry backoff' --no-edit
Created CORE-1 Fix flaky retry backoff

$ kira workon CORE-1
Created branch core/core-1-fix-flaky-retry-backoff for CORE-1
Moved CORE-1: TODO -> IN_PROGRESS

... write the code, commit as usual ...

$ kira move CORE-1 REVIEW
Moved CORE-1: IN_PROGRESS -> REVIEW

$ kira move CORE-1 DONE --resolution done
Moved CORE-1: REVIEW -> DONE
```

`kira workon` does three things at once: creates (or switches to) a per-ticket
branch, records it as your active ticket, and advances the state to the first
`doing` state. It is idempotent — running it again on the same ticket just
switches back to the branch.

Because the state changes happened on the branch, they reach the main line when
the branch does. Omit `--no-edit` to open the new ticket in `$EDITOR` first.

### See what to work on next

```console
$ kira list --filter next
NUMBER  STATE  TYPE    PRIORITY  TITLE
CORE-5  TODO   ticket  P0        Bump protocol version to v3
CORE-3  TODO   ticket  P1        Drain in-flight orders on shutdown
CORE-6  TODO   ticket  -         Audit log retention policy
```

`next` is a saved filter that ships in every new repository; it expands to
`state = TODO AND NOT blocked ORDER BY priority`. Saved filters live under
`filters:` in `.kira/config.yaml`, and `kira config filters` lists the ones your
repository defines.

Narrow it to yourself with `--owner @me`, which resolves to your git user:

```console
$ kira list --owner @me --state TODO
NUMBER  STATE  TYPE    PRIORITY  TITLE
CORE-5  TODO   ticket  P0        Bump protocol version to v3
CORE-3  TODO   ticket  P1        Drain in-flight orders on shutdown
```

Flags AND together, and `--query` ANDs on top of them, so
`kira list --filter next --owner @me --label perf` is a valid narrowing. The
query grammar — `IN`, `IS EMPTY`, ranked comparisons, `ORDER BY` — is covered in
[views and queries](04-cli-reference/views-and-queries.md).

### Capture a follow-up without losing your place

```console
$ kira create ticket --here --no-edit --title 'Benchmark the new cache path'
Created CORE-7 Benchmark the new cache path

$ kira create ticket --here --no-edit --title 'Pick a cache eviction policy' --blocking
Created CORE-8 Pick a cache eviction policy
```

`--here` inherits the active ticket's epic and sprint and stamps a `captured`
label, so a thought you had mid-task lands in the right place with one command.
Adding `--blocking` also records the new item as a blocker of the ticket you are
on. It requires an active ticket (`kira workon`), plus `--no-edit` and
`--title`, so it never interrupts you with an editor.

### Block work, then unblock it

```console
$ kira link CORE-4 --blocked-by CORE-5
Linked CORE-4

$ kira list --query blocked
NUMBER  STATE  TYPE    PRIORITY  TITLE
CORE-4  TODO   ticket  P2        Replace legacy session cache
```

The dependency is stored only on the dependent item — the blocker's own file
records nothing. To ask the reverse question, "what is waiting on this?", query
for it:

```console
$ kira list --query 'blocked_by = CORE-5'
NUMBER  STATE  TYPE    PRIORITY  TITLE
CORE-4  TODO   ticket  P2        Replace legacy session cache
```

`kira show <blocker> --json` answers the same question in its `blocks` array,
derived the same way (the human-readable `kira show` does not print it):

```console
$ kira show CORE-5 --json | jq .blocks
[
  "01KYKD7DK3A2VGB40VAF3715BZ"
]
```

`blocked` is computed, not a field: an item is blocked while any of its
blockers is still open. Closing the blocker is what unblocks it — no bookkeeping
step. To retract the dependency itself, use
`kira link CORE-4 --blocked-by CORE-5 --remove`.

### Group tickets under an epic

```console
$ kira create epic --title 'Order gateway rewrite' --no-edit
Created CORE-2 Order gateway rewrite

$ kira create ticket --title 'Drain in-flight orders on shutdown' --no-edit --parent CORE-2
Created CORE-3 Drain in-flight orders on shutdown

$ kira tree
CORE-1    ticket  Fix flaky retry backoff
CORE-2    epic    Order gateway rewrite
  CORE-3  ticket  Drain in-flight orders on shutdown
  CORE-4  ticket  Replace legacy session cache
  CORE-5  ticket  Bump protocol version to v3
CORE-6    ticket  Audit log retention policy
```

`kira tree CORE-2` scopes the render to one epic's subtree. To reparent an
existing ticket, use `kira link CORE-6 --parent CORE-2`. Epics cannot be nested
inside other epics — the hierarchy is deliberately one level deep. See
[item lifecycle](04-cli-reference/item-lifecycle.md) for the rest of `create`,
`link`, and `move`.

### Run a sprint, including the spillover

```console
$ kira sprint create --key 2026-S30 --name 'Sprint 30' --start 2026-07-27 --end 2026-08-09
Created sprint 2026-S30 (2026-07-27 -> 2026-08-09)

$ kira sprint activate 2026-S30
Activated sprint 2026-S30

$ kira edit CORE-3 CORE-5 CORE-6 --field sprint=2026-S30
Edited CORE-3: sprint
Edited CORE-5: sprint
Edited CORE-6: sprint

$ kira sprint list
* 2026-S30  Sprint 30  2026-07-27 -> 2026-08-09  0/3 done
  2026-S31  Sprint 31  2026-08-10 -> 2026-08-23  0/0 done
```

Work the sprint, then close it and push whatever did not finish into the next
one:

```console
$ kira sprint close 2026-S30 --move-to 2026-S31
Closed sprint 2026-S30
  unfinished: CORE-6
Moved 1 unfinished item(s) to 2026-S31
Cleared the active-sprint pointer
```

Once a sprint is active, `--sprint active` works everywhere a sprint key does —
`kira list --sprint active --owner @me` is your board for the week. Dropping
`--move-to` closes the sprint and reports the spillover without reassigning it.
See [boards and sprints](04-cli-reference/boards-and-sprints.md) for the full
command set.

### Review what a branch does to the backlog

```console
$ kira diff --incoming feature/gateway
CORE-1  Cache warmup on boot
  state: TODO -> IN_PROGRESS
CORE-2  Retire v1 endpoint
  priority: - -> P1
created CORE-3  Add gateway metrics
```

This is the ticket-level equivalent of reading a pull request: it reports what
the branch changes relative to the merge base, in fields rather than diff
hunks. Drop `--incoming` to see the mirror image — *your* changes versus the
merge base — which is the useful direction while you are on the branch.

To review a window of history instead of a branch, `kira changes --since` takes
a git ref or a date and reports every field event across the whole backlog:

```console
$ kira changes --since 2026-07-01
CORE-6  Audit log retention policy
  2026-07-27T19:21:36+05:30  sprint: 2026-S30 -> 2026-S31
  2026-07-27T19:21:29+05:30  sprint:  -> 2026-S30
```

### Resolve a merge conflict on a ticket file

```console
$ git merge teammate
Auto-merging .kira/tickets/01KYHXJKTR99JC2KMANSKD6PKW.md
CONFLICT (content): Merge conflict in .kira/tickets/01KYHXJKTR99JC2KMANSKD6PKW.md
Automatic merge failed; fix conflicts and then commit the result.

$ kira resolve
Resolved CORE-1 (auto-merged: priority)

$ git commit
```

`kira resolve` re-merges the conflicted ticket files field by field and stages
the results, so you never hand-edit YAML conflict markers. Edits to *different*
fields union; edits to the *same* field take the later one.

Better still, prevent the conflict:

```console
$ kira hooks install
hook post-merge: installed
hook prepare-commit-msg: installed
merge driver: registered
```

With the merge driver registered, git resolves ticket files field by field
during the merge itself and never produces markers. `kira resolve` is the
cleanup path for merges that ran without it — a teammate's clone, or a merge
made before you installed the hooks. See
[git integration](04-cli-reference/git-integration.md).

### Check throughput and cycle time

```console
$ kira stats
Stats (project)
  completion: 3/6 done (50%)
  cycle time (days): p50 0  p90 0  n=3
  lead time (days): p50 0  p90 0  n=3
  throughput/week: 0 0 0 0 0 0 0 3
```

Every number is derived from commit dates in git history, so there is nothing to
maintain and nothing that can drift from reality. Cycle time measures first
`doing` state to done; lead time measures creation to done. Scope to one sprint
with `kira stats --sprint 2026-S30`, or to the running one with
`--sprint active`.

---

## Part 2 — Niche and power-user options

### See the tracker as it was on a past date

```console
$ kira list --at 2026-07-01
$ kira list --at HEAD~40
$ kira show CORE-3 --at v1.4.0
$ kira tree --at 2026-07-01
$ kira board --plain --at HEAD~40
```

`--at` takes a git ref or a `YYYY-MM-DD` date and reads the whole backlog as it
stood at that point — useful for "what did we commit to at the start of the
quarter?" and for sprint retrospectives. Read commands accept it; mutating
commands reject it outright, so there is no way to accidentally write to the
past.

### Preview the editor template before you commit to flags

```console
$ kira create ticket --title 'Preview me' --priority P1 --label perf --print-template
---
title: Preview me
type: ticket
subtype:
priority: P1
rank:
owner:
reporter:
labels: [perf]
epic:
sprint:
due:
estimate:
---

## Description

## Acceptance criteria
```

`--print-template` prints exactly what the editor would open, with your flags
already filled in, and creates nothing. It is the fastest way to see which
fields a type offers and how a `--field` value will land. It also resolves
per-subtype templates, so `kira create ticket --subtype bug --print-template`
shows you which of `.kira/templates/ticket.bug.md` or `ticket.md` will win.

### Generate a branch name or a link for scripting

```console
$ kira show CORE-4 --format branch
core-4-replace-legacy-session-cache

$ kira show CORE-4 --format markdown
[CORE-4 Replace legacy session cache](CORE-4)

$ kira show CORE-4 --format number-title
CORE-4 Replace legacy session cache
```

`--format` prints one bare reference form with no decoration, which makes it
safe to interpolate: `git checkout -b "$(kira show CORE-4 --format branch)"`.
The available forms are `id`, `number`, `number-title`, `markdown`, and
`branch`.

### Build a custom report with `--json` and `jq`

```console
$ kira list --json | jq -r '.items | group_by(.state)[] | "\(.[0].state): \(length)"'
DONE: 3
PLANNED: 1
TODO: 2

$ kira list --json | jq -r '.items[] | select(.category != "done") | "\(.number)\t\(.priority // "-")\t\(.title)"'
CORE-4	P2	Replace legacy session cache
CORE-2	-	Order gateway rewrite
CORE-6	-	Audit log retention policy
```

Every command accepts `--json`, and the stream discipline is strict: JSON goes
to stdout, warnings and errors go to stderr, and stdout stays empty when the
command fails. A `jq` pipeline therefore never has to filter noise out of its
input. `kira schema` prints the JSON Schema for these shapes.

### Full-text search across ticket bodies

```console
$ kira find 'session cache'
CORE-4:6:title: "Replace legacy session cache"

$ kira find -i 'RETRY'
CORE-1:6:title: "Fix flaky retry backoff"
```

`kira find` is a ripgrep wrapper scoped to your ticket files, reporting hits by
ticket number and line instead of by path. Any flags after the pattern are
passed straight through to ripgrep, so `-i`, `-C 2`, and `-w` all work. Use it
when you remember a phrase from the description but not the number — `kira list
--query` only searches titles.

### Run any CLI command from inside the TUI

Press `:` in the TUI to open a command bar that accepts any kira command line,
and `.` inside it expands to the item you have selected:

```
:move . REVIEW
:label add . perf
:assign . @me
```

The screen refreshes with the result, so you never leave the UI to make a
change. `/` opens the filter bar instead. See the
[TUI guide](07-tui-guide.md) for the full key map.

### Move an item to another board without breaking old links

```console
$ kira board move CORE-6 INFRA
Moved CORE-6 -> INFRA-1

$ kira show INFRA-1 --format number-title
INFRA-1 Audit log retention policy

$ kira show CORE-6 --format number-title
INFRA-1 Audit log retention policy
```

The item is renumbered into the target board's own sequence, and the old number
is kept as a permanent alias. Every commit message, code comment, and chat
message that mentions `CORE-6` keeps resolving — forever. Boards each carry an
independent numbering sequence; see
[boards and sprints](04-cli-reference/boards-and-sprints.md).

### Catch malformed tickets in CI or a pre-commit hook

```console
$ kira validate .kira/tickets/*.md
ok: 0 errors, 0 warnings, 0 info
see `kira config explain` for the rules these checks enforce
```

Hand-edit a ticket into an invalid state and the exit code turns non-zero:

```console
$ kira validate .kira/tickets/01KYHXXFYP9VE2S3QZC6C1MWKG.md
error   state           CORE-4 state: "SHIPPED" is not a state in the ticket workflow
problems found: 1 error, 0 warnings, 0 info
see `kira config explain` for the rules these checks enforce
kira: validate: 1 problem found

$ echo $?
1
```

`kira validate` checks files against kira's item shape and states and
exits non-zero on any error, which is all a CI step needs:

```yaml
- run: kira validate .kira/tickets/*.md
```

It accepts ticket numbers and ULID prefixes as well as paths, so a pre-commit
hook can validate only the files being committed. Use it for hand-edited
tickets — the ones a text editor or a merge produced rather than the CLI.
For whole-repository consistency (duplicate numbers, dangling references) run
`kira doctor` instead; both are documented in
[setup and schema commands](04-cli-reference/setup-and-schema-commands.md).

### Ask why kira refused something

```console
$ kira config explain
identity (repo)
  project: CORE (demo)
  id style: sequential
  board: CORE (demo) [default]
  board: INFRA (Infrastructure)

vocab (default)
  ...
```

`kira config explain` renders your repository's effective rules in prose, with
the provenance of each one — so when a value is rejected as outside a strict vocabulary, this tells you the rule that stopped it and whether it came from
the repository config, your user config, or the built-in default.
It is the fastest answer to "why did that not work".

---

## See also

- [Overview](01-overview.md) — what kira is and why tickets live in git.
- [Concepts](03-concepts.md) — identity, states, vocabularies, relationships.
- [Item lifecycle](04-cli-reference/item-lifecycle.md) — `create`, `edit`, `move`, `link`.
- [Views and queries](04-cli-reference/views-and-queries.md) — the query grammar, `list`, `tree`, `board`.
- [Boards and sprints](04-cli-reference/boards-and-sprints.md) — numbering namespaces and sprint commands.
- [Git integration](04-cli-reference/git-integration.md) — hooks, merge driver, `sync`, `resolve`.
- [Setup and schema commands](04-cli-reference/setup-and-schema-commands.md) — `init`, `validate`, `doctor`, `schema`.
- [Configuration reference](05-configuration-reference.md) — every key in `.kira/config.yaml`.
- [TUI guide](07-tui-guide.md) — the interactive interface.
