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
	"github.com/cortexlabs/cortex/pkg/utils/util"
)

func (ctx *Context) AllComputedResourceDependencies(resourceID string) map[string]bool {
	dependencies := ctx.DirectComputedResourceDependencies(resourceID)
	for dependency := range util.CopyStrSet(dependencies) {
		for subDependency := range ctx.AllComputedResourceDependencies(dependency) {
			dependencies[subDependency] = true
		}
	}
	return dependencies
}

func (ctx *Context) DirectComputedResourceDependencies(resourceID string) map[string]bool {
	for _, pythonPackage := range ctx.PythonPackages {
		if pythonPackage.GetID() == resourceID {
			return ctx.pythonPackageDependencies(pythonPackage)
		}
	}
	for _, rawFeature := range ctx.RawFeatures {
		if rawFeature.GetID() == resourceID {
			return ctx.rawFeatureDependencies(rawFeature)
		}
	}
	for _, aggregate := range ctx.Aggregates {
		if aggregate.ID == resourceID {
			return ctx.aggregatesDependencies(aggregate)
		}
	}
	for _, transformedFeature := range ctx.TransformedFeatures {
		if transformedFeature.ID == resourceID {
			return ctx.transformedFeatureDependencies(transformedFeature)
		}
	}
	for _, model := range ctx.Models {
		if model.ID == resourceID {
			return ctx.modelDependencies(model)
		}
		if model.Dataset.ID == resourceID {
			return ctx.trainingDatasetDependencies(model)
		}
	}
	for _, api := range ctx.APIs {
		if api.ID == resourceID {
			return ctx.apiDependencies(api)
		}
	}
	return make(map[string]bool)
}

func (ctx *Context) pythonPackageDependencies(pythonPackage *PythonPackage) map[string]bool {
	return make(map[string]bool)
}

func (ctx *Context) rawFeatureDependencies(rawFeature RawFeature) map[string]bool {
	// Currently python packages are a dependency on raw features because raw features share
	// the same workload as transformed features and aggregates.
	dependencies := make(map[string]bool)
	for _, pythonPackage := range ctx.PythonPackages {
		dependencies[pythonPackage.GetID()] = true
	}
	return dependencies
}

func (ctx *Context) aggregatesDependencies(aggregate *Aggregate) map[string]bool {
	rawFeatureNames := aggregate.InputFeatureNames()
	dependencies := make(map[string]bool, len(rawFeatureNames))

	for _, pythonPackage := range ctx.PythonPackages {
		dependencies[pythonPackage.GetID()] = true
	}

	for rawFeatureName := range rawFeatureNames {
		rawFeature := ctx.RawFeatures[rawFeatureName]
		dependencies[rawFeature.GetID()] = true
	}
	return dependencies
}

func (ctx *Context) transformedFeatureDependencies(transformedFeature *TransformedFeature) map[string]bool {
	dependencies := make(map[string]bool)

	for _, pythonPackage := range ctx.PythonPackages {
		dependencies[pythonPackage.GetID()] = true
	}

	rawFeatureNames := transformedFeature.InputFeatureNames()
	for rawFeatureName := range rawFeatureNames {
		rawFeature := ctx.RawFeatures[rawFeatureName]
		dependencies[rawFeature.GetID()] = true
	}

	aggregateNames := transformedFeature.InputAggregateNames(ctx)
	for aggregateName := range aggregateNames {
		aggregate := ctx.Aggregates[aggregateName]
		dependencies[aggregate.GetID()] = true
	}

	return dependencies
}

func (ctx *Context) trainingDatasetDependencies(model *Model) map[string]bool {
	dependencies := make(map[string]bool)
	for _, featureName := range model.AllFeatureNames() {
		feature := ctx.GetFeature(featureName)
		dependencies[feature.GetID()] = true
	}
	return dependencies
}

func (ctx *Context) modelDependencies(model *Model) map[string]bool {
	dependencies := make(map[string]bool)

	for _, pythonPackage := range ctx.PythonPackages {
		dependencies[pythonPackage.GetID()] = true
	}

	dependencies[model.Dataset.ID] = true
	for _, aggregate := range model.Aggregates {
		dependencies[ctx.Aggregates[aggregate].GetID()] = true
	}
	return dependencies
}

func (ctx *Context) apiDependencies(api *API) map[string]bool {
	model := ctx.Models[api.ModelName]
	return map[string]bool{model.ID: true}
}
