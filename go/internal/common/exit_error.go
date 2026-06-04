// Copyright (C) SonarSource Sàrl
// For more information, see https://sonarsource.com/legal/
// mailto:info AT sonarsource DOT com

package common

import "fmt"

// ExitCodeError is an error that carries a specific process exit code.
// main.go unwraps it via errors.As to translate the carried code into
// the process exit status, so a command can fail with a meaningful
// non-zero code (e.g. "missing org mapping" = 2) rather than the
// catch-all 1.
type ExitCodeError struct {
	Code int
	Err  error
}

func (e *ExitCodeError) Error() string {
	if e.Err == nil {
		return fmt.Sprintf("exit code %d", e.Code)
	}
	return e.Err.Error()
}

func (e *ExitCodeError) Unwrap() error {
	return e.Err
}

// NewExitError wraps err with the given exit code. Returns the bare
// error when err is nil to avoid producing an empty-message wrapper.
func NewExitError(code int, err error) error {
	if err == nil {
		return nil
	}
	return &ExitCodeError{Code: code, Err: err}
}
