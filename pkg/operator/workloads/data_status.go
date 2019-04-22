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
	"github.com/cortexlabs/cortex/pkg/operator/api/resource"
	"github.com/cortexlabs/cortex/pkg/operator/config"
)

func GetCurrentDataStatuses(ctx *context.Context) (map[string]*resource.DataStatus, error) {
	dataStatuses := make(map[string]*resource.DataStatus)
	dataResourceWorkloadIDs := ctx.DataResourceWorkloadIDs()
	dataSavedStatuses, err := getDataSavedStatuses(dataResourceWorkloadIDs, ctx.App.Name)
	if err != nil {
		return nil, err
	}

	for resourceID, workloadID := range dataResourceWorkloadIDs {
		if dataSavedStatuses[resourceID] == nil {
			res := ctx.OneResourceByID(resourceID)
			dataSavedStatuses[resourceID] = &resource.DataSavedStatus{
				BaseSavedStatus: resource.BaseSavedStatus{
					ResourceID:   resourceID,
					ResourceType: res.GetResourceType(),
					WorkloadID:   workloadID,
					AppName:      ctx.App.Name,
				},
			}
		}
	}

	for resourceID, dataSavedStatus := range dataSavedStatuses {
		dataStatuses[resourceID] = &resource.DataStatus{
			DataSavedStatus: *dataSavedStatus,
			Code:            dataStatusCode(dataSavedStatus),
		}
	}

	for _, dataStatus := range dataStatuses {
		updateDataStatusCodeByParents(dataStatus, dataStatuses, ctx)
	}

	setSkippedDataStatusCodes(dataStatuses, ctx)
	setInsufficientComputeDataStatusCodes(dataStatuses, ctx)

	return dataStatuses, nil
}

func dataStatusCode(dataSavedStatus *resource.DataSavedStatus) resource.StatusCode {
	if dataSavedStatus.Start == nil {
		return resource.StatusPending
	}
	if dataSavedStatus.End == nil {
		return resource.StatusDataRunning
	}

	switch dataSavedStatus.ExitCode {
	case resource.ExitCodeDataSucceeded:
		return resource.StatusDataSucceeded
	case resource.ExitCodeDataFailed:
		return resource.StatusDataFailed
	case resource.ExitCodeDataKilled:
		return resource.StatusDataKilled
	case resource.ExitCodeDataOOM:
		return resource.StatusDataKilledOOM
	}

	return resource.StatusUnknown
}

func updateDataStatusCodeByParents(dataStatus *resource.DataStatus, dataStatuses map[string]*resource.DataStatus, ctx *context.Context) {
	if dataStatus.Code != resource.StatusPending {
		return
	}
	allDependencies := ctx.AllComputedResourceDependencies(dataStatus.ResourceID)
	numSucceeded := 0
	parentSkipped := false
	for dependency := range allDependencies {
		switch dataStatuses[dependency].Code {
		case resource.StatusDataKilled, resource.StatusDataKilledOOM:
			dataStatus.Code = resource.StatusParentKilled
			return
		case resource.StatusDataFailed:
			dataStatus.Code = resource.StatusParentFailed
			return
		case resource.StatusSkipped:
			parentSkipped = true
		case resource.StatusDataSucceeded:
			numSucceeded++
		}
	}

	if parentSkipped {
		dataStatus.Code = resource.StatusSkipped
		return
	}

	if numSucceeded == len(allDependencies) {
		dataStatus.Code = resource.StatusWaiting
	}
}

func setSkippedDataStatusCodes(dataStatuses map[string]*resource.DataStatus, ctx *context.Context) {
	if !didSparkShortCircuit(dataStatuses, ctx) {
		return
	}
	for _, dataStatus := range dataStatuses {
		if dataStatus.Code == resource.StatusPending || dataStatus.Code == resource.StatusWaiting || dataStatus.Code == resource.StatusPendingCompute || dataStatus.Code == resource.StatusUnknown {
			dataStatus.Code = resource.StatusSkipped
		}
	}
}

func sparkGeneratedResources(ctx *context.Context) map[string]context.Resource {
	resources := make(map[string]context.Resource)
	for _, rawColumn := range ctx.RawColumns {
		resources[rawColumn.GetID()] = rawColumn
	}
	for _, aggregate := range ctx.Aggregates {
		resources[aggregate.ID] = aggregate
	}
	for _, transformedColumn := range ctx.TransformedColumns {
		resources[transformedColumn.ID] = transformedColumn
	}
	for _, model := range ctx.Models {
		resources[model.Dataset.ID] = model.Dataset
	}
	return resources
}

func didSparkShortCircuit(dataStatuses map[string]*resource.DataStatus, ctx *context.Context) bool {
	sparkResources := sparkGeneratedResources(ctx)
	for _, dataStatus := range dataStatuses {
		if _, ok := sparkResources[dataStatus.ResourceID]; !ok {
			continue
		}
		if dataStatus.Code == resource.StatusDataFailed || dataStatus.Code == resource.StatusDataKilled {
			return true
		}
	}
	return false
}

func setInsufficientComputeDataStatusCodes(dataStatuses map[string]*resource.DataStatus, ctx *context.Context) error {
	stalledPods, err := config.Kubernetes.StalledPods()
	if err != nil {
		return err
	}
	stalledWorkloads := strset.New()
	for _, pod := range stalledPods {
		stalledWorkloads.Add(pod.Labels["workloadID"])
	}

	for _, dataStatus := range dataStatuses {
		if dataStatus.Code == resource.StatusPending || dataStatus.Code == resource.StatusWaiting {
			if _, ok := stalledWorkloads[dataStatus.WorkloadID]; ok {
				dataStatus.Code = resource.StatusPendingCompute
			}
		}
	}

	return nil
}
