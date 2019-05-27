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
	"path"

	appsv1b1 "k8s.io/api/apps/v1beta1"
	corev1 "k8s.io/api/core/v1"
	k8sresource "k8s.io/apimachinery/pkg/api/resource"
	intstr "k8s.io/apimachinery/pkg/util/intstr"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/sets/strset"
	"github.com/cortexlabs/cortex/pkg/operator/api/context"
	"github.com/cortexlabs/cortex/pkg/operator/api/userconfig"
	"github.com/cortexlabs/cortex/pkg/operator/config"
)

const (
	apiContainerName       = "api"
	tfServingContainerName = "serve"
)

func apiSpec(
	ctx *context.Context,
	apiName string,
	workloadID string,
	apiCompute *userconfig.APICompute,
) *appsv1b1.Deployment {

	transformResourceList := corev1.ResourceList{}
	tfServingResourceList := corev1.ResourceList{}
	tfServingLimitsList := corev1.ResourceList{}

	if apiCompute.CPU != nil {
		q1, q2 := apiCompute.CPU.SplitInTwo()
		transformResourceList[corev1.ResourceCPU] = *q1
		tfServingResourceList[corev1.ResourceCPU] = *q2
	}
	if apiCompute.Mem != nil {
		q1, q2 := apiCompute.Mem.SplitInTwo()
		transformResourceList[corev1.ResourceMemory] = *q1
		tfServingResourceList[corev1.ResourceMemory] = *q2
	}

	servingImage := config.Cortex.TFServeImage
	if apiCompute.GPU > 0 {
		servingImage = config.Cortex.TFServeImageGPU
		tfServingResourceList["nvidia.com/gpu"] = *k8sresource.NewQuantity(apiCompute.GPU, k8sresource.DecimalSI)
		tfServingLimitsList["nvidia.com/gpu"] = *k8sresource.NewQuantity(apiCompute.GPU, k8sresource.DecimalSI)
	}

	return k8s.Deployment(&k8s.DeploymentSpec{
		Name:     internalAPIName(apiName, ctx.App.Name),
		Replicas: ctx.APIs[apiName].Compute.Replicas,
		Labels: map[string]string{
			"appName":      ctx.App.Name,
			"workloadType": WorkloadTypeAPI,
			"apiName":      apiName,
			"resourceID":   ctx.APIs[apiName].ID,
			"workloadID":   workloadID,
		},
		Selector: map[string]string{
			"appName":      ctx.App.Name,
			"workloadType": WorkloadTypeAPI,
			"apiName":      apiName,
		},
		PodSpec: k8s.PodSpec{
			Labels: map[string]string{
				"appName":      ctx.App.Name,
				"workloadType": WorkloadTypeAPI,
				"apiName":      apiName,
				"resourceID":   ctx.APIs[apiName].ID,
				"workloadID":   workloadID,
				"userFacing":   "true",
			},
			K8sPodSpec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:            apiContainerName,
						Image:           config.Cortex.TFAPIImage,
						ImagePullPolicy: "Always",
						Args: []string{
							"--workload-id=" + workloadID,
							"--port=" + defaultPortStr,
							"--tf-serve-port=" + tfServingPortStr,
							"--context=" + config.Cloud.InternalPath(ctx.Key),
							"--api=" + ctx.APIs[apiName].ID,
							"--model-dir=" + path.Join(consts.EmptyDirMountPath, "model"),
							"--cache-dir=" + consts.ContextCacheDir,
							"--cloud-provider-type=" + config.Cloud.ProviderType.String(),
						},
						Env:          config.Cloud.EnvCredentials(),
						VolumeMounts: append(k8s.DefaultVolumeMounts(), config.Cloud.StorageVolumeMounts()...),
						ReadinessProbe: &corev1.Probe{
							InitialDelaySeconds: 5,
							TimeoutSeconds:      5,
							PeriodSeconds:       5,
							SuccessThreshold:    1,
							FailureThreshold:    2,
							Handler: corev1.Handler{
								HTTPGet: &corev1.HTTPGetAction{
									Path: "/healthz",
									Port: intstr.IntOrString{
										IntVal: defaultPortInt32,
									},
								},
							},
						},
						Resources: corev1.ResourceRequirements{
							Requests: transformResourceList,
						},
					},
					{
						Name:            tfServingContainerName,
						Image:           servingImage,
						ImagePullPolicy: "Always",
						Args: []string{
							"--port=" + tfServingPortStr,
							"--model_base_path=" + path.Join(consts.EmptyDirMountPath, "model"),
						},
						Env:          config.Cloud.EnvCredentials(),
						VolumeMounts: append(k8s.DefaultVolumeMounts(), config.Cloud.StorageVolumeMounts()...),
						ReadinessProbe: &corev1.Probe{
							InitialDelaySeconds: 5,
							TimeoutSeconds:      5,
							PeriodSeconds:       5,
							SuccessThreshold:    1,
							FailureThreshold:    2,
							Handler: corev1.Handler{
								TCPSocket: &corev1.TCPSocketAction{
									Port: intstr.IntOrString{
										IntVal: tfServingPortInt32,
									},
								},
							},
						},
						Resources: corev1.ResourceRequirements{
							Requests: tfServingResourceList,
							Limits:   tfServingLimitsList,
						},
					},
				},
				Volumes:            append(k8s.DefaultVolumes(), config.Cloud.StorageVolumes()...),
				ServiceAccountName: "default",
			},
		},
		Namespace: config.Cortex.Namespace,
	})
}

func ingressSpec(ctx *context.Context, apiName string) *k8s.IngressSpec {
	return &k8s.IngressSpec{
		Name:         internalAPIName(apiName, ctx.App.Name),
		ServiceName:  internalAPIName(apiName, ctx.App.Name),
		ServicePort:  defaultPortInt32,
		Path:         context.APIPath(apiName, ctx.App.Name),
		IngressClass: "apis",
		Labels: map[string]string{
			"appName":      ctx.App.Name,
			"workloadType": WorkloadTypeAPI,
			"apiName":      apiName,
		},
		Namespace: config.Cortex.Namespace,
	}
}

func serviceSpec(ctx *context.Context, apiName string) *k8s.ServiceSpec {
	return &k8s.ServiceSpec{
		Name:       internalAPIName(apiName, ctx.App.Name),
		Port:       defaultPortInt32,
		TargetPort: defaultPortInt32,
		Labels: map[string]string{
			"appName":      ctx.App.Name,
			"workloadType": WorkloadTypeAPI,
			"apiName":      apiName,
		},
		Selector: map[string]string{
			"appName":      ctx.App.Name,
			"workloadType": WorkloadTypeAPI,
			"apiName":      apiName,
		},
		Namespace: config.Cortex.Namespace,
	}
}

func apiWorkloadSpecs(ctx *context.Context) ([]*WorkloadSpec, error) {
	var workloadSpecs []*WorkloadSpec

	deployments, err := deploymentMap(ctx.App.Name)
	if err != nil {
		return nil, err
	}

	for apiName, api := range ctx.APIs {
		workloadID := generateWorkloadID()
		deployment, deploymentExists := deployments[apiName]
		if deploymentExists && deployment.Labels["resourceID"] == api.ID && deployment.DeletionTimestamp == nil {
			currentCompute := APIDeploymentCompute(deployment)
			if api.Compute.Equal(currentCompute) {
				continue // Deployment is already up to date
			}
			workloadID = deployment.Labels["workloadID"] // Reuse workloadID if just modifying compute
		}

		workloadSpecs = append(workloadSpecs, &WorkloadSpec{
			WorkloadID:       workloadID,
			ResourceIDs:      strset.New(api.ID),
			Spec:             apiSpec(ctx, apiName, workloadID, api.Compute),
			K8sAction:        "apply",
			SuccessCondition: k8s.DeploymentSuccessConditionAll,
			WorkloadType:     WorkloadTypeAPI,
		})
	}

	return workloadSpecs, nil
}

func deleteOldAPIs(ctx *context.Context) {
	ingresses, _ := config.Kubernetes.ListIngressesByLabels(map[string]string{
		"appName":      ctx.App.Name,
		"workloadType": WorkloadTypeAPI,
	})
	for _, ingress := range ingresses {
		if _, ok := ctx.APIs[ingress.Labels["apiName"]]; !ok {
			config.Kubernetes.DeleteIngress(ingress.Name)
		}
	}

	services, _ := config.Kubernetes.ListServicesByLabels(map[string]string{
		"appName":      ctx.App.Name,
		"workloadType": WorkloadTypeAPI,
	})
	for _, service := range services {
		if _, ok := ctx.APIs[service.Labels["apiName"]]; !ok {
			config.Kubernetes.DeleteService(service.Name)
		}
	}

	deployments, _ := config.Kubernetes.ListDeploymentsByLabels(map[string]string{
		"appName":      ctx.App.Name,
		"workloadType": WorkloadTypeAPI,
	})
	for _, deployment := range deployments {
		if _, ok := ctx.APIs[deployment.Labels["apiName"]]; !ok {
			config.Kubernetes.DeleteDeployment(deployment.Name)
		}
	}
}

func createServicesAndIngresses(ctx *context.Context) error {
	for apiName := range ctx.APIs {
		ingressExists, err := config.Kubernetes.IngressExists(internalAPIName(apiName, ctx.App.Name))
		if err != nil {
			return errors.Wrap(err, ctx.App.Name, "ingresses", apiName, "create")
		}
		if !ingressExists {
			_, err = config.Kubernetes.CreateIngress(ingressSpec(ctx, apiName))
			if err != nil {
				return errors.Wrap(err, ctx.App.Name, "ingresses", apiName, "create")
			}
		}

		serviceExists, err := config.Kubernetes.ServiceExists(internalAPIName(apiName, ctx.App.Name))
		if err != nil {
			return errors.Wrap(err, ctx.App.Name, "services", apiName, "create")
		}
		if !serviceExists {
			_, err = config.Kubernetes.CreateService(serviceSpec(ctx, apiName))
			if err != nil {
				return errors.Wrap(err, ctx.App.Name, "services", apiName, "create")
			}
		}
	}
	return nil
}

// This returns map apiName -> deployment (not internalName -> deployment)
func deploymentMap(appName string) (map[string]*appsv1b1.Deployment, error) {
	deploymentList, err := config.Kubernetes.ListDeploymentsByLabels(map[string]string{
		"appName":      appName,
		"workloadType": WorkloadTypeAPI,
	})
	if err != nil {
		return nil, errors.Wrap(err, appName)
	}

	deployments := make(map[string]*appsv1b1.Deployment, len(deploymentList))
	for _, deployment := range deploymentList {
		addToDeploymentMap(deployments, deployment)
	}
	return deployments, nil
}

// Avoid pointer in loop issues
func addToDeploymentMap(deployments map[string]*appsv1b1.Deployment, deployment appsv1b1.Deployment) {
	apiName := deployment.Labels["apiName"]
	deployments[apiName] = &deployment
}

func internalAPIName(apiName string, appName string) string {
	return appName + "----" + apiName
}

func APIDeploymentCompute(deployment *appsv1b1.Deployment) userconfig.APICompute {
	replicas := int32(0)
	if deployment.Spec.Replicas != nil {
		replicas = *deployment.Spec.Replicas
	}

	cpu, mem, gpu := APIPodCompute(deployment.Spec.Template.Spec.Containers)

	return userconfig.APICompute{
		Replicas: replicas,
		CPU:      cpu,
		Mem:      mem,
		GPU:      gpu,
	}
}

func APIPodCompute(containers []corev1.Container) (*userconfig.Quantity, *userconfig.Quantity, int64) {
	var totalCPU *userconfig.Quantity
	var totalMem *userconfig.Quantity
	var totalGPU int64

	for _, container := range containers {
		if container.Name != apiContainerName && container.Name != tfServingContainerName {
			continue
		}

		requests := container.Resources.Requests
		if len(requests) == 0 {
			continue
		}

		if cpu, ok := requests[corev1.ResourceCPU]; ok {
			if totalCPU == nil {
				totalCPU = &userconfig.Quantity{}
			}
			totalCPU.Add(cpu)
		}
		if mem, ok := requests[corev1.ResourceMemory]; ok {
			if totalMem == nil {
				totalMem = &userconfig.Quantity{}
			}
			totalMem.Add(mem)
		}
		if gpu, ok := requests["nvidia.com/gpu"]; ok {
			gpuVal, ok := gpu.AsInt64()
			if ok {
				totalGPU += gpuVal
			}
		}
	}

	return totalCPU, totalMem, totalGPU
}
