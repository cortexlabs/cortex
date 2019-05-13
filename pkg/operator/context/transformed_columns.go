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
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/hash"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/operator/api/context"
	"github.com/cortexlabs/cortex/pkg/operator/api/resource"
	"github.com/cortexlabs/cortex/pkg/operator/api/userconfig"
)

func getTransformedColumns(
	config *userconfig.Config,
	constants context.Constants,
	rawColumns context.RawColumns,
	aggregates context.Aggregates,
	userTransformers map[string]*context.Transformer,
	root string,
) (context.TransformedColumns, error) {

	transformedColumns := context.TransformedColumns{}

	for _, transformedColumnConfig := range config.TransformedColumns {
		transformer, err := getTransformer(transformedColumnConfig.Transformer, userTransformers)
		if err != nil {
			return nil, errors.Wrap(err, userconfig.Identify(transformedColumnConfig), userconfig.TransformerKey)
		}

		err = validateTransformedColumnInputs(transformedColumnConfig, constants, rawColumns, aggregates, transformer)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		valueResourceIDMap := make(map[string]string, len(transformedColumnConfig.Inputs.Args))
		valueResourceIDWithTagsMap := make(map[string]string, len(transformedColumnConfig.Inputs.Args))
		for argName, resourceName := range transformedColumnConfig.Inputs.Args {
			resourceNameStr := resourceName.(string)
			resource, err := context.GetValueResource(resourceNameStr, constants, aggregates)
			if err != nil {
				return nil, errors.Wrap(err, userconfig.Identify(transformedColumnConfig), userconfig.InputsKey, userconfig.ArgsKey, argName)
			}
			valueResourceIDMap[argName] = resource.GetID()
			valueResourceIDWithTagsMap[argName] = resource.GetIDWithTags()
		}

		var buf bytes.Buffer
		buf.WriteString(rawColumns.ColumnInputsID(transformedColumnConfig.Inputs.Columns))
		buf.WriteString(s.Obj(valueResourceIDMap))
		buf.WriteString(transformer.ID)
		id := hash.Bytes(buf.Bytes())

		buf.Reset()
		buf.WriteString(rawColumns.ColumnInputsIDWithTags(transformedColumnConfig.Inputs.Columns))
		buf.WriteString(s.Obj(valueResourceIDWithTagsMap))
		buf.WriteString(transformer.IDWithTags)
		buf.WriteString(transformedColumnConfig.Tags.ID())
		idWithTags := hash.Bytes(buf.Bytes())

		transformedColumns[transformedColumnConfig.Name] = &context.TransformedColumn{
			ComputedResourceFields: &context.ComputedResourceFields{
				ResourceFields: &context.ResourceFields{
					ID:           id,
					IDWithTags:   idWithTags,
					ResourceType: resource.TransformedColumnType,
					MetadataKey:  filepath.Join(consts.TransformedColumnsDir, id+"_metadata.json"),
				},
			},
			TransformedColumn: transformedColumnConfig,
			Type:              transformer.OutputType,
		}
	}

	return transformedColumns, nil
}

func validateTransformedColumnInputs(
	transformedColumnConfig *userconfig.TransformedColumn,
	constants context.Constants,
	rawColumns context.RawColumns,
	aggregates context.Aggregates,
	transformer *context.Transformer,
) error {
	if transformedColumnConfig.TransformerPath != nil {
		return nil
	}

	columnRuntimeTypes, err := context.GetColumnRuntimeTypes(transformedColumnConfig.Inputs.Columns, rawColumns)
	if err != nil {
		return errors.Wrap(err, userconfig.Identify(transformedColumnConfig), userconfig.InputsKey, userconfig.ColumnsKey)
	}
	err = userconfig.CheckColumnRuntimeTypesMatch(columnRuntimeTypes, transformer.Inputs.Columns)
	if err != nil {
		return errors.Wrap(err, userconfig.Identify(transformedColumnConfig), userconfig.InputsKey, userconfig.ColumnsKey)
	}

	argTypes, err := getTransformedColumnArgTypes(transformedColumnConfig.Inputs.Args, constants, aggregates)
	if err != nil {
		return errors.Wrap(err, userconfig.Identify(transformedColumnConfig), userconfig.InputsKey, userconfig.ArgsKey)
	}
	err = userconfig.CheckArgRuntimeTypesMatch(argTypes, transformer.Inputs.Args)
	if err != nil {
		return errors.Wrap(err, userconfig.Identify(transformedColumnConfig), userconfig.InputsKey, userconfig.ArgsKey)
	}

	return nil
}

func getTransformedColumnArgTypes(
	args map[string]interface{},
	constants context.Constants,
	aggregates context.Aggregates,
) (map[string]interface{}, error) {

	if len(args) == 0 {
		return nil, nil
	}

	argTypes := make(map[string]interface{}, len(args))
	for argName, valueResourceName := range args {
		valueResourceNameStr := valueResourceName.(string)
		valueResource, err := context.GetValueResource(valueResourceNameStr, constants, aggregates)
		if err != nil {
			return nil, errors.Wrap(err, argName)
		}
		argTypes[argName] = valueResource.GetType()
	}
	return argTypes, nil
}
