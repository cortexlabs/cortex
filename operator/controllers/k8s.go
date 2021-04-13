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

package controllers

import (
	"fmt"
	"path"
	"strings"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/lib/pointer"
	s "github.com/cortexlabs/cortex/pkg/lib/strings"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	kcore "k8s.io/api/core/v1"
	kresource "k8s.io/apimachinery/pkg/api/resource"
)

const (
	DefaultPortInt32   = int32(8888)
	DefaultPortStr     = "8888"
	APIContainerName   = "api"
	ServiceAccountName = "default"
)

const (
	_specCacheDir                                  = "/mnt/spec"
	_clientConfigDir                               = "/mnt/client"
	_emptyDirMountPath                             = "/mnt"
	_emptyDirVolumeName                            = "mnt"
	_tfServingContainerName                        = "serve"
	_requestMonitorContainerName                   = "request-monitor"
	_downloaderInitContainerName                   = "downloader"
	_downloaderLastLog                             = "downloading the %s serving image"
	_tfBaseServingPortInt32, _tfBaseServingPortStr = int32(9000), "9000"
	_neuronRTDContainerName                        = "neuron-rtd"
	_tfServingHost                                 = "localhost"
	_tfServingEmptyModelConfig                     = "/etc/tfs/model_config_server.conf"
	_tfServingMaxNumLoadRetries                    = "0"        // maximum retries to load a model that didn't get loaded the first time
	_tfServingLoadTimeMicros                       = "30000000" // 30 seconds (how much time a model can take to load into memory)
	_apiReadinessFile                              = "/mnt/workspace/api_readiness.txt"
	_neuronRTDSocket                               = "/sock/neuron.sock"
)

var (
	_requestMonitorCPURequest = kresource.MustParse("10m")
	_requestMonitorMemRequest = kresource.MustParse("10Mi")

	// each Inferentia chip requires 128 HugePages with each HugePage having a size of 2Mi
	_hugePagesMemPerInf = int64(128 * 2 * 1024 * 1024) // bytes
)

func PythonPredictorContainers(api *spec.API) ([]kcore.Container, []kcore.Volume) {
	return pythonPredictorContainers(api, getEnvVars(api, APIContainerName))
}

// BaseEnvVars are the base cortex cluster env variables inferred from the cluster config
func BaseEnvVars() []kcore.EnvFromSource {
	return []kcore.EnvFromSource{
		{
			ConfigMapRef: &kcore.ConfigMapEnvSource{
				LocalObjectReference: kcore.LocalObjectReference{
					Name: "env-vars",
				},
			},
		},
	}
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

func DefaultVolumeMounts() []kcore.VolumeMount {
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

func GeneratePreferredNodeAffinities() []kcore.PreferredSchedulingTerm {
	var affinities []kcore.PreferredSchedulingTerm
	numNodeGroups := len(ClusterConfig.NodeGroups)
	for idx, nodeGroup := range ClusterConfig.NodeGroups {
		var nodeGroupPrefix string
		if nodeGroup.Spot {
			nodeGroupPrefix = "cx-ws-"
		} else {
			nodeGroupPrefix = "cx-wd-"
		}
		affinities = append(affinities, kcore.PreferredSchedulingTerm{
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
	}

	return affinities
}

// DownloaderContainer is the init container to download user images
func DownloaderContainer(api *spec.API) kcore.Container {
	downloadArgs := ""

	switch api.Predictor.Type {
	case userconfig.TensorFlowPredictorType:
		downloadArgs = tfDownloadArgs(api)
	case userconfig.PythonPredictorType:
		downloadArgs = pythonDownloadArgs(api)
	}

	return kcore.Container{
		Name:            _downloaderInitContainerName,
		Image:           ClusterConfig.ImageDownloader,
		ImagePullPolicy: "Always",
		Args:            []string{"--download=" + downloadArgs},
		EnvFrom:         BaseEnvVars(),
		Env:             getEnvVars(api, _downloaderInitContainerName),
		VolumeMounts:    DefaultVolumeMounts(),
	}
}

func pythonPredictorContainers(api *spec.API, envVars []kcore.EnvVar) ([]kcore.Container, []kcore.Volume) {
	apiPodResourceList := kcore.ResourceList{}
	apiPodResourceLimitsList := kcore.ResourceList{}
	apiPodVolumeMounts := DefaultVolumeMounts()
	volumes := DefaultVolumes()
	var containers []kcore.Container

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
		neuronContainer := *neuronRuntimeDaemonContainer(api, rtdVolumeMounts)

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
		Env:             envVars,
		EnvFrom:         BaseEnvVars(),
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
			Value: OperatorMetadata.OperatorID,
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
			kcore.EnvVar{
				Name:  "CORTEX_CLI_CONFIG_DIR",
				Value: _clientConfigDir,
			},
		)

		if api.Kind == userconfig.RealtimeAPIKind {
			servingProtocol := "http"
			if api.Predictor != nil && api.Predictor.IsGRPC() {
				servingProtocol = "grpc"
			}
			envVars = append(envVars,
				kcore.EnvVar{
					Name:  "CORTEX_API_SPEC",
					Value: aws.S3Path(ClusterConfig.Bucket, api.PredictorKey),
				},
				kcore.EnvVar{
					Name:  "CORTEX_SERVING_PROTOCOL",
					Value: servingProtocol,
				},
			)
			if servingProtocol == "grpc" {
				envVars = append(envVars, kcore.EnvVar{
					Name:  "CORTEX_PROTOBUF_FILE",
					Value: path.Join(_emptyDirMountPath, "project", *api.Predictor.ProtobufPath),
				})
			}
		} else {
			envVars = append(envVars,
				kcore.EnvVar{
					Name:  "CORTEX_API_SPEC",
					Value: aws.S3Path(ClusterConfig.Bucket, api.Key),
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

func neuronRuntimeDaemonContainer(api *spec.API, volumeMounts []kcore.VolumeMount) *kcore.Container {
	totalHugePages := api.Compute.Inf * _hugePagesMemPerInf
	return &kcore.Container{
		Name:            _neuronRTDContainerName,
		Image:           ClusterConfig.ImageNeuronRTD,
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
