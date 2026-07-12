# Testing Strategy

**Scope:** test pyramid mapped to actual risk surfaces, the merge regression matrix, and CI shape.

Part of the kira design set — see [DESIGN.md](../../DESIGN.md) for decisions and rationale.

## 1. Risk-ranked pyramid

Ranked by what silently regresses worst if untested, highest first:

1. **ID reconciliation** — a bug here misnumbers or duplicates tickets, or (worse) corrupts a cross-reference. Highest blast radius, lowest visibility (wrong by one, not a crash).
2. **Merge policy engine** (`merge.policy: auto`'s field-level three-way merge, and its `kira merge-file` driver twin) — this is what stands between the user and raw git conflict markers by default; a bug here can silently pick the wrong winner, or worse, drop a write with no trace. Same severity class as ID reconciliation, same testing rigor (property tests, not example tests).
3. **Byte-stability / merge behavior** — the "git is our database" claim rests on the frontmatter writer producing minimal diffs; a regression here silently reintroduces conflicts the layout (and the policy engine above) were designed to avoid or auto-resolve cleanly.
4. **`--json` contract** — the nvim plugin and any script depend on exact key names/types; a drift breaks a consumer kira's own test suite never sees.
5. **Index staleness** — wrong data rendered confidently is worse than a crash.
6. **TUI / nvim** — real, but lowest risk: both are thin consumers of the above, a bug here is visible immediately in normal use, not silent.

## 2. Unit tests

- **Frontmatter round-trip, byte-stability golden files.** Load a fixture ticket, apply one field edit via the writer, assert the resulting diff against the *original file* is exactly one changed line. This test **enforces** the one-key-per-line, fixed-key-order merge design (§ Merge strategy in DESIGN.md) — it is not incidental coverage, it is the mechanism that makes multi-field concurrent edits merge cleanly, made concrete as an assertion.
- **State-machine transition validation.** For every configured workflow, assert `kira move` accepts every edge in `transitions[]` and rejects every non-edge (exhaustive over the adjacency map, not spot-checked), plus `--force` bypasses the check but still writes/commits normally.
- **ID allocation + reconciliation**, property-based/fuzz tested. Invariants, checked over randomly generated concurrent-create/merge scenarios:
  - resolution is deterministic (same input state → same renumbering, no ordering nondeterminism)
  - no ticket is ever lost (item count before/after reconciliation is invariant)
  - ULIDs are never touched (only `number`/`aliases` mutate)
  - `aliases[]` is append-only (never rewritten or truncated)
- **Merge policy engine** (`merge.policy: auto`), property-based, over randomly generated `(base, ours, theirs)` field-value triples per field kind:
  - deterministic: same three inputs → same output, every run (the exact-tie rule — incoming/remote side wins, defined by absolute local/remote role, not by git's merge-vs-rebase stage labeling — is exercised explicitly, not left to map/slice ordering)
  - no field is silently lost except a documented LWW loser, and every loser is recoverable from the two merge parents via `kira log` (assert the losing value appears in at least one parent commit)
  - `created` is never mutated by a merge; `updated` is always `max(ours.updated, theirs.updated)`
  - list-field merges never drop an add from either side and always honor an explicit removal (add/remove interleavings are the harder property case, generate them deliberately, not incidentally) — **except `aliases`: union-only, a removal is never honored regardless of base, since alias-forever resolution depends on it**
  - the `kira resolve` (index-stage) code path and the `kira merge-file` (driver, temp-file) code path produce byte-identical merged output for the same three inputs — one policy engine, two entry points, not two implementations

## 3. Golden-file `--json` contract tests

One golden file per command's `--json` output, regenerable via a `-update` flag (write current output as the new golden, for reviewed intentional changes). This is the nvim plugin's contract test suite in effect, even though nvim itself never runs in CI — if a command's JSON shape changes without a corresponding golden-file update in the same PR, CI fails.

## 4. E2E: testscript

`rogpeppe/go-internal/testscript` against real temporary git repos. Chosen because it's the Go toolchain's own tool for exactly this shape of test — spawn a binary, assert on files and stdout, no custom harness to maintain.

Covers: `init`/`create`/`move`/`hooks install`/`doctor` end to end against a real repo, plus a dedicated **merge regression matrix** — the precise scenarios the layout design (file-per-ticket, single-sided edges, one-key-per-line YAML, append-only comments) and the merge-policy engine claim to handle:

| Branch A change | Branch B change | Expected outcome |
|---|---|---|
| edits ticket X | edits ticket Y (different ticket) | clean merge |
| edits field `owner` on ticket X | edits field `labels` on ticket X (different field, same ticket) | clean merge |
| edits field `state` on ticket X | edits field `state` on ticket X (same field, same ticket), `merge.policy: auto` | auto-resolved LWW by `updated`, reported (field + winning side), no conflict markers shown to the user |
| edits field `state` on ticket X | edits field `state` on ticket X (same field, same ticket), `merge.policy: manual` | surfaced git conflict → `kira resolve --interactive` flow |
| adds a comment to ticket X | adds a different comment to ticket X (concurrent comments) | clean merge (disjoint appended blocks; also clean under either policy) |
| creates a new ticket (gets `KIRA-142`) | creates a different new ticket (also gets `KIRA-142`, concurrent creates) | clean merge at the git level, number collision surfaced by `post-merge` → `doctor --fix` renumbers the later-created ULID, old number appended to its `aliases:`, repair committed |
| plain `git merge`/`git rebase` (merge driver registered, no `kira sync` involved) on the same-field-same-ticket case | — | identical output to the `kira sync` row above, asserted byte-for-byte — proof the driver and `kira resolve`/`sync` share one policy engine, not a second implementation |
| edits field `state` on ticket X, **identical `updated` timestamp** on both sides (exact tie) | resolved once via `kira sync` (`git pull --rebase`, so git's stage 2/3 "ours"/"theirs" are locally-inverted relative to a plain merge) and once via a plain `git merge` with the driver registered, same two branches | the **same absolute side** (the one that was actually "incoming"/remote in the real-world scenario, not whichever stage label git happened to assign) wins in both runs — asserted byte-for-byte identical, proving the rebase-vs-merge stage inversion is correctly compensated for rather than silently flipping the outcome |
| branch A `git rm`s ticket X | branch B edits ticket X (modify/delete — the merge driver is never invoked for this pair, git only invokes a driver when both sides modified the blob) | edit wins by default (the file survives); the deletion attempt is reported in the merge/sync output; not auto-resolved mid-merge — this is the one conflict class `merge.policy: auto` cannot intercept, and is cleaned up after the fact via `kira resolve`, per [03-storage-and-git.md §7](03-storage-and-git.md#7-merge-strategy) |

Each row is asserted against a real `git merge` (not a simulated one) — testscript spins up two branches in a temp repo, performs the merge, and asserts on the resulting working tree/exit code/stderr.

Note: the driver's `.gitattributes` glob is `.kira/tickets/*.md merge=kira` only — `.kira/templates/*.md` is plain markdown with no Item frontmatter, so a concurrent edit to a template merges via ordinary git text merge, never the kira driver. A regression test asserts a template-file conflict still produces raw git conflict markers (proving the glob hasn't over-matched), independent of `merge.policy`.

## 5. TUI and nvim

- **TUI**: `teatest` keypress-sequence tests driving the bubbletea program, asserting on rendered-frame snapshots (tree navigation, board column moves, ticket detail).
- **nvim**: `plenary` busted, headless, against a fixture `.kira/` repo — `gk` token recognition, floating preview, `:w` round-trip via `--from-file`, parent-jump stack.

## 6. CI

- GitHub Actions, **linux + macOS matrix** (Windows untested per the v1 non-goals — Go + shell-out design means it isn't actively broken, just not gated on).
- `go test -race`, `go vet`, `golangci-lint` as the default job.
- `testscript` suite as a **separate job** (slower — spins up real git repos per case).
- **Dogfood check**: `kira doctor` run against kira's own `.kira/` directory in the repo — the project tracks its own work in itself, so this is a free continuous integration test of `doctor`'s invariants against real usage data.
- **Deterministic editor-flow tests**: `EDITOR=true` (a no-op command that exits 0 without modifying the file) for any test path that would otherwise block on an interactive `$EDITOR` — makes the create/edit flows fully scriptable in CI.

## 7. Quality gates

Every milestone (M0–M6, see [DESIGN.md](../../DESIGN.md) / `ROADMAP.md`) ends with a named green test suite before the next milestone starts — the specific suites are enumerated per milestone in `ROADMAP.md`, not duplicated here. M3 ("distributed correctness") is explicitly gated on the full merge regression matrix (§4) passing before recommending multi-branch/multi-user use.
