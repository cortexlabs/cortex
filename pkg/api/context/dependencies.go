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
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
)

func (ctx *Context) AllComputedResourceDependencies(resourceID string) strset.Set {
	dependencies := ctx.DirectComputedResourceDependencies(resourceID)
	for dependency := range dependencies.Copy() {
		for subDependency := range ctx.AllComputedResourceDependencies(dependency) {
			dependencies.Add(subDependency)
		}
	}
	return dependencies
}

func (ctx *Context) DirectComputedResourceDependencies(resourceID string) strset.Set {
	for _, pythonPackage := range ctx.PythonPackages {
		if pythonPackage.GetID() == resourceID {
			return ctx.pythonPackageDependencies(pythonPackage)
		}
	}
	for _, rawColumn := range ctx.RawColumns {
		if rawColumn.GetID() == resourceID {
			return ctx.rawColumnDependencies(rawColumn)
		}
	}
	for _, aggregate := range ctx.Aggregates {
		if aggregate.ID == resourceID {
			return ctx.aggregatesDependencies(aggregate)
		}
	}
	for _, transformedColumn := range ctx.TransformedColumns {
		if transformedColumn.ID == resourceID {
			return ctx.transformedColumnDependencies(transformedColumn)
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
	return make(strset.Set)
}

func (ctx *Context) pythonPackageDependencies(pythonPackage *PythonPackage) strset.Set {
	return make(strset.Set)
}

func (ctx *Context) rawColumnDependencies(rawColumn RawColumn) strset.Set {
	// Currently python packages are a dependency on raw features because raw features share
	// the same workload as transformed features and aggregates.
	dependencies := make(strset.Set)
	for _, pythonPackage := range ctx.PythonPackages {
		dependencies.Add(pythonPackage.GetID())
	}
	return dependencies
}

func (ctx *Context) aggregatesDependencies(aggregate *Aggregate) strset.Set {
	rawColumnNames := aggregate.InputColumnNames()
	dependencies := make(strset.Set, len(rawColumnNames))
	for _, pythonPackage := range ctx.PythonPackages {
		dependencies.Add(pythonPackage.GetID())
	}
	for rawColumnName := range rawColumnNames {
		rawColumn := ctx.RawColumns[rawColumnName]
		dependencies.Add(rawColumn.GetID())
	}
	return dependencies
}

func (ctx *Context) transformedColumnDependencies(transformedColumn *TransformedColumn) strset.Set {
	dependencies := make(strset.Set)

	for _, pythonPackage := range ctx.PythonPackages {
		dependencies.Add(pythonPackage.GetID())
	}

	rawColumnNames := transformedColumn.InputColumnNames()
	for rawColumnName := range rawColumnNames {
		rawColumn := ctx.RawColumns[rawColumnName]
		dependencies.Add(rawColumn.GetID())
	}

	aggregateNames := transformedColumn.InputAggregateNames(ctx)
	for aggregateName := range aggregateNames {
		aggregate := ctx.Aggregates[aggregateName]
		dependencies.Add(aggregate.GetID())
	}

	return dependencies
}

func (ctx *Context) trainingDatasetDependencies(model *Model) strset.Set {
	dependencies := make(strset.Set)
	for _, columnName := range model.AllColumnNames() {
		column := ctx.GetColumn(columnName)
		dependencies.Add(column.GetID())
	}
	return dependencies
}

func (ctx *Context) modelDependencies(model *Model) strset.Set {
	dependencies := make(strset.Set)

	for _, pythonPackage := range ctx.PythonPackages {
		dependencies.Add(pythonPackage.GetID())
	}

	dependencies.Add(model.Dataset.ID)
	for _, aggregate := range model.Aggregates {
		dependencies.Add(ctx.Aggregates[aggregate].GetID())
	}
	return dependencies
}

func (ctx *Context) apiDependencies(api *API) strset.Set {
	model := ctx.Models[api.ModelName]
	return strset.New(model.ID)
}
