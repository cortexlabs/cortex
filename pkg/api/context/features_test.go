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

package context_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/cortexlabs/cortex/pkg/api/context"
	"github.com/cortexlabs/cortex/pkg/api/userconfig"
	cr "github.com/cortexlabs/cortex/pkg/utils/configreader"
)

func TestGetFeatureRuntimeTypes(t *testing.T) {
	var featureValues map[string]interface{}
	var expected map[string]interface{}

	rawFeatures := context.RawFeatures{
		"rfInt": &context.RawIntFeature{
			RawIntFeature: &userconfig.RawIntFeature{
				Type: "INT_FEATURE",
			},
		},
		"rfFloat": &context.RawFloatFeature{
			RawFloatFeature: &userconfig.RawFloatFeature{
				Type: "FLOAT_FEATURE",
			},
		},
		"rfStr": &context.RawStringFeature{
			RawStringFeature: &userconfig.RawStringFeature{
				Type: "STRING_FEATURE",
			},
		},
	}

	featureValues = cr.MustReadYAMLStrMap("in: rfInt")
	expected = map[string]interface{}{"in": "INT_FEATURE"}
	checkTestGetFeatureRuntimeTypes(featureValues, rawFeatures, expected, t)

	featureValues = cr.MustReadYAMLStrMap("in: rfStr")
	expected = map[string]interface{}{"in": "STRING_FEATURE"}
	checkTestGetFeatureRuntimeTypes(featureValues, rawFeatures, expected, t)

	featureValues = cr.MustReadYAMLStrMap("in: [rfFloat]")
	expected = map[string]interface{}{"in": []string{"FLOAT_FEATURE"}}
	checkTestGetFeatureRuntimeTypes(featureValues, rawFeatures, expected, t)

	featureValues = cr.MustReadYAMLStrMap("in: [rfInt, rfFloat, rfStr, rfInt]")
	expected = map[string]interface{}{"in": []string{"INT_FEATURE", "FLOAT_FEATURE", "STRING_FEATURE", "INT_FEATURE"}}
	checkTestGetFeatureRuntimeTypes(featureValues, rawFeatures, expected, t)

	featureValues = cr.MustReadYAMLStrMap("in1: [rfInt, rfFloat]\nin2: rfStr")
	expected = map[string]interface{}{"in1": []string{"INT_FEATURE", "FLOAT_FEATURE"}, "in2": "STRING_FEATURE"}
	checkTestGetFeatureRuntimeTypes(featureValues, rawFeatures, expected, t)

	featureValues = cr.MustReadYAMLStrMap("in: 1")
	checkErrTestGetFeatureRuntimeTypes(featureValues, rawFeatures, t)

	featureValues = cr.MustReadYAMLStrMap("in: [1, 2, 3]")
	checkErrTestGetFeatureRuntimeTypes(featureValues, rawFeatures, t)

	featureValues = cr.MustReadYAMLStrMap("in: {in: rfInt}")
	checkErrTestGetFeatureRuntimeTypes(featureValues, rawFeatures, t)

	featureValues = cr.MustReadYAMLStrMap("in: rfMissing")
	checkErrTestGetFeatureRuntimeTypes(featureValues, rawFeatures, t)

	featureValues = cr.MustReadYAMLStrMap("in: [rfMissing]")
	checkErrTestGetFeatureRuntimeTypes(featureValues, rawFeatures, t)
}

func checkTestGetFeatureRuntimeTypes(featureValues map[string]interface{}, rawFeatures context.RawFeatures, expected map[string]interface{}, t *testing.T) {
	runtimeTypes, err := context.GetFeatureRuntimeTypes(featureValues, rawFeatures)
	require.NoError(t, err)
	require.Equal(t, expected, runtimeTypes)
}

func checkErrTestGetFeatureRuntimeTypes(featureValues map[string]interface{}, rawFeatures context.RawFeatures, t *testing.T) {
	_, err := context.GetFeatureRuntimeTypes(featureValues, rawFeatures)
	require.Error(t, err)
}
