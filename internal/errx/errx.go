package errx

import (
	"errors"
	"fmt"
	"strings"
)

const (
	ExitUser     = 1
	ExitConflict = 2
	ExitEnv      = 3
	ExitCrash    = 4
)

type Error struct {
	Code int
	Err  error
	Hint string
}

func (e *Error) Error() string { return e.Err.Error() }
func (e *Error) Unwrap() error { return e.Err }

func (e *Error) WithHint(format string, args ...any) *Error {
	c := *e
	c.Hint = fmt.Sprintf(format, args...)
	return &c
}

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
	hint := ""
	for i, e := range errs {
		msgs[i] = e.Error()
		if hint == "" {
			var ce *Error
			if errors.As(e, &ce) {
				hint = ce.Hint
			}
		}
	}
	out := User("invalid item: %s", strings.Join(msgs, "; "))
	out.Hint = hint
	return out
}
