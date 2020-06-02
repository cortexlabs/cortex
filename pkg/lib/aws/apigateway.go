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

package aws

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/service/apigatewayv2"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
)

// GetVpcLinkID vpc-link id only exists if internal facing loadbalancer
func (c *Client) getVpcLinkID(clusterName string) (string, error) {
	vpcLinks, err := c.APIGatewayv2().GetVpcLinks(&apigatewayv2.GetVpcLinksInput{})
	if err != nil {
		return "", errors.Wrap(err, "failed to get VPC links")
	}

	// find vpc link with tag cortex.dev/cluster-name=<CLUSTERNAME>
	for _, vpcLink := range vpcLinks.Items {
		for tag, value := range vpcLink.Tags {
			if tag == "cortex.dev/cluster-name" && *value == clusterName {
				return *vpcLink.VpcLinkId, nil
			}
		}
	}
	return "", fmt.Errorf("failed to find cortex VPC link")
}

// GetAPIGatewayID gets the API ID of the API Gateway api
func (c *Client) getAPIGatewayID(clusterName string) (string, error) {
	apis, err := c.APIGatewayv2().GetApis(&apigatewayv2.GetApisInput{})
	if err != nil {
		return "", errors.Wrap(err, "failed to get API gateway apis")
	}

	// find API with tag cortex.dev/cluster-name=<CLUSTERNAME>
	for _, api := range apis.Items {
		for tag, value := range api.Tags {
			if tag == "cortex.dev/cluster-name" && *value == clusterName {
				return *api.ApiId, nil
			}
		}
	}
	return "", fmt.Errorf("failed to find API gateway apis")
}

// GetIntegrationIDInternal gets the ID of the created integration for the internal ELB
func (c *Client) GetIntegrationIDInternal(clusterName string) (string, error) {
	apiID, err := c.getAPIGatewayID(clusterName)
	if err != nil {
		return "", err
	}
	vpcLinkID, err := c.getVpcLinkID(clusterName)
	if err != nil {
		return "", err
	}

	integrations, err := c.APIGatewayv2().GetIntegrations(&apigatewayv2.GetIntegrationsInput{
		ApiId: &apiID,
	})
	if err != nil {
		return "", errors.Wrap(err, "failed to get API gateway integrations")
	}

	// find integration which is connected to the cortex VPC link
	for _, integration := range integrations.Items {
		if *integration.ConnectionId == vpcLinkID {
			return *integration.IntegrationId, nil
		}
	}
	// check that there is one integration. There should only be one if we have internal facing loadbalancer
	// if len(integrations.Items) == 0 {
	// 	integrationID := *integrations.Items[0].IntegrationId

	// 	return integrationID, nil
	// }

	return "", fmt.Errorf("no or too many integrations found for API with ID: %v", apiID)
}

// GetRouteID retrieves Route ID
func (c *Client) getRouteID(clusterName string, apiEndpoint string) (string, error) {
	apiID, err := c.getAPIGatewayID(clusterName)
	if err != nil {
		return "", err
	}

	routes, err := c.APIGatewayv2().GetRoutes(&apigatewayv2.GetRoutesInput{
		ApiId: &apiID,
	})
	if err != nil {
		return "", errors.Wrap(err, "failed to get routes for API with endpoint", apiEndpoint, " and ID", apiID)
	}

	// find route which matches the endpoint
	for _, route := range routes.Items {
		if *route.RouteKey == "ANY "+apiEndpoint {
			return *route.RouteId, nil
		}
	}
	return "", fmt.Errorf("failed to find route for API with endpoint %v and ID %v", apiEndpoint, apiID)
}

// GetAPIGatewayEndpoint return URL for API Gateway endpoint
func (c *Client) GetAPIGatewayEndpoint(clusterName string) (string, error) {
	apiID, err := c.getAPIGatewayID(clusterName)
	if err != nil {
		return "", err
	}

	api, err := c.APIGatewayv2().GetApi(&apigatewayv2.GetApiInput{
		ApiId: &apiID,
	})
	if err != nil {
		return "", errors.Wrap(err, "failed to get API with ID", apiID)
	}
	return *api.ApiEndpoint, nil
}

// GetIntegrationIDofRoute return the integrationID of the integration which is attached to a endpoint route
func (c *Client) GetIntegrationIDofRoute(clusterName string, apiEndpoint string) (string, error) {
	apiID, err := c.getAPIGatewayID(clusterName)
	if err != nil {
		return "", err
	}

	routeID, err := c.getRouteID(clusterName, apiEndpoint)
	if err != nil {
		return "", err
	}

	getRouteResponse, err := c.APIGatewayv2().GetRoute(&apigatewayv2.GetRouteInput{
		ApiId:   &apiID,
		RouteId: &routeID,
	})
	if err != nil {
		return "", errors.Wrap(err, "failed to get route for API with endpoint", apiEndpoint)
	}
	// trim of prefix of integrationID.
	// Note: Integrations get attached to routes via a target of the format integrations/<integrationID>
	integrationID := strings.Trim(*getRouteResponse.Target, "integrations/")
	return integrationID, nil
}

// CreateRouteWithIntegration creates a new route and attaches the route to the integration
func (c *Client) CreateRouteWithIntegration(clusterName string, integrationID string, apiEndpoint string) error {
	apiID, err := c.getAPIGatewayID(clusterName)
	if err != nil {
		return err
	}

	// create route key which includes ANY method and endpoint path
	route := "ANY " + apiEndpoint
	// Note: Integrations get attached to routes via a target of the format integrations/<integrationID>
	target := "integrations/" + integrationID
	_, err = c.APIGatewayv2().CreateRoute(&apigatewayv2.CreateRouteInput{
		ApiId:    &apiID,
		RouteKey: &route,
		Target:   &target,
	})
	if err != nil {
		return errors.Wrap(err, "failed to create route for API with ID", apiID)
	}
	return nil
}

// CreateHTTPIntegration creates new HTTP integration for API Gateway returns IntegrationID
func (c *Client) CreateHTTPIntegration(clusterName string, endpointURL string) (string, error) {
	apiID, err := c.getAPIGatewayID(clusterName)
	if err != nil {
		return "", err
	}

	integrationType := "HTTP_PROXY"
	payloadFormatVersion := "1.0"
	integrationMethod := "ANY"
	integrationResponse, err := c.APIGatewayv2().CreateIntegration(&apigatewayv2.CreateIntegrationInput{
		ApiId:                &apiID,
		IntegrationType:      &integrationType,
		IntegrationUri:       &endpointURL,
		PayloadFormatVersion: &payloadFormatVersion,
		IntegrationMethod:    &integrationMethod,
	})
	if err != nil {
		return "", errors.Wrap(err, "failed to create API gateway integration for endpoint", endpointURL)
	}
	return *integrationResponse.IntegrationId, nil
}

// DeleteIntegration deletes integration from API gateway
func (c *Client) DeleteIntegration(clusterName string, integrationID string) error {
	apiID, err := c.getAPIGatewayID(clusterName)
	if err != nil {
		return err
	}

	_, err = c.APIGatewayv2().DeleteIntegration(&apigatewayv2.DeleteIntegrationInput{
		ApiId:         &apiID,
		IntegrationId: &integrationID,
	})
	if err != nil {
		return errors.Wrap(err, "failed to delete HTTP integration with ID", integrationID)
	}
	return nil
}

// DeleteAPIGatewayRoute deletes route from API Gateway
func (c *Client) DeleteAPIGatewayRoute(clusterName string, apiEndpoint string) error {
	apiID, err := c.getAPIGatewayID(clusterName)
	if err != nil {
		return err
	}

	routeID, err := c.getRouteID(clusterName, apiEndpoint)
	if err != nil {
		return err
	}

	_, err = c.APIGatewayv2().DeleteRoute(&apigatewayv2.DeleteRouteInput{
		ApiId:   &apiID,
		RouteId: &routeID,
	})
	if err != nil {
		return errors.Wrap(err, "failed to delete API with endpoint ", apiEndpoint)
	}
	return nil
}

// DoesAPIGatewayExist check if a cortex API Gateway exists
func (c *Client) doesAPIGatewayExist(clusterName string) (bool, error) {
	apis, err := c.APIGatewayv2().GetApis(&apigatewayv2.GetApisInput{})
	if err != nil {
		return false, errors.Wrap(err, "failed to get api gateway apis")
	}

	// find API with tag cortex.dev/cluster-name=<CLUSTERNAME>
	for _, api := range apis.Items {
		for tag, value := range api.Tags {
			if tag == "cortex.dev/cluster-name" && *value == clusterName {
				return true, nil
			}
		}
	}
	return false, nil
}

// DoesVPCLinkExist check if a VPC cortex link exists
func (c *Client) doesVPCLinkExist(clusterName string) (bool, error) {
	vpcLinks, err := c.APIGatewayv2().GetVpcLinks(&apigatewayv2.GetVpcLinksInput{})
	if err != nil {
		return false, errors.Wrap(err, "failed to get vpc links")
	}

	// find vpc link with tag cortex.dev/cluster-name=<CLUSTERNAME>
	for _, vpcLink := range vpcLinks.Items {
		for tag, value := range vpcLink.Tags {
			if tag == "cortex.dev/cluster-name" && *value == clusterName {
				return true, nil
			}
		}
	}
	return false, nil
}

// CheckIfAPIDeployed return a true if API endpoint is deployed in API Gateway
func (c *Client) CheckIfAPIDeployed(clusterName string, apiEndpoint string) (bool, error) {
	apiID, err := c.getAPIGatewayID(clusterName)
	if err != nil {
		return false, err
	}

	routes, err := c.APIGatewayv2().GetRoutes(&apigatewayv2.GetRoutesInput{
		ApiId: &apiID,
	})
	if err != nil {
		return false, errors.Wrap(err, "failed to get routes for API with endpoint", apiEndpoint, " and ID", apiID)
	}

	// find route which matches the endpoint
	for _, route := range routes.Items {
		if *route.RouteKey == "ANY "+apiEndpoint {
			return true, nil
		}
	}
	return false, nil
}

//DeleteVPCLink delete API Gateways vpc link
func (c *Client) DeleteVPCLink(clusterName string) error {
	// first check if VPC Link exists
	vpcLinkExists, err := c.doesVPCLinkExist(clusterName)
	if err != nil {
		return err
	}

	// if vpc link doesn't exist return
	if !vpcLinkExists {
		return nil
	}

	// vpc link exists and gets deleted by vpc link
	vpcLinkID, err := c.getVpcLinkID(clusterName)
	if err != nil {
		return err
	}

	_, err = c.APIGatewayv2().DeleteVpcLink(&apigatewayv2.DeleteVpcLinkInput{
		VpcLinkId: &vpcLinkID,
	})
	if err != nil {
		return err
	}
	return nil
}

//DelteAPIGateway delete API Gateway
func (c *Client) DelteAPIGateway(clusterName string) error {
	// first check if API Gateway exists
	apiExists, err := c.doesAPIGatewayExist(clusterName)
	if err != nil {
		return err
	}

	// if api doesn't exist return
	if !apiExists {
		err = fmt.Errorf("no API gateway found")
		return err
	}

	// api gateway exists and gets deleted by API ID
	apiID, err := c.getAPIGatewayID(clusterName)
	if err != nil {
		return err
	}

	_, err = c.APIGatewayv2().DeleteApi(&apigatewayv2.DeleteApiInput{
		ApiId: &apiID,
	})
	if err != nil {
		return err
	}
	return nil
}
