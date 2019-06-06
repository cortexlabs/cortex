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

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/hash"
	"github.com/cortexlabs/cortex/pkg/operator/api/context"
	"github.com/cortexlabs/cortex/pkg/operator/api/resource"
	"github.com/cortexlabs/cortex/pkg/operator/api/userconfig"
)

func getTransformedColumns(
	config *userconfig.Config,
	constants context.Constants,
	rawColumns context.RawColumns,
	aggregates context.Aggregates,
	aggregators context.Aggregators,
	transformers context.Transformers,
	root string,
) (context.TransformedColumns, error) {

	transformedColumns := context.TransformedColumns{}

	for _, transformedColumnConfig := range config.TransformedColumns {
		transformer := transformers[transformedColumnConfig.Transformer]

		var validInputResources []context.Resource
		for _, res := range constants {
			validInputResources = append(validInputResources, res)
		}
		for _, res := range rawColumns {
			validInputResources = append(validInputResources, res)
		}
		for _, res := range aggregates {
			validInputResources = append(validInputResources, res)
		}

		castedInput, inputID, err := ValidateInput(
			transformedColumnConfig.Input,
			transformer.Input,
			[]resource.Type{resource.RawColumnType, resource.ConstantType, resource.AggregateType},
			validInputResources,
			config.Resources,
			aggregators,
			nil,
		)
		if err != nil {
			return nil, errors.Wrap(err, userconfig.Identify(transformedColumnConfig), userconfig.InputKey)
		}
		transformedColumnConfig.Input = castedInput

		var buf bytes.Buffer
		buf.WriteString(inputID)
		buf.WriteString(transformer.ID)
		id := hash.Bytes(buf.Bytes())

		transformedColumns[transformedColumnConfig.Name] = &context.TransformedColumn{
			ComputedResourceFields: &context.ComputedResourceFields{
				ResourceFields: &context.ResourceFields{
					ID:           id,
					ResourceType: resource.TransformedColumnType,
				},
			},
			TransformedColumn: transformedColumnConfig,
			Type:              transformer.OutputType,
		}
	}

	return transformedColumns, nil
}
