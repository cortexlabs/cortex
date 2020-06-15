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

package local

import (
	"github.com/cortexlabs/cortex/pkg/lib/docker"
)

func StreamLogs(apiName string) error {
	_, err := docker.GetDockerClient()
	if err != nil {
		return err
	}

	_, err = FindAPISpec(apiName)
	if err != nil {
		return err
	}

	containers, err := GetContainersByAPI(apiName)
	if err != nil {
		return err
	}

	if len(containers) == 0 {
		return ErrorAPIContainersNotFound(apiName)
	}

	var containerIDs []string
	for _, container := range containers {
		containerIDs = append(containerIDs, container.ID)
	}

	return docker.StreamDockerLogs(containerIDs[0], containerIDs[1:]...)
}
