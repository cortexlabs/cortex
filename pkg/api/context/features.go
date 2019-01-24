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
	"github.com/cortexlabs/cortex/pkg/api/resource"
	s "github.com/cortexlabs/cortex/pkg/api/strings"
	"github.com/cortexlabs/cortex/pkg/api/userconfig"
	"github.com/cortexlabs/cortex/pkg/utils/cast"
	"github.com/cortexlabs/cortex/pkg/utils/errors"
	"github.com/cortexlabs/cortex/pkg/utils/util"
)

type Features map[string]Feature

type Feature interface {
	ComputedResource
	GetType() string
	IsRaw() bool
}

func (ctx *Context) Features() Features {
	features := Features{}
	for name, feature := range ctx.RawFeatures {
		features[name] = feature
	}
	for name, feature := range ctx.TransformedFeatures {
		features[name] = feature
	}
	return features
}

func (ctx *Context) FeatureNames() map[string]bool {
	names := make(map[string]bool, len(ctx.RawFeatures)+len(ctx.TransformedFeatures))
	for name := range ctx.RawFeatures {
		names[name] = true
	}
	for name := range ctx.TransformedFeatures {
		names[name] = true
	}
	return names
}

func (ctx *Context) GetFeature(name string) Feature {
	if rawFeature, ok := ctx.RawFeatures[name]; ok {
		return rawFeature
	} else if transformedFeature, ok := ctx.TransformedFeatures[name]; ok {
		return transformedFeature
	}
	return nil
}

func (features Features) ID(featureNames []string) string {
	featureIDMap := make(map[string]string)
	for _, featureName := range featureNames {
		featureIDMap[featureName] = features[featureName].GetID()
	}
	return util.HashObj(featureIDMap)
}

func (features Features) IDWithTags(featureNames []string) string {
	featureIDMap := make(map[string]string)
	for _, featureName := range featureNames {
		featureIDMap[featureName] = features[featureName].GetIDWithTags()
	}
	return util.HashObj(featureIDMap)
}

func GetFeatureRuntimeTypes(
	featureValues map[string]interface{},
	rawFeatures RawFeatures,
) (map[string]interface{}, error) {

	err := userconfig.ValidateFeatureValues(featureValues)
	if err != nil {
		return nil, err
	}

	featureRuntimeTypes := make(map[string]interface{}, len(featureValues))

	for inputName, featureValue := range featureValues {
		if rawFeatureName, ok := featureValue.(string); ok {
			rawFeature, ok := rawFeatures[rawFeatureName]
			if !ok {
				return nil, errors.Wrap(userconfig.ErrorUndefinedResource(rawFeatureName, resource.RawFeatureType), inputName)
			}
			featureRuntimeTypes[inputName] = rawFeature.GetType()
			continue
		}

		if rawFeatureNames, ok := cast.InterfaceToStrSlice(featureValue); ok {
			rawFeatureTypes := make([]string, len(rawFeatureNames))
			for i, rawFeatureName := range rawFeatureNames {
				rawFeature, ok := rawFeatures[rawFeatureName]
				if !ok {
					return nil, errors.Wrap(userconfig.ErrorUndefinedResource(rawFeatureName, resource.RawFeatureType), inputName, s.Index(i))
				}
				rawFeatureTypes[i] = rawFeature.GetType()
			}
			featureRuntimeTypes[inputName] = rawFeatureTypes
			continue
		}

		return nil, errors.New(inputName, s.ErrInvalidPrimitiveType(featureValue, s.PrimTypeString, s.PrimTypeStringList)) // unexpected
	}

	return featureRuntimeTypes, nil
}
