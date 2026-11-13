# Current Limitations and Roadmap

Known gaps, each backed by this repository's own tickets. Live state:
`kira list --category todo` in the kira repo.

## Boards and sprints are not items

Tickets and epics are files with their own history, comments and `kira show`.
Boards and sprints are entries in `.kira/config.yaml`, so:

- `kira show CORE`, `kira comment CORE`, `kira log CORE` fail with
  *resolves to no item*; `kira board list` / `kira sprint list` are the only
  views.
- Their history is the history of `.kira/config.yaml`, mixed with every other
  config change.

**Roadmap:** none. Moving them to file-backed storage (CORE-39) was dropped.

## Open bugs

None.
