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

package operator

import (
	"encoding/base64"
	"fmt"
	"math"
	"path"
	"strings"

	"github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/json"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	kapps "k8s.io/api/apps/v1"
	kcore "k8s.io/api/core/v1"
	kresource "k8s.io/apimachinery/pkg/api/resource"
	kunstructured "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	intstr "k8s.io/apimachinery/pkg/util/intstr"
)

const (
	_specCacheDir                          = "/mnt/spec"
	_emptyDirMountPath                     = "/mnt"
	_emptyDirVolumeName                    = "mnt"
	_apiContainerName                      = "api"
	_tfServingContainerName                = "serve"
	_downloaderInitContainerName           = "downloader"
	_downloaderLastLog                     = "pulling the %s serving image"
	_defaultPortInt32, _defaultPortStr     = int32(8888), "8888"
	_tfServingPortInt32, _tfServingPortStr = int32(9000), "9000"
	_requestMonitorReadinessFile           = "/request_monitor_ready.txt"
	_apiReadinessFile                      = "/mnt/api_readiness.txt"
	_apiLivenessFile                       = "/mnt/api_liveness.txt"
	_apiLivenessStalePeriod                = 7 // seconds (there is a 2-second buffer to be safe)
)

var (
	_requestMonitorCPURequest = kresource.MustParse("10m")
	_requestMonitorMemRequest = kresource.MustParse("10Mi")
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

func deploymentSpec(api *spec.API, prevDeployment *kapps.Deployment) *kapps.Deployment {
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

func tfAPISpec(api *spec.API, prevDeployment *kapps.Deployment) *kapps.Deployment {
	apiResourceList := kcore.ResourceList{}
	tfServingResourceList := kcore.ResourceList{}
	tfServingLimitsList := kcore.ResourceList{}

	userPodCPURequest := api.Compute.CPU.Quantity.Copy()
	userPodCPURequest.Sub(_requestMonitorCPURequest)
	q1, q2 := k8s.SplitInTwo(userPodCPURequest)
	apiResourceList[kcore.ResourceCPU] = *q1
	tfServingResourceList[kcore.ResourceCPU] = *q2

	if api.Compute.Mem != nil {
		userPodMemRequest := api.Compute.Mem.Quantity.Copy()
		userPodMemRequest.Sub(_requestMonitorMemRequest)
		q1, q2 := k8s.SplitInTwo(userPodMemRequest)
		apiResourceList[kcore.ResourceMemory] = *q1
		tfServingResourceList[kcore.ResourceMemory] = *q2
	}

	servingImage := config.Cluster.ImagePythonServe
	tfServingImage := config.Cluster.ImageTFAPI
	if api.Compute.GPU > 0 {
		servingImage = config.Cluster.ImageTFServeGPU
		tfServingResourceList["nvidia.com/gpu"] = *kresource.NewQuantity(api.Compute.GPU, kresource.DecimalSI)
		tfServingLimitsList["nvidia.com/gpu"] = *kresource.NewQuantity(api.Compute.GPU, kresource.DecimalSI)
	}
	if api.Predictor.Image != "" {
		servingImage = api.Predictor.Image
		// tfServingImage = api.Predictor.TFServingImage
	}

	return k8s.Deployment(&k8s.DeploymentSpec{
		Name:           k8sName(api.Name),
		Replicas:       getRequestedReplicasFromDeployment(api, prevDeployment),
		MaxSurge:       pointer.String(api.UpdateStrategy.MaxSurge),
		MaxUnavailable: pointer.String(api.UpdateStrategy.MaxUnavailable),
		Labels: map[string]string{
			"apiName":      api.Name,
			"apiID":        api.ID,
			"deploymentID": api.DeploymentID,
		},
		Annotations: api.Autoscaling.ToK8sAnnotations(),
		Selector: map[string]string{
			"apiName": api.Name,
		},
		PodSpec: k8s.PodSpec{
			Labels: map[string]string{
				"apiName":      api.Name,
				"apiID":        api.ID,
				"deploymentID": api.DeploymentID,
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
						Args:            []string{"--download=" + tfDownloadArgs(api)},
						EnvFrom:         _baseEnvVars,
						VolumeMounts:    _defaultVolumeMounts,
					},
				},
				Containers: []kcore.Container{
					{
						Name:            _apiContainerName,
						Image:           tfServingImage,
						ImagePullPolicy: kcore.PullAlways,
						Env: append(
							getEnvVars(api),
							kcore.EnvVar{
								Name:  "CORTEX_MODEL_DIR",
								Value: path.Join(_emptyDirMountPath, "model"),
							},
							kcore.EnvVar{
								Name:  "CORTEX_TF_SERVING_PORT",
								Value: _tfServingPortStr,
							},
						),
						EnvFrom:        _baseEnvVars,
						VolumeMounts:   _defaultVolumeMounts,
						ReadinessProbe: fileExistsProbe(_apiReadinessFile),
						LivenessProbe:  _apiReadinessProbe,
						Resources: kcore.ResourceRequirements{
							Requests: apiResourceList,
						},
						Ports: []kcore.ContainerPort{
							{ContainerPort: _defaultPortInt32},
						},
						SecurityContext: &kcore.SecurityContext{
							Privileged: pointer.Bool(true),
						},
					},
					*requestMonitorContainer(api),
					{
						Name:            _tfServingContainerName,
						Image:           servingImage,
						ImagePullPolicy: kcore.PullAlways,
						Args: []string{
							"--port=" + _tfServingPortStr,
							"--model_base_path=" + path.Join(_emptyDirMountPath, "model"),
						},
						Env:          getEnvVars(api),
						EnvFrom:      _baseEnvVars,
						VolumeMounts: _defaultVolumeMounts,
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
	})
}

func tfDownloadArgs(api *spec.API) string {
	tensorflowModel := *api.Predictor.Model

	downloadConfig := downloadContainerConfig{
		LastLog: fmt.Sprintf(_downloaderLastLog, "tensorflow"),
		DownloadArgs: []downloadContainerArg{
			{
				From:             aws.S3Path(config.Cluster.Bucket, api.ProjectKey),
				To:               path.Join(_emptyDirMountPath, "project"),
				Unzip:            true,
				ItemName:         "the project code",
				HideFromLog:      true,
				HideUnzippingLog: true,
			},
			{
				From:                 tensorflowModel,
				To:                   path.Join(_emptyDirMountPath, "model"),
				Unzip:                strings.HasSuffix(tensorflowModel, ".zip"),
				ItemName:             "the model",
				TFModelVersionRename: path.Join(_emptyDirMountPath, "model", "1"),
			},
		},
	}

	downloadArgsBytes, _ := json.Marshal(downloadConfig)
	return base64.URLEncoding.EncodeToString(downloadArgsBytes)
}

func pythonAPISpec(api *spec.API, prevDeployment *kapps.Deployment) *kapps.Deployment {
	resourceList := kcore.ResourceList{}
	resourceLimitsList := kcore.ResourceList{}

	userPodCPURequest := api.Compute.CPU.Quantity.Copy()
	userPodCPURequest.Sub(_requestMonitorCPURequest)
	resourceList[kcore.ResourceCPU] = *userPodCPURequest

	if api.Compute.Mem != nil {
		userPodMemRequest := api.Compute.Mem.Quantity.Copy()
		userPodMemRequest.Sub(_requestMonitorMemRequest)
		resourceList[kcore.ResourceMemory] = *userPodMemRequest
	}
	
	servingImage := config.Cluster.ImagePythonServe
	if api.Compute.GPU > 0 {
		servingImage = config.Cluster.ImagePythonServeGPU
		resourceList["nvidia.com/gpu"] = *kresource.NewQuantity(api.Compute.GPU, kresource.DecimalSI)
		resourceLimitsList["nvidia.com/gpu"] = *kresource.NewQuantity(api.Compute.GPU, kresource.DecimalSI)
	}
	if api.Predictor.Image != "" {
		servingImage = api.Predictor.Image
	}

	return k8s.Deployment(&k8s.DeploymentSpec{
		Name:           k8sName(api.Name),
		Replicas:       getRequestedReplicasFromDeployment(api, prevDeployment),
		MaxSurge:       pointer.String(api.UpdateStrategy.MaxSurge),
		MaxUnavailable: pointer.String(api.UpdateStrategy.MaxUnavailable),
		Labels: map[string]string{
			"apiName":      api.Name,
			"apiID":        api.ID,
			"deploymentID": api.DeploymentID,
		},
		Annotations: api.Autoscaling.ToK8sAnnotations(),
		Selector: map[string]string{
			"apiName": api.Name,
		},
		PodSpec: k8s.PodSpec{
			Labels: map[string]string{
				"apiName":      api.Name,
				"apiID":        api.ID,
				"deploymentID": api.DeploymentID,
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
						Args:            []string{"--download=" + pythonDownloadArgs(api)},
						EnvFrom:         _baseEnvVars,
						VolumeMounts:    _defaultVolumeMounts,
					},
				},
				Containers: []kcore.Container{
					{
						Name:            _apiContainerName,
						Image:           servingImage,
						ImagePullPolicy: kcore.PullAlways,
						Env:             getEnvVars(api),
						EnvFrom:         _baseEnvVars,
						VolumeMounts:    _defaultVolumeMounts,
						ReadinessProbe:  fileExistsProbe(_apiReadinessFile),
						LivenessProbe:   _apiReadinessProbe,
						Resources: kcore.ResourceRequirements{
							Requests: resourceList,
							Limits:   resourceLimitsList,
						},
						Ports: []kcore.ContainerPort{
							{ContainerPort: _defaultPortInt32},
						},
						SecurityContext: &kcore.SecurityContext{
							Privileged: pointer.Bool(true),
						},
					},
					*requestMonitorContainer(api),
				},
				NodeSelector: map[string]string{
					"workload": "true",
				},
				Tolerations:        _tolerations,
				Volumes:            _defaultVolumes,
				ServiceAccountName: "default",
			},
		},
	})
}

func pythonDownloadArgs(api *spec.API) string {
	downloadConfig := downloadContainerConfig{
		LastLog: fmt.Sprintf(_downloaderLastLog, "python"),
		DownloadArgs: []downloadContainerArg{
			{
				From:             aws.S3Path(config.Cluster.Bucket, api.ProjectKey),
				To:               path.Join(_emptyDirMountPath, "project"),
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

func onnxAPISpec(api *spec.API, prevDeployment *kapps.Deployment) *kapps.Deployment {
	resourceList := kcore.ResourceList{}
	resourceLimitsList := kcore.ResourceList{}

	userPodCPURequest := api.Compute.CPU.Quantity.Copy()
	userPodCPURequest.Sub(_requestMonitorCPURequest)
	resourceList[kcore.ResourceCPU] = *userPodCPURequest

	if api.Compute.Mem != nil {
		userPodMemRequest := api.Compute.Mem.Quantity.Copy()
		userPodMemRequest.Sub(_requestMonitorMemRequest)
		resourceList[kcore.ResourceMemory] = *userPodMemRequest
	}

	servingImage := config.Cluster.ImagePythonServe
	if api.Compute.GPU > 0 {
		servingImage = config.Cluster.ImageONNXServeGPU
		resourceList["nvidia.com/gpu"] = *kresource.NewQuantity(api.Compute.GPU, kresource.DecimalSI)
		resourceLimitsList["nvidia.com/gpu"] = *kresource.NewQuantity(api.Compute.GPU, kresource.DecimalSI)
	}
	if api.Predictor.Image != "" {
		servingImage = api.Predictor.Image
	}

	return k8s.Deployment(&k8s.DeploymentSpec{
		Name:           k8sName(api.Name),
		Replicas:       getRequestedReplicasFromDeployment(api, prevDeployment),
		MaxSurge:       pointer.String(api.UpdateStrategy.MaxSurge),
		MaxUnavailable: pointer.String(api.UpdateStrategy.MaxUnavailable),
		Labels: map[string]string{
			"apiName":      api.Name,
			"apiID":        api.ID,
			"deploymentID": api.DeploymentID,
		},
		Annotations: api.Autoscaling.ToK8sAnnotations(),
		Selector: map[string]string{
			"apiName": api.Name,
		},
		PodSpec: k8s.PodSpec{
			Labels: map[string]string{
				"apiName":      api.Name,
				"apiID":        api.ID,
				"deploymentID": api.DeploymentID,
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
						Args:            []string{"--download=" + onnxDownloadArgs(api)},
						EnvFrom:         _baseEnvVars,
						VolumeMounts:    _defaultVolumeMounts,
					},
				},
				Containers: []kcore.Container{
					{
						Name:            _apiContainerName,
						Image:           servingImage,
						ImagePullPolicy: kcore.PullAlways,
						Env: append(
							getEnvVars(api),
							kcore.EnvVar{
								Name:  "CORTEX_MODEL_DIR",
								Value: path.Join(_emptyDirMountPath, "model"),
							},
						),
						EnvFrom:        _baseEnvVars,
						VolumeMounts:   _defaultVolumeMounts,
						ReadinessProbe: fileExistsProbe(_apiReadinessFile),
						LivenessProbe:  _apiReadinessProbe,
						Resources: kcore.ResourceRequirements{
							Requests: resourceList,
							Limits:   resourceLimitsList,
						},
						Ports: []kcore.ContainerPort{
							{ContainerPort: _defaultPortInt32},
						},
						SecurityContext: &kcore.SecurityContext{
							Privileged: pointer.Bool(true),
						},
					},
					*requestMonitorContainer(api),
				},
				NodeSelector: map[string]string{
					"workload": "true",
				},
				Tolerations:        _tolerations,
				Volumes:            _defaultVolumes,
				ServiceAccountName: "default",
			},
		},
	})
}

func onnxDownloadArgs(api *spec.API) string {
	downloadConfig := downloadContainerConfig{
		LastLog: fmt.Sprintf(_downloaderLastLog, "onnx"),
		DownloadArgs: []downloadContainerArg{
			{
				From:             aws.S3Path(config.Cluster.Bucket, api.ProjectKey),
				To:               path.Join(_emptyDirMountPath, "project"),
				Unzip:            true,
				ItemName:         "the project code",
				HideFromLog:      true,
				HideUnzippingLog: true,
			},
			{
				From:     *api.Predictor.Model,
				To:       path.Join(_emptyDirMountPath, "model"),
				ItemName: "the model",
			},
		},
	}

	downloadArgsBytes, _ := json.Marshal(downloadConfig)
	return base64.URLEncoding.EncodeToString(downloadArgsBytes)
}

func serviceSpec(api *spec.API) *kcore.Service {
	return k8s.Service(&k8s.ServiceSpec{
		Name:       k8sName(api.Name),
		Port:       _defaultPortInt32,
		TargetPort: _defaultPortInt32,
		Labels: map[string]string{
			"apiName": api.Name,
		},
		Selector: map[string]string{
			"apiName": api.Name,
		},
	})
}

func virtualServiceSpec(api *spec.API) *kunstructured.Unstructured {
	return k8s.VirtualService(&k8s.VirtualServiceSpec{
		Name:        k8sName(api.Name),
		Gateways:    []string{"apis-gateway"},
		ServiceName: k8sName(api.Name),
		ServicePort: _defaultPortInt32,
		Path:        *api.Endpoint,
		Rewrite:     pointer.String("predict"),
		Labels: map[string]string{
			"apiName": api.Name,
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

func getEnvVars(api *spec.API) []kcore.EnvVar {
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
		kcore.EnvVar{
			Name:  "CORTEX_WORKERS_PER_REPLICA",
			Value: s.Int32(api.Autoscaling.WorkersPerReplica),
		},
		kcore.EnvVar{
			Name:  "CORTEX_THREADS_PER_WORKER",
			Value: s.Int32(api.Autoscaling.ThreadsPerWorker),
		},
		kcore.EnvVar{
			Name:  "CORTEX_MAX_REPLICA_CONCURRENCY",
			Value: s.Int64(api.Autoscaling.MaxReplicaConcurrency),
		},
		kcore.EnvVar{
			Name: "CORTEX_MAX_WORKER_CONCURRENCY",
			// add 1 because it was required to achieve the target concurrency for 1 worker, 1 thread
			Value: s.Int64(1 + int64(math.Round(float64(api.Autoscaling.MaxReplicaConcurrency)/float64(api.Autoscaling.WorkersPerReplica)))),
		},
		kcore.EnvVar{
			Name:  "CORTEX_SO_MAX_CONN",
			Value: s.Int64(api.Autoscaling.MaxReplicaConcurrency + 100), // add a buffer to be safe
		},
		kcore.EnvVar{
			Name:  "CORTEX_SERVING_PORT",
			Value: _defaultPortStr,
		},
		kcore.EnvVar{
			Name:  "CORTEX_API_SPEC",
			Value: aws.S3Path(config.Cluster.Bucket, api.Key),
		},
		kcore.EnvVar{
			Name:  "CORTEX_CACHE_DIR",
			Value: _specCacheDir,
		},
		kcore.EnvVar{
			Name:  "CORTEX_PROJECT_DIR",
			Value: path.Join(_emptyDirMountPath, "project"),
		},
		kcore.EnvVar{
			Name:  "OVERRIDDEN_IMAGE",
			Value: api.Predictor.Image,
		},
	)

	if api.Predictor.PythonPath != nil {
		envVars = append(envVars, kcore.EnvVar{
			Name:  "PYTHON_PATH",
			Value: path.Join(_emptyDirMountPath, "project", *api.Predictor.PythonPath),
		})
	}

	return envVars
}

func requestMonitorContainer(api *spec.API) *kcore.Container {
	return &kcore.Container{
		Name:            "request-monitor",
		Image:           config.Cluster.ImageRequestMonitor,
		ImagePullPolicy: kcore.PullAlways,
		Args:            []string{api.Name, config.Cluster.LogGroup},
		EnvFrom:         _baseEnvVars,
		VolumeMounts:    _defaultVolumeMounts,
		ReadinessProbe:  fileExistsProbe(_requestMonitorReadinessFile),
		Resources: kcore.ResourceRequirements{
			Requests: kcore.ResourceList{
				kcore.ResourceCPU:    _requestMonitorCPURequest,
				kcore.ResourceMemory: _requestMonitorMemRequest,
			},
		},
	}
}

func k8sName(apiName string) string {
	return "api-" + apiName
}

var _apiReadinessProbe = &kcore.Probe{
	InitialDelaySeconds: 5,
	TimeoutSeconds:      5,
	PeriodSeconds:       5,
	SuccessThreshold:    1,
	FailureThreshold:    3,
	Handler: kcore.Handler{
		Exec: &kcore.ExecAction{
			Command: []string{"/bin/bash", "-c", `now="$(date +%s)" && min="$(($now-` + s.Int(_apiLivenessStalePeriod) + `))" && test "$(cat ` + _apiLivenessFile + ` | tr -d '[:space:]')" -ge "$min"`},
		},
	},
}

func fileExistsProbe(fileName string) *kcore.Probe {
	return &kcore.Probe{
		InitialDelaySeconds: 3,
		TimeoutSeconds:      5,
		PeriodSeconds:       5,
		SuccessThreshold:    1,
		FailureThreshold:    1,
		Handler: kcore.Handler{
			Exec: &kcore.ExecAction{
				Command: []string{"/bin/bash", "-c", fmt.Sprintf("test -f %s", fileName)},
			},
		},
	}
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
	k8s.EmptyDirVolume(_emptyDirVolumeName),
}

var _defaultVolumeMounts = []kcore.VolumeMount{
	k8s.EmptyDirVolumeMount(_emptyDirVolumeName, _emptyDirMountPath),
}
