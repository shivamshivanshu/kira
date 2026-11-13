# The TUI

Full-screen, keyboard-driven front end over the same core operations as the
CLI.

## Launching

`kira tui`, or bare `kira` when `ui.auto_tui` is on, stdout is a terminal and
neither `--json` nor
[`--non-interactive`](04-cli-reference/setup-and-schema-commands.md#global-flags)
is set (else bare `kira` prints help). Related keys: [`ui`](05-configuration-reference.md#ui).

Inside a git work tree with no `.kira/`, it offers to create one (writes
`.kira/`, commits `kira: init`): type a project key and `Enter`, or `Esc` to
cancel. It uses the alternate screen, so scrollback is untouched.

### Frame

- Top line: screen switcher `kira  1:tree  2:board  3:stats`.
- Bottom line: active keys; or the command/filter input; or the last command's
  result; or `filter: <expr>  (/ change · : command)` while a filter is set.
- At ≥ 80 columns the detail pane docks beside the tree/board
  (`ui.tui.split` sets the ratio); below 80 it replaces the list. A pane
  under 30 columns shows `▸ widen`.
- `ui.tui.refresh` re-reads the store on an interval, picking up other
  terminals' changes and merges.

## Tree

Default screen: every item, children indented under epics, fold marker
(`▾`/`▸`), type glyph, number, title, priority, category glyph, `[STATE]`,
overdue marker; epics get a completion bar in the split layout. The selection
drives the detail pane.

| Key | Action |
| --- | --- |
| `j` / `k` (`↓` / `↑`) | Next / previous row |
| `l` / `enter` | Expand a collapsed epic; else focus the detail pane |
| `h` | Collapse an expanded epic; else jump to the parent |
| `gp` | Jump to the parent |
| `tab` / `shift+tab` | Toggle focus tree ↔ detail |
| `gg` / `G` | First / last row |
| `^d` / `^u` | Half page down / up |
| `zM` / `zR` | Collapse / expand all epics (`zM` lands on the top-level ancestor) |
| `e` | Open the item file in the editor |
| `v` | Open a read-only temporary copy in the editor (deleted on exit) |

With the detail pane focused, detail keys act there; the rest (`tab`, `e`,
`v`, …) still act on the tree. `e`/`v` need `ui.editor` or `$VISUAL`/`$EDITOR`;
otherwise the footer reports the error.

## Board

One column per state, WIP counts in headers (`IN_PROGRESS  4/3`, colored at or
over the limit).

| Key | Action |
| --- | --- |
| `h` / `l` (`←` / `→`) | Previous / next column |
| `j` / `k` (`↓` / `↑`) | Next / previous card |
| `H` / `L` | **Move the item** to the previous / next column |
| `b` | Board picker: `All boards` or one active board (shows `board: KEY`) |
| `enter` | Full-screen detail pane |
| `p` | Toggle a docked peek pane that follows the selection |
| `tab` / `shift+tab` | Expand the peek to full screen |
| `gg` / `G` | First / last card in the column |

`H`/`L` is a real `kira move` (written, committed under `commit.mode: auto`).
A refused move leaves the item unchanged and shows the first line of the
error, e.g. `invalid item: REVIEW -> DONE is blocked by open items: DEMO-4`;
success shows `Moved DEMO-1: TODO -> IN_PROGRESS` and keeps the card
selected. Under 80 columns the peek renders full-screen.

## Stats

Read-only: sprint name (when one is active), completion, cycle- and lead-time
percentiles, weekly throughput sparkline, reopened items. Scope is the active
sprint, else everything; with no data it says `No metrics yet — …`.
Keys: `j`/`k`, `gg`/`G`, `^d`/`^u`.

## Detail pane

Number and title; a metadata line (state, subtype, owner, priority, due —
flagged when overdue — labels); the rendered body; comments; linked commits;
referencing commits; the tail of the field history.

| Key | Action |
| --- | --- |
| `j` / `k` (`↓` / `↑`) | Scroll |
| `[` / `]` | Previous / next linked commit (`>` marks it) |
| `enter` | `git show` the selected commit in the pager |
| `gg` / `G` | Top / bottom |
| `^d` / `^u` | Half page |
| `esc` / `q` | Leave the pane |

## Command bar

`:` runs any kira subcommand; a bare `.` argument is replaced by the focused
item's number. `n` opens it pre-filled with `create `.

```
: move . IN_PROGRESS
: link . --blocked-by DEMO-2
: comment . -m "waiting on review"
: workon .
```

- Shell-style splitting: quotes group words.
- Runs non-interactively, so editor-opening forms fail
  (`comment without --message opens $EDITOR`); use `-m`, or `e` on the tree.
- The footer shows the first non-empty line of output or error; success
  reloads the screens.
- `tui` is refused: `cannot launch the tui from the command bar`.
- `esc` closes without running.

## Filter

`/` takes a [query expression](04-cli-reference/views-and-queries.md#query-expressions).
The tree is pruned to matches plus their ancestors; board and stats are
unaffected. Submit an empty expression to clear it.

## Global keys

| Key | Action |
| --- | --- |
| `1` / `2` / `3` | Tree / board / stats |
| `^o` / `^]` | Jump list back / forward |
| `:` / `/` | Command bar / filter |
| `y` | Copy the focused item's id |
| `Y` | Yank picker: every `show --format` form with a preview |
| `n` | Command bar with `create ` |
| `r` | Reload |
| `?` | Toggle the key help for this screen plus globals |
| `q` / `esc` | Close help, then leave detail/peek, then quit |
| `^c` | Quit now |

- Jump list: a position is recorded on each screen switch and drill-in;
  `^o` returns across screens to the same item. Entries for deleted items
  drop on reload.
- Yank: OSC 52 (works over SSH and tmux), plus `pbcopy`, `wl-copy`, `xclip` or
  `xsel` when available. Forms are those of
  [`show --format`](04-cli-reference/views-and-queries.md#kira-show).
- Pickers (yank, board): `j`/`k`, `enter`, a digit to pick that entry,
  `q`/`esc` to cancel.
- Quitting with a write in flight shows
  `finishing in-flight change… (ctrl+c to force quit)` and waits for it.
