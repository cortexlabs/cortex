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
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	"github.com/cortexlabs/cortex/pkg/operator/api/userconfig"
)

type Columns map[string]Column

type Column interface {
	ComputedResource
	GetColumnType() userconfig.ColumnType
	IsRaw() bool
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

func (ctx *Context) ColumnNames() strset.Set {
	names := strset.NewWithSize(len(ctx.RawColumns) + len(ctx.TransformedColumns))
	for name := range ctx.RawColumns {
		names.Add(name)
	}
	for name := range ctx.TransformedColumns {
		names.Add(name)
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
