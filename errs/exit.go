package errs

import "fmt"

// ExitError preserves a child command's public exit and error codes without
// printing a second error message after the child has already written stderr.
type ExitError struct {
	Code      int
	ErrorCode string
}

func (e *ExitError) Error() string {
	return fmt.Sprintf("command exited with code %d", e.Code)
}
