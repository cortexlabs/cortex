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

package apisplitter

import (
	"sort"

	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/operator/operator"
	"github.com/cortexlabs/cortex/pkg/types/status"
	istioclientnetworking "istio.io/client-go/pkg/apis/networking/v1alpha3"
)

func GetStatus(apiName string) (*status.Status, error) {
	var virtualService *istioclientnetworking.VirtualService

	virtualService, err := config.K8s.GetVirtualService(operator.K8sName(apiName))
	if err != nil {
		return nil, err
	}
	if virtualService == nil {
		return nil, errors.ErrorUnexpected("unable to find trafficsplitter", apiName)
	}

	return trafficSplitterStatus(virtualService)
}

func GetAllStatuses() ([]status.Status, error) {
	var virtualServices []istioclientnetworking.VirtualService

	virtualServices, err := config.K8s.ListVirtualServicesByLabel("apiKind", "APISplitter")
	if err != nil {
		return nil, err
	}

	statuses := make([]status.Status, len(virtualServices))
	for i, virtualService := range virtualServices {
		status, err := trafficSplitterStatus(&virtualService)
		if err != nil {
			return nil, err
		}
		statuses[i] = *status
	}
	sort.Slice(statuses, func(i, j int) bool {
		return statuses[i].APIName < statuses[j].APIName
	})

	return statuses, nil
}

func trafficSplitterStatus(virtualService *istioclientnetworking.VirtualService) (*status.Status, error) {

	statusResponse := &status.Status{}
	statusResponse.APIName = virtualService.Labels["apiName"]
	statusResponse.APIID = virtualService.Labels["apiID"]
	// if virtual service deploy the trafficsplitter is actice
	// maybe need to check if backends are active
	statusResponse.Code = status.Live

	return statusResponse, nil
}
