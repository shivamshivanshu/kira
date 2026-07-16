- `internal/core/warnings.go` `emitWarnings()` writes straight to os.Stderr for
  assign/move (WIP-limit, vocabulary) instead of routing through the
  `StderrNotes` + `emitStderrNotes` pattern that list/show/tree/diff/changes/index
  already use. Surfaced by DX-9's contract-test stderr assertion, which needed a
  per-case escape hatch (`stderrWarn`) to tolerate the inconsistency.
- `internal/doctor/check.go`'s `resolutionFindings` only flags a non-nil
  resolution on a non-Done state; it never flags the mirror case (a Done-category
  state with a nil resolution). CORE-35 deliberately doesn't auto-fill a
  resolution when merge/resolve lands in Done with none surviving the merge
  (would fabricate data), so that shape can exist on disk with nothing to catch
  it.
