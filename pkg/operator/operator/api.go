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

package operator

import (
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/parallel"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	ok8s "github.com/cortexlabs/cortex/pkg/operator/k8s"
	"github.com/cortexlabs/cortex/pkg/operator/resources"
	"github.com/cortexlabs/cortex/pkg/operator/resources/syncapi"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
)

func UpdateAPI(apiConfig *userconfig.API, projectID string, force bool) (*spec.API, string, error) {
	deployedResource, err := resources.FindDeployedResourceByName(apiConfig.Name)
	if err != nil {
		return nil, "", err
	}

	if deployedResource != nil && deployedResource.Kind != apiConfig.Kind {
		return nil, "", ErrorCannotChangeKindOfDeployedResource(apiConfig.Name, apiConfig.Kind, deployedResource.Kind)
	}

	if apiConfig.Kind == userconfig.SyncAPIKind {
		return syncapi.UpdateAPI(apiConfig, projectID, force)
	}

	return nil, "", errors.Wrap(ErrorKindNotSupported(apiConfig.Kind), apiConfig.Identify()) // unexpected
}

func RefreshAPI(apiName string, force bool) (string, error) {
	deployedResource, err := resources.FindDeployedResourceByName(apiName)
	if err != nil {
		return "", err
	} else if deployedResource == nil {
		return "", ErrorAPINotDeployed(deployedResource.Name)
	}

	if deployedResource.Kind == userconfig.SyncAPIKind {
		return syncapi.RefreshAPI(apiName, force)
	}

	return "", errors.Wrap(ErrorKindNotSupported(deployedResource.Kind), deployedResource.Identify()) // unexpected
}

func DeleteAPI(apiName string, keepCache bool) error {
	err := parallel.RunFirstErr(
		func() error {
			return syncapi.DeleteAPI(apiName, keepCache)
		},
	)

	if err != nil {
		return err
	}

	return nil
}

// APIBaseURL returns BaseURL of the API without resource endpoint
func APIBaseURL(api *spec.API) (string, error) {
	if api.Networking.APIGateway == userconfig.PublicAPIGatewayType {
		return *config.Cluster.APIGateway.ApiEndpoint, nil
	}
	return ok8s.APILoadBalancerURL()
}
