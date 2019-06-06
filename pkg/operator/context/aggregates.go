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
	"github.com/cortexlabs/cortex/pkg/operator/api/context"
	"github.com/cortexlabs/cortex/pkg/operator/api/resource"
	"github.com/cortexlabs/cortex/pkg/operator/api/userconfig"
)

func getAggregates(
	config *userconfig.Config,
	constants context.Constants,
	rawColumns context.RawColumns,
	aggregators context.Aggregators,
	root string,
) (context.Aggregates, error) {

	aggregates := context.Aggregates{}

	for _, aggregateConfig := range config.Aggregates {
		aggregator := aggregators[aggregateConfig.Aggregator]

		var validInputResources []context.Resource
		for _, res := range constants {
			validInputResources = append(validInputResources, res)
		}
		for _, res := range rawColumns {
			validInputResources = append(validInputResources, res)
		}

		castedInput, inputID, err := ValidateInput(
			aggregateConfig.Input,
			aggregator.Input,
			[]resource.Type{resource.RawColumnType, resource.ConstantType},
			validInputResources,
			config.Resources,
			nil,
			nil,
		)
		if err != nil {
			return nil, errors.Wrap(err, userconfig.Identify(aggregateConfig), userconfig.InputKey)
		}
		aggregateConfig.Input = castedInput

		var buf bytes.Buffer
		buf.WriteString(inputID)
		buf.WriteString(aggregator.ID)
		id := hash.Bytes(buf.Bytes())

		aggregateKey := filepath.Join(
			root,
			consts.AggregatesDir,
			id,
		) + ".msgpack"

		aggregates[aggregateConfig.Name] = &context.Aggregate{
			ComputedResourceFields: &context.ComputedResourceFields{
				ResourceFields: &context.ResourceFields{
					ID:           id,
					ResourceType: resource.AggregateType,
				},
			},
			Aggregate: aggregateConfig,
			Type:      aggregator.OutputType,
			Key:       aggregateKey,
		}
	}

	return aggregates, nil
}
