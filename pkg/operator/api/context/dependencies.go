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
	"sort"

	"github.com/cortexlabs/yaml"

	"github.com/cortexlabs/cortex/pkg/lib/cast"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	"github.com/cortexlabs/cortex/pkg/operator/api/resource"
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
		for _, pythonPackage := range ctx.PythonPackages {
			if pythonPackage.GetID() == resourceID {
				allDependencies.Merge(ctx.pythonPackageDependencies(pythonPackage))
			}
		}
		for _, api := range ctx.APIs {
			if api.ID == resourceID {
				allDependencies.Merge(ctx.apiDependencies(api))
			}
		}
	}
	allDependencies.Remove(resourceIDs...)
	return allDependencies
}

func (ctx *Context) pythonPackageDependencies(pythonPackage *PythonPackage) strset.Set {
	return strset.New()
}

func (ctx *Context) apiDependencies(api *API) strset.Set {
	dependencies := strset.New()

	if api.RequestHandler != nil {
		for _, pythonPackage := range ctx.PythonPackages {
			dependencies.Add(pythonPackage.GetID())
		}
	}
	return dependencies
}

func (ctx *Context) ExtractCortexResources(
	input interface{},
	resourceTypes ...resource.Type, // indicates which resource types to include in the query; if none are passed in, no filter is applied
) []Resource {

	return ExtractCortexResources(input, ctx.AllResources(), resourceTypes...)
}

func ExtractCortexResources(
	input interface{},
	validResources []Resource,
	resourceTypes ...resource.Type, // indicates which resource types to include in the query; if none are passed in, no filter is applied
) []Resource {

	resourceTypeFilter := make(map[resource.Type]bool)
	for _, resourceType := range resourceTypes {
		resourceTypeFilter[resourceType] = true
	}

	validResourcesMap := make(map[string][]Resource)
	for _, res := range validResources {
		validResourcesMap[res.GetName()] = append(validResourcesMap[res.GetName()], res)
	}

	resources := make(map[string]Resource)
	extractCortexResourcesHelper(input, validResourcesMap, resourceTypeFilter, resources)

	// convert to slice and sort by ID
	var resourceIDs []string
	for resourceID := range resources {
		resourceIDs = append(resourceIDs, resourceID)
	}
	sort.Strings(resourceIDs)
	resoucesSlice := make([]Resource, len(resources))
	for i, resourceID := range resourceIDs {
		resoucesSlice[i] = resources[resourceID]
	}

	return resoucesSlice
}

func extractCortexResourcesHelper(
	input interface{},
	validResourcesMap map[string][]Resource, // key is resource name
	resourceTypeFilter map[resource.Type]bool,
	collectedResources map[string]Resource,
) {

	if input == nil {
		return
	}

	if resourceName, ok := yaml.ExtractAtSymbolTextInter(input); ok {
		for _, res := range validResourcesMap[resourceName] {
			foundMatch := false
			if len(resourceTypeFilter) == 0 || resourceTypeFilter[res.GetResourceType()] == true {
				if foundMatch {
					errors.Panic("found multiple resources with the same name", resourceName) // unexpected
				}
				collectedResources[res.GetID()] = res
				foundMatch = true
			}
		}
		return
	}

	if inputSlice, ok := cast.InterfaceToInterfaceSlice(input); ok {
		for _, elem := range inputSlice {
			extractCortexResourcesHelper(elem, validResourcesMap, resourceTypeFilter, collectedResources)
		}
		return
	}

	if inputMap, ok := cast.InterfaceToInterfaceInterfaceMap(input); ok {
		for key, val := range inputMap {
			extractCortexResourcesHelper(key, validResourcesMap, resourceTypeFilter, collectedResources)
			extractCortexResourcesHelper(val, validResourcesMap, resourceTypeFilter, collectedResources)
		}
		return
	}
}
