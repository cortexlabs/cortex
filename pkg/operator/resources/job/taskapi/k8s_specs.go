/*
Copyright 2020 Cortex Labs, Inc.

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

package taskapi

import (
	"path"

	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/parallel"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/operator/operator"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	istioclientnetworking "istio.io/client-go/pkg/apis/networking/v1beta1"
	kbatch "k8s.io/api/batch/v1"
	kcore "k8s.io/api/core/v1"
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
)

const _operatorService = "operator"

func virtualServiceSpec(api *spec.API) *istioclientnetworking.VirtualService {
	return k8s.VirtualService(&k8s.VirtualServiceSpec{
		Name:     operator.K8sName(api.Name),
		Gateways: []string{"apis-gateway"},
		Destinations: []k8s.Destination{{
			ServiceName: _operatorService,
			Weight:      100,
			Port:        uint32(operator.DefaultPortInt32),
		}},
		PrefixPath:  api.Networking.Endpoint,
		Rewrite:     pointer.String(path.Join("tasks", api.Name)),
		Annotations: api.ToK8sAnnotations(),
		Labels: map[string]string{
			"apiName":     api.Name,
			"apiID":       api.ID,
			"specID":      api.SpecID,
			"predictorID": api.PredictorID,
			"apiKind":     api.Kind.String(),
		},
	})
}

func k8sJobSpec(api *spec.API, job *spec.TaskJob) (*kbatch.Job, error) {
	containers, volumes := operator.PythonPredictorContainers(api)
	for i, container := range containers {
		if container.Name == operator.APIContainerName {
			containers[i].Env = append(container.Env, kcore.EnvVar{
				Name:  "CORTEX_TASK_SPEC",
				Value: "s3://" + config.Cluster.Bucket + "/" + job.SpecFilePath(config.Cluster.ClusterName),
			})
		}
	}

	return k8s.Job(&k8s.JobSpec{
		Name:        job.JobKey.K8sName(),
		Parallelism: int32(job.Workers),
		Labels: map[string]string{
			"apiName":     api.Name,
			"apiID":       api.ID,
			"specID":      api.SpecID,
			"predictorID": api.PredictorID,
			"jobID":       job.ID,
			"apiKind":     api.Kind.String(),
		},
		PodSpec: k8s.PodSpec{
			Labels: map[string]string{
				"apiName":     api.Name,
				"predictorID": api.PredictorID,
				"jobID":       job.ID,
				"apiKind":     api.Kind.String(),
			},
			Annotations: map[string]string{
				"traffic.sidecar.istio.io/excludeOutboundIPRanges": "0.0.0.0/0",
			},
			K8sPodSpec: kcore.PodSpec{
				RestartPolicy: "Never",
				InitContainers: []kcore.Container{
					operator.InitContainer(api),
				},
				Containers: containers,
				NodeSelector: map[string]string{
					"workload": "true",
				},
				Tolerations:        operator.Tolerations,
				Volumes:            volumes,
				ServiceAccountName: "default",
			},
		},
	}), nil
}

func applyK8sResources(api *spec.API, prevVirtualService *istioclientnetworking.VirtualService) error {
	newVirtualService := virtualServiceSpec(api)

	if prevVirtualService == nil {
		_, err := config.K8s.CreateVirtualService(newVirtualService)
		return err
	}

	_, err := config.K8s.UpdateVirtualService(prevVirtualService, newVirtualService)
	return err
}

func deleteK8sResources(apiName string) error {
	return parallel.RunFirstErr(
		func() error {
			return deleteAllK8sJobs(apiName)
		},
		func() error {
			_, err := config.K8s.DeleteVirtualService(operator.K8sName(apiName))
			return err
		},
	)
}

func deleteK8sJob(jobKey spec.JobKey) error {
	_, err := config.K8s.DeleteJobs(&kmeta.ListOptions{
		LabelSelector: klabels.SelectorFromSet(
			map[string]string{
				"apiName": jobKey.APIName,
				"apiKind": userconfig.TaskAPIKind.String(),
				"jobID":   jobKey.ID,
			}).String(),
	})
	return err
}

func deleteAllK8sJobs(apiName string) error {
	_, err := config.K8s.DeleteJobs(&kmeta.ListOptions{
		LabelSelector: klabels.SelectorFromSet(
			map[string]string{
				"apiName": apiName,
				"apiKind": userconfig.TaskAPIKind.String(),
			}).String(),
	})
	return err
}
