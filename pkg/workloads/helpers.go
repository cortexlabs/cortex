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
	"path"
	"strings"

	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	kcore "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func K8sName(apiName string) string {
	return "api-" + apiName
}

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

func getProbeSpec(probe *userconfig.Probe) *kcore.Probe {
	if probe == nil {
		return nil
	}

	var httpGetAction *kcore.HTTPGetAction
	var tcpSocketAction *kcore.TCPSocketAction
	var execAction *kcore.ExecAction

	if probe.HTTPGet != nil {
		httpGetAction = &kcore.HTTPGetAction{
			Path: probe.HTTPGet.Path,
			Port: intstr.IntOrString{
				IntVal: probe.HTTPGet.Port,
			},
		}
	}
	if probe.TCPSocket != nil {
		tcpSocketAction = &kcore.TCPSocketAction{
			Port: intstr.IntOrString{
				IntVal: probe.TCPSocket.Port,
			},
		}
	}
	if probe.Exec != nil {
		execAction = &kcore.ExecAction{
			Command: probe.Exec.Command,
		}
	}

	return &kcore.Probe{
		Handler: kcore.Handler{
			HTTPGet:   httpGetAction,
			TCPSocket: tcpSocketAction,
			Exec:      execAction,
		},
		InitialDelaySeconds: probe.InitialDelaySeconds,
		TimeoutSeconds:      probe.TimeoutSeconds,
		PeriodSeconds:       probe.PeriodSeconds,
		SuccessThreshold:    probe.SuccessThreshold,
		FailureThreshold:    probe.FailureThreshold,
	}
}

func baseClusterEnvVars() []kcore.EnvFromSource {
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

func getKubexitEnvVars(containerName string, deathDeps []string, birthDeps []string) []kcore.EnvVar {
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

	if birthDeps != nil {
		envVars = append(envVars,
			kcore.EnvVar{
				Name:  "KUBEXIT_BIRTH_DEPS",
				Value: strings.Join(birthDeps, ","),
			},
			kcore.EnvVar{
				Name:  "KUBEXIT_IGNORE_CODE_ON_DEATH_DEPS",
				Value: "true",
			},
		)
	}

	return envVars
}

func MntVolume() kcore.Volume {
	return k8s.EmptyDirVolume(_emptyDirVolumeName)
}

func CortexVolume() kcore.Volume {
	return k8s.EmptyDirVolume(_cortexDirVolumeName)
}

func SpecVolume(name string) kcore.Volume {
	return kcore.Volume{
		Name: name,
		VolumeSource: kcore.VolumeSource{
			ConfigMap: &kcore.ConfigMapVolumeSource{
				LocalObjectReference: kcore.LocalObjectReference{
					Name: name,
				},
			},
		},
	}
}

func ClientConfigVolume() kcore.Volume {
	return kcore.Volume{
		Name: _clientConfigDirVolume,
		VolumeSource: kcore.VolumeSource{
			ConfigMap: &kcore.ConfigMapVolumeSource{
				LocalObjectReference: kcore.LocalObjectReference{
					Name: _clientConfigConfigMap,
				},
			},
		},
	}
}

func ClusterConfigVolume() kcore.Volume {
	return kcore.Volume{
		Name: _clusterConfigDirVolume,
		VolumeSource: kcore.VolumeSource{
			ConfigMap: &kcore.ConfigMapVolumeSource{
				LocalObjectReference: kcore.LocalObjectReference{
					Name: _clusterConfigConfigMap,
				},
			},
		},
	}
}

func ShmVolume(q resource.Quantity) kcore.Volume {
	return kcore.Volume{
		Name: _shmDirVolumeName,
		VolumeSource: kcore.VolumeSource{
			EmptyDir: &kcore.EmptyDirVolumeSource{
				Medium:    kcore.StorageMediumMemory,
				SizeLimit: k8s.QuantityPtr(q),
			},
		},
	}
}

func KubexitVolume() kcore.Volume {
	return k8s.EmptyDirVolume(_kubexitGraveyardName)
}

func MntMount() kcore.VolumeMount {
	return k8s.EmptyDirVolumeMount(_emptyDirVolumeName, _emptyDirMountPath)
}

func CortexMount() kcore.VolumeMount {
	return k8s.EmptyDirVolumeMount(_cortexDirVolumeName, _cortexDirMountPath)
}

func SpecMount(name string) kcore.VolumeMount {
	return kcore.VolumeMount{
		Name:      name,
		MountPath: path.Join(_cortexDirMountPath, "job_spec.json"),
		SubPath:   "job_spec.json",
	}
}

func ClientConfigMount() kcore.VolumeMount {
	return kcore.VolumeMount{
		Name:      _clientConfigDirVolume,
		MountPath: path.Join(_clientConfigDir, "cli.yaml"),
		SubPath:   "cli.yaml",
	}
}

func ClusterConfigMount() kcore.VolumeMount {
	return kcore.VolumeMount{
		Name:      _clusterConfigDirVolume,
		MountPath: path.Join(_clusterConfigDir, "cluster.yaml"),
		SubPath:   "cluster.yaml",
	}
}

func ShmMount() kcore.VolumeMount {
	return k8s.EmptyDirVolumeMount(_shmDirVolumeName, _shmDirMountPath)
}

func KubexitMount() kcore.VolumeMount {
	return k8s.EmptyDirVolumeMount(_kubexitGraveyardName, _kubexitGraveyardMountPath)
}
