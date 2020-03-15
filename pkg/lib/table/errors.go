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

package table

import (
	"fmt"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
)

type ErrorKind int

const (
	ErrUnknown ErrorKind = iota
	ErrAtLeastOneColumn
	ErrHeaderWiderThanMaxWidth
	ErrHeaderMinWidthGreaterThanMaxWidth
	ErrWrongNumberOfColumns
)

var _errorKinds = []string{
	"table.unknown",
	"table.at_least_one_column",
	"table.header_wider_than_max_width",
	"table.header_min_width_greater_than_max_width",
	"table.wrong_number_of_columns",
}

var _ = [1]int{}[int(ErrWrongNumberOfColumns)-(len(_errorKinds)-1)] // Ensure list length matches

func (t ErrorKind) String() string {
	return _errorKinds[t]
}

// MarshalText satisfies TextMarshaler
func (t ErrorKind) MarshalText() ([]byte, error) {
	return []byte(t.String()), nil
}

// UnmarshalText satisfies TextUnmarshaler
func (t *ErrorKind) UnmarshalText(text []byte) error {
	enum := string(text)
	for i := 0; i < len(_errorKinds); i++ {
		if enum == _errorKinds[i] {
			*t = ErrorKind(i)
			return nil
		}
	}

	*t = ErrUnknown
	return nil
}

// UnmarshalBinary satisfies BinaryUnmarshaler
// Needed for msgpack
func (t *ErrorKind) UnmarshalBinary(data []byte) error {
	return t.UnmarshalText(data)
}

// MarshalBinary satisfies BinaryMarshaler
func (t ErrorKind) MarshalBinary() ([]byte, error) {
	return []byte(t.String()), nil
}

func ErrorAtLeastOneColumn() error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrAtLeastOneColumn,
		Message: "must have at least one column",
	})
}

func ErrorHeaderWiderThanMaxWidth(headerTitle string, maxWidth int) error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrHeaderWiderThanMaxWidth,
		Message: fmt.Sprintf("header %s is wider than max width (%d)", headerTitle, maxWidth),
	})
}

func ErrorHeaderMinWidthGreaterThanMaxWidth(headerTitle string, minWidth int, maxWidth int) error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrHeaderMinWidthGreaterThanMaxWidth,
		Message: fmt.Sprintf("header %s has min width > max width (%d > %d)", headerTitle, minWidth, maxWidth),
	})
}

func ErrorWrongNumberOfColumns(rowNumber int, actualCols int, expectedCols int) error {
	return errors.WithStack(&errors.CortexError{
		Kind:    ErrWrongNumberOfColumns,
		Message: fmt.Sprintf("row %d does not have the expected number of columns (got %d, expected %d)", rowNumber, actualCols, expectedCols),
	})
}
