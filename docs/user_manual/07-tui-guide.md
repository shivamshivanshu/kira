# The TUI

The terminal UI is a full-screen, keyboard-driven front end over the same
operations the CLI exposes. It is built for the loop you actually run all day —
look at what is in flight, read a ticket, move it, get back to work — without
retyping ticket numbers or leaving the terminal.

## Launching

Run `kira tui`. Bare `kira` with no subcommand launches the same UI, but only
when `ui.auto_tui` is enabled in `.kira/config.yaml`, standard output is a real
terminal, and neither `--json` nor non-interactive mode is in effect; otherwise
bare `kira` prints its help text. See the
[configuration reference](05-configuration-reference.md) for `ui.auto_tui`,
`ui.tui.split`, and `ui.tui.refresh`.

If you launch it inside a git work tree that has no `.kira/` yet, kira offers to
create one: it explains that it will write `.kira/` and make a `kira: init`
commit, then asks for a project key. Type the key and press `Enter` to
initialize, or `Esc` to cancel.

The UI takes over the alternate screen, so your scrollback is untouched when you
quit.

### Reading the frame

The top line is the screen switcher — `kira  1:tree  2:board  3:stats`, with the
current screen highlighted. The bottom line is context-sensitive: normally it
lists the keys active right now, but it becomes the input line while the command
bar or filter is open, shows the result of the last command bar invocation, or
shows `filter: <expr>` while a filter is active.

Everything between is the screen body. When the terminal is at least 80 columns
wide, the tree and board screens can show the detail pane docked beside the
list; below 80 columns the detail pane replaces the list instead of splitting
beside it. `ui.tui.split` sets the ratio. A detail pane narrower than 30 columns
renders `▸ widen` rather than unreadably wrapped text.

If `ui.tui.refresh` is set, the UI re-reads the store on that interval, so
changes made by another terminal or by a teammate's merge appear on their own.

## Tree

The default screen. It shows every item as a row — epics with their child
tickets indented beneath, a fold marker (`▾` expanded, `▸` collapsed), the type
glyph, the display number, the title, then priority, category glyph, `[STATE]`,
and an overdue marker when a due date has passed. In the split layout, epics
also carry a completion bar.

This is the screen for "what is on my plate" — hierarchy, reading a ticket in
full, and editing. Selection drives the detail pane, so moving the cursor is
also browsing.

| Key | Action |
| --- | --- |
| `j` / `k` (or `↓` / `↑`) | Move the selection down / up |
| `l` or `enter` | Expand a collapsed epic; otherwise move focus to the detail pane |
| `h` | Collapse an expanded epic; otherwise jump to the parent epic |
| `gp` | Jump to the selected item's parent epic |
| `tab` / `shift+tab` | Toggle focus between the tree and the detail pane |
| `gg` / `G` | First / last row |
| `^d` / `^u` | Half a page down / up |
| `zM` / `zR` | Collapse every epic / expand every epic |
| `e` | Open the item's file in your editor |
| `v` | Open a read-only copy of the item's file in your editor |
| `esc` or `q` | Return focus from the detail pane to the tree (otherwise quits) |

`zM` keeps you oriented: after collapsing everything, the cursor lands on the
top-level ancestor of whatever was selected, and `zR` restores the fold state
without moving the cursor.

While the detail pane holds focus, detail-pane keys (`j`/`k`, `[`/`]`, `enter`,
`gg`/`G`, `^d`/`^u`) act on the detail pane, and the rest — including `tab`,
`e`, and `v` — still work as tree keys.

`e` and `v` need `ui.editor` (or `$EDITOR`) to be resolvable; if it is not, the
footer says so rather than the UI failing silently. `v` writes a read-only
temporary copy and deletes it when the editor exits, so it cannot be saved over
the real file.

## Board

A column-per-state kanban view of the same items, with WIP counts in the column
headers (`IN_PROGRESS  4/3`, coloured when a column is at or over its limit).
Use it for the state of the work rather than its structure — and for moving
things, which is faster here than anywhere else.

| Key | Action |
| --- | --- |
| `h` / `l` (or `←` / `→`) | Previous / next column |
| `j` / `k` (or `↓` / `↑`) | Previous / next card in the column |
| `H` / `L` | **Move the selected item** to the previous / next column |
| `b` | Open the board picker to scope the view to one board |
| `enter` | Open the item in the full-screen detail pane |
| `p` | Toggle the docked peek pane |
| `tab` / `shift+tab` | Expand a docked peek into the full-screen detail pane |
| `gg` / `G` | First / last card in the current column |
| `esc` or `q` | Close the peek or detail pane (otherwise quits) |

`H` and `L` are real mutations, not navigation. Moving a card left or right
transitions the item's state exactly as `kira move` would — writing the file and
committing it under `commit.mode: auto`. The move is checked against the
configured workflow graph first: if the target column is not reachable from the
item's current state in one step, nothing changes and a message appears under
the board, for example
`DEMO-1: TODO -> DONE is not an allowed transition`. A successful move reports
`Moved DEMO-1: TODO -> IN_PROGRESS` and keeps the card selected in its new
column.

`p` and `enter` differ in how much room the detail gets. `p` docks a peek pane
beside the board and tracks the selection as you move — the board keys stay
live, so you can skim a column and read each card's body as you pass it. `enter`
gives the item the whole screen and hands the keys to the detail pane. `tab`
promotes a docked peek to the full-screen form; `esc` steps back out. On a
terminal narrower than 80 columns a docked peek has nowhere to dock, so it
renders full-screen too.

`b` opens a picker listing `All boards` plus each active board; choosing one
filters the board to that board's items and shows `board: KIRA` above the
columns.

## Stats

A read-only summary: sprint name when a sprint is active, completion, cycle-time
and lead-time percentiles, a weekly throughput sparkline, and reopened items. It
reports on the active sprint if there is one, otherwise on everything. Use it at
the end of a sprint, or when someone asks how long things actually take.

| Key | Action |
| --- | --- |
| `j` / `k` (or `↓` / `↑`) | Scroll down / up |
| `gg` / `G` | Top / bottom |
| `^d` / `^u` | Half a page down / up |

With no completed work yet, the screen says so instead of showing zeroes.

## The detail pane

Reached with `enter` (or `tab`) from the tree, and with `enter` or `p` from the
board. It renders the item's number and title, a metadata line (state, subtype,
owner, priority, due date — flagged when overdue — and labels), the rendered
Markdown body, comments, linked commits, commits that merely reference the item,
and the tail of its field history.

| Key | Action |
| --- | --- |
| `j` / `k` (or `↓` / `↑`) | Scroll down / up |
| `[` / `]` | Select the previous / next linked commit |
| `enter` | Run `git show` on the selected commit |
| `gg` / `G` | Top / bottom |
| `^d` / `^u` | Half a page down / up |
| `esc` or `q` | Leave the detail pane |

The commit list is the part worth knowing about. `[` and `]` walk the commits
linked to this item — the `>` marker shows which is selected — and `enter` shells
out to `git show` for it in your pager. Reading a ticket and then reading the
diff that implemented it is two keystrokes apart.

## The command bar

Press `:` to open a command bar at the bottom of the screen. It runs **any kira
CLI subcommand**, with output reported in the footer. This is the escape hatch
that keeps the screens uncluttered: anything without a dedicated key is one
`:` away.

Inside the bar, a bare `.` is substituted with the display number of the
currently focused item. So the item you are looking at is the item the command
acts on, and you never retype a ticket number.

```
: move . IN_PROGRESS
: label add . urgent
: link . --blocked-by DEMO-2
: comment . -m "waiting on review"
: assign . alice
: workon .
```

Details worth knowing:

- Arguments are split on whitespace, and single or double quotes group a
  multi-word argument — `-m "waiting on review"` arrives as one argument.
- Commands run in non-interactive mode, so anything that would open your editor
  fails rather than hanging. `comment .` with no `-m` returns
  `comment without --message opens $EDITOR`; pass `-m`, or use `e` on the tree
  screen when you want the editor.
- Only the first non-empty line of output (or of the error) is shown in the
  footer. Run the command in a normal shell when you want the full output.
- A command that succeeds triggers a reload, so the screens reflect the change
  immediately.
- `tui` is rejected — `cannot launch the tui from the command bar`. No nesting.
- `esc` closes the bar without running anything.

`n` is a shortcut for the same bar pre-filled with `create `, ready for
`create ticket "Some title"`.

## Filtering

`/` opens the same bar in filter mode, taking a `kira list` filter expression.
The tree is pruned to matching items plus the ancestors needed to reach them,
and the footer shows `filter: <expr>  (/ change · : command)` until you clear
it by submitting an empty expression. The filter applies to the tree screen;
the board and stats screens are unaffected.

## Global keys

These work on every screen.

| Key | Action |
| --- | --- |
| `1` / `2` / `3` | Switch to the tree / board / stats screen |
| `^o` / `^]` | Jump backward / forward through the jump list |
| `:` | Open the command bar |
| `/` | Filter |
| `y` | Copy the focused item's id to the clipboard |
| `Y` | Open the yank picker to choose a reference form |
| `n` | Command bar pre-filled with `create ` |
| `r` | Reload from disk |
| `?` | Toggle the help screen |
| `q` or `esc` | Step back one level; quit from the top level |
| `^c` | Quit immediately |

The **jump list** works like a browser's back and forward buttons. kira records
a position every time you switch screens or drill into an item, so `^o` retraces
your steps — including across screens, landing you back on the same item in the
screen you left — and `^]` goes forward again. Entries pointing at items that no
longer exist are dropped on the next reload.

**Yank** copies to the terminal clipboard via OSC 52, which works over SSH and
through tmux; where a local display is available kira also uses `pbcopy`,
`wl-copy`, `xclip`, or `xsel`. `y` copies the item's immutable id. `Y` opens a
picker with every reference form and a live preview of each:

| Form | Example |
| --- | --- |
| `id` | `01KYHXCJK0A3GJ2QWE1VRXV8F2` |
| `number` | `DEMO-3` |
| `number + title` | `DEMO-3 Add search to the docs site` |
| `markdown link` | `[DEMO-3 Add search to the docs site](DEMO-3)` |
| `branch name` | `demo-3-add-search-to-the-docs-site` |

`?` replaces the body with the full key list for the current screen followed by
the global keys — the same bindings the footer abbreviates. Press `?` or `esc`
to dismiss it.

Any picker — the yank picker and the board picker — takes `j`/`k` to move,
`enter` to choose, a digit key to choose that numbered entry directly, and
`q` or `esc` to cancel.

`q` is layered rather than absolute: it closes the help screen if open, then
returns from the detail or peek pane, and only quits once there is nothing left
to back out of. If a mutation is still in flight when you quit, the footer says
`finishing in-flight change… (ctrl+c to force quit)` and kira waits for the
write to complete so it cannot leave a half-written ticket behind.
