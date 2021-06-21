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

package asyncapi

import (
	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/workloads"
	"istio.io/client-go/pkg/apis/networking/v1beta1"
	kapps "k8s.io/api/apps/v1"
	kautoscaling "k8s.io/api/autoscaling/v2beta2"
	kcore "k8s.io/api/core/v1"
)

var _terminationGracePeriodSeconds int64 = 60  // seconds
var _gatewayHPATargetCPUUtilization int32 = 80 // percentage
var _gatewayHPATargetMemUtilization int32 = 80 // percentage

func gatewayDeploymentSpec(api spec.API, queueURL string) kapps.Deployment {
	volumeMounts := []kcore.VolumeMount{
		{
			Name:      "cluster-config",
			MountPath: "/configs/cluster",
		},
	}
	volumes := []kcore.Volume{
		{
			Name: "cluster-config",
			VolumeSource: kcore.VolumeSource{
				ConfigMap: &kcore.ConfigMapVolumeSource{
					LocalObjectReference: kcore.LocalObjectReference{
						Name: "cluster-config",
					},
				},
			},
		},
	}
	container := workloads.AsyncGatewayContainer(api, queueURL, volumeMounts)

	return *k8s.Deployment(&k8s.DeploymentSpec{
		Name:           getGatewayK8sName(api.Name),
		Replicas:       1,
		MaxSurge:       pointer.String(api.UpdateStrategy.MaxSurge),
		MaxUnavailable: pointer.String(api.UpdateStrategy.MaxUnavailable),
		Selector: map[string]string{
			"apiName":          api.Name,
			"apiKind":          api.Kind.String(),
			"cortex.dev/async": "gateway",
		},
		Labels: map[string]string{
			"apiName":          api.Name,
			"apiKind":          api.Kind.String(),
			"cortex.dev/api":   "true",
			"cortex.dev/async": "gateway",
		},
		PodSpec: k8s.PodSpec{
			Labels: map[string]string{
				// ID labels are omitted to avoid restarting the gateway on update/refresh
				"apiName":          api.Name,
				"apiKind":          api.Kind.String(),
				"cortex.dev/api":   "true",
				"cortex.dev/async": "gateway",
			},
			K8sPodSpec: kcore.PodSpec{
				RestartPolicy:                 "Always",
				TerminationGracePeriodSeconds: pointer.Int64(_terminationGracePeriodSeconds),
				Containers:                    []kcore.Container{container},
				NodeSelector:                  workloads.NodeSelectors(),
				Tolerations:                   workloads.GenerateResourceTolerations(),
				Affinity:                      workloads.GenerateNodeAffinities(api.NodeGroups),
				Volumes:                       volumes,
				ServiceAccountName:            workloads.ServiceAccountName,
			},
		},
	})
}

func gatewayHPASpec(api spec.API) (kautoscaling.HorizontalPodAutoscaler, error) {
	var maxReplicas int32 = 1
	if api.Autoscaling != nil {
		maxReplicas = api.Autoscaling.MaxReplicas
	}
	hpa, err := k8s.HPA(&k8s.HPASpec{
		DeploymentName:       getGatewayK8sName(api.Name),
		MinReplicas:          1,
		MaxReplicas:          maxReplicas,
		TargetCPUUtilization: _gatewayHPATargetCPUUtilization,
		TargetMemUtilization: _gatewayHPATargetMemUtilization,
		Labels: map[string]string{
			"apiName":          api.Name,
			"apiKind":          api.Kind.String(),
			"cortex.dev/api":   "true",
			"cortex.dev/async": "hpa",
		},
	})

	if err != nil {
		return kautoscaling.HorizontalPodAutoscaler{}, err
	}
	return *hpa, nil
}

func gatewayServiceSpec(api spec.API) kcore.Service {
	return *k8s.Service(&k8s.ServiceSpec{
		Name:        workloads.K8sName(api.Name),
		PortName:    "http",
		Port:        consts.ProxyListeningPortInt32,
		TargetPort:  consts.ProxyListeningPortInt32,
		Annotations: api.ToK8sAnnotations(),
		Labels: map[string]string{
			"apiName":          api.Name,
			"apiKind":          api.Kind.String(),
			"cortex.dev/api":   "true",
			"cortex.dev/async": "gateway",
		},
		Selector: map[string]string{
			"apiName":          api.Name,
			"apiKind":          api.Kind.String(),
			"cortex.dev/async": "gateway",
		},
	})
}

func gatewayVirtualServiceSpec(api spec.API) v1beta1.VirtualService {
	return *k8s.VirtualService(&k8s.VirtualServiceSpec{
		Name:     workloads.K8sName(api.Name),
		Gateways: []string{"apis-gateway"},
		Destinations: []k8s.Destination{{
			ServiceName: workloads.K8sName(api.Name),
			Weight:      100,
			Port:        uint32(consts.ProxyListeningPortInt32),
		}},
		PrefixPath:  api.Networking.Endpoint,
		Rewrite:     pointer.String("/"),
		Annotations: api.ToK8sAnnotations(),
		Labels: map[string]string{
			"apiName":               api.Name,
			"apiKind":               api.Kind.String(),
			"apiID":                 api.ID,
			"specID":                api.SpecID,
			"initialDeploymentTime": s.Int64(api.InitialDeploymentTime),
			"deploymentID":          api.DeploymentID,
			"podID":                 api.PodID,
			"cortex.dev/api":        "true",
			"cortex.dev/async":      "gateway",
		},
	})
}

func configMapSpec(api spec.API) (kcore.ConfigMap, error) {
	configMapConfig := workloads.ConfigMapConfig{
		Probes: workloads.GetReadinessProbesFromContainers(api.Pod.Containers),
	}

	configMapData, err := configMapConfig.GenerateConfigMapData()
	if err != nil {
		return kcore.ConfigMap{}, err
	}

	return *k8s.ConfigMap(&k8s.ConfigMapSpec{
		Name: workloads.K8sName(api.Name),
		Data: configMapData,
		Labels: map[string]string{
			"apiName":        api.Name,
			"apiKind":        api.Kind.String(),
			"cortex.dev/api": "true",
		},
	}), nil
}

func deploymentSpec(api spec.API, prevDeployment *kapps.Deployment, queueURL string) kapps.Deployment {
	var (
		containers []kcore.Container
		volumes    []kcore.Volume
	)

	containers, volumes = workloads.AsyncContainers(api, queueURL)

	return *k8s.Deployment(&k8s.DeploymentSpec{
		Name:           workloads.K8sName(api.Name),
		Replicas:       getRequestedReplicasFromDeployment(api, prevDeployment),
		MaxSurge:       pointer.String(api.UpdateStrategy.MaxSurge),
		MaxUnavailable: pointer.String(api.UpdateStrategy.MaxUnavailable),
		Labels: map[string]string{
			"apiName":               api.Name,
			"apiKind":               api.Kind.String(),
			"apiID":                 api.ID,
			"specID":                api.SpecID,
			"initialDeploymentTime": s.Int64(api.InitialDeploymentTime),
			"deploymentID":          api.DeploymentID,
			"podID":                 api.PodID,
			"cortex.dev/api":        "true",
			"cortex.dev/async":      "api",
		},
		Annotations: api.ToK8sAnnotations(),
		Selector: map[string]string{
			"apiName":          api.Name,
			"apiKind":          api.Kind.String(),
			"cortex.dev/async": "api",
		},
		PodSpec: k8s.PodSpec{
			Labels: map[string]string{
				"apiName":               api.Name,
				"apiKind":               api.Kind.String(),
				"apiID":                 api.ID,
				"initialDeploymentTime": s.Int64(api.InitialDeploymentTime),
				"deploymentID":          api.DeploymentID,
				"podID":                 api.PodID,
				"cortex.dev/api":        "true",
				"cortex.dev/async":      "api",
			},
			K8sPodSpec: kcore.PodSpec{
				RestartPolicy:                 "Always",
				TerminationGracePeriodSeconds: pointer.Int64(_terminationGracePeriodSeconds),
				Containers:                    containers,
				NodeSelector:                  workloads.NodeSelectors(),
				Tolerations:                   workloads.GenerateResourceTolerations(),
				Affinity:                      workloads.GenerateNodeAffinities(api.NodeGroups),
				Volumes:                       volumes,
				ServiceAccountName:            workloads.ServiceAccountName,
			},
		},
	})
}

func getRequestedReplicasFromDeployment(api spec.API, deployment *kapps.Deployment) int32 {
	requestedReplicas := api.Autoscaling.InitReplicas

	if deployment != nil && deployment.Spec.Replicas != nil {
		requestedReplicas = *deployment.Spec.Replicas
	}

	if requestedReplicas < api.Autoscaling.MinReplicas {
		requestedReplicas = api.Autoscaling.MinReplicas
	}

	if requestedReplicas > api.Autoscaling.MaxReplicas {
		requestedReplicas = api.Autoscaling.MaxReplicas
	}

	return requestedReplicas
}
