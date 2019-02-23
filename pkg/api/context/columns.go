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

package context

import (
	"github.com/cortexlabs/cortex/pkg/api/resource"
	s "github.com/cortexlabs/cortex/pkg/api/strings"
	"github.com/cortexlabs/cortex/pkg/api/userconfig"
	"github.com/cortexlabs/cortex/pkg/utils/cast"
	"github.com/cortexlabs/cortex/pkg/utils/errors"
	"github.com/cortexlabs/cortex/pkg/utils/util"
)

type Columns map[string]Column

type Column interface {
	ComputedResource
	GetType() string
	IsRaw() bool
	GetInputRawColumnNames() []string
}

func (ctx *Context) Columns() Columns {
	columns := Columns{}
	for name, column := range ctx.RawColumns {
		columns[name] = column
	}
	for name, column := range ctx.TransformedColumns {
		columns[name] = column
	}
	return columns
}

func (ctx *Context) ColumnNames() map[string]bool {
	names := make(map[string]bool, len(ctx.RawColumns)+len(ctx.TransformedColumns))
	for name := range ctx.RawColumns {
		names[name] = true
	}
	for name := range ctx.TransformedColumns {
		names[name] = true
	}
	return names
}

func (ctx *Context) GetColumn(name string) Column {
	if rawColumn, ok := ctx.RawColumns[name]; ok {
		return rawColumn
	} else if transformedColumn, ok := ctx.TransformedColumns[name]; ok {
		return transformedColumn
	}
	return nil
}

func (columns Columns) ID(columnNames []string) string {
	columnIDMap := make(map[string]string)
	for _, columnName := range columnNames {
		columnIDMap[columnName] = columns[columnName].GetID()
	}
	return util.HashObj(columnIDMap)
}

func (columns Columns) IDWithTags(columnNames []string) string {
	columnIDMap := make(map[string]string)
	for _, columnName := range columnNames {
		columnIDMap[columnName] = columns[columnName].GetIDWithTags()
	}
	return util.HashObj(columnIDMap)
}

func GetColumnRuntimeTypes(
	columnInputValues map[string]interface{},
	rawColumns RawColumns,
) (map[string]interface{}, error) {

	err := userconfig.ValidateColumnInputValues(columnInputValues)
	if err != nil {
		return nil, err
	}

	columnRuntimeTypes := make(map[string]interface{}, len(columnInputValues))

	for inputName, columnInputValue := range columnInputValues {
		if rawColumnName, ok := columnInputValue.(string); ok {
			rawColumn, ok := rawColumns[rawColumnName]
			if !ok {
				return nil, errors.Wrap(userconfig.ErrorUndefinedResource(rawColumnName, resource.RawColumnType), inputName)
			}
			columnRuntimeTypes[inputName] = rawColumn.GetType()
			continue
		}

		if rawColumnNames, ok := cast.InterfaceToStrSlice(columnInputValue); ok {
			rawColumnTypes := make([]string, len(rawColumnNames))
			for i, rawColumnName := range rawColumnNames {
				rawColumn, ok := rawColumns[rawColumnName]
				if !ok {
					return nil, errors.Wrap(userconfig.ErrorUndefinedResource(rawColumnName, resource.RawColumnType), inputName, s.Index(i))
				}
				rawColumnTypes[i] = rawColumn.GetType()
			}
			columnRuntimeTypes[inputName] = rawColumnTypes
			continue
		}

		return nil, errors.New(inputName, s.ErrInvalidPrimitiveType(columnInputValue, s.PrimTypeString, s.PrimTypeStringList)) // unexpected
	}

	return columnRuntimeTypes, nil
}
