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

	awfv1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/json"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/slices"
	"github.com/cortexlabs/cortex/pkg/operator/api/context"
	"github.com/cortexlabs/cortex/pkg/operator/argo"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	ocontext "github.com/cortexlabs/cortex/pkg/operator/context"
)

func Init() error {
	workflows, err := argo.List(nil)
	if err != nil {
		return errors.Wrap(err, "init", "argo", "list")
	}

	for _, wf := range workflows {
		ctx, err := ocontext.DownloadContext(wf.Labels["ctxID"], wf.Labels["appName"])
		if err != nil {
			fmt.Println("Deleting stale workflow:", wf.Name)
			argo.Delete(wf.Name)
		} else {
			setCurrentContext(ctx)
		}
	}

	return nil
}

func Create(ctx *context.Context) (*awfv1.Workflow, error) {
	err := populateLatestWorkloadIDs(ctx)
	if err != nil {
		return nil, err
	}

	labels := map[string]string{
		"appName": ctx.App.Name,
		"ctxID":   ctx.ID,
	}
	wf := argo.New(ctx.App.Name, labels)

	var allSpecs []*WorkloadSpec

	pythonPackageJobSpecs, err := pythonPackageWorkloadSpecs(ctx)
	if err != nil {
		return nil, err
	}
	allSpecs = append(allSpecs, pythonPackageJobSpecs...)

	dataJobSpecs, err := dataWorkloadSpecs(ctx)
	if err != nil {
		return nil, err
	}
	allSpecs = append(allSpecs, dataJobSpecs...)

	trainingJobSpecs, err := trainingWorkloadSpecs(ctx)
	if err != nil {
		return nil, err
	}
	allSpecs = append(allSpecs, trainingJobSpecs...)

	apiSpecs, err := apiWorkloadSpecs(ctx)
	if err != nil {
		return nil, err
	}
	allSpecs = append(allSpecs, apiSpecs...)

	resourceWorkloadIDs := make(map[string]string)
	for _, spec := range allSpecs {
		for resourceID := range spec.ResourceIDs {
			resourceWorkloadIDs[resourceID] = spec.WorkloadID
		}
	}
	ctx.PopulateWorkloadIDs(resourceWorkloadIDs)

	for _, spec := range allSpecs {
		var dependencyWorkloadIDs []string
		for resourceID := range spec.ResourceIDs {
			for dependencyResourceID := range ctx.AllComputedResourceDependencies(resourceID) {
				workloadID := resourceWorkloadIDs[dependencyResourceID]
				if workloadID != "" && workloadID != spec.WorkloadID {
					dependencyWorkloadIDs = append(dependencyWorkloadIDs, workloadID)
				}
			}
		}

		manifest, err := json.Marshal(spec.Spec)
		if err != nil {
			return nil, errors.Wrap(err, ctx.App.Name, "workloads", spec.WorkloadID)
		}

		argo.AddTask(wf, &argo.WorkflowTask{
			Name:             spec.WorkloadID,
			Action:           spec.K8sAction,
			Manifest:         string(manifest),
			SuccessCondition: spec.SuccessCondition,
			FailureCondition: spec.FailureCondition,
			Dependencies:     slices.UniqueStrings(dependencyWorkloadIDs),
			Labels: map[string]string{
				"appName":      ctx.App.Name,
				"workloadType": spec.WorkloadType,
				"workloadID":   spec.WorkloadID,
			},
		})

		err = uploadWorkloadSpec(spec, ctx)
		if err != nil {
			return nil, err
		}
	}

	return wf, nil
}

func populateLatestWorkloadIDs(ctx *context.Context) error {
	resourceIDs := ctx.ComputedResourceIDs()
	resourceWorkloadIDs, err := getSavedLatestWorkloadIDs(resourceIDs, ctx.App.Name)
	if err != nil {
		return err

	}
	ctx.PopulateWorkloadIDs(resourceWorkloadIDs)
	return nil
}

func Run(wf *awfv1.Workflow, ctx *context.Context, existingWf *awfv1.Workflow) error {
	err := ctx.CheckAllWorkloadIDsPopulated()
	if err != nil {
		return err
	}

	if existingWf != nil {
		existingCtx := CurrentContext(ctx.App.Name)
		if wf.Labels["appName"] != existingWf.Labels["appName"] {
			return ErrorWorkflowAppMismatch()
		}
		if existingCtx != nil && ctx.App.Name != existingCtx.App.Name {
			return ErrorContextAppMismatch()
		}

		err := Stop(existingWf, existingCtx)
		if err != nil {
			return err
		}
	}

	err = argo.Run(wf)
	if err != nil {
		return errors.Wrap(err, ctx.App.Name)
	}

	err = createServicesAndIngresses(ctx)
	if err != nil {
		return err
	}

	deleteOldAPIs(ctx)

	setCurrentContext(ctx)

	resourceWorkloadIDs := ctx.ComputedResourceResourceWorkloadIDs()
	err = uploadLatestWorkloadIDs(resourceWorkloadIDs, ctx.App.Name)
	if err != nil {
		return err
	}

	uncacheDataSavedStatuses(resourceWorkloadIDs, ctx.App.Name)
	uncacheLatestWorkloadIDs(ctx.ComputedResourceIDs(), ctx.App.Name)

	return nil
}

func Stop(wf *awfv1.Workflow, ctx *context.Context) error {
	if wf == nil {
		return nil
	}

	_, err := argo.Delete(wf.Name)
	if err != nil {
		return errors.Wrap(err, ctx.App.Name)
	}

	err = updateKilledDataSavedStatuses(ctx)
	if err != nil {
		return err
	}

	return nil
}

func DeleteApp(appName string, keepCache bool) bool {
	ctx := CurrentContext(appName)
	wasDeployed := false

	if ctx != nil {
		wf, _ := GetWorkflow(appName)
		Stop(wf, ctx)
		wasDeployed = true
	}

	deployments, _ := k8s.ListDeploymentsByLabel("appName", appName)
	for _, deployment := range deployments {
		k8s.DeleteDeployment(deployment.Name)
	}
	ingresses, _ := k8s.ListIngressesByLabel("appName", appName)
	for _, ingress := range ingresses {
		k8s.DeleteIngress(ingress.Name)
	}
	services, _ := k8s.ListServicesByLabel("appName", appName)
	for _, service := range services {
		k8s.DeleteService(service.Name)
	}
	jobs, _ := k8s.ListJobsByLabel("appName", appName)
	for _, job := range jobs {
		k8s.DeleteJob(job.Name)
	}
	pods, _ := k8s.ListPodsByLabel("appName", appName)
	for _, pod := range pods {
		k8s.DeletePod(pod.Name)
	}

	deleteCurrentContext(appName)
	uncacheDataSavedStatuses(nil, appName)
	uncacheLatestWorkloadIDs(nil, appName)

	if !keepCache {
		config.AWS.DeleteFromS3ByPrefix(config.Cortex.Bucket, filepath.Join(consts.AppsDir, appName), true)
	}

	return wasDeployed
}

func GetWorkflow(appName string) (*awfv1.Workflow, error) {
	wfs, err := argo.ListByLabel("appName", appName)
	if err != nil {
		return nil, errors.Wrap(err, appName)
	}
	if len(wfs) > 1 {
		return nil, errors.Wrap(ErrorMoreThanOneWorkflow(), appName)
	}

	if len(wfs) == 0 {
		return nil, nil
	}

	return &wfs[0], nil
}
