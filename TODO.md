- `internal/core/warnings.go` `emitWarnings()` writes straight to os.Stderr for
  assign/move (WIP-limit, vocabulary) instead of routing through the
  `StderrNotes` + `emitStderrNotes` pattern that list/show/tree/diff/changes/index
  already use. Surfaced by DX-9's contract-test stderr assertion, which needed a
  per-case escape hatch (`stderrWarn`) to tolerate the inconsistency.
