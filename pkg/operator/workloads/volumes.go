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

package workloads

import (
	kcore "k8s.io/api/core/v1"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
)

func defaultVolumes() []kcore.Volume {
	return []kcore.Volume{
		k8s.EmptyDirVolume(consts.EmptyDirVolumeName),
	}
}

func defaultVolumeMounts() []kcore.VolumeMount {
	return []kcore.VolumeMount{
		k8s.EmptyDirVolumeMount(consts.EmptyDirVolumeName, consts.EmptyDirMountPath),
	}
}
