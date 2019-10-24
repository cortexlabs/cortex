/*
Copyright 2019 Cortex Labs, Inc.

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
	"os"
	"strings"

	pkgerrors "github.com/pkg/errors"

	s "github.com/cortexlabs/cortex/pkg/lib/strings"
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
	if !HasErrors(errs) {
		return nil
	}
	wrappedErrs := make([]error, len(errs))
	for i, err := range errs {
		wrappedErrs[i] = Wrap(err, strs...)
	}
	return wrappedErrs
}

func HasErrors(errs []error) bool {
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
	var err error
	switch casted := items[0].(type) {
	case error:
		err = casted
	case string:
		err = New(casted)
	default:
		err = New(s.UserStrStripped(casted))
	}

	for i, item := range items {
		if i == 0 {
			continue
		}

		switch casted := item.(type) {
		case error:
			err = Wrap(err, casted.Error())
		case string:
			err = Wrap(err, casted)
		default:
			err = Wrap(err, s.UserStrStripped(casted))
		}
	}

	return err
}

func Exit(items ...interface{}) {
	if len(items) == 0 {
		items = append(items, "empty exit")
	}
	err := MergeErrItems(items...)
	PrintError(err)
	fmt.Println()
	os.Exit(1)
}

func Panic(items ...interface{}) {
	if len(items) == 0 {
		items = append(items, "empty panic")
	}
	err := MergeErrItems(items...)
	// PrintStacktrace(err)
	panic(err)
}

func PrintError(err error, strs ...string) {
	wrappedErr := Wrap(err, strs...)
	fmt.Println("error:", wrappedErr.Error())
	// PrintStacktrace(wrappedErr)
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

func RecoverAndExit(strs ...string) {
	if errInterface := recover(); errInterface != nil {
		err := CastRecoverError(errInterface, strs...)
		PrintError(err)
		os.Exit(1)
	}
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
