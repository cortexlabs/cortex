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
	"fmt"
	"path"
	"strings"

	"github.com/cortexlabs/cortex/pkg/lib/json"
	"github.com/cortexlabs/cortex/pkg/lib/parallel"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	"github.com/cortexlabs/cortex/pkg/types/spec"

	kapps "k8s.io/api/apps/v1"
	kcore "k8s.io/api/core/v1"
	kresource "k8s.io/apimachinery/pkg/api/resource"
	kunstructured "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	intstr "k8s.io/apimachinery/pkg/util/intstr"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/operator/api/context"
	"github.com/cortexlabs/cortex/pkg/operator/api/userconfig"
	"github.com/cortexlabs/cortex/pkg/operator/config"
)

const (
	_spec_cache_dir = "/mnt/spec"
	_empty_dir_mount_path  = "/mnt"
	_empty_dir_volume_name = "mnt"

	_apiContainerName            = "api"
	_tfServingContainerName      = "serve"
	_downloaderInitContainerName = "downloader"
	_downloaderLastLog = "pulling the %s serving image"

	_defaultPortInt32, _defaultPortStr     = int32(8888), "8888"
	_tfServingPortInt32, _tfServingPortStr = int32(9000), "9000"
)

type downloadContainerConfig struct {
	DownloadArgs []downloadContainerArg `json:"download_args"`
	LastLog      string                 `json:"last_log"` // string to log at the conclusion of the downloader (if "" nothing will be logged)
}

type downloadContainerArg struct {
	From                 string `json:"from"`
	To                   string `json:"to"`
	Unzip                bool   `json:"unzip"`
	ItemName             string `json:"item_name"`               // name of the item being downloaded, just for logging (if "" nothing will be logged)
	TFModelVersionRename string `json:"tf_model_version_rename"` // e.g. passing in /mnt/model/1 will rename /mnt/model/* to /mnt/model/1 only if there is one item in /mnt/model/
	HideFromLog          bool   `json:"hide_from_log"`           // if true, don't log where the file is being downloaded from
	HideUnzippingLog     bool   `json:"hide_unzipping_log"`      // if true, don't log when unzipping
}

func deploymentSpec(api spec.API, prevDeployment *kapps.Deployment) *kapps.Deployment {
	switch api.Predictor.Type {
	case userconfig.TensorFlowPredictorType:
		return tfAPISpec(api, prevDeployment)
	case userconfig.ONNXPredictorType:
		return onnxAPISpec(api, prevDeployment)
	case userconfig.PythonPredictorType:
		return pythonAPISpec(api, prevDeployment)
	default:
		return nil // unexpected
	}
}

func tfAPISpec(
	api *spec.API,
	prevDeployment *kapps.Deployment,
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

	return k8s.Deployment(&k8s.DeploymentSpec{
		Name:     api.Name,
		Replicas: getRequestedReplicasFromDeployment(api, prevDeployment),
		Labels: map[string]string{
			"apiName":      api.Name,
			"apiID":   api.ID,
			// these labels are important to determine if the deployment was changed in any way
			"minReplicas": s.Int32(api.MinReplicas),
			"maxReplicas": s.Int32(api.MaxReplicas),
			"targetCPUUtilization": s.Int32(api.TargetCPUUtilization),
		},
		Selector: map[string]string{
			"apiName":      api.Name,
		},
		PodSpec: k8s.PodSpec{
			Labels: map[string]string{
				"apiName":      api.Name,
				"apiID":   api.ID,
			},
			Annotations: map[string]string{
				"traffic.sidecar.istio.io/excludeOutboundIPRanges": "0.0.0.0/0",
			},
			K8sPodSpec: kcore.PodSpec{
				RestartPolicy: "Always",
				InitContainers: []kcore.Container{
					{
						Name:            _downloaderInitContainerName,
						Image:           config.Cluster.ImageDownloader,
						ImagePullPolicy: "Always",
						Args: []string{"--download=" + tfDownloadArgs(api)},
						EnvFrom:      _baseEnvVars,
						VolumeMounts: defaultVolumeMounts(),
					},
				},
				Containers: []kcore.Container{
					{
						Name:            _apiContainerName,
						Image:           config.Cluster.ImageTFAPI,
						ImagePullPolicy: kcore.PullAlways,
						Args: []string{
							"--port=" + _defaultPortStr,
							"--tf-serve-port=" + _tfServingPortStr,
							"--spec=" + config.AWS.S3Path(api.Key),
							"--cache-dir=" + _spec_cache_dir,
							"--model-dir=" + path.Join(_empty_dir_mount_path, "model"),
							"--project-dir=" + path.Join(_empty_dir_mount_path, "project"),
						},
						Env:          getEnvVars(api),
						EnvFrom:      _baseEnvVars,
						VolumeMounts: defaultVolumeMounts(),
						ReadinessProbe: _apiReadinessProbe,
						Resources: kcore.ResourceRequirements{
							Requests: apiResourceList,
						},
						Ports: []kcore.ContainerPort{
							{
								ContainerPort: _defaultPortInt32,
							},
						},
					},
					{
						Name:            _tfServingContainerName,
						Image:           servingImage,
						ImagePullPolicy: kcore.PullAlways,
						Args: []string{
							"--port=" + _tfServingPortStr,
							"--model_base_path=" + path.Join(_empty_dir_mount_path, "model"),
						},
						Env:          getEnvVars(api),
						EnvFrom:      _baseEnvVars,
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
										IntVal: _tfServingPortInt32,
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
								ContainerPort: _tfServingPortInt32,
							},
						},
					},
				},
				NodeSelector: map[string]string{
					"workload": "true",
				},
				Tolerations:        _tolerations,
				Volumes:            _defaultVolumes,
				ServiceAccountName: "default",
			},
		},
		Namespace: "default",
	})
}

func tfDownloadArgs(api *spec.API) string {
	tensorflowModel := *api.Predictor.Model

	downloadConfig := downloadContainerConfig{
		LastLog: fmt.Sprintf(_downloaderLastLog, "tensorflow"),
		DownloadArgs: []downloadContainerArg{
			{
				From:             config.AWS.S3Path(api.ProjectKey),
				To:               path.Join(_empty_dir_mount_path, "project"),
				Unzip:            true,
				ItemName:         "the project code",
				HideFromLog:      true,
				HideUnzippingLog: true,
			},
			{
				From:                 tensorflowModel,
				To:                   path.Join(_empty_dir_mount_path, "model"),
				Unzip:                strings.HasSuffix(tensorflowModel, ".zip"),
				ItemName:             "the model",
				TFModelVersionRename: path.Join(_empty_dir_mount_path, "model", "1"),
			},
		},
	}

	downloadArgsBytes, _ := json.Marshal(downloadConfig)
	return base64.URLEncoding.EncodeToString(downloadArgsBytes)
}

func pythonAPISpec(
	api *spec.API,
	prevDeployment *kapps.Deployment,
) *kapps.Deployment {

	servingImage := config.Cluster.ImagePythonServe
	resourceList := kcore.ResourceList{}
	resourceLimitsList := kcore.ResourceList{}
	resourceList[kcore.ResourceCPU] = api.Compute.CPU.Quantity

	if api.Compute.Mem != nil {
		resourceList[kcore.ResourceMemory] = api.Compute.Mem.Quantity
	}

	if api.Compute.GPU > 0 {
		servingImage = config.Cluster.ImagePythonServeGPU
		resourceList["nvidia.com/gpu"] = *kresource.NewQuantity(api.Compute.GPU, kresource.DecimalSI)
		resourceLimitsList["nvidia.com/gpu"] = *kresource.NewQuantity(api.Compute.GPU, kresource.DecimalSI)
	}

	return k8s.Deployment(&k8s.DeploymentSpec{
		Name:     api.Name,
		Replicas: getRequestedReplicasFromDeployment(api, prevDeployment),
		Labels: map[string]string{
			"apiName":      api.Name,
			"apiID":   api.ID,
			// these labels are important to determine if the deployment was changed in any way
			"minReplicas": s.Int32(api.MinReplicas),
			"maxReplicas": s.Int32(api.MaxReplicas),
			"targetCPUUtilization": s.Int32(api.TargetCPUUtilization),
		},
		Selector: map[string]string{
			"apiName":      api.Name,
		},
		PodSpec: k8s.PodSpec{
			Labels: map[string]string{
				"apiName":      api.Name,
				"apiID":   api.ID,
			},
			Annotations: map[string]string{
				"traffic.sidecar.istio.io/excludeOutboundIPRanges": "0.0.0.0/0",
			},
			K8sPodSpec: kcore.PodSpec{
				RestartPolicy: "Always",
				InitContainers: []kcore.Container{
					{
						Name:            _downloaderInitContainerName,
						Image:           config.Cluster.ImageDownloader,
						ImagePullPolicy: "Always",
						Args: []string{"--download=" + pythonDownloadArgs(api)},
						EnvFrom:      _baseEnvVars,
						VolumeMounts: defaultVolumeMounts(),
					},
				},
				Containers: []kcore.Container{
					{
						Name:            _apiContainerName,
						Image:           servingImage,
						ImagePullPolicy: kcore.PullAlways,
						Args: []string{
							"--port=" + _defaultPortStr,
							"--spec=" + config.AWS.S3Path(api.Key),
							"--cache-dir=" + _spec_cache_dir,
							"--project-dir=" + path.Join(_empty_dir_mount_path, "project"),
						},
						Env:          getEnvVars(api),
						EnvFrom:      _baseEnvVars,
						VolumeMounts: defaultVolumeMounts(),
						ReadinessProbe: _apiReadinessProbe,
						Resources: kcore.ResourceRequirements{
							Requests: resourceList,
							Limits:   resourceLimitsList,
						},
						Ports: []kcore.ContainerPort{
							{
								ContainerPort: _defaultPortInt32,
							},
						},
					},
				},
				NodeSelector: map[string]string{
					"workload": "true",
				},
				Tolerations:        _tolerations,
				Volumes:            _defaultVolumes,
				ServiceAccountName: "default",
			},
		},
		Namespace: "default",
	})
}

func pythonDownloadArgs(api *spec.API) string {
	downloadConfig := downloadContainerConfig{
		LastLog: fmt.Sprintf(_downloaderLastLog, "python"),
		DownloadArgs: []downloadContainerArg{
			{
				From:             config.AWS.S3Path(ctx.ProjectKey),
				To:               path.Join(_empty_dir_mount_path, "project"),
				Unzip:            true,
				ItemName:         "the project code",
				HideFromLog:      true,
				HideUnzippingLog: true,
			},
		},
	}

	downloadArgsBytes, _ := json.Marshal(downloadConfig)
	return base64.URLEncoding.EncodeToString(downloadArgsBytes)
}

func onnxAPISpec(
	api *spec.API,
	prevDeployment *kapps.Deployment,
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

	return k8s.Deployment(&k8s.DeploymentSpec{
		Name:     api.Name,
		Replicas: getRequestedReplicasFromDeployment(api, prevDeployment),
		Labels: map[string]string{
			"apiName":      api.Name,
			"apiID":   api.ID,
			// these labels are important to determine if the deployment was changed in any way
			"minReplicas": s.Int32(api.MinReplicas),
			"maxReplicas": s.Int32(api.MaxReplicas),
			"targetCPUUtilization": s.Int32(api.TargetCPUUtilization),
		},
		Selector: map[string]string{
			"apiName":      api.Name,
		},
		PodSpec: k8s.PodSpec{
			Labels: map[string]string{
				"apiName":      api.Name,
				"apiID":   api.ID,
			},
			Annotations: map[string]string{
				"traffic.sidecar.istio.io/excludeOutboundIPRanges": "0.0.0.0/0",
			},
			K8sPodSpec: kcore.PodSpec{
				InitContainers: []kcore.Container{
					{
						Name:            _downloaderInitContainerName,
						Image:           config.Cluster.ImageDownloader,
						ImagePullPolicy: "Always",
						Args: []string{"--download=" + onnxDownloadArgs(api)},
						EnvFrom:      _baseEnvVars,
						VolumeMounts: defaultVolumeMounts(),
					},
				},
				Containers: []kcore.Container{
					{
						Name:            _apiContainerName,
						Image:           servingImage,
						ImagePullPolicy: kcore.PullAlways,
						Args: []string{
							"--port=" + _defaultPortStr,
							"--spec=" + config.AWS.S3Path(api.Key),
							"--cache-dir=" + _spec_cache_dir,
							"--model-dir=" + path.Join(_empty_dir_mount_path, "model"),
							"--project-dir=" + path.Join(_empty_dir_mount_path, "project"),
						},
						Env:          getEnvVars(api),
						EnvFrom:      _baseEnvVars,
						VolumeMounts: defaultVolumeMounts(),
						ReadinessProbe:  _apiReadinessProbe,,
						Resources: kcore.ResourceRequirements{
							Requests: resourceList,
							Limits:   resourceLimitsList,
						},
						Ports: []kcore.ContainerPort{
							{
								ContainerPort: _defaultPortInt32,
							},
						},
					},
				},
				NodeSelector: map[string]string{
					"workload": "true",
				},
				Tolerations:        _tolerations,
				Volumes:            _defaultVolumes,
				ServiceAccountName: "default",
			},
		},
		Namespace: "default",
	})
}

func onnxDownloadArgs(api *spec.API) string {
	downloadConfig := downloadContainerConfig{
		LastLog: fmt.Sprintf(_downloaderLastLog, "onnx"),
		DownloadArgs: []downloadContainerArg{
			{
				From:             config.AWS.S3Path(ctx.ProjectKey),
				To:               path.Join(_empty_dir_mount_path, "project"),
				Unzip:            true,
				ItemName:         "the project code",
				HideFromLog:      true,
				HideUnzippingLog: true,
			},
			{
				From:     *ctx.APIs[api.Name].Predictor.Model,
				To:       path.Join(_empty_dir_mount_path, "model"),
				ItemName: "the model",
			},
		},
	}

	downloadArgsBytes, _ := json.Marshal(downloadConfig)
	return base64.URLEncoding.EncodeToString(downloadArgsBytes)
}

func serviceSpec(api *spec.API) *kcore.Service {
	return k8s.Service(&k8s.ServiceSpec{
		Name:       api.Name,
		Port:       _defaultPortInt32,
		TargetPort: _defaultPortInt32,
		Labels: map[string]string{
			"apiName":      api.Name,
		},
		Selector: map[string]string{
			"apiName":      api.Name,
		},
		Namespace: "default",
	})
}

func virtualServiceSpec(api *spec.API) *kunstructured.Unstructured {
	return k8s.VirtualService(&k8s.VirtualServiceSpec{
		Name:        api.Name,
		Namespace:   "default",
		Gateways:    []string{"apis-gateway"},
		ServiceName: api.Name,
		ServicePort: _defaultPortInt32,
		Path:        *api.Endpoint,
		Rewrite:     pointer.String("predict"),
		Labels: map[string]string{
			"apiName":      api.Name,
		},
	})
}

func hpaSpec(deployment *kapps.Deployment) *kautoscaling.HorizontalPodAutoscaler {
	return k8s.HPA(&k8s.HPASpec{
		DeploymentName:       internalAPIName(api.Name, ctx.App.Name),
		MinReplicas:          s.ParseInt32(deployment.Labels["minReplicas"]),
		MaxReplicas:          s.ParseInt32(deployment.Labels["maxReplicas"]),
		TargetCPUUtilization: s.ParseInt32(deployment.Labels["targetCPUUtilization"]),,
		Labels: map[string]string{
			"apiName":      deployment.Labels["apiName"],
		},
		Namespace: "default",
	})
}

func getRequestedReplicasFromDeployment(api *spec.API, deployment *kapps.Deployment) int32 {
	var k8sRequested int32
	if deployment != nil && deployment.Spec.Replicas != nil {
		k8sRequested = *deployment.Spec.Replicas
	}
	return getRequestedReplicas(api, k8sRequested)
}

func getRequestedReplicas(api *spec.API, k8sRequested int32) int32 {
	requestedReplicas := api.Compute.InitReplicas
	if k8sRequested > 0 {
		requestedReplicas = k8sRequested
	}
	if requestedReplicas < api.Compute.MinReplicas {
		requestedReplicas = api.Compute.MinReplicas
	}
	if requestedReplicas > api.Compute.MaxReplicas {
		requestedReplicas = api.Compute.MaxReplicas
	}
	return requestedReplicas
}

func getEnvVars(api *spec.API) []kcore.EnvVar{} {
	envVars := []kcore.EnvVar{}

	for name, val := range api.Predictor.Env {
		envVars = append(envVars, kcore.EnvVar{
			Name:  name,
			Value: val,
		})
	}

	envVars = append(envVars,
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
			Value: path.Join(_empty_dir_mount_path, "project", *api.Predictor.PythonPath),
		})
	}

	return envVars
}

var _apiReadinessProbe := &kcore.Probe{
	InitialDelaySeconds: 5,
	TimeoutSeconds:      5,
	PeriodSeconds:       5,
	SuccessThreshold:    1,
	FailureThreshold:    2,
	Handler: kcore.Handler{
		Exec: &kcore.ExecAction{
			Command: []string{"/bin/bash", "-c", "/bin/ps aux | grep \"api.py\" && test -f /health_check.txt"},
		},
	},
}

var _tolerations = []kcore.Toleration{
	{
		Key:      "workload",
		Operator: kcore.TolerationOpEqual,
		Value:    "true",
		Effect:   kcore.TaintEffectNoSchedule,
	},
	{
		Key:      "nvidia.com/gpu",
		Operator: kcore.TolerationOpEqual,
		Value:    "true",
		Effect:   kcore.TaintEffectNoSchedule,
	},
}

var _baseEnvVars = []kcore.EnvFromSource{
	{
		ConfigMapRef: &kcore.ConfigMapEnvSource{
			LocalObjectReference: kcore.LocalObjectReference{
				Name: "env-vars",
			},
		},
	},
	{
		SecretRef: &kcore.SecretEnvSource{
			LocalObjectReference: kcore.LocalObjectReference{
				Name: "aws-credentials",
			},
		},
	},
}

var _defaultVolumes = []kcore.Volume{
	k8s.EmptyDirVolume(_empty_dir_volume_name),
}

var _defaultVolumeMounts = []kcore.VolumeMount{
	k8s.EmptyDirVolumeMount(_empty_dir_volume_name, _empty_dir_mount_path),
}
