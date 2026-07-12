# nvim Plugin

**Scope:** the Lua frontend — module layout, features, visual integration, config surface, and error handling. The plugin is a `--json` consumer of the `kira` binary; it is not a second implementation of anything.
Part of the kira design set — see [DESIGN.md](../../DESIGN.md) for decisions and rationale.

## 1. Architecture

Pure Lua, no compiled component. Every data operation is an async `vim.system({'kira', ..., '--json'}, {text = true}, callback)` call, decoded with `vim.json.decode`. The plugin **never** parses `.kira/tickets/*.md`, `config.yaml`, or any frontmatter, and holds no business logic (state-machine legality, ID resolution, merge/collision handling) — all of that lives once, in Go (`internal/core`), reused by CLI and TUI. This is the single most-repeated argument across the design's source material: two parsers (Go + Lua) drift; one doesn't.

No plugin-side cache — every read is a fresh CLI call. The plugin is thin enough that this costs single-digit milliseconds, which is exactly why the CLI's cold-start budget in [01-architecture.md §2](01-architecture.md#2-language--libraries) matters.

Floor: **nvim ≥ 0.10** (`vim.system` availability). Rejected `plenary.job`: it would be the plugin's only non-optional dependency, for a job-spawning primitive nvim now ships natively — zero-dep wins.

## 2. Module layout

| Module | Responsibility |
|---|---|
| `lua/kira/init.lua` | `setup(opts)` — merges user config over defaults (§5), registers user commands (`:KiraCreate`, `:KiraEdit`, `:KiraDiscover`, `:KiraQuery`, `:KiraBoard`), registers the `gk`/`gp` keymaps, sets up the virtual-text autocmd if enabled. |
| `lua/kira/cli.lua` | The **only** module that shells out. Thin wrapper: builds argv, calls `vim.system`, decodes JSON on exit, normalizes nonzero-exit into `(nil, err)`. Every other module calls through here — never `vim.system` directly. |
| `lua/kira/ui.lua` | Floating window stack: opening/pushing/popping ticket-view floats, rendering decoded JSON into a scratch buffer (`filetype=markdown` for treesitter highlighting), the linked-commit list and its `git show`/diffview dispatch, devicons glyphs in float content when enabled (§4, §5). |
| `lua/kira/picker.lua` | `:KiraDiscover`/`:KiraQuery` picker registration: telescope extension if `pcall(require, 'telescope')` succeeds, `vim.ui.select` fallback otherwise; devicons type/state glyphs on entries when enabled (§4, §5). |
| `lua/kira/health.lua` | `:checkhealth kira` — binary on `PATH`, version handshake, `.git/hooks/post-merge` installed, index freshness. |

## 3. Features

### `gk` — universal ticket jump (the payoff feature)

- **Trigger:** the `gk` normal-mode keymap (configurable, §5), in **any** buffer/filetype — code comments, `COMMIT_EDITMSG`, plain docs, pasted logs. No LSP, no filetype restriction.
- **Behavior:** match a `<key>-\S+`-shaped token under the cursor (a custom pattern scan, not `<cword>` — `-` breaks word boundaries). On match, async-call `cli.show(token)`; on success, push a floating window (`ui.lua`) rendering the ticket.
- **CLI call:** `kira show <token> --json` — the token resolves via the CLI's own ID-resolution order (ULID / prefix / number / alias — see [04-cli.md §1](04-cli.md#1-principles)), so the plugin never needs to know or care which form it is.
- **Project key:** `<key>` is never hardcoded as `KIRA` — `health.lua` fetches `project.key` once via `kira config get project.key --json` at setup (re-checked on `:checkhealth kira`) and caches it for the session; every token pattern below (`gk`, virtual-text, mini.hipatterns §4) is built from that cached value, not a literal.
- **UI:** floating window, header line (number, title, state), glamour-equivalent rendering via `filetype=markdown` + treesitter (no glamour dependency in Lua — that's the Go binary's job), then body / acceptance criteria / comments / linked commits / history tail, exactly the fields in `kira show --json`'s payload ([04-cli.md `kira show`](04-cli.md#kira-show)).

### Floating view stack

- `gp` inside a kira float: calls `cli.show(current.epic)`, pushes a new float on top of the stack (previous float hidden, not closed).
- `q` / `Esc`: pop one level; stack empty → close the window entirely.
- Linked-commit lines: cursor-on-line + `Enter` parses the leading `sha`, opens `git show <sha>` in a scratch buffer (via `vim.system({'git','show',sha})`, no pager). If `pcall(require, 'diffview')` succeeds, prefer `:DiffviewOpen <sha>^..<sha>` instead — an optional, pcall-detected integration, never a hard dependency.
- Repo-relative path tokens appearing in float content (e.g. `bench/burst_test`) are `gf`-openable in a split — nvim's native `gf` machinery over `'path'`/`'suffixesadd'`, no custom file-link handling needed.

### `:KiraCreate` / `:KiraEdit`

- `:KiraCreate [ticket|epic]` opens a scratch buffer prefilled with the template text from `kira create ticket --print-template` ([04-cli.md `kira create`](04-cli.md#kira-create-ticket--kira-create-epic)) — this is why the plugin never reads `.kira/templates/*.md` off disk directly, keeping "zero .kira parsing in Lua" absolute even for prefill text. `:KiraEdit <id>` prefills from `kira show <id> --json`'s `body`/frontmatter, reconstructed into editable form by the same Go writer the file itself uses (not by the plugin).
- `:w` on that scratch buffer (`BufWriteCmd`) does **not** write a real file — it shells `kira create ticket --from-file <tmpfile> --json` (or `kira edit <id> --from-file <tmpfile> --json` for `:KiraEdit`), writing the buffer to a real temp file first. Validation happens entirely in `internal/core`, identically to the CLI's own `--from-file` path ([04-cli.md `kira edit`](04-cli.md#kira-edit)).
- Success: close the scratch buffer, open the resulting item via the same float used by `gk`.
- Validation error: `kira create`/`edit --from-file` exits nonzero with a `kira validate`-shaped error list (`{"errors": [{"field", "message"}]}` — see [04-cli.md `kira validate`](04-cli.md#kira-validate)). Surfaced two ways `(proposed)`: a quickfix list (`setqflist`, one entry per error, jumpable) as the primary navigation aid for multiple errors, plus virtual text on the offending line for at-a-glance context — the buffer stays open, uncommitted, for the user to fix and `:w` again.

### `:KiraDiscover` / `:KiraQuery`

- Candidate source is `kira list --json` / `kira query "<expr>" --json` (never `kira discover`, which has no `--json` — it's an interactive fzf selector in its own right, see [04-cli.md `kira discover`](04-cli.md#kira-discover)).
- Telescope picker (custom finder + `kira show <id> --json`-backed previewer) if `pcall(require, 'telescope')` succeeds; otherwise `vim.ui.select` over item `"KIRA-142  Fix race in ..."` labels, no preview.
- Selecting an entry opens the `gk` float (view) by default; a picker action (telescope `<C-e>` or a second `vim.ui.select` prompt) routes to `:KiraEdit` instead.

### Virtual-text state hints

- Toggleable (`virtual_text.enabled`, §5). On `BufEnter` + debounced `TextChanged`/`CursorMoved` (debounce `virtual_text.debounce_ms`, default 300ms `(proposed)`), scan visible lines for `<key>-\S+` tokens, dedupe, and resolve each via `cli.show` (concurrent `vim.system` calls, bounded by what's on screen — no batch API needed since the visible window is small). Render `[STATE]` as virtual text at end-of-line next to each recognized token.

### `:KiraBoard`

`:terminal kira board [<epic-id>]` — opens the **one** kanban implementation ([05-tui.md §4](05-tui.md#4-kanban-board-view-2)) inside an nvim terminal buffer; no separate rendering path in Lua. `--plain` is the fallback the CLI itself already provides for non-tty contexts ([04-cli.md `kira board`](04-cli.md#kira-board)) — the plugin doesn't need its own fallback logic, it just doesn't force `--plain` since `:terminal` is a real tty.

## 4. Visual integration

State highlighting and iconography are visual sugar on top of the same `--json` data the plugin already fetches — never a new data path.

### mini.hipatterns state highlighting

- Optional, `pcall(require, 'mini.hipatterns')`-detected, same pattern as telescope/diffview (§3).
- kira defines its own highlight groups, one per state *category* plus one for the ticket-number token itself, linked to sensible defaults and overridable by colorschemes/user:

| Group | Meaning | Default link |
|---|---|---|
| `KiraStateTodo` | category `todo` | `Todo` |
| `KiraStateDoing` | category `doing` | `DiagnosticWarn` |
| `KiraStateDone` | category `done` | `DiagnosticOk` |
| `KiraStateDropped` | category `done`, `resolution: dropped` | `Comment` |
| `KiraTicketNumber` | any `<key>-\S+` token | (colorscheme default, no link) |

- When `mini.hipatterns` is present: register its highlighter patterns for (a) `<key>-\S+` tokens (§3's cached `project.key`) → `KiraTicketNumber`, and (b) each configured state *keyword* → its category's group. State keywords are sourced dynamically from `kira config get workflows --json` (nested-subtree shape spec'd in [04-cli.md `kira config`](04-cli.md#kira-config)) — never hardcoded, since states are user-configurable per [02-data-model.md §6](02-data-model.md#6-state-machine); `TODO`/`IN_PROGRESS`/`REVIEW`/`DONE`/`WONT_DO` are this project's config, not the plugin's assumption.
- Fallback when `mini.hipatterns` is absent: plain `nvim_buf_set_extmark` highlighting applied from the plugin's existing token-recognition pass — the same scanner that drives `gk` and the virtual-text hints (§3) is reused here as a second consumer, not reimplemented.
- Category→group mapping is the invariant that must never hardcode state *names* — only categories, matching the state-machine's own telemetry rule ([08-telemetry.md](08-telemetry.md)).

### nvim-web-devicons

- Optional, `pcall(require, 'nvim-web-devicons')`-detected.
- Used for: file-path links rendered inside floating ticket views (§3 floating view stack), item-type icon (epic vs ticket) and state glyph in picker entries (§3 `:KiraDiscover`/`:KiraQuery`) and floats.
- Config toggle `icons = "auto"|"always"|"never"` (§5) — `auto` enables icons when devicons is present *and* the environment passes the same detection algorithm as [05-tui.md §8](05-tui.md#8-icons--visual-language) (`$KIRA_ICONS` override → `$TERM_PROGRAM`/`$TERM` allow-list → else ASCII).
- Every glyph has an ASCII fallback (`[E]`/`[T]` for type, `[ ]`/`[~]`/`[x]`/`[-]` for category) so nothing breaks without a patched font — same icon-set table as [05-tui.md §8](05-tui.md#8-icons--visual-language), one source shared across TUI and nvim.

## 5. Config surface

```lua
require('kira').setup({
  bin = 'kira',                -- binary name or absolute path, resolved via PATH if bare
  keymaps = {
    jump   = 'gk',
    parent = 'gp',
    close  = 'q',
  },
  virtual_text = {
    enabled     = true,
    debounce_ms = 300,
  },
  icons = 'auto',               -- 'auto' | 'always' | 'never' — devicons + hipatterns glyphs, §4
  float = {
    width  = 0.8,               -- fraction of editor width
    height = 0.8,
    border = 'rounded',
  },
})
```

All defaults `(proposed)`. `setup()` is idempotent and optional — calling `require('kira')` with no `setup()` uses these defaults.

## 6. Error UX

`cli.lua` is the single chokepoint: every `vim.system` call's `on_exit` checks the exit code. Nonzero → `vim.notify(stderr or "kira: command failed", vim.log.levels.ERROR)`; the plugin never raises an uncaught Lua error from a CLI failure and never blocks the editor waiting on one — every call is async, UI-affecting work wrapped in `vim.schedule`.

## 7. Testing

plenary busted, headless, against a fixture repo (a small `.kira/` tree with a handful of tickets/epics under a temp dir) — see [09-testing.md](09-testing.md). Unit-level tests exercise `cli.lua`'s JSON decode and error-normalization against a real `kira` binary built in CI (not mocked, consistent with the project's testscript philosophy of testing against the real thing); UI-level tests (`ui.lua` float stack, `picker.lua` fallback selection) may stub `cli.lua`'s return values directly `(proposed)` since their concern is buffer/window state, not CLI behavior.
