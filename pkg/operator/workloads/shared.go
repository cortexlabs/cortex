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
	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/random"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	"github.com/cortexlabs/cortex/pkg/operator/api/context"
	"github.com/cortexlabs/cortex/pkg/operator/api/resource"
	kcore "k8s.io/api/core/v1"
)

// k8s needs all characters to be lower case, and the first to be a letter
func generateWorkloadID() string {
	return random.LowercaseLetters(1) + random.LowercaseString(19)
}

// Check if all resourceIDs have succeeded (only data resource types)
func areAllDataResourcesSucceeded(ctx *context.Context, resourceIDs strset.Set) (bool, error) {
	resourceWorkloadIDs := ctx.DataResourceWorkloadIDs()
	for resourceID := range resourceIDs {
		workloadID := resourceWorkloadIDs[resourceID]
		if workloadID == "" {
			continue
		}

		savedStatus, err := getDataSavedStatus(resourceID, workloadID, ctx.App.Name)
		if err != nil {
			return false, err
		}

		if savedStatus == nil || savedStatus.ExitCode != resource.ExitCodeDataSucceeded {
			return false, nil
		}
	}

	return true, nil
}

// Check if any resourceIDs have succeeded (only data resource types)
func areAnyDataResourcesFailed(ctx *context.Context, resourceIDs strset.Set) (bool, error) {
	resourceWorkloadIDs := ctx.DataResourceWorkloadIDs()
	for resourceID := range resourceIDs {
		workloadID := resourceWorkloadIDs[resourceID]
		if workloadID == "" {
			continue
		}

		savedStatus, err := getDataSavedStatus(resourceID, workloadID, ctx.App.Name)
		if err != nil {
			return false, err
		}

		if savedStatus != nil && savedStatus.ExitCode != resource.ExitCodeDataSucceeded && savedStatus.ExitCode != resource.ExitCodeDataUnknown {
			return true, nil
		}
	}

	return false, nil
}

// Check if all dependencies of targetResourceIDs have succeeded (only data resource types)
func areAllDataDependenciesSucceeded(ctx *context.Context, targetResourceIDs strset.Set) (bool, error) {
	dependencies := ctx.DirectComputedResourceDependencies(targetResourceIDs.Slice()...)
	return areAllDataResourcesSucceeded(ctx, dependencies)
}

func baseEnvVars() []kcore.EnvFromSource {
	return []kcore.EnvFromSource{
		{
			ConfigMapRef: &kcore.ConfigMapEnvSource{
				LocalObjectReference: kcore.LocalObjectReference{
					Name: "env-vars",
				},
			},
		},
		{
			SecretRef: &kcore.SecretEnvSource{
				LocalObjectReference: kcore.LocalObjectReference{
					Name: "aws-credentials",
				},
			},
		},
	}
}

func defaultVolumes() []kcore.Volume {
	return []kcore.Volume{
		k8s.EmptyDirVolume(consts.EmptyDirVolumeName),
	}
}

func defaultVolumeMounts() []kcore.VolumeMount {
	return []kcore.VolumeMount{
		k8s.EmptyDirVolumeMount(consts.EmptyDirVolumeName, consts.EmptyDirMountPath),
	}
}
