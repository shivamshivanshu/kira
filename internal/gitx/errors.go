package gitx

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
)

type CmdError struct{ msg string }

func (e *CmdError) Error() string { return e.msg }

func IsCmdError(err error) bool {
	var ce *CmdError
	return errors.As(err, &ce)
}

func cmdError(prefix string, stderr *bytes.Buffer, err error) *CmdError {
	msg := strings.TrimSpace(stderr.String())
	if msg == "" {
		msg = err.Error()
	}
	return &CmdError{msg: fmt.Sprintf("%s: %s", prefix, msg)}
}
