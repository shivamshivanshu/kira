# Compatibility

From `v1.0.0` kira follows semantic versioning over the surfaces below: a
release that breaks one of them bumps the major version. Anything not listed
may change in any release. Before `v1.0.0`, any release may break anything.

## Stable

| Surface | Breaking | Not breaking |
| --- | --- | --- |
| Item files: `.kira/tickets/<id>.md`, frontmatter keys and value forms, comment markers in the body | a file a 1.x release wrote no longer parses, or parses to different values | a new optional key (older releases reject a file that uses it) |
| Identity ([Concepts](03-concepts.md#identity-id-and-number)) | an `id` changing; an alias no longer resolving | renumbering on a collision (already documented behaviour) |
| Item types, states, subtypes, priorities, resolutions | removing or renaming one, or a transition that existing items rely on | adding one |
| `.kira/config.yaml` keys ([Configuration](05-configuration-reference.md)) | removing or renaming a key, or changing its meaning without bumping `version:` | a new key with a default |
| `--json` output (shapes printed by `kira schema`) | removing or renaming a field, or changing its type | a new field |
| [Exit codes](04-cli-reference/setup-and-schema-commands.md#exit-codes) | any change to a code's meaning | — |
| Commands and flags ([CLI reference](04-cli-reference/README.md)) | removing or renaming one, or changing what it does | a new command or flag |
| Commit linkage: `Kira-Ticket:` / `Kira-Closes:` trailers, `[[KEY-n]]`, leading `KEY-n` | a form that linked a commit no longer linking it | a new form |

## Not covered

- Human-readable output: tables, `show`, `log`, warnings, error wording. Scripts
  parse `--json`.
- The TUI and the web dashboard, including its HTTP routes.
- `.kira/.cache/`: disposable, rebuilt whenever its schema changes.
- Go packages under `internal/`.

## Releases

Every push to `main` that passes CI tags and publishes the next patch version.
A minor or major release is cut with *Run workflow* on the CI workflow, choosing
`bump: minor` or `bump: major`.
