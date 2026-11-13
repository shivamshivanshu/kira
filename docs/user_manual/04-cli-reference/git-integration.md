# Git integration

Tickets are tracked files, so git already moves them between machines. These
commands are the layer that makes that work well: they teach git how to merge a
ticket file field by field, keep display numbers unique when two people file
tickets at once, stamp commits with the ticket they belong to, and give you a
per-ticket branch workflow.

Read this page in three parts:

1. **[`kira commit`](#kira-commit) and `commit.mode`** — when a kira mutation
   becomes a git commit.
2. **[`kira hooks`](#kira-hooks)** — the one-time setup that wires kira into
   `git merge`, `git pull`, and `git commit`.
3. **[`kira sync`](#kira-sync), [`kira resolve`](#kira-resolve),
   [`kira index`](#kira-index), [`kira workon`](#kira-workon)** — the commands
   you use day to day.

All commands accept the global flags `-C`, `--json` and `--no-color`; see the
global flags reference.

---

## `kira commit`

Commits everything currently pending under `.kira/` as one git commit. Whether
you ever need it depends entirely on `commit.mode`.

```
Usage: kira commit [flags]
```

It takes no flags of its own.

### `commit.mode`

`commit.mode` is a config key in `.kira/config.yaml`, not a command-line flag.
It decides what happens to the git working tree after any kira mutation —
`create`, `edit`, `move`, `label`, `link`, and so on.

| Mode | What a mutation does | Is `kira commit` useful? |
| --- | --- | --- |
| `auto` (default) | Stages **and commits** the change immediately, as its own commit. | No — there is never anything pending. |
| `manual` | Stages the change and stops. You commit when you like. | Yes — this is the mode it exists for. |

kira's own repository ships `commit.mode: auto`, and its `.kira/config.yaml`
records it like this:

```yaml
commit:
  mode: auto                  # auto | manual
```

Under `auto`, each mutation lands as a separate commit with a `Kira-Ticket`
trailer:

```console
$ kira create ticket "Document the sync flow" --priority P3 --no-edit
Created ACME-3 Document the sync flow

$ git log -1 --format='%s%n%n%b'
ACME-3 create "Document the sync flow"

Kira-Ticket: ACME-3
```

Asking for a commit when nothing is pending is an error, which is the normal
state of affairs under `auto`:

```console
$ kira commit
kira: no kira changes to commit
  hint: nothing is dirty under .kira/; mutate a ticket first
```

### Batching mutations with `manual`

Switch to `manual` when you would rather review a batch of ticket edits as a
single commit — grooming a backlog, say, or a bulk retriage.

```console
$ kira config set commit.mode manual
Set commit.mode = manual

$ kira move ACME-2 REVIEW
Moved ACME-2: IN_PROGRESS -> REVIEW

$ kira edit ACME-1 --priority P0
Edited ACME-1: priority

$ git status --short
M  .kira/tickets/01KYHXE2ARM4TR3BWNJT7JM8TA.md
M  .kira/tickets/01KYHXE2CDF3YNRE012YE0NZ2R.md
```

Both files are staged but uncommitted. `kira commit` closes the batch, writing
a subject that counts the items and a trailer per ticket:

```console
$ kira commit
committed 2af17bd: update 2 items (2 files)

$ git log -1 --format='%s%n%n%b'
update 2 items

ACME-1 Wire up the payments retry loop
ACME-2 Cache warm-up is slow on cold start

Kira-Ticket: ACME-1
Kira-Ticket: ACME-2
```

You can equally run plain `git commit` yourself; `kira commit` is a
convenience that generates the subject, body, and trailers for you. Note that
kira has already *staged* the files, so a bare `git commit` (no `-a`, no
pathspec) picks them up.

See [Configuration reference](../05-configuration-reference.md) for the
`commit:` block and the fixed `Kira-Ticket`/`Kira-Closes` trailers.

---

## `kira hooks`

`kira hooks` is the one-time setup that connects kira to git's own machinery.
It does two independent things, and the second is the more important one:

- It installs **git hook shims** so that merging, and committing, run kira logic
  at the right moment.
- It registers a **git merge driver** for ticket files, so that plain
  `git merge` and `git pull` merge tickets field by field instead of dumping
  `<<<<<<<` markers into YAML.

Hooks live in `.git/`, which is not part of the repository, so **every clone
needs its own `kira hooks install`.** Cloning a repo where a colleague ran it
does not install it for you.

### `kira hooks install`

```
Usage: kira hooks install [flags]
```

| Flag | Meaning |
| --- | --- |
| `--into-hooks-path` | Install into the `core.hooksPath` directory even when it lives inside the work tree. Required if the repo redirects hooks; otherwise `install` refuses rather than writing somewhere you did not expect. |
| `--validate` | Verify installed hooks and merge-driver registration and report; change nothing. |
| `--with-pre-commit` | Additionally install the `pre-commit` validation hook. |

Two hooks are installed always, and a third only on request:

| Hook | Installed | What kira does in it |
| --- | --- | --- |
| `post-merge` | always | Repair display-number collisions the merge introduced, then refresh the index and apply any `Kira-Closes` transitions the merge landed. |
| `prepare-commit-msg` | always | Append the `Kira-Ticket` trailer for the ticket the current branch belongs to. |
| `pre-commit` | `--with-pre-commit` | Reject a commit whose staged ticket files are invalid. |

A run in a fresh clone:

```console
$ kira hooks install --with-pre-commit
hook post-merge: installed
hook prepare-commit-msg: installed
hook pre-commit: installed
merge driver: registered
```

Each installed hook is a four-line shim that delegates to the kira binary:

```console
$ cat .git/hooks/prepare-commit-msg
#!/bin/sh
# kira prepare-commit-msg hook — delegates to the kira binary
command -v kira >/dev/null 2>&1 || exit 0
exec kira hooks run prepare-commit-msg "$@"
```

The `command -v kira || exit 0` guard is deliberate. A teammate who has the
repository but not the kira binary — or you, inside a container that does not
ship it — still gets working `git merge` and `git commit`; the hook simply does
nothing. kira never makes git unusable by being absent.

### Chaining onto hooks you already have

`install` never clobbers an existing hook. If `.git/hooks/post-merge` already
exists and is a shell script, kira appends a delimited block to the end of it:

```console
$ cat .git/hooks/post-merge
#!/bin/sh
echo "[house] post-merge ran"

$ kira hooks install
hook post-merge: chained onto existing hook
hook prepare-commit-msg: installed
merge driver: registered

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
```

Both then run, in order — the house hook first, kira second, and the original
hook's exit status is preserved:

```console
$ git merge --no-edit feature-b
[house] post-merge ran
Auto-merging .kira/tickets/01KYHXEZGZF09V18RD96CXH9GG.md
Merge made by the 'ort' strategy.
```

kira declines to chain in three cases, and says which instead of guessing:

```console
$ kira hooks install          # existing post-merge is a python script
hook post-merge: refused — .git/hooks/post-merge is an unrecognized existing hook; add kira's block by hand

$ kira hooks install          # existing post-merge ends in `exec /usr/bin/true`
hook post-merge: refused — ends in exec or exit, so appending kira's hook would never run; add kira's line by hand instead

$ kira hooks install          # existing post-merge was hand-edited to call kira
hook post-merge: refused — it already runs kira alongside other commands; remove kira's lines and re-run `kira hooks install`, or keep managing it by hand
```

An appended `sh` block cannot run inside a script with a non-shell interpreter,
and cannot run at all after an `exec` or `exit`. Rather than write something
that silently never fires, kira leaves the file alone and tells you to place
its line yourself. The other hooks and the merge driver are still installed.

### The merge driver

This is what makes `git merge` and `git pull` behave sensibly on ticket files.
`install` writes two pieces of local git configuration:

```console
$ git config --get-regexp '^merge\.kira\.'
merge.kira.name kira field-level auto-merge
merge.kira.driver kira merge-file %O %A %B

$ cat .git/info/attributes
.kira/tickets/*.md merge=kira
```

The attribute goes in `.git/info/attributes` — clone-local, not the tracked
`.gitattributes` — because the driver is only usable in a clone that has the
kira binary and has run `install`. (`kira init` does write one tracked
`.gitattributes` line, `.kira/** text eol=lf`, which is about line endings and
unrelated to merging.)

With the driver registered, git hands kira the base, ours, and theirs versions
of any conflicting ticket file and takes back a merged result. kira merges
**per field** rather than per line, which is the whole point:

Two branches both touch `ACME-4`: `feature-a` sets `priority: P0` and
`owner: alice`, `feature-b` sets `priority: P3` and adds the `security` label.
Merging the first is uneventful; merging the second is where the two sets of
edits meet.

```console
$ git merge --no-edit feature-b
[house] post-merge ran
Auto-merging .kira/tickets/01KYHXEZGZF09V18RD96CXH9GG.md
Merge made by the 'ort' strategy.
 .kira/tickets/01KYHXEZGZF09V18RD96CXH9GG.md | 4 ++--
 1 file changed, 2 insertions(+), 2 deletions(-)
```

No conflict, and a working tree that `git status --short` reports as clean. The
merged ticket carries edits from both sides:

```console
$ kira show ACME-4
ACME-4  Rotate the staging TLS certs  [TODO]
id:        01KYHXEZGZF09V18RD96CXH9GG
type:      ticket
category:  todo
priority:  P3
owner:     alice
labels:    security
created:   2026-07-27T19:20:36+05:30
updated:   2026-07-27T19:21:54+05:30
```

Fields only one side touched (`owner`, `labels`) are taken from that side.
`priority`, which both sides changed, is *arbitrated* by a deterministic policy
rather than reported as a conflict — this is the `merge.policy: auto` default.
Set `merge.policy: manual` in config to turn that off and get ordinary text
conflict markers instead; see
[Configuration reference](../05-configuration-reference.md).

### `kira hooks status`

```
Usage: kira hooks status [flags]
```

Reports one line per hook plus the merge driver. No flags.

```console
$ kira hooks status
hook post-merge: chained
hook prepare-commit-msg: installed
hook pre-commit: missing
merge driver: registered
```

| State | Meaning |
| --- | --- |
| `installed` | The file is byte-for-byte kira's current shim. |
| `chained` | kira's `# kira:chain v1` block is appended to someone else's hook. Working correctly. |
| `missing` | No such hook file. Expected for `pre-commit` unless you asked for it. |
| `drifted` | The file invokes kira, but does not match the current shim or chain block — hand-edited, or written by an older kira. |
| `foreign` | A hook exists but has nothing to do with kira. Run `install` to chain onto it. |

A repository that redirects hooks reports where they live:

```console
$ kira hooks status
hook post-merge: installed
hook prepare-commit-msg: installed
hook pre-commit: installed
merge driver: registered
core.hooksPath: .githooks (hooks live there)
```

`status` exits 0 whatever it finds; it is a report, not a gate. To gate on the
result, read the `ok` field of `--json`:

```console
$ kira hooks status --json
{
  "hooks": [
    { "name": "post-merge", "state": "missing" },
    { "name": "prepare-commit-msg", "state": "missing" },
    { "name": "pre-commit", "state": "missing" }
  ],
  "merge_driver": false,
  "ok": false
}
```

`ok` tolerates a missing `pre-commit` — that hook is opt-in — but not a missing
`post-merge`, `prepare-commit-msg`, or merge driver.

`kira hooks install --validate` is a narrower check covering only the two
always-installed hooks and the driver, and it treats `chained` as installed:

```console
$ kira hooks install --validate
hook post-merge: installed
hook prepare-commit-msg: installed
merge driver: registered
```

### `kira hooks uninstall`

```
Usage: kira hooks uninstall [flags]
```

Removes kira's shims, restores whatever it chained onto, and unregisters the
merge driver. No flags.

```console
$ kira hooks uninstall
hook post-merge: unchained
hook prepare-commit-msg: removed
hook pre-commit: removed
merge driver: unregistered
```

| Reported state | Meaning |
| --- | --- |
| `removed` | The file was purely kira's shim; deleted. |
| `unchained` | kira's block was cut out; the rest of the file was left alone. |
| `absent` | Nothing was there. |
| `left` | Kept — the file was hand-edited past recognition. A warning names it so you can clean it up yourself. |

The chained hook comes back exactly as it was, and the driver's configuration
is gone:

```console
$ cat .git/hooks/post-merge
#!/bin/sh
echo "[house] post-merge ran"

$ git config --get-regexp '^merge\.kira\.'      # no output

$ cat .git/info/attributes                       # no output
```

### `kira hooks run`

```
Usage: kira hooks run <hook> [args...] [flags]
```

Plumbing. This is the entry point the installed shims call; you would not
normally type it. It is worth knowing about for two reasons: it names the hooks
kira handles, and it lets you run a hook's logic directly when you are working
out why a hook did not seem to fire.

```console
$ kira hooks run post-merge          # reconcile + reindex + apply landed closes

$ kira hooks run pre-commit          # validate what is staged right now
kira: invalid items: .kira/tickets/01KYHXFGMQW5N0VWEXP5VG04BW.md: field "state": "BOGUS_STATE" is not a state in the ticket workflow

$ kira hooks run bogus
kira: unknown hook "bogus"
  hint: kira handles: [post-merge prepare-commit-msg pre-commit]
```

`prepare-commit-msg` needs the message-file argument git would have passed:

```console
$ kira hooks run prepare-commit-msg
kira: prepare-commit-msg needs the commit message file argument
```

Running kira from a store-less directory is not an error here — the hook
silently does nothing, so a shim in a repo that has no `.kira/` never breaks
`git commit`.

### The `Kira-Ticket` trailer, in practice

Once `prepare-commit-msg` is installed, your ordinary code commits get stamped
with the ticket automatically. On a ticket branch:

```console
$ git commit -m "Add the dark palette"

$ git log -1 --format='%s%n%n%b'
Add the dark palette

Kira-Ticket: WEB-1
```

kira decides which ticket that is by looking at the current branch: it uses the
active-ticket pointer if that pointer names this branch, and otherwise infers
the ticket from a branch name that contains a ticket reference. On `main`, or
any branch that matches neither, no trailer is added.

---

## `kira sync`

Brings your clone up to date with the remote and repairs anything the update
broke, in one command. It is the everyday replacement for `git pull` in a repo
with kira tickets.

```
Usage: kira sync [flags]
```

| Flag | Argument | Meaning |
| --- | --- | --- |
| `--commit` | — | Commit dirty paths under `.kira/` before pulling. |
| `--push` | — | Push to the remote after a clean sync. |
| `--remote` | name | Remote to sync with. Defaults to the branch's upstream. |

The steps, in order, one report line each:

| Step | What it does |
| --- | --- |
| `prepare` | Refuse a dirty `.kira/` working tree, or commit it with `--commit`. |
| `pull` | `git pull --rebase`, auto-resolving kira conflicts as the rebase replays. |
| `reconcile` | Repair display-number collisions the pull introduced. |
| `reindex` | Refresh the local cache index. |
| `closes` | Apply `Kira-Closes` transitions for commits that just landed. |
| `push` | Only with `--push`, or `sync.push: true` in config. |

A quiet run:

```console
$ kira sync
prepare:   done (working tree clean)
pull:      done (rebased)
reconcile: done (no collisions)
reindex:   done (4 items)
closes:    done (no transitions)
```

### Two people, one ticket number

`reconcile` is the step you will not find in `git pull`. Display numbers are
allocated locally, so two people filing a ticket at the same time both mint the
same next number. Nothing detects that until the branches meet.

Here both clones created `ACME-5` independently, and the second one to sync
gets renumbered:

```console
$ kira create ticket "Trim the docker image" --priority P2 --no-edit
Created ACME-5 Trim the docker image

$ kira sync
prepare:   done (working tree clean)
pull:      done (rebased)
reconcile: done (1 renumbered)
reindex:   done (6 items)
closes:    done (no transitions)

$ kira list
NUMBER  STATE        TYPE    PRIORITY  TITLE
ACME-1  TODO         ticket  P1        Wire up the payments retry loop
ACME-4  TODO         ticket  P1        Rotate the staging TLS certs
ACME-2  IN_PROGRESS  ticket  P2        Cache warm-up is slow on cold start
ACME-5  TODO         ticket  P2        Add a health endpoint
ACME-6  TODO         ticket  P2        Trim the docker image
ACME-3  TODO         ticket  P3        Document the sync flow
```

The remote's `ACME-5` keeps the number; the local one becomes `ACME-6`. The
ticket's immutable id never changes — only the human-facing number does — so
nothing that referenced it by id breaks. Renumbering is deterministic: every
clone reconciling the same history arrives at the same answer.

The `post-merge` hook runs the same reconcile step, which is why a plain
`git pull` or `git merge` in a repo with hooks installed also repairs
collisions. `kira sync` does it whether or not hooks are installed.

### A dirty working tree

If anything under `.kira/` is uncommitted when you sync, `prepare` has to do
something with it — a rebase over dirty files will not start. kira refuses
rather than guess whether the changes are an accident of timing or a batch you
are still assembling:

```console
$ kira sync
kira: uncommitted kira changes; re-run with --commit
```

`--commit` commits them first as a checkpoint, then pulls:

```console
$ kira sync --commit
prepare:   done (committed 1 paths)
pull:      done (rebased)
reconcile: done (no collisions)
reindex:   done (6 items)
closes:    done (no transitions)

$ git log -1 --format='%s'
sync checkpoint
```

### Conflicts during the pull

`sync` rebases. If a replayed commit conflicts on a ticket file, kira resolves
it in place and continues the rebase, repeating until the rebase finishes:

```console
pull:      done (rebased with auto-resolved kira conflicts)
```

If anything is still unmerged after that pass — a conflict in your *code*, or a
ticket file kira cannot parse — the rebase is aborted rather than left
half-done, and the offending paths are named:

```console
pull:      failed (unresolved conflicts: src/server.go)
```

You are back where you started; resolve it with an ordinary `git pull --rebase`
and your usual conflict tooling. With `merge.policy: manual`, `sync` does not
attempt auto-resolution at all and aborts immediately:

```console
pull:      failed (merge.policy manual: conflicts left for you)
```

### Publishing

```console
$ kira sync --push
prepare:   done (working tree clean)
pull:      done (rebased)
reconcile: done (no collisions)
reindex:   done (4 items)
closes:    done (no transitions)
push:      done
```

Set `sync.push: true` in config to make that the default and drop the flag.
`--push` only runs after every earlier step succeeded, so a sync that hit a
conflict never publishes a half-repaired state.

`--json` returns the same steps as structured data, which is the useful form in
a script:

```console
$ kira sync --json
{
  "steps": [
    { "name": "prepare",   "status": "done", "detail": "working tree clean" },
    { "name": "pull",      "status": "done", "detail": "rebased" },
    { "name": "reconcile", "status": "done", "detail": "no collisions" },
    { "name": "reindex",   "status": "done", "detail": "6 items" },
    { "name": "closes",    "status": "done", "detail": "no transitions" }
  ]
}
```

---

## `kira resolve`

Resolves ticket files that a merge or rebase left conflicted, arbitrating each
conflicting field instead of making you edit YAML around `<<<<<<<` markers.

```
Usage: kira resolve [id...] [flags]
```

| Flag | Meaning |
| --- | --- |
| `--interactive` | Pick each conflicting field by hand. Requires a terminal. |

Positional ids narrow it to specific tickets; with none, it takes every
conflicted ticket file in the merge.

### When you need it

If the merge driver is registered, you usually do not — the driver resolves
ticket conflicts during the merge itself and you never see them. `kira resolve`
is for the cases where the driver did not run:

- A clone where nobody ran `kira hooks install`. This is the common one.
- `merge.policy: manual`, which deliberately produces text conflicts.
- A merge performed by something other than your local git — a web UI, a CI
  job, a server-side merge.

It only ever touches paths git reports as unmerged, and only the ones that are
ticket files. Conflicts in your code are left entirely alone.

### A full walkthrough

A clone with no hooks installed, and two branches that both changed `ACME-2`:

```console
$ kira hooks status
hook post-merge: missing
hook prepare-commit-msg: missing
hook pre-commit: missing
merge driver: not registered (run `kira hooks install`)

$ git merge --no-edit triage
Auto-merging .kira/tickets/01KYHXE2CDF3YNRE012YE0NZ2R.md
CONFLICT (content): Merge conflict in .kira/tickets/01KYHXE2CDF3YNRE012YE0NZ2R.md
Automatic merge failed; fix conflicts and then commit the result.
```

Which leaves exactly the mess kira exists to avoid — YAML with conflict markers
in it:

```console
$ head -21 .kira/tickets/01KYHXE2CDF3YNRE012YE0NZ2R.md
---
id: 01KYHXE2CDF3YNRE012YE0NZ2R
number: ACME-2
aliases: []
type: ticket
subtype: bug
title: "Cache warm-up is slow on cold start"
state: IN_PROGRESS
<<<<<<< HEAD
priority: P0
||||||| 91312bf
priority: P2
=======
priority: P1
owner: carol
>>>>>>> triage
labels: []
epic: null
blocked_by: []
created: 2026-07-27T19:20:06+05:30
updated: 2026-07-27T19:22:11+05:30
---
```

`kira resolve` reads the three staged versions of the file out of the index —
base, ours, theirs — merges them field by field, and stages the result:

```console
$ kira resolve
Resolved ACME-2 (auto-merged: priority)

$ git status --short
M  .kira/tickets/01KYHXE2CDF3YNRE012YE0NZ2R.md
```

The `auto-merged:` list names the fields that genuinely conflicted and had to
be arbitrated. `owner` is not in it: only one side set it, so keeping it was not
a decision. Only `priority` was contested.

```console
$ head -16 .kira/tickets/01KYHXE2CDF3YNRE012YE0NZ2R.md
---
id: 01KYHXE2CDF3YNRE012YE0NZ2R
number: ACME-2
aliases: []
type: ticket
subtype: bug
title: "Cache warm-up is slow on cold start"
state: IN_PROGRESS
priority: P1
owner: carol
labels: []
epic: null
blocked_by: []
created: 2026-07-27T19:20:06+05:30
updated: 2026-07-27T19:22:11+05:30
---
```

The file is staged, so the merge finishes normally:

```console
$ git commit --no-edit
$ git log --oneline -1
3ea3b38 Merge branch 'triage'
```

If nothing is conflicted, it says so and exits 0 — safe to run speculatively:

```console
$ kira resolve
No conflicted kira items to resolve
```

Files it cannot handle — one side deleted the ticket, or the content will not
parse as a kira item — are reported on stderr as
`Skipped <path> (needs manual resolution)` and left for you.

### `--interactive`

When you want to make the calls yourself, `--interactive` prompts once per
arbitrated field. Each prompt shows both candidate values; `o` takes ours, `t`
takes theirs, and empty input accepts the automatic choice.

```console
$ kira resolve --interactive
title: [o]urs=Document the sync and resolve flow [t]heirs=Write the sync runbook (default: auto) o
priority: [o]urs=P0 [t]heirs=P2 (default: auto) t
Resolved ACME-3 (auto-merged: title, priority)
```

The result takes `title` from ours and `priority` from theirs, exactly as
chosen:

```console
$ head -8 .kira/tickets/01KYHXE2DWKHYPWP856W7PE2YR.md
---
id: 01KYHXE2DWKHYPWP856W7PE2YR
number: ACME-3
aliases: []
type: ticket
title: "Document the sync and resolve flow"
state: TODO
priority: P2
```

You are only asked about fields that actually conflict — a merge with one
contested field asks one question. Without a terminal the flag is rejected up
front rather than silently ignored:

```console
$ kira resolve --interactive
kira: --interactive needs a terminal; rerun without it to auto-resolve
  hint: run in an interactive shell to pick fields by hand
```

---

## `kira index`

Refreshes the local cache index kira keeps under `.kira/.cache/`.

```
Usage: kira index [flags]
```

| Flag | Meaning |
| --- | --- |
| `--closes` | Apply `Kira-Closes` trailer transitions for commits that have landed on `git.landed_ref`. |
| `--full` | Force a complete rebuild instead of an incremental refresh. |
| `--quiet` | Suppress the summary and closed-ticket lines. |

The index is a **disposable derived cache** — a SQLite database built from the
ticket files, used to make `list`, `show`, `board`, and friends fast on large
repositories. It is never a source of truth. The Markdown files are. You can
delete `.kira/.cache/` at any moment and the next command rebuilds it; it is
git-ignored (via `.kira/.gitignore`) and never committed.

Commands that need the index refresh it themselves, so running `kira index` by
hand is rare — it is mostly a way to force the work to happen now, or to
recover if something looks stale.

```console
$ kira index
index fresh (up-to-date): 6 items

$ kira index --full
index full (forced): 6 items
```

The output is `index <action> (<reason>): <n> items`. Actions are `fresh` (no
work needed), `incremental` (only changed items reread), and `full` (rebuilt
from scratch). Reasons include `up-to-date`, `forced`, `missing`,
`head-advanced`, `working-tree-changed`, and `history-rewritten` — the last
being what forces a full rebuild after a rebase or amend rewrites history the
cache was keyed on.

### `--closes`

A code commit can close a ticket by carrying a trailer:

```console
$ git commit -m "Trim the docker image

Kira-Closes: ACME-6"
```

Nothing happens at commit time — the work is not done until it lands. When the
commit reaches the ref configured as `git.landed_ref` — defaulting to whatever
branch `origin/HEAD` points at, usually `origin/main`, and falling back to
`main` — `--closes` walks the newly landed commits and moves each referenced
ticket to its done state:

```console
$ kira index --closes
index incremental (head-advanced): 6 items
closed ACME-6 (Kira-Closes)
```

A ticket in `TODO` goes straight to `DONE` without walking `IN_PROGRESS` and
`REVIEW` — an ordinary move, since any state may reach any other. The move is
forced, so it also closes a ticket whose blockers are still open, and kira
warns when that happens. Tickets already done are skipped, as are tickets
edited *after* the closing commit was made, on the assumption you reopened them
deliberately. A watermark is recorded so the same commits are never scanned
twice.

**You rarely run this yourself.** `kira sync` runs it as its `closes` step, and
so does the `post-merge` hook. Both are how a ticket closes itself when the
branch that fixes it merges. Reach for `kira index --closes` directly only when
neither has had a chance to run.

The trailer key is `Kira-Closes`, distinct from `Kira-Ticket`, which merely
links a commit to a ticket without changing its state.

---

## `kira workon`

Starts work on a ticket: creates or switches to its branch, moves it into a
doing state, and marks it as your active ticket.

```
Usage: kira workon <id> [flags]
```

| Flag | Meaning |
| --- | --- |
| `--no-move` | Do not transition the ticket into a doing state. |
| `--worktree` | Create a dedicated git worktree instead of switching branches in place. |

Three things happen, and the second and third are what distinguish it from
`git checkout -b`:

```console
$ kira workon WEB-1
Created branch web/web-1-dark-mode-toggle for WEB-1
Moved WEB-1: TODO -> IN_PROGRESS

$ git status -sb
## web/web-1-dark-mode-toggle
```

1. **The branch.** Named from `workon.branch_pattern` (default
   `{key}/{number}-{slug}`) with words joined by `-` — so `WEB-1 Dark mode
   toggle` becomes `web/web-1-dark-mode-toggle`. See
   [Configuration reference](../05-configuration-reference.md) for the key.
2. **The state.** The ticket moves to the first state in the `doing` category
   its workflow declares. A ticket already in a doing state is left alone.
3. **The active pointer.** The ticket becomes *active* in this clone, which is
   what `kira now` reports, what `kira create --here` files against, and what
   the `prepare-commit-msg` hook stamps commits with.

```console
$ kira now
WEB-1  Dark mode toggle  [IN_PROGRESS]
in state:    just now
branch:      web/web-1-dark-mode-toggle
commits:     0 since last state change
```

And with the hook installed, ordinary commits on that branch pick up the ticket
automatically:

```console
$ git commit -m "Add the dark palette"

$ git log -1 --format='%s%n%n%b'
Add the dark palette

Kira-Ticket: WEB-1
```

Running `workon` again for a ticket whose branch already exists switches to it
rather than failing:

```console
$ kira workon WEB-1
Switched to branch web/web-1-dark-mode-toggle for WEB-1
Moved WEB-1: IN_PROGRESS -> IN_PROGRESS
```

The second line is a quirk worth recognizing: kira decided whether to move the
ticket from the state visible on the branch you were *leaving* (`TODO` on
`main`), then performed the move on the branch you arrived at, where the ticket
was already `IN_PROGRESS`. Nothing changed; it is a no-op re-assertion, not a
second transition.

The branch pattern is honoured, so a repo that prefers flat names gets them:

```console
$ kira config set workon.branch_pattern "{key}-{number}"
$ kira workon WEB-5
Created branch web-web-5 for WEB-5
```

### `--no-move`

Take the branch without advancing the ticket. Useful when you are opening a
branch to investigate rather than committing to the work:

```console
$ kira workon WEB-2 --no-move
Created branch web/web-2-rate-limit-the-public-api for WEB-2

$ kira list
NUMBER  STATE  TYPE    PRIORITY  TITLE
WEB-2   TODO   ticket  P0        Rate-limit the public API
WEB-1   TODO   ticket  P1        Dark mode toggle
```

The branch and the active pointer are still set up; only the state transition is
skipped.

### `--worktree`

Checks the branch out in a separate git worktree instead of switching your
current one — so you can work two tickets at once without stashing, and keep a
long build in one directory while you edit another.

```console
$ kira workon ACME-3 --worktree
Created worktree /tmp/kira-docs-git/acme-acme-acme-3-document-the-sync-flow on branch acme/acme-3-document-the-sync-flow for ACME-3
Moved ACME-3: TODO -> IN_PROGRESS

$ git worktree list
/tmp/kira-docs-git/acme                                     beb599c [main]
/tmp/kira-docs-git/acme-acme-acme-3-document-the-sync-flow  14ad980 [acme/acme-3-document-the-sync-flow]
```

The worktree is created beside the repository, named from the repository
directory and the branch. Your original working directory is untouched — note
it is still on `main` above. The active pointer is written **inside the new
worktree**, so `kira now` reports the ticket there and continues to report
whatever was active in the original directory here.

Set `workon.worktree: true` in config to make this the default without the
flag, and `workon.worktree_dir` to control where worktrees are placed; see
[Configuration reference](../05-configuration-reference.md).

---

## Setting up a new clone

Putting it together, the two commands after cloning a repository that uses
kira:

```console
$ git clone git@example.com:team/service.git
$ cd service
$ kira hooks install
hook post-merge: installed
hook prepare-commit-msg: installed
merge driver: registered

$ kira index
index full (missing): 2 items
```

`hooks install` is the one that matters — without it, ticket merges produce raw
conflict markers, commits go unstamped, and number collisions go unrepaired
until someone runs `kira sync`. `kira index` is optional; the first command
that needs the cache builds it anyway.
