# Git integration

`commit`, `hooks`, `sync`, `resolve`, `index`, `workon`. Global flags:
[Setup and schema commands](setup-and-schema-commands.md#global-flags).

Samples: project `ACME`, cloned from a bare `origin`.

## Commit linking

kira writes and reads these markers; they are fixed, not configurable.

| Marker | Effect |
| --- | --- |
| `Kira-Ticket: KEY-n` trailer | links the commit to the item (kira's own commits and the `prepare-commit-msg` hook add it) |
| `[[KEY-n]]` in the subject | link |
| `KEY-n` leading the subject | link |
| `KEY-n` elsewhere in the message | reference (weaker; `show --json` `referenced_by`) |
| `Kira-Closes: KEY-n` trailer | moves the item to its type's first `done` state when the commit lands on `git.landed_ref` (see [`index --closes`](#--closes)) |

## `kira commit`

```
kira commit
```

Commits everything staged under `.kira/` as one commit. Only useful under
`commit.mode: manual`, where mutations stage but do not commit (under `auto`
there is never anything pending). The subject is `update {count} items` (up to
five numbers listed; personal override `commit.subject`), the body lists each
item, one `Kira-Ticket` trailer per item. A plain `git commit` works too.

```console
$ kira create ticket "Document the sync flow" --priority P3 --no-edit
Created ACME-1 Document the sync flow
$ git log -1 --format="%s%n%n%b"
ACME-1 create "Document the sync flow"

Kira-Ticket: ACME-1
$ kira commit
kira: no kira changes to commit
  hint: nothing is dirty under .kira/; mutate a ticket first
$ kira config set commit.mode manual
Set commit.mode = manual
$ kira move ACME-3 REVIEW
Moved ACME-3: IN_PROGRESS -> REVIEW
$ kira edit ACME-2 --priority P0
Edited ACME-2: priority
$ git status --short
M  .kira/tickets/01M7ANYWR578W13V1V9K3JKHZN.md
M  .kira/tickets/01M7ANYWRSFEM0BXNAJM8G51GT.md
$ kira commit
committed 5e269ec: update 2 items (2 files)
$ git log -1 --format="%s%n%n%b"
update 2 items

ACME-2 Wire up the payments retry loop
ACME-3 Cache warm-up is slow

Kira-Ticket: ACME-2
Kira-Ticket: ACME-3
```

## `kira hooks`

Hooks live in `.git/`, so **each clone runs `kira hooks install` once**.

### `kira hooks install`

| Flag | Meaning |
| --- | --- |
| `--with-pre-commit` | Also install `pre-commit`. |
| `--into-hooks-path` | Install into `core.hooksPath` even when it is inside the work tree (otherwise refused). |
| `--validate` | Report only; checks `post-merge`, `prepare-commit-msg` and the driver, treating `chained` as installed. |

| Hook | Installed | Runs |
| --- | --- | --- |
| `post-merge` | always | reconcile number collisions, refresh the index, apply landed `Kira-Closes` |
| `prepare-commit-msg` | always | add `Kira-Ticket` for the active ticket of this branch (pointer naming the branch, else the number in the branch name) |
| `pre-commit` | `--with-pre-commit` | reject a commit whose staged item files fail validation |

Each hook is a shim that exits 0 when `kira` is not on `PATH`. An existing
shell hook gets kira's block appended (`chained`) and keeps its exit status; a
non-shell hook, or one ending in `exec`/`exit`, is left alone with a reason. A
drifted block (older kira, hand-edited) is rewritten in place.

```console
$ kira hooks install --with-pre-commit
hook post-merge: chained onto existing hook
hook prepare-commit-msg: installed
hook pre-commit: installed
merge driver: registered
$ cat .git/hooks/prepare-commit-msg
#!/bin/sh
# kira prepare-commit-msg hook — delegates to the kira binary
command -v kira >/dev/null 2>&1 || exit 0
exec kira hooks run prepare-commit-msg "$@"
$ cat .git/hooks/post-merge
#!/bin/sh
echo "[house] post-merge ran"
# kira:chain v1
rc=$?
if command -v kira >/dev/null 2>&1; then
  kira hooks run post-merge "$@" || exit
fi
exit $rc
# /kira:chain
$ kira hooks install
hook post-merge: refused — .git/hooks/post-merge is an unrecognized existing hook; add kira's block by hand
hook prepare-commit-msg: installed
merge driver: registered
$ kira hooks install
hook post-merge: refused — ends in exec or exit, so appending kira's hook would never run; add kira's line by hand instead
hook prepare-commit-msg: installed
merge driver: registered
```

### The merge driver

`install` registers a clone-local driver; the attribute goes in
`.git/info/attributes`, not the tracked `.gitattributes` (which `kira init`
gives only `.kira/** text eol=lf`).

```console
$ git config --get-regexp "^merge\.kira\."
merge.kira.name kira field-level auto-merge
merge.kira.driver kira merge-file %O %A %B
$ cat .git/info/attributes
.kira/tickets/*.md merge=kira
```

It merges item files per field: a field changed on one side is taken from that
side; a field changed on both is arbitrated by policy (`merge.policy: auto`;
`manual` leaves conflict markers —
[config](../05-configuration-reference.md#merge)). Here `feature-a` set
priority P0 and owner alice, `feature-b` priority P3 and label security:

```console
$ git merge --no-edit feature-b
Auto-merging .kira/tickets/01M7ANYX1SX24C8TV8PA6NHMBN.md
Merge made by the 'ort' strategy.
 .kira/tickets/01M7ANYX1SX24C8TV8PA6NHMBN.md | 4 ++--
 1 file changed, 2 insertions(+), 2 deletions(-)
[house] post-merge ran
$ kira show ACME-4
ACME-4  Rotate the staging TLS certs  [TODO]
id:       01M7ANYX1SX24C8TV8PA6NHMBN
type:     ticket
category: todo
priority: P3
owner:    alice
labels:   security
...
```

### `kira hooks status`

One line per hook plus the driver; always exits 0. `--json` `ok` is false if
`post-merge`, `prepare-commit-msg` or the driver is missing (`pre-commit` is
optional). With `core.hooksPath` set it adds a line naming the directory.

| State | Meaning |
| --- | --- |
| `installed` | kira's current shim |
| `chained` | kira's block appended to another hook |
| `missing` | no file |
| `drifted` | invokes kira but differs from the current shim/block |
| `foreign` | a hook unrelated to kira; `install` chains onto it |

```console
$ kira hooks status
hook post-merge: chained
hook prepare-commit-msg: installed
hook pre-commit: installed
merge driver: registered
```

### `kira hooks uninstall`

Removes shims (`removed`), cuts kira's block out of chained hooks
(`unchained`), unregisters the driver. A file edited past recognition is
`left` with a warning; nothing there is `absent`.

```console
$ kira hooks uninstall
hook post-merge: unchained
hook prepare-commit-msg: removed
hook pre-commit: removed
merge driver: unregistered
$ cat .git/hooks/post-merge
#!/bin/sh
echo "[house] post-merge ran"
```

### `kira hooks run`

```
kira hooks run <hook> [args...]
```

The entry point the shims call; run it by hand to debug a hook. Outside a kira
store it does nothing and exits 0.

```console
$ kira hooks run bogus
kira: unknown hook "bogus"
  hint: kira handles: [post-merge prepare-commit-msg pre-commit]
$ kira hooks run prepare-commit-msg
kira: prepare-commit-msg needs the commit message file argument
```

## `kira sync`

```
kira sync [--commit] [--push] [--remote <name>]
```

| Step | Action |
| --- | --- |
| `prepare` | Refuse if `.kira/` has uncommitted changes; with `--commit`, commit them as `sync checkpoint`. |
| `pull` | `git pull --rebase`, field-merging item conflicts at each replayed commit (not under `merge.policy: manual`). |
| `reconcile` | Renumber display-number collisions; the item with the smaller id keeps the number. |
| `reindex` | Refresh the cache. |
| `closes` | Apply landed `Kira-Closes`. |
| `push` | With `--push` or `sync.push: true`, only after every step succeeded. |

`--remote` defaults to the branch's upstream. A conflict kira cannot resolve
(code, an unparsable item), or any conflict under `merge.policy: manual`,
aborts the rebase, leaving the branch as it was:

```console
$ kira sync
prepare:   done (working tree clean)
pull:      failed (unresolved conflicts: code.txt)
kira: rebase halted, could not auto-resolve: code.txt
$ kira sync
prepare:   done (working tree clean)
pull:      done (rebased with auto-resolved kira conflicts)
reconcile: done (no collisions)
reindex:   done (1 items)
closes:    done (no transitions)
$ kira sync
prepare:   done (working tree clean)
pull:      failed (merge.policy manual: conflicts left for you)
kira: rebase halted with conflicts (merge.policy: manual)
```

The three runs: a code conflict; an item-only conflict; the same under
`merge.policy: manual`.

Here both clones created `ACME-6`; the other clone's item is renumbered:

```console
$ kira create ticket "Trim the docker image" --priority P2 --no-edit
Created ACME-6 Trim the docker image
$ kira sync
prepare:   done (working tree clean)
pull:      done (rebased)
reconcile: done (1 renumbered)
reindex:   done (7 items)
closes:    done (no transitions)
$ kira list
NUMBER  STATE   TYPE    PRIORITY  TITLE
ACME-2  TODO    ticket  P0        Wire up the payments retry loop
ACME-3  REVIEW  ticket  P1        Cache warm-up is slow
ACME-5  TODO    ticket  P2        Add a health endpoint
ACME-6  TODO    ticket  P2        Trim the docker image
ACME-1  TODO    ticket  P3        Document the sync flow
ACME-4  TODO    ticket  P3        Rotate the staging TLS certs
ACME-7  TODO    ticket  -         Add metrics endpoint
$ kira sync
kira: uncommitted kira changes; re-run with --commit
$ kira sync --commit
prepare:   done (committed 1 paths)
pull:      done (rebased)
reconcile: done (no collisions)
reindex:   done (7 items)
closes:    done (no transitions)
$ kira sync --push
prepare:   done (working tree clean)
pull:      done (rebased)
reconcile: done (no collisions)
reindex:   done (7 items)
closes:    done (no transitions)
push:      done
```

`--json` returns `{"steps": [{"name", "status", "detail"}, ...]}`.

## `kira resolve`

```
kira resolve [<id>...] [--interactive]
```

Field-merges item files left conflicted by a merge or rebase that ran without
the driver (no `hooks install`, `merge.policy: manual`, a merge done
elsewhere), and stages them. Only unmerged item files are touched. Deleted-on-one-side
or unparsable files print `Skipped <path> (needs manual resolution)`.

```console
$ git merge --no-edit triage
Auto-merging .kira/tickets/01M7ANYWRSFEM0BXNAJM8G51GT.md
CONFLICT (content): Merge conflict in .kira/tickets/01M7ANYWRSFEM0BXNAJM8G51GT.md
Automatic merge failed; fix conflicts and then commit the result.
$ kira resolve
Resolved ACME-3 (auto-merged: priority)
$ git status --short
M  .kira/tickets/01M7ANYWRSFEM0BXNAJM8G51GT.md
$ kira resolve
No conflicted kira items to resolve
```

`auto-merged:` lists only fields both sides changed (`owner`, set on one side,
is not). `--interactive` asks per arbitrated field — `o` ours, `t` theirs,
empty for the automatic choice — and needs a terminal:

```console
$ kira resolve --interactive
kira: --interactive needs a terminal; rerun without it to auto-resolve
  hint: run in an interactive shell to pick fields by hand
```

## `kira index`

```
kira index [--full] [--closes] [--quiet]
```

Refreshes the SQLite cache in `.kira/.cache/` (git-ignored, disposable; every
command refreshes it as needed). Output `index <action> (<reason>): <n> items`;
actions `fresh`, `incremental`, `full`; reasons include `up-to-date`,
`forced`, `missing`, `head-advanced`, `working-tree-changed`,
`history-rewritten`. `--quiet` suppresses output.

```console
$ kira index
index fresh (up-to-date): 7 items
$ kira index --full
index full (forced): 7 items
$ rm -rf .kira/.cache && kira index
index full (missing): 7 items
```

### `--closes`

Scans commits that reached `git.landed_ref` since the last scan (a watermark
per ref) and moves each `Kira-Closes` item to its first `done` state. The move
is forced (passes the blocker guard, with a warning); items already done, or
updated after the closing commit, are skipped. `kira sync` and `post-merge`
run this; call it directly only when neither has.

```console
$ kira config set git.landed_ref origin/main
Set git.landed_ref = origin/main
$ git log -1 --format="%s%n%n%b"
Trim the docker image

Kira-Closes: ACME-6
$ kira index --closes
index incremental (head-advanced): 7 items
closed ACME-6 (Kira-Closes)
```

## `kira workon`

```
kira workon <id> [--no-move] [--worktree]
```

1. Creates or switches to the item's branch, named by
   `workon.branch_pattern` ([config](../05-configuration-reference.md#workon)).
2. Moves it to the type's first `doing` state, unless already in a `doing`
   state or `--no-move`.
3. Makes it the active ticket for this checkout (`kira now`, `create --here`,
   `prepare-commit-msg`).

`--worktree` (or `workon.worktree: true`) creates a git worktree at
`workon.worktree_dir` instead of switching; the active pointer is written in
the new worktree.

```console
$ kira workon ACME-5
Created branch acme/acme-5-add-a-health-endpoint for ACME-5
Moved ACME-5: TODO -> IN_PROGRESS
$ git commit -q --allow-empty -m "Add the health handler" && git log -1 --format="%s%n%n%b"
Add the health handler

Kira-Ticket: ACME-5
$ kira workon ACME-5
Switched to branch acme/acme-5-add-a-health-endpoint for ACME-5
Moved ACME-5: IN_PROGRESS -> IN_PROGRESS
$ kira workon ACME-2 --no-move
Created branch acme/acme-2-wire-up-the-payments-retry-loop for ACME-2
$ kira workon ACME-1 --worktree
Created worktree /src/acme-acme-acme-1-document-the-sync-flow on branch acme/acme-1-document-the-sync-flow for ACME-1
Moved ACME-1: TODO -> IN_PROGRESS
```

The `IN_PROGRESS -> IN_PROGRESS` line: the move decision reads the state on
the branch being left (`TODO` on `main`), and the move then lands on the
branch already in `IN_PROGRESS` — a no-op.

## New clone checklist

```sh
git clone <url> && cd <repo>
kira hooks install
```

Without it, item merges produce conflict markers, commits get no
`Kira-Ticket`, and collisions wait for `kira sync`.
