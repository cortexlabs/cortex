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
	"github.com/cortexlabs/cortex/pkg/lib/parallel"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/operator/deployment/batch"
	"github.com/cortexlabs/cortex/pkg/operator/deployment/sync"
	ok8s "github.com/cortexlabs/cortex/pkg/operator/k8s"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
)

// TODO rename this function
func UpdateAPI(apiConfig *userconfig.API, projectID string, force bool) (*spec.API, string, error) {
	virtualService, err := config.K8s.GetVirtualService(ok8s.K8sName(apiConfig.Name))
	if err != nil {
		return nil, "", err
	}

	if virtualService != nil {
		existingDeploymentType := userconfig.APITypeFromString(virtualService.GetLabels()["apiType"])
		if existingDeploymentType != apiConfig.Type {
			return nil, "", ErrorCannotChangeTypeOfDeployedAPI(apiConfig.Name, existingDeploymentType, apiConfig.Type)
		}
	}

	if apiConfig.Type == userconfig.APIAPIType {
		return sync.UpdateAPI(apiConfig, projectID, force)
	}

	return batch.UpdateAPI(apiConfig, projectID)
}

func RefreshAPI(apiName string, force bool) (string, error) {
	return sync.RefreshAPI(apiName, force)
}

// TODO get deployment kind first
func DeleteAPI(apiName string, keepCache bool) error {
	err := parallel.RunFirstErr(
		func() error {
			return sync.DeleteAPI(apiName, keepCache)
		},
		func() error {
			return batch.DeleteAPI(apiName, keepCache)
		},
	)

	if err != nil {
		return err
	}

	return nil
}

func IsAPIDeployed(apiName string) (bool, error) {
	virtualService, err := config.K8s.GetVirtualService(ok8s.K8sName(apiName))
	if err != nil {
		return false, err
	}
	return virtualService != nil, nil
}

func APIsBaseURL() (string, error) {
	service, err := config.K8sIstio.GetService("ingressgateway-apis")
	if err != nil {
		return "", err
	}
	if service == nil {
		return "", ErrorCortexInstallationBroken()
	}
	if len(service.Status.LoadBalancer.Ingress) == 0 {
		return "", ErrorLoadBalancerInitializing()
	}
	return "http://" + service.Status.LoadBalancer.Ingress[0].Hostname, nil
}
