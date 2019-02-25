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

package userconfig

import (
	"github.com/cortexlabs/cortex/pkg/api/resource"
	s "github.com/cortexlabs/cortex/pkg/api/strings"
	"github.com/cortexlabs/cortex/pkg/lib/cast"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/slices"
)

type Column interface {
	Resource
	IsRaw() bool
}

func (config *Config) ValidateColumns() error {
	columnResources := make([]Resource, len(config.RawColumns)+len(config.TransformedColumns))
	for i, res := range config.RawColumns {
		columnResources[i] = res
	}

	for i, res := range config.TransformedColumns {
		columnResources[i+len(config.RawColumns)] = res
	}

	dups := FindDuplicateResourceName(columnResources...)
	if len(dups) > 0 {
		return ErrorDuplicateResourceName(dups...)
	}

	for _, aggregate := range config.Aggregates {
		err := ValidateColumnInputsExistAndRaw(aggregate.Inputs.Columns, config)
		if err != nil {
			return errors.Wrap(err, Identify(aggregate), InputsKey, ColumnsKey)
		}
	}

	for _, transformedColumn := range config.TransformedColumns {
		err := ValidateColumnInputsExistAndRaw(transformedColumn.Inputs.Columns, config)
		if err != nil {
			return errors.Wrap(err, Identify(transformedColumn), InputsKey, ColumnsKey)
		}
	}

	return nil
}

func ValidateColumnInputsExistAndRaw(columnInputValues map[string]interface{}, config *Config) error {
	for columnInputName, columnInputValue := range columnInputValues {
		if columnName, ok := columnInputValue.(string); ok {
			err := ValidateColumnNameExistsAndRaw(columnName, config)
			if err != nil {
				return errors.Wrap(err, columnInputName)
			}
			continue
		}
		if columnNames, ok := cast.InterfaceToStrSlice(columnInputValue); ok {
			for i, columnName := range columnNames {
				err := ValidateColumnNameExistsAndRaw(columnName, config)
				if err != nil {
					return errors.Wrap(err, columnInputName, s.Index(i))
				}
			}
			continue
		}
		return errors.New(columnInputName, s.ErrInvalidPrimitiveType(columnInputValue, s.PrimTypeString, s.PrimTypeStringList)) // unexpected
	}
	return nil
}

func ValidateColumnNameExistsAndRaw(columnName string, config *Config) error {
	if config.IsTransformedColumn(columnName) {
		return ErrorColumnMustBeRaw(columnName)
	}
	if !config.IsRawColumn(columnName) {
		return ErrorUndefinedResource(columnName, resource.RawColumnType)
	}
	return nil
}

func (config *Config) ColumnNames() []string {
	return append(config.RawColumns.Names(), config.TransformedColumns.Names()...)
}

func (config *Config) IsRawColumn(name string) bool {
	return slices.HasString(name, config.RawColumns.Names())
}

func (config *Config) IsTransformedColumn(name string) bool {
	return slices.HasString(name, config.TransformedColumns.Names())
}
