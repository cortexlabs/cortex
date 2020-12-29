package parallel

import (
	"fmt"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
)

const (
	ErrUnexpectedError = "parallel.unexpected_error"
)

func ErrorUnexpectedError(msg string) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrUnexpectedError,
		Message: fmt.Sprintf("unexpected error occurred in parallel execution: %s", msg),
	})
}
