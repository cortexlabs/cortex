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

	"github.com/aws/aws-sdk-go/service/apigatewayv2"
	"github.com/cortexlabs/cortex/pkg/lib/errors"
)

// GetVpcLinkID vpc-link id only exists if internal facing loadbalancer
func (c *Client) GetVpcLinkID(clusterName string) (string, error) {

	vpcLinks, err := c.APIGatewayv2().GetVpcLinks(&apigatewayv2.GetVpcLinksInput{})
	if err != nil {
		return "", errors.Wrap(err, "failed to get vpc links")
	}
	// find vpc link with tag cortex.dev/cluster-name=<CLUSTERNAME>
	for _, vpcLink := range vpcLinks.Items {
		for tag, value := range vpcLink.Tags {
			if tag == "cortex.dev/cluster-name" && *value == clusterName {
				return *vpcLink.VpcLinkId, nil
			}

		}
	}
	return "", fmt.Errorf("failed to find cortex vpc link")

}

// GetAPIGatewayID gets the API ID of the API Gateway api
func (c *Client) GetAPIGatewayID(clusterName string) (string, error) {

	apis, err := c.APIGatewayv2().GetApis(&apigatewayv2.GetApisInput{})
	if err != nil {
		return "", errors.Wrap(err, "failed to get api gateway apis")
	}
	// find API with tag cortex.dev/cluster-name=<CLUSTERNAME>
	for _, api := range apis.Items {
		for tag, value := range api.Tags {
			if tag == "cortex.dev/cluster-name" && *value == clusterName {
				return *api.ApiId, nil
			}

		}
	}
	return "", fmt.Errorf("failed to find api gateway apis")

}

// DoesAPIGatewayExist check if an API Gateway exists
func (c *Client) DoesAPIGatewayExist(clusterName string) (bool, error) {

	apiID, err := c.GetAPIGatewayID(clusterName)
	if err != nil {
		return false, err
	}
	if apiID != "" {
		return true, nil
	}
	return false, nil

}

// DoesVPCLinkExist check if a vpc link exists
func (c *Client) DoesVPCLinkExist(clusterName string) (bool, error) {

	vpcLinkID, err := c.GetVpcLinkID(clusterName)
	if err != nil {
		return false, err
	}
	if vpcLinkID != "" {
		return true, nil
	}
	return false, nil

}

//DeleteVPCLink delete API Gateway vpc link
func (c *Client) DeleteVPCLink(clusterName string) error {

	// first check if VPC Link exists
	vpcLinkExists, err := c.DoesVPCLinkExist(clusterName)
	if err != nil {
		return err
	}
	// if vpc link doesn't exist return
	if !vpcLinkExists {
		return nil
	}

	// vpc link exists and gets deleted by vpc link
	vpcLinkID, err := c.GetVpcLinkID(clusterName)
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
	apiExists, err := c.DoesAPIGatewayExist(clusterName)
	if err != nil {
		return err
	}
	// if api doesn't exist return
	if !apiExists {
		return nil
	}

	// api gateway exists and gets deleted by API ID
	apiID, err := c.GetAPIGatewayID(clusterName)
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
