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

package k8s

import (
	kcore "k8s.io/api/core/v1"
)

func EmptyDirVolume(volumeName string) kcore.Volume {
	return kcore.Volume{
		Name: volumeName,
		VolumeSource: kcore.VolumeSource{
			EmptyDir: &kcore.EmptyDirVolumeSource{},
		},
	}
}

func EmptyDirVolumeMount(volumeName string, mountPath string) kcore.VolumeMount {
	return kcore.VolumeMount{
		Name:      volumeName,
		MountPath: mountPath,
	}
}
