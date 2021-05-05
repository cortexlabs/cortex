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

package workloads

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"path"
	"strings"

	"github.com/cortexlabs/cortex/pkg/config"
	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/types/clusterconfig"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	kcore "k8s.io/api/core/v1"
	kresource "k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	DefaultPortInt32               = int32(8888)
	DefaultPortStr                 = "8888"
	DefaultRequestMonitorPortStr   = "15000"
	DefaultRequestMonitorPortInt32 = int32(15000)
	APIContainerName               = "api"
	ServiceAccountName             = "default"
	APISpecPath                    = "/mnt/spec/spec.json"
	TaskSpecPath                   = "/mnt/spec/task.json"
	BatchSpecPath                  = "/mnt/spec/batch.json"
)

const (
	_clientConfigDir                               = "/mnt/client"
	_emptyDirMountPath                             = "/mnt"
	_emptyDirVolumeName                            = "mnt"
	_tfServingContainerName                        = "serve"
	_requestMonitorContainerName                   = "request-monitor"
	_gatewayContainerName                          = "gateway"
	_downloaderInitContainerName                   = "downloader"
	_downloaderLastLog                             = "downloading the %s serving image"
	_neuronRTDContainerName                        = "neuron-rtd"
	_tfBaseServingPortInt32, _tfBaseServingPortStr = int32(9000), "9000"
	_tfServingHost                                 = "localhost"
	_tfServingEmptyModelConfig                     = "/etc/tfs/model_config_server.conf"
	_tfServingMaxNumLoadRetries                    = "0"        // maximum retries to load a model that didn't get loaded the first time
	_tfServingLoadTimeMicros                       = "30000000" // 30 seconds (how much time a model can take to load into memory)
	_tfServingBatchConfig                          = "/etc/tfs/batch_config.conf"
	_apiReadinessFile                              = "/mnt/workspace/api_readiness.txt"
	_neuronRTDSocket                               = "/sock/neuron.sock"
	_requestMonitorReadinessFile                   = "/request_monitor_ready.txt"
	_kubexitInitContainerName                      = "kubexit"
	_kubexitGraveyardName                          = "graveyard"
	_kubexitGraveyardMountPath                     = "/graveyard"
)

var (
	_requestMonitorCPURequest = kresource.MustParse("10m")
	_requestMonitorMemRequest = kresource.MustParse("10Mi")

	_asyncGatewayCPURequest = kresource.MustParse("100m")
	_asyncGatewayMemRequest = kresource.MustParse("100Mi")

	// each Inferentia chip requires 128 HugePages with each HugePage having a size of 2Mi
	_hugePagesMemPerInf = int64(128 * 2 * 1024 * 1024) // bytes
)

func TaskInitContainer(api *spec.API, job *spec.TaskJob) kcore.Container {
	downloadConfig := downloadContainerConfig{
		LastLog: fmt.Sprintf(_downloaderLastLog, "task"),
		DownloadArgs: []downloadContainerArg{
			{
				From:             aws.S3Path(config.ClusterConfig.Bucket, api.ProjectKey),
				To:               path.Join(_emptyDirMountPath, "project"),
				Unzip:            true,
				ItemName:         "the project code",
				HideFromLog:      true,
				HideUnzippingLog: true,
			},
			{
				From:             aws.S3Path(config.ClusterConfig.Bucket, api.Key),
				To:               APISpecPath,
				Unzip:            false,
				ToFile:           true,
				ItemName:         "the api spec",
				HideFromLog:      true,
				HideUnzippingLog: true,
			},
			{
				From:             aws.S3Path(config.ClusterConfig.Bucket, job.SpecFilePath(config.ClusterConfig.ClusterName)),
				To:               TaskSpecPath,
				Unzip:            false,
				ToFile:           true,
				ItemName:         "the task spec",
				HideFromLog:      true,
				HideUnzippingLog: true,
			},
		},
	}

	downloadArgsBytes, _ := json.Marshal(downloadConfig)
	downloadArgs := base64.URLEncoding.EncodeToString(downloadArgsBytes)

	return kcore.Container{
		Name:            _downloaderInitContainerName,
		Image:           config.ClusterConfig.ImageDownloader,
		ImagePullPolicy: kcore.PullAlways,
		Args:            []string{"--download=" + downloadArgs},
		EnvFrom:         baseEnvVars(),
		Env:             downloaderEnvVars(api),
		VolumeMounts:    defaultVolumeMounts(),
	}
}

func KubexitInitContainer() kcore.Container {
	return kcore.Container{
		Name:            _kubexitInitContainerName,
		Image:           config.ClusterConfig.ImageKubexit,
		ImagePullPolicy: kcore.PullAlways,
		Command:         []string{"cp", "/bin/kubexit", "/mnt/kubexit"},
		VolumeMounts:    defaultVolumeMounts(),
	}
}

func BatchInitContainer(api *spec.API, job *spec.BatchJob) kcore.Container {
	downloadConfig := downloadContainerConfig{
		LastLog: fmt.Sprintf(_downloaderLastLog, api.Handler.Type.String()),
		DownloadArgs: []downloadContainerArg{
			{
				From:             aws.S3Path(config.ClusterConfig.Bucket, api.ProjectKey),
				To:               path.Join(_emptyDirMountPath, "project"),
				Unzip:            true,
				ItemName:         "the project code",
				HideFromLog:      true,
				HideUnzippingLog: true,
			},
			{
				From:             aws.S3Path(config.ClusterConfig.Bucket, api.Key),
				To:               APISpecPath,
				Unzip:            false,
				ToFile:           true,
				ItemName:         "the api spec",
				HideFromLog:      true,
				HideUnzippingLog: true,
			},
			{
				From:             aws.S3Path(config.ClusterConfig.Bucket, job.SpecFilePath(config.ClusterConfig.ClusterName)),
				To:               BatchSpecPath,
				Unzip:            false,
				ToFile:           true,
				ItemName:         "the job spec",
				HideFromLog:      true,
				HideUnzippingLog: true,
			},
		},
	}

	downloadArgsBytes, _ := json.Marshal(downloadConfig)
	downloadArgs := base64.URLEncoding.EncodeToString(downloadArgsBytes)

	return kcore.Container{
		Name:            _downloaderInitContainerName,
		Image:           config.ClusterConfig.ImageDownloader,
		ImagePullPolicy: kcore.PullAlways,
		Args:            []string{"--download=" + downloadArgs},
		EnvFrom:         baseEnvVars(),
		Env:             downloaderEnvVars(api),
		VolumeMounts:    defaultVolumeMounts(),
	}
}

// for async and realtime apis
func InitContainer(api *spec.API) kcore.Container {
	downloadConfig := downloadContainerConfig{
		LastLog: fmt.Sprintf(_downloaderLastLog, api.Handler.Type.String()),
		DownloadArgs: []downloadContainerArg{
			{
				From:             aws.S3Path(config.ClusterConfig.Bucket, api.ProjectKey),
				To:               path.Join(_emptyDirMountPath, "project"),
				Unzip:            true,
				ItemName:         "the project code",
				HideFromLog:      true,
				HideUnzippingLog: true,
			},
			{
				From:             aws.S3Path(config.ClusterConfig.Bucket, api.HandlerKey),
				To:               APISpecPath,
				Unzip:            false,
				ToFile:           true,
				ItemName:         "the api spec",
				HideFromLog:      true,
				HideUnzippingLog: true,
			},
		},
	}

	downloadArgsBytes, _ := json.Marshal(downloadConfig)
	downloadArgs := base64.URLEncoding.EncodeToString(downloadArgsBytes)

	return kcore.Container{
		Name:            _downloaderInitContainerName,
		Image:           config.ClusterConfig.ImageDownloader,
		ImagePullPolicy: kcore.PullAlways,
		Args:            []string{"--download=" + downloadArgs},
		EnvFrom:         baseEnvVars(),
		Env:             downloaderEnvVars(api),
		VolumeMounts:    defaultVolumeMounts(),
	}
}

func TaskContainers(api *spec.API) ([]kcore.Container, []kcore.Volume) {
	apiPodResourceList := kcore.ResourceList{}
	apiPodResourceLimitsList := kcore.ResourceList{}
	apiPodVolumeMounts := defaultVolumeMounts()
	volumes := DefaultVolumes()
	var containers []kcore.Container

	if api.Compute.GPU > 0 {
		apiPodResourceList["nvidia.com/gpu"] = *kresource.NewQuantity(api.Compute.GPU, kresource.DecimalSI)
		apiPodResourceLimitsList["nvidia.com/gpu"] = *kresource.NewQuantity(api.Compute.GPU, kresource.DecimalSI)
	} else if api.Compute.Inf > 0 {
		volumes = append(volumes, kcore.Volume{
			Name: "neuron-sock",
		})
		rtdVolumeMounts := []kcore.VolumeMount{
			k8s.EmptyDirVolumeMount(_emptyDirVolumeName, _emptyDirMountPath),
			{
				Name:      "neuron-sock",
				MountPath: "/sock",
			},
			{
				Name:      _kubexitGraveyardName,
				MountPath: _kubexitGraveyardMountPath,
			},
		}
		apiPodVolumeMounts = append(apiPodVolumeMounts, rtdVolumeMounts...)
		neuronContainer := neuronRuntimeDaemonContainer(api, rtdVolumeMounts, getKubexitEnvVars(_neuronRTDContainerName, APIContainerName))

		if api.Compute.CPU != nil {
			q1, q2 := k8s.SplitInTwo(k8s.QuantityPtr(api.Compute.CPU.Quantity.DeepCopy()))
			apiPodResourceList[kcore.ResourceCPU] = *q1
			neuronContainer.Resources.Requests[kcore.ResourceCPU] = *q2
		}

		if api.Compute.Mem != nil {
			q1, q2 := k8s.SplitInTwo(k8s.QuantityPtr(api.Compute.Mem.Quantity.DeepCopy()))
			apiPodResourceList[kcore.ResourceMemory] = *q1
			neuronContainer.Resources.Requests[kcore.ResourceMemory] = *q2
		}

		containers = append(containers, neuronContainer)
	} else {
		if api.Compute.CPU != nil {
			apiPodResourceList[kcore.ResourceCPU] = api.Compute.CPU.DeepCopy()
		}
		if api.Compute.Mem != nil {
			apiPodResourceList[kcore.ResourceMemory] = api.Compute.Mem.DeepCopy()
		}
	}

	if api.TaskDefinition.ShmSize != nil {
		volumes = append(volumes, kcore.Volume{
			Name: "dshm",
			VolumeSource: kcore.VolumeSource{
				EmptyDir: &kcore.EmptyDirVolumeSource{
					Medium:    kcore.StorageMediumMemory,
					SizeLimit: k8s.QuantityPtr(api.TaskDefinition.ShmSize.Quantity),
				},
			},
		})
		apiPodVolumeMounts = append(apiPodVolumeMounts, kcore.VolumeMount{
			Name:      "dshm",
			MountPath: "/dev/shm",
		})
	}

	containers = append(containers, kcore.Container{
		Name:            APIContainerName,
		Image:           api.TaskDefinition.Image,
		ImagePullPolicy: kcore.PullAlways,
		Env:             taskEnvVars(api),
		EnvFrom:         baseEnvVars(),
		VolumeMounts:    apiPodVolumeMounts,
		Resources: kcore.ResourceRequirements{
			Requests: apiPodResourceList,
			Limits:   apiPodResourceLimitsList,
		},
		Ports: []kcore.ContainerPort{
			{ContainerPort: DefaultPortInt32},
		},
		SecurityContext: &kcore.SecurityContext{
			Privileged: pointer.Bool(true),
		}},
	)

	return containers, volumes
}

func AsyncPythonHandlerContainers(api spec.API, queueURL string) ([]kcore.Container, []kcore.Volume) {
	return pythonHandlerContainers(&api, getAsyncAPIEnvVars(api, queueURL), false)
}

func AsyncTensorflowHandlerContainers(api spec.API, queueURL string) ([]kcore.Container, []kcore.Volume) {
	return tensorFlowHandlerContainers(&api, getAsyncAPIEnvVars(api, queueURL), false)
}

func AsyncGatewayContainers(api spec.API, queueURL string, volumeMounts []kcore.VolumeMount) kcore.Container {
	image := config.ClusterConfig.ImageAsyncGateway
	region := config.ClusterConfig.Region
	bucket := config.ClusterConfig.Bucket
	clusterName := config.ClusterConfig.ClusterName

	return kcore.Container{
		Name:            _gatewayContainerName,
		Image:           image,
		ImagePullPolicy: kcore.PullAlways,
		Args: []string{
			"-queue", queueURL,
			"-region", region,
			"-bucket", bucket,
			"-cluster", clusterName,
			"-port", s.Int32(DefaultPortInt32),
			api.Name,
		},
		Ports: []kcore.ContainerPort{
			{ContainerPort: DefaultPortInt32},
		},
		Env: []kcore.EnvVar{
			{
				Name:  "CORTEX_LOG_LEVEL",
				Value: strings.ToUpper(api.Handler.LogLevel.String()),
			},
		},
		Resources: kcore.ResourceRequirements{
			Requests: kcore.ResourceList{
				kcore.ResourceCPU:    _asyncGatewayCPURequest,
				kcore.ResourceMemory: _asyncGatewayMemRequest,
			},
		},
		LivenessProbe: &kcore.Probe{
			Handler: kcore.Handler{
				HTTPGet: &kcore.HTTPGetAction{
					Path: "/healthz",
					Port: intstr.FromInt(8888),
				},
			},
		},
		ReadinessProbe: &kcore.Probe{
			Handler: kcore.Handler{
				HTTPGet: &kcore.HTTPGetAction{
					Path: "/healthz",
					Port: intstr.FromInt(8888),
				},
			},
		},
		VolumeMounts: volumeMounts,
	}
}

func PythonHandlerContainers(api *spec.API) ([]kcore.Container, []kcore.Volume) {
	return pythonHandlerContainers(api, apiContainerEnvVars(api), false)
}

func PythonHandlerJobContainers(api *spec.API) ([]kcore.Container, []kcore.Volume) {
	return pythonHandlerContainers(api, apiContainerEnvVars(api), true)
}

func pythonHandlerContainers(api *spec.API, envVars []kcore.EnvVar, isJob bool) ([]kcore.Container, []kcore.Volume) {
	apiPodResourceList := kcore.ResourceList{}
	apiPodResourceLimitsList := kcore.ResourceList{}
	apiPodVolumeMounts := defaultVolumeMounts()
	volumes := DefaultVolumes()

	var containers []kcore.Container

	var neuronRTDEnvVars []kcore.EnvVar
	if isJob {
		envVars = append(envVars, getKubexitEnvVars(APIContainerName)...)
		volumes = append(volumes, k8s.EmptyDirVolume(_kubexitGraveyardName))
		apiPodVolumeMounts = append(apiPodVolumeMounts,
			kcore.VolumeMount{Name: _kubexitGraveyardName, MountPath: _kubexitGraveyardMountPath},
		)
		neuronRTDEnvVars = getKubexitEnvVars(_neuronRTDContainerName, APIContainerName)
	}

	if api.Compute.Inf == 0 {
		if api.Compute.CPU != nil {
			userPodCPURequest := k8s.QuantityPtr(api.Compute.CPU.Quantity.DeepCopy())
			if api.Kind == userconfig.RealtimeAPIKind {
				userPodCPURequest.Sub(_requestMonitorCPURequest)
			}
			apiPodResourceList[kcore.ResourceCPU] = *userPodCPURequest
		}

		if api.Compute.Mem != nil {
			userPodMemRequest := k8s.QuantityPtr(api.Compute.Mem.Quantity.DeepCopy())
			if api.Kind == userconfig.RealtimeAPIKind {
				userPodMemRequest.Sub(_requestMonitorMemRequest)
			}
			apiPodResourceList[kcore.ResourceMemory] = *userPodMemRequest
		}

		if api.Compute.GPU > 0 {
			apiPodResourceList["nvidia.com/gpu"] = *kresource.NewQuantity(api.Compute.GPU, kresource.DecimalSI)
			apiPodResourceLimitsList["nvidia.com/gpu"] = *kresource.NewQuantity(api.Compute.GPU, kresource.DecimalSI)
		}
	} else {
		volumes = append(volumes, kcore.Volume{
			Name: "neuron-sock",
		})
		rtdVolumeMounts := []kcore.VolumeMount{
			{
				Name:      "neuron-sock",
				MountPath: "/sock",
			},
		}

		apiPodVolumeMounts = append(apiPodVolumeMounts, rtdVolumeMounts...)

		if isJob {
			rtdVolumeMounts = append(rtdVolumeMounts,
				k8s.EmptyDirVolumeMount(_emptyDirVolumeName, _emptyDirMountPath),
				kcore.VolumeMount{Name: _kubexitGraveyardName, MountPath: _kubexitGraveyardMountPath},
			)
		}

		neuronContainer := neuronRuntimeDaemonContainer(api, rtdVolumeMounts, neuronRTDEnvVars)

		if api.Compute.CPU != nil {
			userPodCPURequest := k8s.QuantityPtr(api.Compute.CPU.Quantity.DeepCopy())
			if api.Kind == userconfig.RealtimeAPIKind {
				userPodCPURequest.Sub(_requestMonitorCPURequest)
			}
			q1, q2 := k8s.SplitInTwo(userPodCPURequest)
			apiPodResourceList[kcore.ResourceCPU] = *q1
			neuronContainer.Resources.Requests[kcore.ResourceCPU] = *q2
		}

		if api.Compute.Mem != nil {
			userPodMemRequest := k8s.QuantityPtr(api.Compute.Mem.Quantity.DeepCopy())
			if api.Kind == userconfig.RealtimeAPIKind {
				userPodMemRequest.Sub(_requestMonitorMemRequest)
			}
			q1, q2 := k8s.SplitInTwo(userPodMemRequest)
			apiPodResourceList[kcore.ResourceMemory] = *q1
			neuronContainer.Resources.Requests[kcore.ResourceMemory] = *q2
		}
		containers = append(containers, neuronContainer)
	}

	if api.Handler.ShmSize != nil {
		volumes = append(volumes, kcore.Volume{
			Name: "dshm",
			VolumeSource: kcore.VolumeSource{
				EmptyDir: &kcore.EmptyDirVolumeSource{
					Medium:    kcore.StorageMediumMemory,
					SizeLimit: k8s.QuantityPtr(api.Handler.ShmSize.Quantity),
				},
			},
		})
		apiPodVolumeMounts = append(apiPodVolumeMounts, kcore.VolumeMount{
			Name:      "dshm",
			MountPath: "/dev/shm",
		})
	}

	containers = append(containers, kcore.Container{
		Name:            APIContainerName,
		Image:           api.Handler.Image,
		ImagePullPolicy: kcore.PullAlways,
		Env:             envVars,
		EnvFrom:         baseEnvVars(),
		VolumeMounts:    apiPodVolumeMounts,
		ReadinessProbe:  FileExistsProbe(_apiReadinessFile),
		Lifecycle:       nginxGracefulStopper(api.Kind),
		Resources: kcore.ResourceRequirements{
			Requests: apiPodResourceList,
			Limits:   apiPodResourceLimitsList,
		},
		Ports: []kcore.ContainerPort{
			{ContainerPort: DefaultPortInt32},
		},
		SecurityContext: &kcore.SecurityContext{
			Privileged: pointer.Bool(true),
		}},
	)

	return containers, volumes
}

func TensorFlowHandlerContainers(api *spec.API) ([]kcore.Container, []kcore.Volume) {
	return tensorFlowHandlerContainers(api, apiContainerEnvVars(api), false)
}

func TensorFlowHandlerJobContainers(api *spec.API) ([]kcore.Container, []kcore.Volume) {
	return tensorFlowHandlerContainers(api, apiContainerEnvVars(api), true)
}

func tensorFlowHandlerContainers(api *spec.API, envVars []kcore.EnvVar, isJob bool) ([]kcore.Container, []kcore.Volume) {
	apiResourceList := kcore.ResourceList{}
	tfServingResourceList := kcore.ResourceList{}
	tfServingLimitsList := kcore.ResourceList{}
	volumeMounts := defaultVolumeMounts()
	volumes := DefaultVolumes()

	var neuronRTDEnvVars []kcore.EnvVar
	tfServingEnvVars := tensorflowServingEnvVars(api)
	if isJob {
		envVars = append(envVars, getKubexitEnvVars(APIContainerName)...)
		tfServingEnvVars = append(tfServingEnvVars, getKubexitEnvVars(_tfServingContainerName, APIContainerName)...)

		volumes = append(volumes, k8s.EmptyDirVolume(_kubexitGraveyardName))
		volumeMounts = append(volumeMounts,
			kcore.VolumeMount{Name: _kubexitGraveyardName, MountPath: _kubexitGraveyardMountPath},
		)
		neuronRTDEnvVars = getKubexitEnvVars(_neuronRTDContainerName, APIContainerName)
	}

	var containers []kcore.Container
	if api.Compute.Inf == 0 {
		if api.Compute.CPU != nil {
			userPodCPURequest := k8s.QuantityPtr(api.Compute.CPU.Quantity.DeepCopy())
			if api.Kind == userconfig.RealtimeAPIKind {
				userPodCPURequest.Sub(_requestMonitorCPURequest)
			}
			q1, q2 := k8s.SplitInTwo(userPodCPURequest)
			apiResourceList[kcore.ResourceCPU] = *q1
			tfServingResourceList[kcore.ResourceCPU] = *q2
		}

		if api.Compute.Mem != nil {
			userPodMemRequest := k8s.QuantityPtr(api.Compute.Mem.Quantity.DeepCopy())
			if api.Kind == userconfig.RealtimeAPIKind {
				userPodMemRequest.Sub(_requestMonitorMemRequest)
			}
			q1, q2 := k8s.SplitInTwo(userPodMemRequest)
			apiResourceList[kcore.ResourceMemory] = *q1
			tfServingResourceList[kcore.ResourceMemory] = *q2
		}

		if api.Compute.GPU > 0 {
			tfServingResourceList["nvidia.com/gpu"] = *kresource.NewQuantity(api.Compute.GPU, kresource.DecimalSI)
			tfServingLimitsList["nvidia.com/gpu"] = *kresource.NewQuantity(api.Compute.GPU, kresource.DecimalSI)
		}
	} else {
		volumes = append(volumes, kcore.Volume{
			Name: "neuron-sock",
		})
		rtdVolumeMounts := []kcore.VolumeMount{
			{
				Name:      "neuron-sock",
				MountPath: "/sock",
			},
		}
		volumeMounts = append(volumeMounts, rtdVolumeMounts...)

		if isJob {
			rtdVolumeMounts = append(rtdVolumeMounts,
				k8s.EmptyDirVolumeMount(_emptyDirVolumeName, _emptyDirMountPath),
				kcore.VolumeMount{Name: _kubexitGraveyardName, MountPath: _kubexitGraveyardMountPath},
			)
		}

		neuronContainer := neuronRuntimeDaemonContainer(api, rtdVolumeMounts, neuronRTDEnvVars)

		if api.Compute.CPU != nil {
			userPodCPURequest := k8s.QuantityPtr(api.Compute.CPU.Quantity.DeepCopy())
			if api.Kind == userconfig.RealtimeAPIKind {
				userPodCPURequest.Sub(_requestMonitorCPURequest)
			}
			q1, q2, q3 := k8s.SplitInThree(userPodCPURequest)
			apiResourceList[kcore.ResourceCPU] = *q1
			tfServingResourceList[kcore.ResourceCPU] = *q2
			neuronContainer.Resources.Requests[kcore.ResourceCPU] = *q3
		}

		if api.Compute.Mem != nil {
			userPodMemRequest := k8s.QuantityPtr(api.Compute.Mem.Quantity.DeepCopy())
			if api.Kind == userconfig.RealtimeAPIKind {
				userPodMemRequest.Sub(_requestMonitorMemRequest)
			}
			q1, q2, q3 := k8s.SplitInThree(userPodMemRequest)
			apiResourceList[kcore.ResourceMemory] = *q1
			tfServingResourceList[kcore.ResourceMemory] = *q2
			neuronContainer.Resources.Requests[kcore.ResourceMemory] = *q3
		}

		containers = append(containers, neuronContainer)
	}

	if api.Handler.ShmSize != nil {
		volumes = append(volumes, kcore.Volume{
			Name: "dshm",
			VolumeSource: kcore.VolumeSource{
				EmptyDir: &kcore.EmptyDirVolumeSource{
					Medium:    kcore.StorageMediumMemory,
					SizeLimit: k8s.QuantityPtr(api.Handler.ShmSize.Quantity),
				},
			},
		})
		volumeMounts = append(volumeMounts, kcore.VolumeMount{
			Name:      "dshm",
			MountPath: "/dev/shm",
		})
	}

	containers = append(containers, kcore.Container{
		Name:            APIContainerName,
		Image:           api.Handler.Image,
		ImagePullPolicy: kcore.PullAlways,
		Env:             envVars,
		EnvFrom:         baseEnvVars(),
		VolumeMounts:    volumeMounts,
		ReadinessProbe:  FileExistsProbe(_apiReadinessFile),
		Lifecycle:       nginxGracefulStopper(api.Kind),
		Resources: kcore.ResourceRequirements{
			Requests: apiResourceList,
		},
		Ports: []kcore.ContainerPort{
			{ContainerPort: DefaultPortInt32},
		},
		SecurityContext: &kcore.SecurityContext{
			Privileged: pointer.Bool(true),
		}},
		tensorflowServingContainer(
			api,
			volumeMounts,
			kcore.ResourceRequirements{
				Limits:   tfServingLimitsList,
				Requests: tfServingResourceList,
			},
			tfServingEnvVars,
		),
	)

	return containers, volumes
}

func taskEnvVars(api *spec.API) []kcore.EnvVar {
	envVars := apiContainerEnvVars(api)
	envVars = append(envVars,

		kcore.EnvVar{
			Name:  "CORTEX_TASK_SPEC",
			Value: TaskSpecPath,
		},
	)
	return envVars
}

func getAsyncAPIEnvVars(api spec.API, queueURL string) []kcore.EnvVar {
	envVars := apiContainerEnvVars(&api)

	envVars = append(envVars,
		kcore.EnvVar{
			Name:  "CORTEX_QUEUE_URL",
			Value: queueURL,
		},
		kcore.EnvVar{
			Name:  "CORTEX_ASYNC_WORKLOAD_PATH",
			Value: aws.S3Path(config.ClusterConfig.Bucket, fmt.Sprintf("%s/apis/%s/workloads", config.ClusterConfig.ClusterName, api.Name)),
		},
	)

	return envVars
}

func requestMonitorEnvVars(api *spec.API) []kcore.EnvVar {
	if api.Kind == userconfig.TaskAPIKind {
		return []kcore.EnvVar{
			{
				Name:  "CORTEX_LOG_LEVEL",
				Value: strings.ToUpper(api.TaskDefinition.LogLevel.String()),
			},
		}
	}
	return []kcore.EnvVar{
		{
			Name:  "CORTEX_LOG_LEVEL",
			Value: strings.ToUpper(api.Handler.LogLevel.String()),
		},
	}
}

func downloaderEnvVars(api *spec.API) []kcore.EnvVar {
	if api.Kind == userconfig.TaskAPIKind {
		return []kcore.EnvVar{
			{
				Name:  "CORTEX_LOG_LEVEL",
				Value: strings.ToUpper(api.TaskDefinition.LogLevel.String()),
			},
		}
	}
	return []kcore.EnvVar{
		{
			Name:  "CORTEX_LOG_LEVEL",
			Value: strings.ToUpper(api.Handler.LogLevel.String()),
		},
	}
}

func tensorflowServingEnvVars(api *spec.API) []kcore.EnvVar {
	envVars := []kcore.EnvVar{
		{
			Name:  "TF_CPP_MIN_LOG_LEVEL",
			Value: s.Int(userconfig.TFNumericLogLevelFromLogLevel(api.Handler.LogLevel)),
		},
		{
			Name:  "TF_PROCESSES",
			Value: s.Int32(api.Handler.ProcessesPerReplica),
		},
		{
			Name:  "CORTEX_TF_BASE_SERVING_PORT",
			Value: _tfBaseServingPortStr,
		},
		{
			Name:  "TF_EMPTY_MODEL_CONFIG",
			Value: _tfServingEmptyModelConfig,
		},
		{
			Name:  "TF_MAX_NUM_LOAD_RETRIES",
			Value: _tfServingMaxNumLoadRetries,
		},
		{
			Name:  "TF_LOAD_RETRY_INTERVAL_MICROS",
			Value: _tfServingLoadTimeMicros,
		},
		{
			Name:  "TF_GRPC_MAX_CONCURRENT_STREAMS",
			Value: fmt.Sprintf(`--grpc_channel_arguments="grpc.max_concurrent_streams=%d"`, api.Handler.ThreadsPerProcess+10),
		},
	}

	if api.Handler.ServerSideBatching != nil {
		var numBatchedThreads int32
		if api.Compute.Inf > 0 {
			// because there are processes_per_replica TF servers
			numBatchedThreads = 1
		} else {
			numBatchedThreads = api.Handler.ProcessesPerReplica
		}

		envVars = append(envVars,
			kcore.EnvVar{
				Name:  "TF_MAX_BATCH_SIZE",
				Value: s.Int32(api.Handler.ServerSideBatching.MaxBatchSize),
			},
			kcore.EnvVar{
				Name:  "TF_BATCH_TIMEOUT_MICROS",
				Value: s.Int64(api.Handler.ServerSideBatching.BatchInterval.Microseconds()),
			},
			kcore.EnvVar{
				Name:  "TF_NUM_BATCHED_THREADS",
				Value: s.Int32(numBatchedThreads),
			},
		)
	}

	if api.Compute.Inf > 0 {
		envVars = append(envVars,
			kcore.EnvVar{
				Name:  "NEURONCORE_GROUP_SIZES",
				Value: s.Int64(api.Compute.Inf * consts.NeuronCoresPerInf / int64(api.Handler.ProcessesPerReplica)),
			},
			kcore.EnvVar{
				Name:  "NEURON_RTD_ADDRESS",
				Value: fmt.Sprintf("unix:%s", _neuronRTDSocket),
			},
		)
	}

	return envVars
}

func apiContainerEnvVars(api *spec.API) []kcore.EnvVar {
	envVars := []kcore.EnvVar{
		{
			Name:  "CORTEX_TELEMETRY_SENTRY_USER_ID",
			Value: config.OperatorMetadata.OperatorID,
		},
		{
			Name:  "CORTEX_TELEMETRY_SENTRY_ENVIRONMENT",
			Value: "api",
		},
		{
			Name:  "CORTEX_DEBUGGING",
			Value: "false",
		},
		{
			Name: "HOST_IP",
			ValueFrom: &kcore.EnvVarSource{
				FieldRef: &kcore.ObjectFieldSelector{
					FieldPath: "status.hostIP",
				},
			},
		},
		{
			Name:  "CORTEX_API_SPEC",
			Value: APISpecPath,
		},
	}

	if api.Handler != nil {
		for name, val := range api.Handler.Env {
			envVars = append(envVars, kcore.EnvVar{
				Name:  name,
				Value: val,
			})
		}
		if api.Handler.Type == userconfig.TensorFlowHandlerType {
			envVars = append(envVars,
				kcore.EnvVar{
					Name:  "CORTEX_TF_BASE_SERVING_PORT",
					Value: _tfBaseServingPortStr,
				},
				kcore.EnvVar{
					Name:  "CORTEX_TF_SERVING_HOST",
					Value: _tfServingHost,
				},
			)
		}
	}

	if api.TaskDefinition != nil {
		for name, val := range api.TaskDefinition.Env {
			envVars = append(envVars, kcore.EnvVar{
				Name:  name,
				Value: val,
			})
		}
	}

	return envVars
}

func tensorflowServingContainer(api *spec.API, volumeMounts []kcore.VolumeMount, resources kcore.ResourceRequirements, envVars []kcore.EnvVar) kcore.Container {
	var cmdArgs []string
	ports := []kcore.ContainerPort{
		{
			ContainerPort: _tfBaseServingPortInt32,
		},
	}

	if api.Compute.Inf > 0 {
		numPorts := api.Handler.ProcessesPerReplica
		for i := int32(1); i < numPorts; i++ {
			ports = append(ports, kcore.ContainerPort{
				ContainerPort: _tfBaseServingPortInt32 + i,
			})
		}
	}

	if api.Compute.Inf == 0 {
		// the entrypoint is different for Inferentia-based APIs
		cmdArgs = []string{
			"--port=" + _tfBaseServingPortStr,
			"--model_config_file=" + _tfServingEmptyModelConfig,
			"--max_num_load_retries=" + _tfServingMaxNumLoadRetries,
			"--load_retry_interval_micros=" + _tfServingLoadTimeMicros,
			fmt.Sprintf(`--grpc_channel_arguments="grpc.max_concurrent_streams=%d"`, api.Handler.ProcessesPerReplica*api.Handler.ThreadsPerProcess+10),
		}
		if api.Handler.ServerSideBatching != nil {
			cmdArgs = append(cmdArgs,
				"--enable_batching=true",
				"--batching_parameters_file="+_tfServingBatchConfig,
			)
		}
	}

	var probeHandler kcore.Handler
	if len(ports) == 1 {
		probeHandler = kcore.Handler{
			TCPSocket: &kcore.TCPSocketAction{
				Port: intstr.IntOrString{
					IntVal: _tfBaseServingPortInt32,
				},
			},
		}
	} else {
		probeHandler = kcore.Handler{
			Exec: &kcore.ExecAction{
				Command: []string{"/bin/bash", "-c", `test $(nc -zv localhost ` + fmt.Sprintf("%d-%d", _tfBaseServingPortInt32, _tfBaseServingPortInt32+int32(len(ports))-1) + ` 2>&1 | wc -l) -eq ` + fmt.Sprintf("%d", len(ports))},
			},
		}
	}

	return kcore.Container{
		Name:            _tfServingContainerName,
		Image:           api.Handler.TensorFlowServingImage,
		ImagePullPolicy: kcore.PullAlways,
		Args:            cmdArgs,
		Env:             envVars,
		EnvFrom:         baseEnvVars(),
		VolumeMounts:    volumeMounts,
		ReadinessProbe: &kcore.Probe{
			InitialDelaySeconds: 5,
			TimeoutSeconds:      5,
			PeriodSeconds:       5,
			SuccessThreshold:    1,
			FailureThreshold:    2,
			Handler:             probeHandler,
		},
		Lifecycle: waitAPIContainerToStop(api.Kind),
		Resources: resources,
		Ports:     ports,
	}
}

func neuronRuntimeDaemonContainer(api *spec.API, volumeMounts []kcore.VolumeMount, envVars []kcore.EnvVar) kcore.Container {
	totalHugePages := api.Compute.Inf * _hugePagesMemPerInf
	return kcore.Container{
		Name:            _neuronRTDContainerName,
		Image:           config.ClusterConfig.ImageNeuronRTD,
		ImagePullPolicy: kcore.PullAlways,
		Env:             envVars,
		SecurityContext: &kcore.SecurityContext{
			Capabilities: &kcore.Capabilities{
				Add: []kcore.Capability{
					"SYS_ADMIN",
					"IPC_LOCK",
				},
			},
		},
		VolumeMounts:   volumeMounts,
		ReadinessProbe: socketExistsProbe(_neuronRTDSocket),
		Lifecycle:      waitAPIContainerToStop(api.Kind),
		Resources: kcore.ResourceRequirements{
			Requests: kcore.ResourceList{
				"hugepages-2Mi":         *kresource.NewQuantity(totalHugePages, kresource.BinarySI),
				"aws.amazon.com/neuron": *kresource.NewQuantity(api.Compute.Inf, kresource.DecimalSI),
			},
			Limits: kcore.ResourceList{
				"hugepages-2Mi":         *kresource.NewQuantity(totalHugePages, kresource.BinarySI),
				"aws.amazon.com/neuron": *kresource.NewQuantity(api.Compute.Inf, kresource.DecimalSI),
			},
		},
	}
}

func RequestMonitorContainer(api *spec.API) kcore.Container {
	requests := kcore.ResourceList{}
	if api.Compute != nil {
		if api.Compute.CPU != nil {
			requests[kcore.ResourceCPU] = _requestMonitorCPURequest
		}
		if api.Compute.Mem != nil {
			requests[kcore.ResourceMemory] = _requestMonitorMemRequest
		}
	}

	return kcore.Container{
		Name:            _requestMonitorContainerName,
		Image:           config.ClusterConfig.ImageRequestMonitor,
		ImagePullPolicy: kcore.PullAlways,
		Args:            []string{"-p", DefaultRequestMonitorPortStr},
		Ports: []kcore.ContainerPort{
			{Name: "metrics", ContainerPort: DefaultRequestMonitorPortInt32},
		},
		Env:            requestMonitorEnvVars(api),
		EnvFrom:        baseEnvVars(),
		VolumeMounts:   defaultVolumeMounts(),
		ReadinessProbe: FileExistsProbe(_requestMonitorReadinessFile),
		Resources: kcore.ResourceRequirements{
			Requests: requests,
		},
	}
}

func baseEnvVars() []kcore.EnvFromSource {
	envVars := []kcore.EnvFromSource{
		{
			ConfigMapRef: &kcore.ConfigMapEnvSource{
				LocalObjectReference: kcore.LocalObjectReference{
					Name: "env-vars",
				},
			},
		},
	}

	return envVars
}

func getKubexitEnvVars(containerName string, deathDeps ...string) []kcore.EnvVar {
	envVars := []kcore.EnvVar{
		{
			Name:  "KUBEXIT_NAME",
			Value: containerName,
		},
		{
			Name:  "KUBEXIT_GRAVEYARD",
			Value: _kubexitGraveyardMountPath,
		},
	}

	if deathDeps != nil {
		envVars = append(envVars,
			kcore.EnvVar{
				Name:  "KUBEXIT_DEATH_DEPS",
				Value: strings.Join(deathDeps, ","),
			},
			kcore.EnvVar{
				Name:  "KUBEXIT_IGNORE_CODE_ON_DEATH_DEPS",
				Value: "true",
			},
		)
	}

	return envVars
}

func DefaultVolumes() []kcore.Volume {
	return []kcore.Volume{
		k8s.EmptyDirVolume(_emptyDirVolumeName),
		{
			Name: "client-config",
			VolumeSource: kcore.VolumeSource{
				ConfigMap: &kcore.ConfigMapVolumeSource{
					LocalObjectReference: kcore.LocalObjectReference{
						Name: "client-config",
					},
				},
			},
		},
	}
}

func defaultVolumeMounts() []kcore.VolumeMount {
	return []kcore.VolumeMount{
		k8s.EmptyDirVolumeMount(_emptyDirVolumeName, _emptyDirMountPath),
		{
			Name:      "client-config",
			MountPath: path.Join(_clientConfigDir, "cli.yaml"),
			SubPath:   "cli.yaml",
		},
	}
}

func NodeSelectors() map[string]string {
	return map[string]string{
		"workload": "true",
	}
}

func GenerateResourceTolerations() []kcore.Toleration {
	tolerations := []kcore.Toleration{
		{
			Key:      "workload",
			Operator: kcore.TolerationOpEqual,
			Value:    "true",
			Effect:   kcore.TaintEffectNoSchedule,
		},
		{
			Key:      "nvidia.com/gpu",
			Operator: kcore.TolerationOpExists,
			Effect:   kcore.TaintEffectNoSchedule,
		},
		{
			Key:      "aws.amazon.com/neuron",
			Operator: kcore.TolerationOpEqual,
			Value:    "true",
			Effect:   kcore.TaintEffectNoSchedule,
		},
	}

	return tolerations
}

func GenerateNodeAffinities(apiNodeGroups []string) *kcore.Affinity {
	// node groups are ordered according to how the cluster config node groups are ordered
	var nodeGroups []*clusterconfig.NodeGroup
	for _, clusterNodeGroup := range config.ClusterConfig.NodeGroups {
		for _, apiNodeGroupName := range apiNodeGroups {
			if clusterNodeGroup.Name == apiNodeGroupName {
				nodeGroups = append(nodeGroups, clusterNodeGroup)
			}
		}
	}

	numNodeGroups := len(apiNodeGroups)
	if apiNodeGroups == nil {
		nodeGroups = config.ClusterConfig.NodeGroups
		numNodeGroups = len(config.ClusterConfig.NodeGroups)
	}

	var requiredNodeGroups []string
	var preferredAffinities []kcore.PreferredSchedulingTerm

	for idx, nodeGroup := range nodeGroups {
		var nodeGroupPrefix string
		if nodeGroup.Spot {
			nodeGroupPrefix = "cx-ws-"
		} else {
			nodeGroupPrefix = "cx-wd-"
		}

		preferredAffinities = append(preferredAffinities, kcore.PreferredSchedulingTerm{
			Weight: int32(100 * (1 - float64(idx)/float64(numNodeGroups))),
			Preference: kcore.NodeSelectorTerm{
				MatchExpressions: []kcore.NodeSelectorRequirement{
					{
						Key:      "alpha.eksctl.io/nodegroup-name",
						Operator: kcore.NodeSelectorOpIn,
						Values:   []string{nodeGroupPrefix + nodeGroup.Name},
					},
				},
			},
		})
		requiredNodeGroups = append(requiredNodeGroups, nodeGroupPrefix+nodeGroup.Name)
	}

	var requiredNodeSelector *kcore.NodeSelector
	if apiNodeGroups != nil {
		requiredNodeSelector = &kcore.NodeSelector{
			NodeSelectorTerms: []kcore.NodeSelectorTerm{
				{
					MatchExpressions: []kcore.NodeSelectorRequirement{
						{
							Key:      "alpha.eksctl.io/nodegroup-name",
							Operator: kcore.NodeSelectorOpIn,
							Values:   requiredNodeGroups,
						},
					},
				},
			},
		}
	}

	return &kcore.Affinity{
		NodeAffinity: &kcore.NodeAffinity{
			PreferredDuringSchedulingIgnoredDuringExecution: preferredAffinities,
			RequiredDuringSchedulingIgnoredDuringExecution:  requiredNodeSelector,
		},
	}
}
