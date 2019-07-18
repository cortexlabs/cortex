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

package k8s

import (
	kcore "k8s.io/api/core/v1"

	"github.com/cortexlabs/cortex/pkg/consts"
)

func EmptyDirVolume() kcore.Volume {
	volume := kcore.Volume{
		Name: consts.EmptyDirVolumeName,
		VolumeSource: kcore.VolumeSource{
			EmptyDir: &kcore.EmptyDirVolumeSource{},
		},
	}
	return volume
}

func EmptyDirVolumeMount() kcore.VolumeMount {
	volumeMount := kcore.VolumeMount{
		Name:      consts.EmptyDirVolumeName,
		MountPath: consts.EmptyDirMountPath,
	}
	return volumeMount
}

func CortexConfigVolume() kcore.Volume {
	volume := kcore.Volume{
		Name: consts.CortexConfigName,
		VolumeSource: kcore.VolumeSource{
			ConfigMap: &kcore.ConfigMapVolumeSource{
				LocalObjectReference: kcore.LocalObjectReference{
					Name: consts.CortexConfigName,
				},
			},
		},
	}
	return volume
}

func CortexConfigVolumeMount() kcore.VolumeMount {
	volumeMount := kcore.VolumeMount{
		Name:      consts.CortexConfigName,
		MountPath: consts.CortexConfigPath,
	}
	return volumeMount
}

func DefaultVolumes() []kcore.Volume {
	volumes := []kcore.Volume{
		EmptyDirVolume(),
	}
	return volumes
}

func DefaultVolumeMounts() []kcore.VolumeMount {
	volumeMounts := []kcore.VolumeMount{
		EmptyDirVolumeMount(),
	}
	return volumeMounts
}
