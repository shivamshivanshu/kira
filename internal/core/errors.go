package core

import "fmt"

// Exit codes per docs/design/04-cli.md §1. Success (0) is the absence of an error.
const (
	// ExitUser is a user or validation error: a bad flag, an invalid state
	// transition, malformed frontmatter.
	ExitUser = 1
	// ExitConflict is a conflict or consistency error: an ID collision or an
	// unresolved merge conflict.
	ExitConflict = 2
	// ExitEnv is an environment error: not a git repo, .kira/ missing, $EDITOR
	// unset when needed, git binary not found.
	ExitEnv = 3
)

// Error carries the process exit code alongside the underlying error, so cli
// can map a failure to the exit-code policy without a table of sentinel checks.
type Error struct {
	Code int
	Err  error
}

func (e *Error) Error() string { return e.Err.Error() }
func (e *Error) Unwrap() error { return e.Err }

func userErr(format string, args ...any) *Error {
	return &Error{Code: ExitUser, Err: fmt.Errorf(format, args...)}
}

func conflictErr(format string, args ...any) *Error {
	return &Error{Code: ExitConflict, Err: fmt.Errorf(format, args...)}
}

func envErr(format string, args ...any) *Error {
	return &Error{Code: ExitEnv, Err: fmt.Errorf(format, args...)}
}

// NewUserError and NewEnvError construct exit-coded errors for the cli layer,
// which needs to raise a policy exit code (1 or 3) without reaching the
// unexported constructors (docs/design/04-cli.md §1).
func NewUserError(format string, args ...any) error { return userErr(format, args...) }
func NewEnvError(format string, args ...any) error  { return envErr(format, args...) }
