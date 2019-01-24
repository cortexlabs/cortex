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
)

type TransformedFeatures map[string]*TransformedFeature

type TransformedFeature struct {
	*userconfig.TransformedFeature
	*ComputedResourceFields
	Type string `json:"type"`
}

func (feature *TransformedFeature) GetType() string {
	return feature.Type
}

// Returns map[string]string because after autogen, arg values are constant or aggregate names
func (transformedFeature *TransformedFeature) Args() map[string]string {
	args, _ := cast.InterfaceToStrStrMap(transformedFeature.Inputs.Args)
	return args
}

func (transformedFeature *TransformedFeature) InputAggregateNames(ctx *Context) map[string]bool {
	inputAggregateNames := make(map[string]bool)
	for _, valueResourceName := range transformedFeature.Args() {
		if _, ok := ctx.Aggregates[valueResourceName]; ok {
			inputAggregateNames[valueResourceName] = true
		}
	}
	return inputAggregateNames
}

func (transformedFeatures TransformedFeatures) OneByID(id string) *TransformedFeature {
	for _, transformedFeature := range transformedFeatures {
		if transformedFeature.ID == id {
			return transformedFeature
		}
	}
	return nil
}
