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
	"github.com/aws/aws-sdk-go/service/apigatewayv2"
	"github.com/cortexlabs/cortex/pkg/lib/urls"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/types/clusterconfig"
	"github.com/cortexlabs/cortex/pkg/types/spec"
	"github.com/cortexlabs/cortex/pkg/types/userconfig"
)

func addAPIToAPIGateway(loadBalancerScheme clusterconfig.LoadBalancerScheme, api *spec.API) error {
	if api.Networking.APIGateway == userconfig.NoneAPIGatewayType {
		return nil
	}

	apiGateway, err := APIGateway()
	if err != nil {
		return err
	}

	// check if API Gateway route already exists
	existingRoute, err := config.AWS.GetRoute(*apiGateway.ApiId, *api.Endpoint)
	if err != nil {
		return err
	} else if existingRoute == nil {
		return nil
	}

	if loadBalancerScheme == clusterconfig.InternalLoadBalancerScheme {
		vpcLink, err := config.AWS.GetVPCLinkByTag(clusterconfig.ClusterNameTag, config.Cluster.ClusterName)
		if err != nil {
			return err
		} else if vpcLink == nil {
			return ErrorNoVPCLink()
		}

		integration, err := config.AWS.GetVPCLinkIntegration(*apiGateway.ApiId, *vpcLink.VpcLinkId)
		if err != nil {
			return err
		} else if integration == nil {
			return ErrorNoVPCLinkIntegration()
		}

		err = config.AWS.CreateRoute(*apiGateway.ApiId, *integration.IntegrationId, *api.Endpoint)
		if err != nil {
			return err
		}
	}

	if loadBalancerScheme == clusterconfig.InternetFacingLoadBalancerScheme {
		loadBalancerURL, err := APILoadBalancerURL()
		if err != nil {
			return err
		}

		targetEndpoint := urls.Join(loadBalancerURL, *api.Endpoint)

		integrationID, err := config.AWS.CreateHTTPIntegration(*apiGateway.ApiId, targetEndpoint)
		if err != nil {
			return err
		}

		err = config.AWS.CreateRoute(*apiGateway.ApiId, integrationID, *api.Endpoint)
		if err != nil {
			return err
		}
	}

	return nil
}

func removeAPIFromAPIGateway(loadBalancerScheme clusterconfig.LoadBalancerScheme, api *spec.API) error {
	if api.Networking.APIGateway == userconfig.NoneAPIGatewayType {
		return nil
	}

	apiGateway, err := APIGateway()
	if err != nil {
		return err
	}

	if loadBalancerScheme == clusterconfig.InternalLoadBalancerScheme {
		_, err := config.AWS.DeleteRoute(*apiGateway.ApiId, *api.Endpoint)
		if err != nil {
			return err
		}
	}

	if loadBalancerScheme == clusterconfig.InternetFacingLoadBalancerScheme {
		integrationID, err := config.AWS.GetRouteIntegrationID(*apiGateway.ApiId, *api.Endpoint)
		if err != nil {
			return err
		}

		_, err = config.AWS.DeleteRoute(*apiGateway.ApiId, *api.Endpoint)
		if err != nil {
			return err
		}

		if integrationID != "" {
			err = config.AWS.DeleteIntegration(config.Cluster.ClusterName, integrationID)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// APIGateway returns the Cortex API Gateway (an error is returned if unable to find it)
func APIGateway() (*apigatewayv2.Api, error) {
	apiGateway, err := config.AWS.GetAPIGatewayByTag(clusterconfig.ClusterNameTag, config.Cluster.ClusterName)
	if err != nil {
		return nil, err
	} else if apiGateway == nil {
		return nil, ErrorNoAPIGateway()
	}

	return apiGateway, nil
}
