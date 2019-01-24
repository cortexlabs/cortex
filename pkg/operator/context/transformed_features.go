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

	"github.com/cortexlabs/cortex/pkg/api/context"
	"github.com/cortexlabs/cortex/pkg/api/resource"
	s "github.com/cortexlabs/cortex/pkg/api/strings"
	"github.com/cortexlabs/cortex/pkg/api/userconfig"
	"github.com/cortexlabs/cortex/pkg/utils/errors"
	"github.com/cortexlabs/cortex/pkg/utils/util"
)

func getTransformedFeatures(
	config *userconfig.Config,
	constants context.Constants,
	rawFeatures context.RawFeatures,
	aggregates context.Aggregates,
	userTransformers map[string]*context.Transformer,
	root string,
) (context.TransformedFeatures, error) {

	transformedFeatures := context.TransformedFeatures{}

	for _, transformedFeatureConfig := range config.TransformedFeatures {
		transformer, err := getTransformer(transformedFeatureConfig.Transformer, userTransformers)
		if err != nil {
			return nil, errors.Wrap(err, userconfig.Identify(transformedFeatureConfig), userconfig.TransformerKey)
		}

		err = validateTransformedFeatureInputs(transformedFeatureConfig, constants, rawFeatures, aggregates, transformer)
		if err != nil {
			return nil, errors.Wrap(err)
		}

		valueResourceIDMap := make(map[string]string, len(transformedFeatureConfig.Inputs.Args))
		valueResourceIDWithTagsMap := make(map[string]string, len(transformedFeatureConfig.Inputs.Args))
		for argName, resourceName := range transformedFeatureConfig.Inputs.Args {
			resourceNameStr := resourceName.(string)
			resource, err := context.GetValueResource(resourceNameStr, constants, aggregates)
			if err != nil {
				return nil, errors.Wrap(err, userconfig.Identify(transformedFeatureConfig), userconfig.InputsKey, userconfig.ArgsKey, argName)
			}
			valueResourceIDMap[argName] = resource.GetID()
			valueResourceIDWithTagsMap[argName] = resource.GetIDWithTags()
		}

		var buf bytes.Buffer
		buf.WriteString(rawFeatures.FeatureInputsID(transformedFeatureConfig.Inputs.Features))
		buf.WriteString(s.Obj(valueResourceIDMap))
		buf.WriteString(transformer.ID)
		id := util.HashBytes(buf.Bytes())

		buf.Reset()
		buf.WriteString(rawFeatures.FeatureInputsIDWithTags(transformedFeatureConfig.Inputs.Features))
		buf.WriteString(s.Obj(valueResourceIDWithTagsMap))
		buf.WriteString(transformer.IDWithTags)
		buf.WriteString(transformedFeatureConfig.Tags.ID())
		idWithTags := util.HashBytes(buf.Bytes())

		transformedFeatures[transformedFeatureConfig.Name] = &context.TransformedFeature{
			ComputedResourceFields: &context.ComputedResourceFields{
				ResourceFields: &context.ResourceFields{
					ID:           id,
					IDWithTags:   idWithTags,
					ResourceType: resource.TransformedFeatureType,
				},
			},
			TransformedFeature: transformedFeatureConfig,
			Type:               transformer.OutputType,
		}
	}

	return transformedFeatures, nil
}

func validateTransformedFeatureInputs(
	transformedFeatureConfig *userconfig.TransformedFeature,
	constants context.Constants,
	rawFeatures context.RawFeatures,
	aggregates context.Aggregates,
	transformer *context.Transformer,
) error {

	featureRuntimeTypes, err := context.GetFeatureRuntimeTypes(transformedFeatureConfig.Inputs.Features, rawFeatures)
	if err != nil {
		return errors.Wrap(err, userconfig.Identify(transformedFeatureConfig), userconfig.InputsKey, userconfig.FeaturesKey)
	}
	err = userconfig.CheckFeatureRuntimeTypesMatch(featureRuntimeTypes, transformer.Inputs.Features)
	if err != nil {
		return errors.Wrap(err, userconfig.Identify(transformedFeatureConfig), userconfig.InputsKey, userconfig.FeaturesKey)
	}

	argTypes, err := getTransformedFeatureArgTypes(transformedFeatureConfig.Inputs.Args, constants, aggregates)
	if err != nil {
		return errors.Wrap(err, userconfig.Identify(transformedFeatureConfig), userconfig.InputsKey, userconfig.ArgsKey)
	}
	err = userconfig.CheckArgRuntimeTypesMatch(argTypes, transformer.Inputs.Args)
	if err != nil {
		return errors.Wrap(err, userconfig.Identify(transformedFeatureConfig), userconfig.InputsKey, userconfig.ArgsKey)
	}

	return nil
}

func getTransformedFeatureArgTypes(
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
