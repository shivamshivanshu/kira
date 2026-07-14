# Git Integration

**Scope:** commit-trailer linking, incremental commit scanning, installed hooks, ID collision repair, the remote/collaboration model (`kira sync`), rewrite/squash caveats, the `Kira-Closes` auto-transition trailer, the cross-repo model, and the Jira-import fidelity ceiling.
Part of the kira design set — see [DESIGN.md](../../DESIGN.md) for decisions and rationale.

## 1. Commit-linking convention

Association splits into two tiers: **linked** (primary, deliberate — one ticket per commit) and **referenced by** (secondary, weak — an incidental mention).

**Linked** — sources configured by `commit.link_markers` (list, default `[trailer, subject]`):
- `trailer`: a git **trailer**, machine-written by kira's auto-commits:
  ```
  Kira-Ticket: KIRA-142
  ```
  Repeatable; key configurable (`commit.trailer`, default `Kira-Ticket`); parsed with `git interpret-trailers --parse`, not regex-first — trailers are a structured, tool-supported git feature (same mechanism as `Signed-off-by`, `Change-Id`, Gerrit's own tooling).
- `subject`: a human marker `[[<project.key>-\d+]]` (e.g. `[[KIRA-142]]`, key never hardcoded) in the **subject line only**. At most one marker is promoted to linked (the first), and only when no trailer already links the commit — so a trailer always wins a conflict.

**Referenced by** — sources configured by `commit.reference_markers` (list, default `[bare]`):
- `bare`: a regex scan for `<project.key>-\d+` tokens anywhere in the message (subject or body, outside the trailer block), with `[[..]]` brackets *stripped* rather than skipped so any marker that wasn't promoted — a second marker, a body marker, or one demoted by a trailer conflict — falls through as an ordinary bare ref. Catches commits written before a trailer habit formed, or hand-typed. There is exactly one uniform rule: linked is the trailer or the promoted marker, referenced is the bare scan over everything else. Setting `reference_markers: []` therefore silences the entire referenced tier (bare refs and demoted markers alike), for explicit-only teams.

`kira show` lists linked commits and, when non-empty, a separate "Referenced by" section; `--json` exposes `linked_commits` and `referenced_by`.

Rejected: the founder's original freeform `KIRA TICKET: xxxxxx` string convention. Trailers are the established, tool-supported pattern for exactly this (`Signed-off-by`, `Change-Id`, gerrit's review tooling) — reusing it means `git interpret-trailers`, existing git tooling, and reviewer muscle memory all already understand the format; a bespoke string does not.

## 2. Incremental scan

Full history rescans are the thing to avoid — the scan is watermark-based per ref:

- `.cache/meta.json` stores a per-ref **watermark SHA**: the last commit already scanned for trailers on that ref.
- On each `kira index` (implicit or explicit), scan only `git log <watermark>..HEAD --pretty='%H%x00%(trailers:key=Kira-Ticket,valueonly,separator=%x01)'` (or equivalent) — O(new commits), not O(history).
- Each new commit is upserted into the index as `(ulid, sha, subject, author, ts, kind)` — resolving refs (`KIRA-142`) to `ulid` via the current number/alias table ([02-data-model.md §7](02-data-model.md#7-id-scheme)); `kind` records the linked/referenced tier.
- **Rewrite detection**: before trusting the watermark, check `git merge-base --is-ancestor <watermark> HEAD`. If false, the watermark commit is no longer an ancestor of HEAD — history was rewritten (rebase, force-push, filter-branch) — and the scan falls back to a full rescan instead of silently missing or misattributing commits.
- First-scan cost on a large pre-existing repo history is bounded by `git.scan_since` (a config-settable date/ref to start scanning from, rather than the repo's root commit) `(proposed)`.

## 3. Hooks

All hook scripts are tracked in `.kira/hooks/` (plain, reviewable, versioned like any other file). `kira hooks install` copies/symlinks them into `.git/hooks/`. Chaining onto an existing hook of the same name is fully specified, not best-effort:

- `.git/hooks/<name>` doesn't exist: kira's script is installed directly (copy/symlink).
- It exists, is a shell script (shebang check, e.g. `#!/bin/sh` / `#!/usr/bin/env bash`), and has no kira marker yet: kira **appends** a marker-bracketed block —
  ```sh
  # kira:chain v1
  .kira/hooks/<name> "$@"
  # /kira:chain
  ```
  — so the pre-existing hook still runs first, then kira's.
- It already contains the `# kira:chain v1` marker: no-op. Reinstalling is idempotent; it never double-appends.
- It exists but kira can't safely append to it (binary, symlink to something else, no recognizable shebang): `hooks install` **refuses**, printing a message that names the exact file — a documented limitation, not silent clobbering or silent skipping.

`--validate` (`kira hooks install --validate` or standalone `kira doctor`) means exactly one thing: checks installed-hook and merge-driver registration integrity — are the tracked hooks reachable from `.git/hooks/`, is the merge driver registered in `.git/config`/`.git/info/attributes` — **without modifying anything**. Installing the opt-in `pre-commit` hook is a distinct, separate action (`hooks install --with-pre-commit` `(proposed)`), kept off `--validate` so "check" and "install" never collide in one flag.

Hooks in v1:

| Hook | Action |
|---|---|
| `post-merge` | Runs `kira doctor --fix` — ID collision repair, see [§4](#4-id-collision-repair-flow). |
| `prepare-commit-msg` | Auto-appends the `Kira-Ticket` trailer, sourced from (a) the **active-ticket pointer** at `.kira/.cache/active` — a `{ticket, branch}` pair (set by `kira move X <state> --activate` or `kira workon`), honored **only when HEAD's current branch matches the recorded branch**; on mismatch it falls through — a stale pointer after a manual `git checkout` must never mis-trailer the new branch's commits — to (b) the current branch name matching a configurable pattern `(proposed)`. The pattern's casing comes from the **same config key** `workon`'s slug helper uses (one casing source — e.g. `kira/kira-142-*` — or the branch-name fallback never fires), and pattern-matched numbers resolve **through the alias table** ([02-data-model.md §7](02-data-model.md#7-id-scheme)): post-renumber branch names carry retired numbers and must keep resolving ([§4](#4-id-collision-repair-flow)). |
| `pre-commit` (opt-in, installed by `hooks install --with-pre-commit` `(proposed)`) | Runs `kira validate` on staged `.kira/` paths; blocks the commit if any staged file fails a reject-class rule ([02-data-model.md §10](02-data-model.md#10-validation-rules)). |

`kira hooks install` also registers the `auto`-policy merge driver, so plain `git merge`/`git pull` outside `kira sync` gets identical conflict-free behavior:

- `git config merge.kira.driver 'kira merge-file %O %A %B'` — local `.git/config`, not tracked.
- `.kira/tickets/*.md merge=kira` appended to `.git/info/attributes` — local, per-clone, deliberately **not** a tracked `.gitattributes` (a tracked attributes file would need every clone to already have the `kira` binary and driver registered before the attribute even means anything; local registration is the thing `hooks install` already exists to do). Scoped to `tickets/*.md` specifically, not `.kira/**/*.md`: `templates/*.md` files are plain `$EDITOR` skeletons with no Item frontmatter, so they merge via git's ordinary text merge, not the ticket-aware driver.
- Fallback: a clone that never ran `hooks install` gets plain git's default merge behavior — conflict markers can appear on kira files in that case. `kira doctor` detects an unregistered driver and nags to run `hooks install`; `kira resolve <id>` (no flags) applies the `auto` policy after the fact to clean up any markers a plain merge left behind.

## 4. ID collision repair flow

Two branches can independently allocate the same `number` (no shared counter — [02-data-model.md §7](02-data-model.md#7-id-scheme)). Repair runs automatically via the `post-merge` hook, step by step:

1. `kira doctor --fix` scans all items in `tickets/` and detects a `number` collision — two ULIDs sharing one value in the uniqueness domain, which is the **union of every item's `number` and every item's `aliases`** ([02-data-model.md §7](02-data-model.md#7-id-scheme)). This covers both live-vs-live collisions (two items with the same `number`) and live-vs-alias collisions (one item's live `number` matches a different item's retired `aliases` entry).
2. Deterministic tiebreak on ULID ordering (ULIDs are time-sortable): the **later-created** ULID is the one renumbered — the item holding the number as a live `number` in a live-vs-alias collision is never the one moved, since the alias-holder already gave that number up once.
3. That item's `number` is reassigned to the next free `n` — "next free" is computed against the same uniqueness domain, so it never reissues a number still reserved by anyone's `aliases`; its previous `number` is appended to its own `aliases` list — never dropped, never overwritten.
4. The repair itself is committed, visibly, as an ordinary kira commit:
   ```
   kira: doctor renumbered KIRA-143 -> KIRA-151
   ```
5. Any reference that still says `KIRA-143` — including trailers on commits already pushed and immutable — keeps resolving correctly forever, because `aliases` is checked in the ID resolution order ([02-data-model.md §7](02-data-model.md#7-id-scheme)).

Independent of the hook: **every index build hard-warns on duplicate numbers** by itself. If the `post-merge` hook was never installed (fresh clone, `hooks install` skipped), the failure mode is a loud, visible warning on the next `kira index`/`kira list`/etc. — never silent ambiguity between two items sharing a display number.

## 5. Remote & collaboration model

**Thesis: the remote is the git remote.** No kira server, no second remote, no new sync protocol. `.kira/` is just files traveling inside the repo, so every git collaboration mechanism transfers to tickets verbatim: fetch/pull/push, branches, PRs, protected branches, forks. Multi-user kira is multi-user git — this is the design's answer to "how do multiple people work on the same kira state," not a bolt-on. Under the default `merge.policy: auto` ([03-storage-and-git.md §7](03-storage-and-git.md#7-merge-strategy)), `kira sync` is **conflict-free by default**: no raw git conflict markers on kira files, no interactive pick required — JIRA-simple, with the git audit trail JIRA doesn't have.

### `kira sync`

One "get up to date (and optionally publish)" verb, composing the pieces from [§1](#1-commit-linking-convention)–[§4](#4-id-collision-repair-flow):

```
kira sync [--push] [--commit|--stash] [--remote <name>]   (proposed flags)
```

1. Ensure clean kira state: auto-commit staged kira paths per `commit.mode`, or force one path explicitly with `--commit` (commit first) / `--stash` (stash, pull, pop).
2. `git pull --rebase [--remote <name>]`. Under `merge.policy: auto` (default): `kira sync` detects conflicted kira paths mid-rebase, reads the three stages (base/ours/theirs) directly from the index, applies the field-level merge policy ([03-storage-and-git.md §7](03-storage-and-git.md#7-merge-strategy)), `git add`s the resolved file, and lets the rebase continue — the user is never dropped into conflict markers on a kira file. Non-kira paths, or any file under `merge.policy: manual`, still surface as an ordinary git conflict for the user to resolve by hand.
3. `kira doctor --fix` — ID reconciliation ([§4](#4-id-collision-repair-flow)).
4. Incremental reindex (full rescan if the rebase tripped the ancestor-check in [§2](#2-incremental-scan)).
5. Report: items pulled/changed, any renumbers `doctor` performed, any fields auto-merged (with the base/ours/theirs values and which one won), any conflicts still needing manual resolution (non-kira paths, or `manual` policy).

`--push` (or config `sync.push: true`) pushes after step 4 completes with a clean `doctor` pass — i.e. only once the repo is in a known-consistent state. Default remains pull-only: imitates git's own pull/push separation, publishing stays an explicit act rather than a side effect of getting up to date.

### Workflow guidance (imitates git)

| Pattern | Looks like |
|---|---|
| Trunk-based shared state | mutate tickets on `main`, `kira sync --push` for near-instant team visibility; kira commits are small and conflict-avoidant by design ([03-storage-and-git.md §7](03-storage-and-git.md#7-merge-strategy)) |
| Branch-local state | a ticket sitting `IN_PROGRESS` on a feature branch describes *that branch's* reality, not a stale copy — merging the branch merges the state change along with the code, same as any other tracked file |
| Review | ticket edits riding a PR are diffed and reviewed like any other file change — no separate ticket-review surface to build |

**Tradeoff, stated plainly**: ticket changes made on a feature branch reach the rest of the team at code-merge cadence, not instantly — because that's when the branch (and its `.kira/tickets/*.md` changes) actually reaches `main`. Teams wanting JIRA-like instant propagation mutate tickets directly on trunk and `kira sync --push`, rather than carrying ticket state on long-lived branches.

### Evaluated and deferred (v2): dedicated ticket ref

A git-bug-style dedicated ref (e.g. an orphan `refs/kira` or a standing `kira-state` branch) would sync ticket state independently of code branches, decoupling ticket propagation from code-review cadence. Rejected for v1 for three concrete reasons:

- Breaks "tickets are plain files in the working tree" — a ref you don't check out isn't greppable with `rg`, isn't hand-editable, and doesn't show up in a normal PR diff.
- Complicates checkout/branch semantics — a new contributor now needs to understand two ref namespaces instead of one.
- Doubles the merge machinery this design otherwise avoids by construction ([03-storage-and-git.md §7](03-storage-and-git.md#7-merge-strategy)) — the whole point of file-per-ticket + single-sided edges is one merge surface, not two.

Revisit if merge-cadence propagation (the tradeoff above) proves painful in practice.

## 6. Rewrite/squash caveats

- **Derived history** ([03-storage-and-git.md §5](03-storage-and-git.md#5-history)) collapses under squash-merge: if a feature branch made five separate `kira: KIRA-142 ...` commits and the branch is squash-merged to main, `kira log` on `main` sees one merged commit instead of five field-change events — the intermediate states are gone from that branch's history (though still recoverable from the feature branch/reflog while it exists).
- **Frontmatter timestamps survive squashes** — `created`/`updated` are literal file content, not git metadata, so lead-time stats (`created` → done-transition) remain accurate even when the commit-level audit trail collapses. Cycle-time granularity (time in each intermediate state) is what's lost.
- Guidance: if your team squash-merges by convention, prefer `commit.mode: manual` with fewer, more semantic kira commits over `auto`'s one-commit-per-mutation — there's less fine-grained history to lose in the first place. This trade surfaces in [DESIGN.md](../../DESIGN.md)'s open questions.

## 7. `Kira-Closes` auto-transition

A commit can drive a workflow transition — GitHub's "Closes #123", made git-native — via a second trailer:

```
Kira-Closes: KIRA-142
```

- Repeatable, like `Kira-Ticket`. Trailer key configurable (`commit.close_trailer`, default `Kira-Closes`).
- **Fires the transition only when the commit is detected reaching the configured landed ref** — config `git.landed_ref`, default the remote's default branch (e.g. `origin/HEAD` → `main`). Detection happens in the *same scan* that already processes `Kira-Ticket` trailers ([§2](#2-incremental-scan)): the per-ref watermark walk over the landed ref, run by `kira sync` and the `post-merge` hook. No new machinery — a `Kira-Closes` value seen on a newly-landed commit is resolved to its ULID and the item is transitioned.
- **Target state:** the workflow's first `done`-category state, or a per-workflow configured `close_target` when set ([02-data-model.md §10](02-data-model.md#10-validation-rules)).

**Why the landed ref, and not commit-time or push-time:**

- **Not on commit** — a commit exists the moment it's authored, on any branch, including throwaway work. "I wrote a commit mentioning KIRA-142" is not "KIRA-142 is done"; branch work isn't done work.
- **Not on branch push** — pushing a feature branch publishes work in progress, not completion. Only arrival at the landed ref (the branch that represents shipped state) means the change is actually in.
- **Squash/rebase-safe by construction** — because detection keys on the commit *reaching the landed ref*, not on the original commit object, it survives squash-merge and rebase: the merge commit (or the squashed commit) that lands on the ref carries the trailer, and the watermark walk sees it there. **Squash caveat:** a `Kira-Closes` trailer that lives only in a dropped intermediate commit is lost — squash keeps trailers from the *final* commit message. Put `Kira-Closes` in the PR/merge commit message (or the commit that will survive the squash), same discipline as [§6](#6-rewritesquash-caveats).

**Failure modes:**

- Unknown ticket (trailer value resolves to no item) → recorded, surfaced as a `kira doctor` warning; never a hard failure of the scan.
- Item already in a `done`-category state → no-op (idempotent; re-landing or a rescan won't re-fire).

**Implementation slot:** extends WP-2.2 (M2) — it rides the watermarked trailer scan that WP-2.2 already builds, adding only trailer-key config, landed-ref resolution, and the transition call. (Previously deferred to v2; the open question that blocked it — "which ref counts as landed" — is now answered by `git.landed_ref`.)

## 8. Cross-repo model

**Monorepo (supported today):** one `.kira/` at the repo root; subprojects are distinguished by `labels` or `subtype`, not by separate ticket stores. No special mechanism — it falls out of file-per-item plus the existing query grammar.

**Polyrepo federation (v2 stance):** config lists sibling repo paths; foreign ULIDs are resolved **read-only** when the sibling repo is present locally, and left as opaque IDs when it is not. Links stay ULID-based, so a cross-repo link degrades gracefully to an unresolved identifier rather than breaking — the link is never rewritten, only its resolution is best-effort.

```yaml
# v2 — cross-repo federation (not implemented in v1)
federation:
  siblings:
    - ../platform-core      # path to a sibling kira repo
    - ../shared-libs
  resolve: read-only        # foreign ULIDs resolve when present, opaque when absent
```

## 9. Jira-import fidelity ceiling

kira history is **derived** from `git log` ([03-storage-and-git.md §5](03-storage-and-git.md#5-history)), so there is no place to replay a foreign changelog into — a Jira issue's per-field change history cannot be reconstructed as kira events. The import contract is therefore lossy by design:

- **Current-state frontmatter** — status, assignee, priority, labels, etc. mapped to kira fields as of import time.
- **Comments** — brought over as append blocks ([02-data-model.md](02-data-model.md)), preserving author and timestamp in the block text.
- **One synthetic comment** carrying the *flattened* Jira changelog (a human-readable transcript of transitions/edits), since it can't become real kira history.
- **Attachments** — require the attachments pattern (`.kira/attachments/<ulid>/`, see [DESIGN.md §3](../../DESIGN.md#3-constraints-and-non-goals)); without it they are dropped and listed in an import manifest rather than silently lost.

This is the **M6 import contract**, stated up front so the ceiling is a documented decision, not a surprise at implementation time.

## 10. Deferred v2

Superseded, no longer deferred: the list-field set-union merge driver is now the v1 `auto` merge policy ([03-storage-and-git.md §7](03-storage-and-git.md#7-merge-strategy)) — the founder's JIRA-simplicity requirement (no raw conflict markers on kira files) means auto-resolution can't wait for v2. The original deferral reasons (per-clone `.gitattributes` registration burden, opaque mid-merge rewriting) no longer hold: `kira hooks install` is already the per-clone step everything else in this doc rides on, and the "opaque rewriting" concern is now the explicit goal (invisible-to-the-user resolution), not a downside.

Also no longer deferred: the `Kira-Closes` auto-transition trailer, now specced for M2 in [§7](#7-kira-closes-auto-transition) — its blocking open question (which ref counts as "landed") is resolved by the `git.landed_ref` config.
