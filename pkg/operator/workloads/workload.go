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

package workloads

import (
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	"github.com/cortexlabs/cortex/pkg/operator/api/context"
)

const (
	workloadTypeAPI            = "api"
	workloadTypeHPA            = "hpa"
	workloadTypeSpark          = "spark-job"
	workloadTypeTrain          = "training-job"
	workloadTypePythonPackager = "python-packager"
)

type Workload interface {
	BaseWorkloadInterface
	CanRun(*context.Context) (bool, error)      // All of the dependencies are satisfied and the workload can be started
	Start(*context.Context) error               // Start the workload
	IsStarted(*context.Context) (bool, error)   // The workload was started on the most recent deploy (might be running, succeeded, or failed). It's ok if this doesn't remain accurate across cx deploys
	IsRunning(*context.Context) (bool, error)   // The workload is currently running
	IsSucceeded(*context.Context) (bool, error) // The workload succeeded
	IsFailed(*context.Context) (bool, error)    // The workload failed
}

type BaseWorkload struct {
	AppName      string
	WorkloadID   string
	WorkloadType string
	Resources    map[string]context.ResourceFields
}

type BaseWorkloadInterface interface {
	GetAppName() string
	GetWorkloadID() string
	GetWorkloadType() string
	GetResources() map[string]context.ResourceFields
	CreatesResource(resourceID string) bool
	AddResource(res context.ComputedResource)
	GetResourceIDs() strset.Set
	GetSingleResourceID() string
	GetBaseWorkloadPtr() *BaseWorkload
}

func (bw *BaseWorkload) GetBaseWorkloadPtr() *BaseWorkload {
	return bw
}

func (bw *BaseWorkload) GetAppName() string {
	return bw.AppName
}

func (bw *BaseWorkload) GetWorkloadID() string {
	return bw.WorkloadID
}

func (bw *BaseWorkload) GetWorkloadType() string {
	return bw.WorkloadType
}

func (bw *BaseWorkload) GetResources() map[string]context.ResourceFields {
	if bw.Resources == nil {
		bw.Resources = make(map[string]context.ResourceFields)
	}
	return bw.Resources
}

func (bw *BaseWorkload) GetResourceIDs() strset.Set {
	resourceIDs := strset.NewWithSize(len(bw.Resources))
	for resourceID := range bw.Resources {
		resourceIDs.Add(resourceID)
	}
	return resourceIDs
}

func (bw *BaseWorkload) GetSingleResourceID() string {
	for resourceID := range bw.Resources {
		return resourceID
	}
	return ""
}

func (bw *BaseWorkload) CreatesResource(resourceID string) bool {
	if bw.Resources == nil {
		bw.Resources = make(map[string]context.ResourceFields)
	}
	_, ok := bw.Resources[resourceID]
	return ok
}

func (bw *BaseWorkload) AddResource(res context.ComputedResource) {
	if bw.Resources == nil {
		bw.Resources = make(map[string]context.ResourceFields)
	}
	bw.Resources[res.GetID()] = context.ResourceFields{
		ID:           res.GetID(),
		ResourceType: res.GetResourceType(),
	}
}

func (bw *BaseWorkload) Copy() *BaseWorkload {
	if bw == nil {
		return nil
	}

	copiedResources := make(map[string]context.ResourceFields, len(bw.Resources))
	for resID, res := range bw.Resources {
		copiedResources[resID] = res
	}

	return &BaseWorkload{
		AppName:      bw.AppName,
		WorkloadID:   bw.WorkloadID,
		WorkloadType: bw.WorkloadType,
		Resources:    copiedResources,
	}
}

func BaseWorkloadPtrsEqual(bw1 *BaseWorkload, bw2 *BaseWorkload) bool {
	if bw1 == nil && bw2 == nil {
		return true
	}
	if bw1 == nil || bw2 == nil {
		return false
	}
	return bw1.Equal(*bw2)
}

func (bw *BaseWorkload) Equal(bw2 BaseWorkload) bool {
	if bw.AppName != bw2.AppName {
		return false
	}
	if bw.WorkloadID != bw2.WorkloadID {
		return false
	}
	if bw.WorkloadType != bw2.WorkloadType {
		return false
	}
	if len(bw.Resources) != len(bw2.Resources) {
		return false
	}
	for resID, res := range bw.Resources {
		res2, ok := bw2.Resources[resID]
		if !ok {
			return false
		}
		if res.ID != res2.ID {
			return false
		}
		if res.ResourceType != res2.ResourceType {
			return false
		}
	}
	return true
}

func emptyBaseWorkload(appName string, workloadID string, workloadType string) BaseWorkload {
	return BaseWorkload{
		AppName:      appName,
		WorkloadID:   workloadID,
		WorkloadType: workloadType,
		Resources:    make(map[string]context.ResourceFields),
	}
}

func singleBaseWorkload(res context.ComputedResource, appName string, workloadType string) BaseWorkload {
	bw := emptyBaseWorkload(appName, res.GetWorkloadID(), workloadType)
	bw.AddResource(res)
	return bw
}
