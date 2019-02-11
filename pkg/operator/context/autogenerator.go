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
	"strings"

	"github.com/cortexlabs/cortex/pkg/api/context"
	"github.com/cortexlabs/cortex/pkg/api/resource"
	s "github.com/cortexlabs/cortex/pkg/api/strings"
	"github.com/cortexlabs/cortex/pkg/api/userconfig"
	"github.com/cortexlabs/cortex/pkg/utils/errors"
)

func autoGenerateConfig(
	config *userconfig.Config,
	userAggregators map[string]*context.Aggregator,
	userTransformers map[string]*context.Transformer,
) error {

	for _, aggregate := range config.Aggregates {
		for argName, argVal := range aggregate.Inputs.Args {
			if argValStr, ok := argVal.(string); ok {
				if s.HasPrefixAndSuffix(argValStr, "\"") {
					argVal = s.TrimPrefixAndSuffix(argValStr, "\"")
				} else {
					continue // assume it's a reference to a constant
				}
			}

			aggregator, err := getAggregator(aggregate.Aggregator, userAggregators)
			if err != nil {
				return errors.Wrap(err, userconfig.Identify(aggregate), userconfig.AggregatorKey)
			}
			argType, ok := aggregator.Inputs.Args[argName]
			if !ok {
				return errors.New(userconfig.Identify(aggregate), userconfig.InputsKey, userconfig.ArgsKey, s.ErrUnsupportedKey(argName))
			}

			constantName := strings.Join([]string{
				resource.AggregateType.String(),
				aggregate.Name,
				userconfig.InputsKey,
				userconfig.ArgsKey,
				argName,
			}, "/")

			constant := &userconfig.Constant{
				Name:  constantName,
				Type:  argType,
				Value: argVal,
				Tags:  make(map[string]interface{}),
			}
			config.Constants = append(config.Constants, constant)

			aggregate.Inputs.Args[argName] = constantName
		}
	}

	for _, transformedColumn := range config.TransformedColumns {
		for argName, argVal := range transformedColumn.Inputs.Args {
			if argValStr, ok := argVal.(string); ok {
				if s.HasPrefixAndSuffix(argValStr, "\"") {
					argVal = s.TrimPrefixAndSuffix(argValStr, "\"")
				} else {
					continue // assume it's a reference to a constant or aggregate
				}
			}

			transformer, err := getTransformer(transformedColumn.Transformer, userTransformers)
			if err != nil {
				return errors.Wrap(err, userconfig.Identify(transformedColumn), userconfig.TransformerKey)
			}
			argType, ok := transformer.Inputs.Args[argName]
			if !ok {
				return errors.New(userconfig.Identify(transformedColumn), userconfig.InputsKey, userconfig.ArgsKey, s.ErrUnsupportedKey(argName))
			}

			constantName := strings.Join([]string{
				resource.TransformedColumnType.String(),
				transformedColumn.Name,
				userconfig.InputsKey,
				userconfig.ArgsKey,
				argName,
			}, "/")

			constant := &userconfig.Constant{
				Name:  constantName,
				Type:  argType,
				Value: argVal,
				Tags:  make(map[string]interface{}),
			}
			config.Constants = append(config.Constants, constant)

			transformedColumn.Inputs.Args[argName] = constantName
		}
	}

	if err := config.Validate(config.Environment.Name); err != nil {
		return err
	}
	return nil
}
