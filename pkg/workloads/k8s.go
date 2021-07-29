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
	"fmt"
	"path"
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

	ProxyContainerName    = "proxy"
	DequeuerContainerName = "dequeuer"
	GatewayContainerName  = "gateway"

	_kubexitGraveyardName      = "graveyard"
	_kubexitGraveyardMountPath = "/graveyard"

	_shmDirMountPath = "/dev/shm"

	_clientConfigDirVolume = "client-config"
	_clientConfigConfigMap = "client-config"

	_clusterConfigDirVolume = "cluster-config"
	_clusterConfigConfigMap = "cluster-config"
	_clusterConfigDir       = "/configs/cluster"
)

var (
	_asyncGatewayCPURequest = kresource.MustParse("100m")
	_asyncGatewayMemRequest = kresource.MustParse("100Mi")

	_statsdAddress = fmt.Sprintf("prometheus-statsd-exporter.%s:9125", consts.PrometheusNamespace)

	// each Inferentia chip requires 128 HugePages with each HugePage having a size of 2Mi
	_hugePagesMemPerInf = int64(128 * 2 * 1024 * 1024) // bytes
)

func AsyncGatewayContainer(api spec.API, queueURL string, volumeMounts []kcore.VolumeMount) kcore.Container {
	return kcore.Container{
		Name:            GatewayContainerName,
		Image:           config.ClusterConfig.ImageAsyncGateway,
		ImagePullPolicy: kcore.PullAlways,
		Args: []string{
			"--cluster-config", consts.DefaultInClusterConfigPath,
			"--port", s.Int32(consts.ProxyPortInt32),
			"--queue", queueURL,
			api.Name,
		},
		Ports: []kcore.ContainerPort{
			{ContainerPort: consts.ProxyPortInt32},
		},
		Env:     BaseEnvVars,
		EnvFrom: BaseClusterEnvVars(),
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

func asyncDequeuerProxyContainer(api spec.API, queueURL string) (kcore.Container, kcore.Volume) {
	return kcore.Container{
		Name:            DequeuerContainerName,
		Image:           config.ClusterConfig.ImageDequeuer,
		ImagePullPolicy: kcore.PullAlways,
		Command: []string{
			"/dequeuer",
		},
		Args: []string{
			"--cluster-config", consts.DefaultInClusterConfigPath,
			"--cluster-uid", config.ClusterConfig.ClusterUID,
			"--probes-path", path.Join(_cortexDirMountPath, "spec", "probes.json"),
			"--queue", queueURL,
			"--api-kind", api.Kind.String(),
			"--api-name", api.Name,
			"--statsd-address", _statsdAddress,
			"--user-port", s.Int32(*api.Pod.Port),
			"--admin-port", consts.AdminPortStr,
		},
		Env:     BaseEnvVars,
		EnvFrom: BaseClusterEnvVars(),
		Ports: []kcore.ContainerPort{
			{
				Name:          consts.AdminPortName,
				ContainerPort: consts.AdminPortInt32,
			},
		},
		Resources: kcore.ResourceRequirements{
			Requests: kcore.ResourceList{
				kcore.ResourceCPU:    consts.CortexDequeuerCPU,
				kcore.ResourceMemory: consts.CortexDequeuerMem,
			},
		},
		ReadinessProbe: &kcore.Probe{
			Handler: kcore.Handler{
				HTTPGet: &kcore.HTTPGetAction{
					Path: "/healthz",
					Port: intstr.FromInt(int(consts.AdminPortInt32)),
				},
			},
			InitialDelaySeconds: 1,
			TimeoutSeconds:      1,
			PeriodSeconds:       10,
			SuccessThreshold:    1,
			FailureThreshold:    1,
		},
		VolumeMounts: []kcore.VolumeMount{
			ClusterConfigMount(),
		},
	}, ClusterConfigVolume()
}

func batchDequeuerProxyContainer(api spec.API, jobID, queueURL string) (kcore.Container, kcore.Volume) {
	return kcore.Container{
		Name:            DequeuerContainerName,
		Image:           config.ClusterConfig.ImageDequeuer,
		ImagePullPolicy: kcore.PullAlways,
		Command: []string{
			"/dequeuer",
		},
		Args: []string{
			"--cluster-config", consts.DefaultInClusterConfigPath,
			"--cluster-uid", config.ClusterConfig.ClusterUID,
			"--probes-path", path.Join(_cortexDirMountPath, "spec", "probes.json"),
			"--queue", queueURL,
			"--api-kind", api.Kind.String(),
			"--api-name", api.Name,
			"--job-id", jobID,
			"--statsd-address", _statsdAddress,
			"--user-port", s.Int32(*api.Pod.Port),
			"--admin-port", consts.AdminPortStr,
		},
		Env:     BaseEnvVars,
		EnvFrom: BaseClusterEnvVars(),
		Resources: kcore.ResourceRequirements{
			Requests: kcore.ResourceList{
				kcore.ResourceCPU:    consts.CortexDequeuerCPU,
				kcore.ResourceMemory: consts.CortexDequeuerMem,
			},
		},
		ReadinessProbe: &kcore.Probe{
			Handler: kcore.Handler{
				HTTPGet: &kcore.HTTPGetAction{
					Path: "/healthz",
					Port: intstr.FromInt(int(consts.AdminPortInt32)),
				},
			},
			InitialDelaySeconds: 1,
			TimeoutSeconds:      1,
			PeriodSeconds:       10,
			SuccessThreshold:    1,
			FailureThreshold:    1,
		},
		VolumeMounts: []kcore.VolumeMount{
			ClusterConfigMount(),
			CortexMount(),
		},
	}, ClusterConfigVolume()
}

func realtimeProxyContainer(api spec.API) (kcore.Container, kcore.Volume) {
	proxyHasTCPProbe := HasReadinessProbes(api.Pod.Containers)

	var readinessProbe *kcore.Probe
	if proxyHasTCPProbe {
		readinessProbe = &kcore.Probe{
			Handler: kcore.Handler{
				HTTPGet: &kcore.HTTPGetAction{
					Path: "/healthz",
					Port: intstr.FromInt(int(consts.AdminPortInt32)),
				},
			},
			InitialDelaySeconds: 1,
			TimeoutSeconds:      3,
			PeriodSeconds:       10,
			SuccessThreshold:    1,
			FailureThreshold:    3,
		}
	}

	return kcore.Container{
		Name:            ProxyContainerName,
		Image:           config.ClusterConfig.ImageProxy,
		ImagePullPolicy: kcore.PullAlways,
		Args: []string{
			"--cluster-config",
			consts.DefaultInClusterConfigPath,
			"--port",
			consts.ProxyPortStr,
			"--admin-port",
			consts.AdminPortStr,
			"--user-port",
			s.Int32(*api.Pod.Port),
			"--max-concurrency",
			s.Int32(int32(api.Pod.MaxConcurrency)),
			"--max-queue-length",
			s.Int32(int32(api.Pod.MaxQueueLength)),
			"--has-tcp-probe",
			s.Bool(proxyHasTCPProbe),
		},
		Ports: []kcore.ContainerPort{
			{Name: consts.AdminPortName, ContainerPort: consts.AdminPortInt32},
			{ContainerPort: consts.ProxyPortInt32},
		},
		Env:     BaseEnvVars,
		EnvFrom: BaseClusterEnvVars(),
		VolumeMounts: []kcore.VolumeMount{
			ClusterConfigMount(),
		},
		Resources: kcore.ResourceRequirements{
			Requests: kcore.ResourceList{
				kcore.ResourceCPU:    consts.CortexProxyCPU,
				kcore.ResourceMemory: consts.CortexProxyMem,
			},
		},
		ReadinessProbe: readinessProbe,
	}, ClusterConfigVolume()
}

func RealtimeContainers(api spec.API) ([]kcore.Container, []kcore.Volume) {
	containers, volumes := userPodContainers(api)
	proxyContainer, proxyVolume := realtimeProxyContainer(api)

	containers = append(containers, proxyContainer)
	volumes = append(volumes, proxyVolume)

	return containers, volumes
}

func AsyncContainers(api spec.API, queueURL string) ([]kcore.Container, []kcore.Volume) {
	k8sName := K8sName(api.Name)

	containers, volumes := userPodContainers(api)
	dequeuerContainer, dequeuerVolume := asyncDequeuerProxyContainer(api, queueURL)
	dequeuerContainer.VolumeMounts = append(dequeuerContainer.VolumeMounts, APIConfigMount(k8sName))

	containers = append(containers, dequeuerContainer)
	volumes = append(volumes, dequeuerVolume, APIConfigVolume(k8sName))

	return containers, volumes
}

func TaskContainers(api spec.API, job *spec.JobKey) ([]kcore.Container, []kcore.Volume) {
	containers, volumes := userPodContainers(api)
	k8sName := job.K8sName()

	volumes = append(volumes,
		KubexitVolume(),
		APIConfigVolume(k8sName),
	)

	containerNames := userconfig.GetContainerNames(api.Pod.Containers)
	for i, c := range containers {
		containers[i].VolumeMounts = append(containers[i].VolumeMounts,
			KubexitMount(),
			APIConfigMount(k8sName),
		)

		containerDeathDependencies := containerNames.Copy()
		containerDeathDependencies.Remove(c.Name)
		containerDeathEnvVars := getKubexitEnvVars(c.Name, containerDeathDependencies.Slice(), nil)
		containers[i].Env = append(containers[i].Env, containerDeathEnvVars...)

		if c.Command[0] != "/cortex/kubexit" {
			containers[i].Command = append([]string{"/cortex/kubexit"}, c.Command...)
		}
	}

	return containers, volumes
}

func BatchContainers(api spec.API, job *spec.BatchJob) ([]kcore.Container, []kcore.Volume) {
	userContainers, userVolumes := userPodContainers(api)
	dequeuerContainer, dequeuerVolume := batchDequeuerProxyContainer(api, job.ID, job.SQSUrl)

	// make sure the dequeuer starts first to allow it to start watching the graveyard before user containers begin
	containers := append([]kcore.Container{dequeuerContainer}, userContainers...)
	volumes := append([]kcore.Volume{dequeuerVolume}, userVolumes...)

	k8sName := job.K8sName()

	volumes = append(volumes,
		KubexitVolume(),
		APIConfigVolume(k8sName),
	)

	containerNames := userconfig.GetContainerNames(api.Pod.Containers)
	containerNames.Add(dequeuerContainer.Name)

	for i, c := range containers {
		containers[i].VolumeMounts = append(containers[i].VolumeMounts,
			KubexitMount(),
			APIConfigMount(k8sName),
		)

		containerDeathDependencies := containerNames.Copy()
		containerDeathDependencies.Remove(c.Name)
		containerDeathEnvVars := getKubexitEnvVars(c.Name, containerDeathDependencies.Slice(), nil)
		containers[i].Env = append(containers[i].Env, containerDeathEnvVars...)

		if c.Command[0] != "/cortex/kubexit" {
			containers[i].Command = append([]string{"/cortex/kubexit"}, c.Command...)
		}
	}

	return containers, volumes
}

func userPodContainers(api spec.API) ([]kcore.Container, []kcore.Volume) {
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

	var containers []kcore.Container
	for _, container := range api.Pod.Containers {
		containerResourceList := kcore.ResourceList{}
		containerResourceLimitsList := kcore.ResourceList{}
		securityContext := kcore.SecurityContext{
			Privileged: pointer.Bool(true),
		}

		var readinessProbe *kcore.Probe
		if api.Kind == userconfig.RealtimeAPIKind {
			readinessProbe = GetProbeSpec(container.ReadinessProbe)
		}

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

		if container.Compute.Inf > 0 {
			totalHugePages := container.Compute.Inf * _hugePagesMemPerInf
			containerResourceList["aws.amazon.com/neuron"] = *kresource.NewQuantity(container.Compute.Inf, kresource.DecimalSI)
			containerResourceList["hugepages-2Mi"] = *kresource.NewQuantity(totalHugePages, kresource.BinarySI)
			containerResourceLimitsList["aws.amazon.com/neuron"] = *kresource.NewQuantity(container.Compute.Inf, kresource.DecimalSI)
			containerResourceLimitsList["hugepages-2Mi"] = *kresource.NewQuantity(totalHugePages, kresource.BinarySI)

			securityContext.Capabilities = &kcore.Capabilities{
				Add: []kcore.Capability{
					"SYS_ADMIN",
					"IPC_LOCK",
				},
			}
		}

		if container.Compute.Shm != nil {
			volumes = append(volumes, ShmVolume(container.Compute.Shm.Quantity, "dshm-"+container.Name))
			containerMounts = append(containerMounts, ShmMount("dshm-"+container.Name))
		}

		containerEnvVars := BaseEnvVars

		containerEnvVars = append(containerEnvVars, kcore.EnvVar{
			Name:  "CORTEX_CLI_CONFIG_DIR",
			Value: _clientConfigDir,
		})

		if api.Kind != userconfig.TaskAPIKind {
			containerEnvVars = append(containerEnvVars, kcore.EnvVar{
				Name:  "CORTEX_PORT",
				Value: s.Int32(*api.Pod.Port),
			})
		}

		for k, v := range container.Env {
			containerEnvVars = append(containerEnvVars, kcore.EnvVar{
				Name:  k,
				Value: v,
			})
		}

		containers = append(containers, kcore.Container{
			Name:           container.Name,
			Image:          container.Image,
			Command:        container.Command,
			Args:           container.Args,
			Env:            containerEnvVars,
			VolumeMounts:   containerMounts,
			LivenessProbe:  GetProbeSpec(container.LivenessProbe),
			ReadinessProbe: readinessProbe,
			Resources: kcore.ResourceRequirements{
				Requests: containerResourceList,
				Limits:   containerResourceLimitsList,
			},
			ImagePullPolicy: kcore.PullAlways,
			SecurityContext: &securityContext,
		})
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
	var nodeGroups []*clusterconfig.NodeGroup
	for _, clusterNodeGroup := range config.ClusterConfig.NodeGroups {
		for _, apiNodeGroupName := range apiNodeGroups {
			if clusterNodeGroup.Name == apiNodeGroupName {
				nodeGroups = append(nodeGroups, clusterNodeGroup)
			}
		}
	}

	if apiNodeGroups == nil {
		nodeGroups = config.ClusterConfig.NodeGroups
	}

	var requiredNodeGroups []string
	var preferredAffinities []kcore.PreferredSchedulingTerm

	for _, nodeGroup := range nodeGroups {
		var nodeGroupPrefix string
		if nodeGroup.Spot {
			nodeGroupPrefix = "cx-ws-"
		} else {
			nodeGroupPrefix = "cx-wd-"
		}

		preferredAffinities = append(preferredAffinities, kcore.PreferredSchedulingTerm{
			Weight: int32(nodeGroup.Priority),
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

var BaseEnvVars = []kcore.EnvVar{
	{
		Name:  "CORTEX_VERSION",
		Value: consts.CortexVersion,
	},
	{
		Name:  "CORTEX_LOG_LEVEL",
		Value: strings.ToUpper(userconfig.InfoLogLevel.String()),
	},
}
