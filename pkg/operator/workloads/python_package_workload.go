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
	"strings"

	kcore "k8s.io/api/core/v1"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/operator/api/context"
	"github.com/cortexlabs/cortex/pkg/operator/config"
)

type PythonPackagesWorkload struct {
	BaseWorkload
}

func populatePythonPackageWorkloadIDs(ctx *context.Context, latestResourceWorkloadIDs map[string]string) {
	pythonPackagesWorkloadID := generateWorkloadID()

	for _, pythonPackage := range ctx.PythonPackages {
		if pythonPackage.WorkloadID != "" {
			continue
		}
		if workloadID := latestResourceWorkloadIDs[pythonPackage.ID]; workloadID != "" {
			pythonPackage.WorkloadID = workloadID
			continue
		}
		pythonPackage.WorkloadID = pythonPackagesWorkloadID
	}
}

func extractPythonPackageWorkloads(ctx *context.Context) []Workload {
	workloadMap := make(map[string]*PythonPackagesWorkload)
	for _, pythonPackage := range ctx.PythonPackages {
		if _, ok := workloadMap[pythonPackage.WorkloadID]; !ok {
			workloadMap[pythonPackage.WorkloadID] = &PythonPackagesWorkload{
				emptyBaseWorkload(ctx.App.Name, pythonPackage.WorkloadID, workloadTypePythonPackager),
			}
		}
		workloadMap[pythonPackage.WorkloadID].AddResource(pythonPackage)
	}

	workloads := make([]Workload, 0, len(workloadMap))
	for _, workload := range workloadMap {
		workloads = append(workloads, workload)
	}
	return workloads
}

func (pyw *PythonPackagesWorkload) Start(ctx *context.Context) error {
	spec := &k8s.JobSpec{
		Name: pyw.WorkloadID,
		Labels: map[string]string{
			"appName":      ctx.App.Name,
			"workloadType": workloadTypePythonPackager,
			"workloadID":   pyw.WorkloadID,
		},
		PodSpec: k8s.PodSpec{
			Labels: map[string]string{
				"appName":      ctx.App.Name,
				"workloadType": workloadTypePythonPackager,
				"workloadID":   pyw.WorkloadID,
				"userFacing":   "true",
			},
			K8sPodSpec: kcore.PodSpec{
				RestartPolicy: "Never",
				Containers: []kcore.Container{
					{
						Name:            "python-packager",
						Image:           config.Cortex.PythonPackagerImage,
						ImagePullPolicy: "Always",
						Args: []string{
							"--workload-id=" + pyw.WorkloadID,
							"--context=" + config.AWS.S3Path(ctx.Key),
							"--cache-dir=" + consts.ContextCacheDir,
							"--python-packages=" + strings.Join(pyw.GetResourceIDs().Slice(), ","),
							"--build",
						},
						Env:          k8s.AWSCredentials(),
						VolumeMounts: k8s.DefaultVolumeMounts(),
					},
				},
				Volumes:            k8s.DefaultVolumes(),
				ServiceAccountName: "default",
			},
		},
		Namespace: config.Cortex.Namespace,
	}

	_, err := config.Kubernetes.CreateJob(k8s.Job(spec))
	if err != nil {
		return err
	}
	return nil
}

func (pyw *PythonPackagesWorkload) IsStarted(ctx *context.Context) (bool, error) {
	return config.Kubernetes.JobExists(pyw.WorkloadID)
}

func (pyw *PythonPackagesWorkload) IsRunning(ctx *context.Context) (bool, error) {
	return config.Kubernetes.IsJobRunning(pyw.WorkloadID)
}

func (pyw *PythonPackagesWorkload) CanRun(ctx *context.Context) (bool, error) {
	return areAllDataDependenciesSucceeded(ctx, pyw.GetResourceIDs())
}

func (pyw *PythonPackagesWorkload) IsSucceeded(ctx *context.Context) (bool, error) {
	return areAllDataResourcesSucceeded(ctx, pyw.GetResourceIDs())
}

func (pyw *PythonPackagesWorkload) IsFailed(ctx *context.Context) (bool, error) {
	return areAnyDataResourcesFailed(ctx, pyw.GetResourceIDs())
}
