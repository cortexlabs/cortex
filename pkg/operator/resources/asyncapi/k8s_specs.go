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
	"fmt"

	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/operator/operator"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	"istio.io/client-go/pkg/apis/networking/v1beta1"
	kapps "k8s.io/api/apps/v1"
	kautoscaling "k8s.io/api/autoscaling/v2beta2"
	kcore "k8s.io/api/core/v1"
)

var _terminationGracePeriodSeconds int64 = 60  // seconds
var _gatewayHPATargetCPUUtilization int32 = 80 // percentage
var _gatewayHPATargetMemUtilization int32 = 80 // percentage

func gatewayDeploymentSpec(api spec.API, prevDeployment *kapps.Deployment, queueURL string) kapps.Deployment {
	container := operator.AsyncGatewayContainers(api, queueURL)

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
			"apiID":            api.ID,
			"specID":           api.SpecID,
			"deploymentID":     api.DeploymentID,
			"handlerID":        api.HandlerID,
			"cortex.dev/api":   "true",
			"cortex.dev/async": "gateway",
		},
		PodSpec: k8s.PodSpec{
			Labels: map[string]string{
				"apiName":          api.Name,
				"apiKind":          api.Kind.String(),
				"deploymentID":     api.DeploymentID,
				"handlerID":        api.HandlerID,
				"cortex.dev/api":   "true",
				"cortex.dev/async": "gateway",
			},
			K8sPodSpec: kcore.PodSpec{
				RestartPolicy:                 "Always",
				TerminationGracePeriodSeconds: pointer.Int64(_terminationGracePeriodSeconds),
				Containers:                    []kcore.Container{container},
				NodeSelector:                  operator.NodeSelectors(),
				Tolerations:                   operator.GenerateResourceTolerations(),
				Affinity:                      operator.GenerateNodeAffinities(api.Compute.NodeGroups),
				ServiceAccountName:            operator.ServiceAccountName,
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
			"apiID":            api.ID,
			"specID":           api.SpecID,
			"deploymentID":     api.DeploymentID,
			"handlerID":        api.HandlerID,
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
		Name:        operator.K8sName(api.Name),
		PortName:    "http",
		Port:        operator.DefaultPortInt32,
		TargetPort:  operator.DefaultPortInt32,
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
		Name:     operator.K8sName(api.Name),
		Gateways: []string{"apis-gateway"},
		Destinations: []k8s.Destination{{
			ServiceName: operator.K8sName(api.Name),
			Weight:      100,
			Port:        uint32(operator.DefaultPortInt32),
		}},
		PrefixPath:  api.Networking.Endpoint,
		Rewrite:     pointer.String("/"),
		Annotations: api.ToK8sAnnotations(),
		Labels: map[string]string{
			"apiName":          api.Name,
			"apiKind":          api.Kind.String(),
			"apiID":            api.ID,
			"specID":           api.SpecID,
			"deploymentID":     api.DeploymentID,
			"handlerID":        api.HandlerID,
			"cortex.dev/api":   "true",
			"cortex.dev/async": "gateway",
		},
	})
}

func apiDeploymentSpec(api spec.API, prevDeployment *kapps.Deployment, queueURL string) kapps.Deployment {
	var (
		containers []kcore.Container
		volumes    []kcore.Volume
	)

	switch api.Handler.Type {
	case userconfig.PythonHandlerType:
		containers, volumes = operator.AsyncPythonHandlerContainers(api, queueURL)
	case userconfig.TensorFlowHandlerType:
		containers, volumes = operator.AsyncTensorflowHandlerContainers(api, queueURL)
	default:
		panic(fmt.Sprintf("invalid handler type: %s", api.Handler.Type))
	}

	return *k8s.Deployment(&k8s.DeploymentSpec{
		Name:           operator.K8sName(api.Name),
		Replicas:       getRequestedReplicasFromDeployment(api, prevDeployment),
		MaxSurge:       pointer.String(api.UpdateStrategy.MaxSurge),
		MaxUnavailable: pointer.String(api.UpdateStrategy.MaxUnavailable),
		Labels: map[string]string{
			"apiName":          api.Name,
			"apiKind":          api.Kind.String(),
			"apiID":            api.ID,
			"specID":           api.SpecID,
			"deploymentID":     api.DeploymentID,
			"handlerID":        api.HandlerID,
			"cortex.dev/api":   "true",
			"cortex.dev/async": "api",
		},
		Annotations: api.ToK8sAnnotations(),
		Selector: map[string]string{
			"apiName":          api.Name,
			"apiKind":          api.Kind.String(),
			"cortex.dev/async": "api",
		},
		PodSpec: k8s.PodSpec{
			Labels: map[string]string{
				"apiName":          api.Name,
				"apiKind":          api.Kind.String(),
				"deploymentID":     api.DeploymentID,
				"handlerID":        api.HandlerID,
				"cortex.dev/api":   "true",
				"cortex.dev/async": "api",
			},
			K8sPodSpec: kcore.PodSpec{
				RestartPolicy:                 "Always",
				TerminationGracePeriodSeconds: pointer.Int64(_terminationGracePeriodSeconds),
				InitContainers: []kcore.Container{
					operator.InitContainer(&api),
				},
				Containers:         containers,
				NodeSelector:       operator.NodeSelectors(),
				Tolerations:        operator.GenerateResourceTolerations(),
				Affinity:           operator.GenerateNodeAffinities(api.Compute.NodeGroups),
				Volumes:            volumes,
				ServiceAccountName: operator.ServiceAccountName,
			},
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
