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
	"fmt"
	"path/filepath"

	kresource "k8s.io/apimachinery/pkg/api/resource"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	"github.com/cortexlabs/cortex/pkg/operator/api/context"
	"github.com/cortexlabs/cortex/pkg/operator/api/resource"
	"github.com/cortexlabs/cortex/pkg/operator/api/userconfig"
	"github.com/cortexlabs/cortex/pkg/operator/config"
)

var cortexCPUReserve = kresource.MustParse("800m")   // FluentD (200), Nvidia (50), StatsD (100), Kube Procy, (100) Node capacity - Node availability 300 CPU
var cortexMemReserve = kresource.MustParse("1500Mi") // FluentD (200), Nvidia (50), StatsD (100), KubeReserved (800), AWS node memory - Node capacity (200)

func Init() error {
	err := reloadCurrentContexts()
	if err != nil {
		return errors.Wrap(err, "init")
	}

	go cronRunner()

	return nil
}

func PopulateWorkloadIDs(ctx *context.Context) error {
	resourceIDs := ctx.ComputedResourceIDs()
	latestResourceWorkloadIDs, err := getSavedLatestWorkloadIDs(resourceIDs, ctx.App.Name)
	if err != nil {
		return err
	}

	populateAPIWorkloadIDs(ctx, latestResourceWorkloadIDs)

	if err := ctx.CheckAllWorkloadIDsPopulated(); err != nil {
		return err
	}
	return nil
}

func extractWorkloads(ctx *context.Context) []Workload {
	var workloads []Workload
	workloads = append(workloads, extractAPIWorkloads(ctx)...)
	workloads = append(workloads, extractHPAWorkloads(ctx)...)
	return workloads
}

func Run(ctx *context.Context) error {
	if err := ctx.CheckAllWorkloadIDsPopulated(); err != nil {
		return err
	}

	prevCtx := CurrentContext(ctx.App.Name)
	err := deleteOldDataJobs(prevCtx)
	if err != nil {
		return err
	}

	deleteOldAPIs(ctx)

	err = setCurrentContext(ctx)
	if err != nil {
		return err
	}

	resourceWorkloadIDs := ctx.ComputedResourceResourceWorkloadIDs()
	err = uploadLatestWorkloadIDs(resourceWorkloadIDs, ctx.App.Name)
	if err != nil {
		return err
	}

	uncacheDataSavedStatuses(resourceWorkloadIDs, ctx.App.Name)
	uncacheLatestWorkloadIDs(ctx.ComputedResourceIDs(), ctx.App.Name)

	runCronNow()

	return nil
}

func deleteOldDataJobs(ctx *context.Context) error {
	if ctx == nil {
		return nil
	}

	jobs, _ := config.Kubernetes.ListJobsByLabel("appName", ctx.App.Name)
	for _, job := range jobs {
		config.Kubernetes.DeleteJob(job.Name)
	}

	err := updateKilledDataSavedStatuses(ctx)
	if err != nil {
		return err
	}

	return nil
}

func DeleteApp(appName string, keepCache bool) bool {
	wasDeployed := false
	if ctx := CurrentContext(appName); ctx != nil {
		updateKilledDataSavedStatuses(ctx)
		wasDeployed = true
	}

	deleteCurrentContext(appName)
	uncacheDataSavedStatuses(nil, appName)
	uncacheLatestWorkloadIDs(nil, appName)

	virtualServices, _ := config.Kubernetes.ListVirtualServicesByLabel(consts.K8sNamespace, "appName", appName)
	for _, virtualService := range virtualServices {
		config.Kubernetes.DeleteVirtualService(virtualService.GetName(), consts.K8sNamespace)
	}
	services, _ := config.Kubernetes.ListServicesByLabel("appName", appName)
	for _, service := range services {
		config.Kubernetes.DeleteService(service.Name)
	}
	hpas, _ := config.Kubernetes.ListHPAsByLabel("appName", appName)
	for _, hpa := range hpas {
		config.Kubernetes.DeleteHPA(hpa.Name)
	}
	jobs, _ := config.Kubernetes.ListJobsByLabel("appName", appName)
	for _, job := range jobs {
		config.Kubernetes.DeleteJob(job.Name)
	}
	deployments, _ := config.Kubernetes.ListDeploymentsByLabel("appName", appName)
	for _, deployment := range deployments {
		config.Kubernetes.DeleteDeployment(deployment.Name)
	}

	if !keepCache {
		config.AWS.DeleteFromS3ByPrefix(filepath.Join(consts.AppsDir, appName), true)
	}

	return wasDeployed
}

func UpdateWorkflows() error {
	currentWorkloadIDs := make(map[string]strset.Set)

	for _, ctx := range CurrentContexts() {
		err := updateWorkflow(ctx)
		if err != nil {
			return err
		}

		currentWorkloadIDs[ctx.App.Name] = ctx.ComputedResourceWorkloadIDs()
	}

	uncacheBaseWorkloads(currentWorkloadIDs)

	return nil
}

func updateWorkflow(ctx *context.Context) error {
	workloads := extractWorkloads(ctx)

	err := uploadBaseWorkloadsFromWorkloads(workloads)
	if err != nil {
		return err
	}

	for _, workload := range workloads {
		isSucceeded, err := workload.IsSucceeded(ctx)
		if err != nil {
			return err
		}
		if isSucceeded {
			continue
		}

		isFailed, err := workload.IsFailed(ctx)
		if err != nil {
			return err
		}
		if isFailed {
			continue
		}

		isStarted, err := workload.IsStarted(ctx)
		if err != nil {
			return err
		}
		if isStarted {
			continue
		}

		canRun, err := workload.CanRun(ctx)
		if err != nil {
			return err
		}
		if !canRun {
			continue
		}

		err = workload.Start(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

func IsWorkloadEnded(appName string, workloadID string) (bool, error) {
	ctx := CurrentContext(appName)
	if ctx == nil {
		return false, nil
	}

	for _, workload := range extractWorkloads(ctx) {
		if workload.GetWorkloadID() == workloadID {
			isSucceeded, err := workload.IsSucceeded(ctx)
			if err != nil {
				return false, err
			}
			if isSucceeded {
				return true, nil
			}

			isFailed, err := workload.IsFailed(ctx)
			if err != nil {
				return false, err
			}
			if isFailed {
				return true, nil
			}

			return false, nil
		}
	}

	return false, errors.New("workload not found in the current context")
}

func GetDeploymentStatus(appName string) (resource.DeploymentStatus, error) {
	ctx := CurrentContext(appName)
	if ctx == nil {
		return resource.UnknownDeploymentStatus, nil
	}

	isUpdating := false
	for _, workload := range extractWorkloads(ctx) {

		// HPA workloads don't really count
		if workload.GetWorkloadType() == workloadTypeHPA {
			continue
		}

		isSucceeded, err := workload.IsSucceeded(ctx)
		if err != nil {
			return resource.UnknownDeploymentStatus, err
		}
		if isSucceeded {
			continue
		}

		isFailed, err := workload.IsFailed(ctx)
		if err != nil {
			return resource.UnknownDeploymentStatus, err
		}
		if isFailed {
			return resource.ErrorDeploymentStatus, nil
		}

		canRun, err := workload.CanRun(ctx)
		if err != nil {
			return resource.UnknownDeploymentStatus, err
		}
		if !canRun {
			continue
		}
		isUpdating = true
	}

	if isUpdating {
		return resource.UpdatingDeploymentStatus, nil
	}
	return resource.UpdatedDeploymentStatus, nil
}

func ValidateDeploy(ctx *context.Context) error {
	// maxCPU := config.Cortex.NodeCPU.Copy()
	// maxCPU.Sub(cortexCPUReserve)
	// maxMem := config.Cortex.NodeMem.Copy()
	// maxMem.Sub(cortexMemReserve)
	// maxGPU := config.Cortex.NodeGPU.Copy()
	if err := CheckAPIEndpointCollisions(ctx); err != nil {
		return err
	}

	nodes, err := config.Kubernetes.ListNodes(nil)
	if err != nil {
		return err
	}

	var maxCPU, maxMem kresource.Quantity
	var maxGPU int64
	for _, node := range nodes {
		curCPU := node.Status.Capacity.Cpu()
		curMem := node.Status.Capacity.Memory()

		var curGPU int64
		if GPUQuantity, ok := node.Status.Allocatable["nvidia.com/gpu"]; ok {
			curGPU, _ = GPUQuantity.AsInt64()
		}

		if curCPU != nil && maxCPU.Cmp(*curCPU) < 0 {
			maxCPU = *curCPU
		}

		if curMem != nil && maxMem.Cmp(*curMem) < 0 {
			maxMem = *curMem
		}

		if curGPU > maxGPU {
			maxGPU = curGPU
		}
	}

	for _, api := range ctx.APIs {
		if maxCPU.Cmp(api.Compute.CPU.Quantity) < 0 {
			return errors.Wrap(ErrorNoAvailableNodeComputeLimit("CPU", api.Compute.CPU.String(), maxCPU.String()), userconfig.Identify(api))
		}
		if api.Compute.Mem != nil {
			if maxMem.Cmp(api.Compute.Mem.Quantity) < 0 {
				return errors.Wrap(ErrorNoAvailableNodeComputeLimit("Memory", api.Compute.Mem.String(), maxMem.String()), userconfig.Identify(api))
			}
		}
		gpu := api.Compute.GPU
		if gpu > maxGPU.Value() {
			return errors.Wrap(ErrorNoAvailableNodeComputeLimit("GPU", fmt.Sprintf("%d", gpu), fmt.Sprintf("%d", maxGPU.Value())), userconfig.Identify(api))
		}
	}
	return nil
}

func CheckAPIEndpointCollisions(ctx *context.Context) error {
	apiEndpoints := map[string]string{} // endpoint -> API identifiction string
	for _, api := range ctx.APIs {
		apiEndpoints[*api.Endpoint] = userconfig.Identify(api)
	}

	virtualServices, err := config.Kubernetes.ListVirtualServices(consts.K8sNamespace, nil)
	if err != nil {
		return err
	}

	for _, virtualService := range virtualServices {
		gateways, err := k8s.GetVirtualServiceGateways(&virtualService)
		if err != nil {
			return err
		}
		if !gateways.Has("apis-gateway") {
			continue
		}

		// Collisions within a deployment will already have been caught by config validation
		labels := virtualService.GetLabels()
		if labels["appName"] == ctx.App.Name {
			continue
		}

		endpoints, err := k8s.GetVirtualServiceEndpoints(&virtualService)
		if err != nil {
			return err
		}

		for endpoint := range endpoints {
			if apiIdentifier, ok := apiEndpoints[endpoint]; ok {
				return errors.Wrap(ErrorDuplicateEndpointOtherDeployment(labels["appName"], labels["apiName"]), apiIdentifier, userconfig.EndpointKey, endpoint)
			}
		}
	}

	return nil
}
