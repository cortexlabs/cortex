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

package realtimeapi

import (
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/operator/operator"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	istioclientnetworking "istio.io/client-go/pkg/apis/networking/v1alpha3"
	kapps "k8s.io/api/apps/v1"
	kcore "k8s.io/api/core/v1"
)

func deploymentSpec(api *spec.API, prevDeployment *kapps.Deployment) *kapps.Deployment {
	switch api.Predictor.Type {
	case userconfig.TensorFlowPredictorType:
		return tensorflowAPISpec(api, prevDeployment)
	case userconfig.ONNXPredictorType:
		return onnxAPISpec(api, prevDeployment)
	case userconfig.PythonPredictorType:
		return pythonAPISpec(api, prevDeployment)
	default:
		return nil // unexpected
	}
}

func tensorflowAPISpec(api *spec.API, prevDeployment *kapps.Deployment) *kapps.Deployment {

	containers, volumes := operator.TensorFlowPredictorContainers(api)
	containers = append(containers, operator.RequestMonitorContainer(api))

	return k8s.Deployment(&k8s.DeploymentSpec{
		Name:           operator.K8sName(api.Name),
		Replicas:       getRequestedReplicasFromDeployment(api, prevDeployment),
		MaxSurge:       pointer.String(api.UpdateStrategy.MaxSurge),
		MaxUnavailable: pointer.String(api.UpdateStrategy.MaxUnavailable),
		Labels: map[string]string{
			"apiName":      api.Name,
			"apiKind":      api.Kind.String(),
			"apiID":        api.ID,
			"deploymentID": api.DeploymentID,
		},
		Annotations: api.ToK8sAnnotations(),
		Selector: map[string]string{
			"apiName": api.Name,
			"apiKind": api.Kind.String(),
		},
		PodSpec: k8s.PodSpec{
			Labels: map[string]string{
				"apiName":      api.Name,
				"apiKind":      api.Kind.String(),
				"apiID":        api.ID,
				"deploymentID": api.DeploymentID,
			},
			Annotations: map[string]string{
				"traffic.sidecar.istio.io/excludeOutboundIPRanges": "0.0.0.0/0",
			},
			K8sPodSpec: kcore.PodSpec{
				RestartPolicy: "Always",
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
	})
}

func pythonAPISpec(api *spec.API, prevDeployment *kapps.Deployment) *kapps.Deployment {
	containers, volumes := operator.PythonPredictorContainers(api)
	containers = append(containers, operator.RequestMonitorContainer(api))

	return k8s.Deployment(&k8s.DeploymentSpec{
		Name:           operator.K8sName(api.Name),
		Replicas:       getRequestedReplicasFromDeployment(api, prevDeployment),
		MaxSurge:       pointer.String(api.UpdateStrategy.MaxSurge),
		MaxUnavailable: pointer.String(api.UpdateStrategy.MaxUnavailable),
		Labels: map[string]string{
			"apiName":      api.Name,
			"apiKind":      api.Kind.String(),
			"apiID":        api.ID,
			"deploymentID": api.DeploymentID,
		},
		Annotations: api.ToK8sAnnotations(),
		Selector: map[string]string{
			"apiName": api.Name,
			"apiKind": api.Kind.String(),
		},
		PodSpec: k8s.PodSpec{
			Labels: map[string]string{
				"apiName":      api.Name,
				"apiKind":      api.Kind.String(),
				"apiID":        api.ID,
				"deploymentID": api.DeploymentID,
			},
			Annotations: map[string]string{
				"traffic.sidecar.istio.io/excludeOutboundIPRanges": "0.0.0.0/0",
			},
			K8sPodSpec: kcore.PodSpec{
				RestartPolicy: "Always",
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
	})
}

func onnxAPISpec(api *spec.API, prevDeployment *kapps.Deployment) *kapps.Deployment {
	containers := operator.ONNXPredictorContainers(api)
	containers = append(containers, operator.RequestMonitorContainer(api))

	return k8s.Deployment(&k8s.DeploymentSpec{
		Name:           operator.K8sName(api.Name),
		Replicas:       getRequestedReplicasFromDeployment(api, prevDeployment),
		MaxSurge:       pointer.String(api.UpdateStrategy.MaxSurge),
		MaxUnavailable: pointer.String(api.UpdateStrategy.MaxUnavailable),
		Labels: map[string]string{
			"apiName":      api.Name,
			"apiKind":      api.Kind.String(),
			"apiID":        api.ID,
			"deploymentID": api.DeploymentID,
		},
		Annotations: api.ToK8sAnnotations(),
		Selector: map[string]string{
			"apiName": api.Name,
			"apiKind": api.Kind.String(),
		},
		PodSpec: k8s.PodSpec{
			Labels: map[string]string{
				"apiName":      api.Name,
				"apiKind":      api.Kind.String(),
				"apiID":        api.ID,
				"deploymentID": api.DeploymentID,
			},
			Annotations: map[string]string{
				"traffic.sidecar.istio.io/excludeOutboundIPRanges": "0.0.0.0/0",
			},
			K8sPodSpec: kcore.PodSpec{
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
	})
}

func serviceSpec(api *spec.API) *kcore.Service {
	return k8s.Service(&k8s.ServiceSpec{
		Name:        operator.K8sName(api.Name),
		Port:        operator.DefaultPortInt32,
		TargetPort:  operator.DefaultPortInt32,
		Annotations: api.ToK8sAnnotations(),
		Labels: map[string]string{
			"apiName": api.Name,
			"apiKind": api.Kind.String(),
		},
		Selector: map[string]string{
			"apiName": api.Name,
			"apiKind": api.Kind.String(),
		},
	})
}

func virtualServiceSpec(api *spec.API) *istioclientnetworking.VirtualService {
	return k8s.VirtualService(&k8s.VirtualServiceSpec{
		Name:     operator.K8sName(api.Name),
		Gateways: []string{"apis-gateway"},
		Destinations: []k8s.Destination{{
			ServiceName: operator.K8sName(api.Name),
			Weight:      100,
			Port:        uint32(operator.DefaultPortInt32),
		}},
		ExactPath:   api.Networking.Endpoint,
		Rewrite:     pointer.String("predict"),
		Annotations: api.ToK8sAnnotations(),
		Labels: map[string]string{
			"apiName": api.Name,
			"apiKind": api.Kind.String(),
		},
	})
}

func getRequestedReplicasFromDeployment(api *spec.API, deployment *kapps.Deployment) int32 {
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
