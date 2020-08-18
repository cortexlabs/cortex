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

package batchapi

import (
	"path"

	"github.com/cortexlabs/cortex/pkg/lib/hash"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/operator/operator"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	istioclientnetworking "istio.io/client-go/pkg/apis/networking/v1alpha3"
	kbatch "k8s.io/api/batch/v1"
	kcore "k8s.io/api/core/v1"
)

const _operatorService = "operator"

func k8sJobSpec(api *spec.API, job *spec.Job) (*kbatch.Job, error) {
	switch api.Predictor.Type {
	case userconfig.TensorFlowPredictorType:
		return tensorFlowPredictorJobSpec(api, job)
	case userconfig.ONNXPredictorType:
		return onnxPredictorJobSpec(api, job)
	case userconfig.PythonPredictorType:
		return pythonPredictorJobSpec(api, job)
	default:
		return nil, nil // unexpected
	}
}

func pythonPredictorJobSpec(api *spec.API, job *spec.Job) (*kbatch.Job, error) {
	containers, volumes := operator.PythonPredictorContainers(api)
	for i, container := range containers {
		if container.Name == operator.APIContainerName {
			containers[i].Env = append(container.Env, kcore.EnvVar{
				Name:  "CORTEX_JOB_SPEC",
				Value: "s3://" + config.Cluster.Bucket + "/" + job.SpecFilePath(),
			})
		}
	}

	return k8s.Job(&k8s.JobSpec{
		Name:        job.JobKey.K8sName(),
		Parallelism: int32(job.Workers),
		Labels: map[string]string{
			"apiName": api.Name,
			"apiID":   api.ID,
			"jobID":   job.ID,
			"apiKind": api.Kind.String(),
		},
		PodSpec: k8s.PodSpec{
			Labels: map[string]string{
				"apiName": api.Name,
				"apiID":   api.ID,
				"jobID":   job.ID,
				"apiKind": api.Kind.String(),
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

func tensorFlowPredictorJobSpec(api *spec.API, job *spec.Job) (*kbatch.Job, error) {
	containers, volumes := operator.TensorFlowPredictorContainers(api)
	for i, container := range containers {
		if container.Name == operator.APIContainerName {
			containers[i].Env = append(container.Env, kcore.EnvVar{
				Name:  "CORTEX_JOB_SPEC",
				Value: "s3://" + config.Cluster.Bucket + "/" + job.SpecFilePath(),
			})
		}
	}

	return k8s.Job(&k8s.JobSpec{
		Name:        job.JobKey.K8sName(),
		Parallelism: int32(job.Workers),
		Labels: map[string]string{
			"apiName": api.Name,
			"apiID":   api.ID,
			"jobID":   job.ID,
			"apiKind": api.Kind.String(),
		},
		PodSpec: k8s.PodSpec{
			Labels: map[string]string{
				"apiName": api.Name,
				"apiID":   api.ID,
				"jobID":   job.ID,
				"apiKind": api.Kind.String(),
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

func onnxPredictorJobSpec(api *spec.API, job *spec.Job) (*kbatch.Job, error) {
	containers := operator.ONNXPredictorContainers(api)

	for i, container := range containers {
		if container.Name == operator.APIContainerName {
			containers[i].Env = append(container.Env, kcore.EnvVar{
				Name:  "CORTEX_JOB_SPEC",
				Value: "s3://" + config.Cluster.Bucket + "/" + job.SpecFilePath(),
			})
		}
	}

	return k8s.Job(&k8s.JobSpec{
		Name:        job.JobKey.K8sName(),
		Parallelism: int32(job.Workers),
		Labels: map[string]string{
			"apiName": api.Name,
			"apiID":   api.ID,
			"jobID":   job.ID,
			"apiKind": api.Kind.String(),
		},
		PodSpec: k8s.PodSpec{
			Labels: map[string]string{
				"apiName": api.Name,
				"apiID":   api.ID,
				"jobID":   job.ID,
				"apiKind": api.Kind.String(),
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
				Volumes:            operator.DefaultVolumes,
				ServiceAccountName: "default",
			},
		},
	}), nil
}

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
		Rewrite:     pointer.String(path.Join("batch", api.Name)),
		Annotations: api.ToK8sAnnotations(),
		Labels: map[string]string{
			"apiName":   api.Name,
			"apiID":     api.ID,
			"apiKind":   api.Kind.String(),
			"computeID": hash.String(api.Compute.UserStr()), // Including computeID to determine updating
		},
	})
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
