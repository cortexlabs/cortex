/*
Copyright 2022 Cortex Labs, Inc.

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

	"github.com/cortexlabs/cortex/pkg/config"
	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/parallel"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	"github.com/cortexlabs/cortex/pkg/workloads"
	istioclientnetworking "istio.io/client-go/pkg/apis/networking/v1beta1"
	kbatch "k8s.io/api/batch/v1"
	kcore "k8s.io/api/core/v1"
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	klabels "k8s.io/apimachinery/pkg/labels"
)

const _operatorService = "operator"

func virtualServiceSpec(api *spec.API) *istioclientnetworking.VirtualService {
	return k8s.VirtualService(&k8s.VirtualServiceSpec{
		Name:     workloads.K8sName(api.Name),
		Gateways: []string{"apis-gateway"},
		Destinations: []k8s.Destination{{
			ServiceName: _operatorService,
			Weight:      100,
			Port:        uint32(consts.ProxyPortInt32),
		}},
		PrefixPath:  api.Networking.Endpoint,
		Rewrite:     pointer.String(path.Join("tasks", api.Name)),
		Annotations: api.ToK8sAnnotations(),
		Labels: map[string]string{
			"apiName":               api.Name,
			"apiID":                 api.ID,
			"specID":                api.SpecID,
			"podID":                 api.PodID,
			"initialDeploymentTime": s.Int64(api.InitialDeploymentTime),
			"apiKind":               api.Kind.String(),
			"cortex.dev/api":        "true",
		},
	})
}

func k8sJobSpec(api *spec.API, job *spec.TaskJob) *kbatch.Job {
	containers, volumes := workloads.TaskContainers(*api, &job.JobKey)

	return k8s.Job(&k8s.JobSpec{
		Name:        job.JobKey.K8sName(),
		Parallelism: int32(job.Workers),
		Labels: map[string]string{
			"apiName":        api.Name,
			"apiID":          api.ID,
			"specID":         api.SpecID,
			"podID":          api.PodID,
			"jobID":          job.ID,
			"apiKind":        api.Kind.String(),
			"cortex.dev/api": "true",
		},
		PodSpec: k8s.PodSpec{
			Labels: map[string]string{
				"apiName":        api.Name,
				"podID":          api.PodID,
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
					workloads.KubexitInitContainer(),
				},
				Containers:         containers,
				NodeSelector:       workloads.NodeSelectors(),
				Tolerations:        workloads.GenerateResourceTolerations(),
				Affinity:           workloads.GenerateNodeAffinities(api.NodeGroups),
				Volumes:            volumes,
				ServiceAccountName: workloads.ServiceAccountName,
			},
		},
	})
}

func k8sConfigMap(api spec.API, job spec.TaskJob, configMapData map[string]string) kcore.ConfigMap {
	return *k8s.ConfigMap(&k8s.ConfigMapSpec{
		Name: job.JobKey.K8sName(),
		Data: configMapData,
		Labels: map[string]string{
			"apiName":        api.Name,
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
			_, err := config.K8s.DeleteJobs(&kmeta.ListOptions{
				LabelSelector: klabels.SelectorFromSet(
					map[string]string{
						"apiName": apiName,
						"apiKind": userconfig.TaskAPIKind.String(),
					}).String(),
			})
			return err
		},
		func() error {
			_, err := config.K8s.DeleteVirtualService(workloads.K8sName(apiName))
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

func createK8sJob(apiSpec *spec.API, jobSpec *spec.TaskJob) error {
	k8sJob := k8sJobSpec(apiSpec, jobSpec)

	_, err := config.K8s.CreateJob(k8sJob)
	if err != nil {
		return err
	}

	return nil
}

func deleteK8sConfigMap(jobKey spec.JobKey) error {
	_, err := config.K8s.DeleteConfigMap(jobKey.K8sName())
	return err
}

func createK8sConfigMap(configMap kcore.ConfigMap) error {
	_, err := config.K8s.CreateConfigMap(&configMap)
	return err
}
