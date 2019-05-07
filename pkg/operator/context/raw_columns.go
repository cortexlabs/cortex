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
	"bytes"
	"path/filepath"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/configreader"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/hash"
	"github.com/cortexlabs/cortex/pkg/lib/slices"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/operator/api/context"
	"github.com/cortexlabs/cortex/pkg/operator/api/resource"
	"github.com/cortexlabs/cortex/pkg/operator/api/userconfig"
)

func getRawColumns(
	config *userconfig.Config,
	env *context.Environment,
) (context.RawColumns, error) {

	rawColumns := context.RawColumns{}

	for _, columnConfig := range config.RawColumns {
		var buf bytes.Buffer
		buf.WriteString(env.ID)
		buf.WriteString(columnConfig.GetName())
		buf.WriteString(columnConfig.GetType().String())

		var rawColumn context.RawColumn
		switch typedColumnConfig := columnConfig.(type) {
		case *userconfig.RawIntColumn:
			buf.WriteString(s.Bool(typedColumnConfig.Required))
			buf.WriteString(s.Obj(typedColumnConfig.Min))
			buf.WriteString(s.Obj(typedColumnConfig.Max))
			buf.WriteString(s.Obj(slices.SortInt64sCopy(typedColumnConfig.Values)))
			id := hash.Bytes(buf.Bytes())
			idWithTags := hash.String(id + typedColumnConfig.Tags.ID())
			rawColumn = &context.RawIntColumn{
				ComputedResourceFields: &context.ComputedResourceFields{
					ResourceFields: &context.ResourceFields{
						ID:           id,
						IDWithTags:   idWithTags,
						ResourceType: resource.RawColumnType,
						MetadataKey:  filepath.Join(consts.RawColumnsDir, id+"_metadata.json"),
					},
				},
				RawIntColumn: typedColumnConfig,
			}
		case *userconfig.RawFloatColumn:
			buf.WriteString(s.Bool(typedColumnConfig.Required))
			buf.WriteString(s.Obj(typedColumnConfig.Min))
			buf.WriteString(s.Obj(typedColumnConfig.Max))
			buf.WriteString(s.Obj(slices.SortFloat32sCopy(typedColumnConfig.Values)))
			id := hash.Bytes(buf.Bytes())
			idWithTags := hash.String(id + typedColumnConfig.Tags.ID())
			rawColumn = &context.RawFloatColumn{
				ComputedResourceFields: &context.ComputedResourceFields{
					ResourceFields: &context.ResourceFields{
						ID:           id,
						IDWithTags:   idWithTags,
						ResourceType: resource.RawColumnType,
						MetadataKey:  filepath.Join(consts.RawColumnsDir, id+"_metadata.json"),
					},
				},
				RawFloatColumn: typedColumnConfig,
			}
		case *userconfig.RawStringColumn:
			buf.WriteString(s.Bool(typedColumnConfig.Required))
			buf.WriteString(s.Obj(slices.SortStrsCopy(typedColumnConfig.Values)))
			id := hash.Bytes(buf.Bytes())
			idWithTags := hash.String(id + typedColumnConfig.Tags.ID())
			rawColumn = &context.RawStringColumn{
				ComputedResourceFields: &context.ComputedResourceFields{
					ResourceFields: &context.ResourceFields{
						ID:           id,
						IDWithTags:   idWithTags,
						ResourceType: resource.RawColumnType,
						MetadataKey:  filepath.Join(consts.RawColumnsDir, id+"_metadata.json"),
					},
				},
				RawStringColumn: typedColumnConfig,
			}
		default:
			return nil, errors.Wrap(configreader.ErrorInvalidStr(userconfig.TypeKey, userconfig.IntegerColumnType.String(), userconfig.FloatColumnType.String(), userconfig.StringColumnType.String()), userconfig.Identify(columnConfig)) // unexpected error
		}

		rawColumns[columnConfig.GetName()] = rawColumn
	}

	return rawColumns, nil
}
