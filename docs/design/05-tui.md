# TUI

**Scope:** the bubbletea program's frame, panels, keymaps, and visual language — `kira tui` (bare `kira`).
Part of the kira design set — see [DESIGN.md](../../DESIGN.md) for decisions and rationale.

## 1. Stack and write path

Bare `kira` (no subcommand) launches the TUI — this is the **primary, daily-driver interface**, matching the founder's explicit target: lazygit-style, "a GUI in the terminal." `kira <noun> <verb>` subcommands are the scripting/automation surface; both call the identical `internal/core` functions, so nothing in this doc is a second implementation of anything in [04-cli.md](04-cli.md).

Stack: bubbletea (program/model/update loop) + bubbles (list, viewport, textinput components) + lipgloss (layout/styling) + glamour (markdown rendering of ticket bodies). `kira tui --board` opens directly on the kanban view (see [04-cli.md `kira tui`](04-cli.md#kira-tui)).

Every mutation a panel or popup performs (`move`, `assign`, `link`, `comment`, `edit`) calls the exact same `internal/core` function the CLI command calls — see [01-architecture.md §6](01-architecture.md#6-single-write-path-guarantee).

## 2. Lazygit-style interaction model

kira's TUI is a **persistent-frame, multi-panel program**, not a sequence of full-screen pages that replace each other.

- **Frame.** One always-visible frame. A view occupies the main area: **Tree** (`1`, default), **Board** (`2`), **Stats** (`3`) — the founder's synthesis's "screens" are the *views* of this one frame, not standalone pages. Tree additionally splits its main area into a tree pane and a detail pane (§3, §5); Board and Stats use the full main area.
- **Focus cycling.** `Tab` / `Shift-Tab` cycles focus between panes within the current view (e.g. tree pane ↔ detail pane). Number keys `1`..`3` jump directly to a view from anywhere — lazygit's panel-number convention.
- **Hint bar.** A single line pinned to the bottom of the frame, always visible, listing the keys valid for the currently focused pane. Generated from the same per-view keymap table that backs the `?` overlay (§7) — one source, so hint bar and help can never drift apart.
- **Popups for input, never full-screen forms.** Single-keystroke triggers open a small modal popup over the current view:
  - `m` — move popup: lists only the states reachable from the focused item's current state, per its type's transition adjacency map ([02-data-model.md §6](02-data-model.md#6-state-machine)); unreachable targets aren't listed at all here (contrast Board's `H`/`L`, which shows them greyed rather than hiding them — §4).
  - `a` — assign popup: prompts for a user (validated against `people.known` if strict).
  - `c` — comment popup: opens `$EDITOR` (same as `kira comment`).
  - `n` — new-ticket popup: title, type, parent, then `$EDITOR` for the body — same path as `kira create`.
  - Destructive-feeling operations (`--force` move off the adjacency map, a `WONT_DO`/dropped transition) get a yes/no confirmation popup, defaulting to no.
  - Every popup submits through the same `internal/core` call its CLI equivalent uses (§1) — a popup is an input surface, never a second code path.
- `?` opens the full keymap overlay (every pane's bindings, not just the focused one). `q`/`Esc` is uniform: closes a popup, or pops focus/navigation back one level, or — at the frame's top level — quits.

## 3. Tree / explorer (view `1`, default)

```
┌─ kira ── 1:tree  2:board  3:stats ──────────┬─ KIRA-142 ─────────────────────┐
│ ▾ KIRA-100  Order book hardening      [epic]│ Fix race in order-book snapshot│
│   ▸ KIRA-140  Fix snapshot dedup      [TODO]│ merge                          │
│   ▾ KIRA-142  Fix race in ...  [IN_PROGRESS]│                                 │
│   ▸ KIRA-150  Add burst regression    [TODO]│ owner: shivam   P1   bug orderbo│
│ ▸ KIRA-200  Telemetry pipeline        [epic]│                                 │
│                                              │ The snapshot merge path drops  │
│                                              │ updates when two feed threads  │
│                                              │ race on the same price level.  │
│                                              │ ...                            │
├──────────────────────────────────────────────┴─────────────────────────────┤
│ j/k move  h/l collapse/drill  gp parent  Tab pane  m move  ? help          │
└──────────────────────────────────────────────────────────────────────────────┘
```

Collapsible epic→ticket tree pane on the left; live detail pane on the right (§5), updating as the tree selection moves. Netrw/file-explorer semantics — the target user's existing muscle memory.

| Key | Action |
|---|---|
| `j` / `k` | move selection down / up |
| `l` / `Enter` | drill in: expand a collapsed epic, or focus the detail pane for a leaf |
| `h` / `Esc` | pop: collapse the current epic, or leave the detail pane back to the tree |
| `gp` | jump to the selected item's parent epic |
| `Tab` | cycle focus between tree pane and detail pane (§2) |
| `/` | fuzzy filter (§7) |
| `:` | ex-command bar (§7) |
| `?` | help overlay (§7) |
| `r` | refresh |
| `q` | quit |

## 4. Kanban board (view `2`)

```
┌─ kira ── 1:tree  2:board  3:stats ── KIRA-100 ─────────────────────────────┐
│  TODO            │ IN_PROGRESS       │ REVIEW            │ DONE            │
│  ─────           │ ───────────       │ ──────            │ ────            │
│  KIRA-150         │▸KIRA-142          │                   │ KIRA-101        │
│  Add burst regr.. │ Fix race in ...   │                   │ Initial index   │
│                   │                   │                   │ KIRA-110        │
│                   │                   │                   │ Config loader   │
├──────────────────────────────────────────────────────────────────────────┤
│ h/l column  j/k card  H/L move (validated)  Enter detail  ? help          │
└──────────────────────────────────────────────────────────────────────────────┘
```

Columns are the item type's configured `workflows.<type>.states` in declared order ([02-data-model.md §6](02-data-model.md#6-state-machine)). Same board implementation as `kira board --plain` ([04-cli.md `kira board`](04-cli.md#kira-board)). Column headers render WIP-limit tinting per [02-data-model.md §6](02-data-model.md#6-state-machine) — `n/wip` whenever `wip:` is configured, where `n` is the global per-state count from the index, never the rendered subset (an epic-scoped board or active `/` filter must not understate real column pressure).

| Key | Action |
|---|---|
| `h` / `l` | move column focus left / right |
| `j` / `k` | move card focus within the focused column |
| `H` / `L` | move the focused card to the previous / next state via the validated `kira move` path; targets not reachable per the transition adjacency map are greyed out and `H`/`L` is a no-op toward them |
| `Enter` | open the detail pane (§5) as an overlay popup for the focused card — Board has no persistent detail pane, unlike Tree |
| `/` | fuzzy filter (§7) |
| `:` | ex-command bar (§7) |
| `?` | help overlay (§7) |
| `r` | refresh |
| `q` / `Esc` | close the detail overlay if open, else back to Tree |

## 5. Detail pane

```
┌─ KIRA-142 ── IN_PROGRESS ───────────────────────────────────────────────────┐
│  Fix race in order-book snapshot merge         owner: shivam   P1  bug order│
│                                                                              │
│  The snapshot merge path drops updates when two feed threads race on the    │
│  same price level. Repro: bench/burst_test --dup-updates=high.              │
│                                                                              │
│  Acceptance criteria                                                        │
│  [ ] TSan clean on order_book_test                                         │
│  [ ] No p99 regression on hot path                                         │
│                                                                              │
│  Comments                                                                   │
│  shivam  2026-07-11 18:30  Confirmed repro with TSan; missing acquire fence │
│                                                                              │
│  Linked commits                                                             │
│  a1b2c3d  fix acquire fence on the consumer side                           │
│                                                                              │
│  History                                                                    │
│  2026-07-11 18:30  state: TODO -> IN_PROGRESS                              │
└──────────────────────────────────────────────────────────────────────────────┘
```

Not a separate "screen" — one renderer, mounted two ways: as the persistent right-hand pane in Tree (§3, live, follows tree selection) and as an overlay popup opened from Board (§4, `Enter` on a card). Fields header, glamour-rendered body (description + acceptance criteria + comments), linked commits, history tail (last N events — full history via `kira log <id>`, [04-cli.md `kira log`](04-cli.md#kira-log)).

| Key | Action |
|---|---|
| `gp` | jump to parent epic |
| `e` | open `$EDITOR` on the ticket file (same as `kira edit <id>` with no flags) |
| `c` | comment popup (§2) |
| `m` | move popup (§2) |
| `a` | assign popup (§2) |
| `Enter` (on a linked-commit line) | `git show <sha>` in a pager |
| `j` / `k` | scroll body / history |
| `q` / `Esc` | pop back (to Tree's tree pane, or close the Board overlay) |

## 6. Stats (view `3`)

```
┌─ kira ── 1:tree  2:board  3:stats ── KIRA-100 ─────────────────────────────┐
│ completion   62%  (18/29, recursive)                                       │
│ cycle time   p50 1.2d   p90 4.0d     ▂▃▅▂▇▃▄▂▅▃▂▆                          │
│ lead time    p50 3.0d   p90 9.5d     ▁▂▄▇▃▂▅▆▃▂▁▃                          │
│ throughput   4.2/week (trailing 6)   ▃▄▅▃▆▄▅▃▇▄▅▆                          │
└──────────────────────────────────────────────────────────────────────────────┘
```

Completion / cycle time / lead time / throughput, per [08-telemetry.md](08-telemetry.md), rendered with lipgloss sparklines. `r` refresh, `q`/`Esc` back to Tree; scrolls with `j`/`k` if content overflows.

## 7. Cross-cutting

- **`/` fuzzy filter** — in-process, filters the currently visible tree/board/list without a subprocess call; clears on `Esc`.
- **`:` ex-command bar** — reuses the **exact CLI argv grammar** as its command language: `:move REVIEW`, `:assign alice`, `:link --blocked-by KIRA-99`. Parsed and dispatched to the same `internal/core` call the CLI's cobra tree would invoke, scoped to the currently-focused item. This is the one command language shared across CLI, TUI, and nvim's `:KiraCreate`/`:KiraEdit` — see [06-nvim-plugin.md](06-nvim-plugin.md).
- **`?` help overlay** — rendered from the same keymap table that defines each view's bindings in code and drives the hint bar (§2): one `[]KeyBinding{key, description}` slice per view, three consumers (hint bar, `?` overlay, this doc's tables). When the action palette lands ([STRETCH_GOALS.md](../STRETCH_GOALS.md)), rows gain an action-msg field; hint bar and `?` ignore it.
- **`r` refresh, no fs watcher.** Refresh re-runs the staleness check ([01-architecture.md §4](01-architecture.md#4-index-design)) on demand only. No background watcher: the workload is ms-scale, so "stale until you press r" is simpler and cheaper than a watcher thread with its own lifecycle and race conditions, consistent with the no-daemon stance in [01-architecture.md §3](01-architecture.md#3-process-model-stateless-cli-no-daemon).

## 8. Icons & visual language

One icon-set table, shared by Tree/Board/Detail — no per-view glyph literals. Nerd Font glyphs for item-type, state category, and a priority marker; lipgloss styles the glyph, config `ui.icons: auto|always|never` controls whether it renders at all.

| Concept | Nerd Font glyph | ASCII fallback |
|---|---|---|
| type: epic |  (`nf-oct-milestone`) | `[E]` |
| type: ticket |  (`nf-oct-issue_opened`) | `[T]` |
| category: todo |  (`nf-fa-circle_o`) | `[ ]` |
| category: doing |  (`nf-fa-adjust`) | `[~]` |
| category: done |  (`nf-fa-check_circle`) | `[x]` |
| category: done, `resolution: dropped` |  (`nf-fa-ban`) | `[-]` |
| priority marker (`P0`/`P1` only) |  (`nf-fa-exclamation`) | `!` |

`ui.icons: auto` (default) resolves via one detection algorithm, checked in order `(proposed)`:
1. `$KIRA_ICONS` env var set to `always`/`never` — explicit user override, wins unconditionally.
2. Else, `$TERM_PROGRAM`/`$TERM` against a small allow-list of Nerd-Font-likely terminals (e.g. `TERM_PROGRAM=WezTerm|kitty|iTerm.app`, `TERM` containing `kitty`/`alacritty`) — present → icons on.
3. Else → ASCII.

`always`/`never` on `ui.icons` bypass this algorithm entirely. This is the single detection algorithm — nvim's `icons` toggle ([06-nvim-plugin.md §4](06-nvim-plugin.md#4-visual-integration)) runs the identical check rather than inventing its own, so TUI and nvim never disagree on the same terminal. Every table row's Nerd Font and ASCII forms are defined at the **same fixed display width**, so column alignment in Tree/Board is identical in either mode — the ASCII fallback is a first-class rendering, not a degraded afterthought.

## 9. Rendering constraints

- Tested target: plain terminals (kitty, alacritty) **and** nvim's `:terminal` (used by `:KiraBoard`, [06-nvim-plugin.md](06-nvim-plugin.md)) — no assumptions that break inside a nested terminal emulator (true-color detection, cursor reporting).
- `kira board --plain` / non-tty stdout: static lipgloss table render, no bubbletea event loop — for pipes, CI, and scripting.
- No mouse required for any operation; mouse support optional `(proposed)` — click-to-select on tree/board, no drag.
- `Ctrl-i` is byte-identical to `Tab` (0x09) on legacy terminal encodings (incl. nvim `:terminal`) — never bind it distinctly; forward-jump uses `Ctrl-]`, with `Ctrl-i` as an alias only under negotiated kitty keyboard protocol.
- Test determinism: teatest pins the termenv color profile and background; all time-dependent rendering goes through a clock injected at refresh — goldens are pure functions of fixture + key sequence.
- One shared min-width constant gates Board's detail-peek auto-collapse, the medium layout tier (Tree's detail pane hidden, `Enter` overlays it), and the epic progress-bar collapse — three features, one number.

## 10. Testing hook

teatest (bubbletea's test harness) drives scripted key sequences against fixture repos and asserts golden terminal-cell snapshots for representative states per view (tree drill-in, board card move including a greyed-illegal-target case, detail-pane render in both mount points, popup open/submit/cancel, icon mode vs ASCII-fallback mode). See [09-testing.md](09-testing.md) for the full harness and CI wiring.
