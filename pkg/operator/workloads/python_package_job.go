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

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"

	"github.com/cortexlabs/cortex/pkg/operator/api/context"
	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	"github.com/cortexlabs/cortex/pkg/operator/argo"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
)

func pythonPackageJobSpec(ctx *context.Context, pythonPackages strset.Set, workloadID string) *batchv1.Job {
	spec := k8s.Job(&k8s.JobSpec{
		Name: workloadID,
		Labels: map[string]string{
			"appName":      ctx.App.Name,
			"workloadType": workloadTypePythonPackager,
			"workloadID":   workloadID,
		},
		PodSpec: k8s.PodSpec{
			Labels: map[string]string{
				"appName":      ctx.App.Name,
				"workloadType": workloadTypePythonPackager,
				"workloadID":   workloadID,
				"userFacing":   "true",
			},
			K8sPodSpec: corev1.PodSpec{
				RestartPolicy: "Never",
				Containers: []corev1.Container{
					{
						Name:            "python-packager",
						Image:           config.Cortex.PythonPackagerImage,
						ImagePullPolicy: "Always",
						Args: []string{
							"--workload-id=" + workloadID,
							"--context=" + aws.AWS.S3Path(config.Cortex.Bucket, ctx.Key),
							"--cache-dir=" + consts.ContextCacheDir,
							"--python-packages=" + strings.Join(pythonPackages.Slice(), ","),
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
	})
	argo.EnableGC(spec)
	return spec
}

func pythonPackageWorkloadSpecs(ctx *context.Context) ([]*WorkloadSpec, error) {
	resourceIDs := strset.New()

	for _, pythonPackage := range ctx.PythonPackages {
		isPythonPackageCached, err := checkResourceCached(pythonPackage, ctx)
		if err != nil {
			return nil, err
		}
		if isPythonPackageCached {
			continue
		}
		resourceIDs.Add(pythonPackage.GetID())
	}

	if len(resourceIDs) == 0 {
		return nil, nil
	}

	workloadID := generateWorkloadID()

	spec := pythonPackageJobSpec(ctx, resourceIDs, workloadID)
	workloadSpec := &WorkloadSpec{
		WorkloadID:       workloadID,
		ResourceIDs:      resourceIDs,
		Spec:             spec,
		K8sAction:        "create",
		SuccessCondition: k8s.JobSuccessCondition,
		FailureCondition: k8s.JobFailureCondition,
		WorkloadType:     workloadTypePythonPackager,
	}

	return []*WorkloadSpec{workloadSpec}, nil
}
