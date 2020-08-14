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
	"strings"

	"github.com/cortexlabs/cortex/pkg/consts"
	"github.com/cortexlabs/cortex/pkg/lib/docker"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/prompt"
	"github.com/cortexlabs/cortex/pkg/operator/schema"
	"github.com/cortexlabs/cortex/pkg/types/spec"
)

func Delete(apiName string, keepCache, deleteForce bool) (schema.DeleteResponse, error) {
	_, err := docker.GetDockerClient()
	if err != nil {
		return schema.DeleteResponse{}, err
	}

	var apiSpec *spec.API = nil
	if apiSpec, err = FindAPISpec(apiName); err != nil {
		if errors.GetKind(err) == ErrCortexVersionMismatch {
			var incompatibleVersion string
			if incompatibleVersion, err = GetVersionFromAPISpec(apiName); err != nil {
				return schema.DeleteResponse{}, err
			}

			incompatibleMinorVersion := strings.Join(strings.Split(incompatibleVersion, ".")[:2], ".")
			if consts.CortexVersionMinor != incompatibleMinorVersion && !deleteForce {
				prompt.YesOrExit(
					fmt.Sprintf(
						"api %s was deployed using CLI version %s but the current CLI version is %s; "+
							"deleting %s with current CLI version %s might lead to an unexpected state; any cached models won't be deleted\n\n"+
							"it is recommended to download version %s of the CLI from https://docs.cortex.dev/v/%s/install, delete the API using version %s of the CLI and then re-deploy the API using the latest version of the CLI\n\n"+
							"do you still want to delete?",
						apiName, incompatibleMinorVersion, consts.CortexVersionMinor, apiName, consts.CortexVersionMinor, incompatibleMinorVersion, incompatibleMinorVersion, incompatibleMinorVersion),
					"", "",
				)
			}

			if err = DeleteAPI(apiName); err != nil {
				return schema.DeleteResponse{}, err
			}
			return schema.DeleteResponse{
				Message: fmt.Sprintf("deleting api %s with current CLI version %s", apiName, consts.CortexVersion),
			}, nil
		}

		return schema.DeleteResponse{}, DeleteAPI(apiName)
	}

	if keepCache {
		err = DeleteAPI(apiName)
	} else {
		err = errors.FirstError(
			DeleteAPI(apiName),
			DeleteCachedModels(apiName, apiSpec.ModelIDs()),
		)
	}
	if err != nil {
		return schema.DeleteResponse{}, err
	}

	return schema.DeleteResponse{
		Message: fmt.Sprintf("deleting %s", apiName),
	}, nil
}
