# Setup and schema commands

This page covers the commands that stand up a kira project and describe it to
other tools: `init`, `config`, `schema`, `validate`, `version`, and
`completion`.

It also documents the **global flags** that every kira command accepts. Other
pages in this reference link back here rather than repeating them.

All transcripts below were captured from a scratch repository called `acme`,
initialized with the project key `ACME`.

## Global flags

These three flags are defined on the root command, so they are valid on every
subcommand and may appear before or after the subcommand name.

| Flag | Description |
| --- | --- |
| `-C, --C <path>` | Run as if kira had been invoked from `<path>`. |
| `--json` | Emit machine-readable JSON on stdout instead of human-readable text. |
| `--no-color` | Disable ANSI color in human-readable output. |

### `-C <path>` — run from elsewhere

`-C` changes where kira begins looking for the project. Repo-root discovery
still walks *up* from that path, so you may point it at any directory inside
the repository, not only the root:

```console
$ cd /tmp
$ kira -C ~/src/acme list
NUMBER  STATE  TYPE    PRIORITY  TITLE
ACME-1  TODO   ticket  P1        Retry failed webhook deliveries
ACME-2  TODO   ticket  P2        Add idempotency keys

$ kira -C ~/src/acme/src/webhooks list
NUMBER  STATE  TYPE    PRIORITY  TITLE
ACME-1  TODO   ticket  P1        Retry failed webhook deliveries
ACME-2  TODO   ticket  P2        Add idempotency keys
```

Without it, kira searches upward from the current working directory and fails
if no project is found:

```console
$ cd /tmp && kira list
kira: no .kira/ found in "/tmp" or any parent directory
  hint: run `kira init` to create one here
$ echo $?
3
```

This makes `-C` the flag to reach for in scripts, cron jobs, and editor
integrations that do not control their own working directory.

### `--json` — machine-readable output

Every command that produces output has a documented JSON shape. Passing
`--json` prints that shape on stdout in place of the human rendering:

```console
$ kira config filters --json
{
  "filters": [
    {
      "name": "hot",
      "query": "priority <= P1 AND state != DONE ORDER BY priority"
    },
    {
      "name": "mine",
      "query": "owner = dana AND state != DONE"
    },
    {
      "name": "next",
      "query": "state = TODO AND NOT blocked ORDER BY priority"
    }
  ]
}
```

Errors are also JSON-encoded in this mode, so a script never has to parse
prose off stderr:

```console
$ kira config set commit.mode bogus --json
{"error":"config: commit.mode: invalid value \"bogus\", want one of [auto manual]","hint":"","code":1}
```

The authoritative description of every one of these shapes is
[`kira schema`](#kira-schema). Two commands are exceptions worth knowing:
`kira config` with no subcommand rejects `--json` outright, and `kira schema`
already emits JSON so the flag has no effect there.

### Exit codes

Human and JSON modes use the same exit codes. The `code` field in a JSON error
object always matches the process exit status.

| Code | Meaning |
| --- | --- |
| `0` | Success. |
| `1` | User error — bad flag, bad value, failed validation. |
| `2` | Conflict — the requested change collides with existing state. |
| `3` | Environment error — no project found, unresolvable config directory. |
| `4` | Crash — an internal failure; a crash log path is reported as the hint. |

### `--no-color` — plain output

`--no-color` strips ANSI styling from human output. You usually do not need
it: color is already **auto-disabled whenever stdout is not a terminal**, so
piping or redirecting produces clean text on its own.

The following was measured by counting escape bytes in piped output under each
setting:

| Configuration | Piped output contains ANSI? |
| --- | --- |
| `ui.color: auto` (default) | no |
| `ui.color: always` | yes |
| `ui.color: always` + `--no-color` | no |
| `ui.color: always` + `NO_COLOR=1` | no |

So the precedence is: `--no-color` and the `NO_COLOR` environment variable
both win over everything; otherwise the `ui.color` config key decides, and its
default of `auto` means "color only on a terminal".

## `kira init`

Creates a `.kira/` store in the current git repository. The repository must
already exist — `kira init` does not run `git init` for you.

| Flag | Description |
| --- | --- |
| `--key <string>` | Project key used for display IDs. Default: derived from the directory name. |
| `--force` | Reinitialize over an existing `.kira/`. |

```console
$ cd ~/src/acme
$ kira init
Initialized kira in .kira (project key ACME)
```

If `--key` is omitted and you are running on a terminal, kira prompts for the
key and offers the derived value as the default. Off a terminal it accepts the
derived value silently, **unvalidated** — the derivation just strips
non-alphanumerics and uppercases the rest, with no length cap, so a long
directory name yields a long key (see the getting-started walkthrough for a
real example). An *explicit* `--key` is held to a stricter rule — 2–10
uppercase letters or digits starting with a letter — and rejected before
anything is written if it does not match:

```console
$ kira init --key KIRADOCSDEMO
kira: project key "KIRADOCSDEMO" must match ^[A-Z][A-Z0-9]{1,9}$
  hint: keys are 2-10 uppercase letters/digits starting with a letter, e.g. ABC
```

### What gets created

```console
$ tree -a --noreport .kira
.kira
├── .gitignore
├── config.yaml
├── templates
│   ├── epic.md
│   └── ticket.md
└── tickets
```

- `config.yaml` — the project config, scaffolded with the default
  vocabularies and commented-out optional blocks.
- `tickets/` — where item files will be written. Empty at first.
- `templates/ticket.md`, `templates/epic.md` — the bodies new items start from.
- `.kira/.gitignore` — ignores `.cache/`, the disposable index kira maintains
  underneath `.kira/`.

One file outside `.kira/` is also touched. `init` appends a line to
`.gitattributes` (creating it if absent) so item files always use LF endings,
which keeps them mergeable across platforms:

```console
$ cat .gitattributes
.kira/** text eol=lf
```

Everything is then committed in one go:

```console
$ git log --oneline
c5398ae kira: init
90cdbf3 initial commit
```

`init` does **not** prompt about git hooks and does not install any. Hook
installation is a separate, explicit step — see `kira hooks install`.

### Reinitializing

Running `init` against an existing store is refused:

```console
$ kira init
kira: .kira already exists
  hint: use `--force` to reinitialize over it
```

`--force` proceeds, rewriting `config.yaml` and the templates — so any edits
you made to those are lost. Note also that a `--force` run which changes nothing has nothing
to commit and will report a git failure; pass a different `--key`, or simply
leave the store alone.

## `kira config`

With no subcommand, `kira config` reports where your personal config lives and
which of its files exist:

```console
$ kira config
/home/dana/.config/kira
  present: none
```

The directory is `$XDG_CONFIG_HOME/kira` when that variable is set, otherwise
`$HOME/.config/kira`. This form does not accept `--json`.

kira reads configuration from three tiers. Built-in **defaults** are the base;
the **repo** file `.kira/config.yaml` is versioned with the project and is
authoritative for anything the whole team must agree on; the **user** file
`~/.config/kira/config.yaml` holds personal presentation and workflow
preferences that should not be shared across clones. Where a key belongs to
both, the repo wins.

### `kira config explain`

Prints the *effective* configuration as readable rules, grouped into sections,
each tagged with the tier it came from. When you want to answer "why is this
project behaving this way?", this is the command — it shows the resolved rules
rather than making you diff three YAML files in your head.

```console
$ kira config explain
identity (repo)
  project: ACME (acme)
  board: ACME (acme) [default]

commit (default)
  mode: auto
  subject: "kira: update {count} items" (up to 5 ticket numbers listed, then a count)
  trailer: Kira-Ticket (up to 20 numbers)
  close_trailer: Kira-Closes
  link marker: trailer — a "Kira-Ticket" git trailer naming the ticket
  link marker: subject — a `[KEY-n]` marker in the commit subject
  link marker: leading_number — a bare KEY-n token leading the subject
  reference marker: bare — a KEY-n token anywhere in the commit body

vocab (default)
  system label: captured (auto-added by `kira create --here`; excluded from vocabulary checks)
  labels: none (advisory)
  people: none (advisory)
  priorities: P0, P1, P2, P3 (advisory)
  subtypes: bug, task, feature, support (advisory)
  resolutions: done, dropped, duplicate, cannot-reproduce (advisory)
  resolutions_dropped: dropped (a done-category resolution treated as a non-completion outcome for epic progress)

ui (default)
  icons: auto (override with the KIRA_ICONS env var: nerd|emoji|text)
  color: auto
  background: dark

workon (default)
  branch_pattern: {key}/{number}-{slug}
  worktree: false (dir ../{repo}-{branch})

merge (default)
  policy: auto

sync (default)
  push: false
```

**Reading the provenance tag.** The tag applies to the section as a whole, not
to each individual line. A section reads `default` when its values still match
kira's built-ins. When they differ, the tag names the tier that *owns* that
block: `ui` and `workon` are personal-preference blocks and are tagged `user`,
everything else is tagged `repo`. Pinning `ui.icons` in the personal config and
`workon.branch_pattern` in the project config therefore both show up as `user`:

```console
$ kira config explain
...
ui (user)
  icons: text (override with the KIRA_ICONS env var: nerd|emoji|text)
  color: auto
  background: dark

workon (user)
  branch_pattern: {key}-{number}
  worktree: false (dir ../{repo}-{branch})
```

`explain` also supports `--json`, which returns the same sections as
structured data:

```console
$ kira config explain --json | head -11
{
  "sections": [
    {
      "name": "identity",
      "provenance": "repo",
      "lines": [
        "project: ACME (acme)",
        "board: ACME (acme) [default]"
      ]
    },
```

### `kira config init`

Scaffolds your **personal** config directory — distinct from the project's
`.kira/config.yaml`, and never committed to any repository.

```console
$ kira config init
Created user config in /home/dana/.config/kira
  config.yaml
```

`config.yaml` holds personal preferences. It ships fully commented out, so
creating it changes no behavior until you uncomment something:

```console
$ cat ~/.config/kira/config.yaml
# # kira user preferences. Every line below is the built-in default.
# # Uncomment a line to pin that value for yourself, unless the repo's config sets the same key (repo wins on collision).
# # These keys are personal; repo-authoritative keys here are ignored with a warning.

# ui:                                              # terminal UI and board appearance
#   icons: auto                                    # glyph set: auto | nerd | emoji | text
#   background: dark                               # palette the TUI and board assume: dark | light
#   color: auto                                    # ANSI color: auto | always | never
#   editor: ""                                     # editor command for drafts, run via sh -c so shell quoting works; empty falls back to $EDITOR then $VISUAL
#   auto_tui: true                                 # launch the TUI when kira is run bare in a terminal
#   list:
#     columns: [number, state, type, priority, title]   # columns shown by `kira list`
#   tui:
#     split: 0.5                                   # tree/detail split fraction, open interval (0,1)
#     refresh: "0"                                 # TUI auto-refresh interval, e.g. 5s; "0" disables it (min 1s)
#   theme:                                         # override individual palette slots with hex colors (example values)
#     accent: "#7dcfff"
...
# workon:                                          # `kira workon` branch and worktree behavior
#   worktree: false                                # create a git worktree instead of switching the current checkout
#   worktree_dir: "../{repo}-{branch}"             # where worktrees are created; tokens: {repo} {branch} {key} {number}; absolute paths used as-is
```

The command is safe to re-run: if the file already exists it touches
nothing and simply reports what it found.

```console
$ kira config init
/home/dana/.config/kira already exists; leaving it untouched
  present: config.yaml
```

### `kira config filters`

Lists the named saved queries defined in the project config's `filters:`
block. Each is a query expression you can run with `kira list --filter <name>`
instead of retyping it.

A filter definition is a plain `name: query` mapping. kira's own repository
defines these, for instance:

```yaml
filters:
  backlog: "state = TODO ORDER BY priority"
  core: "label = core AND state != DONE"
```

A freshly initialized project ships with a single example filter, `next`:

```console
$ kira config filters
next  state = TODO AND NOT blocked ORDER BY priority
```

After adding two more to `.kira/config.yaml`, all three are listed, sorted by
name:

```console
$ kira config filters
hot   priority <= P1 AND state != DONE ORDER BY priority
mine  owner = dana AND state != DONE
next  state = TODO AND NOT blocked ORDER BY priority
```

If the project defines none:

```console
$ kira config filters
no filters configured
```

### `kira config set <key> <value>`

Sets one scalar key in the **project** config, `.kira/config.yaml`. It does
not write to your personal config — edit that file directly.

Only a fixed set of scalar keys can be set this way. Naming any other key
lists the valid ones:

```console
$ kira config set ui.nope true
kira: config: unknown key "ui.nope"; valid keys: commit.close_trailer, commit.mode, commit.trailer, estimate.unit, git.landed_ref, merge.policy, project.name, sync.push, ui.background, ui.icons, workon.branch_pattern
```

Values are validated against the same rules the config loader enforces, so a
bad value is rejected rather than written and discovered later:

```console
$ kira config set commit.mode bogus
kira: config: commit.mode: invalid value "bogus", want one of [auto manual]
```

**Comments and formatting are preserved.** kira splices the single line that
holds the value rather than re-encoding the document, so surrounding comments,
blank lines, and alignment survive untouched. Here is the `commit` block
before and after a change, with its inline documentation intact:

```console
$ grep -A2 '^commit:' .kira/config.yaml
commit:
  mode: auto                  # auto | manual
  trailer: Kira-Ticket        # trailer key that links a commit to its ticket

$ kira config set commit.mode manual
Set commit.mode = manual

$ grep -A2 '^commit:' .kira/config.yaml
commit:
  mode: manual                  # auto | manual
  trailer: Kira-Ticket        # trailer key that links a commit to its ticket
```

If the key is not present in the file at all — including when it exists only
as a commented-out example — the value is appended as a new block, and the
commented documentation is left as it was:

```console
$ kira config set workon.branch_pattern "{key}-{number}"
Set workon.branch_pattern = {key}-{number}

$ tail -6 .kira/config.yaml
workon:
  branch_pattern: '{key}-{number}'

# sprints: sprint entities, created by kira sprint create; items key into this list.
#sprints:
```

Like other mutations, the edit is committed when the project uses
`commit.mode: auto`:

```console
$ git log --oneline -1
50350ac kira: config set commit.mode
```

## `kira schema`

Prints the JSON Schema describing every `--json` output shape. This is what
you point a code generator or a validator at when you script against kira's
JSON mode.

The document is a JSON Schema draft 2020-12 file of roughly 47 KB. Its top
level is a `oneOf` over the 46 command result shapes, with 81 named
definitions under `$defs`:

```console
$ kira schema | head -20
{
  "$defs": {
    "Actor": {
      "additionalProperties": false,
      "properties": {
        "email": {
          "type": "string"
        },
        "name": {
          "type": "string"
        }
      },
      "required": [
        "name",
        "email"
      ],
      "type": "object"
    },
```

Individual result shapes are ordinary definitions. `VersionResult`, the shape
behind `kira version --json`, is a small representative example:

```console
$ kira schema | jq '.["$defs"].VersionResult'
{
  "additionalProperties": false,
  "properties": {
    "commit": {
      "type": "string"
    },
    "go": {
      "type": "string"
    },
    "json_contract": {
      "type": "integer"
    },
    "version": {
      "type": "string"
    }
  },
  "required": [
    "version",
    "json_contract",
    "go"
  ],
  "type": "object"
}
```

Note that `additionalProperties: false` throughout means the schema is strict:
a payload with an unexpected key fails validation.

Useful one-liners:

```console
$ kira schema > kira.schema.json                          # save for a codegen step
$ kira schema | jq -r '.oneOf[]."$ref"' | head -5         # list the result shapes
#/$defs/CreateResult
#/$defs/ListResult
#/$defs/TreeResult
#/$defs/ShowResult
#/$defs/MutationResult
```

`kira schema` always emits JSON, so passing `--json` changes nothing.

## `kira doctor`

Checks the whole repository for problems that a single item-level write would
not catch: id collisions, dangling references, and schema violations. Where
`kira validate` checks files you hand it, `doctor` checks everything already
in the store.

```
Usage: kira doctor [flags]
```

It takes no flags beyond the global ones. A clean repository reports only
informational lines:

```console
$ kira doctor
info    index-freshness index not built yet; it builds on first read
ok: 0 errors, 0 warnings, 1 info
see `kira config explain` for the rules these checks enforce
```

A dangling reference — here a `blocked_by` pointing at an id that resolves to
no item, reached by a hand edit rather than through `kira link` — is reported
as an error, and the command exits non-zero:

```console
$ kira doctor
error   dangling-ref    ACME-1 blocked_by: blocked_by references "01HAAAAAAAAAAAAAAAAAAAAAAA", which resolves to no item
info    index-freshness index not built yet; it builds on first read
problems found: 1 error, 0 warnings, 1 info
see `kira config explain` for the rules these checks enforce
kira: doctor: 1 problem found
```

`--json` gives one object per finding, the same shape `kira validate` uses:

```console
$ kira doctor --json
{
  "ok": false,
  "findings": [
    {
      "class": "dangling-ref",
      "severity": "error",
      "path": "01KYHYS0P0FRHDH9ZSC2GG1X5P.md",
      "item_id": "01KYHYS0P0FRHDH9ZSC2GG1X5P",
      "number": "ACME-1",
      "field": "blocked_by",
      "message": "blocked_by references \"01HAAAAAAAAAAAAAAAAAAAAAAA\", which resolves to no item"
    }
  ],
  "summary": { "error": 1, "warning": 0, "info": 1 }
}
```

Run it after a hand edit, an import, or a merge you are unsure about — it is
read-only and safe to run anytime.

## `kira changes --since <ref>`

Reports every field-level change across the *entire* backlog since a git ref
— the multi-item counterpart to `kira log <id>`, which is scoped to one item.
Use it for a standup summary or a release note, not for one ticket's history.

```
Usage: kira changes --since <ref> [flags]
```

| Flag | Description |
| --- | --- |
| `--since` | git ref or sha to compare from. Required. |

`--since` is mandatory; omitting it is an error rather than defaulting to
something surprising:

```console
$ kira changes
kira: changes: --since <ref> is required
  hint: example: kira changes --since HEAD~10
```

```console
$ kira changes --since HEAD~4
created ACME-1  Add health endpoint
created ACME-2  Trim docker image
  2026-07-27T19:43:44+05:30  priority: P2 -> P0
```

An item created inside the window is prefixed `created`, exactly like `kira
diff --incoming`; one that only changed fields lists each event underneath
its number and title. `--json` gives the same data structurally, including
the resolved commit shas for both ends of the range:

```console
$ kira changes --since HEAD~4 --json
{
  "since": "9691f2a3e4d4735e3ecedc201a563c8c3345843d",
  "head": "2453ee859f526cbd05806c066445a3de4989c9b0",
  "items": [
    { "id": "01KYHYS0P0FRHDH9ZSC2GG1X5P", "number": "ACME-1", "title": "Add health endpoint", "status": "created", "events": [] },
    { "id": "01KYHYS0XFZGAR0F47ZK3YR1N4", "number": "ACME-2", "title": "Trim docker image", "status": "created",
      "events": [ { "ts": "2026-07-27T19:43:44+05:30", "field": "priority", "old": "P2", "new": "P0", "commit": "2453ee859f526cbd05806c066445a3de4989c9b0" } ] }
  ]
}
```

`kira diff` compares two whole snapshots (your branch vs. a merge-base);
`kira changes` walks the commit-by-commit history in between. Reach for
`diff` to answer "what would merging change", and `changes` to answer "what
happened between these two points in time".

## `kira validate <file>...`

Checks one or more items against the project's schema and config without
importing, writing, or committing anything. Use it in CI or a pre-commit hook
to catch malformed items before they land, and to check a hand-written file
before feeding it to `kira create --from-file`.

Arguments may be file paths or item references, and the two can be mixed. A
clean run reports counts and exits `0`:

```console
$ kira validate .kira/tickets/*.md
ok: 0 errors, 0 warnings, 0 info
see `kira config explain` for the rules these checks enforce
```

A hand-edited file with problems reports one finding per line, each carrying a
severity and the class of check that produced it:

```console
$ kira validate ~/import/draft.md
error   state           ACME-9 state: "BLOCKED" is not a state in the ticket workflow
warning vocab           ACME-9 labels: "security" is not in the known labels vocabulary
warning vocab           ACME-9 priority: "P9" is not in the known priority vocabulary
problems found: 1 error, 2 warnings, 0 info
see `kira config explain` for the rules these checks enforce
kira: validate: 1 problem found
$ echo $?
1
```

**Only errors fail the command.** Warnings and info findings are reported but
exit `0`, so advisory vocabularies do not break a build. Fixing the invalid
state above, while leaving the unknown label and priority in place, is enough
to pass:

```console
$ kira validate ~/import/draft.md
warning vocab           ACME-9 labels: "security" is not in the known labels vocabulary
warning vocab           ACME-9 priority: "P9" is not in the known priority vocabulary
ok: 0 errors, 2 warnings, 0 info
see `kira config explain` for the rules these checks enforce
$ echo $?
0
```

Whether a vocabulary violation is a warning or an error depends on the
project. `labels.strict` in `.kira/config.yaml` promotes unknown values to
errors, and the built-in `priorities`, `subtypes`, and `resolutions`
vocabularies follow it. `kira config explain` shows the outcome per vocabulary, tagged
either `(advisory)` or `(strict)`:

```console
$ kira config explain
...
vocab (repo)
  system label: captured (auto-added by `kira create --here`; excluded from vocabulary checks)
  labels: none (strict)
  people: none (strict)
  priorities: P0, P1, P2, P3 (strict)
  subtypes: bug, task, feature, support (strict)
  resolutions: done, dropped, duplicate, cannot-reproduce (strict)
  resolutions_dropped: dropped (a done-category resolution treated as a non-completion outcome for epic progress)
```

With that in force, the same two findings above are errors rather than
warnings, and the command exits `1`:

```console
$ kira validate ~/import/draft.md
error   vocab           ACME-9 labels: "security" is not in the known labels vocabulary
error   vocab           ACME-9 priority: "P9" is not in the known priority vocabulary
problems found: 2 errors, 0 warnings, 0 info
see `kira config explain` for the rules these checks enforce
kira: validate: 2 problems found
```

For scripting, `--json` gives one object per finding with the file path, item
identity, offending field, and message:

```console
$ kira validate ~/import/draft.md --json
{
  "ok": false,
  "findings": [
    {
      "class": "state",
      "severity": "error",
      "path": "/home/dana/import/draft.md",
      "item_id": "01KYHXM54ST9V9RK3HX922DDZD",
      "number": "ACME-9",
      "field": "state",
      "message": "\"BLOCKED\" is not a state in the ticket workflow"
    }
  ],
  "summary": {
    "error": 1,
    "warning": 0,
    "info": 0
  }
}
```

An argument that is neither a readable file nor a resolvable item reference is
an error in itself:

```console
$ kira validate ~/import/nope.md
kira: "/home/dana/import/nope.md" is neither a file nor a resolvable ticket id
```

## `kira version`

Prints the version string. Release binaries print their tag; a binary built
from source, as below, prints `dev`:

```console
$ kira version
dev
```

`--json` additionally reports the JSON output contract version, the Go
toolchain the binary was built with, and the commit it was built from. Check
`json_contract` if your tooling needs to guard against output-shape changes:

```console
$ kira version --json
{
  "version": "dev",
  "json_contract": 1,
  "go": "go1.27-devel_027a5bf27f Sat Apr 11 15:28:25 2026 -0700",
  "commit": "6967dfd4e1b22efe2e5e2285636f4711f54eefec"
}
```

The `commit` field is present only when the binary carries git build
information; `version`, `json_contract`, and `go` are always reported.

## `kira completion`

Generates a shell completion script. Four shells are supported:

```console
$ kira completion --help
Generate the autocompletion script for kira for the specified shell.
See each sub-command's help for details on how to use the generated script.

Usage:
  kira completion [command]

Available Commands:
  bash        Generate the autocompletion script for bash
  fish        Generate the autocompletion script for fish
  powershell  Generate the autocompletion script for powershell
  zsh         Generate the autocompletion script for zsh
```

Each subcommand accepts `--no-descriptions`, which omits the inline
descriptions from the completion candidates.

### bash

Requires the `bash-completion` package. To load completions in the current
session:

```sh
source <(kira completion bash)
```

To load them for every new session, run once:

```sh
# Linux
kira completion bash > /etc/bash_completion.d/kira

# macOS
kira completion bash > $(brew --prefix)/etc/bash_completion.d/kira
```

### zsh

If completion is not already enabled in your environment, enable it once:

```sh
echo "autoload -U compinit; compinit" >> ~/.zshrc
```

Then, for the current session:

```sh
source <(kira completion zsh)
```

Or, for every new session:

```sh
# Linux
kira completion zsh > "${fpath[1]}/_kira"

# macOS
kira completion zsh > $(brew --prefix)/share/zsh/site-functions/_kira
```

### fish

```sh
kira completion fish | source                              # current session
kira completion fish > ~/.config/fish/completions/kira.fish # every session
```

In every case you need to start a new shell for the persistent setup to take
effect.

### What gets completed

The scripts do more than complete subcommand names. kira registers dynamic
completers that read your project, so tab-completion offers real ticket
numbers, the configured vocabularies, and the states declared by an item's
workflow.

- Item references for `show`, `edit`, `comment`, `log`, `blame`, `workon`,
  `link`, `move`, `assign`, and `label add`/`rm`, annotated with the item
  title.
- The legal target states for `kira move <id>`, computed from that item's type
  and current state.
- Config vocabularies behind `--priority`, `--label`, `--owner`,
  `--reporter`, `--subtype`, `--resolution`, `--state`, `--sprint`,
  `--filter`, and `--board`.
- The valid keys for `kira config set`.

Because these completers query the project, they return nothing when you are
outside a kira repository — completion degrades quietly rather than erroring.
