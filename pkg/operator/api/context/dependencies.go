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

// Get all dependencies for resourceID(s). Note: provided resourceIDs are not included in the dependency set
func (ctx *Context) AllComputedResourceDependencies(resourceIDs ...string) strset.Set {
	allDependencies := strset.New()
	for _, resourceID := range resourceIDs {
		ctx.allComputedResourceDependenciesHelper(resourceID, allDependencies)
	}
	allDependencies.Remove(resourceIDs...)
	return allDependencies
}

func (ctx *Context) allComputedResourceDependenciesHelper(resourceID string, allDependencies strset.Set) {
	subDependencies := ctx.DirectComputedResourceDependencies(resourceID)
	subDependencies.Subtract(allDependencies)
	allDependencies.Merge(subDependencies)

	for dependency := range subDependencies {
		ctx.allComputedResourceDependenciesHelper(dependency, allDependencies)
	}
}

// Get all dependencies for resourceID(s). Note: provided resourceIDs are not included in the dependency set
func (ctx *Context) DirectComputedResourceDependencies(resourceIDs ...string) strset.Set {
	allDependencies := strset.New()
	for _, resourceID := range resourceIDs {
		for _, api := range ctx.APIs {
			if api.ID == resourceID {
				allDependencies.Merge(ctx.apiDependencies(api))
			}
		}
	}
	allDependencies.Remove(resourceIDs...)
	return allDependencies
}

func (ctx *Context) apiDependencies(api *API) strset.Set {
	dependencies := strset.New()
	return dependencies
}
