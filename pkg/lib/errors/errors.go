/*
Copyright 2022 Cortex Labs, Inc.

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

	s "github.com/cortexlabs/cortex/pkg/lib/strings"
)

const (
	ErrUnexpected = "errors.unexpected"
)

func ErrorUnexpected(msgs ...interface{}) error {
	strs := make([]string, len(msgs))
	for i, msg := range msgs {
		strs[i] = s.ObjFlatNoQuotes(msg)
	}

	return WithStack(&Error{
		Kind:    ErrUnexpected,
		Message: strings.Join(strs, ": "),
	})
}

func ListOfErrors(errKind string, shouldPrint bool, errors ...error) error {
	var errorsContents string
	for i, err := range errors {
		if err != nil {
			errorsContents += fmt.Sprintf("error #%d: %s\n", i+1, err.Error())
		}
	}
	if errorsContents == "" {
		return nil
	}
	return WithStack(&Error{
		Kind:    errKind,
		Message: errorsContents,
		NoPrint: !shouldPrint,
	})
}
