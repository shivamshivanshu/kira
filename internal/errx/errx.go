package errx

import (
	"errors"
	"fmt"
	"strings"
)

type ExitCode int

const (
	ExitUser     ExitCode = 1
	ExitConflict ExitCode = 2
	ExitEnv      ExitCode = 3
	ExitCrash    ExitCode = 4
)

type Error struct {
	Code ExitCode
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

func Invalid(prefix string, errs []error) *Error {
	hint := ""
	for _, e := range errs {
		var ce *Error
		if errors.As(e, &ce) && ce.Hint != "" {
			hint = ce.Hint
			break
		}
	}
	out := User("%s", JoinErrors(prefix, errs))
	out.Hint = hint
	return out
}

func JoinErrors(prefix string, errs []error) string {
	msgs := make([]string, len(errs))
	for i, e := range errs {
		msgs[i] = e.Error()
	}
	return fmt.Sprintf("%s: %s", prefix, strings.Join(msgs, "; "))
}
