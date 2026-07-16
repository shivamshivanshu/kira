# Manual code review plan

Read bottom-up by dependency layer: each layer only uses layers above it in this list, so nothing forward-references. ~278 files. Mark `- [x]` as you go.

## 1. Domain model — `internal/datamodel` (pure, zero deps)
Verify: struct/tag consistency with docs/design/02; the field-descriptor registry covers every mutable field; no behavior beyond data.

- [ ] internal/datamodel/board.go — Kanban board result types (columns with WIP/items) plus Empty check.
- [ ] internal/datamodel/comment.go — Comment struct (id, author, timestamp, body).
- [ ] internal/datamodel/config.go — Project config schema (workflows, vocab, sprints, git, UI) with VocabFor lookup.
- [ ] internal/datamodel/enums.go — Enum types and allowed values for ID style, commit, merge, icons, estimate, category.
- [ ] internal/datamodel/fields.go — Field descriptor registry driving change-detection, get, and copy across item fields.
- [ ] internal/datamodel/hooks.go — Result types for git hook install/validate status.
- [ ] internal/datamodel/item.go — Core Item model, field keys, type/link/date validators, ParseError.
- [ ] internal/datamodel/progress.go — EpicProgress counter (done/total).
- [ ] internal/datamodel/reconcile.go — Renumbering and reconcile result types.
- [ ] internal/datamodel/sprint.go — Sprint struct plus Config sprint lookup/existence helpers.
- [ ] internal/datamodel/view.go — JSON output DTOs for list/show/diff/stats/log/blame and other command results.
- [ ] internal/datamodel/workflow.go — Workflow/state/transition schema with scalar-transition YAML unmarshalling.
- [ ] internal/datamodel/workon.go — WorkonResult describing branch/worktree creation and state move.

## 2. Leaf utilities — `internal/errx`, `internal/id`, `internal/ptr`
Verify: exit-code taxonomy (1 user / 2 conflict / 3 env / 4 crash); ULID monotonicity assumptions; resolver precedence (full ULID → prefix → number/alias).

- [ ] internal/errx/errx.go — Typed CLI errors with exit codes and hints (User/Conflict/Env/Invalid).
- [ ] internal/errx/suggest.go — Nearest-candidate suggestion via edit-distance threshold matching.
- [ ] internal/errx/suggest_test.go — Tests for Nearest and editDistance.
- [ ] internal/id/id.go — Two-tier identity: mints monotonic ULIDs and parses them.
- [ ] internal/id/number.go — KIRA-n display numbers: parse, allocate next, hash-style generation.
- [ ] internal/id/resolve.go — Resolves tokens (ULID/prefix/number/alias) to ULIDs with ambiguity/not-found errors.
- [ ] internal/id/sortkey.go — SortKey ordering items by numeric number then string then ULID.
- [ ] internal/id/id_test.go — Tests for ULID minting monotonicity and parsing.
- [ ] internal/id/test/number_test.go — Tests for number allocation, parsing, and hashing.
- [ ] internal/id/test/resolve_test.go — Tests for token resolution behavior and errors.
- [ ] internal/ptr/ptr.go — Generic pointer helper Deref returning zero for nil.

## 3. Codec — `internal/codec` (byte-stable frontmatter)
Look hard at: parse→serialize round-trip byte-stability (everything downstream assumes it); CRLF normalization vs the LF-only writer; the unknown-key carrier never leaking into serialization.

- [ ] internal/codec/comments.go — parses, lints, splits, appends kira HTML-marker comment blocks in item bodies
- [ ] internal/codec/parse.go — parses frontmatter+body markdown into Item, collecting all field errors
- [ ] internal/codec/serialize.go — serializes an Item back to canonical frontmatter+body markdown
- [ ] internal/codec/test/codec_test.go — golden round-trip, field-parse, and canonicalization tests for the codec
- [ ] internal/codec/test/property_test.go — randomized round-trip and fuzz tests for parse/serialize idempotence

## 4. Config — `internal/config`
Look hard at: line-splice edits (set/sprintwrite) preserving bytes they don't own; defaults-then-overlay unmarshal semantics; validation completeness vs the write path.

- [ ] internal/config/defaults.go — builds the default Config with workflows, vocab, and settings
- [ ] internal/config/load.go — loads, parses, defaults, and validates .kira/config.yaml
- [ ] internal/config/set.go — edits one config scalar by line-splicing to preserve comments/formatting
- [ ] internal/config/set_test.go — tests scalar-set line preservation, block creation, and rejections
- [ ] internal/config/sprintwrite.go — appends a sprint entry to config.yaml via line-splicing
- [ ] internal/config/validate.go — validates config enums, workflows, vocab, filters, and sprints
- [ ] internal/config/test/config_test.go — tests config parsing, defaults, overrides, and validation rejections
- [ ] internal/config/test/sprintwrite_test.go — tests sprint appending across list shapes with comment preservation

## 5. Storage — `internal/storage`
Look hard at: atomic write (tmp+fsync+rename); flock scope and the 2s waiter timeout; skip-and-warn semantics in LoadAll (one bad file must not brick reads).

- [ ] internal/storage/lock.go — acquires an exclusive flock on the store cache dir with timeout
- [ ] internal/storage/read.go — reads items from disk, loads all tickets sorted, builds id snapshots
- [ ] internal/storage/store.go — discovers .kira store by walking up, exposes store paths
- [ ] internal/storage/ticketpath.go — classifies repo-relative paths and filenames as ticket files
- [ ] internal/storage/write.go — atomically writes an item via temp-file-plus-rename
- [ ] internal/storage/discover_test.go — tests Discover returns the ErrStoreNotFound sentinel
- [ ] internal/storage/test/read_test.go — tests LoadAll sorting, temp-file skipping, and malformed-file warnings
- [ ] internal/storage/test/store_test.go — tests store discovery walking up and env-error on missing .kira

## 6. External-tool wrappers — `gitx`, `rgx`, `fzfx`, `termx`, `editorx`, `clipx`, `showfmt`
Look hard at: NUL-delimited git output parsing (forgery resistance — bodies/authors are attacker-influenced); no shell interpolation anywhere; every subprocess concern stays inside these packages.

- [ ] internal/gitx/attributes.go — Repo.InfoAttributeHasLine: checks if git info/attributes contains an exact line.
- [ ] internal/gitx/branch.go — Repo branch/checkout/worktree ops: current, list, create, checkout, worktree add/lookup.
- [ ] internal/gitx/errors.go — CmdError type and IsCmdError helper wrapping git command failures.
- [ ] internal/gitx/gitx.go — Core Repo type: runs git, stages/commits, cat-file batch, three-way MergeText.
- [ ] internal/gitx/index.go — Repo tree/status helpers: toplevel+HEAD, ancestry check, porcelain status, diff name-status.
- [ ] internal/gitx/show.go — Repo.ShowCmd: builds a `git show <sha>` command for a repo.
- [ ] internal/gitx/staged.go — Repo staged/dirty path listing and showing staged file content.
- [ ] internal/gitx/sync.go — Repo remote/rebase/stash ops: pull-rebase, push, rebase continue/abort, stash push/pop.
- [ ] internal/gitx/trailer.go — Repo.AddTrailer: inserts a commit-message trailer into a file in place.
- [ ] internal/gitx/trailers.go — Repo.LogTrailers: parses commit log into Commit records with ticket/close trailers.
- [ ] internal/gitx/treeish.go — Repo treeish resolution: rev-parse, at-date, merge-base, ls-tree names, no-index numstat.
- [ ] internal/gitx/trailers_test.go — Tests LogTrailers resists forged trailer/record injection from commit bodies.
- [ ] internal/rgx/rgx.go — ripgrep wrapper: Line type, Installed, ParseLine, Search over a directory.
- [ ] internal/rgx/test/rgx_test.go — Tests rgx.ParseLine on match/context/separator ripgrep output lines.
- [ ] internal/fzfx/fzfx.go — fzf wrapper: Installed check and Pick to fuzzy-select a row with preview.
- [ ] internal/termx/termx.go — Terminal helpers: IsTerminal, IsInteractive, Confirm, ReadLineDefault prompts.
- [ ] internal/editorx/editorx.go — Editor launcher: resolves $EDITOR/$VISUAL and opens a file for editing.
- [ ] internal/clipx/clipx.go — Clipboard copy via OSC 52 (tmux-wrapped) plus external tool detection.
- [ ] internal/clipx/test/clipx_test.go — Tests clipboard copy chain matrix and OSC52 tmux-passthrough encoding.
- [ ] internal/showfmt/showfmt.go — Item-reference formatting: id/number/title/markdown/branch forms shared by TUI and CLI.
- [ ] internal/showfmt/showfmt_test.go — Tests showfmt.Format across all forms and branch-slug punctuation collapsing.

## 7. Derived index — `internal/index` (disposable sqlite cache)
Look hard at: the staleness state machine (HEAD SHA + dirty-content hash; fresh/incremental/full/rewrite edges); deleting `.kira/.cache` must change nothing observable; watermark ordering (closes advance only after transitions); error classification (transient git ≠ corruption).

- [ ] internal/index/events.go — Caches and serves per-item change-event logs and commit links, re-deriving on head change.
- [ ] internal/index/index.go — Opens/creates the disposable SQLite cache with WAL pragmas; self-heals on corruption.
- [ ] internal/index/load.go — Load/Refresh entrypoints that reindex then return items, retrying via cache discard.
- [ ] internal/index/meta.go — Reads/atomically writes cache meta.json (schema version, head, dirty state, watermarks).
- [ ] internal/index/paths.go — Order-preserving deduplicating union of path slices.
- [ ] internal/index/read.go — Read-only item scan from the cache, attaching aliases, labels, and links.
- [ ] internal/index/rebuild.go — Full and per-path incremental reindex, inserting items and hashing dirty state.
- [ ] internal/index/schema.go — Declares the SQLite DDL and version, ensuring/validating the schema.
- [ ] internal/index/staleness.go — Decides fresh/incremental/full reindex from head and dirty state; Probe reports freshness.
- [ ] internal/index/trailers.go — Scans commit trailers to link commits to tickets and collect close candidates.
- [ ] internal/index/test/index_test.go — Tests index build, staleness, commit links, event cache, and corruption recovery.

## 8. Distributed-correctness blocks — `merge`, `treeish`, `reconcile`, `hooks`, `workon`, `sync`
Look hard at: LWW tie-breaks (equal timestamps → remote side; rebase inverts ours/theirs); losing side always recoverable from parents; reconcile determinism (earlier ULID keeps its number); hook chaining idempotency.

- [ ] internal/merge/body.go — Three-way merges item prose and unions/dedups/sorts embedded comments.
- [ ] internal/merge/merge.go — Field-level three-way auto-merge of items, reporting arbitrated conflict fields.
- [ ] internal/merge/scalar.go — Three-way scalar/pointer merge with last-writer-wins tiebreak by updated time.
- [ ] internal/merge/set.go — Set-style merges for labels/blocked_by/links and alias union.
- [ ] internal/merge/side.go — Defines Ours/Theirs side enum and the pluggable TextMerger signature.
- [ ] internal/merge/test/merge_test.go — Tests merge policy across scalars, sets, aliases, body, and determinism.
- [ ] internal/treeish/loader.go — Loads items and config from any git tree-ish in one cat-file pass for time-travel.
- [ ] internal/treeish/test/loader_test.go — Tests time-travel load at a treeish and alias resolution.
- [ ] internal/reconcile/reconcile.go — Computes deterministic ID-collision repair plans (renumber later ULIDs).
- [ ] internal/reconcile/test/reconcile_test.go — Tests collision renumbering, determinism, and fuzzed plan invariants.
- [ ] internal/hooks/hooks.go — Embeds git-hook scripts and chains kira's hooks onto existing ones idempotently.
- [ ] internal/hooks/test/hooks_test.go — Tests hook script invocation, classification, shell detection, and chain idempotency.
- [ ] internal/workon/workon.go — Branch naming/slug/matching and active-ticket pointer parsing for workon.
- [ ] internal/workon/test/workon_test.go — Tests slug casing, branch render/match, number inference, active pointer.
- [ ] internal/sync/sync.go — Defines sync report steps, dirty-tree policy, and the reindexer seam interface.
- [ ] internal/sync/test/sync_test.go — Tests ordered report step accumulation and no-op reindexer skip.

## 9. Query language — `internal/query`
Look hard at: predicate/order compile against the field registry; date/priority comparison semantics; nulls-last ordering total-order properties.

- [ ] internal/query/eval.go — Compiles parsed query expressions into item-matching predicates over all fields.
- [ ] internal/query/lexer.go — Tokenizes query input into words, strings, operators, and punctuation.
- [ ] internal/query/order.go — Builds sort keys and comparison for ORDER BY, nulls last.
- [ ] internal/query/parser.go — Parses tokens into a boolean expression tree plus ORDER BY clause.
- [ ] internal/query/lexer_test.go — Tests token kinds, string escapes, positions, and lex errors.
- [ ] internal/query/parser_test.go — Tests canonical parse output and parse error positions.
- [ ] internal/query/test/query_test.go — Tests predicate evaluation, sprint=active, compile errors, and Match.
- [ ] internal/query/test/order_test.go — Tests ORDER BY sorting across priority, rank, dates, estimate, owner.

## 10a. Core: store + read verbs — `internal/core`
Look hard at: single indexed read path with linear-scan fallback + degraded stderr note; the writability guard (unknown keys / CRLF) firing on the loaded original at every mutation entry.

- [ ] internal/core/doc.go — declares core as kira's service layer for every mutation/read.
- [ ] internal/core/store.go — Store type wrapping storage; discover, load, resolve refs, writability guards.
- [ ] internal/core/init.go — Init scaffolds .kira dir, config, templates; derives project key.
- [ ] internal/core/inittemplate.go — holds the default config.yaml template; renders it with key/name.
- [ ] internal/core/inittemplate_test.go — verifies seed config comments are inert and byte-stable on edits.
- [ ] internal/core/view.go — builds ListItem/ShowResult view structs from items and config.
- [ ] internal/core/list.go — List query: filter, compile predicates/order, sort, epic-tree grouping.
- [ ] internal/core/show.go — Show one ticket with linked commits and history tail.
- [ ] internal/core/filter.go — lists configured saved filters; errors on unknown filter names.
- [ ] internal/core/find.go — full-text search across tickets via ripgrep or regex fallback.
- [ ] internal/core/find_test.go — tests find fallback matching, flags, arg parsing, error codes.
- [ ] internal/core/tree.go — builds epic/child hierarchy trees with cycle detection.
- [ ] internal/core/board.go — assembles kanban board columns by workflow state with WIP/counts.
- [ ] internal/core/sprint.go — sprint create/list/activate/close and active-sprint pointer management.
- [ ] internal/core/slices.go — nonNil helper coercing nil slices to empty.
- [ ] internal/core/sort.go — precedence sort key by rank, priority, then number.
- [ ] internal/core/sort_test.go — tests precedence ordering and legacy free-form degradation.
- [ ] internal/core/sortbykey.go — generic stable decorate-sort-by-comparable-key helper.
- [ ] internal/core/active.go — read/write the active workon pointer in cache.
- [ ] internal/core/at.go — historical views: resolve --at treeish/date, load items at that revision.
- [ ] internal/core/discover.go — lists ticket candidates (id/number/title) for completion.
- [ ] internal/core/warnings.go — prints load warnings to stderr.
- [ ] internal/core/workflow.go — workflow state/transition lookups, move targets, category helpers.

## 10b. Core: mutation pipeline
Look hard at: every mutation flowing through mutate.go's lock-resolve-validate-commit; editor input gathered before the lock; comment.go's deliberate append-only bypass staying byte-suffix-clean.

- [ ] internal/core/mutate.go — Generic lock-resolve-validate-commit pipeline shared by all field mutations.
- [ ] internal/core/create.go — Creates new tickets/epics from template, flags, file, or editor.
- [ ] internal/core/edit.go — Edits item fields via --field, file, or full-item editor.
- [ ] internal/core/move.go — Transitions item state, enforcing workflow guards, WIP limits, resolution.
- [ ] internal/core/assign.go — Sets an item's owner or reporter to a user.
- [ ] internal/core/comment.go — Appends a comment as a clean-merging byte suffix, bypassing mutate.
- [ ] internal/core/link.go — Adds/removes epic, blocked-by, and typed links between items.
- [ ] internal/core/draft.go — Draft frontmatter parse/serialize plus editor loop with error banners.
- [ ] internal/core/draft_test.go — Tests draft round-trip, flag overrides, banner strip, editor retry.
- [ ] internal/core/validate.go — Validates item fields against config vocab, workflow, and refs.
- [ ] internal/core/validate_test.go — Tests vocab strictness, field validation parity, field-presence coverage.
- [ ] internal/core/configset.go — Sets a scalar config key and commits the change.
- [ ] internal/core/git.go — Store git helpers: repo access, repo-presence guard, stage/commit finalize.

## 10c. Core: git-derived features
Look hard at: the frontmatter-fence gate in patchdiff (body text must never forge events — feeds stats); log/blame single derivation through the event cache; blame's value-form parity between current and event values.

- [ ] internal/core/gitshow.go — Builds a `git show <sha>` command for the store's repo.
- [ ] internal/core/log.go — Derives per-item field-change history by diffing git frontmatter.
- [ ] internal/core/blame.go — Attributes each field's current value to a commit/creation/synthetic source.
- [ ] internal/core/blame_test.go — Tests blame source classification and merge-loss degraded flagging.
- [ ] internal/core/history.go — Computes per-item state-transition metrics (cycle/done/reopens) from cached events.
- [ ] internal/core/history_test.go — Tests item metrics and cached state-event chronology across histories.
- [ ] internal/core/stats.go — Aggregates completion, cycle/lead, burndown, and velocity statistics over scopes.
- [ ] internal/core/stats_test.go — Tests burndown, velocity, and closed-sprint computations.
- [ ] internal/core/metrics.go — Computes completion, percentiles, throughput, estimate, and reopen rollups.
- [ ] internal/core/metrics_test.go — Tests completion, cycle, lead, throughput, estimate, reopen calculations.
- [ ] internal/core/progress.go — Rolls up epic completion progress over children, cycle-safe.
- [ ] internal/core/progress_test.go — Tests epic progress accumulation with dropped items and cycles.
- [ ] internal/core/closes.go — Closes tickets referenced by commit close-trailers, advancing landed watermark.
- [ ] internal/core/closes_test.go — Tests close application, watermark advancement, and unknown-ticket notes.
- [ ] internal/core/diff.go — Diffs item snapshots between two git treeishes into field changes.
- [ ] internal/core/patchdiff.go — Parses git diff hunks to extract frontmatter field add/remove pairs.
- [ ] internal/core/patchdiff_test.go — Tests that forged body "state:" lines don't fake transitions.

## 10d. Core: merge/sync orchestration + hints
Look hard at: sync's deferred stash-pop on every error path; the unmerged-is-terminal guard in the resolve loop; MergeFile honoring merge.policy per merge.

- [ ] internal/core/mergefile.go — Git merge-driver entry: field-level three-way merge of item files.
- [ ] internal/core/mergedriver.go — Registers the kira git merge driver and gitattributes line.
- [ ] internal/core/reconcile.go — Renumbers colliding item numbers, recording aliases, during doctor/sync.
- [ ] internal/core/resolve.go — Auto-resolves unmerged ticket conflicts, optionally interactive field picking.
- [ ] internal/core/sync.go — Orchestrates stash/commit, pull-rebase, reconcile, reindex, closes, push.
- [ ] internal/core/hooks.go — Installs/validates git hooks and merge driver, chaining existing scripts.
- [ ] internal/core/hookrun.go — Hook runtime: prepare-commit-msg trailer injection and staged-item validation.
- [ ] internal/core/index.go — Loads/refreshes the item index with linear-scan fallback and closes.
- [ ] internal/core/workon.go — Switches to a ticket's branch/worktree and transitions it to doing.
- [ ] internal/core/hints.go — Builds did-you-mean hints for fields, sprints, states, transitions, IDs.

## 10e. Core: black-box tests — `internal/core/test`
- [ ] internal/core/test/board_test.go — Board column ordering, WIP counts, epic-scoping, historical --at, transition adjacency
- [ ] internal/core/test/failsafe_test.go — Forward-compat: reads tolerate unknown keys/CRLF, writes/merge/resolve refuse them
- [ ] internal/core/test/fixture_test.go — Shared test helpers: git repo, store, create/show/edit fixtures
- [ ] internal/core/test/init_test.go — Init scaffolds a parseable config with key, empty vocab, auto-commit
- [ ] internal/core/test/list_test.go — List ordering, flag/saved filters, sprint resolution, sorted filters view
- [ ] internal/core/test/mutations_test.go — Move/comment/link/assign: transitions, guards, WIP warnings, resolutions, vocab
- [ ] internal/core/test/tree_test.go — Tree/list grouping, hierarchy, scoping, cycle detection, query-plus-flag ANDing

## 11. Doctor, seed, testutil
Look hard at: doctor staying pure (IO only in cli); collision Keep semantics matching reconcile (earliest keeps); seed fixtures never producing states a real move can't.

- [ ] internal/doctor/check.go — Checks one item's state/vocab/scalars/refs against config, returns findings.
- [ ] internal/doctor/collision.go — Detects display-number collisions across items (live/alias) with a keeper.
- [ ] internal/doctor/doctor.go — Read-only repo consistency runner aggregating all checks into a Report; also scoped Validate.
- [ ] internal/doctor/env.go — Derives environment/hook/index-freshness findings from an Env snapshot.
- [ ] internal/doctor/epic.go — Detects cycles in the epic parent chain, one finding per member.
- [ ] internal/doctor/finding.go — Defines Finding/Severity/Class/Collision/Report types and report summarization.
- [ ] internal/doctor/lint.go — Parses/lints one item's content: schema errors, unknown keys, bad comments.
- [ ] internal/doctor/test/doctor_test.go — Tests for lint, checks, collisions, epic cycles, Run/Validate, freshness seam.
- [ ] internal/seed/recipe.go — Builds a deterministic size-parametric spec list of epics/tickets from a seed.
- [ ] internal/seed/seed.go — Materializes recipe specs into committed kira item files, returning counts.
- [ ] internal/seed/test/seed_test.go — Tests recipe determinism/shape and that Seed produces a clean committed backlog.
- [ ] internal/testutil/testutil.go — Test helpers to init hermetic git repos.

## 12. CLI adapters — `internal/cli`
Look hard at: commands stay thin (open store → core call → render/json); one error render path; stream discipline (json to stdout, notes/errors to stderr); completion never touching git.

- [ ] internal/cli/cli.go — Builds the cobra command tree, global flags, error rendering, store opening.
- [ ] internal/cli/output.go — Shared JSON encoder and tabwriter output helpers.
- [ ] internal/cli/init.go — `init` command: initialize a .kira store in the repo.
- [ ] internal/cli/create.go — `create <type>` command with field flags and template printing.
- [ ] internal/cli/edit.go — `edit` command: apply field edits or round-trip an item file.
- [ ] internal/cli/move.go — `move` command: transition an item to a new state.
- [ ] internal/cli/assign.go — `assign` command: set an item's owner or reporter.
- [ ] internal/cli/comment.go — `comment` command: append a comment to an item.
- [ ] internal/cli/complete.go — Registers shell completions for commands, args, and vocab flags.
- [ ] internal/cli/link.go — `link` command: add/remove epic, blocked-by, or typed links.
- [ ] internal/cli/list.go — `list` command with filter flags and row/tree rendering.
- [ ] internal/cli/show.go — `show` command: render one item, optionally at a ref or reference-form.
- [ ] internal/cli/find.go — `find` command: ripgrep-wrapping full-text search over ticket files.
- [ ] internal/cli/filter.go — `filter list` command: list named saved queries from config.
- [ ] internal/cli/query.go — `query` command: filter items by a query expression (tree/flat).
- [ ] internal/cli/tree.go — `tree` command and shared epic-hierarchy/tree-group rendering.
- [ ] internal/cli/board.go — `board` command: kanban board, launching TUI or static table.
- [ ] internal/cli/sprint.go — `sprint` create/list/activate/close subcommands.
- [ ] internal/cli/log.go — `log` command: item field history interleaved with commits.
- [ ] internal/cli/blame.go — `blame` command: per-field current value and last-setting commit.
- [ ] internal/cli/stats.go — `stats` command: completion, cycle/lead time, throughput, burndown, velocity.
- [ ] internal/cli/diff.go — `diff` command: semantic backlog change from merge-base to a ref.
- [ ] internal/cli/sync.go — `sync` command: pull-rebase/reconcile/reindex and optional push.
- [ ] internal/cli/resolve.go — `resolve` command: auto-resolve conflicted items during a merge.
- [ ] internal/cli/mergefile.go — Hidden `merge-file` command: git merge-driver entry point for items.
- [ ] internal/cli/hooks.go — `hooks` install plus hidden hook entry points (post-merge, prepare-commit-msg, pre-commit).
- [ ] internal/cli/index.go — `index` command: refresh or rebuild the derived cache index.
- [ ] internal/cli/discover.go — `discover` command: interactively pick an item via fzf or bubbles.
- [ ] internal/cli/doctor.go — `doctor` command: run repo checks and render/exit on the report.
- [ ] internal/cli/doctorenv.go — Gathers the doctor Env (git, hooks, merge driver, index freshness) from disk.
- [ ] internal/cli/config.go — `config set` command: set a scalar config key preserving formatting.
- [ ] internal/cli/validate.go — `validate` command: check given ticket files against schema/config.
- [ ] internal/cli/workon.go — `workon` command: switch to/create a per-ticket branch or worktree.
- [ ] internal/cli/tui.go — `tui` command and TUI launch, uninit-offer, and in-process command runner.
- [ ] internal/cli/version.go — `version` command printing the link-time build version.
- [ ] internal/cli/tui_test.go — Tests the in-process command runner drives core services and reports errors.

## 13. TUI — `internal/tui` (+ theme)
Look hard at: crash containment (safeCmd + guardRun + panic-proof handler → exit 4); no store I/O in View(); per-item detail memo invalidation; the theme package as the only color home (a test enforces it).

- [ ] internal/tui/app.go — Root Bubbletea model: view switching, key routing, title/footer/help rendering.
- [ ] internal/tui/run.go — Public Run/Options entry point; wires store, theme, program, panic guard.
- [ ] internal/tui/registry.go — screen interface and per-view screen factory registry.
- [ ] internal/tui/keys.go — KeyBinding type; formats keys into hint line and help body.
- [ ] internal/tui/cmd.go — Async tea.Cmd builders loading tree/detail data with panic recovery.
- [ ] internal/tui/cmdbar.go — Command/filter input bar: tokenizing, running commands, footer rendering.
- [ ] internal/tui/crash.go — Crash handling: restores terminal, writes crash log, reports to stderr.
- [ ] internal/tui/layout.go — Width thresholds deciding split-detail layout and tree pane width.
- [ ] internal/tui/scroll.go — Frame style and scrollable viewport line rendering helper.
- [ ] internal/tui/util.go — Small helpers: clamp integers and truncate strings to width.
- [ ] internal/tui/empty.go — Constant empty-tree placeholder message.
- [ ] internal/tui/uninit.go — Standalone TUI prompting for project key to initialize kira.
- [ ] internal/tui/boardscreen.go — Board screen: column/card navigation, moves, peek/overlay detail panel.
- [ ] internal/tui/board.go — Board model and kanban column/card rendering, plus plain export.
- [ ] internal/tui/treescreen.go — Tree screen: navigation, collapse, pane focus, detail sync/caching.
- [ ] internal/tui/tree.go — Tree model: flatten hierarchy, cursor/scroll, row rendering with progress.
- [ ] internal/tui/detail.go — Formats a ticket's metadata line (state, owner, priority, labels).
- [ ] internal/tui/detailfull.go — Detail panel: scroll, commit selection, git show, cached content lines.
- [ ] internal/tui/filter.go — Loads filtered tree, pruning nodes to matches and their ancestors.
- [ ] internal/tui/stats.go — Stats screen rendering burndown sparklines and velocity bars.
- [ ] internal/tui/jumplist.go — Bounded back/forward jump history of view+item navigation entries.
- [ ] internal/tui/markdown.go — Renders markdown to plain terminal text via cached glamour renderers.
- [ ] internal/tui/progress.go — Builds epic progress bar and done/total label.
- [ ] internal/tui/sparkline.go — Sparkline and horizontal bar glyph rendering from float series.
- [ ] internal/tui/icons.go — Nerd-vs-ASCII glyph set selection for types, categories, priorities.
- [ ] internal/tui/yank.go — Copy selected item's IDs/forms to clipboard via picker.
- [ ] internal/tui/theme/theme.go — Central color/style theme: adaptive palette, category/priority styles, renderer.
- [ ] internal/tui/cmdbar_test.go — Tests command-bar tokenizing, argv forwarding, refresh, errors, nested-TUI rejection.
- [ ] internal/tui/crash_test.go — Tests crash handling: terminal restore, report lines, log fallback.
- [ ] internal/tui/detailfull_test.go — Tests detail panel rendering, commit selection, caching, scroll clamp.
- [ ] internal/tui/filter_test.go — Tests node pruning and filter narrowing preserved across refresh.
- [ ] internal/tui/board_render_test.go — Golden/behavior tests for board rendering, WIP tint, moves, peek.
- [ ] internal/tui/render_test.go — Tree view golden snapshots, icon modes, detail memo tests.
- [ ] internal/tui/sparkline_test.go — Tests sparkline ramp mapping and horizontal bar scaling.
- [ ] internal/tui/stats_test.go — Tests stats screen rendering, empty state, caching, invalidation.
- [ ] internal/tui/uninit_test.go — Tests uninit flow creating .kira on key confirm, rejecting empty.
- [ ] internal/tui/yank_test.go — Tests yank copying selected ID and picker form choices via OSC52.
- [ ] internal/tui/fixture_test.go — Test helpers building git repos, stores, and tickets.
- [ ] internal/tui/theme/test/theme_test.go — Tests no-color plain output, adaptive background colors, priority fallback.

## 14. Entrypoint
- [ ] cmd/kira/main.go — Binary entrypoint; calls cli.Main and exits with its code.

## 15. Test harnesses
Look hard at: the contract scrubber (ULID/timestamp/SHA/dir normalization — would a new nondeterminism slip through?); the merge regression matrix's byte-identical sync-vs-driver invariant; perf harness spawn counting.

- [ ] tests/contract/contract_test.go — Execs kira binary, compares each command's --json/error output to golden files.
- [ ] tests/e2e/e2e_test.go — Runs testscript-based end-to-end command scripts against kira.
- [ ] tests/e2e/complete_spawn_test.go — Asserts shell-completion serves from cache without spawning git.
- [ ] tests/e2e/crash_exit_test.go — Asserts injected TUI panic exits with code 4.
- [ ] tests/e2e/find_discover_test.go — Testscript find/discover flows with rg/fzf deliberately absent.
- [ ] tests/e2e/uninit_exit_test.go — Asserts bare TUI on non-tty uninitialized repo exits 3.
- [ ] tests/integration/integration_test.go — Verifies auto/manual commit modes, editor round-trips, list filters.
- [ ] tests/integration/diff_test.go — Verifies diff detects deletions, body changes, non-alias number edits.
- [ ] tests/integration/merge_test.go — Verifies merge driver byte-parity, conflict resolution, manual-policy behavior.
- [ ] tests/integration/matrix_test.go — Merge regression matrix across sync/driver paths and policies.
- [ ] tests/nocolor/nocolor_test.go — Fails if color literals appear outside the theme package.
- [ ] tests/perf/doc.go — Package doc for the non-gating perf-budget harness.
- [ ] tests/perf/harness_test.go — Shared perf harness: builds binary, seeds fixtures, counts git spawns.
- [ ] tests/perf/bench_test.go — Benchmarks core read commands against a seeded fixture.
- [ ] tests/perf/coldstart_test.go — Tripwire asserting `kira version` cold start stays under 50ms.
- [ ] tests/perf/scaling_test.go — Reports git-spawn growth across fixture sizes (non-gating).
- [ ] tests/perf/spawn_test.go — Reports and checks deterministic per-command git-spawn counts.

## 16. Design docs (read last — verify code matches spec)
- [ ] docs/design/01-architecture.md — System shape: process model, index, single-write-path guarantee.
- [ ] docs/design/02-data-model.md — Item schema, edges, comments, state machine, ID scheme.
- [ ] docs/design/03-storage-and-git.md — `.kira/` layout, write invariants, history derivation, merge strategy.
- [ ] docs/design/04-cli.md — Every subcommand: flags, JSON shapes, query/find/discover subsystems.
- [ ] docs/design/05-tui.md — Bubbletea frame, panels, keymaps, lazygit-style interaction model.
- [ ] docs/design/06-nvim-plugin.md — Thin Lua `--json` frontend: modules, features, config, health.
- [ ] docs/design/07-git-integration.md — Commit-trailer linking, incremental scan, hooks, sync, collision repair.
- [ ] docs/design/08-telemetry.md — Metric definitions, edge cases, `kira stats` output contract.
- [ ] docs/design/09-testing.md — Risk-ranked test pyramid, merge regression matrix, CI shape.
