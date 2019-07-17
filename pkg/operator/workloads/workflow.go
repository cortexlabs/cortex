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
	"path/filepath"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	"github.com/cortexlabs/cortex/pkg/operator/api/context"
	"github.com/cortexlabs/cortex/pkg/operator/api/userconfig"
	"github.com/cortexlabs/cortex/pkg/operator/config"
)

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

	populatePythonPackageWorkloadIDs(ctx, latestResourceWorkloadIDs)
	populateSparkWorkloadIDs(ctx, latestResourceWorkloadIDs)
	populateTrainingWorkloadIDs(ctx, latestResourceWorkloadIDs)
	populateAPIWorkloadIDs(ctx, latestResourceWorkloadIDs)

	if err := ctx.CheckAllWorkloadIDsPopulated(); err != nil {
		return err
	}
	return nil
}

func extractWorkloads(ctx *context.Context) []Workload {
	var workloads []Workload
	workloads = append(workloads, extractPythonPackageWorkloads(ctx)...)
	workloads = append(workloads, extractSparkWorkloads(ctx)...)
	workloads = append(workloads, extractTrainingWorkloads(ctx)...)
	workloads = append(workloads, extractAPIWorkloads(ctx)...)
	return workloads
}

func ValidateDeploy(ctx *context.Context) error {
	if ctx.Environment != nil {
		rawDatasetExists, err := config.AWS.IsS3File(filepath.Join(ctx.RawDataset.Key, "_SUCCESS"))
		if err != nil {
			return errors.Wrap(err, ctx.App.Name, "raw dataset")
		}
		if !rawDatasetExists {
			externalPath := ctx.Environment.Data.GetPath()
			externalDataExists, err := aws.IsS3aPathPrefixExternal(externalPath)
			if !externalDataExists || err != nil {
				return errors.Wrap(userconfig.ErrorExternalNotFound(externalPath), ctx.App.Name, userconfig.Identify(ctx.Environment), userconfig.DataKey, userconfig.PathKey)
			}
		}
	}

	return nil
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
	sparkApps, _ := config.Spark.ListByLabel("appName", ctx.App.Name)
	for _, sparkApp := range sparkApps {
		config.Spark.Delete(sparkApp.Name)
	}

	err := updateKilledDataSavedStatuses(ctx)
	if err != nil {
		return err
	}

	return nil
}

func DeleteApp(appName string, keepCache bool) bool {
	deployments, _ := config.Kubernetes.ListDeploymentsByLabel("appName", appName)
	for _, deployment := range deployments {
		config.Kubernetes.DeleteDeployment(deployment.Name)
	}
	ingresses, _ := config.Kubernetes.ListIngressesByLabel("appName", appName)
	for _, ingress := range ingresses {
		config.Kubernetes.DeleteIngress(ingress.Name)
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
	sparkApps, _ := config.Spark.ListByLabel("appName", appName)
	for _, sparkApp := range sparkApps {
		config.Spark.Delete(sparkApp.Name)
	}
	pods, _ := config.Kubernetes.ListPodsByLabel("appName", appName)
	for _, pod := range pods {
		config.Kubernetes.DeletePod(pod.Name)
	}

	wasDeployed := false
	if ctx := CurrentContext(appName); ctx != nil {
		updateKilledDataSavedStatuses(ctx)
		wasDeployed = true
	}

	deleteCurrentContext(appName)
	uncacheDataSavedStatuses(nil, appName)
	uncacheLatestWorkloadIDs(nil, appName)

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

		isRunning, err := workload.IsRunning(ctx)
		if err != nil {
			return err
		}
		if isRunning {
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

func IsWorkloadPending(appName string, workloadID string) (bool, error) {
	ctx := CurrentContext(appName)
	if ctx == nil {
		return false, nil
	}

	for _, workload := range extractWorkloads(ctx) {
		if workload.GetWorkloadID() != workloadID {
			continue
		}

		isSucceeded, err := workload.IsSucceeded(ctx)
		if err != nil {
			return false, err
		}
		if isSucceeded {
			continue
		}

		isRunning, err := workload.IsRunning(ctx)
		if err != nil {
			return false, err
		}
		if isRunning {
			continue
		}

		return true, nil
	}

	return false, nil
}

func IsDeploymentUpdating(appName string) (bool, error) {
	ctx := CurrentContext(appName)
	if ctx == nil {
		return false, nil
	}

	for _, workload := range extractWorkloads(ctx) {
		isRunning, err := workload.IsRunning(ctx)
		if err != nil {
			return false, err
		}
		if isRunning {
			return true, nil
		}
	}

	return false, nil
}
