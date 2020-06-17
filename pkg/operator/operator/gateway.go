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
	"github.com/cortexlabs/cortex/pkg/lib/aws"
	"github.com/cortexlabs/cortex/pkg/lib/urls"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/types/clusterconfig"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	"istio.io/client-go/pkg/apis/networking/v1alpha3"
)

func addAPIToAPIGateway(endpoint string, apiGatewayType userconfig.APIGatewayType) error {
	if apiGatewayType == userconfig.NoneAPIGatewayType {
		return nil
	}

	apiGatewayID := *config.Cluster.APIGateway.ApiId

	// check if API Gateway route already exists
	existingRoute, err := config.AWS.GetRoute(apiGatewayID, endpoint)
	if err != nil {
		return err
	} else if existingRoute != nil {
		return nil
	}

	if config.Cluster.APILoadBalancerScheme == clusterconfig.InternalLoadBalancerScheme {
		err = config.AWS.CreateRoute(apiGatewayID, *config.Cluster.VPCLinkIntegration.IntegrationId, endpoint)
		if err != nil {
			return err
		}
	}

	if config.Cluster.APILoadBalancerScheme == clusterconfig.InternetFacingLoadBalancerScheme {
		loadBalancerURL, err := APILoadBalancerURL()
		if err != nil {
			return err
		}

		targetEndpoint := urls.Join(loadBalancerURL, endpoint)

		integrationID, err := config.AWS.CreateHTTPIntegration(apiGatewayID, targetEndpoint)
		if err != nil {
			return err
		}

		err = config.AWS.CreateRoute(apiGatewayID, integrationID, endpoint)
		if err != nil {
			return err
		}
	}

	return nil
}

func removeAPIFromAPIGateway(endpoint string, apiGatewayType userconfig.APIGatewayType) error {
	if apiGatewayType == userconfig.NoneAPIGatewayType {
		return nil
	}

	apiGatewayID := *config.Cluster.APIGateway.ApiId

	route, err := config.AWS.DeleteRoute(apiGatewayID, endpoint)
	if err != nil {
		return err
	}

	if config.Cluster.APILoadBalancerScheme == clusterconfig.InternetFacingLoadBalancerScheme && route != nil {
		integrationID := aws.ExtractRouteIntegrationID(route)
		if integrationID != "" {
			err = config.AWS.DeleteIntegration(apiGatewayID, integrationID)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func updateAPIGateway(
	prevEndpoint string,
	prevAPIGatewayType userconfig.APIGatewayType,
	newEndpoint string,
	newAPIGatewayType userconfig.APIGatewayType,
) error {

	if prevAPIGatewayType == userconfig.NoneAPIGatewayType && newAPIGatewayType == userconfig.NoneAPIGatewayType {
		return nil
	}

	if prevAPIGatewayType == userconfig.PublicAPIGatewayType && newAPIGatewayType == userconfig.NoneAPIGatewayType {
		return removeAPIFromAPIGateway(prevEndpoint, prevAPIGatewayType)
	}

	if prevAPIGatewayType == userconfig.NoneAPIGatewayType && newAPIGatewayType == userconfig.PublicAPIGatewayType {
		return addAPIToAPIGateway(newEndpoint, newAPIGatewayType)
	}

	if prevEndpoint == newEndpoint {
		return nil
	}

	// the endpoint has changed
	if err := addAPIToAPIGateway(newEndpoint, newAPIGatewayType); err != nil {
		return err
	}
	if err := removeAPIFromAPIGateway(prevEndpoint, prevAPIGatewayType); err != nil {
		return err
	}

	return nil
}

func removeAPIFromAPIGatewayK8s(virtualService *v1alpha3.VirtualService) error {
	if virtualService == nil {
		return nil // API is not running
	}

	apiGatewayType, err := userconfig.APIGatewayFromAnnotations(virtualService)
	if err != nil {
		return err
	}

	endpoint, err := GetEndpointFromVirtualService(virtualService)
	if err != nil {
		return err
	}

	return removeAPIFromAPIGateway(endpoint, apiGatewayType)
}

func updateAPIGatewayK8s(prevVirtualService *v1alpha3.VirtualService, newAPI *spec.API) error {
	prevAPIGatewayType, err := userconfig.APIGatewayFromAnnotations(prevVirtualService)
	if err != nil {
		return err
	}

	prevEndpoint, err := GetEndpointFromVirtualService(prevVirtualService)
	if err != nil {
		return err
	}

	return updateAPIGateway(prevEndpoint, prevAPIGatewayType, *newAPI.Endpoint, newAPI.Networking.APIGateway)
}
