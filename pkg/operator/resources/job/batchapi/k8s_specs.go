/*
Copyright 2021 Cortex Labs, Inc.

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
	"context"
	"path"

	batch "github.com/cortexlabs/cortex/pkg/crds/apis/batch/v1alpha1"
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
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const _operatorService = "operator"

func k8sJobSpec(api *spec.API, job *spec.BatchJob) (*kbatch.Job, error) {
	switch api.Handler.Type {
	case userconfig.TensorFlowHandlerType:
		return tensorFlowHandlerJobSpec(api, job)
	case userconfig.PythonHandlerType:
		return pythonHandlerJobSpec(api, job)
	default:
		return nil, nil // unexpected
	}
}

func pythonHandlerJobSpec(api *spec.API, job *spec.BatchJob) (*kbatch.Job, error) {
	containers, volumes := operator.PythonHandlerContainers(api)
	for i, container := range containers {
		if container.Name == operator.APIContainerName {
			containers[i].Env = append(container.Env, kcore.EnvVar{
				Name:  "CORTEX_JOB_SPEC",
				Value: operator.BatchSpecPath,
			})
		}
	}

	return k8s.Job(&k8s.JobSpec{
		Name:        job.JobKey.K8sName(),
		Parallelism: int32(job.Workers),
		Labels: map[string]string{
			"apiName":        api.Name,
			"apiID":          api.ID,
			"specID":         api.SpecID,
			"handlerID":      api.HandlerID,
			"jobID":          job.ID,
			"apiKind":        api.Kind.String(),
			"cortex.dev/api": "true",
		},
		PodSpec: k8s.PodSpec{
			Labels: map[string]string{
				"apiName":        api.Name,
				"handlerID":      api.HandlerID,
				"jobID":          job.ID,
				"apiKind":        api.Kind.String(),
				"cortex.dev/api": "true",
			},
			Annotations: map[string]string{
				"traffic.sidecar.istio.io/excludeOutboundIPRanges": "0.0.0.0/0",
				"cluster-autoscaler.kubernetes.io/safe-to-evict":   "false",
			},
			K8sPodSpec: kcore.PodSpec{
				RestartPolicy: "Never",
				InitContainers: []kcore.Container{
					operator.BatchInitContainer(api, job),
				},
				Containers:         containers,
				NodeSelector:       operator.NodeSelectors(),
				Tolerations:        operator.GenerateResourceTolerations(),
				Affinity:           operator.GenerateNodeAffinities(api.Compute.NodeGroups),
				Volumes:            volumes,
				ServiceAccountName: operator.ServiceAccountName,
			},
		},
	}), nil
}

func tensorFlowHandlerJobSpec(api *spec.API, job *spec.BatchJob) (*kbatch.Job, error) {
	containers, volumes := operator.TensorFlowHandlerContainers(api)
	for i, container := range containers {
		if container.Name == operator.APIContainerName {
			containers[i].Env = append(container.Env, kcore.EnvVar{
				Name:  "CORTEX_JOB_SPEC",
				Value: operator.BatchSpecPath,
			})
		}
	}

	return k8s.Job(&k8s.JobSpec{
		Name:        job.JobKey.K8sName(),
		Parallelism: int32(job.Workers),
		Labels: map[string]string{
			"apiName":        api.Name,
			"apiID":          api.ID,
			"specID":         api.SpecID,
			"handlerID":      api.HandlerID,
			"jobID":          job.ID,
			"apiKind":        api.Kind.String(),
			"cortex.dev/api": "true",
		},
		PodSpec: k8s.PodSpec{
			Labels: map[string]string{
				"apiName":        api.Name,
				"handlerID":      api.HandlerID,
				"jobID":          job.ID,
				"apiKind":        api.Kind.String(),
				"cortex.dev/api": "true",
			},
			Annotations: map[string]string{
				"traffic.sidecar.istio.io/excludeOutboundIPRanges": "0.0.0.0/0",
				"cluster-autoscaler.kubernetes.io/safe-to-evict":   "false",
			},
			K8sPodSpec: kcore.PodSpec{
				RestartPolicy: "Never",
				InitContainers: []kcore.Container{
					operator.BatchInitContainer(api, job),
				},
				Containers:         containers,
				NodeSelector:       operator.NodeSelectors(),
				Tolerations:        operator.GenerateResourceTolerations(),
				Affinity:           operator.GenerateNodeAffinities(api.Compute.NodeGroups),
				Volumes:            volumes,
				ServiceAccountName: operator.ServiceAccountName,
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
			"apiName":        api.Name,
			"apiID":          api.ID,
			"specID":         api.SpecID,
			"handlerID":      api.HandlerID,
			"apiKind":        api.Kind.String(),
			"cortex.dev/api": "true",
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

func deleteK8sResources(apiName string) error {
	return parallel.RunFirstErr(
		func() error {
			err := config.K8s.DeleteAllOf(
				context.Background(),
				&batch.BatchJob{},
				client.InNamespace(config.K8s.Namespace),
				client.MatchingLabels{"apiName": apiName},
			)
			return client.IgnoreNotFound(err)
		},
		func() error {
			_, err := config.K8s.DeleteVirtualService(operator.K8sName(apiName))
			return err
		},
	)
}
