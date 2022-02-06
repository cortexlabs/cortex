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

package realtimeapi

import (
	"fmt"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/workloads"
	istionetworking "istio.io/api/networking/v1beta1"
	istioclientnetworking "istio.io/client-go/pkg/apis/networking/v1beta1"
	kapps "k8s.io/api/apps/v1"
	kcore "k8s.io/api/core/v1"
)

var _terminationGracePeriodSeconds int64 = 60 // seconds

func deploymentSpec(api *spec.API, prevDeployment *kapps.Deployment) *kapps.Deployment {
	containers, volumes := workloads.RealtimeContainers(*api)

	return k8s.Deployment(&k8s.DeploymentSpec{
		Name:           workloads.K8sName(api.Name),
		Replicas:       getRequestedReplicasFromDeployment(*api, prevDeployment),
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
		},
		Annotations: api.ToK8sAnnotations(),
		Selector: map[string]string{
			"apiName": api.Name,
			"apiKind": api.Kind.String(),
		},
		PodSpec: k8s.PodSpec{
			Labels: map[string]string{
				"apiName":               api.Name,
				"apiKind":               api.Kind.String(),
				"initialDeploymentTime": s.Int64(api.InitialDeploymentTime),
				"deploymentID":          api.DeploymentID,
				"podID":                 api.PodID,
				"cortex.dev/api":        "true",
			},
			Annotations: map[string]string{
				"traffic.sidecar.istio.io/excludeOutboundIPRanges": "0.0.0.0/0",
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

func serviceSpec(api *spec.API) *kcore.Service {
	return k8s.Service(&k8s.ServiceSpec{
		Name:        workloads.K8sName(api.Name),
		PortName:    "http",
		Port:        consts.ProxyPortInt32,
		TargetPort:  consts.ProxyPortInt32,
		Annotations: api.ToK8sAnnotations(),
		Labels: map[string]string{
			"apiName":        api.Name,
			"apiKind":        api.Kind.String(),
			"cortex.dev/api": "true",
		},
		Selector: map[string]string{
			"apiName": api.Name,
			"apiKind": api.Kind.String(),
		},
	})
}

func virtualServiceSpec(api *spec.API) *istioclientnetworking.VirtualService {
	var activatorWeight int32
	if api.Autoscaling.InitReplicas == 0 {
		activatorWeight = 100
	}

	return k8s.VirtualService(&k8s.VirtualServiceSpec{
		Name:     workloads.K8sName(api.Name),
		Gateways: []string{"apis-gateway"},
		Destinations: []k8s.Destination{
			{
				ServiceName: workloads.K8sName(api.Name),
				Weight:      100 - activatorWeight,
				Port:        uint32(consts.ProxyPortInt32),
				Headers: &istionetworking.Headers{
					Response: &istionetworking.Headers_HeaderOperations{
						Set: map[string]string{
							consts.CortexOriginHeader: "api",
						},
					},
				},
			},
			{
				ServiceName: consts.ActivatorName,
				Weight:      activatorWeight,
				Port:        uint32(consts.ActivatorPortInt32),
				Headers: &istionetworking.Headers{
					Request: &istionetworking.Headers_HeaderOperations{
						Set: map[string]string{
							consts.CortexAPINameHeader: api.Name,
							consts.CortexTargetServiceHeader: fmt.Sprintf(
								"http://%s.%s:%d",
								workloads.K8sName(api.Name),
								consts.DefaultNamespace,
								consts.ProxyPortInt32,
							),
						},
					},
					Response: &istionetworking.Headers_HeaderOperations{
						Set: map[string]string{
							consts.CortexOriginHeader: consts.ActivatorName,
						},
					},
				},
			},
		},
		PrefixPath:  api.Networking.Endpoint,
		Rewrite:     pointer.String("/"),
		Retries:     pointer.Int32(0),
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
		},
	})
}

func getRequestedReplicasFromDeployment(api spec.API, deployment *kapps.Deployment) int32 {
	requestedReplicas := api.Autoscaling.InitReplicas

	if deployment != nil && deployment.Spec.Replicas != nil && *deployment.Spec.Replicas > 0 {
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
