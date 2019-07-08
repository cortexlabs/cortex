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
	autoscaling "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	k8sresource "k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func tfAPISpec(
	ctx *context.Context,
	api *context.API,
	workloadID string,
	desiredReplicas int32,
) *appsv1b1.Deployment {
	transformResourceList := corev1.ResourceList{}
	tfServingResourceList := corev1.ResourceList{}
	tfServingLimitsList := corev1.ResourceList{}

	q1, q2 := api.Compute.CPU.SplitInTwo()
	transformResourceList[corev1.ResourceCPU] = *q1
	tfServingResourceList[corev1.ResourceCPU] = *q2

	if api.Compute.Mem != nil {
		q1, q2 := api.Compute.Mem.SplitInTwo()
		transformResourceList[corev1.ResourceMemory] = *q1
		tfServingResourceList[corev1.ResourceMemory] = *q2
	}

	servingImage := config.Cortex.TFServeImage
	if api.Compute.GPU > 0 {
		servingImage = config.Cortex.TFServeImageGPU
		tfServingResourceList["nvidia.com/gpu"] = *k8sresource.NewQuantity(api.Compute.GPU, k8sresource.DecimalSI)
		tfServingLimitsList["nvidia.com/gpu"] = *k8sresource.NewQuantity(api.Compute.GPU, k8sresource.DecimalSI)
	}

	return k8s.Deployment(&k8s.DeploymentSpec{
		Name:     internalAPIName(api.Name, ctx.App.Name),
		Replicas: desiredReplicas,
		Labels: map[string]string{
			"appName":      ctx.App.Name,
			"workloadType": WorkloadTypeAPI,
			"apiName":      api.Name,
			"resourceID":   ctx.APIs[api.Name].ID,
			"workloadID":   workloadID,
		},
		Selector: map[string]string{
			"appName":      ctx.App.Name,
			"workloadType": WorkloadTypeAPI,
			"apiName":      api.Name,
		},
		PodSpec: k8s.PodSpec{
			Labels: map[string]string{
				"appName":      ctx.App.Name,
				"workloadType": WorkloadTypeAPI,
				"apiName":      api.Name,
				"resourceID":   ctx.APIs[api.Name].ID,
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
							"--context=" + config.AWS.S3Path(ctx.Key),
							"--api=" + ctx.APIs[api.Name].ID,
							"--model-dir=" + path.Join(consts.EmptyDirMountPath, "model"),
							"--cache-dir=" + consts.ContextCacheDir,
						},
						Env:          k8s.AWSCredentials(),
						VolumeMounts: k8s.DefaultVolumeMounts(),
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
						Env:          k8s.AWSCredentials(),
						VolumeMounts: k8s.DefaultVolumeMounts(),
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
				Volumes:            k8s.DefaultVolumes(),
				ServiceAccountName: "default",
			},
		},
		Namespace: config.Cortex.Namespace,
	})
}

func onnxAPISpec(
	ctx *context.Context,
	api *context.API,
	workloadID string,
	desiredReplicas int32,
) *appsv1b1.Deployment {
	resourceList := corev1.ResourceList{}
	resourceList[corev1.ResourceCPU] = api.Compute.CPU.Quantity

	if api.Compute.Mem != nil {
		resourceList[corev1.ResourceMemory] = api.Compute.Mem.Quantity
	}

	return k8s.Deployment(&k8s.DeploymentSpec{
		Name:     internalAPIName(api.Name, ctx.App.Name),
		Replicas: desiredReplicas,
		Labels: map[string]string{
			"appName":      ctx.App.Name,
			"workloadType": WorkloadTypeAPI,
			"apiName":      api.Name,
			"resourceID":   ctx.APIs[api.Name].ID,
			"workloadID":   workloadID,
		},
		Selector: map[string]string{
			"appName":      ctx.App.Name,
			"workloadType": WorkloadTypeAPI,
			"apiName":      api.Name,
		},
		PodSpec: k8s.PodSpec{
			Labels: map[string]string{
				"appName":      ctx.App.Name,
				"workloadType": WorkloadTypeAPI,
				"apiName":      api.Name,
				"resourceID":   ctx.APIs[api.Name].ID,
				"workloadID":   workloadID,
				"userFacing":   "true",
			},
			K8sPodSpec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:            apiContainerName,
						Image:           config.Cortex.ONNXServeImage,
						ImagePullPolicy: "Always",
						Args: []string{
							"--workload-id=" + workloadID,
							"--port=" + defaultPortStr,
							"--context=" + config.AWS.S3Path(ctx.Key),
							"--api=" + ctx.APIs[api.Name].ID,
							"--model-dir=" + path.Join(consts.EmptyDirMountPath, "model"),
							"--cache-dir=" + consts.ContextCacheDir,
						},
						Env:          k8s.AWSCredentials(),
						VolumeMounts: k8s.DefaultVolumeMounts(),
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
							Requests: resourceList,
						},
					},
				},
				Volumes:            k8s.DefaultVolumes(),
				ServiceAccountName: "default",
			},
		},
		Namespace: config.Cortex.Namespace,
	})
}

func ingressSpec(ctx *context.Context, api *context.API) *k8s.IngressSpec {
	return &k8s.IngressSpec{
		Name:         internalAPIName(api.Name, ctx.App.Name),
		ServiceName:  internalAPIName(api.Name, ctx.App.Name),
		ServicePort:  defaultPortInt32,
		Path:         context.APIPath(api.Name, ctx.App.Name),
		IngressClass: "apis",
		Labels: map[string]string{
			"appName":      ctx.App.Name,
			"workloadType": WorkloadTypeAPI,
			"apiName":      api.Name,
		},
		Namespace: config.Cortex.Namespace,
	}
}

func serviceSpec(ctx *context.Context, api *context.API) *k8s.ServiceSpec {
	return &k8s.ServiceSpec{
		Name:       internalAPIName(api.Name, ctx.App.Name),
		Port:       defaultPortInt32,
		TargetPort: defaultPortInt32,
		Labels: map[string]string{
			"appName":      ctx.App.Name,
			"workloadType": WorkloadTypeAPI,
			"apiName":      api.Name,
		},
		Selector: map[string]string{
			"appName":      ctx.App.Name,
			"workloadType": WorkloadTypeAPI,
			"apiName":      api.Name,
		},
		Namespace: config.Cortex.Namespace,
	}
}

func hpaSpec(ctx *context.Context, api *context.API) *autoscaling.HorizontalPodAutoscaler {
	return k8s.HPA(&k8s.HPASpec{
		DeploymentName:       internalAPIName(api.Name, ctx.App.Name),
		MinReplicas:          api.Compute.MinReplicas,
		MaxReplicas:          api.Compute.MaxReplicas,
		TargetCPUUtilization: api.Compute.TargetCPUUtilization,
		Labels: map[string]string{
			"appName":      ctx.App.Name,
			"workloadType": WorkloadTypeAPI,
			"apiName":      api.Name,
		},
		Namespace: config.Cortex.Namespace,
	})
}

func apiWorkloadSpecs(ctx *context.Context) ([]*WorkloadSpec, error) {
	var workloadSpecs []*WorkloadSpec

	deployments, err := APIDeploymentMap(ctx.App.Name)
	if err != nil {
		return nil, err
	}

	hpas, err := apiHPAMap(ctx.App.Name)
	if err != nil {
		return nil, err
	}

	for apiName, api := range ctx.APIs {
		workloadID := generateWorkloadID()
		desiredReplicas := api.Compute.InitReplicas

		deployment, deploymentExists := deployments[apiName]
		if deploymentExists && deployment.Labels["resourceID"] == api.ID && deployment.DeletionTimestamp == nil {
			hpa := hpas[apiName]

			if !apiComputeNeedsUpdating(api, deployment, hpa) {
				continue // Deployment is fully up to date (model and compute/replicas)
			}

			// Reuse workloadID if just modifying compute/replicas
			workloadID = deployment.Labels["workloadID"]

			// Use current replicas or min replicas
			if deployment.Spec.Replicas != nil {
				desiredReplicas = *deployment.Spec.Replicas
			}
			if hpa != nil && hpa.Spec.MinReplicas != nil && *hpa.Spec.MinReplicas > desiredReplicas {
				desiredReplicas = *hpa.Spec.MinReplicas
			}
		}

		var spec metav1.Object

		switch api.ModelFormat {
		case userconfig.TensorFlowModelFormat:
			spec = tfAPISpec(ctx, api, workloadID, desiredReplicas)
		case userconfig.ONNXModelFormat:
			spec = onnxAPISpec(ctx, api, workloadID, desiredReplicas)
		default:
			return nil, errors.New(api.Name, "unknown model format encountered") // unexpected
		}

		workloadSpecs = append(workloadSpecs, &WorkloadSpec{
			WorkloadID:   workloadID,
			ResourceIDs:  strset.New(api.ID),
			K8sSpecs:     []metav1.Object{spec, hpaSpec(ctx, api)},
			K8sAction:    "apply",
			WorkloadType: WorkloadTypeAPI,
			// SuccessCondition: k8s.DeploymentSuccessConditionAll,  # Currently success conditions don't work for multi-resource config
		})
	}

	return workloadSpecs, nil
}

func apiComputeNeedsUpdating(api *context.API, deployment *appsv1b1.Deployment, hpa *autoscaling.HorizontalPodAutoscaler) bool {
	if hpa == nil {
		return true
	}

	if hpa.Spec.MinReplicas != nil && api.Compute.MinReplicas != *hpa.Spec.MinReplicas {
		return true
	}
	if api.Compute.MaxReplicas != hpa.Spec.MaxReplicas {
		return true
	}
	if hpa.Spec.TargetCPUUtilizationPercentage != nil && api.Compute.TargetCPUUtilization != *hpa.Spec.TargetCPUUtilizationPercentage {
		return true
	}

	curCPU, curMem, curGPU := APIPodCompute(deployment.Spec.Template.Spec.Containers)
	if !userconfig.QuantityPtrsEqual(curCPU, &api.Compute.CPU) {
		return true
	}
	if !userconfig.QuantityPtrsEqual(curMem, api.Compute.Mem) {
		return true
	}
	if curGPU != api.Compute.GPU {
		return true
	}

	return false
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

	hpas, _ := config.Kubernetes.ListHPAsByLabels(map[string]string{
		"appName":      ctx.App.Name,
		"workloadType": WorkloadTypeAPI,
	})
	for _, hpa := range hpas {
		if _, ok := ctx.APIs[hpa.Labels["apiName"]]; !ok {
			config.Kubernetes.DeleteHPA(hpa.Name)
		}
	}
}

func createServicesAndIngresses(ctx *context.Context) error {
	for _, api := range ctx.APIs {
		ingressExists, err := config.Kubernetes.IngressExists(internalAPIName(api.Name, ctx.App.Name))
		if err != nil {
			return errors.Wrap(err, ctx.App.Name, "ingresses", api.Name, "create")
		}
		if !ingressExists {
			_, err = config.Kubernetes.CreateIngress(ingressSpec(ctx, api))
			if err != nil {
				return errors.Wrap(err, ctx.App.Name, "ingresses", api.Name, "create")
			}
		}

		serviceExists, err := config.Kubernetes.ServiceExists(internalAPIName(api.Name, ctx.App.Name))
		if err != nil {
			return errors.Wrap(err, ctx.App.Name, "services", api.Name, "create")
		}
		if !serviceExists {
			_, err = config.Kubernetes.CreateService(serviceSpec(ctx, api))
			if err != nil {
				return errors.Wrap(err, ctx.App.Name, "services", api.Name, "create")
			}
		}
	}
	return nil
}

// This returns map apiName -> deployment (not internalName -> deployment)
func APIDeploymentMap(appName string) (map[string]*appsv1b1.Deployment, error) {
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

// This returns map apiName -> hpa (not internalName -> hpa)
func apiHPAMap(appName string) (map[string]*autoscaling.HorizontalPodAutoscaler, error) {
	hpaList, err := config.Kubernetes.ListHPAsByLabels(map[string]string{
		"appName":      appName,
		"workloadType": WorkloadTypeAPI,
	})
	if err != nil {
		return nil, errors.Wrap(err, appName)
	}

	hpas := make(map[string]*autoscaling.HorizontalPodAutoscaler, len(hpaList))
	for _, hpa := range hpaList {
		addToHPAMap(hpas, hpa)
	}
	return hpas, nil
}

// Avoid pointer in loop issues
func addToHPAMap(hpas map[string]*autoscaling.HorizontalPodAutoscaler, hpa autoscaling.HorizontalPodAutoscaler) {
	apiName := hpa.Labels["apiName"]
	hpas[apiName] = &hpa
}

func internalAPIName(apiName string, appName string) string {
	return appName + "----" + apiName
}

func APIsBaseURL() (string, error) {
	service, err := config.Kubernetes.GetService("nginx-controller-apis")
	if err != nil {
		return "", err
	}
	if service == nil {
		return "", ErrorCortexInstallationBroken()
	}
	if len(service.Status.LoadBalancer.Ingress) == 0 {
		return "", ErrorLoadBalancerInitializing()
	}
	return "https://" + service.Status.LoadBalancer.Ingress[0].Hostname, nil
}

func APIPodComputeID(containers []corev1.Container) string {
	cpu, mem, gpu := APIPodCompute(containers)
	if cpu == nil {
		cpu = &userconfig.Quantity{} // unexpected, since 0 is disallowed
	}
	podAPICompute := userconfig.APICompute{
		CPU: *cpu,
		Mem: mem,
		GPU: gpu,
	}
	return podAPICompute.IDWithoutReplicas()
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
