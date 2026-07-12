# Data Model

**Scope:** the Item schema, edges between items, comments, labels/people, the state machine, and the ID scheme.
Part of the kira design set — see [DESIGN.md](../../DESIGN.md) for decisions and rationale.

## 1. Item schema

One schema for both tickets and epics — see [§2](#2-unified-ticketepic-decision). Stored as YAML frontmatter + markdown body in `.kira/tickets/<ulid>.md`. Exact field list:

| Field | Type | Required | Semantics | Mutability |
|---|---|---|---|---|
| `id` | string (ULID) | yes | Immutable identity. Filename stem. The only key used in cross-references and the sqlite index. | immutable |
| `number` | string (`KIRA-n`) | yes | Human display ID. Sequentially allocated — see [§7](#7-id-scheme). | mutable (renumber on collision only, via `doctor`) |
| `aliases` | []string | yes (may be `[]`) | Prior `number` values, retained after a `doctor` renumber so old references keep resolving forever. | append-only |
| `type` | enum: `ticket`\|`epic` | yes | Discriminates the two use-cases sharing this schema. | immutable |
| `subtype` | string \| `null` | no | Config-validated enum from `subtypes` (e.g. `bug`\|`story`\|`task`); **orthogonal** to `type` (a `ticket` may be a `bug`). Free-form when `subtypes` is empty. | mutable |
| `title` | string | yes | One-line summary. | mutable |
| `state` | string | yes | Must be a `key` in this item's `type`'s configured workflow. | mutable (validated — [§6](#6-state-machine)) |
| `resolution` | string \| `null` | no | Outcome recorded on entering a `done`-category state, from config `resolutions` (global list) or a state's own `resolution:` key. Set by `move`/transition `set:`; cleared when leaving a done state. | mutable |
| `priority` | string \| `null` | no | When config `priorities` is non-empty, must be a member — the list also defines the **ranked order** used by `>`/`<` comparison and default sort ([04-cli.md §4](04-cli.md#4-query-expression-grammar)). Empty `priorities` ⇒ free-form legacy behaviour (equality-only, unvalidated). | mutable |
| `rank` | string \| `null` | no | Lexicographic ordering key for manual backlog grooming — sparse, free-form (fractional-index style). Primary sort key when present; see [04-cli.md §4](04-cli.md#4-query-expression-grammar). | mutable |
| `owner` | string | no | Must be in `people.known` if `people.strict: true`. | mutable |
| `reporter` | string | no | Same constraint as `owner`. | mutable, normally set once at creation |
| `labels` | []string | yes (may be `[]`) | Controlled vocabulary — [§5](#5-labels--people). | mutable |
| `epic` | string (ULID) \| `null` | yes | Single parent. Stored on the child only — [§3](#3-edges). | mutable |
| `blocked_by` | []string (ULIDs) | yes (may be `[]`) | Stored side of the blocking edge — [§3](#3-edges). | mutable |
| `links` | map: type → []string (ULIDs) | no (omitted when empty) | Typed cross-references beyond `epic`/`blocked_by`. v1 types: `relates`, `duplicate_of`. Single-sided storage like `blocked_by` — [§3](#3-edges). | mutable |
| `sprint` | string \| `null` | no | Key into config `sprints` — the item's sprint assignment ([04-cli.md §3 `kira sprint`](04-cli.md#kira-sprint)). | mutable |
| `due` | string (RFC3339 date) \| `null` | no | Target completion date; supports date comparison in queries. | mutable |
| `estimate` | number \| `null` | no | Unit from `config.yaml` `estimate.unit` (points\|hours). | mutable |
| `created` | string (RFC3339) | yes | Set once at creation. | immutable |
| `updated` | string (RFC3339) | yes | Rewritten by the writer on every mutating command. | mutable |
| body | markdown | yes | Three sections: `## Description`, `## Acceptance criteria`, `## Comments`. | mutable ([§4](#4-comments) for the comments section) |

All new optional scalars (`subtype`, `resolution`, `priority`, `rank`, `sprint`, `due`) merge last-write-wins by `updated`, exactly like the other scalars; `links` lists three-way set-merge per link-type, exactly like `labels` — the merge-policy rules in [03-storage-and-git.md](03-storage-and-git.md) apply unchanged.

Canonical serialization order extends the original list — `type, subtype, title, state, resolution, priority, rank, owner, reporter, labels, epic, blocked_by, links, sprint, due, estimate, ...`. Absent optional keys emit no line, so a file that predates these fields round-trips byte-identically.

## 2. Unified ticket/epic decision

One directory (`.kira/tickets/`), one schema, `type: ticket|epic` discriminates. An epic is a ticket with children; children are a query (`epic = <ulid>`), not a separate structure.

Rejected: separate `boards/` directory for epics. Two directories means two globs, two ID counters, and duplicated validation/index code for zero semantic gain — an epic has no field an ordinary item lacks.

The founder's original `kira create board` is this command renamed — "board" now names the kanban view ([05-tui.md §4](05-tui.md#4-kanban-board-view-2)) to avoid the collision.

## 3. Edges

| Edge | Stored on | Derived | Direction |
|---|---|---|---|
| parent/child | child's `epic` field | epic's children = index query `WHERE epic = <ulid>` | child → parent |
| blocks/blocked_by | dependent's `blocked_by` field | blocker's `blocks` = index query `WHERE <ulid> IN blocked_by` | dependent → blocker |
| typed links | source's `links.<type>` list | inverse = index query `WHERE <ulid> IN links.<type>` | source → target |

`links` follows the same **single-sided storage** rule below: each typed link is stored once on the source item; the reciprocal view is derived from the index, never written to the target's file. v1 link types (`relates`, `duplicate_of`) are stored exactly as authored — kira writes no symmetric copy even for conceptually-symmetric relations like `relates`, keeping the invariant uniform.

**Single-sided storage, always.** No field is ever mutated by editing a *different* item's file. This was the one place the fan-in consensus overruled a numeric majority (3/5 samples proposed dual-write: store both `blocked_by` on the dependent and `blocks` on the blocker, keeping both in sync). Reason for overruling it:

- Dual-write is two copies of one fact. Every one of the three dual-write samples independently needed a `doctor`-style reconciliation pass to fix drift when only one side got edited (concurrent branches, hand edits, partial writes) — that's the failure mode showing up in the design itself.
- Single-sided storage has exactly one source of truth per edge, matching the unanimous (5/5) parent-on-child pattern already adopted. Consistency is by construction, not by reconciliation.
- The derived side never touches disk, so it can never itself drift, never conflicts across branches, and needs no `doctor` check.

Cost: reading "what does X block" requires the index (or a full scan), not a direct file read. Acceptable — `kira show <id>` reads the item's own frontmatter/body directly from the file, but fills `blocks`, `linked_commits`, and `history_tail` from the index; those three fields degrade gracefully (empty, with a stderr note) when the index is absent or stale, rather than failing the command.

## 4. Comments

Anchored, append-only HTML-comment blocks inside the `## Comments` section of the body. Exact marker grammar:

```markdown
<!-- kira:comment id=<ULID> author=<string> ts=<RFC3339> -->
<comment body, markdown, free-form>
<!-- /kira:comment -->
```

- `id` is a fresh ULID minted per comment (identity for future edit/delete tooling; not cross-referenced elsewhere in v1).
- Blocks are appended in chronological order; never reordered, never rewritten in place except by an explicit (future) `kira comment --edit`.
- Concurrent comments from two branches land as disjoint appended regions in the same file → clean line-wise git merge, no conflict.

Rejected: one-file-per-comment. Structurally conflict-proof (new file per comment, no shared line ever touched), but it fragments a ticket across N+1 files, breaks "grep one file to read the whole ticket," and buys robustness the append-block form already has in practice — two comments appended in the same merge window are disjoint diff hunks, not a real conflict.

## 5. Labels & people

- `labels` and `owner`/`reporter`/`people` values are checked against `config.yaml`'s `labels.known` / `people.known` lists — controlled vocabularies, not free strings.
- Strictness is configurable per-list (`labels.strict`, `people.strict`; both default `false` — warn on unknown, don't reject). See [§10](#10-validation-rules) for how this interacts with `validate`/`doctor`.
- `--force` on `create`/`edit` bypasses a `strict: true` rejection for that one write; the write still happens and is still auditable (normal frontmatter, normal commit) — force is an escape hatch, not a stealth path.

## 6. State machine

Each `type` (`ticket`, `epic`) has its own workflow block in `config.yaml`:

```yaml
states:
  - { key: TODO,        category: todo }
  - { key: IN_PROGRESS, category: doing }
  - { key: REVIEW,      category: doing }
  - { key: DONE,        category: done }
  - { key: WONT_DO,     category: done, resolution: dropped }
initial: TODO
transitions:
  TODO:        [IN_PROGRESS, WONT_DO]
  IN_PROGRESS: [REVIEW, TODO, WONT_DO]
  REVIEW:      [DONE, IN_PROGRESS]
  DONE:        []
  WONT_DO:     []
enforce_transitions: true
```

- `category` ∈ `todo`\|`doing`\|`done` — every state must declare one.
- `resolution` is optional, only meaningful on `category: done` states that aren't a "real" completion (e.g. `WONT_DO` → `resolution: dropped`). Telemetry reports resolved-not-done separately from done ([03-storage-and-git.md item on telemetry lives with stats — see synthesis §Telemetry]).
- `initial` is the state a new item of this `type` is created in.
- `transitions` is an adjacency map: `kira move <id> <state>` succeeds only if the target is listed for the item's current state.
- `enforce_transitions: true` makes off-graph moves fail; `--force` overrides for that one call. A forced move is still a normal, audited frontmatter write (normal commit, normal `kira log` entry) — it is not a bypass of history, only of the graph check.
- **Telemetry keys off `category`/`resolution`, never off state-name strings.** This is deliberate: state names are config, renamed freely per project (`REVIEW` → `IN_REVIEW`, `WONT_DO` → `CLOSED_NA`); if stats matched on the literal string `"DONE"`, renaming a state would silently break every completion/cycle-time/throughput number with no error. Category is the stable contract between config and code.

**WIP limits, transition guards, and the `resolution` field** (all optional, all additive to the block above):

- **`wip:` per state** — an int cap on how many items may sit in that state. `0` or absent = unlimited. Advisory only: `kira move` **warns** on breach, never blocks (there is no `--force` for it because it does not reject) — see [04-cli.md `kira move`](04-cli.md#kira-move). Board rendering colours over-limit columns (M4, [05-tui.md](05-tui.md)).
- **`require:` / `set:` per transition** — a transition target may be written as a map instead of a bare state key, carrying guards:
  ```yaml
  transitions:
    REVIEW:
      - { to: DONE, require: [resolution], set: { resolution: done } }
      - IN_PROGRESS          # bare string = no guards (back-compat form)
    IN_PROGRESS: [REVIEW, TODO, WONT_DO]   # whole list may stay bare
  ```
  - `require:` — frontmatter fields that must be **non-null** for the move to succeed (Definition-of-Done enforcement, e.g. require `resolution` before `DONE`). A missing field fails `move` with exit 1; `--force` overrides, same as the adjacency check.
  - `set:` — `field: value` assignments applied atomically as part of the transition write (e.g. stamp `resolution: done` on entering `DONE`). Applied after the `require:` check passes.
  - Both forms coexist per state: a state's transition list may mix bare strings and guard maps.
- **`resolution` as a frontmatter field** — the item-level `resolution` ([§1](#1-item-schema)) is populated when entering a `done`-category state: from the state's own `resolution:` key (e.g. `WONT_DO` → `dropped`), from a transition `set: { resolution: ... }`, or from the `--resolution` flag on `move`. It is validated against config `resolutions` and cleared automatically when an item leaves a done-category state (e.g. reopened). State-definition `resolution:` (the WONT_DO tag) and item-field `resolution` stay consistent: the state tag is the default the field takes on entry when nothing else sets it.

## 7. ID scheme

Two-tier: immutable **ULID** identity + human **number**.

- **ULID** — minted locally at creation, no coordination needed (embeds a timestamp, so items sort creation-order by ID). It is the filename (`tickets/<ulid>.md`) and the only value ever used in `epic`, `blocked_by`, and any future cross-reference. A numbering bug can therefore never corrupt a link — this is treated as a hard invariant and fuzz-tested.
- **number** (`KIRA-n`) — allocated at creation as `max(visible numbers on this branch) + 1`. No shared counter file: a counter file would conflict on every concurrent create across branches, which is exactly the failure mode two-tier IDs exist to avoid.
- **Uniqueness domain**: `number` must be unique across the **union of every item's `number` *and* every item's `aliases`** — not just live numbers. A number retired into `aliases` after a renumber still permanently reserves that slot, since stale trailers/links must keep resolving to the item that used to hold it. `doctor` checks this union, including live-vs-alias collisions (a live `number` colliding with someone else's `aliases` entry), not just live-vs-live; "next free `n`" during both initial allocation and collision repair skips any number appearing in either set.
- **Collision repair**: two branches can independently allocate the same `n`. Resolved post-merge by `kira doctor --fix` — full mechanics in [07-git-integration.md §4](07-git-integration.md#4-id-collision-repair-flow). Short version: the later-created ULID (deterministic tiebreak) gets renumbered to the next free `n` (per the uniqueness domain above); its old number is appended to `aliases`, never dropped.
- **Resolution order** — anywhere the CLI accepts an item reference, it tries, in order: full ULID → unique ULID prefix (git-short-SHA style) → `number` (current or in `aliases`) → nothing else. Ambiguous prefixes are a hard error, not a guess.
- **Normalization**: hand-typed `KIRA-n` references inside frontmatter (e.g. an `epic:` or `blocked_by:` entry someone typed by hand instead of picking via `kira link`) are rewritten to the underlying ULID by `kira edit`/`kira validate` on write. Cross-references never persist as numbers.
- **`id.style: hash` fallback** — display ID derived directly from the ULID (`KIRA-N6T4X2`) instead of sequential allocation. Zero reconciliation machinery — no collisions possible, no `doctor --fix` renumber path, no `aliases` growth. Prefer this over sequential numbering when: reconciliation bugs surface in practice, the team has no attachment to sequential JIRA-style numbers, or commit-trailer stability across heavy rebasing matters more than pretty IDs. Default remains sequential for JIRA-familiarity; this is the documented hedge if that default proves troublesome.

## 8. Example ticket file

`.kira/tickets/01J8X8Q7RZTN5Y3VXW2A9K4E7F.md`:

```markdown
---
id: 01J8X8Q7RZTN5Y3VXW2A9K4E7F
number: KIRA-142
aliases: []
type: ticket
subtype: bug
title: "Fix race in order-book snapshot merge"
state: IN_PROGRESS
priority: P1
rank: "0|hzzzzz:"
owner: shivam
reporter: shivam
labels: [bug, orderbook]
epic: 01J8X7B1Q2W3E4R5T6Y7U8I9O0
blocked_by: [01J8X9F2M3W7VJQK8N5R6T1B0C]
links:
  relates: [01J8XB3K9P0Q2R4S6T8V0W1X2Y]
  duplicate_of: [01J8XC4M0N1P2Q3R4S5T6U7V8W]
sprint: 2026-S14
due: 2026-07-20
estimate: 3
created: 2026-07-10T09:14:00+05:30
updated: 2026-07-12T11:02:00+05:30
---

## Description

The snapshot merge path drops updates when two feed threads race on the
same price level. Repro: `bench/burst_test --dup-updates=high`.

## Acceptance criteria
- [ ] TSan clean on order_book_test
- [ ] No p99 regression on hot path

## Comments

<!-- kira:comment id=01J8XA1F6Q2N9K3M7V0R5T8B4C author=shivam ts=2026-07-11T18:30:00+05:30 -->
Confirmed repro with TSan; missing acquire fence on the consumer side.
<!-- /kira:comment -->
```

## 9. Example config

`.kira/config.yaml`:

```yaml
version: 1

project:
  key: KIRA
  name: kira

id:
  style: sequential          # sequential (default, reconciled post-merge) | hash (ULID-derived, zero reconciliation)

workflows:
  ticket:
    states:
      - { key: TODO,        category: todo }
      - { key: IN_PROGRESS, category: doing, wip: 3 }   # advisory column cap (0/absent = unlimited)
      - { key: REVIEW,      category: doing, wip: 2 }
      - { key: DONE,        category: done }
      - { key: WONT_DO,     category: done, resolution: dropped }
    initial: TODO
    transitions:
      TODO:        [IN_PROGRESS, WONT_DO]
      IN_PROGRESS: [REVIEW, TODO, WONT_DO]
      REVIEW:                                            # bare strings and guard maps may mix
        - { to: DONE, require: [resolution], set: { resolution: done } }
        - IN_PROGRESS
      DONE:        []
      WONT_DO:     []
    enforce_transitions: true
  epic:
    states:
      - { key: PLANNED, category: todo }
      - { key: ACTIVE,  category: doing }
      - { key: DONE,    category: done }
    initial: PLANNED
    transitions:
      PLANNED: [ACTIVE]
      ACTIVE:  [DONE]
      DONE:    []

labels:
  known: [bug, feature, perf, tech-debt, orderbook, infra, p0, p1, p2]
  strict: false               # true: reject unknown labels (--force overrides)

people:
  known: [shivam, alice]
  strict: false

priorities: [P0, P1, P2, P3]  # ordered high→low: validates `priority`, defines ranked sort
                              # and >/< query compare. Empty [] = free-form legacy (equality-only).
subtypes:    [bug, story, task, spike]   # validates `subtype`; [] = free-form
resolutions: [done, dropped, duplicate, cannot-reproduce]  # validates `resolution`; [] = free-form

filters:                      # named saved queries → `kira list --filter <name>`, board chips
  mine-active: "owner=shivam AND category=doing"
  blocked:     "blocked_by IS NOT EMPTY"
  overdue:     "due<2026-07-12 AND NOT category=done"

sprints:                      # scrum sprint entities; `sprint` frontmatter is a key into this list
  - { key: 2026-S13, name: "Sprint 13", start: 2026-06-29, end: 2026-07-12 }
  - { key: 2026-S14, name: "Sprint 14", start: 2026-07-13, end: 2026-07-26 }

commit:
  mode: auto                  # auto | manual | prompt
  trailer: Kira-Ticket
  # close_trailer: Kira-Closes   # v2: auto-transition to done when landed

merge:
  policy: auto                # auto | manual

sync:
  push: false                 # true: `kira sync` (no flags) also pushes after a clean doctor pass

ui:
  icons: auto                 # auto | always | never

# git:
#   scan_since: ...            # (proposed) bound the first trailer scan on a large pre-existing history

estimate:
  unit: points                # points | hours

fields: {}                    # reserved: per-type extra-field declarations (v2)
```

## 10. Validation rules

`kira validate <file>` (plumbing, called on every write and standalone by nvim's `BufWritePost`) and `kira doctor` (repo-wide sweep) enforce:

| Check | Failure mode |
|---|---|
| Frontmatter parses as the Item schema | reject: malformed YAML or wrong types |
| No unknown top-level fields | warn (or reject with `--strict`, proposed) |
| `state` ∈ this item's `type`'s configured `states[].key` | reject |
| `epic` / `blocked_by` ULIDs resolve to an existing item | reject (dangling ref) |
| `epic` chain is acyclic (walking `epic` parents from any item never revisits an item) | reject |
| `labels` ⊆ `labels.known` | warn or reject per `labels.strict` |
| `owner`/`reporter` ∈ `people.known` | warn or reject per `people.strict` |
| `priority` ∈ config `priorities` (when non-empty) | warn or reject, mirroring the labels strict/warn convention (`labels.strict` governs; free-form when `priorities` empty) |
| `subtype` ∈ config `subtypes` (when non-empty) | warn or reject per the same strict/warn convention; free-form when `subtypes` empty |
| `resolution` ∈ config `resolutions` (when non-empty) | warn or reject per the same convention; free-form when `resolutions` empty |
| `rank` is a non-empty string | reject if present-but-empty; otherwise unconstrained (free-form lexicographic key) |
| `links.<type>` ULIDs resolve to an existing item; `<type>` is a known v1 link type (`relates`, `duplicate_of`) | reject (dangling ref or unknown link type) |
| `sprint` is a `key` in config `sprints` | reject (unknown sprint) |
| `due` is a valid RFC3339 date | reject |
| transition `require:`/`set:` name known frontmatter fields; `set:` values satisfy that field's own vocab (e.g. `set: {resolution: ...}` ∈ `resolutions`) | reject in `config`/`config edit` validation |
| `created`/`updated` are valid RFC3339 | reject |
| `number` uniqueness across the union of all `number` and all `aliases` values ([§7](#7-id-scheme)) | `doctor` only — see [07-git-integration.md §4](07-git-integration.md#4-id-collision-repair-flow) for the repair flow (live-vs-live and live-vs-alias collisions both handled); every index build independently hard-warns on duplicates |
| Comment blocks well-formed (matching open/close markers, valid `id`/`ts`) | warn |

`doctor` additionally checks index freshness and installed-hooks presence — those are cross-cutting concerns, covered in [03-storage-and-git.md](03-storage-and-git.md) and [07-git-integration.md](07-git-integration.md).
