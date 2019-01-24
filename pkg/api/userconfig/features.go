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

package userconfig

import (
	"github.com/cortexlabs/cortex/pkg/api/resource"
	s "github.com/cortexlabs/cortex/pkg/api/strings"
	"github.com/cortexlabs/cortex/pkg/utils/cast"
	"github.com/cortexlabs/cortex/pkg/utils/errors"
	"github.com/cortexlabs/cortex/pkg/utils/util"
)

type Feature interface {
	GetName() string
	IsRaw() bool
}

func (config *Config) ValidateFeatures() error {
	dups := util.FindDuplicateStrs(config.FeatureNames())
	if len(dups) > 0 {
		return ErrorDuplicateConfigName(dups[0], resource.RawFeatureType, resource.TransformedFeatureType)
	}

	for _, aggregate := range config.Aggregates {
		err := ValidateFeatureInputsExistAndRaw(aggregate.Inputs.Features, config)
		if err != nil {
			return errors.Wrap(err, Identify(aggregate), InputsKey, FeaturesKey)
		}
	}

	for _, transformedFeature := range config.TransformedFeatures {
		err := ValidateFeatureInputsExistAndRaw(transformedFeature.Inputs.Features, config)
		if err != nil {
			return errors.Wrap(err, Identify(transformedFeature), InputsKey, FeaturesKey)
		}
	}

	return nil
}

func ValidateFeatureInputsExistAndRaw(featureValues map[string]interface{}, config *Config) error {
	for featureInputName, featureValue := range featureValues {
		if featureName, ok := featureValue.(string); ok {
			err := ValidateFeatureNameExistsAndRaw(featureName, config)
			if err != nil {
				return errors.Wrap(err, featureInputName)
			}
			continue
		}
		if featureNames, ok := cast.InterfaceToStrSlice(featureValue); ok {
			for i, featureName := range featureNames {
				err := ValidateFeatureNameExistsAndRaw(featureName, config)
				if err != nil {
					return errors.Wrap(err, featureInputName, s.Index(i))
				}
			}
			continue
		}
		return errors.New(featureInputName, s.ErrInvalidPrimitiveType(featureValue, s.PrimTypeString, s.PrimTypeStringList)) // unexpected
	}
	return nil
}

func ValidateFeatureNameExistsAndRaw(featureName string, config *Config) error {
	if config.IsTransformedFeature(featureName) {
		return ErrorFeatureMustBeRaw(featureName)
	}
	if !config.IsRawFeature(featureName) {
		return ErrorUndefinedResource(featureName, resource.RawFeatureType)
	}
	return nil
}

func (config *Config) FeatureNames() []string {
	return append(config.RawFeatures.Names(), config.TransformedFeatures.Names()...)
}

func (config *Config) IsRawFeature(name string) bool {
	return util.IsStrInSlice(name, config.RawFeatures.Names())
}

func (config *Config) IsTransformedFeature(name string) bool {
	return util.IsStrInSlice(name, config.TransformedFeatures.Names())
}
