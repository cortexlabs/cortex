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
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	pkgerrors "github.com/pkg/errors"
)

func New(strs ...string) error {
	strs = removeEmptyStrs(strs)
	errStr := strings.Join(strs, ": ")
	return pkgerrors.New(errStr)
}

func Wrap(err error, strs ...string) error {
	if err == nil {
		return nil
	}
	strs = removeEmptyStrs(strs)
	if len(strs) == 0 {
		return pkgerrors.WithStack(err)
	}
	errStr := strings.Join(strs, ": ")
	return pkgerrors.Wrap(err, errStr)
}

func WithStack(err error) error {
	return pkgerrors.WithStack(err)
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
				err = New(casted)
			} else {
				err = Wrap(err, casted)
			}
		default:
			if err == nil {
				err = New(s.UserStrStripped(casted))
			} else {
				err = Wrap(err, s.UserStrStripped(casted))
			}
		}
	}

	return err
}

func PrintError(err error, strs ...string) {
	wrappedErr := Wrap(err, strs...)
	fmt.Print("error: ", s.EnsureSingleTrailingNewLine(Message(wrappedErr)))
	// PrintStacktrace(wrappedErr)
}

func Message(err error, strs ...string) string {
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
		err = New(fmt.Sprint(errInterface))
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
