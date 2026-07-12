package errx

import (
	"fmt"
	"strings"
)

const (
	ExitUser     = 1
	ExitConflict = 2
	ExitEnv      = 3
)

type Error struct {
	Code int
	Err  error
}

func (e *Error) Error() string { return e.Err.Error() }
func (e *Error) Unwrap() error { return e.Err }

func User(format string, args ...any) *Error {
	return &Error{Code: ExitUser, Err: fmt.Errorf(format, args...)}
}

func Conflict(format string, args ...any) *Error {
	return &Error{Code: ExitConflict, Err: fmt.Errorf(format, args...)}
}

func Env(format string, args ...any) *Error {
	return &Error{Code: ExitEnv, Err: fmt.Errorf(format, args...)}
}

func Invalid(errs []error) *Error {
	msgs := make([]string, len(errs))
	for i, e := range errs {
		msgs[i] = e.Error()
	}
	return User("invalid item: %s", strings.Join(msgs, "; "))
}
