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
	"github.com/cortexlabs/cortex/pkg/utils/cast"
)

type TransformedColumns map[string]*TransformedColumn

type TransformedColumn struct {
	*userconfig.TransformedColumn
	*ComputedResourceFields
	Type string `json:"type"`
}

func (column *TransformedColumn) GetType() string {
	return column.Type
}

// Returns map[string]string because after autogen, arg values are constant or aggregate names
func (transformedColumn *TransformedColumn) Args() map[string]string {
	args, _ := cast.InterfaceToStrStrMap(transformedColumn.Inputs.Args)
	return args
}

func (transformedColumn *TransformedColumn) InputAggregateNames(ctx *Context) map[string]bool {
	inputAggregateNames := make(map[string]bool)
	for _, valueResourceName := range transformedColumn.Args() {
		if _, ok := ctx.Aggregates[valueResourceName]; ok {
			inputAggregateNames[valueResourceName] = true
		}
	}
	return inputAggregateNames
}

func (transformedColumns TransformedColumns) OneByID(id string) *TransformedColumn {
	for _, transformedColumn := range transformedColumns {
		if transformedColumn.ID == id {
			return transformedColumn
		}
	}
	return nil
}
