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
	"encoding/base64"
	"path"
	"strings"

	"github.com/cortexlabs/cortex/pkg/lib/json"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"

	kapps "k8s.io/api/apps/v1"
	kcore "k8s.io/api/core/v1"
	kresource "k8s.io/apimachinery/pkg/api/resource"
	kunstructured "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	intstr "k8s.io/apimachinery/pkg/util/intstr"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/operator/api/context"
	"github.com/cortexlabs/cortex/pkg/operator/api/userconfig"
	"github.com/cortexlabs/cortex/pkg/operator/config"
)

const (
	apiContainerName            = "api"
	tfServingContainerName      = "serve"
	downloaderInitContainerName = "downloader"

	defaultPortInt32, defaultPortStr     = int32(8888), "8888"
	tfServingPortInt32, tfServingPortStr = int32(9000), "9000"
)

type APIWorkload struct {
	BaseWorkload
}

func populateAPIWorkloadIDs(ctx *context.Context, latestResourceWorkloadIDs map[string]string) {
	for _, api := range ctx.APIs {
		if api.WorkloadID != "" {
			continue
		}
		if workloadID := latestResourceWorkloadIDs[api.ID]; workloadID != "" {
			api.WorkloadID = workloadID
			continue
		}
		api.WorkloadID = generateWorkloadID()
	}
}

func extractAPIWorkloads(ctx *context.Context) []Workload {
	workloads := make([]Workload, 0, len(ctx.APIs))

	for _, api := range ctx.APIs {
		workloads = append(workloads, &APIWorkload{
			singleBaseWorkload(api, ctx.App.Name, workloadTypeAPI),
		})
	}

	return workloads
}

func (aw *APIWorkload) Start(ctx *context.Context) error {
	api := ctx.APIs.OneByID(aw.GetSingleResourceID())

	k8sDeloymentName := internalAPIName(api.Name, ctx.App.Name)
	k8sDeloyment, err := config.Kubernetes.GetDeployment(k8sDeloymentName)
	if err != nil {
		return err
	}
	hpa, err := config.Kubernetes.GetHPA(k8sDeloymentName)
	if err != nil {
		return err
	}

	desiredReplicas := getRequestedReplicasFromDeployment(api, k8sDeloyment, hpa)

	var deploymentSpec *kapps.Deployment
	switch {
	case api.TensorFlow != nil:
		deploymentSpec = tfAPISpec(ctx, api, aw.WorkloadID, desiredReplicas)
	case api.ONNX != nil:
		deploymentSpec = onnxAPISpec(ctx, api, aw.WorkloadID, desiredReplicas)
	case api.Predictor != nil:
		deploymentSpec = predictorAPISpec(ctx, api, aw.WorkloadID, desiredReplicas)
	default:
		return errors.New(api.Name, "unknown model format encountered") // unexpected
	}

	_, err = config.Kubernetes.ApplyService(serviceSpec(ctx, api))
	if err != nil {
		return err
	}

	_, err = config.Kubernetes.ApplyVirtualService(virtualServiceSpec(ctx, api))
	if err != nil {
		return err
	}

	if k8sDeloyment != nil && k8sDeloyment.Status.ReadyReplicas == 0 {
		config.Kubernetes.DeleteDeployment(k8sDeloymentName)
	}

	_, err = config.Kubernetes.ApplyDeployment(deploymentSpec)
	if err != nil {
		return err
	}

	// Delete HPA while updating replicas to avoid unwanted autoscaling
	_, err = config.Kubernetes.DeleteHPA(k8sDeloymentName)
	if err != nil {
		return err
	}

	return nil
}

func (aw *APIWorkload) IsSucceeded(ctx *context.Context) (bool, error) {
	api := ctx.APIs.OneByID(aw.GetSingleResourceID())
	k8sDeloymentName := internalAPIName(api.Name, ctx.App.Name)

	k8sDeployment, err := config.Kubernetes.GetDeployment(k8sDeloymentName)
	if err != nil {
		return false, err
	}
	if k8sDeployment == nil || k8sDeployment.Labels["resourceID"] != api.ID || k8sDeployment.DeletionTimestamp != nil {
		return false, nil
	}

	if doesAPIComputeNeedsUpdating(api, k8sDeployment) {
		return false, nil
	}

	updatedReplicas, err := numUpdatedReadyReplicas(ctx, api)
	if err != nil {
		return false, err
	}
	if updatedReplicas < api.Compute.MinReplicas {
		return false, nil
	}

	return true, nil
}

func (aw *APIWorkload) IsRunning(ctx *context.Context) (bool, error) {
	api := ctx.APIs.OneByID(aw.GetSingleResourceID())
	k8sDeloymentName := internalAPIName(api.Name, ctx.App.Name)

	k8sDeployment, err := config.Kubernetes.GetDeployment(k8sDeloymentName)
	if err != nil {
		return false, err
	}
	if k8sDeployment == nil || k8sDeployment.Labels["resourceID"] != api.ID || k8sDeployment.DeletionTimestamp != nil {
		return false, nil
	}

	if doesAPIComputeNeedsUpdating(api, k8sDeployment) {
		return false, nil
	}

	updatedReplicas, err := numUpdatedReadyReplicas(ctx, api)
	if err != nil {
		return false, err
	}
	if updatedReplicas < api.Compute.MinReplicas {
		return true, nil
	}

	return false, nil
}

func (aw *APIWorkload) IsStarted(ctx *context.Context) (bool, error) {
	api := ctx.APIs.OneByID(aw.GetSingleResourceID())
	k8sDeloymentName := internalAPIName(api.Name, ctx.App.Name)

	k8sDeployment, err := config.Kubernetes.GetDeployment(k8sDeloymentName)
	if err != nil {
		return false, err
	}
	if k8sDeployment == nil || k8sDeployment.Labels["resourceID"] != api.ID || k8sDeployment.DeletionTimestamp != nil {
		return false, nil
	}

	if doesAPIComputeNeedsUpdating(api, k8sDeployment) {
		return false, nil
	}

	return true, nil
}

func (aw *APIWorkload) CanRun(ctx *context.Context) (bool, error) {
	return areAllDataDependenciesSucceeded(ctx, aw.GetResourceIDs())
}

func (aw *APIWorkload) IsFailed(ctx *context.Context) (bool, error) {
	api := ctx.APIs.OneByID(aw.GetSingleResourceID())

	pods, err := config.Kubernetes.ListPodsByLabels(map[string]string{
		"appName":      ctx.App.Name,
		"workloadType": workloadTypeAPI,
		"apiName":      api.Name,
		"resourceID":   api.ID,
		"workloadID":   aw.GetWorkloadID(),
		"userFacing":   "true",
	})
	if err != nil {
		return false, err
	}

	for _, pod := range pods {
		podStatus := k8s.GetPodStatus(&pod)
		if podStatus == k8s.PodStatusFailed || podStatus == k8s.PodStatusKilled || podStatus == k8s.PodStatusKilledOOM {
			return true, nil
		}
	}

	return false, nil
}

type downloadContainerArg struct {
	From                 string `json:"from"`
	To                   string `json:"to"`
	Unzip                bool   `json:"unzip"`
	ItemName             string `json:"item_name"`               // name of the item being downloaded, just for logging (if "" nothing will be logged)
	TFModelVersionRename string `json:"tf_model_version_rename"` // e.g. passing in /mnt/model/1 will rename /mnt/model/* to /mnt/model/1 only if there is one item in /mnt/model/
}

func tfAPISpec(
	ctx *context.Context,
	api *context.API,
	workloadID string,
	desiredReplicas int32,
) *kapps.Deployment {
	apiResourceList := kcore.ResourceList{}
	tfServingResourceList := kcore.ResourceList{}
	tfServingLimitsList := kcore.ResourceList{}

	q1, q2 := api.Compute.CPU.SplitInTwo()
	apiResourceList[kcore.ResourceCPU] = *q1
	tfServingResourceList[kcore.ResourceCPU] = *q2

	if api.Compute.Mem != nil {
		q1, q2 := api.Compute.Mem.SplitInTwo()
		apiResourceList[kcore.ResourceMemory] = *q1
		tfServingResourceList[kcore.ResourceMemory] = *q2
	}

	servingImage := config.Cluster.ImageTFServe
	if api.Compute.GPU > 0 {
		servingImage = config.Cluster.ImageTFServeGPU
		tfServingResourceList["nvidia.com/gpu"] = *kresource.NewQuantity(api.Compute.GPU, kresource.DecimalSI)
		tfServingLimitsList["nvidia.com/gpu"] = *kresource.NewQuantity(api.Compute.GPU, kresource.DecimalSI)
	}

	downloadArgs := []downloadContainerArg{
		{
			From:                 ctx.APIs[api.Name].TensorFlow.Model,
			To:                   path.Join(consts.EmptyDirMountPath, "model"),
			Unzip:                strings.HasSuffix(ctx.APIs[api.Name].TensorFlow.Model, ".zip"),
			ItemName:             "model",
			TFModelVersionRename: path.Join(consts.EmptyDirMountPath, "model", "1"),
		},
	}

	if api.TensorFlow.RequestHandler != nil {
		downloadArgs = append(downloadArgs, downloadContainerArg{
			From:     config.AWS.S3Path(ctx.ProjectKey),
			To:       path.Join(consts.EmptyDirMountPath, "project"),
			Unzip:    true,
			ItemName: "project code",
		})
	}

	envVars := append(
		k8s.AWSCredentials(),
		kcore.EnvVar{
			Name: "HOST_IP",
			ValueFrom: &kcore.EnvVarSource{
				FieldRef: &kcore.ObjectFieldSelector{
					FieldPath: "status.hostIP",
				},
			},
		},
	)
	if api.TensorFlow.PythonPath != nil {
		envVars = append(envVars, kcore.EnvVar{
			Name:  "PYTHON_PATH",
			Value: path.Join(consts.EmptyDirMountPath, "project", *api.TensorFlow.PythonPath),
		})
	}

	downloadArgsBytes, _ := json.Marshal(downloadArgs)
	downloadArgsStr := base64.URLEncoding.EncodeToString(downloadArgsBytes)
	return k8s.Deployment(&k8s.DeploymentSpec{
		Name:     internalAPIName(api.Name, ctx.App.Name),
		Replicas: desiredReplicas,
		Labels: map[string]string{
			"appName":      ctx.App.Name,
			"workloadType": workloadTypeAPI,
			"apiName":      api.Name,
			"resourceID":   ctx.APIs[api.Name].ID,
			"workloadID":   workloadID,
		},
		Selector: map[string]string{
			"appName":      ctx.App.Name,
			"workloadType": workloadTypeAPI,
			"apiName":      api.Name,
		},
		PodSpec: k8s.PodSpec{
			Labels: map[string]string{
				"appName":      ctx.App.Name,
				"workloadType": workloadTypeAPI,
				"apiName":      api.Name,
				"resourceID":   ctx.APIs[api.Name].ID,
				"workloadID":   workloadID,
				"userFacing":   "true",
				"logGroupName": ctx.LogGroupName(api.Name),
			},
			Annotations: map[string]string{
				"traffic.sidecar.istio.io/excludeOutboundIPRanges": "0.0.0.0/0",
			},
			K8sPodSpec: kcore.PodSpec{
				RestartPolicy: "Always",
				InitContainers: []kcore.Container{
					{
						Name:            downloaderInitContainerName,
						Image:           config.Cluster.ImageDownloader,
						ImagePullPolicy: "Always",
						Args: []string{
							"--download=" + downloadArgsStr,
						},
						Env:          k8s.AWSCredentials(),
						VolumeMounts: defaultVolumeMounts(),
					},
				},
				Containers: []kcore.Container{
					{
						Name:            apiContainerName,
						Image:           config.Cluster.ImageTFAPI,
						ImagePullPolicy: kcore.PullAlways,
						Args: []string{
							"--workload-id=" + workloadID,
							"--port=" + defaultPortStr,
							"--tf-serve-port=" + tfServingPortStr,
							"--context=" + config.AWS.S3Path(ctx.Key),
							"--api=" + ctx.APIs[api.Name].ID,
							"--model-dir=" + path.Join(consts.EmptyDirMountPath, "model"),
							"--cache-dir=" + consts.ContextCacheDir,
							"--project-dir=" + path.Join(consts.EmptyDirMountPath, "project"),
						},
						Env:          envVars,
						VolumeMounts: defaultVolumeMounts(),
						ReadinessProbe: &kcore.Probe{
							InitialDelaySeconds: 5,
							TimeoutSeconds:      5,
							PeriodSeconds:       5,
							SuccessThreshold:    1,
							FailureThreshold:    2,
							Handler: kcore.Handler{
								HTTPGet: &kcore.HTTPGetAction{
									Path: "/healthz",
									Port: intstr.IntOrString{
										IntVal: defaultPortInt32,
									},
								},
							},
						},
						Resources: kcore.ResourceRequirements{
							Requests: apiResourceList,
						},
						Ports: []kcore.ContainerPort{
							{
								ContainerPort: defaultPortInt32,
							},
						},
					},
					{
						Name:            tfServingContainerName,
						Image:           servingImage,
						ImagePullPolicy: kcore.PullAlways,
						Args: []string{
							"--port=" + tfServingPortStr,
							"--model_base_path=" + path.Join(consts.EmptyDirMountPath, "model"),
						},
						Env:          k8s.AWSCredentials(),
						VolumeMounts: defaultVolumeMounts(),
						ReadinessProbe: &kcore.Probe{
							InitialDelaySeconds: 5,
							TimeoutSeconds:      5,
							PeriodSeconds:       5,
							SuccessThreshold:    1,
							FailureThreshold:    2,
							Handler: kcore.Handler{
								TCPSocket: &kcore.TCPSocketAction{
									Port: intstr.IntOrString{
										IntVal: tfServingPortInt32,
									},
								},
							},
						},
						Resources: kcore.ResourceRequirements{
							Requests: tfServingResourceList,
							Limits:   tfServingLimitsList,
						},
						Ports: []kcore.ContainerPort{
							{
								ContainerPort: tfServingPortInt32,
							},
						},
					},
				},
				Volumes:            defaultVolumes(),
				ServiceAccountName: "default",
			},
		},
		Namespace: consts.K8sNamespace,
	})
}

func predictorAPISpec(
	ctx *context.Context,
	api *context.API,
	workloadID string,
	desiredReplicas int32,
) *kapps.Deployment {
	servingImage := config.Cluster.ImagePredictorServe
	resourceList := kcore.ResourceList{}
	resourceLimitsList := kcore.ResourceList{}
	resourceList[kcore.ResourceCPU] = api.Compute.CPU.Quantity

	if api.Compute.Mem != nil {
		resourceList[kcore.ResourceMemory] = api.Compute.Mem.Quantity
	}

	if api.Compute.GPU > 0 {
		servingImage = config.Cluster.ImagePredictorServeGPU
		resourceList["nvidia.com/gpu"] = *kresource.NewQuantity(api.Compute.GPU, kresource.DecimalSI)
		resourceLimitsList["nvidia.com/gpu"] = *kresource.NewQuantity(api.Compute.GPU, kresource.DecimalSI)
	}

	downloadArgs := []downloadContainerArg{
		{
			From:     config.AWS.S3Path(ctx.ProjectKey),
			To:       path.Join(consts.EmptyDirMountPath, "project"),
			Unzip:    true,
			ItemName: "project code",
		},
	}

	if api.Predictor.Model != nil {
		downloadArgs = append(downloadArgs, downloadContainerArg{
			From:     *api.Predictor.Model,
			To:       path.Join(consts.EmptyDirMountPath, "model"),
			ItemName: "model",
		})
	}

	downloadArgsBytes, _ := json.Marshal(downloadArgs)
	downloadArgsStr := base64.URLEncoding.EncodeToString(downloadArgsBytes)

	envVars := append(
		k8s.AWSCredentials(),
		kcore.EnvVar{
			Name: "HOST_IP",
			ValueFrom: &kcore.EnvVarSource{
				FieldRef: &kcore.ObjectFieldSelector{
					FieldPath: "status.hostIP",
				},
			},
		},
	)
	if api.Predictor.PythonPath != nil {
		envVars = append(envVars, kcore.EnvVar{
			Name:  "PYTHON_PATH",
			Value: path.Join(consts.EmptyDirMountPath, "project", *api.Predictor.PythonPath),
		})
	}

	return k8s.Deployment(&k8s.DeploymentSpec{
		Name:     internalAPIName(api.Name, ctx.App.Name),
		Replicas: desiredReplicas,
		Labels: map[string]string{
			"appName":      ctx.App.Name,
			"workloadType": workloadTypeAPI,
			"apiName":      api.Name,
			"resourceID":   ctx.APIs[api.Name].ID,
			"workloadID":   workloadID,
		},
		Selector: map[string]string{
			"appName":      ctx.App.Name,
			"workloadType": workloadTypeAPI,
			"apiName":      api.Name,
		},
		PodSpec: k8s.PodSpec{
			Labels: map[string]string{
				"appName":      ctx.App.Name,
				"workloadType": workloadTypeAPI,
				"apiName":      api.Name,
				"resourceID":   ctx.APIs[api.Name].ID,
				"workloadID":   workloadID,
				"userFacing":   "true",
				"logGroupName": ctx.LogGroupName(api.Name),
			},
			Annotations: map[string]string{
				"traffic.sidecar.istio.io/excludeOutboundIPRanges": "0.0.0.0/0",
			},
			K8sPodSpec: kcore.PodSpec{
				RestartPolicy: "Always",
				InitContainers: []kcore.Container{
					{
						Name:            downloaderInitContainerName,
						Image:           config.Cluster.ImageDownloader,
						ImagePullPolicy: "Always",
						Args: []string{
							"--download=" + downloadArgsStr,
						},
						Env:          k8s.AWSCredentials(),
						VolumeMounts: defaultVolumeMounts(),
					},
				},
				Containers: []kcore.Container{
					{
						Name:            apiContainerName,
						Image:           servingImage,
						ImagePullPolicy: kcore.PullAlways,
						Args: []string{
							"--workload-id=" + workloadID,
							"--port=" + defaultPortStr,
							"--context=" + config.AWS.S3Path(ctx.Key),
							"--api=" + ctx.APIs[api.Name].ID,
							"--model-dir=" + path.Join(consts.EmptyDirMountPath, "model"),
							"--cache-dir=" + consts.ContextCacheDir,
							"--project-dir=" + path.Join(consts.EmptyDirMountPath, "project"),
						},
						Env:          envVars,
						VolumeMounts: defaultVolumeMounts(),
						ReadinessProbe: &kcore.Probe{
							InitialDelaySeconds: 5,
							TimeoutSeconds:      5,
							PeriodSeconds:       5,
							SuccessThreshold:    1,
							FailureThreshold:    2,
							Handler: kcore.Handler{
								HTTPGet: &kcore.HTTPGetAction{
									Path: "/healthz",
									Port: intstr.IntOrString{
										IntVal: defaultPortInt32,
									},
								},
							},
						},
						Resources: kcore.ResourceRequirements{
							Requests: resourceList,
							Limits:   resourceLimitsList,
						},
						Ports: []kcore.ContainerPort{
							{
								ContainerPort: defaultPortInt32,
							},
						},
					},
				},
				Volumes:            defaultVolumes(),
				ServiceAccountName: "default",
			},
		},
		Namespace: consts.K8sNamespace,
	})
}

func onnxAPISpec(
	ctx *context.Context,
	api *context.API,
	workloadID string,
	desiredReplicas int32,
) *kapps.Deployment {
	servingImage := config.Cluster.ImageONNXServe
	resourceList := kcore.ResourceList{}
	resourceLimitsList := kcore.ResourceList{}
	resourceList[kcore.ResourceCPU] = api.Compute.CPU.Quantity

	if api.Compute.Mem != nil {
		resourceList[kcore.ResourceMemory] = api.Compute.Mem.Quantity
	}

	if api.Compute.GPU > 0 {
		servingImage = config.Cluster.ImageONNXServeGPU
		resourceList["nvidia.com/gpu"] = *kresource.NewQuantity(api.Compute.GPU, kresource.DecimalSI)
		resourceLimitsList["nvidia.com/gpu"] = *kresource.NewQuantity(api.Compute.GPU, kresource.DecimalSI)
	}

	downloadArgs := []downloadContainerArg{
		{
			From:     ctx.APIs[api.Name].ONNX.Model,
			To:       path.Join(consts.EmptyDirMountPath, "model"),
			ItemName: "model",
		},
	}

	if api.ONNX.RequestHandler != nil {
		downloadArgs = append(downloadArgs, downloadContainerArg{
			From:     config.AWS.S3Path(ctx.ProjectKey),
			To:       path.Join(consts.EmptyDirMountPath, "project"),
			Unzip:    true,
			ItemName: "project code",
		})
	}

	envVars := append(
		k8s.AWSCredentials(),
		kcore.EnvVar{
			Name: "HOST_IP",
			ValueFrom: &kcore.EnvVarSource{
				FieldRef: &kcore.ObjectFieldSelector{
					FieldPath: "status.hostIP",
				},
			},
		},
	)
	if api.ONNX.PythonPath != nil {
		envVars = append(envVars, kcore.EnvVar{
			Name:  "PYTHON_PATH",
			Value: path.Join(consts.EmptyDirMountPath, "project", *api.ONNX.PythonPath),
		})
	}

	downloadArgsBytes, _ := json.Marshal(downloadArgs)
	downloadArgsStr := base64.URLEncoding.EncodeToString(downloadArgsBytes)
	return k8s.Deployment(&k8s.DeploymentSpec{
		Name:     internalAPIName(api.Name, ctx.App.Name),
		Replicas: desiredReplicas,
		Labels: map[string]string{
			"appName":      ctx.App.Name,
			"workloadType": workloadTypeAPI,
			"apiName":      api.Name,
			"resourceID":   ctx.APIs[api.Name].ID,
			"workloadID":   workloadID,
		},
		Selector: map[string]string{
			"appName":      ctx.App.Name,
			"workloadType": workloadTypeAPI,
			"apiName":      api.Name,
		},
		PodSpec: k8s.PodSpec{
			Labels: map[string]string{
				"appName":      ctx.App.Name,
				"workloadType": workloadTypeAPI,
				"apiName":      api.Name,
				"resourceID":   ctx.APIs[api.Name].ID,
				"workloadID":   workloadID,
				"userFacing":   "true",
				"logGroupName": ctx.LogGroupName(api.Name),
			},
			Annotations: map[string]string{
				"traffic.sidecar.istio.io/excludeOutboundIPRanges": "0.0.0.0/0",
			},
			K8sPodSpec: kcore.PodSpec{
				InitContainers: []kcore.Container{
					{
						Name:            downloaderInitContainerName,
						Image:           config.Cluster.ImageDownloader,
						ImagePullPolicy: "Always",
						Args: []string{
							"--download=" + downloadArgsStr,
						},
						Env:          k8s.AWSCredentials(),
						VolumeMounts: defaultVolumeMounts(),
					},
				},
				Containers: []kcore.Container{
					{
						Name:            apiContainerName,
						Image:           servingImage,
						ImagePullPolicy: kcore.PullAlways,
						Args: []string{
							"--workload-id=" + workloadID,
							"--port=" + defaultPortStr,
							"--context=" + config.AWS.S3Path(ctx.Key),
							"--api=" + ctx.APIs[api.Name].ID,
							"--model-dir=" + path.Join(consts.EmptyDirMountPath, "model"),
							"--cache-dir=" + consts.ContextCacheDir,
							"--project-dir=" + path.Join(consts.EmptyDirMountPath, "project"),
						},
						Env:          envVars,
						VolumeMounts: defaultVolumeMounts(),
						ReadinessProbe: &kcore.Probe{
							InitialDelaySeconds: 5,
							TimeoutSeconds:      5,
							PeriodSeconds:       5,
							SuccessThreshold:    1,
							FailureThreshold:    2,
							Handler: kcore.Handler{
								HTTPGet: &kcore.HTTPGetAction{
									Path: "/healthz",
									Port: intstr.IntOrString{
										IntVal: defaultPortInt32,
									},
								},
							},
						},
						Resources: kcore.ResourceRequirements{
							Requests: resourceList,
							Limits:   resourceLimitsList,
						},
						Ports: []kcore.ContainerPort{
							{
								ContainerPort: defaultPortInt32,
							},
						},
					},
				},
				Volumes:            defaultVolumes(),
				ServiceAccountName: "default",
			},
		},
		Namespace: consts.K8sNamespace,
	})
}

func virtualServiceSpec(ctx *context.Context, api *context.API) *kunstructured.Unstructured {
	return k8s.VirtualService(&k8s.VirtualServiceSpec{
		Name:        internalAPIName(api.Name, ctx.App.Name),
		Namespace:   consts.K8sNamespace,
		Gateways:    []string{"apis-gateway"},
		ServiceName: internalAPIName(api.Name, ctx.App.Name),
		ServicePort: defaultPortInt32,
		Path:        *api.Endpoint,
		Rewrite:     pointer.String("predict"),
		Labels: map[string]string{
			"appName":      ctx.App.Name,
			"workloadType": workloadTypeAPI,
			"apiName":      api.Name,
		},
	})
}

func serviceSpec(ctx *context.Context, api *context.API) *kcore.Service {
	return k8s.Service(&k8s.ServiceSpec{
		Name:       internalAPIName(api.Name, ctx.App.Name),
		Port:       defaultPortInt32,
		TargetPort: defaultPortInt32,
		Labels: map[string]string{
			"appName":      ctx.App.Name,
			"workloadType": workloadTypeAPI,
			"apiName":      api.Name,
		},
		Selector: map[string]string{
			"appName":      ctx.App.Name,
			"workloadType": workloadTypeAPI,
			"apiName":      api.Name,
		},
		Namespace: consts.K8sNamespace,
	})
}

func doesAPIComputeNeedsUpdating(api *context.API, k8sDeployment *kapps.Deployment) bool {
	requestedReplicas := getRequestedReplicasFromDeployment(api, k8sDeployment, nil)
	if k8sDeployment.Spec.Replicas == nil || *k8sDeployment.Spec.Replicas != requestedReplicas {
		return true
	}

	curCPU, curMem, curGPU := APIPodCompute(k8sDeployment.Spec.Template.Spec.Containers)
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
	virtualServices, _ := config.Kubernetes.ListVirtualServicesByLabels(consts.K8sNamespace, map[string]string{
		"appName":      ctx.App.Name,
		"workloadType": workloadTypeAPI,
	})
	for _, virtualService := range virtualServices {
		if _, ok := ctx.APIs[virtualService.GetLabels()["apiName"]]; !ok {
			config.Kubernetes.DeleteVirtualService(virtualService.GetName(), consts.K8sNamespace)
		}
	}

	services, _ := config.Kubernetes.ListServicesByLabels(map[string]string{
		"appName":      ctx.App.Name,
		"workloadType": workloadTypeAPI,
	})
	for _, service := range services {
		if _, ok := ctx.APIs[service.Labels["apiName"]]; !ok {
			config.Kubernetes.DeleteService(service.Name)
		}
	}

	deployments, _ := config.Kubernetes.ListDeploymentsByLabels(map[string]string{
		"appName":      ctx.App.Name,
		"workloadType": workloadTypeAPI,
	})
	for _, deployment := range deployments {
		if _, ok := ctx.APIs[deployment.Labels["apiName"]]; !ok {
			config.Kubernetes.DeleteDeployment(deployment.Name)
		}
	}

	hpas, _ := config.Kubernetes.ListHPAsByLabels(map[string]string{
		"appName":      ctx.App.Name,
		"workloadType": workloadTypeAPI,
	})
	for _, hpa := range hpas {
		if _, ok := ctx.APIs[hpa.Labels["apiName"]]; !ok {
			config.Kubernetes.DeleteHPA(hpa.Name)
		}
	}
}

// This returns map apiName -> deployment (not internalName -> deployment)
func apiDeploymentMap(appName string) (map[string]*kapps.Deployment, error) {
	deploymentList, err := config.Kubernetes.ListDeploymentsByLabels(map[string]string{
		"appName":      appName,
		"workloadType": workloadTypeAPI,
	})
	if err != nil {
		return nil, errors.Wrap(err, appName)
	}

	deployments := make(map[string]*kapps.Deployment, len(deploymentList))
	for _, deployment := range deploymentList {
		addToDeploymentMap(deployments, deployment)
	}
	return deployments, nil
}

// Avoid pointer in loop issues
func addToDeploymentMap(deployments map[string]*kapps.Deployment, deployment kapps.Deployment) {
	apiName := deployment.Labels["apiName"]
	deployments[apiName] = &deployment
}

func internalAPIName(apiName string, appName string) string {
	return appName + "----" + apiName
}

func APIsBaseURL() (string, error) {
	service, err := config.IstioKubernetes.GetService("apis-ingressgateway")
	if err != nil {
		return "", err
	}
	if service == nil {
		return "", ErrorCortexInstallationBroken()
	}
	if len(service.Status.LoadBalancer.Ingress) == 0 {
		return "", ErrorLoadBalancerInitializing()
	}
	return "http://" + service.Status.LoadBalancer.Ingress[0].Hostname, nil
}

func APIPodComputeID(containers []kcore.Container) string {
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

func APIPodCompute(containers []kcore.Container) (*userconfig.Quantity, *userconfig.Quantity, int64) {
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

		if cpu, ok := requests[kcore.ResourceCPU]; ok {
			if totalCPU == nil {
				totalCPU = &userconfig.Quantity{}
			}
			totalCPU.Add(cpu)
		}
		if mem, ok := requests[kcore.ResourceMemory]; ok {
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
