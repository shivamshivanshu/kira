# Current Limitations and Roadmap

kira is under active development, and some things you might reasonably expect
it to do it does not do yet. This page lists those gaps honestly: what you
might try, what actually happens today, and whether a fix is planned. Every
entry here is drawn from kira's own open tickets — kira tracks its own
development in kira, so the tickets behind this page live in `.kira/tickets/`
in the kira repository alongside the code.

## Boards and sprints are not tracked items

Tickets and epics are files under `.kira/tickets/`, each with its own git
history, its own comment thread, and its own `kira show`. Boards and sprints
are not. They are entries in `.kira/config.yaml`, managed through
`kira board create` / `rename` / `archive` and `kira sprint create` / `activate`
/ `close`.

The practical consequences:

- `kira show CORE` fails with *resolves to no item*. There is no detail view
  for a board or a sprint; `kira board list` and `kira sprint list` are the
  only views.
- `kira comment CORE -m "..."` fails the same way. You cannot attach a
  discussion thread to a board or a sprint.
- `kira log CORE` fails too. Changes to a board are committed, but as edits to
  `.kira/config.yaml` — so the history is the config file's history, mixed in
  with every other config change, not a per-board timeline.

This is a deliberate split, not a bug: a board is a grouping construct that
lives in configuration, while a ticket is a first-class tracked artifact. It is
worth knowing about because the asymmetry is invisible until you try to comment
on a board.

**Roadmap:** active but unscheduled. CORE-39 tracks moving boards and sprints
onto the same file-backed storage as tickets, which would give them the history
and commentary that tickets have.

A related, deeper change — reworking kira's internal representation of items
entirely — is recorded in the design notes as "maybe never." It is not on a
committed roadmap, and you should not plan around it.

---

This page reflects kira's open tickets as of the last documentation update. For
the current live state, run `kira list --category todo` in the kira repository
— it lists every open, not-yet-started ticket across all boards, and
`kira list --category doing` shows what is in flight. `kira show <id>` on any
of them gives the full description and discussion.
