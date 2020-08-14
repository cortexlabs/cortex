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
	"encoding/json"
	"fmt"
	"math"
	"path"
	"strings"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/maps"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/lib/urls"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/types"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	istioclientnetworking "istio.io/client-go/pkg/apis/networking/v1alpha3"
	kcore "k8s.io/api/core/v1"
	kresource "k8s.io/apimachinery/pkg/api/resource"
	kmeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	intstr "k8s.io/apimachinery/pkg/util/intstr"
)

const (
	DefaultPortInt32 = int32(8888)
	DefaultPortStr   = "8888"
	APIContainerName = "api"
)

const (
	_specCacheDir                                  = "/mnt/spec"
	_emptyDirMountPath                             = "/mnt"
	_emptyDirVolumeName                            = "mnt"
	_tfServingContainerName                        = "serve"
	_tfServingModelName                            = "model"
	_downloaderInitContainerName                   = "downloader"
	_downloaderLastLog                             = "downloading the %s serving image"
	_neuronRTDContainerName                        = "neuron-rtd"
	_tfBaseServingPortInt32, _tfBaseServingPortStr = int32(9000), "9000"
	_tfServingHost                                 = "localhost"
	_tfServingEmptyModelConfig                     = "/etc/tfs/model_config_server.conf"
	_tfServingBatchConfig                          = "/etc/tfs/batch_config.conf"
	_apiReadinessFile                              = "/mnt/workspace/api_readiness.txt"
	_apiLivenessFile                               = "/mnt/workspace/api_liveness.txt"
	_neuronRTDSocket                               = "/sock/neuron.sock"
	_apiLivenessStalePeriod                        = 7 // seconds (there is a 2-second buffer to be safe)
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
	From                 string `json:"from"`
	To                   string `json:"to"`
	Unzip                bool   `json:"unzip"`
	ItemName             string `json:"item_name"`               // name of the item being downloaded, just for logging (if "" nothing will be logged)
	TFModelVersionRename string `json:"tf_model_version_rename"` // e.g. passing in /mnt/model/1 will rename /mnt/model/* to /mnt/model/1 only if there is one item in /mnt/model/
	HideFromLog          bool   `json:"hide_from_log"`           // if true, don't log where the file is being downloaded from
	HideUnzippingLog     bool   `json:"hide_unzipping_log"`      // if true, don't log when unzipping
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
		Image:           config.Cluster.ImageDownloader,
		ImagePullPolicy: "Always",
		Args:            []string{"--download=" + downloadArgs},
		EnvFrom:         BaseEnvVars,
		VolumeMounts:    DefaultVolumeMounts,
	}
}

func PythonPredictorContainers(api *spec.API) ([]kcore.Container, []kcore.Volume) {
	apiPodResourceList := kcore.ResourceList{}
	apiPodResourceLimitsList := kcore.ResourceList{}
	apiPodVolumeMounts := DefaultVolumeMounts
	volumes := DefaultVolumes
	containers := []kcore.Container{}

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

	containers = append(containers, kcore.Container{
		Name:            APIContainerName,
		Image:           api.Predictor.Image,
		ImagePullPolicy: kcore.PullAlways,
		Env:             getEnvVars(api, APIContainerName),
		EnvFrom:         BaseEnvVars,
		VolumeMounts:    apiPodVolumeMounts,
		ReadinessProbe:  FileExistsProbe(_apiReadinessFile),
		LivenessProbe:   _apiLivenessProbe,
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
	volumeMounts := DefaultVolumeMounts
	volumes := DefaultVolumes
	containers := []kcore.Container{}

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

	containers = append(containers, kcore.Container{
		Name:            APIContainerName,
		Image:           api.Predictor.Image,
		ImagePullPolicy: kcore.PullAlways,
		Env:             getEnvVars(api, APIContainerName),
		EnvFrom:         BaseEnvVars,
		VolumeMounts:    volumeMounts,
		ReadinessProbe:  FileExistsProbe(_apiReadinessFile),
		LivenessProbe:   _apiLivenessProbe,
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

func ONNXPredictorContainers(api *spec.API) []kcore.Container {
	resourceList := kcore.ResourceList{}
	resourceLimitsList := kcore.ResourceList{}
	containers := []kcore.Container{}

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

	containers = append(containers, kcore.Container{
		Name:            APIContainerName,
		Image:           api.Predictor.Image,
		ImagePullPolicy: kcore.PullAlways,
		Env:             getEnvVars(api, APIContainerName),
		EnvFrom:         BaseEnvVars,
		VolumeMounts:    DefaultVolumeMounts,
		ReadinessProbe:  FileExistsProbe(_apiReadinessFile),
		LivenessProbe:   _apiLivenessProbe,
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

	return containers
}

func getEnvVars(api *spec.API, container string) []kcore.EnvVar {
	envVars := []kcore.EnvVar{
		{
			Name:  "CORTEX_PROVIDER",
			Value: types.AWSProviderType.String(),
		},
		{
			Name:  "CORTEX_KIND",
			Value: api.Kind.String(),
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
		)

		if api.Autoscaling != nil {
			envVars = append(envVars,
				kcore.EnvVar{
					Name:  "CORTEX_MAX_REPLICA_CONCURRENCY",
					Value: s.Int64(api.Autoscaling.MaxReplicaConcurrency),
				},
				kcore.EnvVar{
					Name: "CORTEX_MAX_PROCESS_CONCURRENCY",
					// add 1 because it was required to achieve the target concurrency for 1 process, 1 thread
					Value: s.Int64(1 + int64(math.Round(float64(api.Autoscaling.MaxReplicaConcurrency)/float64(api.Predictor.ProcessesPerReplica)))),
				},
				kcore.EnvVar{
					Name:  "CORTEX_SO_MAX_CONN",
					Value: s.Int64(api.Autoscaling.MaxReplicaConcurrency + 100), // add a buffer to be safe
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

		if api.Predictor.Type == userconfig.ONNXPredictorType {
			envVars = append(envVars,
				kcore.EnvVar{
					Name:  "CORTEX_MODEL_DIR",
					Value: path.Join(_emptyDirMountPath, "model"),
				},
				kcore.EnvVar{
					Name:  "CORTEX_MODELS",
					Value: strings.Join(api.ModelNames(), ","),
				},
			)
		}

		if api.Predictor.Type == userconfig.TensorFlowPredictorType {
			envVars = append(envVars,
				kcore.EnvVar{
					Name:  "CORTEX_MODEL_DIR",
					Value: path.Join(_emptyDirMountPath, "model"),
				},
				kcore.EnvVar{
					Name:  "CORTEX_MODELS",
					Value: strings.Join(api.ModelNames(), ","),
				},
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
						Name:  "CORTEX_MODEL_DIR",
						Value: path.Join(_emptyDirMountPath, "model"),
					},
					kcore.EnvVar{
						Name:  "TF_EMPTY_MODEL_CONFIG",
						Value: _tfServingEmptyModelConfig,
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
				From:             aws.S3Path(config.Cluster.Bucket, api.ProjectKey),
				To:               path.Join(_emptyDirMountPath, "project"),
				Unzip:            true,
				ItemName:         "the project code",
				HideFromLog:      true,
				HideUnzippingLog: true,
			},
		},
	}

	rootModelPath := path.Join(_emptyDirMountPath, "model")
	for _, model := range api.Predictor.Models {
		var itemName string
		if model.Name == consts.SingleModelName {
			itemName = "the model"
		} else {
			itemName = fmt.Sprintf("model %s", model.Name)
		}
		downloadConfig.DownloadArgs = append(downloadConfig.DownloadArgs, downloadContainerArg{
			From:                 model.ModelPath,
			To:                   path.Join(rootModelPath, model.Name),
			Unzip:                strings.HasSuffix(model.ModelPath, ".zip"),
			ItemName:             itemName,
			TFModelVersionRename: path.Join(rootModelPath, model.Name, "1"),
		})
	}

	downloadArgsBytes, _ := json.Marshal(downloadConfig)
	return base64.URLEncoding.EncodeToString(downloadArgsBytes)
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
		},
	}

	rootModelPath := path.Join(_emptyDirMountPath, "model")
	for _, model := range api.Predictor.Models {
		var itemName string
		if model.Name == consts.SingleModelName {
			itemName = "the model"
		} else {
			itemName = fmt.Sprintf("model %s", model.Name)
		}
		downloadConfig.DownloadArgs = append(downloadConfig.DownloadArgs, downloadContainerArg{
			From:     model.ModelPath,
			To:       path.Join(rootModelPath, model.Name),
			ItemName: itemName,
		})
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
		EnvFrom:         BaseEnvVars,
		VolumeMounts:    volumeMounts,
		ReadinessProbe: &kcore.Probe{
			InitialDelaySeconds: 5,
			TimeoutSeconds:      5,
			PeriodSeconds:       5,
			SuccessThreshold:    1,
			FailureThreshold:    2,
			Handler:             probeHandler,
		},
		Resources: resources,
		Ports:     ports,
	}
}

func neuronRuntimeDaemonContainer(api *spec.API, volumeMounts []kcore.VolumeMount) *kcore.Container {
	totalHugePages := api.Compute.Inf * _hugePagesMemPerInf
	return &kcore.Container{
		Name:            _neuronRTDContainerName,
		Image:           config.Cluster.ImageNeuronRTD,
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
		Resources: kcore.ResourceRequirements{
			Requests: kcore.ResourceList{
				"hugepages-2Mi":       *kresource.NewQuantity(totalHugePages, kresource.BinarySI),
				"aws.amazon.com/infa": *kresource.NewQuantity(api.Compute.Inf, kresource.DecimalSI),
			},
			Limits: kcore.ResourceList{
				"hugepages-2Mi":       *kresource.NewQuantity(totalHugePages, kresource.BinarySI),
				"aws.amazon.com/infa": *kresource.NewQuantity(api.Compute.Inf, kresource.DecimalSI),
			},
		},
	}
}

func RequestMonitorContainer(api *spec.API) kcore.Container {
	return kcore.Container{
		Name:            "request-monitor",
		Image:           config.Cluster.ImageRequestMonitor,
		ImagePullPolicy: kcore.PullAlways,
		Args:            []string{api.Name, config.Cluster.ClusterName},
		EnvFrom:         BaseEnvVars,
		VolumeMounts:    DefaultVolumeMounts,
		ReadinessProbe:  FileExistsProbe(_requestMonitorReadinessFile),
		Resources: kcore.ResourceRequirements{
			Requests: kcore.ResourceList{
				kcore.ResourceCPU:    _requestMonitorCPURequest,
				kcore.ResourceMemory: _requestMonitorMemRequest,
			},
		},
	}
}

var _apiLivenessProbe = &kcore.Probe{
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
	{
		Key:      "aws.amazon.com/infa",
		Operator: kcore.TolerationOpEqual,
		Value:    "true",
		Effect:   kcore.TaintEffectNoSchedule,
	},
}

var BaseEnvVars = []kcore.EnvFromSource{
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

var DefaultVolumes = []kcore.Volume{
	k8s.EmptyDirVolume(_emptyDirVolumeName),
}

var DefaultVolumeMounts = []kcore.VolumeMount{
	k8s.EmptyDirVolumeMount(_emptyDirVolumeName, _emptyDirMountPath),
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
		Operator: kcore.TolerationOpEqual,
		Value:    "true",
		Effect:   kcore.TaintEffectNoSchedule,
	},
	{
		Key:      "aws.amazon.com/infa",
		Operator: kcore.TolerationOpEqual,
		Value:    "true",
		Effect:   kcore.TaintEffectNoSchedule,
	},
}

func K8sName(apiName string) string {
	return "api-" + apiName
}

// APILoadBalancerURL returns http endpoint of cluster ingress elb
func APILoadBalancerURL() (string, error) {
	service, err := config.K8sIstio.GetService("ingressgateway-apis")
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

func APIEndpoint(api *spec.API) (string, error) {
	var err error
	baseAPIEndpoint := ""

	if api.Networking.APIGateway == userconfig.PublicAPIGatewayType && config.Cluster.APIGateway != nil {
		baseAPIEndpoint = *config.Cluster.APIGateway.ApiEndpoint
	} else {
		baseAPIEndpoint, err = APILoadBalancerURL()
		if err != nil {
			return "", err
		}
		baseAPIEndpoint = strings.Replace(baseAPIEndpoint, "https://", "http://", 1)
	}

	return urls.Join(baseAPIEndpoint, *api.Networking.Endpoint), nil
}

func GetEndpointFromVirtualService(virtualService *istioclientnetworking.VirtualService) (string, error) {
	endpoints := k8s.ExtractVirtualServiceEndpoints(virtualService)

	if len(endpoints) != 1 {
		return "", errors.ErrorUnexpected("expected 1 endpoint, but got", endpoints)
	}

	return endpoints.GetOne(), nil
}

func DoCortexAnnotationsMatch(obj1, obj2 kmeta.Object) bool {
	cortexAnnotations1 := extractCortexAnnotations(obj1)
	cortexAnnotations2 := extractCortexAnnotations(obj2)
	return maps.StrMapsEqual(cortexAnnotations1, cortexAnnotations2)
}

func extractCortexAnnotations(obj kmeta.Object) map[string]string {
	cortexAnnotations := make(map[string]string)
	for key, value := range obj.GetAnnotations() {
		if strings.Contains(key, "cortex.dev/") {
			cortexAnnotations[key] = value
		}
	}
	return cortexAnnotations
}
