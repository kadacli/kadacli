package cmd

import "fmt"

// Process exit codes. These are part of the CLI's contract with CI systems —
// a pipeline gates on them — so they are defined in one place and never
// derived from the text of an error message.
const (
	exitCompliant = 0 // no errors, nothing to change
	exitDrift     = 1 // ran cleanly, but live state differs from desired state
	exitFailure   = 2 // something went wrong: bad input, auth, API, usage
)

// exitError is an error that carries the exit code the process should end
// with. Commands return one of these; Execute reads the code off it.
type exitError struct {
	code int
	err  error
}

func (e *exitError) Error() string { return e.err.Error() }
func (e *exitError) Unwrap() error { return e.err }

// failure wraps an existing error as an exit-2 failure, preserving its message
// and its chain.
func failure(err error) error {
	return &exitError{code: exitFailure, err: err}
}

// failuref reports an exit-2 failure: the command could not complete.
func failuref(format string, a ...any) error {
	return &exitError{code: exitFailure, err: fmt.Errorf(format, a...)}
}

// driftf reports an exit-1 result: the command completed successfully and
// found the live environment out of sync with the declared policies.
func driftf(format string, a ...any) error {
	return &exitError{code: exitDrift, err: fmt.Errorf(format, a...)}
}
