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

package operator

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"path"
	"strings"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/lib/urls"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/types"
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
)

const (
	_specCacheDir                                  = "/mnt/spec"
	_modelDir                                      = "/mnt/model"
	_emptyDirMountPath                             = "/mnt"
	_emptyDirVolumeName                            = "mnt"
	_tfServingContainerName                        = "serve"
	_requestMonitorContainerName                   = "request-monitor"
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
)

var (
	_requestMonitorCPURequest = kresource.MustParse("10m")
	_requestMonitorMemRequest = kresource.MustParse("10Mi")

	// each Inferentia chip requires 128 HugePages with each HugePage having a size of 2Mi
	_hugePagesMemPerInf = int64(128 * 2 * 1024 * 1024) // bytes
)

type downloadContainerConfig struct {
	DownloadArgs []downloadContainerArg `json:"download_args"`
	LastLog      string                 `json:"last_log"` // string to log at the conclusion of the downloader (if "" nothing will be logged)
}

type downloadContainerArg struct {
	From             string `json:"from"`
	To               string `json:"to"`
	ToFile           bool   `json:"to_file"` // whether "To" path reflects the path to a file or just the directory in which "From" object is copied to
	Unzip            bool   `json:"unzip"`
	ItemName         string `json:"item_name"`          // name of the item being downloaded, just for logging (if "" nothing will be logged)
	HideFromLog      bool   `json:"hide_from_log"`      // if true, don't log where the file is being downloaded from
	HideUnzippingLog bool   `json:"hide_unzipping_log"` // if true, don't log when unzipping
}

func TaskInitContainer(api *spec.API) kcore.Container {
	return kcore.Container{
		Name:            _downloaderInitContainerName,
		Image:           config.ImageDownloader(),
		ImagePullPolicy: "Always",
		Args:            []string{"--download=" + pythonDownloadArgs(api)},
		EnvFrom:         baseEnvVars(),
		Env: []kcore.EnvVar{
			{
				Name:  "CORTEX_LOG_LEVEL",
				Value: strings.ToUpper(api.TaskDefinition.LogLevel.String()),
			},
		},
		VolumeMounts: defaultVolumeMounts(),
	}
}

func InitContainer(api *spec.API) kcore.Container {
	downloadArgs := ""

	switch api.Predictor.Type {
	case userconfig.TensorFlowPredictorType:
		downloadArgs = tfDownloadArgs(api)
	case userconfig.ONNXPredictorType:
		downloadArgs = onnxDownloadArgs(api)
	case userconfig.PythonPredictorType:
		downloadArgs = pythonDownloadArgs(api)
	}

	return kcore.Container{
		Name:            _downloaderInitContainerName,
		Image:           config.ImageDownloader(),
		ImagePullPolicy: "Always",
		Args:            []string{"--download=" + downloadArgs},
		EnvFrom:         baseEnvVars(),
		Env:             getEnvVars(api, _downloaderInitContainerName),
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
			{
				Name:      "neuron-sock",
				MountPath: "/sock",
			},
		}
		apiPodVolumeMounts = append(apiPodVolumeMounts, rtdVolumeMounts...)
		neuronContainer := *neuronRuntimeDaemonContainer(api, rtdVolumeMounts)

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

	containers = append(containers, kcore.Container{
		Name:            APIContainerName,
		Image:           api.TaskDefinition.Image,
		ImagePullPolicy: kcore.PullAlways,
		Env:             getTaskEnvVars(api, APIContainerName),
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

func PythonPredictorContainers(api *spec.API) ([]kcore.Container, []kcore.Volume) {
	apiPodResourceList := kcore.ResourceList{}
	apiPodResourceLimitsList := kcore.ResourceList{}
	apiPodVolumeMounts := defaultVolumeMounts()
	volumes := DefaultVolumes()
	var containers []kcore.Container

	if api.Compute.Inf == 0 {
		if api.Compute.CPU != nil {
			userPodCPURequest := k8s.QuantityPtr(api.Compute.CPU.Quantity.DeepCopy())
			userPodCPURequest.Sub(_requestMonitorCPURequest)
			apiPodResourceList[kcore.ResourceCPU] = *userPodCPURequest
		}

		if api.Compute.Mem != nil {
			userPodMemRequest := k8s.QuantityPtr(api.Compute.Mem.Quantity.DeepCopy())
			userPodMemRequest.Sub(_requestMonitorMemRequest)
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
		neuronContainer := *neuronRuntimeDaemonContainer(api, rtdVolumeMounts)

		if api.Compute.CPU != nil {
			userPodCPURequest := k8s.QuantityPtr(api.Compute.CPU.Quantity.DeepCopy())
			userPodCPURequest.Sub(_requestMonitorCPURequest)
			q1, q2 := k8s.SplitInTwo(userPodCPURequest)
			apiPodResourceList[kcore.ResourceCPU] = *q1
			neuronContainer.Resources.Requests[kcore.ResourceCPU] = *q2
		}

		if api.Compute.Mem != nil {
			userPodMemRequest := k8s.QuantityPtr(api.Compute.Mem.Quantity.DeepCopy())
			userPodMemRequest.Sub(_requestMonitorMemRequest)
			q1, q2 := k8s.SplitInTwo(userPodMemRequest)
			apiPodResourceList[kcore.ResourceMemory] = *q1
			neuronContainer.Resources.Requests[kcore.ResourceMemory] = *q2
		}

		containers = append(containers, neuronContainer)
	}

	if api.Predictor.ShmSize != nil {
		volumes = append(volumes, kcore.Volume{
			Name: "dshm",
			VolumeSource: kcore.VolumeSource{
				EmptyDir: &kcore.EmptyDirVolumeSource{
					Medium:    kcore.StorageMediumMemory,
					SizeLimit: k8s.QuantityPtr(api.Predictor.ShmSize.Quantity),
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
		Image:           api.Predictor.Image,
		ImagePullPolicy: kcore.PullAlways,
		Env:             getEnvVars(api, APIContainerName),
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

func TensorFlowPredictorContainers(api *spec.API) ([]kcore.Container, []kcore.Volume) {
	apiResourceList := kcore.ResourceList{}
	tfServingResourceList := kcore.ResourceList{}
	tfServingLimitsList := kcore.ResourceList{}
	volumeMounts := defaultVolumeMounts()
	volumes := DefaultVolumes()
	var containers []kcore.Container

	if api.Compute.Inf == 0 {
		if api.Compute.CPU != nil {
			userPodCPURequest := k8s.QuantityPtr(api.Compute.CPU.Quantity.DeepCopy())
			userPodCPURequest.Sub(_requestMonitorCPURequest)
			q1, q2 := k8s.SplitInTwo(userPodCPURequest)
			apiResourceList[kcore.ResourceCPU] = *q1
			tfServingResourceList[kcore.ResourceCPU] = *q2
		}

		if api.Compute.Mem != nil {
			userPodMemRequest := k8s.QuantityPtr(api.Compute.Mem.Quantity.DeepCopy())
			userPodMemRequest.Sub(_requestMonitorMemRequest)
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

		neuronContainer := *neuronRuntimeDaemonContainer(api, rtdVolumeMounts)

		if api.Compute.CPU != nil {
			userPodCPURequest := k8s.QuantityPtr(api.Compute.CPU.Quantity.DeepCopy())
			userPodCPURequest.Sub(_requestMonitorCPURequest)
			q1, q2, q3 := k8s.SplitInThree(userPodCPURequest)
			apiResourceList[kcore.ResourceCPU] = *q1
			tfServingResourceList[kcore.ResourceCPU] = *q2
			neuronContainer.Resources.Requests[kcore.ResourceCPU] = *q3
		}

		if api.Compute.Mem != nil {
			userPodMemRequest := k8s.QuantityPtr(api.Compute.Mem.Quantity.DeepCopy())
			userPodMemRequest.Sub(_requestMonitorMemRequest)
			q1, q2, q3 := k8s.SplitInThree(userPodMemRequest)
			apiResourceList[kcore.ResourceMemory] = *q1
			tfServingResourceList[kcore.ResourceMemory] = *q2
			neuronContainer.Resources.Requests[kcore.ResourceMemory] = *q3
		}

		containers = append(containers, neuronContainer)
	}

	if api.Predictor.ShmSize != nil {
		volumes = append(volumes, kcore.Volume{
			Name: "dshm",
			VolumeSource: kcore.VolumeSource{
				EmptyDir: &kcore.EmptyDirVolumeSource{
					Medium:    kcore.StorageMediumMemory,
					SizeLimit: k8s.QuantityPtr(api.Predictor.ShmSize.Quantity),
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
		Image:           api.Predictor.Image,
		ImagePullPolicy: kcore.PullAlways,
		Env:             getEnvVars(api, APIContainerName),
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
		*tensorflowServingContainer(
			api,
			volumeMounts,
			kcore.ResourceRequirements{
				Limits:   tfServingLimitsList,
				Requests: tfServingResourceList,
			},
		),
	)

	return containers, volumes
}

func ONNXPredictorContainers(api *spec.API) ([]kcore.Container, []kcore.Volume) {
	resourceList := kcore.ResourceList{}
	resourceLimitsList := kcore.ResourceList{}
	apiPodVolumeMounts := defaultVolumeMounts()
	volumes := DefaultVolumes()
	var containers []kcore.Container

	if api.Compute.CPU != nil {
		userPodCPURequest := k8s.QuantityPtr(api.Compute.CPU.Quantity.DeepCopy())
		userPodCPURequest.Sub(_requestMonitorCPURequest)
		resourceList[kcore.ResourceCPU] = *userPodCPURequest
	}

	if api.Compute.Mem != nil {
		userPodMemRequest := k8s.QuantityPtr(api.Compute.Mem.Quantity.DeepCopy())
		userPodMemRequest.Sub(_requestMonitorMemRequest)
		resourceList[kcore.ResourceMemory] = *userPodMemRequest
	}

	if api.Compute.GPU > 0 {
		resourceList["nvidia.com/gpu"] = *kresource.NewQuantity(api.Compute.GPU, kresource.DecimalSI)
		resourceLimitsList["nvidia.com/gpu"] = *kresource.NewQuantity(api.Compute.GPU, kresource.DecimalSI)
	}

	if api.Predictor.ShmSize != nil {
		volumes = append(volumes, kcore.Volume{
			Name: "dshm",
			VolumeSource: kcore.VolumeSource{
				EmptyDir: &kcore.EmptyDirVolumeSource{
					Medium:    kcore.StorageMediumMemory,
					SizeLimit: k8s.QuantityPtr(api.Predictor.ShmSize.Quantity),
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
		Image:           api.Predictor.Image,
		ImagePullPolicy: kcore.PullAlways,
		Env:             getEnvVars(api, APIContainerName),
		EnvFrom:         baseEnvVars(),
		VolumeMounts:    apiPodVolumeMounts,
		ReadinessProbe:  FileExistsProbe(_apiReadinessFile),
		Lifecycle:       nginxGracefulStopper(api.Kind),
		Resources: kcore.ResourceRequirements{
			Requests: resourceList,
			Limits:   resourceLimitsList,
		},
		Ports: []kcore.ContainerPort{
			{ContainerPort: DefaultPortInt32},
		},
		SecurityContext: &kcore.SecurityContext{
			Privileged: pointer.Bool(true),
		},
	})

	return containers, volumes
}

func getTaskEnvVars(api *spec.API, container string) []kcore.EnvVar {
	envVars := []kcore.EnvVar{
		{
			Name:  "CORTEX_KIND",
			Value: api.Kind.String(),
		},
		{
			Name:  "CORTEX_LOG_LEVEL",
			Value: strings.ToUpper(api.TaskDefinition.LogLevel.String()),
		},
	}

	for name, val := range api.TaskDefinition.Env {
		envVars = append(envVars, kcore.EnvVar{
			Name:  name,
			Value: val,
		})
	}

	if container == APIContainerName {
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
				Name:  "CORTEX_PROJECT_DIR",
				Value: path.Join(_emptyDirMountPath, "project"),
			},
			kcore.EnvVar{
				Name:  "CORTEX_CACHE_DIR",
				Value: _specCacheDir,
			},
			kcore.EnvVar{
				Name:  "CORTEX_API_SPEC",
				Value: config.BucketPath(api.Key),
			},
		)

		cortexPythonPath := path.Join(_emptyDirMountPath, "project")
		if api.TaskDefinition.PythonPath != nil {
			cortexPythonPath = path.Join(_emptyDirMountPath, "project", *api.TaskDefinition.PythonPath)
		}
		envVars = append(envVars, kcore.EnvVar{
			Name:  "CORTEX_PYTHON_PATH",
			Value: cortexPythonPath,
		})

		if api.Compute.Inf > 0 {
			envVars = append(envVars,
				kcore.EnvVar{
					Name:  "NEURONCORE_GROUP_SIZES",
					Value: s.Int64(api.Compute.Inf * consts.NeuronCoresPerInf),
				},
				kcore.EnvVar{
					Name:  "NEURON_RTD_ADDRESS",
					Value: fmt.Sprintf("unix:%s", _neuronRTDSocket),
				},
			)
		}
	}
	return envVars
}

func getEnvVars(api *spec.API, container string) []kcore.EnvVar {
	if container == _requestMonitorContainerName || container == _downloaderInitContainerName {
		return []kcore.EnvVar{
			{
				Name:  "CORTEX_LOG_LEVEL",
				Value: strings.ToUpper(api.Predictor.LogLevel.String()),
			},
		}
	}

	envVars := []kcore.EnvVar{
		{
			Name:  "CORTEX_KIND",
			Value: api.Kind.String(),
		},
		{
			Name:  "CORTEX_TELEMETRY_SENTRY_USER_ID",
			Value: config.OperatorMetadata.OperatorID,
		},
		{
			Name:  "CORTEX_TELEMETRY_SENTRY_ENVIRONMENT",
			Value: "api",
		},
	}

	for name, val := range api.Predictor.Env {
		envVars = append(envVars, kcore.EnvVar{
			Name:  name,
			Value: val,
		})
	}

	if container == APIContainerName {
		envVars = append(envVars,
			kcore.EnvVar{
				Name:  "CORTEX_LOG_LEVEL",
				Value: strings.ToUpper(api.Predictor.LogLevel.String()),
			},
			kcore.EnvVar{
				Name: "HOST_IP",
				ValueFrom: &kcore.EnvVarSource{
					FieldRef: &kcore.ObjectFieldSelector{
						FieldPath: "status.hostIP",
					},
				},
			},
			kcore.EnvVar{
				Name:  "CORTEX_PROCESSES_PER_REPLICA",
				Value: s.Int32(api.Predictor.ProcessesPerReplica),
			},
			kcore.EnvVar{
				Name:  "CORTEX_THREADS_PER_PROCESS",
				Value: s.Int32(api.Predictor.ThreadsPerProcess),
			},
			kcore.EnvVar{
				Name:  "CORTEX_SERVING_PORT",
				Value: DefaultPortStr,
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
				Name:  "CORTEX_DEPENDENCIES_PIP",
				Value: api.Predictor.Dependencies.Pip,
			},
			kcore.EnvVar{
				Name:  "CORTEX_DEPENDENCIES_CONDA",
				Value: api.Predictor.Dependencies.Conda,
			},
			kcore.EnvVar{
				Name:  "CORTEX_DEPENDENCIES_SHELL",
				Value: api.Predictor.Dependencies.Shell,
			},
		)

		if api.Kind == userconfig.RealtimeAPIKind {
			envVars = append(envVars,
				kcore.EnvVar{
					Name:  "CORTEX_API_SPEC",
					Value: config.BucketPath(api.PredictorKey),
				},
			)
		} else {
			envVars = append(envVars,
				kcore.EnvVar{
					Name:  "CORTEX_API_SPEC",
					Value: config.BucketPath(api.Key),
				},
			)
		}

		if api.Autoscaling != nil {
			envVars = append(envVars,
				kcore.EnvVar{
					Name:  "CORTEX_MAX_REPLICA_CONCURRENCY",
					Value: s.Int64(api.Autoscaling.MaxReplicaConcurrency),
				},
			)
		}

		if api.Predictor.Type != userconfig.PythonPredictorType || api.Predictor.MultiModelReloading != nil {
			envVars = append(envVars,
				kcore.EnvVar{
					Name:  "CORTEX_MODEL_DIR",
					Value: path.Join(_emptyDirMountPath, "model"),
				},
			)
		}

		cortexPythonPath := path.Join(_emptyDirMountPath, "project")
		if api.Predictor.PythonPath != nil {
			cortexPythonPath = path.Join(_emptyDirMountPath, "project", *api.Predictor.PythonPath)
		}
		envVars = append(envVars, kcore.EnvVar{
			Name:  "CORTEX_PYTHON_PATH",
			Value: cortexPythonPath,
		})

		if api.Predictor.Type == userconfig.TensorFlowPredictorType {
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

	if container == _tfServingContainerName {
		envVars = append(envVars, kcore.EnvVar{
			Name:  "TF_CPP_MIN_LOG_LEVEL",
			Value: s.Int(userconfig.TFNumericLogLevelFromLogLevel(api.Predictor.LogLevel)),
		})

		if api.Predictor.ServerSideBatching != nil {
			var numBatchedThreads int32
			if api.Compute.Inf > 0 {
				// because there are processes_per_replica TF servers
				numBatchedThreads = 1
			} else {
				numBatchedThreads = api.Predictor.ProcessesPerReplica
			}

			envVars = append(envVars,
				kcore.EnvVar{
					Name:  "TF_MAX_BATCH_SIZE",
					Value: s.Int32(api.Predictor.ServerSideBatching.MaxBatchSize),
				},
				kcore.EnvVar{
					Name:  "TF_BATCH_TIMEOUT_MICROS",
					Value: s.Int64(api.Predictor.ServerSideBatching.BatchInterval.Microseconds()),
				},
				kcore.EnvVar{
					Name:  "TF_NUM_BATCHED_THREADS",
					Value: s.Int32(numBatchedThreads),
				},
			)
		}
	}

	if api.Compute.Inf > 0 {
		if (api.Predictor.Type == userconfig.PythonPredictorType && container == APIContainerName) ||
			(api.Predictor.Type == userconfig.TensorFlowPredictorType && container == _tfServingContainerName) {
			envVars = append(envVars,
				kcore.EnvVar{
					Name:  "NEURONCORE_GROUP_SIZES",
					Value: s.Int64(api.Compute.Inf * consts.NeuronCoresPerInf / int64(api.Predictor.ProcessesPerReplica)),
				},
				kcore.EnvVar{
					Name:  "NEURON_RTD_ADDRESS",
					Value: fmt.Sprintf("unix:%s", _neuronRTDSocket),
				},
			)
		}

		if api.Predictor.Type == userconfig.TensorFlowPredictorType {
			if container == _tfServingContainerName {
				envVars = append(envVars,
					kcore.EnvVar{
						Name:  "TF_PROCESSES",
						Value: s.Int32(api.Predictor.ProcessesPerReplica),
					},
					kcore.EnvVar{
						Name:  "CORTEX_TF_BASE_SERVING_PORT",
						Value: _tfBaseServingPortStr,
					},
					kcore.EnvVar{
						Name:  "TF_EMPTY_MODEL_CONFIG",
						Value: _tfServingEmptyModelConfig,
					},
					kcore.EnvVar{
						Name:  "TF_MAX_NUM_LOAD_RETRIES",
						Value: _tfServingMaxNumLoadRetries,
					},
					kcore.EnvVar{
						Name:  "TF_LOAD_RETRY_INTERVAL_MICROS",
						Value: _tfServingLoadTimeMicros,
					},
					kcore.EnvVar{
						Name:  "TF_GRPC_MAX_CONCURRENT_STREAMS",
						Value: fmt.Sprintf(`--grpc_channel_arguments="grpc.max_concurrent_streams=%d"`, api.Predictor.ThreadsPerProcess+10),
					},
				)
			}
			if container == APIContainerName {
				envVars = append(envVars,
					kcore.EnvVar{
						Name:  "CORTEX_MULTIPLE_TF_SERVERS",
						Value: "yes",
					},
					kcore.EnvVar{
						Name:  "CORTEX_ACTIVE_NEURON",
						Value: "yes",
					},
				)
			}
		}
	}

	return envVars
}

func tfDownloadArgs(api *spec.API) string {
	downloadConfig := downloadContainerConfig{
		LastLog: fmt.Sprintf(_downloaderLastLog, "tensorflow"),
		DownloadArgs: []downloadContainerArg{
			{
				From:             config.BucketPath(api.ProjectKey),
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

func pythonDownloadArgs(api *spec.API) string {
	downloadConfig := downloadContainerConfig{
		LastLog: fmt.Sprintf(_downloaderLastLog, "python"),
		DownloadArgs: []downloadContainerArg{
			{
				From:             config.BucketPath(api.ProjectKey),
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

func onnxDownloadArgs(api *spec.API) string {
	downloadContainerArs := []downloadContainerArg{
		{
			From:             config.BucketPath(api.ProjectKey),
			To:               path.Join(_emptyDirMountPath, "project"),
			Unzip:            true,
			ItemName:         "the project code",
			HideFromLog:      true,
			HideUnzippingLog: true,
		},
	}

	if api.Predictor.Models.Path != nil && strings.HasSuffix(*api.Predictor.Models.Path, ".onnx") {
		downloadContainerArs = append(downloadContainerArs, downloadContainerArg{
			From:     *api.Predictor.Models.Path,
			To:       path.Join(_modelDir, consts.SingleModelName, "1"),
			ItemName: "the onnx model",
		})
	}

	for _, model := range api.Predictor.Models.Paths {
		if model == nil {
			continue
		}
		if strings.HasSuffix(model.Path, ".onnx") {
			downloadContainerArs = append(downloadContainerArs, downloadContainerArg{
				From:     model.Path,
				To:       path.Join(_modelDir, model.Name, "1"),
				ItemName: fmt.Sprintf("%s onnx model", model.Name),
			})
		}
	}

	downloadConfig := downloadContainerConfig{
		LastLog:      fmt.Sprintf(_downloaderLastLog, "onnx"),
		DownloadArgs: downloadContainerArs,
	}

	downloadArgsBytes, _ := json.Marshal(downloadConfig)
	return base64.URLEncoding.EncodeToString(downloadArgsBytes)
}

func tensorflowServingContainer(api *spec.API, volumeMounts []kcore.VolumeMount, resources kcore.ResourceRequirements) *kcore.Container {
	var cmdArgs []string
	ports := []kcore.ContainerPort{
		{
			ContainerPort: _tfBaseServingPortInt32,
		},
	}

	if api.Compute.Inf > 0 {
		numPorts := api.Predictor.ProcessesPerReplica
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
			fmt.Sprintf(`--grpc_channel_arguments="grpc.max_concurrent_streams=%d"`, api.Predictor.ProcessesPerReplica*api.Predictor.ThreadsPerProcess+10),
		}
		if api.Predictor.ServerSideBatching != nil {
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

	return &kcore.Container{
		Name:            _tfServingContainerName,
		Image:           api.Predictor.TensorFlowServingImage,
		ImagePullPolicy: kcore.PullAlways,
		Args:            cmdArgs,
		Env:             getEnvVars(api, _tfServingContainerName),
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

func neuronRuntimeDaemonContainer(api *spec.API, volumeMounts []kcore.VolumeMount) *kcore.Container {
	totalHugePages := api.Compute.Inf * _hugePagesMemPerInf
	return &kcore.Container{
		Name:            _neuronRTDContainerName,
		Image:           config.CoreConfig.ImageNeuronRTD,
		ImagePullPolicy: kcore.PullAlways,
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
	var image string
	if config.Provider == types.AWSProviderType {
		image = config.CoreConfig.ImageRequestMonitor
	} else if config.Provider == types.GCPProviderType {
		image = config.GCPCoreConfig.ImageRequestMonitor
	}

	return kcore.Container{
		Name:            _requestMonitorContainerName,
		Image:           image,
		ImagePullPolicy: kcore.PullAlways,
		Args:            []string{"-p", DefaultRequestMonitorPortStr},
		Ports: []kcore.ContainerPort{
			{Name: "metrics", ContainerPort: DefaultRequestMonitorPortInt32},
		},
		Env:            getEnvVars(api, _requestMonitorContainerName),
		EnvFrom:        baseEnvVars(),
		VolumeMounts:   defaultVolumeMounts(),
		ReadinessProbe: FileExistsProbe(_requestMonitorReadinessFile),
		Resources: kcore.ResourceRequirements{
			Requests: kcore.ResourceList{
				kcore.ResourceCPU:    _requestMonitorCPURequest,
				kcore.ResourceMemory: _requestMonitorMemRequest,
			},
		},
	}
}

func FileExistsProbe(fileName string) *kcore.Probe {
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

func socketExistsProbe(socketName string) *kcore.Probe {
	return &kcore.Probe{
		InitialDelaySeconds: 3,
		TimeoutSeconds:      5,
		PeriodSeconds:       5,
		SuccessThreshold:    1,
		FailureThreshold:    1,
		Handler: kcore.Handler{
			Exec: &kcore.ExecAction{
				Command: []string{"/bin/bash", "-c", fmt.Sprintf("test -S %s", socketName)},
			},
		},
	}
}

func nginxGracefulStopper(apiKind userconfig.Kind) *kcore.Lifecycle {
	if apiKind == userconfig.RealtimeAPIKind {
		return &kcore.Lifecycle{
			PreStop: &kcore.Handler{
				Exec: &kcore.ExecAction{
					// the sleep is required to wait for any k8s-related race conditions
					// as described in https://medium.com/codecademy-engineering/kubernetes-nginx-and-zero-downtime-in-production-2c910c6a5ed8
					Command: []string{"/bin/sh", "-c", "sleep 5; /usr/sbin/nginx -s quit; while pgrep -x nginx; do sleep 1; done"},
				},
			},
		}
	}
	return nil
}

func waitAPIContainerToStop(apiKind userconfig.Kind) *kcore.Lifecycle {
	if apiKind == userconfig.RealtimeAPIKind {
		return &kcore.Lifecycle{
			PreStop: &kcore.Handler{
				Exec: &kcore.ExecAction{
					Command: []string{"/bin/sh", "-c", fmt.Sprintf("while curl localhost:%s/nginx_status; do sleep 1; done", DefaultPortStr)},
				},
			},
		}
	}
	return nil
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

	if config.Provider == types.AWSProviderType {
		envVars = append(envVars, kcore.EnvFromSource{
			SecretRef: &kcore.SecretEnvSource{
				LocalObjectReference: kcore.LocalObjectReference{
					Name: "aws-credentials",
				},
			},
		})
	}

	return envVars
}

func DefaultVolumes() []kcore.Volume {
	var defaultVolumes = []kcore.Volume{
		k8s.EmptyDirVolume(_emptyDirVolumeName),
	}

	if config.Provider == types.GCPProviderType {
		defaultVolumes = append(defaultVolumes, kcore.Volume{
			Name: "gcp-credentials",
			VolumeSource: kcore.VolumeSource{
				Secret: &kcore.SecretVolumeSource{
					SecretName: "gcp-credentials",
				},
			},
		})
	}

	return defaultVolumes
}

func defaultVolumeMounts() []kcore.VolumeMount {
	var volumeMounts = []kcore.VolumeMount{
		k8s.EmptyDirVolumeMount(_emptyDirVolumeName, _emptyDirMountPath),
	}

	if config.Provider == types.GCPProviderType {
		volumeMounts = append(volumeMounts, kcore.VolumeMount{
			Name:      "gcp-credentials",
			ReadOnly:  true,
			MountPath: "/var/secrets/google",
		})
	}

	return volumeMounts
}

func NodeSelectors() map[string]string {
	nodeSelectors := map[string]string{}
	if config.IsManaged() {
		nodeSelectors["workload"] = "true"
	}
	return nodeSelectors
}

var Tolerations = []kcore.Toleration{
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

func K8sName(apiName string) string {
	return "api-" + apiName
}

// APILoadBalancerURL returns the http endpoint of the ingress load balancer for deployed APIs
func APILoadBalancerURL() (string, error) {
	return getLoadBalancerURL("ingressgateway-apis")
}

// LoadBalancerURL returns the http endpoint of the ingress load balancer for the operator
func LoadBalancerURL() (string, error) {
	return getLoadBalancerURL("ingressgateway-operator")
}

func getLoadBalancerURL(name string) (string, error) {
	service, err := config.K8sIstio.GetService(name)
	if err != nil {
		return "", err
	}
	if service == nil {
		return "", ErrorCortexInstallationBroken()
	}
	if len(service.Status.LoadBalancer.Ingress) == 0 {
		return "", ErrorLoadBalancerInitializing()
	}
	if service.Status.LoadBalancer.Ingress[0].Hostname != "" {
		return "http://" + service.Status.LoadBalancer.Ingress[0].Hostname, nil
	}
	return "http://" + service.Status.LoadBalancer.Ingress[0].IP, nil
}

func APIEndpoint(api *spec.API) (string, error) {
	var err error
	baseAPIEndpoint := ""

	baseAPIEndpoint, err = APILoadBalancerURL()
	if err != nil {
		return "", err
	}
	baseAPIEndpoint = strings.Replace(baseAPIEndpoint, "https://", "http://", 1)

	return urls.Join(baseAPIEndpoint, *api.Networking.Endpoint), nil
}
