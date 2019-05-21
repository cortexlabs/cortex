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
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	"github.com/cortexlabs/cortex/pkg/operator/api/resource"
	userconfig "github.com/cortexlabs/cortex/pkg/operator/api/userconfig"
	"github.com/cortexlabs/cortex/pkg/operator/config"
)

type Context struct {
	ID                 string               `json:"id"`
	Key                string               `json:"key"`
	CortexConfig       *config.CortexConfig `json:"cortex_config"`
	DatasetVersion     string               `json:"dataset_version"`
	Root               string               `json:"root"`
	RawDataset         RawDataset           `json:"raw_dataset"`
	StatusPrefix       string               `json:"status_prefix"`
	App                *App                 `json:"app"`
	Environment        *Environment         `json:"environment"`
	PythonPackages     PythonPackages       `json:"python_packages"`
	RawColumns         RawColumns           `json:"-"`
	Aggregates         Aggregates           `json:"aggregates"`
	TransformedColumns TransformedColumns   `json:"transformed_columns"`
	Models             Models               `json:"models"`
	APIs               APIs                 `json:"apis"`
	Constants          Constants            `json:"constants"`
	Aggregators        Aggregators          `json:"aggregators"`
	Transformers       Transformers         `json:"transformers"`
}

type RawDataset struct {
	Key         string `json:"key"`
	MetadataKey string `json:"metadata_key"`
}

type Resource interface {
	userconfig.Resource
	GetID() string
	GetIDWithTags() string
	GetResourceFields() *ResourceFields
	GetMetadataKey() string
}

type ComputedResource interface {
	Resource
	GetWorkloadID() string
	SetWorkloadID(string)
}

type ValueResource interface {
	Resource
	GetType() interface{}
}

type ResourceFields struct {
	ID           string        `json:"id"`
	IDWithTags   string        `json:"id_with_tags"`
	ResourceType resource.Type `json:"resource_type"`
	MetadataKey  string        `json:"metadata_key"`
}

type ComputedResourceFields struct {
	*ResourceFields
	WorkloadID string `json:"workload_id"`
}

func (r *ResourceFields) GetID() string {
	return r.ID
}

func (r *ResourceFields) GetIDWithTags() string {
	return r.IDWithTags
}

func (r *ResourceFields) GetResourceFields() *ResourceFields {
	return r
}

func (r *ResourceFields) GetMetadataKey() string {
	return r.MetadataKey
}

func (r *ComputedResourceFields) GetWorkloadID() string {
	return r.WorkloadID
}

func (r *ComputedResourceFields) SetWorkloadID(workloadID string) {
	r.WorkloadID = workloadID
}

func ExtractResourceWorkloadIDs(resources []ComputedResource) map[string]string {
	resourceWorkloadIDs := make(map[string]string, len(resources))
	for _, resource := range resources {
		resourceWorkloadIDs[resource.GetID()] = resource.GetWorkloadID()
	}
	return resourceWorkloadIDs
}

func (ctx *Context) DataComputedResources() []ComputedResource {
	var resources []ComputedResource
	for _, pythonPackage := range ctx.PythonPackages {
		resources = append(resources, pythonPackage)
	}
	for _, rawColumn := range ctx.RawColumns {
		resources = append(resources, rawColumn)
	}
	for _, aggregate := range ctx.Aggregates {
		resources = append(resources, aggregate)
	}
	for _, transformedColumn := range ctx.TransformedColumns {
		resources = append(resources, transformedColumn)
	}
	for _, model := range ctx.Models {
		resources = append(resources, model, model.Dataset)
	}
	return resources
}

func (ctx *Context) APIResources() []ComputedResource {
	resources := make([]ComputedResource, len(ctx.APIs))
	i := 0
	for _, api := range ctx.APIs {
		resources[i] = api
		i++
	}
	return resources
}

func (ctx *Context) ComputedResources() []ComputedResource {
	return append(ctx.DataComputedResources(), ctx.APIResources()...)
}

func (ctx *Context) AllResources() []Resource {
	var resources []Resource
	for _, resource := range ctx.ComputedResources() {
		resources = append(resources, resource)
	}
	for _, constant := range ctx.Constants {
		resources = append(resources, constant)
	}
	for _, aggregator := range ctx.Aggregators {
		resources = append(resources, aggregator)
	}
	for _, transformer := range ctx.Transformers {
		resources = append(resources, transformer)
	}
	return resources
}

func (ctx *Context) ComputedResourceIDs() strset.Set {
	resourceIDs := make(strset.Set)
	for _, resource := range ctx.ComputedResources() {
		resourceIDs.Add(resource.GetID())
	}
	return resourceIDs
}

func (ctx *Context) DataResourceWorkloadIDs() map[string]string {
	return ExtractResourceWorkloadIDs(ctx.DataComputedResources())
}

func (ctx *Context) APIResourceWorkloadIDs() map[string]string {
	return ExtractResourceWorkloadIDs(ctx.APIResources())
}

func (ctx *Context) ComputedResourceResourceWorkloadIDs() map[string]string {
	return ExtractResourceWorkloadIDs(ctx.ComputedResources())
}

func (ctx *Context) ComputedResourceWorkloadIDs() strset.Set {
	workloadIDs := strset.New()
	for _, workloadID := range ExtractResourceWorkloadIDs(ctx.ComputedResources()) {
		workloadIDs.Add(workloadID)
	}
	return workloadIDs
}

// Note: there may be >1 resources with the ID, this returns one of them
func (ctx *Context) OneResourceByID(resourceID string) Resource {
	for _, resource := range ctx.AllResources() {
		if resource.GetID() == resourceID {
			return resource
		}
	}
	return nil
}

// Overwrites any existing workload IDs
func (ctx *Context) PopulateWorkloadIDs(resourceWorkloadIDs map[string]string) {
	for _, resource := range ctx.ComputedResources() {
		if workloadID, ok := resourceWorkloadIDs[resource.GetID()]; ok {
			resource.SetWorkloadID(workloadID)
		}
	}
}

func (ctx *Context) CheckAllWorkloadIDsPopulated() error {
	for _, resource := range ctx.ComputedResources() {
		if resource.GetWorkloadID() == "" {
			return errors.New(ctx.App.Name, "resource", resource.GetID(), "workload ID is missing") // unexpected
		}
	}
	return nil
}

func (ctx *Context) VisibleResourcesMap() map[string][]ComputedResource {
	resources := make(map[string][]ComputedResource)
	for name, pythonPackage := range ctx.PythonPackages {
		resources[name] = append(resources[name], pythonPackage)
	}
	for name, rawColumn := range ctx.RawColumns {
		resources[name] = append(resources[name], rawColumn)
	}
	for name, aggregate := range ctx.Aggregates {
		resources[name] = append(resources[name], aggregate)
	}
	for name, transformedColumn := range ctx.TransformedColumns {
		resources[name] = append(resources[name], transformedColumn)
	}
	for name, model := range ctx.Models {
		resources[name] = append(resources[name], model)
		resources[model.Dataset.Name] = append(resources[model.Dataset.Name], model.Dataset)
	}
	for name, api := range ctx.APIs {
		resources[name] = append(resources[name], api)
	}
	return resources
}

func (ctx *Context) VisibleResourcesByName(name string) []ComputedResource {
	return ctx.VisibleResourcesMap()[name]
}

func (ctx *Context) VisibleResourceByName(name string) (ComputedResource, error) {
	resources := ctx.VisibleResourcesByName(name)
	if len(resources) == 0 {
		return nil, resource.ErrorNameNotFound(name)
	}
	if len(resources) > 1 {
		validStrs := make([]string, len(resources))
		for i, resource := range resources {
			resourceTypeStr := resource.GetResourceType().String()
			validStrs[i] = resourceTypeStr + " " + name
		}
		return nil, resource.ErrorBeMoreSpecific(validStrs...)
	}
	return resources[0], nil
}

func (ctx *Context) VisibleResourceByNameAndType(name string, resourceTypeStr string) (ComputedResource, error) {
	resourceType := resource.TypeFromString(resourceTypeStr)

	switch resourceType {
	case resource.PythonPackageType:
		res := ctx.PythonPackages[name]
		if res == nil {
			return nil, resource.ErrorNotFound(name, resourceType)
		}
		return res, nil
	case resource.RawColumnType:
		res := ctx.RawColumns[name]
		if res == nil {
			return nil, resource.ErrorNotFound(name, resourceType)
		}
		return res, nil
	case resource.AggregateType:
		res := ctx.Aggregates[name]
		if res == nil {
			return nil, resource.ErrorNotFound(name, resourceType)
		}
		return res, nil
	case resource.TransformedColumnType:
		res := ctx.TransformedColumns[name]
		if res == nil {
			return nil, resource.ErrorNotFound(name, resourceType)
		}
		return res, nil
	case resource.ModelType:
		res := ctx.Models[name]
		if res == nil {
			return nil, resource.ErrorNotFound(name, resourceType)
		}
		return res, nil
	case resource.APIType:
		res := ctx.APIs[name]
		if res == nil {
			return nil, resource.ErrorNotFound(name, resourceType)
		}
		return res, nil
	}

	return nil, resource.ErrorInvalidType(resourceTypeStr)
}

func (ctx *Context) Validate() error {
	return nil
}
