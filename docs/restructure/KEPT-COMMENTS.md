# KEPT-COMMENTS — comments retained by the WP-8 zero-comment sweep, for user ruling

Disposition ∈ {delete (rationale), move-to-design-doc §X, keep-in-code}.

## WP-8a (foundation: datamodel, codec, config, id, query, storage, wrappers)

| file:symbol | verbatim comment | proposed disposition |
|---|---|---|
| internal/codec/serialize.go:117 `emitTimestamp` | "verbatim, never EmitScalar: yaml.v3 tags an unquoted RFC3339 scalar as implicit !!timestamp, so re-marshaling would double-quote it and break the unquoted canonical form." | keep-in-code — mechanism trap no name can carry; goldens' byte-stability depends on it |
| internal/codec/serialize.go:122 `emitDate` | "valid dates go verbatim for the same !!timestamp reason as emitTimestamp; invalid ones (due is parsed shape-only) still need EmitScalar's quoting to stay parseable YAML." | keep-in-code — same !!timestamp trap plus the invalid-date quoting split |
| internal/config/defaults.go:51 `Filters` | "deliberately empty, unlike the 02-data-model §9 example: its filter and sprint entries are illustrations, not defaults" | keep-in-code — deliberate-omission trap vs the doc example a maintainer would copy from |
| internal/config/sprintwrite.go:12 `AppendSprint` | "splices lines rather than re-encoding the document: a whole-file yaml re-encode would normalize comment alignment and flow styles across the hand-edited config.yaml" | keep-in-code — formatting-preservation invariant invisible in the splice code itself |

## WP-8b (app layer: core, cli, tests, wrappers)

| file:symbol | verbatim comment | proposed disposition |
|---|---|---|
| internal/core/comment.go:19 `Store.Comment` | "Comment bypasses the mutate pipeline on purpose: a pure byte-suffix append that never bumps `updated` or reserializes frontmatter, so concurrent comments on two branches stay disjoint appended regions and merge cleanly." | keep-in-code — the missing `updated` bump reads as a bug without it; the merge-cleanliness invariant spans codec.AppendComment + storage.WriteItemRaw and no local name can carry it (docs 02 §4 states it, but the trap is at this call site) |
| internal/core/find.go:40 `NewFindResult` | "Both backends append a file's matches in ascending line order and sortByKey is stable, so hits within one item stay line-ordered with no explicit tiebreak — a backend that appended out of order would need one." | keep-in-code — cross-backend ordering contract resting on sort stability; not doc-cited, would break silently if either backend or the sort changed |
| internal/core/move.go:120 `Store.setActive` | ".cache/ needs no MkdirAll here: every Move holds the store lock, whose acquisition creates the directory." | keep-in-code — lock-side-effect precondition (locking-class invariant); alternatively delete and add MkdirAll, but that hides the coupling |
| internal/cli/discover.go:62 `pickCandidate` | "An empty ref with a nil error means the user cancelled: the caller exits 0." | delete if signature is changed to return an explicit `cancelled bool`; kept for now as the ("", nil) sentinel is un-nameable |
| internal/cli/find.go:13 `knownGlobalFlags` | "The global `-C <path>` chdir is deliberately absent: inside `find`, `-C` means ripgrep's context flag and must pass through." | keep-in-code — deliberate-omission trap: adding "-C" here looks like an obvious fix and breaks rg context passthrough (docs 04 find states it, but not at the list a maintainer would edit) |
| internal/cli/version.go:9 `version` | "version is the build version, overridable at link time: go build -ldflags \"-X github.com/shivamshivanshu/kira/internal/cli.version=v1.2.3\"" | keep-in-code — link-time -X contract is invisible in source; alternatively move-to-README/release doc once one exists |
| tests/contract/contract_test.go:59 `baseEnv` | "rg/fzf are deliberately absent from toolBin, so find always takes its deterministic pure-Go fallback regardless of what the host has installed." | keep-in-code — deliberate-omission that golden byte-stability depends on |
| tests/e2e/find_discover_test.go:33 `setupCleanBin` | "testscript runs `kira` by re-execing the test binary via a PATH symlink; a script that sets PATH=$CLEANBIN would lose it, so mirror that symlink here. rg/fzf are deliberately omitted to simulate their absence." | keep-in-code — testscript re-exec plumbing; removing the symlink fails scripts with a misleading "kira: not found" |

## WP-3.5.2 (self-documenting seed config + comment-preserving `config set`)

| file:symbol | verbatim comment | proposed disposition |
|---|---|---|
| internal/config/set.go:42 `SetScalar` | "SetScalar edits one scalar by splicing its single line rather than re-encoding the document, so every comment and untouched line stays byte-identical (a whole-file yaml re-encode reflows comment alignment and flow styles)." | keep-in-code — same byte-preserving-splice invariant as `AppendSprint` (row above), invisible in the splice code; it is the reason `config set` exists as line surgery rather than a Node round-trip |

## Ratified conventions

- One-line package doc-comments are exempt from the zero-comment rule (covers internal/cli/cli.go, internal/core/doc.go, internal/config/load.go, internal/id/id.go, internal/query/eval.go).
- Test layout: each module's `test/` subdir holds black-box `package X_test` tests; the 10 white-box files in core/id/query stay inline beside the code they exercise (moving them would force export shims); cross-package coverage of `internal/X` from `X/test` needs `-coverpkg`.
