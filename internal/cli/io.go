package cli

import (
	"fmt"
	"os"
)

// Exit codes are an external contract.
const (
	ExitOK       = 0
	ExitRuntime  = 1
	ExitUsage    = 2
	ExitGuard    = 3
	ExitNotFound = 4
	ExitConflict = 5
)

type ExitError struct {
	Code int
	Err  error
}

func (e ExitError) Error() string { return e.Err.Error() }
func (e ExitError) Unwrap() error { return e.Err }

func Exitf(code int, format string, a ...any) *ExitError {
	return &ExitError{Code: code, Err: fmt.Errorf(format, a...)}
}

// Command Results
func resultf(format string, a ...any) { _, _ = fmt.Fprintf(os.Stdout, format, a...) }

// Human facing
func infof(format string, a ...any) {
	_, _ = fmt.Fprintf(os.Stderr, "▸ "+format+"\n", a...)
}

func warnf(format string, a ...any) {
	_, _ = fmt.Fprintf(os.Stderr, "⚠ "+format+"\n", a...)
}
