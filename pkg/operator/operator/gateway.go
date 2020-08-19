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
	"github.com/cortexlabs/cortex/pkg/lib/errors"
	"github.com/cortexlabs/cortex/pkg/lib/urls"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/types/clusterconfig"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
	istioclientnetworking "istio.io/client-go/pkg/apis/networking/v1alpha3"
)

type routeToIntegrationMapping struct {
	APIGatewayRoute  string
	IntegrationRoute string
}

func getRouteToIntegrationMapping(apiEndpoint string, isRoutePrefix bool) []routeToIntegrationMapping {
	mappings := []routeToIntegrationMapping{
		{
			apiEndpoint,
			apiEndpoint,
		},
	}

	if isRoutePrefix {
		// Use {proxy+} instead of {proxy} for greedy matching.
		// e.g. if the api gateway route is: /api_name/{proxy}
		// /api_name/a   MATCH
		// /api_name/a/  will not MATCH
		// /api_name/a/b will not MATCH

		// {proxy} is being used for now because greedy matching is not required at the moment.

		// Regardless of whether api gateway route uses {proxy} or {proxy+}, the integration route should always use {proxy}
		mappings = append(mappings, routeToIntegrationMapping{APIGatewayRoute: urls.Join(apiEndpoint, "{proxy}"), IntegrationRoute: urls.Join(apiEndpoint, "{proxy}")})
	}

	return mappings
}

func AddAPIToAPIGateway(apiEndpoint string, apiGatewayType userconfig.APIGatewayType, isRoutePrefix bool) error {
	if config.Cluster.APIGateway == nil {
		return nil
	}

	if apiGatewayType == userconfig.NoneAPIGatewayType {
		return nil
	}

	routeToIntegrationMapping := getRouteToIntegrationMapping(apiEndpoint, isRoutePrefix)

	apiGatewayID := *config.Cluster.APIGateway.ApiId

	errs := []error{}

	for _, routeMap := range routeToIntegrationMapping {
		// check if API Gateway route already exists
		existingRoute, err := config.AWS.GetRoute(apiGatewayID, routeMap.APIGatewayRoute)
		if err != nil {
			errs = append(errs, err)
			continue
		} else if existingRoute != nil {
			return nil
		}

		if config.Cluster.APILoadBalancerScheme == clusterconfig.InternalLoadBalancerScheme {
			err = config.AWS.CreateRoute(apiGatewayID, *config.Cluster.VPCLinkIntegration.IntegrationId, routeMap.APIGatewayRoute)
			if err != nil {
				errs = append(errs, err)
				continue
			}
		}

		if config.Cluster.APILoadBalancerScheme == clusterconfig.InternetFacingLoadBalancerScheme {
			loadBalancerURL, err := APILoadBalancerURL()
			if err != nil {
				errs = append(errs, err)
				continue
			}

			targetEndpoint := urls.Join(loadBalancerURL, routeMap.IntegrationRoute)

			integrationID, err := config.AWS.CreateHTTPIntegration(apiGatewayID, targetEndpoint)
			if err != nil {
				errs = append(errs, err)
				continue
			}

			err = config.AWS.CreateRoute(apiGatewayID, integrationID, routeMap.APIGatewayRoute)
			if err != nil {
				errs = append(errs, err)
				continue
			}
		}
	}
	return errors.FirstError(errs...)
}

func RemoveAPIFromAPIGateway(apiEndpoint string, apiGatewayType userconfig.APIGatewayType, isRoutePrefix bool) error {
	if config.Cluster.APIGateway == nil {
		return nil
	}

	if apiGatewayType == userconfig.NoneAPIGatewayType {
		return nil
	}

	routeToIntegrationMapping := getRouteToIntegrationMapping(apiEndpoint, isRoutePrefix)

	apiGatewayID := *config.Cluster.APIGateway.ApiId

	errs := []error{}

	for _, routeMap := range routeToIntegrationMapping {
		route, err := config.AWS.DeleteRoute(apiGatewayID, routeMap.APIGatewayRoute)
		if err != nil {
			errs = append(errs, err)
		}

		if config.Cluster.APILoadBalancerScheme == clusterconfig.InternetFacingLoadBalancerScheme && route != nil {
			integrationID := aws.ExtractRouteIntegrationID(route)
			if integrationID != "" {
				err = config.AWS.DeleteIntegration(apiGatewayID, integrationID)
				if err != nil {
					errs = append(errs, err)
				}
			}
		}
	}
	return errors.FirstError(errs...)
}

func UpdateAPIGateway(
	prevEndpoint string,
	prevAPIGatewayType userconfig.APIGatewayType,
	newEndpoint string,
	newAPIGatewayType userconfig.APIGatewayType,
	isRoutePrefix bool,
) error {
	if config.Cluster.APIGateway == nil {
		return nil
	}

	if prevAPIGatewayType == userconfig.NoneAPIGatewayType && newAPIGatewayType == userconfig.NoneAPIGatewayType {
		return nil
	}

	if prevAPIGatewayType == userconfig.PublicAPIGatewayType && newAPIGatewayType == userconfig.NoneAPIGatewayType {
		return RemoveAPIFromAPIGateway(prevEndpoint, prevAPIGatewayType, isRoutePrefix)
	}

	if prevAPIGatewayType == userconfig.NoneAPIGatewayType && newAPIGatewayType == userconfig.PublicAPIGatewayType {
		return AddAPIToAPIGateway(newEndpoint, newAPIGatewayType, isRoutePrefix)
	}

	if prevEndpoint == newEndpoint {
		return nil
	}

	// the endpoint has changed
	if err := AddAPIToAPIGateway(newEndpoint, newAPIGatewayType, isRoutePrefix); err != nil {
		return err
	}
	if err := RemoveAPIFromAPIGateway(prevEndpoint, prevAPIGatewayType, isRoutePrefix); err != nil {
		return err
	}

	return nil
}

func RemoveAPIFromAPIGatewayK8s(virtualService *istioclientnetworking.VirtualService, isRoutePrefix bool) error {
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

	return RemoveAPIFromAPIGateway(endpoint, apiGatewayType, isRoutePrefix)
}

func UpdateAPIGatewayK8s(prevVirtualService *istioclientnetworking.VirtualService, newAPI *spec.API, isRoutePrefix bool) error {
	prevAPIGatewayType, err := userconfig.APIGatewayFromAnnotations(prevVirtualService)
	if err != nil {
		return err
	}

	prevEndpoint, err := GetEndpointFromVirtualService(prevVirtualService)
	if err != nil {
		return err
	}

	return UpdateAPIGateway(prevEndpoint, prevAPIGatewayType, *newAPI.Networking.Endpoint, newAPI.Networking.APIGateway, isRoutePrefix)
}
