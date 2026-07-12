# Telemetry

**Scope:** metric definitions, edge-case handling, and the `kira stats` output contract.

Part of the kira design set — see [DESIGN.md](../../DESIGN.md) for decisions and rationale.

## 1. Data source

Everything below is computed on demand from two inputs, nothing else:

1. **The index** (`.cache/index.db`) — current item state, frontmatter fields, `created`/`updated` timestamps.
2. **The git-derived transition-event stream** — populated during incremental reindex by replaying `git log --follow -p` per ticket file (same mechanism as [`kira log`](04-cli.md#kira-log)) and caching the resulting field-change events in `index.db`'s event cache. No telemetry data is stored independently of this cache; deleting `.cache/` loses zero information — the next reindex rederives it.

Category and resolution membership (`todo`/`doing`/`done`, `resolution: dropped`) come **only** from the `category`/`resolution` tags in `config.yaml`'s workflow definitions — never from matching on state *names*. This is what lets a team rename `IN_PROGRESS` to `DOING` without breaking cycle-time history: the category tag on the state definition is what telemetry keys off, not the string.

## 2. Metric definitions

| Metric | Formula |
|---|---|
| **Completion %** | `count(category=done) / count(total)`, recursive over an epic's subtree. `resolution: dropped` items are excluded from the numerator and reported separately as "resolved, not done" (not simply excluded from the denominator — a dropped ticket is a completed decision, not an open one). |
| **Cycle time** | first transition event into a `doing`-category state → first transition event into a `done`-category state, per item. Aggregated as p50/p90 across the item set. |
| **Lead time** | `created` (frontmatter) → first transition into a `done`-category state, per item. p50/p90. |
| **Throughput** | count of `done`-category transition events per week, trailing N weeks (`--weeks N`). |
| **Estimate rollup** | sum of `estimate` over the item set (only where populated). **Estimate-vs-actual ratio** = `estimate / actual`, defined **only when `estimate.unit: hours`** — `actual` = cycle-time (days) × `estimate.hours_per_day` (config, default `8`), i.e. calendar hours spent in `doing`-category states, not effort hours. Under `estimate.unit: points` there is no defined days→points conversion, so the ratio is omitted entirely rather than guessed — see §4. |

## 3. Edge cases

| Case | Handling |
|---|---|
| **Reopened items** (re-entry into a `doing`-category state after a `done`-category state) | Cycle time still uses the item's *first* entry into `doing` → *first* entry into `done` (unaffected). A separate `reopen_count` field is tracked and surfaced per item and in aggregate. *(proposed)* |
| **Squashed history** (feature-branch commits collapsed on merge) | Transition *events* between the collapsed commits are lost — cycle time becomes best-effort (may skip intermediate `doing`→`doing` transitions, degrading p50/p90 precision for that item). Frontmatter `created`/`updated` timestamps survive any squash (they live in the file's current content, not history), so **lead time is robust**; only cycle time degrades. State this explicitly in `stats` output when squash-loss is detected (created/updated gap with no matching event) rather than silently reporting a number. |
| **Items created already-done** | Cycle time is undefined (no `doing`-category entry precedes the `done`-category one) — excluded from the cycle-time set, still counted in completion % and lead time (lead time is 0 or near-0, which is correct). |
| **`WONT_DO` / `resolution: dropped` exclusion** | Excluded from cycle-time and lead-time aggregates entirely (they never meaningfully progressed through the workflow); counted in completion %'s "resolved, not done" bucket only. |

## 4. Output

```
kira stats [<epic-id>] [--since DATE] [--weeks N] [--json]
```

Human render: lipgloss table (completion %, cycle/lead p50/p90, throughput) plus a sparkline for the trailing-N-week throughput series.

`--json` is the actual deliverable — the target workflow is exporting into pandas for the user's own analysis, not consuming kira's own charts:

```json
{
  "scope": {"epic": "01J8X7B1Q2W3E4R5T6Y7U8I9O0", "epic_number": "KIRA-100", "since": "2026-06-01", "weeks": 8},
  "completion": {"done": 12, "total": 20, "dropped": 2, "pct": 0.60},
  "cycle_time_days": {"p50": 2.1, "p90": 6.4, "n": 15, "degraded_n": 1},
  "lead_time_days": {"p50": 4.8, "p90": 11.2, "n": 18},
  "throughput_per_week": [1, 3, 2, 0, 4, 2, 1, 3],
  "estimate": {"total": 45, "unit": "points"},
  "reopens": {"count": 2, "items": ["KIRA-118", "KIRA-131"]}
}
```

`degraded_n` counts items whose cycle time is flagged best-effort due to squashed history (§3). `estimate.actual_ratio_p50` is present only when `estimate.unit: hours` (see §2) — the example above uses `points`, so the key is omitted; under `hours` it would appear alongside `total`/`unit`.

## 5. Non-goals

- No kira-owned charting or dashboard beyond the TUI's sparklines and table — `--json` + pandas is the intended analysis path.
- No stored rollups; every number is recomputed from the index + event cache on each invocation.
- No burndown chart or velocity-planning UI in v1.
