package gitx

import "errors"

type CmdError struct{ msg string }

func (e *CmdError) Error() string { return e.msg }

func IsCmdError(err error) bool {
	var ce *CmdError
	return errors.As(err, &ce)
}
