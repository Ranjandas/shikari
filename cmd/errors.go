package cmd

import (
	"errors"
)

// exitCodeError makes the process exit with code instead of 1, eg: to pass on
// the exit status of a remote command.
type exitCodeError struct {
	code int
	err  error
}

func (e *exitCodeError) Error() string {
	return e.err.Error()
}

func (e *exitCodeError) Unwrap() error {
	return e.err
}

// exitCode returns the exit code for the error returned by a command.
func exitCode(err error) int {
	var codeErr *exitCodeError
	if errors.As(err, &codeErr) && codeErr.code > 0 {
		return codeErr.code
	}

	return 1
}
