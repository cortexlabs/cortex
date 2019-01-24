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
	corev1 "k8s.io/api/core/v1"

	"github.com/cortexlabs/cortex/pkg/consts"
)

func EmptyDirVolume() corev1.Volume {
	volume := corev1.Volume{
		Name: consts.EmptyDirVolumeName,
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}
	return volume
}

func EmptyDirVolumeMount() corev1.VolumeMount {
	volumeMount := corev1.VolumeMount{
		Name:      consts.EmptyDirVolumeName,
		MountPath: consts.EmptyDirMountPath,
	}
	return volumeMount
}

func CortexConfigVolume() corev1.Volume {
	volume := corev1.Volume{
		Name: consts.CortexConfigName,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: consts.CortexConfigName,
				},
			},
		},
	}
	return volume
}

func CortexConfigVolumeMount() corev1.VolumeMount {
	volumeMount := corev1.VolumeMount{
		Name:      consts.CortexConfigName,
		MountPath: consts.CortexConfigPath,
	}
	return volumeMount
}

func DefaultVolumes() []corev1.Volume {
	volumes := []corev1.Volume{
		EmptyDirVolume(),
	}
	return volumes
}

func DefaultVolumeMounts() []corev1.VolumeMount {
	volumeMounts := []corev1.VolumeMount{
		EmptyDirVolumeMount(),
	}
	return volumeMounts
}
