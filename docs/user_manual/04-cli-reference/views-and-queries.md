# Views and queries

Read-only commands: `list`, `show`, `tree`, `find`, `log`, `blame`, `stats`,
`now`, `diff`, `changes`. None writes a file or a commit. Global flags:
[Setup and schema commands](setup-and-schema-commands.md#global-flags).

Samples: project `PAY`, epics `PAY-1`/`PAY-2`, people `ada` (the git user),
`grace`, `linus`, active sprint `S2`, and a branch `fraud-scoring`.

## Git refs and dates

`--at` (list, show, tree, board), `diff --since` and `changes --since` take a
git ref or `YYYY-MM-DD`. A date resolves to the last commit on `HEAD` at or
before **00:00** of that day, so today's date usually fails:

```console
$ kira list --at 2026-11-13
kira: resolving --at 2026-11-13: no commit on HEAD at or before 2026-11-13T00:00:00
```

A ref need not be an ancestor of `HEAD`: `--at fraud-scoring` reads that
branch.

## `kira list`

| Flag | Value |
| --- | --- |
| `--state`, `--category`, `--type`, `--subtype`, `--priority`, `--label` | exact match (`--category` `todo`\|`doing`\|`done`, `--type` `ticket`\|`epic`) |
| `--owner` | person or `@me` |
| `--parent` | item; direct children only |
| `--sprint` | key or `active` |
| `--board` | board key |
| `--filter` | name from `filters:` |
| `--query` | [query expression](#query-expressions) |
| `--tree` | group the result under parents |
| `--at` | ref or date |

All flags AND together and with `--query`; use `OR` inside `--query`.
Default order: items with a `rank` first (lexicographic), then priority
(`P0` first), then number. `ORDER BY` overrides, with that order as tie-break.
Columns come from `ui.list.columns`.

```console
$ kira list
NUMBER  STATE        TYPE    PRIORITY  TITLE
PAY-3   IN_PROGRESS  ticket  P0        Tokenize card numbers at ingest
PAY-5   DONE         ticket  P0        Settlement report off by one day
PAY-6   IN_PROGRESS  ticket  P0        Batch scoring is 4x slower than target
PAY-1   ACTIVE       epic    P1        Card payment rails
PAY-4   TODO         ticket  P1        Retry failed authorizations with backoff
PAY-2   ACTIVE       epic    P2        Fraud detection
PAY-7   TODO         ticket  P2        Spike: graph features for chargeback risk
PAY-8   DONE         ticket  P2        Drop the legacy /v1/charge endpoint
$ kira list --type ticket --category doing
NUMBER  STATE        TYPE    PRIORITY  TITLE
PAY-3   IN_PROGRESS  ticket  P0        Tokenize card numbers at ingest
PAY-6   IN_PROGRESS  ticket  P0        Batch scoring is 4x slower than target
$ kira list --sprint active
NUMBER  STATE        TYPE    PRIORITY  TITLE
PAY-3   IN_PROGRESS  ticket  P0        Tokenize card numbers at ingest
PAY-6   IN_PROGRESS  ticket  P0        Batch scoring is 4x slower than target
PAY-4   TODO         ticket  P1        Retry failed authorizations with backoff
$ kira list --tree
PAY-1                             (epic)
  PAY-3  IN_PROGRESS  ticket  P0  Tokenize card numbers at ingest
  PAY-5  DONE         ticket  P0  Settlement report off by one day
  PAY-4  TODO         ticket  P1  Retry failed authorizations with backoff
PAY-2                             (epic)
  PAY-6  IN_PROGRESS  ticket  P0  Batch scoring is 4x slower than target
  PAY-7  TODO         ticket  P2  Spike: graph features for chargeback risk
(no parent)
  PAY-8  DONE  ticket  P2  Drop the legacy /v1/charge endpoint
$ kira list --filter next
NUMBER  STATE  TYPE    PRIORITY  TITLE
PAY-7   TODO   ticket  P2        Spike: graph features for chargeback risk
$ kira list --at fraud-scoring
NUMBER  STATE   TYPE    PRIORITY  TITLE
PAY-3   TODO    ticket  P0        Tokenize card numbers at ingest
PAY-5   REVIEW  ticket  P0        Settlement report off by one day
PAY-1   ACTIVE  epic    P1        Card payment rails
PAY-4   TODO    ticket  P1        Retry failed authorizations with backoff
PAY-6   REVIEW  ticket  P1        Batch scoring is 4x slower than target
PAY-2   ACTIVE  epic    P2        Fraud detection
PAY-8   DONE    ticket  P2        Drop the legacy /v1/charge endpoint
PAY-9   TODO    ticket  P2        Cache feature vectors between scoring runs
PAY-7   TODO    ticket  P3        Spike: graph features for chargeback risk
```

`next` (`state = TODO AND NOT blocked ORDER BY priority`) ships with
`kira init`; `PAY-4` is excluded because `PAY-3` blocks it. `--tree` groups
the filtered rows under a header per epic (number and `(epic)` only) that
matches or has a matching child, parentless rows under `(no parent)`;
`kira tree` is the unfiltered hierarchy.

`--json`: `id` is stable, `number` can change on renumbering; `parent` is an
id and `parent_number` its display number.

```console
$ kira list --label payments --category done --json
{
  "items": [
    {
      "id": "01M7ANRREEK7Z3SYGQ8ZCH6VWB",
      "number": "PAY-5",
      "board": "PAY",
      "title": "Settlement report off by one day",
      "type": "ticket",
      "subtype": "bug",
      "state": "DONE",
      "category": "done",
      "owner": "grace",
      "labels": [
        "payments"
      ],
      "parent": "01M7ANRRBRHVH4M3S738HPBEGG",
      "parent_number": "PAY-1",
      "priority": "P0",
      "resolution": "done",
      "due": "2026-11-20"
    }
  ],
  "count": 1
}
```

## Query expressions

Used by `--query`, `filters:`, the TUI `/` filter and the web dashboard.

- **Combinators:** `AND`, `OR`, `NOT`, parentheses; keywords case-insensitive;
  adjacent predicates without a connective are ANDed.
- **`ORDER BY <field> [asc|desc]`:** once, last.
- **Operators:** `=`, `!=` on every field; `<`, `<=`, `>`, `>=` only on
  dates, `estimate` and `priority` (elsewhere a parse error);
  `field IN (a, b)`; `field IS EMPTY` / `IS NOT EMPTY`.
- **Bare word or quoted string:** case-insensitive substring of the **title**
  (body text: [`kira find`](#kira-find)).
- **Strings:** double-quote values with spaces or punctuation; escapes `\"`,
  `\\`.

| Field | Kind | Notes |
| --- | --- | --- |
| `state`, `category`, `type`, `subtype`, `resolution`, `board` | text | `board` case-insensitive |
| `owner`, `reporter` | text | accept `@me` |
| `priority` | ordered | `priority < P2` = more urgent than P2 |
| `rank` | text | |
| `parent`, `sprint` | reference | `parent` accepts a number; legacy name `epic`; `sprint = active` |
| `label`, `blocked_by`, `links` | list | `=` means contains; no `ORDER BY` |
| `estimate` | number | |
| `created`, `updated`, `due` | date | `YYYY-MM-DD`, by calendar day; `due` also takes RFC 3339 |
| `blocked` | boolean | bare: `blocked`, `NOT blocked` — any blocker not done |

`state`, `type`, `category`, `created`, `updated`, `board` reject `IS EMPTY`.

```console
$ kira list --query "label = payments AND category != done ORDER BY priority"
NUMBER  STATE        TYPE    PRIORITY  TITLE
PAY-3   IN_PROGRESS  ticket  P0        Tokenize card numbers at ingest
PAY-4   TODO         ticket  P1        Retry failed authorizations with backoff
$ kira list --query "priority IN (P0, P1) AND owner IS NOT EMPTY ORDER BY owner desc"
NUMBER  STATE        TYPE    PRIORITY  TITLE
PAY-6   IN_PROGRESS  ticket  P0        Batch scoring is 4x slower than target
PAY-5   DONE         ticket  P0        Settlement report off by one day
PAY-3   IN_PROGRESS  ticket  P0        Tokenize card numbers at ingest
PAY-1   ACTIVE       epic    P1        Card payment rails
$ kira list --query "NOT blocked AND estimate > 2 ORDER BY estimate desc"
NUMBER  STATE        TYPE    PRIORITY  TITLE
PAY-6   IN_PROGRESS  ticket  P0        Batch scoring is 4x slower than target
PAY-3   IN_PROGRESS  ticket  P0        Tokenize card numbers at ingest
$ kira list --query "due IS NOT EMPTY OR subtype = bug"
NUMBER  STATE        TYPE    PRIORITY  TITLE
PAY-5   DONE         ticket  P0        Settlement report off by one day
PAY-6   IN_PROGRESS  ticket  P0        Batch scoring is 4x slower than target
$ kira list --query "parent = PAY-2 AND priority < P3"
NUMBER  STATE        TYPE    PRIORITY  TITLE
PAY-6   IN_PROGRESS  ticket  P0        Batch scoring is 4x slower than target
PAY-7   TODO         ticket  P2        Spike: graph features for chargeback risk
$ kira list --query "scoring"
NUMBER  STATE        TYPE    PRIORITY  TITLE
PAY-6   IN_PROGRESS  ticket  P0        Batch scoring is 4x slower than target
$ kira list --query "asignee = ada"
kira: query: unknown field "asignee" (available: blocked, blocked_by, board, category, created, due, estimate, label, links, owner, parent, priority, rank, reporter, resolution, sprint, state, subtype, type, updated) at position 8
  hint: check the query expression syntax
$ kira list --query "state IS EMPTY"
kira: query: field state is never empty at position 6
  hint: check the query expression syntax
```

## `kira show`

```
kira show <id> [--at <ref>] [--format <form>]
```

Prints the header, the set fields and the body. The human view omits `due`,
`sprint`, `estimate` and `resolution`; `--json` has every field plus derived
`blocks`, `parent_number` and `*_numbers` companions, `comments`, `linked_commits`,
`referenced_by` and `history_tail`.

```console
$ kira show PAY-3
PAY-3  Tokenize card numbers at ingest  [IN_PROGRESS]
id:       01M7ANQPWAT67HC9WGRK3E1VFN
type:     ticket
subtype:  feature
category: doing
priority: P0
owner:    ada
labels:   payments, security
parent:   PAY-1
blocks:   PAY-4
created:  2026-11-13T20:12:09+05:30
updated:  2026-11-13T20:12:09+05:30


## Description

## Acceptance criteria
```

`--format` prints one reference form and cannot be combined with `--json`:

| Form | Output for `PAY-3` |
| --- | --- |
| `id` | the ULID |
| `number` | `PAY-3` |
| `number-title` | `PAY-3 Tokenize card numbers at ingest` |
| `markdown` | a Markdown link: text `PAY-3 Tokenize card numbers at ingest`, target `PAY-3` |
| `branch` | `pay/pay-3-tokenize-card-numbers-at-ingest` (from `workon.branch_pattern`) |

```console
$ kira show PAY-3 --format branch
pay/pay-3-tokenize-card-numbers-at-ingest
$ kira show PAY-3 --format slug
kira: unknown format "slug", want one of id, number, number-title, markdown, branch
```

With `--at`, a number that named a different item at that ref prints a
warning on stderr; stdout shows the item the number resolved to there.

## `kira tree`

```
kira tree [<id>] [--at <ref>]
```

The whole hierarchy, unfiltered, children ordered by number; `<id>` scopes it
to one subtree.

```console
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

## `kira find`

```
kira find <pattern> [-i|--ignore-case] [-w|--word-regexp]
```

Regex search (Go [RE2](https://github.com/google/re2/wiki/Syntax)), line by
line, over the item files — frontmatter and body. Output `NUMBER:LINE:TEXT`;
`--json` sorted by number. `-C` is the global chdir flag, not context lines.

```console
$ kira find backoff
PAY-4:7:title: "Retry failed authorizations with backoff"
$ kira find -i CHARGEBACK
PAY-7:7:title: "Spike: graph features for chargeback risk"
$ kira find backoff --json
{
  "matches": [
    {
      "id": "01M7ANQPWZTWX5W7BHVQYJD1TV",
      "number": "PAY-4",
      "line": 7,
      "text": "title: \"Retry failed authorizations with backoff\""
    }
  ]
}
$ kira find "[unclosed"
kira: find: invalid pattern: error parsing regexp: missing closing ]: `[unclosed`
```

## `kira log`

```
kira log <id>
```

Newest first: field events (`field: old -> new`) interleaved with commits
linked to the item — via a `Kira-Ticket:` trailer (`Add PAN tokenizer`), a
leading number in the subject, or a mention (`PAY-4 link blocked-by PAY-3`
names `PAY-3`). List-valued fields produce no events (CORE-67). `--json`
entries carry `kind: "event"` or `"commit"`.

```console
$ kira log PAY-3
2026-11-13T09:25:00+05:30  commit 459a055 PAY-3 memoize tokenizer lookups (Demo User)
2026-11-13T09:24:00+05:30  commit 3ce31d2 Add PAN tokenizer (Demo User)
2026-11-13T09:23:00+05:30  state: TODO -> IN_PROGRESS
2026-11-13T09:23:00+05:30  commit ba32b75 PAY-3 state TODO -> IN_PROGRESS (Demo User)
2026-11-13T09:15:00+05:30  commit 77f1fc9 PAY-4 link blocked-by PAY-3 (Demo User)
2026-11-13T09:08:00+05:30  commit d751646 PAY-3 create "Tokenize card numbers at ingest" (Demo User)
```

## `kira blame`

```
kira blame <id>
```

Per set scalar field: current value and the commit that last set it. `SOURCE`
is `created` (unchanged since creation) or `commit`.

```console
$ kira blame PAY-6
FIELD     VALUE                                   WHEN                       BY         SOURCE
id        01M7ANQPY7QF63QAVBKYWYS5E7              2026-11-13T09:11:00+05:30  Demo User  created
number    PAY-6                                   2026-11-13T09:11:00+05:30  Demo User  created
type      ticket                                  2026-11-13T09:11:00+05:30  Demo User  created
subtype   bug                                     2026-11-13T09:11:00+05:30  Demo User  created
title     Batch scoring is 4x slower than target  2026-11-13T09:11:00+05:30  Demo User  created
state     IN_PROGRESS                             2026-11-13T09:19:00+05:30  Demo User  commit
priority  P0                                      2026-11-13T09:20:00+05:30  Demo User  commit
owner     linus                                   2026-11-13T09:11:00+05:30  Demo User  created
parent    01M7ANQPVP1YKNQQ5MCVWW778V              2026-11-13T09:11:00+05:30  Demo User  created
sprint    S2                                      2026-11-13T09:11:00+05:30  Demo User  created
estimate  8                                       2026-11-13T09:11:00+05:30  Demo User  created
```

## `kira stats`

```
kira stats [<parent-id>] [--sprint <key>|active] [--since YYYY-MM-DD] [--weeks N]
```

| Metric | Definition |
| --- | --- |
| completion | items in a `done` state / all, with `dropped` counted separately |
| cycle time | first `doing` state → done, days, p50/p90, `n`; shown only when `n > 0` |
| lead time | created → done, same form |
| throughput/week | done per trailing week, oldest first; `--weeks` (default 8) sets the length |
| reopens | re-entries into a `doing` state after done (`DONE` → `TODO` alone is not one); shown when > 0 |

`<parent-id>` scopes to that subtree, `--sprint` to one sprint, `--since` to
items created on or after the date. Dropped items are excluded from cycle,
lead and throughput.

```console
$ kira stats
Stats (project)
  completion: 2/8 done (25%)
  cycle time (days): p50 0  p90 0  n=1
  lead time (days): p50 0  p90 0  n=2
  throughput/week: 0 0 0 0 0 0 0 2
$ kira stats PAY-1
Stats (parent PAY-1)
  completion: 1/3 done (33%)
  cycle time (days): p50 0  p90 0  n=1
  lead time (days): p50 0  p90 0  n=1
  throughput/week: 0 0 0 0 0 0 0 1
$ kira stats --sprint active
Stats (sprint S2)
  completion: 0/3 done (0%)
  throughput/week: 0 0 0 0 0 0 0 0
```

## `kira now`

The active ticket: time in state, branch, open blockers with their states, due
date (`overdue` once past), and commits on the current branch since its last
state change.

The active ticket is the pointer set by `kira workon`, `move --activate` or
`create --here`, else the number inferred from the branch name; `--json`
`source` says which (`pointer`/`branch`). Neither: `kira: no active ticket`.

```console
$ kira now
PAY-3  Tokenize card numbers at ingest  [IN_PROGRESS]
in state:    2h
branch:      main
commits:     4 since last state change
  398f71e  PAY-7 edit priority,owner
  2930945  PAY-5 state REVIEW -> DONE
  459a055  PAY-3 memoize tokenizer lookups
  3ce31d2  Add PAN tokenizer
$ kira now
PAY-4  Retry failed authorizations with backoff  [IN_PROGRESS]
in state:    1h
branch:      main
blocked by:  PAY-3 [IN_PROGRESS]
commits:     0 since last state change
```

## `kira diff`

```
kira diff <ref> [--incoming]
kira diff --since <ref|date>
```

Field-level differences between two snapshots of the item set.

| Form | Compares |
| --- | --- |
| `diff <ref>` | merge-base(`HEAD`, ref) → `HEAD` — your side |
| `diff --incoming <ref>` | merge-base → `<ref>` — what a merge would bring |
| `diff --since <x>` | `<x>` → `HEAD` |

`-` means unset; new and removed items are prefixed `created`/`deleted`; a
renumbering shows as `renumbered X -> Y`. Identical snapshots print
`no backlog differences`.

```console
$ kira diff fraud-scoring
PAY-3  Tokenize card numbers at ingest
  state: TODO -> IN_PROGRESS
PAY-4  Retry failed authorizations with backoff
  state: TODO -> IN_PROGRESS
PAY-5  Settlement report off by one day
  state: REVIEW -> DONE
  resolution: - -> done
PAY-7  Spike: graph features for chargeback risk
  priority: P3 -> P2
  owner: - -> grace
$ kira diff --incoming fraud-scoring
PAY-6  Batch scoring is 4x slower than target
  state: IN_PROGRESS -> REVIEW
  priority: P0 -> P1
  labels: [risk, perf] -> [risk, perf, tech-debt]
created PAY-9  Cache feature vectors between scoring runs
$ kira diff --since HEAD~3 --incoming
kira: --since cannot be combined with --incoming
$ kira diff main --since HEAD~3
kira: provide a ref positionally or --since, not both
$ kira diff
kira: diff requires a <ref> argument or --since
```

## `kira changes`

```
kira changes --since <ref|date>
```

Every field event across all items in `(since, HEAD]`, grouped by item —
where `diff --since` compares two snapshots, `changes` walks the commits
between them. Items created in the window are prefixed `created`. `--json`
adds the resolved `since`/`head` SHAs and each event's commit.

```console
$ kira changes --since HEAD~6
PAY-3  Tokenize card numbers at ingest
  2026-11-13T09:23:00+05:30  state: TODO -> IN_PROGRESS
PAY-4  Retry failed authorizations with backoff
  2026-11-13T10:18:00+05:30  state: TODO -> IN_PROGRESS
PAY-5  Settlement report off by one day
  2026-11-13T09:26:00+05:30  state: REVIEW -> DONE
  2026-11-13T09:26:00+05:30  resolution:  -> done
PAY-7  Spike: graph features for chargeback risk
  2026-11-13T09:27:00+05:30  priority: P3 -> P2
  2026-11-13T09:27:00+05:30  owner:  -> grace
$ kira changes
kira: changes: --since <ref> is required
  hint: example: kira changes --since HEAD~10
```
