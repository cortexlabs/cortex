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
	"github.com/cortexlabs/cortex/pkg/api/userconfig"
	"github.com/cortexlabs/cortex/pkg/lib/cast"
	"github.com/cortexlabs/cortex/pkg/lib/hash"
)

type RawColumns map[string]RawColumn

type RawColumn interface {
	Column
	GetCompute() *userconfig.SparkCompute
	GetUserConfig() userconfig.Resource
}

type RawIntColumn struct {
	*userconfig.RawIntColumn
	*ComputedResourceFields
}

type RawFloatColumn struct {
	*userconfig.RawFloatColumn
	*ComputedResourceFields
}

type RawStringColumn struct {
	*userconfig.RawStringColumn
	*ComputedResourceFields
}

func (rawColumns RawColumns) OneByID(id string) RawColumn {
	for _, rawColumn := range rawColumns {
		if rawColumn.GetID() == id {
			return rawColumn
		}
	}
	return nil
}

func (rawColumns RawColumns) columnInputsID(columnInputValues map[string]interface{}, includeTags bool) string {
	columnIDMap := make(map[string]string)
	for columnInputName, columnInputValue := range columnInputValues {
		if columnName, ok := columnInputValue.(string); ok {
			if includeTags {
				columnIDMap[columnInputName] = rawColumns[columnName].GetIDWithTags()
			} else {
				columnIDMap[columnInputName] = rawColumns[columnName].GetID()
			}
		}
		if columnNames, ok := cast.InterfaceToStrSlice(columnInputValue); ok {
			var columnIDs string
			for _, columnName := range columnNames {
				if includeTags {
					columnIDs = columnIDs + rawColumns[columnName].GetIDWithTags()
				} else {
					columnIDs = columnIDs + rawColumns[columnName].GetID()
				}
			}
			columnIDMap[columnInputName] = columnIDs
		}
	}
	return hash.Any(columnIDMap)
}

func (rawColumns RawColumns) ColumnInputsID(columnInputValues map[string]interface{}) string {
	return rawColumns.columnInputsID(columnInputValues, false)
}

func (rawColumns RawColumns) ColumnInputsIDWithTags(columnInputValues map[string]interface{}) string {
	return rawColumns.columnInputsID(columnInputValues, true)
}

func (rawColumn *RawIntColumn) GetInputRawColumnNames() []string {
	return []string{rawColumn.GetName()}
}

func (rawColumn *RawFloatColumn) GetInputRawColumnNames() []string {
	return []string{rawColumn.GetName()}
}

func (rawColumn *RawStringColumn) GetInputRawColumnNames() []string {
	return []string{rawColumn.GetName()}
}
