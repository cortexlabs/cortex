/*
Copyright 2022 Cortex Labs, Inc.

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
	"strings"

	"github.com/cortexlabs/cortex/pkg/config"
	"github.com/cortexlabs/cortex/pkg/lib/urls"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
)

// APILoadBalancerURL returns the http endpoint of the ingress load balancer for deployed APIs
func APILoadBalancerURL() (string, error) {
	return getLoadBalancerURL("ingressgateway-apis")
}

// LoadBalancerURL returns the http endpoint of the ingress load balancer for the operator
func LoadBalancerURL() (string, error) {
	return getLoadBalancerURL("ingressgateway-operator")
}

func getLoadBalancerURL(name string) (string, error) {
	service, err := config.K8sIstio.GetService(name)
	if err != nil {
		return "", err
	}
	if service == nil {
		return "", ErrorCortexInstallationBroken()
	}
	if len(service.Status.LoadBalancer.Ingress) == 0 {
		return "", ErrorLoadBalancerInitializing()
	}
	if service.Status.LoadBalancer.Ingress[0].Hostname != "" {
		return "http://" + service.Status.LoadBalancer.Ingress[0].Hostname, nil
	}
	return "http://" + service.Status.LoadBalancer.Ingress[0].IP, nil
}

func APIEndpoint(api *spec.API) (string, error) {
	var err error
	baseAPIEndpoint := ""

	baseAPIEndpoint, err = APILoadBalancerURL()
	if err != nil {
		return "", err
	}
	baseAPIEndpoint = strings.Replace(baseAPIEndpoint, "https://", "http://", 1)

	return urls.Join(baseAPIEndpoint, *api.Networking.Endpoint), nil
}

func APIEndpointFromResource(deployedResource *DeployedResource) (string, error) {
	apiEndpoint, err := userconfig.EndpointFromAnnotation(deployedResource.VirtualService)
	if err != nil {
		return "", err
	}

	baseAPIEndpoint := ""

	baseAPIEndpoint, err = APILoadBalancerURL()
	if err != nil {
		return "", err
	}
	baseAPIEndpoint = strings.Replace(baseAPIEndpoint, "https://", "http://", 1)

	return urls.Join(baseAPIEndpoint, apiEndpoint), nil
}
