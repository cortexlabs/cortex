/*
Copyright 2021 Cortex Labs, Inc.

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

const (
	ErrAtLeastOneColumn                  = "table.at_least_one_column"
	ErrHeaderWiderThanMaxWidth           = "table.header_wider_than_max_width"
	ErrHeaderMinWidthGreaterThanMaxWidth = "table.header_min_width_greater_than_max_width"
	ErrWrongNumberOfColumns              = "table.wrong_number_of_columns"
)

func ErrorAtLeastOneColumn() error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrAtLeastOneColumn,
		Message: "must have at least one column",
	})
}

func ErrorHeaderWiderThanMaxWidth(headerTitle string, maxWidth int) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrHeaderWiderThanMaxWidth,
		Message: fmt.Sprintf("header %s is wider than max width (%d)", headerTitle, maxWidth),
	})
}

func ErrorHeaderMinWidthGreaterThanMaxWidth(headerTitle string, minWidth int, maxWidth int) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrHeaderMinWidthGreaterThanMaxWidth,
		Message: fmt.Sprintf("header %s has min width > max width (%d > %d)", headerTitle, minWidth, maxWidth),
	})
}

func ErrorWrongNumberOfColumns(rowNumber int, actualCols int, expectedCols int) error {
	return errors.WithStack(&errors.Error{
		Kind:    ErrWrongNumberOfColumns,
		Message: fmt.Sprintf("row %d does not have the expected number of columns (got %d, expected %d)", rowNumber, actualCols, expectedCols),
	})
}
