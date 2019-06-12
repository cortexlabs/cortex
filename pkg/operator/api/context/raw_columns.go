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
	"github.com/cortexlabs/cortex/pkg/operator/api/userconfig"
)

type RawColumns map[string]RawColumn

type RawColumn interface {
	Column
	GetCompute() *userconfig.SparkCompute
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

type RawInferredColumn struct {
	*userconfig.RawInferredColumn
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

func GetRawColumnUserConfig(rawColumn RawColumn) userconfig.Resource {
	switch rawColumn.GetColumnType() {
	case userconfig.IntegerColumnType:
		return rawColumn.(*RawIntColumn).RawIntColumn
	case userconfig.FloatColumnType:
		return rawColumn.(*RawFloatColumn).RawFloatColumn
	case userconfig.StringColumnType:
		return rawColumn.(*RawStringColumn).RawStringColumn
	case userconfig.InferredColumnType:
		return rawColumn.(*RawInferredColumn).RawInferredColumn
	}

	return nil
}
