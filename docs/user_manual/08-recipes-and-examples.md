# Recipes and Examples

Task-oriented sequences; flags and full semantics live in the
[CLI reference](04-cli-reference/README.md). Samples come from one repository,
key `CORE`, run in order; `@me` resolves to the git user `Demo User` →
`Demo-User`.

## Idea to done

```console
$ kira create ticket "Fix flaky retry backoff" --no-edit
Created CORE-1 Fix flaky retry backoff
$ kira workon CORE-1
Created branch core/core-1-fix-flaky-retry-backoff for CORE-1
Moved CORE-1: TODO -> IN_PROGRESS
$ kira move CORE-1 REVIEW
Moved CORE-1: IN_PROGRESS -> REVIEW
$ kira move CORE-1 DONE
Moved CORE-1: REVIEW -> DONE
```

`workon` creates or switches to the item's branch, sets it active and moves it
to doing ([`workon`](04-cli-reference/git-integration.md#kira-workon)). The
state commits land on the branch and reach `main` with it.

## Epics, nested

```console
$ kira create epic "Order gateway rewrite" --no-edit
Created CORE-2 Order gateway rewrite
$ kira create ticket "Drain in-flight orders on shutdown" --no-edit --parent CORE-2 --priority P1
Created CORE-3 Drain in-flight orders on shutdown
$ kira tree
CORE-1      ticket  Fix flaky retry backoff
CORE-2      epic    Order gateway rewrite
  CORE-3    ticket  Drain in-flight orders on shutdown
  CORE-4    ticket  Replace legacy session cache
  CORE-5    epic    Wire protocol v3
    CORE-6  ticket  Bump protocol version to v3
CORE-7      ticket  Audit log retention policy
```

`CORE-4`…`CORE-7` were created the same way; `CORE-5` is an epic under an
epic. Reparent with
`kira link <id> --parent <epic>`.

## What next, and what is mine

```console
$ kira list --filter next
NUMBER  STATE  TYPE    PRIORITY  TITLE
CORE-6  TODO   ticket  P0        Bump protocol version to v3
CORE-3  TODO   ticket  P1        Drain in-flight orders on shutdown
CORE-4  TODO   ticket  P2        Replace legacy session cache
CORE-7  TODO   ticket  -         Audit log retention policy
$ kira list --owner @me --state TODO
NUMBER  STATE  TYPE    PRIORITY  TITLE
CORE-6  TODO   ticket  P0        Bump protocol version to v3
CORE-3  TODO   ticket  P1        Drain in-flight orders on shutdown
```

`next` is the filter `kira init` writes. Flags, `--filter` and `--query` AND
together.

## Blockers

```console
$ kira link CORE-4 --blocked-by CORE-6
Linked CORE-4
$ kira list --query blocked
NUMBER  STATE  TYPE    PRIORITY  TITLE
CORE-4  TODO   ticket  P2        Replace legacy session cache
$ kira list --query "blocked_by = CORE-6"
NUMBER  STATE  TYPE    PRIORITY  TITLE
CORE-4  TODO   ticket  P2        Replace legacy session cache
$ kira show CORE-6 --json | jq -c ".blocks_numbers"
["CORE-4"]
```

Only the dependent stores the edge; `blocks` is derived. `blocked` clears when
every blocker is in a done state. Remove the edge with `--remove`.

## Capture a follow-up mid-task

```console
$ kira workon CORE-3
Created branch core/core-3-drain-in-flight-orders-on-shutdown for CORE-3
Moved CORE-3: TODO -> IN_PROGRESS
$ kira create ticket --here --no-edit --title "Benchmark the drain path"
Created CORE-8 Benchmark the drain path
$ kira create ticket --here --no-edit --title "Pick a drain timeout" --blocking
Created CORE-9 Pick a drain timeout
$ kira show CORE-9 --format number-title; kira list --query "blocked_by = CORE-9"
CORE-9 Pick a drain timeout
NUMBER  STATE        TYPE    PRIORITY  TITLE
CORE-3  IN_PROGRESS  ticket  P1        Drain in-flight orders on shutdown
```

`--here` copies the active item's parent and sprint and adds the `captured`
label; `--blocking` makes the new item block the active one
([`--here`](04-cli-reference/item-lifecycle.md#--here----blocking)).

## A sprint with spillover

```console
$ kira sprint create --key 2026-S46 --name "Sprint 46" --start 2026-11-09 --end 2026-11-20
Created sprint 2026-S46 (2026-11-09 -> 2026-11-20)
$ kira sprint activate 2026-S46
Activated sprint 2026-S46
$ kira edit CORE-4 CORE-6 --sprint 2026-S46
Edited CORE-4: sprint
Edited CORE-6: sprint
$ kira sprint list
* 2026-S46  Sprint 46  2026-11-09 -> 2026-11-20  1/2 done
  2026-S48  Sprint 48  2026-11-23 -> 2026-12-04  0/0 done
$ kira sprint close 2026-S46 --move-to 2026-S48
Closed sprint 2026-S46
  unfinished: CORE-4
Moved 1 unfinished item(s) to 2026-S48
Cleared the active-sprint pointer
```

While a sprint is active, `--sprint active` works wherever a sprint key does.

## Review a branch's effect on the backlog

```console
$ kira diff --incoming feature/gateway
CORE-4  Replace legacy session cache
  state: TODO -> IN_PROGRESS
CORE-7  Audit log retention policy
  priority: - -> P1
created CORE-10  Add gateway metrics
$ kira changes --since HEAD~2
CORE-4  Replace legacy session cache
  2026-11-13T09:35:00+05:30  sprint: 2026-S46 -> 2026-S48
CORE-6  Bump protocol version to v3
  2026-11-13T09:33:00+05:30  state: TODO -> DONE
  2026-11-13T09:33:00+05:30  resolution:  -> done
```

`diff --incoming` shows what the branch would bring in, per field; plain
`diff` shows yours. `changes` lists field events since a ref or date.

## Merge conflict on an item file

Without the merge driver, both sides set `CORE-7`'s priority:

```console
$ git merge --no-edit teammate
Auto-merging .kira/tickets/01M7APKGZ28XD31HHB43CPMAYT.md
CONFLICT (content): Merge conflict in .kira/tickets/01M7APKGZ28XD31HHB43CPMAYT.md
Automatic merge failed; fix conflicts and then commit the result.
$ kira resolve
Resolved CORE-7 (auto-merged: priority)
$ kira hooks install
hook post-merge: installed
hook prepare-commit-msg: installed
merge driver: registered
```

`resolve` merges per field and stages; on a same-field clash the side with the
later `updated` wins (here this branch's `P0`; a tie goes to the incoming
side). Then `git commit`. With the merge driver installed, git does this
during the merge.

## Throughput and cycle time

```console
$ kira stats
Stats (project)
  completion: 2/9 done (22%)
  cycle time (days): p50 0  p90 0  n=1
  lead time (days): p50 0  p90 0  n=2
  throughput/week: 0 0 0 0 0 0 0 2
```

Scope with `--sprint <key|active>` ([`stats`](04-cli-reference/views-and-queries.md#kira-stats)).

## The backlog at a past point

```console
$ kira list --at v0.1.0 --state DONE
NUMBER  STATE  TYPE    PRIORITY  TITLE
CORE-1  DONE   ticket  -         Fix flaky retry backoff
```

`--at` takes a ref or `YYYY-MM-DD` on `list`, `show`, `tree` and `board`
(read-only).

## Scripting

```console
$ kira show CORE-4 --format branch
core/core-4-replace-legacy-session-cache
$ kira list --json | jq -r ".items | group_by(.state)[] | \"\(.[0].state): \(length)\""
DONE: 2
IN_PROGRESS: 1
PLANNED: 2
TODO: 4
$ kira list --json | jq -r ".items[] | select(.category != \"done\") | \"\(.number)\t\(.priority // \"-\")\t\(.title)\""
CORE-7	P0	Audit-log retention policy
CORE-3	P1	Drain in-flight orders on shutdown
CORE-4	P2	Replace legacy session cache
CORE-2	-	Order gateway rewrite
CORE-5	-	Wire protocol v3
CORE-8	-	Benchmark the drain path
CORE-9	-	Pick a drain timeout
```

`--json` goes to stdout, errors to stderr; shapes from `kira schema`. To preview
a new item's file without creating it: `kira create <type> --print-template`.

## Search item text

```console
$ kira find "session cache"
CORE-4:6:title: "Replace legacy session cache"
$ kira find -i RETRY
CORE-1:6:title: "Fix flaky retry backoff"
```

Regex over item files, hits as `NUMBER:LINE:text`; `--query` matches fields,
`find` matches text, bodies included.

## Move an item to another board

```console
$ kira board move CORE-7 INFRA
Moved CORE-7 -> INFRA-1
$ kira show CORE-7 --format number-title
INFRA-1 Audit-log retention policy
```

The old number stays an alias, so references to `CORE-7` keep resolving.

## Validate hand-edited files

```console
$ kira validate CORE-4; echo "exit=$?"
error   state           CORE-4 state: "SHIPPED" is not a state in the ticket workflow
problems found: 1 error, 0 warnings, 0 info
see `kira config explain` for the rules these checks enforce
kira: validate: 1 problem found
exit=1
```

In CI: `kira validate .kira/tickets/*.md`; for cross-item checks,
`kira doctor`. When a value is refused, `kira config explain` shows the rule
and which config tier set it.
