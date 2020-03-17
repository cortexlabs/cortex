/*
Copyright 2020 Cortex Labs, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package errors

import (
	"errors"
	"fmt"
	"strings"

	pkgerrors "github.com/pkg/errors"
)

const (
	ErrNotCortexError = "errors.not_cortex_error"
)

// TODO rename to Error?
type Error struct {
	Kind        string
	Message     string
	NoTelemetry bool
	NoPrint     bool
	cause       error
	stack       *stack
}

func (cortexError *Error) Error() string {
	return cortexError.Message
}

func (cortexError *Error) Cause() error {
	if cortexError.cause != nil {
		return cortexError.cause
	}
	return cortexError
}

func (cortexError *Error) StackTrace() pkgerrors.StackTrace {
	stackTrace := make([]pkgerrors.Frame, len(*cortexError.stack))
	for i := 0; i < len(stackTrace); i++ {
		stackTrace[i] = pkgerrors.Frame((*cortexError.stack)[i])
	}
	return stackTrace
}

func WithStack(err error) error {
	if err == nil {
		return nil
	}

	cortexError := getCortexError(err)

	if cortexError == nil {
		cortexError = &Error{
			Kind:    ErrNotCortexError, // TODO
			Message: err.Error(),
			cause:   err,
		}
	}

	if cortexError.stack == nil {
		cortexError.stack = callers()
	}

	return cortexError
}

func Wrap(err error, strs ...string) error {
	if err == nil {
		return nil
	}

	cortexError := WithStack(err).(*Error)

	strs = removeEmptyStrs(strs)
	strs = append(strs, cortexError.Message)
	cortexError.Message = strings.Join(strs, ": ")

	return cortexError
}

// TODO private? delete?
func new(strs ...string) error {
	strs = removeEmptyStrs(strs)
	errStr := strings.Join(strs, ": ")
	return WithStack(errors.New(errStr))
}

func getCortexError(err error) *Error {
	if cortexError, ok := err.(*Error); ok {
		return cortexError
	}
	return nil
}

// TODO: add a NotCortexError kind?
func GetKind(err error) string {
	if cortexError, ok := err.(*Error); ok {
		return cortexError.Kind
	}
	return ErrNotCortexError
}

func IsNoTelemetry(err error) bool {
	if cortexError, ok := err.(*Error); ok {
		return cortexError.NoTelemetry
	}
	return false
}

func SetNoTelemetry(err error) error {
	cortexError := WithStack(err).(*Error)
	cortexError.NoTelemetry = true
	return cortexError
}

func IsNoPrint(err error) bool {
	if cortexError, ok := err.(*Error); ok {
		return cortexError.NoPrint
	}
	return false
}

func SetNoPrint(err error) error {
	cortexError := WithStack(err).(*Error)
	cortexError.NoPrint = true
	return cortexError
}

func Cause(err error) error {
	if cortexError, ok := err.(*Error); ok {
		return cortexError.Cause()
	}
	return nil
}

func PrintStacktrace(err error) {
	fmt.Printf("%+v\n", err)
}

func CastRecoverError(errInterface interface{}, strs ...string) error {
	var err error
	var ok bool
	err, ok = errInterface.(error)
	if !ok {
		err = new(fmt.Sprint(errInterface))
	}
	return Wrap(err, strs...)
}

func removeEmptyStrs(strs []string) []string {
	var cleanStrs []string
	for _, str := range strs {
		if str != "" {
			cleanStrs = append(cleanStrs, str)
		}
	}
	return cleanStrs
}
