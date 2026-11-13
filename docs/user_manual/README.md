# kira User Manual

kira is a git-native, terminal-first ticket tracker: tickets are Markdown
files with YAML frontmatter, committed alongside the code they track. This
manual covers using the `kira` CLI and TUI day to day — not kira's own
internals.

## Start here

| Document | Read this when... |
| --- | --- |
| [Overview](01-overview.md) | You want the 5-minute mental model before touching anything. |
| [Getting Started](02-getting-started.md) | You're setting up kira for the first time and want a hands-on walkthrough. |
| [Concepts](03-concepts.md) | You want the full mental model: identity, workflow, boards, sprints, relationships, config layers. |

## Reference

| Document | Covers |
| --- | --- |
| [CLI Reference](04-cli-reference/README.md) | Every command and flag, grouped by area. |
| [Configuration Reference](05-configuration-reference.md) | Every key in `.kira/config.yaml`. |
| [TUI Guide](07-tui-guide.md) | Screens, keybindings, and the `:` command bar. |

## Practical

| Document | Covers |
| --- | --- |
| [Recipes and Examples](08-recipes-and-examples.md) | Everyday workflows and niche/power-user options, worked end to end. |
| [Limitations and Roadmap](09-limitations-and-roadmap.md) | What kira genuinely does not do yet, sourced from its own open tickets. |

## CLI reference, by area

The [CLI Reference](04-cli-reference/README.md) is split into five files so
each stays a manageable size:

- [Item lifecycle](04-cli-reference/item-lifecycle.md) — `create`, `edit`, `move`, `link`, `label`, `assign`, `comment`
- [Views and queries](04-cli-reference/views-and-queries.md) — `list`, `show`, `tree`, `find`, `log`, `blame`, `stats`, `now`, `discover`, `diff`
- [Boards and sprints](04-cli-reference/boards-and-sprints.md) — `board *`, `sprint *`
- [Git integration](04-cli-reference/git-integration.md) — `sync`, `resolve`, `hooks *`, `commit`, `index`, `workon`
- [Setup and schema commands](04-cli-reference/setup-and-schema-commands.md) — `init`, `config *`, `schema`, `validate`, `version`, `completion`, and the global flags shared by every command

## A note on accuracy

Every command, flag, and example in this manual was verified against a real
kira build — either quoted from `--help` output, read from source, or
captured by actually running the command in a scratch repository. Where a
real, currently-open limitation or bug was found while writing these pages,
it's called out inline (and filed as a kira ticket in this repo's own
tracker) rather than glossed over.
