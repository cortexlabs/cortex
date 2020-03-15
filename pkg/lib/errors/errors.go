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
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/cortexlabs/cortex/pkg/lib/cast"
	"github.com/cortexlabs/cortex/pkg/lib/print"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	pkgerrors "github.com/pkg/errors"
)

type Error interface {
	error
	GetKind() ErrorKind
	IsNoTelemetry() bool
	SetNoTelemetry()
	IsNoPrint() bool
	SetNoPrint()
}

type ErrorKind interface {
	String() string
}

type CortexError struct {
	Kind        ErrorKind
	Message     string
	NoTelemetry bool
	NoPrint     bool
}

func (e *CortexError) Error() string {
	return e.Message
}

func (e *CortexError) GetKind() ErrorKind {
	return e.Kind
}

func (e *CortexError) IsNoTelemetry() bool {
	return e.NoTelemetry
}

func (e *CortexError) SetNoTelemetry() {
	e.NoTelemetry = true
}

func (e *CortexError) IsNoPrint() bool {
	return e.NoPrint
}

func (e *CortexError) SetNoPrint() {
	e.NoPrint = true
}

type CortexErrorWithCause struct {
	*CortexError
	origErr error
}

func (e *CortexErrorWithCause) Cause() error {
	return e.origErr
}

// TODO private?
func new(strs ...string) error {
	strs = removeEmptyStrs(strs)
	errStr := strings.Join(strs, ": ")
	err := CortexError{Kind: ErrUnknown, Message: errStr}
	return pkgerrors.WithStack(&err)
}

func Wrap(err error, strs ...string) error {
	if err == nil {
		return nil
	}

	strs = removeEmptyStrs(strs)
	if len(strs) == 0 {
		return WithStack(err)
	}

	errStr := strings.Join(strs, ": ")

	if !isCortexError(err) {
		err = CortexErrorWithCause{origErr: err, CortexError: &CortexError{Kind: ErrUnknown, Message: errStr}}
	}

	return pkgerrors.Wrap(err, errStr)
}

func WithStack(err error) error {
	if !isCortexError(err) {
		err = CortexErrorWithCause{origErr: err, CortexError: &CortexError{Kind: ErrUnknown, Message: err.Error()}}
	}

	return pkgerrors.WithStack(err)
}

// TODO: add a NotCortexError kind?
func GetKind(err error) ErrorKind {
	if cortexError := getCortexError(err); cortexError != nil {
		return cortexError.GetKind()
	}

	return ErrUnknown
}

func IsNoTelemetry(err error) bool {
	if cortexError := getCortexError(err); cortexError != nil {
		return cortexError.IsNoTelemetry()
	}
	return false
}

func SetNoTelemetry(err error) error {
	if cortexError := getCortexError(err); cortexError != nil {
		cortexError.SetNoTelemetry()
		return err
	}

	cortexError := CortexErrorWithCause{origErr: err, CortexError: &CortexError{Kind: ErrUnknown, Message: err.Error(), NoTelemetry: true}}
	return WithStack(cortexError)
}

func IsNoPrint(err error) bool {
	if cortexError := getCortexError(err); cortexError != nil {
		return cortexError.IsNoPrint()
	}
	return false
}

func SetNoPrint(err error) error {
	if cortexError := getCortexError(err); cortexError != nil {
		cortexError.SetNoPrint()
		return err
	}

	cortexError := CortexErrorWithCause{origErr: err, CortexError: &CortexError{Kind: ErrUnknown, Message: err.Error(), NoPrint: true}}
	return WithStack(cortexError)
}

func getCortexError(err error) Error {
	type causer interface {
		Cause() error
	}

	for err != nil {
		if cortexError, ok := err.(Error); ok {
			return cortexError
		}
		errCasted, ok := err.(causer)
		if !ok {
			break
		}
		err = errCasted.Cause()
	}
	return nil
}

func isCortexError(err error) bool {
	return getCortexError(err) != nil
}

func Cause(err error) error {
	return pkgerrors.Cause(err)
}

func AddError(errs []error, err error, strs ...string) ([]error, bool) {
	ok := false
	if err != nil {
		errs = append(errs, Wrap(err, strs...))
		ok = true
	}
	return errs, ok
}

func AddErrors(errs []error, newErrs []error, strs ...string) ([]error, bool) {
	ok := false
	for _, err := range newErrs {
		if err != nil {
			errs = append(errs, Wrap(err, strs...))
			ok = true
		}
	}
	return errs, ok
}

func WrapAll(errs []error, strs ...string) []error {
	if !HasError(errs) {
		return nil
	}
	wrappedErrs := make([]error, len(errs))
	for i, err := range errs {
		wrappedErrs[i] = Wrap(err, strs...)
	}
	return wrappedErrs
}

func HasError(errs []error) bool {
	for _, err := range errs {
		if err != nil {
			return true
		}
	}
	return false
}

func FirstError(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

func MergeErrItems(items ...interface{}) error {
	items = cast.FlattenInterfaceSlices(items...)

	if len(items) == 0 {
		return nil
	}

	var err error

	for _, item := range items {
		if item == nil {
			continue
		}

		switch casted := item.(type) {
		case error:
			if err == nil {
				err = casted
			} else {
				err = Wrap(err, Message(casted))
			}
		case string:
			if err == nil {
				err = new(casted)
			} else {
				err = Wrap(err, casted)
			}
		default:
			if err == nil {
				err = new(s.UserStrStripped(casted))
			} else {
				err = Wrap(err, s.UserStrStripped(casted))
			}
		}
	}

	return err
}

func PrintError(err error, strs ...string) {
	fmt.Println(errorStr(err, strs...))
	// PrintStacktrace(wrappedErr)
}

func PrintErrorPretty(err error, strs ...string) {
	print.ForUser(errorStr(err, strs...))
}

func errorStr(err error, strs ...string) string {
	wrappedErr := Wrap(err, strs...)
	return "error: " + strings.TrimRight(Message(wrappedErr), "\n")
}

func Message(err error, strs ...string) string {
	wrappedErr := Wrap(err, strs...)
	errStr := wrappedErr.Error()
	return s.RemoveTrailingNewLines(errStr)
}

func MessageFirstLine(err error, strs ...string) string {
	wrappedErr := Wrap(err, strs...)

	var errStr string
	if _, ok := Cause(wrappedErr).(awserr.Error); ok {
		errStr = strings.Split(wrappedErr.Error(), "\n")[0]
	} else {
		errStr = wrappedErr.Error()
	}

	return s.RemoveTrailingNewLines(errStr)
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

type errorKind int

const (
	ErrUnknown errorKind = iota
)

var _errorKinds = []string{
	"unknown",
}

var _ = [1]int{}[int(ErrUnknown)-(len(_errorKinds)-1)] // Ensure list length matches

func (t errorKind) String() string {
	return _errorKinds[t]
}

// MarshalText satisfies TextMarshaler
func (t errorKind) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

// UnmarshalText satisfies TextUnmarshaler
func (t *errorKind) UnmarshalText(text []byte) error {
	enum := string(text)
	for i := 0; i < len(_errorKinds); i++ {
		if enum == _errorKinds[i] {
			*t = errorKind(i)
			return nil
		}
	}

	*t = ErrUnknown
	return nil
}

// UnmarshalBinary satisfies BinaryUnmarshaler
// Needed for msgpack
func (t *errorKind) UnmarshalBinary(data []byte) error {
	return t.UnmarshalText(data)
}

// MarshalBinary satisfies BinaryMarshaler
func (t errorKind) MarshalBinary() ([]byte, error) {
	return []byte(t.String()), nil
}
