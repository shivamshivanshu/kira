# Views and Queries

Every command on this page is read-only: none of them writes a ticket file or
creates a commit. They are the commands you reach for when you want to know
what the backlog looks like — now, at some point in the past, or on another
branch.

The global flags `-C <path>` (run as if invoked from another directory),
`--json` (machine-readable output on stdout) and `--no-color` (drop ANSI
color) work on all of them and are documented once in the
[global flags](setup-and-schema-commands.md#global-flags) section. One exception is called out
below: `kira find` deliberately does not take `-C`.

All examples on this page come from a demo repository with project key `PAY`:
two epics, seven tickets, two sprints, and a mixture of states, priorities and
labels.

---

## `kira list`

The workhorse. `list` prints a flat table of the items that match the filters
you give it; with no filters it prints everything.

Results are ordered by *precedence*: items carrying an explicit `rank` come
first, sorted lexicographically by that rank; the rest follow, sorted by
priority (`P0` sorts first, then `P1`, `P2`, `P3`), and ties broken by item number. An explicit
`ORDER BY` in a query expression overrides that, with precedence retained only
as the tie-break.

| Flag | Value | Effect |
| --- | --- | --- |
| `--at` | git ref or `YYYY-MM-DD` | Read the item set as it stood at that point, anchored on `HEAD`. |
| `--category` | `todo`\|`doing`\|`done` | Filter by workflow category rather than a specific state. |
| `--filter` | filter name | Run a named saved query from `filters:` in `.kira/config.yaml`. |
| `--label` | label | Only items carrying that label. |
| `--owner` | person or `@me` | Filter by owner. `@me` resolves to your git identity. |
| `--parent` | item reference | Only items whose parent epic is that item. |
| `--priority` | priority | Only items at that priority. |
| `--query` | query expression | Filter by an expression (see [Query expressions](#query-expressions)). |
| `--sprint` | sprint key or `active` | Filter by sprint. `active` resolves your locally activated sprint. |
| `--state` | state key | Only items in that exact state. |
| `--subtype` | subtype | Only items of that subtype (`bug`, `task`, …). |
| `--tree` | — | Group the results under their epics instead of printing a flat table. |
| `--type` | `ticket`\|`epic` | Restrict to one item type. |

Every flag is ANDed with every other flag, and with `--query`. There is no
way to express OR across flags — use a query expression for that.

### Plain listing

```
$ kira list
NUMBER  STATE        TYPE    PRIORITY  TITLE
PAY-3   IN_PROGRESS  ticket  P0        Tokenize card numbers at ingest
PAY-5   REVIEW       ticket  P0        Settlement report off by one day
PAY-6   IN_PROGRESS  ticket  P0        Batch scoring is 4x slower than target
PAY-1   ACTIVE       epic    P1        Card payment rails
PAY-4   TODO         ticket  P1        Retry failed authorizations with backoff
PAY-2   ACTIVE       epic    P2        Fraud detection
PAY-8   DONE         ticket  P2        Drop the legacy /v1/charge endpoint
PAY-7   TODO         ticket  P3        Spike: graph features for chargeback risk
```

Epics and tickets are interleaved, because both are items and both carry a
priority. `--type ticket` separates them.

### Filtering by flag

```
$ kira list --category doing --priority P0
NUMBER  STATE        TYPE    PRIORITY  TITLE
PAY-3   IN_PROGRESS  ticket  P0        Tokenize card numbers at ingest
PAY-5   REVIEW       ticket  P0        Settlement report off by one day
PAY-6   IN_PROGRESS  ticket  P0        Batch scoring is 4x slower than target
```

`--category doing` matches every `doing` state (`IN_PROGRESS` and `REVIEW`),
so you do not have to enumerate the in-flight columns.

`--owner @me` resolves your git `user.name`/`user.email` to a person in the
`people:` roster, so the same command works for everyone on the team:

```
$ kira list --owner @me
NUMBER  STATE        TYPE    PRIORITY  TITLE
PAY-3   IN_PROGRESS  ticket  P0        Tokenize card numbers at ingest
PAY-1   ACTIVE       epic    P1        Card payment rails
PAY-2   ACTIVE       epic    P2        Fraud detection
PAY-8   DONE         ticket  P2        Drop the legacy /v1/charge endpoint
```

If your git identity is not mapped in `people:`, kira falls back to your git
name with spaces replaced by hyphens; if neither name nor email is set, `@me`
is an error telling you to configure git.

`--sprint active` uses the sprint you selected with `kira sprint activate`.
That pointer is local to your clone and is not committed, so `--sprint active`
means something different for each teammate — that is the point.

```
$ kira list --sprint active
NUMBER  STATE        TYPE    PRIORITY  TITLE
PAY-3   IN_PROGRESS  ticket  P0        Tokenize card numbers at ingest
PAY-6   IN_PROGRESS  ticket  P0        Batch scoring is 4x slower than target
PAY-4   TODO         ticket  P1        Retry failed authorizations with backoff
```

### Grouping by epic — `--tree`

```
$ kira list --tree
PAY-1                             (epic)
  PAY-3  IN_PROGRESS  ticket  P0  Tokenize card numbers at ingest
  PAY-5  REVIEW       ticket  P0  Settlement report off by one day
  PAY-4  TODO         ticket  P1  Retry failed authorizations with backoff
PAY-2                             (epic)
  PAY-6  IN_PROGRESS  ticket  P0  Batch scoring is 4x slower than target
  PAY-7  TODO         ticket  P3  Spike: graph features for chargeback risk
(no epic)
  PAY-8  DONE  ticket  P2  Drop the legacy /v1/charge endpoint
```

Unparented items are collected under `(no epic)`. `--tree` reshapes whatever
the other filters selected, so `kira list --state TODO --tree` shows only the
epics that still have TODO children. This is a filtered view; `kira tree`
below is the unfiltered structural view.

### Named filters — `--filter`

Saved queries live under `filters:` in `.kira/config.yaml`, keyed by name:

```yaml
filters:
  next: state = TODO AND NOT blocked ORDER BY priority
  hot: priority IN (P0, P1) AND category != done ORDER BY priority
  stale-review: state = REVIEW AND updated < 2026-07-01
```

They are committed with the rest of the config, so the whole team shares the
same vocabulary for "what should I pick up next".

```
$ kira list --filter next
NUMBER  STATE  TYPE    PRIORITY  TITLE
PAY-7   TODO   ticket  P3        Spike: graph features for chargeback risk
```

Only one TODO ticket comes back, because `PAY-4` is blocked by `PAY-3` and the
`next` filter excludes blocked work.

```
$ kira list --filter hot
NUMBER  STATE        TYPE    PRIORITY  TITLE
PAY-3   IN_PROGRESS  ticket  P0        Tokenize card numbers at ingest
PAY-5   REVIEW       ticket  P0        Settlement report off by one day
PAY-6   IN_PROGRESS  ticket  P0        Batch scoring is 4x slower than target
PAY-1   ACTIVE       epic    P1        Card payment rails
PAY-4   TODO         ticket  P1        Retry failed authorizations with backoff
```

`--filter` composes with the other flags, so `kira list --filter hot --owner @me`
narrows a shared filter to your own work without editing the config.

### Historical views — `--at`

`--at` takes a git ref or a `YYYY-MM-DD` date and reconstructs the item set
from that tree. It answers "what did the board look like then", and it works
across branches as well as backwards in time:

```
$ kira list --at fraud-scoring
NUMBER  STATE        TYPE    PRIORITY  TITLE
PAY-3   IN_PROGRESS  ticket  P0        Tokenize card numbers at ingest
PAY-5   REVIEW       ticket  P0        Settlement report off by one day
PAY-1   ACTIVE       epic    P1        Card payment rails
PAY-4   TODO         ticket  P1        Retry failed authorizations with backoff
PAY-6   REVIEW       ticket  P1        Batch scoring is 4x slower than target
PAY-2   ACTIVE       epic    P2        Fraud detection
PAY-8   DONE         ticket  P2        Drop the legacy /v1/charge endpoint
PAY-9   TODO         ticket  P2        Cache feature vectors between scoring runs
PAY-7   TODO         ticket  P3        Spike: graph features for chargeback risk
```

A date is resolved to the last commit on `HEAD` at or before **midnight** of
that day, so passing today's date usually fails — none of today's commits are
older than this morning's `00:00`:

```
$ kira list --at 2026-07-27
kira: resolving --at 2026-07-27: no commit on HEAD at or before 2026-07-27T00:00:00
```

Use yesterday's date, or a ref like `HEAD~10`, when you want "recently".

### JSON output

```
$ kira list --state REVIEW --json
{
  "items": [
    {
      "id": "01KYHXAZGRXX2RQ5YFY5D65ETV",
      "number": "PAY-5",
      "board": "PAY",
      "title": "Settlement report off by one day",
      "type": "ticket",
      "state": "REVIEW",
      "category": "doing",
      "owner": "grace",
      "labels": [
        "payments"
      ],
      "epic": "01KYHXAGRMM0WBZX58AR8HKR9K",
      "epic_number": "PAY-1",
      "priority": "P0",
      "resolution": null,
      "due": "2026-07-30"
    }
  ],
  "count": 1
}
```

Note the two identity fields: `id` is the item's permanent identifier, stable
across renumbering and merges, while `number` is the human-facing display
number and can change when a merge reconciles a collision. Script against
`id`; show `number` to people. `epic` carries the parent's `id` and
`epic_number` its display number, for the same reason.

---

## Query expressions

`--query` (and every saved `filters:` entry) accepts a small expression
language evaluated against each item.

**Structure.** Predicates combine with `AND`, `OR` and `NOT`, grouped with
parentheses. Keywords are case-insensitive. Adjacent predicates with no
connective are implicitly ANDed, so `state = TODO owner = ada` means the same
as `state = TODO AND owner = ada`. An `ORDER BY <field> [asc|desc]` clause may
appear once, and must be the last thing in the expression.

**Fields.**

| Field | Kind | Notes |
| --- | --- | --- |
| `state`, `category`, `type`, `subtype`, `resolution`, `board` | text | `board` matching is case-insensitive. |
| `owner`, `reporter` | text | Accept `@me`. |
| `priority`, `rank` | ordered / text | `priority` compares by priority order (`P0` highest), so `priority < P2` means "more urgent than P2". |
| `epic`, `sprint` | reference | `sprint = active` resolves your active sprint. |
| `label`, `blocked_by`, `links` | list | Equality means "contains"; cannot be used with `ORDER BY`. |
| `estimate` | number | Supports `<`, `<=`, `>`, `>=`. |
| `created`, `updated`, `activity`, `due` | date | Supports ordered comparison. |
| `blocked` | boolean | Write it bare: `blocked`, or `NOT blocked`. |

**Operators.** `=` and `!=` work on every field. The ordered comparisons
`<`, `<=`, `>`, `>=` are accepted only on the date fields, `estimate`, and
`priority`; using them elsewhere is a parse error rather than a silent
string comparison.

**Set membership.** `field IN (a, b, c)`.

**Presence.** `field IS EMPTY` and `field IS NOT EMPTY`. Fields that can never
be absent — `state`, `type`, `category`, `created`, `updated`, `activity`,
`board` — reject this and tell you so.

**Bare terms.** A word or quoted string that is not a field predicate is a
case-insensitive substring match against the item **title** — not the body.
Use `kira find` when you need to search body text.

**Dates.** `created`, `updated` and `activity` compare by calendar day and
require `YYYY-MM-DD`. `due` additionally accepts a full RFC 3339 timestamp.

**Strings.** Double-quote any value containing spaces or punctuation;
`\"` and `\\` are the two escapes.

### Worked examples

```
$ kira list --query 'label = payments AND state != DONE ORDER BY priority'
NUMBER  STATE        TYPE    PRIORITY  TITLE
PAY-3   IN_PROGRESS  ticket  P0        Tokenize card numbers at ingest
PAY-5   REVIEW       ticket  P0        Settlement report off by one day
PAY-1   ACTIVE       epic    P1        Card payment rails
PAY-4   TODO         ticket  P1        Retry failed authorizations with backoff
```

```
$ kira list --query 'priority IN (P0, P1) AND owner IS NOT EMPTY ORDER BY updated desc'
NUMBER  STATE        TYPE    PRIORITY  TITLE
PAY-3   IN_PROGRESS  ticket  P0        Tokenize card numbers at ingest
PAY-5   REVIEW       ticket  P0        Settlement report off by one day
PAY-6   IN_PROGRESS  ticket  P0        Batch scoring is 4x slower than target
PAY-1   ACTIVE       epic    P1        Card payment rails
PAY-4   TODO         ticket  P1        Retry failed authorizations with backoff
```

Boolean and numeric fields:

```
$ kira list --query 'blocked'
NUMBER  STATE  TYPE    PRIORITY  TITLE
PAY-4   TODO   ticket  P1        Retry failed authorizations with backoff

$ kira list --query 'NOT blocked AND estimate > 2 ORDER BY estimate desc'
NUMBER  STATE        TYPE    PRIORITY  TITLE
PAY-6   IN_PROGRESS  ticket  P0        Batch scoring is 4x slower than target
PAY-3   IN_PROGRESS  ticket  P0        Tokenize card numbers at ingest
```

A bare term, matching only the title:

```
$ kira list --query 'scoring'
NUMBER  STATE        TYPE    PRIORITY  TITLE
PAY-6   IN_PROGRESS  ticket  P0        Batch scoring is 4x slower than target
```

Presence checks combined with `OR`:

```
$ kira list --query 'due IS NOT EMPTY OR subtype = spike'
NUMBER  STATE   TYPE    PRIORITY  TITLE
PAY-5   REVIEW  ticket  P0        Settlement report off by one day
PAY-7   TODO    ticket  P3        Spike: graph features for chargeback risk
```

Errors name the offending position and list what was valid there, so typos
fail loudly instead of silently matching nothing:

```
$ kira list --query 'asignee = ada'
kira: query: unknown field "asignee" (available: activity, blocked, blocked_by,
board, category, created, due, epic, estimate, label, links, owner, priority,
rank, reporter, resolution, sprint, state, subtype, type, updated) at position 0
  hint: check the query expression syntax
```

Because flags and `--query` are ANDed, mixing them is often the shortest
route:

```
$ kira list --label perf --query 'category != done'
NUMBER  STATE        TYPE    PRIORITY  TITLE
PAY-6   IN_PROGRESS  ticket  P0        Batch scoring is 4x slower than target
```

---

## `kira show`

Prints one item in full: its header line, the frontmatter fields that are set,
and its Markdown body. With `--format` it instead prints a single
short rendering of the reference, which is what you want in scripts and
shell substitutions.

| Flag | Value | Effect |
| --- | --- | --- |
| `--at` | git ref or `YYYY-MM-DD` | Show the item as it stood at that point in history. |
| `--format` | `id`\|`number`\|`number-title`\|`markdown`\|`branch` | Print one reference form instead of the full item. |

`--format` and `--json` are mutually exclusive.

```
$ kira show PAY-3
PAY-3  Tokenize card numbers at ingest  [IN_PROGRESS]
id:        01KYHXAZ71KAVZ6MQJQ7P7DJF5
type:      ticket
subtype:   story
category:  doing
priority:  P0
owner:     ada
labels:    payments, security
epic:      01KYHXAGRMM0WBZX58AR8HKR9K
created:   2026-07-27T19:18:25+05:30
updated:   2026-07-27T19:18:42+05:30


## Description

## Acceptance criteria
```

Empty fields are omitted rather than printed blank, so the height of the
output tells you how much of the item is actually filled in.

### Reference forms

All five forms, run against the same ticket:

```
$ kira show PAY-3 --format id
01KYHXAZ71KAVZ6MQJQ7P7DJF5

$ kira show PAY-3 --format number
PAY-3

$ kira show PAY-3 --format number-title
PAY-3 Tokenize card numbers at ingest

$ kira show PAY-3 --format markdown
[PAY-3 Tokenize card numbers at ingest](PAY-3)

$ kira show PAY-3 --format branch
pay-3-tokenize-card-numbers-at-ingest
```

Each has an obvious use: `id` for anything that must survive renumbering,
`number` for a commit trailer, `number-title` for a commit subject,
`markdown` for a pull-request description, and `branch` for a branch name.
An unrecognised form is rejected with the list of valid ones:

```
$ kira show PAY-3 --format slug
kira: unknown format "slug", want one of id, number, number-title, markdown, branch
```

### Reading history

`--at` accepts the same refs and dates as `kira list --at`, and shows the item
as that tree recorded it:

```
$ kira show PAY-5 --at HEAD~6
PAY-5  Settlement report off by one day  [REVIEW]
id:        01KYHXAZGRXX2RQ5YFY5D65ETV
...
```

If the display number you asked for pointed at a *different* item back then —
which can happen after a renumbering — kira prints a warning on stderr saying
so, while still showing you the item that number resolved to at that ref.
Standard output stays clean, so piping is unaffected.

---

## `kira tree`

Renders the epic hierarchy: every epic with its children indented beneath it,
and unparented items at the top level. Unlike `kira list --tree` it applies no
filters — it is the structural map of the backlog, so it always shows every
item.

| Flag | Value | Effect |
| --- | --- | --- |
| `--at` | git ref or `YYYY-MM-DD` | Render the hierarchy as it stood at that point. |

```
$ kira tree
PAY-1    epic    Card payment rails
  PAY-3  ticket  Tokenize card numbers at ingest
  PAY-4  ticket  Retry failed authorizations with backoff
  PAY-5  ticket  Settlement report off by one day
PAY-2    epic    Fraud detection
  PAY-6  ticket  Batch scoring is 4x slower than target
  PAY-7  ticket  Spike: graph features for chargeback risk
PAY-8    ticket  Drop the legacy /v1/charge endpoint
```

Children are ordered by number, not by priority — this view is about shape,
not about what to work on next.

Pass an item to scope the tree to one subtree:

```
$ kira tree PAY-2
PAY-2    epic    Fraud detection
  PAY-6  ticket  Batch scoring is 4x slower than target
  PAY-7  ticket  Spike: graph features for chargeback risk
```

---

## `kira find`

Full-text search across the ticket files themselves — body prose, frontmatter
values, everything on disk. It is a wrapper around
[ripgrep](https://github.com/BurntSushi/ripgrep), so anything you already know
about `rg` applies, and unrecognised arguments are handed straight through.
The output is rewritten to show display numbers instead of file paths, which
is the whole reason to use it rather than running `rg` yourself.

`find` takes no flags of its own beyond `--help`. Note that **`-C` is not the
global "change directory" flag here** — inside `find` it is ripgrep's
context-lines flag, and is passed through as such.

```
$ kira find backoff
PAY-4:7:title: "Retry failed authorizations with backoff"
```

The format is `NUMBER:LINE:TEXT`. Ripgrep flags compose as usual:

```
$ kira find -i CHARGEBACK
PAY-7:7:title: "Spike: graph features for chargeback risk"
```

```
$ kira find -i tokenize -C 1
PAY-3-6-subtype: story
PAY-3:7:title: "Tokenize card numbers at ingest"
PAY-3-8-state: IN_PROGRESS
```

Context lines use `-` as the separator instead of `:`, matching ripgrep's own
convention, so you can still tell matches from context.

One caveat: flags that change ripgrep's output shape entirely, such as `-l`
(list matching files), bypass the number rewriting and give you raw paths:

```
$ kira find -l payments
.kira/tickets/01KYHXAGRMM0WBZX58AR8HKR9K.md
.kira/tickets/01KYHXAZGRXX2RQ5YFY5D65ETV.md
.kira/tickets/01KYHXAZ71KAVZ6MQJQ7P7DJF5.md
.kira/tickets/01KYHXAZBXX4YYPN5GABEF32D4.md
```

Those filenames are the items' permanent identifiers, so `kira show <id>`
turns any of them back into a readable ticket.

`--json` emits structured matches:

```
$ kira find backoff --json
{
  "matches": [
    {
      "id": "01KYHXAZBXX4YYPN5GABEF32D4",
      "number": "PAY-4",
      "line": 7,
      "text": "title: \"Retry failed authorizations with backoff\""
    }
  ]
}
```

---

## `kira log`

The history of one item: every field change kira recorded, interleaved in
reverse-chronological order with the commits that touched or referenced it.
This is the "how did this ticket get here" view.

`log` takes no flags of its own.

```
$ kira log PAY-3
2026-07-27T19:19:01+05:30  commit 9a8f7da PAY-3 memoize tokenizer lookups (Ada Byron)
2026-07-27T19:19:01+05:30  commit 73bcba5 Add PAN tokenizer (Ada Byron)
2026-07-27T19:18:42+05:30  state: TODO -> IN_PROGRESS
2026-07-27T19:18:42+05:30  commit c1a10bc PAY-3 state TODO -> IN_PROGRESS (Ada Byron)
2026-07-27T19:18:41+05:30  commit 5d7f617 PAY-4 link blocked-by PAY-3 (Ada Byron)
2026-07-27T19:18:25+05:30  commit c109cf6 PAY-3 create "Tokenize card numbers at ingest" (Ada Byron)
```

Two kinds of entry are interleaved here. Lines like `state: TODO ->
IN_PROGRESS` are *field events* — what actually changed on the item. Lines
beginning `commit` are the commits linked to it, whether that link came from a
`Kira-Ticket:` trailer (`73bcba5`), a bare number leading the subject
(`9a8f7da`), or a mention picked up as a weak reference — which is why
`5d7f617`, a commit about `PAY-4`, appears in `PAY-3`'s log: it names `PAY-3`
as a blocker. Which markers count as links is configurable; see
[git integration](./git-integration.md).

Note that two field changes made in the same edit appear as separate events:

```
$ kira log PAY-6
2026-07-27T19:18:42+05:30  priority: P1 -> P0
2026-07-27T19:18:42+05:30  state: TODO -> IN_PROGRESS
2026-07-27T19:18:42+05:30  commit a33484b PAY-6 edit priority (Ada Byron)
2026-07-27T19:18:42+05:30  commit 4399e73 PAY-6 state TODO -> IN_PROGRESS (Ada Byron)
2026-07-27T19:18:41+05:30  commit 3b37787 PAY-7 link relates PAY-6 (Ada Byron)
2026-07-27T19:18:25+05:30  commit 9a059c1 PAY-6 create "Batch scoring is 4x slower than target" (Ada Byron)
```

Under `--json` each entry carries a `kind` of `"event"` or `"commit"`, so the
two streams can be separated again:

```
$ kira log PAY-6 --json
{
  "id": "01KYHXAZNMGR9PNVV5XGXW714K",
  "number": "PAY-6",
  "entries": [
    {
      "kind": "event",
      "ts": "2026-07-27T19:18:42+05:30",
      "field": "priority",
      "old": "P1",
      "new": "P0",
      "sha": "a33484b73ee660fb7a6fcae0324c10b263c7c807"
    },
    ...
  ]
}
```

---

## `kira blame`

Where `log` is chronological, `blame` is a snapshot: for each field of an
item, its *current* value and the commit that last set it. It answers "who
made this P0, and when", in the same spirit as `git blame`.

`blame` takes no flags of its own.

```
$ kira blame PAY-6
FIELD     VALUE                                   WHEN                       BY         SOURCE
id        01KYHXAZNMGR9PNVV5XGXW714K              2026-07-27T19:18:25+05:30  Ada Byron  created
number    PAY-6                                   2026-07-27T19:18:25+05:30  Ada Byron  created
type      ticket                                  2026-07-27T19:18:25+05:30  Ada Byron  created
subtype   bug                                     2026-07-27T19:18:25+05:30  Ada Byron  created
title     Batch scoring is 4x slower than target  2026-07-27T19:18:25+05:30  Ada Byron  created
state     IN_PROGRESS                             2026-07-27T19:18:42+05:30  Ada Byron  commit
priority  P0                                      2026-07-27T19:18:42+05:30  Ada Byron  commit
owner     linus                                   2026-07-27T19:18:25+05:30  Ada Byron  created
epic      01KYHXAGXCR4BZ5VCRY84XS56Z              2026-07-27T19:18:25+05:30  Ada Byron  created
sprint    2026-S2                                 2026-07-27T19:18:25+05:30  Ada Byron  created
estimate  8                                       2026-07-27T19:18:25+05:30  Ada Byron  created
```

The `SOURCE` column distinguishes a value that has held since the item was
created from one changed later by a commit. Here `state` and `priority` read
`commit` because both were edited after creation; everything else still holds
its original value. Fields that were never set do not appear at all.

---

## `kira stats`

Delivery metrics computed from the item history. Three things are reported:

- **completion** — how many items are `done`, out of how many, as a
  percentage. Items resolved as dropped are counted separately.
- **cycle time** and **lead time**, in days, as p50 and p90 with the sample
  size `n`. Cycle time measures how long work took once started; lead time
  measures the whole span from creation. Both are reported only when there are
  completed items to measure.
- **throughput/week** — a series of completion counts, one entry per trailing
  week, oldest first.

| Flag | Value | Default | Effect |
| --- | --- | --- | --- |
| `--since` | `YYYY-MM-DD` | — | Include only items created on or after this date. |
| `--sprint` | sprint key or `active` | — | Scope the metrics to one sprint. |
| `--weeks` | integer | `8` | Length of the trailing throughput series. |

An optional positional epic id scopes everything to that epic's subtree.

```
$ kira stats
Stats (project)
  completion: 1/8 done (13%)
  cycle time (days): p50 0  p90 0  n=1
  lead time (days): p50 0  p90 0  n=1
  throughput/week: 0 0 0 0 0 0 0 1
```

The header line names the scope, which makes the scoped forms self-labelling:

```
$ kira stats PAY-1
Stats (epic PAY-1)
  completion: 0/3 done (0%)
  throughput/week: 0 0 0 0 0 0 0 0
```

```
$ kira stats --sprint active
Stats (sprint 2026-S2)
  completion: 0/3 done (0%)
  throughput/week: 0 0 0 0 0 0 0 0
```

Neither scoped run prints cycle or lead time, because neither scope has a
completed item to measure yet.

`--weeks` changes only the length of the throughput series, not the
completion or percentile figures:

```
$ kira stats --weeks 3
Stats (project)
  completion: 1/8 done (13%)
  cycle time (days): p50 0  p90 0  n=1
  lead time (days): p50 0  p90 0  n=1
  throughput/week: 0 0 1
```

`--since` does change the population, since it drops items created before the
date. In this repository every item was created inside the window, so the
figures are unchanged and only the scope label differs — on a long-lived
backlog it is the flag that stops last year's abandoned tickets from dragging
the completion percentage down:

```
$ kira stats --since 2026-07-20
Stats (since 2026-07-20)
  completion: 1/8 done (13%)
  cycle time (days): p50 0  p90 0  n=1
  lead time (days): p50 0  p90 0  n=1
  throughput/week: 0 0 0 0 0 0 0 1
```

The `--json` form adds a `reopens` block — items that left a done state and
came back — which the human output only surfaces when non-zero.

---

## `kira now`

A focused status line for the ticket you are currently working on: how long it
has been in its current state, which branch you are on, anything blocking it,
and the commits made since it last changed state.

`now` takes no flags of its own.

```
$ kira now
PAY-3  Tokenize card numbers at ingest  [IN_PROGRESS]
in state:    1m
branch:      main
commits:     6 since last state change
  9e8737d  kira: map git identities to people
  9a8f7da  PAY-3 memoize tokenizer lookups
  73bcba5  Add PAN tokenizer
  ...
```

The commit list is everything on the current branch since the ticket last
changed state — a quick answer to "what have I done since I picked this up",
and a reminder when the answer is nothing.

Switching to a ticket that has an unmet dependency surfaces it:

```
$ kira now
PAY-4  Retry failed authorizations with backoff  [IN_PROGRESS]
in state:    just now
branch:      main
blocked by:  PAY-3 [IN_PROGRESS]
commits:     0 since last state change
```

Each blocker is shown with its own current state, so you can tell at a glance
whether the thing in your way is being worked on. If the active item has a due
date it is shown too, annotated `overdue` once it has passed.

kira determines the active ticket two ways, in order: an explicit pointer set
by `kira workon <id>`, `kira move --activate`, or `kira create --here`; and
failing that, by inferring the item number from your current branch name. The
`--json` output records which applied in its `source` field (`"pointer"` or
`"branch"`). With neither available the command is an error:

```
kira: no active ticket
  hint: start one with `kira workon <id>`
```

See [git integration](./git-integration.md) for `workon` and branch naming.

---

## `kira diff`

Compares two versions of the backlog and reports the field-level differences —
which items changed, what each field went from and to, and which items were
created or deleted. Unlike `git diff`, it speaks in tickets and fields rather
than lines of YAML.

| Flag | Value | Effect |
| --- | --- | --- |
| `--incoming` | — | Show what changed on `<ref>` since the merge-base, instead of what changed on your side. |
| `--since` | git ref or `YYYY-MM-DD` | Show changes on `HEAD` since that point. Cannot be combined with a positional ref or `--incoming`. |

You must supply either a positional `<ref>` or `--since`; with neither, the
command tells you so:

```
$ kira diff
kira: diff requires a <ref> argument or --since
```

### Outgoing: what have I changed?

By default `kira diff <ref>` compares your `HEAD` against the merge-base with
`<ref>` — that is, it shows *your* divergence, ignoring anything that landed
on `<ref>` in the meantime. Run from `main` after some local edits:

```
$ kira diff fraud-scoring
PAY-5  Settlement report off by one day
  state: REVIEW -> DONE
  resolution: - -> done
PAY-7  Spike: graph features for chargeback risk
  priority: P3 -> P2
  owner: - -> grace
```

A `-` on the left of an arrow means the field was previously unset.

### Incoming: what will a merge do to my tickets?

`--incoming` flips the direction: it reports what changed on `<ref>` relative
to the merge-base — the changes that would arrive if you merged or pulled it.
This is the command to run before merging a long-lived branch, or after a
`git fetch` and before a `git pull`, when you want to know whether someone
else has reassigned your tickets or closed something you thought was open:

```
$ kira diff --incoming fraud-scoring
PAY-6  Batch scoring is 4x slower than target
  state: IN_PROGRESS -> REVIEW
  priority: P0 -> P1
  labels: [risk, perf] -> [risk, perf, tech-debt]
created PAY-9  Cache feature vectors between scoring runs
```

Created and deleted items are prefixed `created` and `deleted`. If a merge
would renumber an item, that is reported too, as a `renumbered X -> Y` line
under the item.

### Since a point in time

`--since` compares `HEAD` against an earlier ref or date rather than against a
branch — useful for a standup summary of what moved:

```
$ kira diff --since HEAD~3
PAY-5  Settlement report off by one day
  state: REVIEW -> DONE
  resolution: - -> done
PAY-7  Spike: graph features for chargeback risk
  priority: P3 -> P2
  owner: - -> grace
```

Dates resolve the same way as `--at`: to the last commit at or before midnight
of the given day, so a date with no commits behind it is an error rather than
an empty diff.

When nothing differs, `kira diff` says so explicitly:

```
no backlog differences
```
