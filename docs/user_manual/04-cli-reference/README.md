# CLI Reference

Every kira command, grouped by what it's used for. Each file below covers a
full area with a flag-by-flag reference and verified examples.

| File | Commands |
| --- | --- |
| [Item lifecycle](item-lifecycle.md) | `create`, `edit`, `move`, `link`, `label add/rm/create/list`, `assign`, `comment` — the commands that create and mutate items. |
| [Views and queries](views-and-queries.md) | `list`, `show`, `tree`, `find`, `log`, `blame`, `stats`, `now`, `discover`, `diff` — read-only commands. |
| [Boards and sprints](boards-and-sprints.md) | `board *`, `sprint *` — the grouping and time-boxing constructs. |
| [Git integration](git-integration.md) | `sync`, `resolve`, `hooks *`, `commit`, `index`, `workon` — how kira and git work together. |
| [Setup and schema commands](setup-and-schema-commands.md) | `init`, `config *`, `schema`, `validate`, `version`, `completion`, plus the global flags (`-C`, `--json`, `--no-color`) shared by every command. |

Back to the [manual index](../README.md).
