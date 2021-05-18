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
	"strings"

	"github.com/cortexlabs/cortex/pkg/config"
	"github.com/cortexlabs/cortex/pkg/consts"
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
	ServiceAccountName = "default"
)

const (
	_cortexDirVolumeName = "cortex"
	_cortexDirMountPath  = "/cortex"
	_clientConfigDir     = "/cortex/client"

	_emptyDirVolumeName = "mnt"
	_emptyDirMountPath  = "/mnt"

	_proxyContainerName = "proxy"

	_gatewayContainerName = "gateway"

	_neuronRTDContainerName = "neuron-rtd"
	_neuronRTDSocket        = "/sock/neuron.sock"

	_kubexitGraveyardName      = "graveyard"
	_kubexitGraveyardMountPath = "/graveyard"

	_shmDirVolumeName = "dshm"
	_shmDirMountPath  = "/dev/shm"

	_clientConfigDirVolume = "client-config"
	_clientConfigConfigMap = "client-config"
)

var (
	_requestMonitorCPURequest = kresource.MustParse("10m")
	_requestMonitorMemRequest = kresource.MustParse("10Mi")

	_asyncGatewayCPURequest = kresource.MustParse("100m")
	_asyncGatewayMemRequest = kresource.MustParse("100Mi")

	// each Inferentia chip requires 128 HugePages with each HugePage having a size of 2Mi
	_hugePagesMemPerInf = int64(128 * 2 * 1024 * 1024) // bytes
)

func AsyncGatewayContainer(api spec.API, queueURL string, volumeMounts []kcore.VolumeMount) kcore.Container {
	return kcore.Container{
		Name:            _gatewayContainerName,
		Image:           config.ClusterConfig.ImageAsyncGateway,
		ImagePullPolicy: kcore.PullAlways,
		Args: []string{
			"-port", s.Int32(consts.ProxyListeningPortInt32),
			"-queue", queueURL,
			"-cluster-config", consts.DefaultInClusterConfigPath,
			api.Name,
		},
		Ports: []kcore.ContainerPort{
			{ContainerPort: consts.ProxyListeningPortInt32},
		},
		Env: []kcore.EnvVar{
			{
				Name:  "CORTEX_LOG_LEVEL",
				Value: strings.ToUpper(userconfig.InfoLogLevel.String()),
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

func UserPodContainers(api spec.API) ([]kcore.Container, []kcore.Volume) {
	requiresKubexit := api.Kind == userconfig.BatchAPIKind || api.Kind == userconfig.TaskAPIKind

	volumes := []kcore.Volume{
		MntVolume(),
		CortexVolume(),
		ClientConfigVolume(),
	}
	containerMounts := []kcore.VolumeMount{
		MntMount(),
		CortexMount(),
		ClientConfigMount(),
	}

	if requiresKubexit {
		volumes = append(volumes, KubexitVolume())
		containerMounts = append(containerMounts, KubexitMount())
	}
	if api.Pod.ShmSize != nil {
		volumes = append(volumes, ShmVolume(api.Pod.ShmSize.Quantity))
		containerMounts = append(containerMounts, ShmMount())
	}

	var containers []kcore.Container
	var podHasInf bool
	containerNames := userconfig.GetContainerNames(api.Pod.Containers)
	for _, container := range api.Pod.Containers {
		containerResourceList := kcore.ResourceList{}
		containerResourceLimitsList := kcore.ResourceList{}

		if container.Compute.CPU != nil {
			containerResourceList[kcore.ResourceCPU] = *k8s.QuantityPtr(container.Compute.CPU.Quantity.DeepCopy())
		}

		if container.Compute.Mem != nil {
			containerResourceList[kcore.ResourceMemory] = *k8s.QuantityPtr(container.Compute.Mem.Quantity.DeepCopy())
		}

		if container.Compute.GPU > 0 {
			containerResourceList["nvidia.com/gpu"] = *kresource.NewQuantity(container.Compute.GPU, kresource.DecimalSI)
			containerResourceLimitsList["nvidia.com/gpu"] = *kresource.NewQuantity(container.Compute.GPU, kresource.DecimalSI)
		}

		containerVolumeMounts := containerMounts

		if container.Compute.Inf > 0 {
			volumes = append(volumes, kcore.Volume{
				Name: "neuron-sock",
			})
			rtdVolumeMounts := []kcore.VolumeMount{
				{
					Name:      "neuron-sock",
					MountPath: "/sock",
				},
			}

			containerVolumeMounts = append(containerVolumeMounts, rtdVolumeMounts...)

			if requiresKubexit {
				rtdVolumeMounts = append(rtdVolumeMounts,
					k8s.EmptyDirVolumeMount(_emptyDirVolumeName, _emptyDirMountPath),
					kcore.VolumeMount{Name: _kubexitGraveyardName, MountPath: _kubexitGraveyardMountPath},
				)
				neuronRTDEnvVars := getKubexitEnvVars(_neuronRTDContainerName, containerNames.Slice(), nil)
				containers = append(containers, neuronRuntimeDaemonContainer(container.Compute.Inf, rtdVolumeMounts, neuronRTDEnvVars))
			} else {
				containers = append(containers, neuronRuntimeDaemonContainer(container.Compute.Inf, rtdVolumeMounts, nil))
			}

			podHasInf = true
		}

		containerEnvVars := []kcore.EnvVar{
			{
				Name: "HOST_IP",
				ValueFrom: &kcore.EnvVarSource{
					FieldRef: &kcore.ObjectFieldSelector{
						FieldPath: "status.hostIP",
					},
				},
			},
			{
				Name:  "CORTEX_PORT",
				Value: s.Int32(*api.Pod.Port),
			},
		}
		if requiresKubexit {
			containerDeathDependencies := containerNames.Copy()
			containerDeathDependencies.Remove(container.Name)
			if podHasInf {
				containerEnvVars = getKubexitEnvVars(container.Name, containerDeathDependencies.Slice(), []string{"neuron-rtd"})
			} else {
				containerEnvVars = getKubexitEnvVars(container.Name, containerDeathDependencies.Slice(), nil)
			}
		}

		for k, v := range container.Env {
			containerEnvVars = append(containerEnvVars, kcore.EnvVar{
				Name:  k,
				Value: v,
			})
		}

		var containerCmd []string
		if requiresKubexit && container.Command[0] != "/mnt/kubexit" {
			containerCmd = append([]string{"/mnt/kubexit"}, container.Command...)
		}

		containers = append(containers, kcore.Container{
			Name:         container.Name,
			Image:        container.Image,
			Command:      containerCmd,
			Args:         container.Args,
			Env:          containerEnvVars,
			VolumeMounts: containerVolumeMounts,
			Resources: kcore.ResourceRequirements{
				Requests: containerResourceList,
				Limits:   containerResourceLimitsList,
			},
			ImagePullPolicy: kcore.PullAlways,
			SecurityContext: &kcore.SecurityContext{
				Privileged: pointer.Bool(true),
			}},
		)
	}

	return containers, volumes
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

func neuronRuntimeDaemonContainer(computeInf int64, volumeMounts []kcore.VolumeMount, envVars []kcore.EnvVar) kcore.Container {
	totalHugePages := computeInf * _hugePagesMemPerInf
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
		ReadinessProbe: SocketExistsProbe(_neuronRTDSocket),
		Resources: kcore.ResourceRequirements{
			Requests: kcore.ResourceList{
				"hugepages-2Mi":         *kresource.NewQuantity(totalHugePages, kresource.BinarySI),
				"aws.amazon.com/neuron": *kresource.NewQuantity(computeInf, kresource.DecimalSI),
			},
			Limits: kcore.ResourceList{
				"hugepages-2Mi":         *kresource.NewQuantity(totalHugePages, kresource.BinarySI),
				"aws.amazon.com/neuron": *kresource.NewQuantity(computeInf, kresource.DecimalSI),
			},
		},
	}
}

func RealtimeProxyContainer(api spec.API) kcore.Container {
	return kcore.Container{
		Name:            _proxyContainerName,
		Image:           config.ClusterConfig.ImageProxy,
		ImagePullPolicy: kcore.PullAlways,
		Args: []string{
			"-port",
			consts.ProxyListeningPortStr,
			"-metrics-port",
			consts.MetricsPortStr,
			"-user-port",
			s.Int32(*api.Pod.Port),
			"-max-concurrency",
			"1",
			"-max-queue-length",
			"1",
		},
		Ports: []kcore.ContainerPort{
			{Name: "metrics", ContainerPort: consts.MetricsPortInt32},
			{ContainerPort: consts.ProxyListeningPortInt32},
		},
		Env: []kcore.EnvVar{
			{
				Name:  "CORTEX_LOG_LEVEL",
				Value: strings.ToUpper(userconfig.InfoLogLevel.String()),
			},
		},
		EnvFrom: baseClusterEnvVars(),
	}
}

// func getAsyncAPIEnvVars(api spec.API, queueURL string) []kcore.EnvVar {
// 	envVars := apiContainerEnvVars(&api)

// 	envVars = append(envVars,
// 		kcore.EnvVar{
// 			Name:  "CORTEX_QUEUE_URL",
// 			Value: queueURL,
// 		},
// 		kcore.EnvVar{
// 			Name:  "CORTEX_ASYNC_WORKLOAD_PATH",
// 			Value: aws.S3Path(config.ClusterConfig.Bucket, fmt.Sprintf("%s/workloads/%s", config.ClusterConfig.ClusterUID, api.Name)),
// 		},
// 	)

// 	return envVars
// }
