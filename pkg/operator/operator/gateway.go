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
	"github.com/cortexlabs/cortex/pkg/lib/urls"
	"github.com/cortexlabs/cortex/pkg/operator/config"
	"github.com/cortexlabs/cortex/pkg/types/clusterconfig"
	"github.com/cortexlabs/cortex/pkg/types/spec"
)

func addAPItoAPIGateway(loadBalancerScheme clusterconfig.LoadBalancerScheme, api *spec.API) error {
	// internal facing API loadbalancer
	if loadBalancerScheme.String() == "internal" {
		// API should be exposed to public with API gateway
		if api.Networking.APIGateway.String() == "public" {
			integrationID, err := config.AWS.GetIntegrationIDInternal(config.Cluster.ClusterName)
			if err != nil {
				return err
			}
			err = config.AWS.CreateRouteWithIntegration(config.Cluster.ClusterName, integrationID, *api.Endpoint)
			if err != nil {
				return err
			}
		}
	}
	// public facing API loadbalancer
	if loadBalancerScheme.String() == "internet-facing" {
		if api.Networking.APIGateway.String() == "public" {
			endpointURL, err := APIsInternalBaseURL()
			if err != nil {
				return err
			}
			endpointURL = urls.Join(endpointURL, *api.Endpoint)
			integrationID, err := config.AWS.CreateHTTPIntegration(config.Cluster.ClusterName, endpointURL)
			if err != nil {
				return err
			}
			err = config.AWS.CreateRouteWithIntegration(config.Cluster.ClusterName, integrationID, *api.Endpoint)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func removeAPIfromAPIGateway(loadBalancerScheme clusterconfig.LoadBalancerScheme, api *spec.API) error {
	if loadBalancerScheme.String() == "internal" {
		if api.Networking.APIGateway.String() == "public" {
			err := config.AWS.DeleteAPIGatewayRoute(config.Cluster.ClusterName, *api.Endpoint)
			if err != nil {
				return err
			}
		}
	}
	if loadBalancerScheme.String() == "internet-facing" {
		if api.Networking.APIGateway.String() == "public" {
			integrationID, err := config.AWS.GetIntegrationIDofRoute(config.Cluster.ClusterName, *api.Endpoint)
			if err != nil {
				return err
			}
			err = config.AWS.DeleteAPIGatewayRoute(config.Cluster.ClusterName, *api.Endpoint)
			if err != nil {
				return err
			}
			err = config.AWS.DeleteIntegration(config.Cluster.ClusterName, integrationID)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
