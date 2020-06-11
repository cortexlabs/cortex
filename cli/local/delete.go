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
	"fmt"

	"github.com/cortexlabs/cortex/pkg/lib/docker"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/cortexlabs/cortex/pkg/types/spec"
)

func Delete(apiName string, keepCache bool) (schema.DeleteResponse, error) {
	_, err := docker.GetDockerClient()
	if err != nil {
		return schema.DeleteResponse{}, err
	}

	var apiSpec *spec.API = nil
	if !keepCache {
		if apiSpec, err = FindAPISpec(apiName); err != nil {
			return schema.DeleteResponse{}, err
		}
	}

	err = errors.FirstError(
		DeleteAPI(apiName),
		DeleteCachedModels(apiName, apiSpec.ModelIDs()),
	)
	if err != nil {
		return schema.DeleteResponse{}, err
	}

	return schema.DeleteResponse{
		Message: fmt.Sprintf("deleting %s", apiName),
	}, nil
}
