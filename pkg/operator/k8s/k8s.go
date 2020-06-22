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

package k8s

import (
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/k8s"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	istioclientnetworking "istio.io/client-go/pkg/apis/networking/v1alpha3"
)

func Name(apiName string) string {
	return "api-" + apiName
}

// APILoadBalancerURL returns http endpoint of cluster ingress elb
func APILoadBalancerURL() (string, error) {
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

func GetEndpointFromVirtualService(virtualService *istioclientnetworking.VirtualService) (string, error) {
	endpoints := k8s.ExtractVirtualServiceEndpoints(virtualService)

	if len(endpoints) != 1 {
		return "", errors.ErrorUnexpected("expected 1 endpoint, but got", endpoints)
	}

	return endpoints.GetOne(), nil
}
