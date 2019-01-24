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
	"github.com/cortexlabs/cortex/pkg/api/userconfig"
	"github.com/cortexlabs/cortex/pkg/utils/cast"
	"github.com/cortexlabs/cortex/pkg/utils/util"
)

type RawFeatures map[string]RawFeature

type RawFeature interface {
	Feature
	GetCompute() *userconfig.SparkCompute
	GetUserConfig() userconfig.Resource
}

type RawIntFeature struct {
	*userconfig.RawIntFeature
	*ComputedResourceFields
}

type RawFloatFeature struct {
	*userconfig.RawFloatFeature
	*ComputedResourceFields
}

type RawStringFeature struct {
	*userconfig.RawStringFeature
	*ComputedResourceFields
}

func (rawFeatures RawFeatures) OneByID(id string) RawFeature {
	for _, rawFeature := range rawFeatures {
		if rawFeature.GetID() == id {
			return rawFeature
		}
	}
	return nil
}

func (rawFeatures RawFeatures) featureInputsID(featuresValues map[string]interface{}, includeTags bool) string {
	featureIDMap := make(map[string]string)
	for featureInputName, featureValue := range featuresValues {
		if featureName, ok := featureValue.(string); ok {
			if includeTags {
				featureIDMap[featureInputName] = rawFeatures[featureName].GetIDWithTags()
			} else {
				featureIDMap[featureInputName] = rawFeatures[featureName].GetID()
			}
		}
		if featureNames, ok := cast.InterfaceToStrSlice(featureValue); ok {
			var featureIDs string
			for _, featureName := range featureNames {
				if includeTags {
					featureIDs = featureIDs + rawFeatures[featureName].GetIDWithTags()
				} else {
					featureIDs = featureIDs + rawFeatures[featureName].GetID()
				}
			}
			featureIDMap[featureInputName] = featureIDs
		}
	}
	return util.HashObj(featureIDMap)
}

func (rawFeatures RawFeatures) FeatureInputsID(featuresValues map[string]interface{}) string {
	return rawFeatures.featureInputsID(featuresValues, false)
}

func (rawFeatures RawFeatures) FeatureInputsIDWithTags(featuresValues map[string]interface{}) string {
	return rawFeatures.featureInputsID(featuresValues, true)
}
